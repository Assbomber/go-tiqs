package tiqs

import (
	"fmt"
	"sync"
	"time"
)

const BARS_MAX_LEN = 1500

// ---------------------------------------------------------------------------

// AddStrategy creates a new strategy and adds it to the trader.
//
// name: Name of the strategy.
// symbol: Symbol of the strategy.
// onTick: Function to be called when a new tick is received.
func (at *AutoTrader) AddStrategy(name string, symbol string, onTick OnTickFn) IStrategy {
	at.log(DEBUG, "adding strategy :", name)
	at.strategiesLock.Lock()
	defer at.strategiesLock.Unlock()
	// already exists
	if _, ok := at.strategies[name]; ok {
		at.log(DEBUG, "strategy already exists :", name)
		return nil
	}
	// Create a new strategy instance and initialize its fields.
	s := &strategy{
		at:                            at,                          // trader this strategy belongs to
		name:                          name,                        // Name of the strategy.
		symbol:                        symbol,                      // Symbol
		openPos:                       make(map[string]*Position),  // Open positions.
		ordEntry:                      make(map[string]EntryOpts),  // Entry orders.
		ordExit:                       make(map[string]ExitOpts),   // Exit orders.
		ordCancel:                     make(map[string]bool),       // Cancel orders
		closedPos:                     make(map[string][]Position), // closed positions.
		tiqsOrderIdToLocalOrderId:     make(map[string]string),
		onTick:                        onTick, // Function to be called when a new tick is received.
		openPosLock:                   &sync.RWMutex{},
		ordEntryLock:                  &sync.RWMutex{},
		ordExitLock:                   &sync.RWMutex{},
		closedPosLock:                 &sync.RWMutex{},
		ordCancelLock:                 &sync.RWMutex{},
		tiqsOrderIdToLocalOrderIdLock: &sync.RWMutex{},
		ticksChan:                     make(chan Tick, 50),
		ordUpdatesChan:                make(chan OrderUpdate, 50),
		bars:                          make([]float64, 0, BARS_MAX_LEN),
		strategyPnL:                   0,
		stopTickListenerSig:           make(chan bool, 1),
	}

	// Get the token for the provided symbol and append the new strategy as a tick listener.
	symbolToken, err := at.getTokenFromSymbol(symbol)
	if err != nil {
		at.log(ERROR, "failed to get token for symbol :", symbol)
		return nil
	}
	at.strategies[name] = s

	// subscribe for ticks for this symbol
	at.socket.AddSubscription(symbolToken)

	at.tickListenersLock.Lock()
	defer at.tickListenersLock.Unlock()
	at.tickListeners[symbolToken] = append(at.tickListeners[symbolToken], s)

	// start listeners
	go s.startTicksListener()
	go s.startOrderUpdatesListener()

	at.log(INFO, "➕ new strategy created :", name)
	return s
}

/*
--------------------------------------------------------------------------

Returns the symbol name
*/
func (st *strategy) GetSymbol() string {
	return st.symbol
}

/*
---------------------------------------------------------------------------

Stops tick listener for this strategy.
*/
func (st *strategy) stopTicksListener() {
	st.stopTickListenerSig <- true
	close(st.ticksChan)
}

/*
---------------------------------------------------------------------------

Stops order updates listener for this strategy.
*/
func (st *strategy) stopOrderUpdatesListener() {
	close(st.ordUpdatesChan)
}

/*
---------------------------------------------------------------------------

Continuously listens and executes the strategy for each tick.
!This is blocking
*/
func (st *strategy) startTicksListener() {
	for tick := range st.ticksChan {
		select {
		case <-st.stopTickListenerSig:
			st.at.log(DEBUG, "ticks listener stopped, strategy: ", st.name)
			return
		default:
			st.at.log(DEBUG, "↓ recieved tick , strategy: ", st.name, ", symbol: ", st.symbol, "ts", tick.Time)
			st.insertBar(tick)
			st.execute(st.bars, tick)
		}

	}
}

