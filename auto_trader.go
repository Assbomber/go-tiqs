package tiqs

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"
)

var autoTraderLogo = `
/$$$$$$              /$$            /$$$$$$$$                       /$$                    
/$$__  $$            | $$           |__  $$__/                      | $$                    
| $$  \ $$ /$$   /$$ /$$$$$$    /$$$$$$ | $$  /$$$$$$  /$$$$$$   /$$$$$$$  /$$$$$$   /$$$$$$ 
| $$$$$$$$| $$  | $$|_  $$_/   /$$__  $$| $$ /$$__  $$|____  $$ /$$__  $$ /$$__  $$ /$$__  $$
| $$__  $$| $$  | $$  | $$    | $$  \ $$| $$| $$  \__/ /$$$$$$$| $$  | $$| $$$$$$$$| $$  \__/
| $$  | $$| $$  | $$  | $$ /$$| $$  | $$| $$| $$      /$$__  $$| $$  | $$| $$_____/| $$      
| $$  | $$|  $$$$$$/  |  $$$$/|  $$$$$$/| $$| $$     |  $$$$$$$|  $$$$$$$|  $$$$$$$| $$      
|__/  |__/ \______/    \___/   \______/ |__/|__/      \_______/ \_______/ \_______/|__/  by Tiqs.in  
`

type LogLvl string

const (
	INFO  LogLvl = "INFO"
	ERROR LogLvl = "ERROR"
	DEBUG LogLvl = "DEBUG"
)

type AutoTrader struct {
	*Client

	// Tiqs websocket client
	socket *TiqsWSClient
	// map to store deployed strategies using strategy key
	strategiesLock *sync.RWMutex
	strategies     map[string]*strategy
	// tiqs order Ids to strategy name
	tiqsOrderIdsToStrategyLock *sync.RWMutex
	tiqsOrderIdsToStrategy     map[string]string
	// map that stores symbol token to strategy names which are deployed on that symbol
	tickListenersLock *sync.RWMutex
	tickListeners     map[int][]*strategy
	// bool to represent if the debug logging is enable by the user
	enableDebugLog bool
	// stores last 1500 bars per token

	// stores LTP for subscribed symbols
	ltpsLock *sync.RWMutex
	ltps     map[int]float64

	// stores symbol to token mapping
	symbolToTokenMap map[string]int
	// stores token to symbol mapping
	tokenToSymbolMap map[int]string

	// Stores underlying to strike price to its PE and CE symbol
	optionChainSymbols map[string]map[int]OptionSymbol

	// closed positions
	closedPositionsMutex *sync.Mutex
	closedPositions      []Position
}

// NewAutoTrader returns a new instance of AutoTrader.
//
// AutoTrader is a high level interface on top of Tiqs client.
// It provides a simpler interface to deploy trading strategies on
// different symbols.
//
// enableDebugLog is an optional parameter. If set to true, it will
// enable debug logging which can be useful for debugging purposes.
//
// It also starts two go routines:
//  1. startTickListener: Listens for new ticks and notifies the
//     deployed strategies.
//  2. orderUpdateListener: Listens for order updates and notifies the
//     deployed strategies.
//
// It returns an error if it fails to fetch symbol name and token.
func (c *Client) NewAutoTrader(enableDebugLog bool) (*AutoTrader, error) {
	fmt.Println(autoTraderLogo)
	socket, err := c.NewSocket(enableDebugLog)
	if err != nil {
		return nil, err
	}
	at := &AutoTrader{
		Client:                 c,
		socket:                 socket,
		enableDebugLog:         enableDebugLog,
		strategies:             make(map[string]*strategy),
		tiqsOrderIdsToStrategy: make(map[string]string),
		tickListeners:          make(map[int][]*strategy),
		ltps:                   make(map[int]float64),
		tokenToSymbolMap: map[int]string{
			26009: "NIFTYBANK",
			26000: "NIFTY50",
			26037: "FINNIFTY",
			26074: "MIDCPNIFTY",
		},
		symbolToTokenMap: map[string]int{
			"NIFTYBANK":  26009,
			"NIFTY50":    26000,
			"FINNIFTY":   26037,
			"MIDCPNIFTY": 26074,
		},
		tickListenersLock:          &sync.RWMutex{},
		strategiesLock:             &sync.RWMutex{},
		tiqsOrderIdsToStrategyLock: &sync.RWMutex{},
		ltpsLock:                   &sync.RWMutex{},
		optionChainSymbols:         make(map[string]map[int]OptionSymbol),
		closedPositionsMutex:       &sync.Mutex{},
		closedPositions:            make([]Position, 0),
	}
	// Starting tick listener in a separate go routine
	go at.startTickListener()

	// Starting order update listener in a separate go routine
	go at.orderUpdateListener()

	// Fetching SymbolName and token
	err = at.fetchingSymbolNameAndToken()
	if err != nil {
		return nil, err
	}

	return at, nil
}