/*
---------------------------------------------------------------------------

Continuously listens for order updates for this strategy.
!This is blocking
*/
func (st *strategy) startOrderUpdatesListener() {
	for orderUpdate := range st.ordUpdatesChan {
		localOrdID, ok := st.getTiqsOrderIdToLocalOrderId(orderUpdate.ID)
		if !ok {
			st.at.log(ERROR, "no corresponding order id found for tiqs order id : ", orderUpdate.ID)
			continue
		}
		pos, ok := st.getOpenPos(localOrdID)
		if !ok {
			st.at.log(ERROR, "no position found for local order id  : ", localOrdID)
			continue
		}

		var isEntryUpdate bool
		if pos.TiqsEntryOrdID == orderUpdate.ID {
			isEntryUpdate = true
		}

		switch orderUpdate.Status {

		case string(COMPLETE): // ------------------------
			if isEntryUpdate { // entry case
				pos.Qty = orderUpdate.Qty
				pos.EntryTime = orderUpdate.ExchangeTime
				pos.EntryPx = orderUpdate.AvgPrice
				pos.Status = EntryComplete

			} else { // exit case
				// calculating qty left
				pos.Qty = pos.Qty - orderUpdate.Qty

				// creating clone for closed position
				copyPos := *pos
				copyPos.ExitTime = orderUpdate.ExchangeTime
				copyPos.ExitPx = orderUpdate.AvgPrice
				copyPos.Qty = orderUpdate.Qty
				copyPos.TiqsExitOrdID = orderUpdate.ID
				copyPos.Reason = orderUpdate.Reason

				if pos.Qty == 0 { // if this was full exit case
					st.deleteOpenPos(localOrdID)
					copyPos.Status = ExitComplete
				} else { // partial exit
					// clearing exit orderid cuz may require to another exit order to clear qty left
					pos.TiqsExitOrdID = ""
					pos.Status = ExitPartial
				}
				st.insertClosedPos(localOrdID, copyPos)
				fmt.Println("Closed positions", st.GetAllClosedPositions())
			}

			// since this tiqs order ID is completed...no further use of these mappings
			st.deleteTiqsOrderIdToLocalOrderId(orderUpdate.ID)
			st.at.deleteTiqsOrderIdToStrategy(orderUpdate.ID)

		case string(REJECTED), string(CANCELED): // ---------------------------
			if isEntryUpdate { // entry case
				// deleting this position as it was rejected by TIQS
				st.deleteOpenPos(localOrdID)
			} else { // exit case
				// removing the tiqs exit order id from pos, since was rejected
				pos.TiqsExitOrdID = ""
				pos.Status = EntryComplete
			}

			// since this tiqs order ID is REJECTED/CANCELLED...no further use of these mappings
			st.deleteTiqsOrderIdToLocalOrderId(orderUpdate.ID)
			st.at.deleteTiqsOrderIdToStrategy(orderUpdate.ID)

		case string(OPEN): // -------------------------------
			if isEntryUpdate { // entry case
				pos.Status = EntryOpen
			} else { // exit case
				pos.Status = ExitOpen
			}
		}
	}
}

/*
---------------------------------------------------------------------------

Adjusts a new tick to the bars array.
*/
func (st *strategy) insertBar(t Tick) {
	price := float64(t.LTP) / 100

	if len(st.bars) == BARS_MAX_LEN {
		// bars max length must be BARS_MAX_LEN
		st.bars = st.bars[1:]
	}
	st.bars = append(st.bars, price)
}

/*
---------------------------------------------------------------------------

Returns the trader reference which this strategy belongs to
*/
func (st *strategy) GetTrader() *AutoTrader {
	return st.at
}

/*
---------------------------------------------------------------------------

Returns the name of the strategy
*/
func (st *strategy) GetName() string {
	return st.name
}

/*
---------------------------------------------------------------------------

Returns the PnL of the strategy
*/
func (st *strategy) GetPnL() float64 {
	return st.strategyPnL
}

/*
---------------------------------------------------------------------------

getTiqsOrderIdToLocalOrderId returns the local order id associated with the given TIQS order id
from the tiqsOrderIdToLocalOrderId map. If the order id is not found, an empty string and false are returned.
*/
func (s *strategy) getTiqsOrderIdToLocalOrderId(tiqsID string) (string, bool) {
	s.tiqsOrderIdToLocalOrderIdLock.RLock()
	defer s.tiqsOrderIdToLocalOrderIdLock.RUnlock()
	localOrderId, ok := s.tiqsOrderIdToLocalOrderId[tiqsID]
	return localOrderId, ok
}

/*
---------------------------------------------------------------------------

deleteTiqsOrderIdToLocalOrderId removes the mapping of a tiqs order id to
the associated local order id from the tiqsOrderIdToLocalOrderId map.
This is usually done when an order is canceled.
*/
func (s *strategy) deleteTiqsOrderIdToLocalOrderId(tiqsID string) {
	s.tiqsOrderIdToLocalOrderIdLock.Lock()
	defer s.tiqsOrderIdToLocalOrderIdLock.Unlock()
	delete(s.tiqsOrderIdToLocalOrderId, tiqsID)
}

/*
---------------------------------------------------------------------------

insertOrdExit inserts a new exit order in the ordExit map.
This is usually done when an exit order is placed.
*/
func (s *strategy) insertOrdExit(orderID string, exit ExitOpts) {
	s.ordExitLock.Lock()
	defer s.ordExitLock.Unlock()
	s.ordExit[orderID] = exit
}

/*
---------------------------------------------------------------------------

insertOrdEntry inserts a new entry order in the ordEntry map.
This is usually done when an entry order is placed.
*/
func (s *strategy) insertOrdEntry(orderID string, entry EntryOpts) {
	s.ordEntryLock.Lock()
	defer s.ordEntryLock.Unlock()
	s.ordEntry[orderID] = entry
}

/*
	---------------------------------------------------------------------------

getOpenPos returns the open position associated with the given orderID.
If the order id is not found, (nil, false) is returned.
*/
func (s *strategy) getOpenPos(orderID string) (*Position, bool) {
	s.openPosLock.RLock()
	defer s.openPosLock.RUnlock()
	p, ok := s.openPos[orderID]
	return p, ok
}

/*
---------------------------------------------------------------------------

insertOpenPos inserts a new open position in the open positions map.
This is usually done when an entry order is executed.
*/
func (s *strategy) insertOpenPos(orderID string, pos *Position) {
	s.openPosLock.Lock()
	defer s.openPosLock.Unlock()
	s.openPos[orderID] = pos
}

/*
---------------------------------------------------------------------------

deleteOpenPos deletes a position from the open positions map.
This is usually done when a position is closed by an exit order.
*/
func (s *strategy) deleteOpenPos(orderID string) {
	s.openPosLock.Lock()
	defer s.openPosLock.Unlock()
	delete(s.openPos, orderID)
}

/*
	---------------------------------------------------------------------------

insertClosedPos inserts a closed position in the closed positions map.
This is usually done when a position is closed by an exit order.
*/
func (s *strategy) insertClosedPos(orderID string, pos Position) {
	s.closedPosLock.Lock()
	defer s.closedPosLock.Unlock()
	s.closedPos[orderID] = append(s.closedPos[orderID], pos)
}

/*
-----------------------------------------------------------------------------

It is a command to enter market position.
If an order with the same ID is already pending, it is possible to modify the order.
If there is no order with the specified ID, a new order is placed.
To deactivate an entry order, the command strategy.Cancel or strategy.Cancel_all should be used.
*/
func (s *strategy) Entry(orderID string, opts EntryOpts) error {
	s.at.log(DEBUG, "📝 added entry order.", "orderId : ", orderID, "strategy : ", s.name)

	opts.OrderID = orderID
	if err := validate.Struct(opts); err != nil {
		return err
	}

	s.insertOrdEntry(orderID, opts)
	return nil
}

/*
------------------------------------------------------------------------------

It is a command to exit either a specific entry.
If an order with the same ID is already pending, it is possible to modify the order.
If an entry order was not filled, but an exit order is generated,
the exit order will wait till entry order is filled and then the exit order is placed.
To deactivate an exit order, the command strategy.Cancel or strategy.Cancel_all should be used.
*/
func (s *strategy) Exit(orderID string, opts ExitOpts) error {
	s.at.log(DEBUG, "📕 added exit order.", "orderId : ", orderID, "strategy : ", s.name)

	opts.OrderID = orderID
	if err := validate.Struct(opts); err != nil {
		return err
	}

	s.insertOrdExit(orderID, opts)
	return nil
}