// does socket subscription for all tokens in option chain
func (at *AutoTrader) SubscribeFullOptionChain() {
	for token := range at.tokenToSymbolMap {
		at.socket.AddSubscription(token)
	}
}

// Listens for new ticks and forwards it to deployed strategies.
func (at *AutoTrader) startTickListener() {
	at.log(DEBUG, "started tick listener")
	for tick := range at.socket.GetDataChannel() {
		token := int(tick.Token)

		// save ltp for this token.
		price := float64(tick.LTP) / 100
		at.ltpsLock.Lock()
		at.ltps[token] = price
		at.ltpsLock.Unlock()

		at.tickListenersLock.RLock()
		listners := at.tickListeners[token]
		at.tickListenersLock.RUnlock()
		for _, listener := range listners {
			listener.newTick(tick)
		}
	}
}

// Listeners to order updates from websockets and updates the existing positions
func (at *AutoTrader) orderUpdateListener() {
	at.log(DEBUG, "started order listener")
	for orderUpdate := range at.socket.GetOrderChannel() {

		at.log(DEBUG, "🔔 recieved order update , tiqsOrderID :", orderUpdate.ID, ", status : ", orderUpdate.Status, ", reason:", orderUpdate.Reason)
		strategyName, ok := at.getTiqsOrderIdToStrategyName(orderUpdate.ID)
		if !ok {
			at.log(ERROR, "no strategy name found for tiqs order ID : ", orderUpdate.ID)
			continue
		}
		strategy, ok := at.getStrategy(strategyName)
		if !ok {
			at.log(ERROR, "no strategy found with name : ", strategyName)
			continue
		}

		strategy.ordUpdatesChan <- orderUpdate
	}
}

// Returns token from symbol. if not found returns error instead
func (at *AutoTrader) getTokenFromSymbol(symbol string) (int, error) {
	token, ok := at.symbolToTokenMap[symbol]
	if !ok {
		return 0, fmt.Errorf("token not found for symbol %s", symbol)
	}
	return token, nil
}

// Returns token from symbol.  if not found returns error instead
func (at *AutoTrader) getSymbolFromToken(token int) (string, error) {
	symbol, ok := at.tokenToSymbolMap[token]
	if !ok {
		return "", fmt.Errorf("token not found")
	}
	return symbol, nil
}

// Returns LTP for a symbol. if not present returns 0
func (at *AutoTrader) GetLTP(symbol string) (float64, error) {

	token, err := at.getTokenFromSymbol(symbol)
	if err != nil {
		return 0, err
	}
	ltp, ok := at.ltps[token]
	if !ok {
		return 0, fmt.Errorf("ltp not found for symbol %s", symbol)
	}
	return ltp, nil
}

// getStrategy returns the strategy associated with the given name.
// If the strategy does not exist, (nil, false) is returned.
func (at *AutoTrader) getStrategy(strategyName string) (*strategy, bool) {
	at.strategiesLock.RLock()
	defer at.strategiesLock.RUnlock()
	strategy, ok := at.strategies[strategyName]
	return strategy, ok
}

// deleteTiqsOrderIdToStrategy removes the mapping of a tiqs order id to
// the associated strategy name
func (at *AutoTrader) deleteTiqsOrderIdToStrategy(tiqsID string) {
	at.tiqsOrderIdsToStrategyLock.Lock()
	defer at.tiqsOrderIdsToStrategyLock.Unlock()
	delete(at.tiqsOrderIdsToStrategy, tiqsID)
}

// getTiqsOrderIdToStrategyName returns the strategy name associated with the given TIQS order id
// if the order id is not found, an empty string and false are returned
func (at *AutoTrader) getTiqsOrderIdToStrategyName(tiqsID string) (string, bool) {
	at.tiqsOrderIdsToStrategyLock.RLock()
	defer at.tiqsOrderIdsToStrategyLock.RUnlock()
	s, ok := at.tiqsOrderIdsToStrategy[tiqsID]
	return s, ok
}

// Returns all deployed strategies
// !Important. Returned strategy is a pointer. Modifications results modification of original strategy.
func (at *AutoTrader) GetAllStrategies() []*strategy {
	allStrategies := []*strategy{}
	at.strategiesLock.RLock()
	for _, s := range at.strategies {
		allStrategies = append(allStrategies, s)
	}
	at.strategiesLock.RUnlock()
	return allStrategies
}