/*
------------------------------------------------------------------------------

Calls onTick method of a strategy and takes further actions
based on what strategy just calculate.
*/
func (s *strategy) execute(closeSeries []float64, tick Tick) {
	s.at.log(DEBUG, "⚡ executing orders.", "strategy : ", s.name)

	defer func() {
		if r := recover(); r != nil {
			s.at.log(ERROR, "error executing order on tick", "strategy :", s.name, "error :", r)
		}
	}()
	if s.onTick != nil {
		s.onTick(s, tick, closeSeries)
	}

	// strategy has been marked as unplugged
	if s.unplug {
		s.shutdown()
		return
	}

	ltp := float64(tick.LTP) / 100

	// process Pnls
	s.processPnls(tick)
	// Entry orders
	s.processEntryOrders(tick)
	// Exit orders
	s.processExitOrders(ltp)
	// Cancel orders
	s.processCancelOrders()
	s.at.log(DEBUG, "⚡ executed orders.", "strategy : ", s.name)
}

// ----------------------------------------------------------------------

// !UNIMPLEMENTED

// It is a command to cancel/deactivate pending orders by referencing their orderID
// If order is not found, will wait for it to be placed
func (s *strategy) Cancel(orderID string) {
	// s.log(DEBUG, "canceling order.", "orderId : ", orderID, "strategy : ", s.name)
	// s.ordCancel[orderID] = true
}

/*
---------------------------------------------------------------------------

Returns an open position with the given orderID if available, else returns nil
*/
func (s *strategy) GetOpenPositionByOrderID(orderID string) *Position {
	p, _ := s.getOpenPos(orderID)
	return p
}

/*
---------------------------------------------------------------------------

Returns all closed positions in this strategy
*/
func (s *strategy) GetAllClosedPositions() []Position {
	s.closedPosLock.RLock()
	pos := make([]Position, 0, len(s.closedPos))
	for _, p := range s.closedPos {
		pos = append(pos, p...)
	}
	s.closedPosLock.RUnlock()
	return pos
}

/*
---------------------------------------------------------------------------

Returns slice of positions for the given orderID
*/
func (s *strategy) GetClosedPositionsByOrderID(orderID string) []Position {
	s.closedPosLock.RLock()
	p := s.closedPos[orderID]
	s.closedPosLock.RUnlock()
	return p
}

/*
---------------------------------------------------------------------------

Marks as strategy to be removed
*/
func (s *strategy) Unplug() {
	s.at.log(DEBUG, "🔌 strategy marked as unplugged.", "strategy :", s.name)
	s.unplug = true
}

/*
---------------------------------------------------------------------------

Process all entry orders that are ready to be executed.
Loops through ordEntry map and places orders to tiqs backend.
*/
func (s *strategy) processEntryOrders(tick Tick) {
	s.at.log(DEBUG, "processing entry orders, strategy :", s.name)

	ltp := float64(tick.LTP) / 100
	tickTS := time.Unix(int64(tick.Time), 0)

	deletedEntryIds := []string{}

	s.ordEntryLock.RLock()
	for id, e := range s.ordEntry {

		// if this order is already executed and is open, continue
		_, found := s.getOpenPos(id)
		if found {
			s.at.log(DEBUG, "⏭ entry position already exists. skipping.", "orderId : ", id, "strategy : ", s.name)
			deletedEntryIds = append(deletedEntryIds, e.OrderID)
			continue
		}

		// figure out buy or sell
		action := Buy
		if e.Direction == Short {
			action = Sell
		}

		s.tiqsOrderIdToLocalOrderIdLock.Lock()
		s.at.tiqsOrderIdsToStrategyLock.Lock()

		// place order to tiqs backend.
		symbolToken, err := s.at.getTokenFromSymbol(s.symbol)
		if err != nil {
			s.at.log(ERROR, err, "orderID :", e.OrderID," strategy:",s.name)
		} else {
			s.at.log(DEBUG, "🛒 placing order to backend for:", s.symbol," strategy:",s.name)
			res, err := s.at.placeOrder(prepareOrder(
				prepareOrderArgs{
					Symbol: s.symbol,
					Token:  symbolToken,
					Qty:    e.Qty,
					Limit:  e.Limit,
					Stop:   e.Stop,
					LTP:    ltp,
					action: action,
				},
			))
			if err != nil {
				s.at.log(ERROR, err, "orderID :", e.OrderID," strategy:",s.name)
			} else {
				// order success... store as open position
				s.insertOpenPos(e.OrderID, &Position{
					Symbol:         s.symbol,
					EntryPx:        ltp,
					EntryTime:      tickTS,
					Direction:      e.Direction,
					Qty:            e.Qty,
					OrdID:          e.OrderID,
					TiqsEntryOrdID: res.Data.OrderNo,
					Status:         EntryPending,
				})

				// storing tiqs order ID to our local order ID for future lookups
				// ? will be helpful to manage updates to order via socket.
				s.tiqsOrderIdToLocalOrderId[res.Data.OrderNo] = e.OrderID
				// storing tiqs order ID to current strategy name for future lookups
				// ? will be helpful to manage updates to order via socket.
				s.at.tiqsOrderIdsToStrategy[res.Data.OrderNo] = s.name
				deletedEntryIds = append(deletedEntryIds, e.OrderID)
			}
		}

		s.at.tiqsOrderIdsToStrategyLock.Unlock()
		s.tiqsOrderIdToLocalOrderIdLock.Unlock()
	}
	s.ordEntryLock.RUnlock()

	// removed converted entries from map
	s.ordEntryLock.Lock()
	for _, id := range deletedEntryIds {
		delete(s.ordEntry, id)
	}
	s.ordEntryLock.Unlock()
	s.at.log(DEBUG, "processed entry orders, strategy :", s.name)
}

/*
---------------------------------------------------------------------------

Process all exit orders that are ready to be executed.
Loops through ordExit map and places orders to tiqs backend.
*/
func (s *strategy) processExitOrders(ltp float64) {
	s.at.log(DEBUG, "processing exit orders, strategy :", s.name)

	deletedExitIds := []string{}
	// convert positions into exit orders
	s.ordExitLock.RLock()
	for id, e := range s.ordExit {
		p, found := s.getOpenPos(id)

		// if position yet to come or exit already placed, continue
		if !found || p.TiqsExitOrdID != "" {
			s.at.log(DEBUG, "🤷‍♂️ exit position not found or exit already placed. removing exit", "orderId : ", id, "strategy : ", s.name)
			deletedExitIds = append(deletedExitIds, e.OrderID)
			continue
		}

		if p.Status < EntryComplete {
			s.at.log(DEBUG, "⏳ waiting for entry to complete. skipping.", "orderId : ", id, "strategy : ", s.name)
			continue
		}

		// figure out buy or sell
		action := Buy
		if p.Direction == Long {
			action = Sell
		}

		s.tiqsOrderIdToLocalOrderIdLock.Lock()
		s.at.tiqsOrderIdsToStrategyLock.Lock()

		// place order to tiqs backend.
		symbolToken, err := s.at.getTokenFromSymbol(s.symbol)
		if err != nil {
			s.at.log(ERROR, err, "orderID :", e.OrderID," strategy:",s.name)
		} else {
			s.at.log(DEBUG, "🛒 placing order to backend for:", s.symbol," strategy:",s.name)
			res, err := s.at.placeOrder(prepareOrder(
				prepareOrderArgs{
					Symbol: s.symbol,
					Token:  symbolToken,
					Qty:    min(e.Qty, p.Qty),
					Limit:  e.Limit,
					Stop:   e.Stop,
					LTP:    ltp,
					action: action,
				},
			))

			if err != nil {
				s.at.log(ERROR, err, "orderID :", e.OrderID," strategy:",s.name)
			} else {
				// order success... update your position
				p.Status = ExitPending
				p.TiqsExitOrdID = res.Data.OrderNo
				// ? not updating qty,exit price here, will updated on socket confirmation

				// storing tiqs order ID to our local order ID for future lookups
				// ? will be helpful to manage updates to order via socket.
				s.tiqsOrderIdToLocalOrderId[res.Data.OrderNo] = e.OrderID
				// storing tiqs order ID to current strategy name for future lookups
				// ? will be helpful to manage updates to order via socket.
				s.at.tiqsOrderIdsToStrategy[res.Data.OrderNo] = s.name
				deletedExitIds = append(deletedExitIds, e.OrderID)
			}
		}

		s.at.tiqsOrderIdsToStrategyLock.Unlock()
		s.tiqsOrderIdToLocalOrderIdLock.Unlock()
	}
	s.ordExitLock.RUnlock()

	// removed converted exits from map
	s.ordExitLock.Lock()
	for _, id := range deletedExitIds {
		delete(s.ordExit, id)
	}
	s.ordExitLock.Unlock()
	s.at.log(DEBUG, "processed exit orders, strategy :", s.name)
}