// Returns strategy using strategy key
// !Important. Returned strategy is a pointer. Modifications results modification of original strategy.
func (at *AutoTrader) GetStrategy(key string) *strategy {
	at.strategiesLock.RLock()
	s := at.strategies[key]
	at.strategiesLock.RUnlock()
	return s
}

// Removes strategy using strategy key from auto trader
func (at *AutoTrader) removeStrategy(key string) {
	at.log(DEBUG, "removing strategy : ", key)

	// get this strategy
	strategy := at.GetStrategy(key)
	if strategy == nil {
		at.log(ERROR, "strategy not found for key : ", key)
		return
	}

	// delete from the strategies map
	at.strategiesLock.Lock()
	delete(at.strategies, key)
	at.strategiesLock.Unlock()

	// remove this strategy from list of tick listeners
	at.tickListenersLock.Lock()
	// base index for not found cases
	idx := -1
	token, err := at.getTokenFromSymbol(strategy.symbol)
	if err != nil {
		at.log(ERROR, "token not found for symbol : ", strategy.symbol)
	}

	// fetch the index for this strategy from tick listeners
	for i, s := range at.tickListeners[token] {
		if s == strategy {
			idx = i
			break
		}
	}

	// remove that index
	if idx != -1 {
		at.tickListeners[token] = append(at.tickListeners[token][:idx], at.tickListeners[token][idx+1:]...)
	}
	at.tickListenersLock.Unlock()
	at.log(DEBUG, "removed strategy : ", key)
}

// Response represents the structure of the cancel API response
type cancelResponse struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
	Status string `json:"status"`
}

func (at *AutoTrader) log(lvl LogLvl, msg ...any) {
	// not required
	if !at.enableDebugLog {
		return
	}

	switch lvl {
	case INFO:
		log.Println("ℹ️ ", msg)
	case ERROR:
		log.Println("❗", msg, string(debug.Stack()))
	case DEBUG:
		log.Println("👾", msg)
	default:
		log.Println(msg...)
	}
}

func (at *AutoTrader) fetchingSymbolNameAndToken() error {

	expiryDate, err := at.GetExpiryDates()
	if err != nil {
		return fmt.Errorf("error while fetching expiry dates: %w", ErrGettingExpiryDates)
	}

	// Fetching Symbol Names and Token for BANKNIFTY
	currentExpiryDate := expiryDate.Data.BANKNIFTY[0]
	token, err := at.getTokenFromSymbol("NIFTYBANK")
	if err != nil {
		return fmt.Errorf("error getting token for NIFTYBANK")
	}
	err = at.insertingSymbolsName(token, currentExpiryDate)
	if err != nil {
		return err
	}

	// Fetching Symbol Names and Token for NIFTY
	currentExpiryDate = expiryDate.Data.NIFTY[0]
	token, err = at.getTokenFromSymbol("NIFTY50")
	if err != nil {
		return fmt.Errorf("error getting token for NIFTY50")
	}
	err = at.insertingSymbolsName(token, currentExpiryDate)
	if err != nil {
		return err
	}

	// Fetching Symbol Names and Token for MIDCPNIFTY
	currentExpiryDate = expiryDate.Data.MIDCPNIFTY[0]
	token, err = at.getTokenFromSymbol("MIDCPNIFTY")
	if err != nil {
		return fmt.Errorf("error getting token for MIDCPNIFTY")
	}
	err = at.insertingSymbolsName(token, currentExpiryDate)
	if err != nil {
		return err
	}

	// Fetching Symbol Names and Token for FINNIFTY
	currentExpiryDate = expiryDate.Data.FINNIFTY[0]
	token, err = at.getTokenFromSymbol("FINNIFTY")
	if err != nil {
		return fmt.Errorf("error getting token for FINNIFTY")
	}
	err = at.insertingSymbolsName(token, currentExpiryDate)
	if err != nil {
		return err
	}

	// Same way if any another index comes in future then add here

	return nil
}