/*
---------------------------------------------------------------------------

Process all cancel orders that are ready to be executed.
Loops through ordCancel map and cancels orders from tiqs backend.
*/
func (s *strategy) processCancelOrders() {
	s.at.log(DEBUG, "processing cancel orders, strategy :", s.name)

	deletedCancelIds := []string{}
	s.ordCancelLock.RLock()
	for id := range s.ordCancel {
		p, found := s.getOpenPos(id)
		// if position yet to come, continue
		if !found {
			continue
		}

		var tiqsID string
		if p.Status < EntryComplete {
			// cancel entry order
			tiqsID = p.TiqsEntryOrdID

		} else if p.TiqsExitOrdID != "" {
			// cancel exit order
			tiqsID = p.TiqsExitOrdID
		} else {
			s.at.log(ERROR, "canceling order failed. No open/pending orders found :", p," strategy:",s.name)
			deletedCancelIds = append(deletedCancelIds, id)
			continue
		}

		_, err := s.at.cancelOrder(tiqsID)
		if err != nil {
			s.at.log(ERROR, err, "orderID :", p.OrdID," strategy:",s.name)
			continue
		}
		deletedCancelIds = append(deletedCancelIds, id)
	}
	s.ordCancelLock.RUnlock()

	// removed converted cancels from map
	s.ordCancelLock.Lock()
	for _, id := range deletedCancelIds {
		delete(s.ordCancel, id)
	}
	s.ordCancelLock.Unlock()
	s.at.log(DEBUG, "processed cancel orders, strategy :", s.name)
}

/*
-------------------------------------------------------------------------

Closes all open position if any
*/
func (s *strategy) closeOpenPositions(ltp float64) {
	s.at.log(DEBUG, "🚧 closing all open positions for strategy :", s.name)

	s.openPosLock.RLock()
	for _, p := range s.openPos {
		s.insertOrdExit(p.OrdID, ExitOpts{OrderID: p.OrdID, Qty: p.Qty})
	}
	s.openPosLock.RUnlock()
	s.processExitOrders(ltp)
	s.at.log(DEBUG, "🚧 all positions closed for strategy :", s.name)
}

/*
------------------------------------------------------------------------------

Calculates PNL of all open positions
*/
func (s *strategy) processPnls(tick Tick) {
	ltp := float64(tick.LTP) / 100
	var strategyPNL float64 = 0
	s.openPosLock.RLock()
	for _, ps := range s.openPos {
		// getting the PNL of position
		pnl := (ltp - ps.EntryPx)
		// setting the PNL for position
		ps.PnL = pnl

		// sum for all positions PNL : Overall Stragey PNL
		strategyPNL += pnl
	}
	s.openPosLock.RUnlock()
	s.strategyPnL = strategyPNL
}

// Graceful shutdown of open positions
// !It does not gaurantees that all open positions will be closed.
func (s *strategy) shutdown() {
	s.at.log(DEBUG, "📢 Shut down signal recieved, strategy :", s.name)
	// close ticks channel, so that no more executes happen
	s.stopTicksListener()

	// gracefully shut down this strategy
	s.closeOpenPositions(s.bars[len(s.bars)-1])

	// wait to 2 seconds to let the order updates come and do their job
	time.Sleep(2 * time.Second)

	// since this strategy will be removed, persisting it at autotrader level.
	s.at.closedPositionsMutex.Lock()
	s.at.closedPositions = append(s.at.closedPositions, s.GetAllClosedPositions()...)
	s.at.closedPositionsMutex.Unlock()

	// removing strategy from trader.
	s.at.removeStrategy(s.name)

	// close order updates channel
	s.stopOrderUpdatesListener()
	s.at.log(DEBUG, "💤 Shut down successful, strategy :", s.name)
}

/*
------------------------------------------------------------------------------

Accepts a new tick for this strategy and sends it to the ticksListener channel
*/
func (s *strategy) newTick(tick Tick) {
	defer func() {
		if r := recover(); r != nil {
			s.at.log(DEBUG, "channel has been closed, strategy :", s.name)
		}
	}()
	s.ticksChan <- tick
}