func (at *AutoTrader) insertingSymbolsName(token int, currentExpiryDate string) error {
	optionChainRequest := OptionChainRequest{
		Token:    fmt.Sprintf("%d", token),
		Exchange: "INDEX",
		Count:    "20",
		Expiry:   currentExpiryDate,
	}
	optChainResp, err := at.GetOptionChain(optionChainRequest)
	if err != nil {
		return fmt.Errorf("error while fetching option chain: %w", ErrOptionChainFailed)
	}
	underlying, err := at.getSymbolFromToken(token)
	if err != nil {
		return fmt.Errorf("error while fetching underlying symbol: %w", err)
	}

	at.optionChainSymbols[underlying] = map[int]OptionSymbol{}

	for _, opt := range optChainResp.Data {
		token, err := strconv.Atoi(opt.Token)
		if err != nil {
			return fmt.Errorf("error parsing symbol token, %v", err)
		}

		sp, _ := strconv.ParseFloat(opt.StrikePrice, 64)
		strikePrice := int(sp)

		strikeSymbols := at.optionChainSymbols[underlying][strikePrice]
		if opt.OptionType == "CE" {
			at.optionChainSymbols[underlying][strikePrice] = OptionSymbol{CE: opt.Symbol, PE: strikeSymbols.PE}
		} else {
			at.optionChainSymbols[underlying][strikePrice] = OptionSymbol{PE: opt.Symbol, CE: strikeSymbols.CE}
		}
		at.symbolToTokenMap[opt.Symbol] = token
		at.tokenToSymbolMap[token] = opt.Symbol
	}
	return nil
}

// Returns CE & PE symbols using strik price
func (at *AutoTrader) GetOptionSymbolsUsingStrike(underlying string, strike int) (string, string, error) {

	symbol, ok := at.optionChainSymbols[underlying][strike]
	if !ok {
		return "", "", fmt.Errorf("symbol not found for %s & strike ,%d", underlying, strike)
	}
	return symbol.CE, symbol.PE, nil
}

type prepareOrderArgs struct {
	Symbol string
	Token  int
	Qty    int
	Limit  float64
	Stop   float64
	LTP    float64
	action action
}

func prepareOrder(args prepareOrderArgs) OrderRequest {

	order := OrderRequest{
		AMO:             false,
		DisclosedQty:    "0",
		Exchange:        "NFO",
		Order:           "MKT",
		Price:           "0",
		Product:         "M",
		Quantity:        fmt.Sprint(args.Qty),
		Symbol:          args.Symbol,
		Token:           fmt.Sprint(args.Token),
		TransactionType: "B",
		TriggerPrice:    "0",
		Validity:        "DAY",
	}

	// action type
	if args.action == "Sell" {
		order.TransactionType = "S"
	} else {
		order.TransactionType = "B"
	}

	// order type
	if args.Limit == 0 && args.Stop != 0 {
		order.Order = "SL-MKT"
		order.TriggerPrice = fmt.Sprintf("%f", args.Stop)
		order.Price = fmt.Sprintf("%f", args.LTP)
	} else if args.Limit != 0 && args.Stop == 0 {
		order.Order = "LMT"
		order.Price = fmt.Sprintf("%f", args.Limit)
	} else if args.Limit != 0 && args.Stop != 0 {
		order.Order = "SL-LMT"
		order.TriggerPrice = fmt.Sprintf("%f", args.Stop)
		order.Price = fmt.Sprintf("%f", args.Limit)
	} else {
		order.Order = "MKT"
		order.Price = fmt.Sprintf("%f", args.LTP)
	}
	return order
}

// Graceful Shutdown
func (at *AutoTrader) Shutdown() {
	at.log(DEBUG, "🚨 Shutting down AutoTrader...")
	// shutdown each strategy
	wg := sync.WaitGroup{}
	wg.Add(len(at.strategies))
	for _, s := range at.strategies {
		go func(s *strategy) {
			defer wg.Done()
			s.shutdown()
		}(s)
	}
	wg.Wait()

	// sorting by entry time
	sort.Slice(at.closedPositions, func(i, j int) bool {
		return at.closedPositions[i].EntryTime.Before(at.closedPositions[j].EntryTime)
	})

	// output file
	outputFile, err := os.Create(fmt.Sprintf("closed_positions_%s.csv", time.Now().Format("20060102-150405")))
	if err != nil {
		log.Fatal(err)
	}

	w := csv.NewWriter(outputFile)
	defer w.Flush()

	// header
	err = w.Write([]string{
		"Symbol",
		"EntryPx",
		"ExitPx",
		"EntryTime",
		"ExitTime",
		"Qty",
		"Direction",
		"OrdID",
		"TiqsEntryOrdID",
		"TiqsExitOrdID",
		"Reason",
	})
	if err != nil {
		log.Fatal(err)
	}

	// write each row
	for _, position := range at.closedPositions {
		err = w.Write([]string{
			position.Symbol,
			fmt.Sprintf("%.2f", position.EntryPx),
			fmt.Sprintf("%.2f", position.ExitPx),
			position.EntryTime.Format("2006-01-02 15:04:05"),
			position.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%d", position.Qty),
			string(position.Direction),
			position.OrdID,
			position.TiqsEntryOrdID,
			position.TiqsExitOrdID,
			position.Reason,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	at.log(INFO, "🛑 AutoTrader shutdown successful")
}
