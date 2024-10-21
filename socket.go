package tiqs

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Constants
const (
	CODE_SUB        = "sub"
	CODE_UNSUB      = "unsub"
	MODE_FULL       = "full"
	maxRetries      = 20
	BUFFER_SIZE     = 100000
	FULLTICK_LENGTH = 197
)

// EndPoints
const SOCKET_URL = "wss://wss.tiqs.trading"

// Info Messages
const (
	InfoSocketConnected                    = "🟢 Connected to socket"
	INFO_RECONNECT_REQUEST_IGNORED         = "🙈 Reconnect request ignored. Already requested."
	InfoReconnectLimitReached              = "✋ Socket reconnection limit reached."
	InfoSocketConnecting                   = "⏳ Connecting to socket..."
	INFO_SOCKET_PING_DIFFERENCE            = "🆚 Socket ping difference exceeded: Reconnecting..."
	INFO_PROCCESSING_PENDING_REQUESTS      = "⏳ Processing pending requests..."
	INFO_PROCCESSING_PREVIOUS_SUBSCRIPTION = "⏳ Processing previous subscriptions if any..."
	INFO_CLOSED_WEBSOCKET                  = "🔴 WebSocket connection closed"
	INFO_INVALID_TICK_DATA                 = "Invalid tick data length "
)

type OrderStatus string

const (
	COMPLETE OrderStatus = "COMPLETE"
	REJECTED OrderStatus = "REJECTED"
	PENDING  OrderStatus = "PENDING"
	OPEN     OrderStatus = "OPEN"
	CANCELED OrderStatus = "CANCELED"
)

// NewSocket sets up the WebSocket connection and related processes
// This function should be called from your main function
func (c *Client) NewSocket(enableLog bool) (*TiqsWSClient, error) {
	tiqsWSClient := TiqsWSClient{
		appID:         c.appID,
		accessToken:   c.accessToken,
		subscriptions: make(map[int]struct{}),
		tickChannel:   make(chan Tick, BUFFER_SIZE),
		orderChannel:  make(chan OrderUpdate, BUFFER_SIZE),
		enableLog:     enableLog,
		wsURL:         fmt.Sprintf("%s?appId=%s&token=%s", SOCKET_URL, c.appID, c.accessToken),
	}
	tiqsWSClient.connectSocket()

	return &tiqsWSClient, nil
}

// connectSocket establishes a WebSocket connection to the given URL
// It also initializes various processes like ping checking and subscription handling
func (t *TiqsWSClient) connectSocket() {

	dialer := websocket.DefaultDialer
	dialer.ReadBufferSize = 8192 // Increase buffer size (adjust as needed)

	var err error
	for i := 0; i < maxRetries; i++ {
		t.logger(InfoSocketConnecting, ". attempt:", i+1)
		t.socket, _, err = dialer.Dial(t.wsURL, nil)
		if err != nil { // failed to dail
			t.logger(ErrSocketConnection, ". reason:", err)

			// max limit reached. exit
			if i == maxRetries-1 {
				t.logger(InfoReconnectLimitReached)
				close(t.orderChannel)
				close(t.tickChannel)
				return
			}

			time.Sleep(3 * time.Second)
		} else { // dial was successful, break from loop
			break
		}
	}

	t.logger(InfoSocketConnected)
	// ping checker
	t.startPingChecker()
	// process previous connections
	t.subscribePreviousSubscriptions()
	t.processPendingRequests()

	t.socket.SetReadLimit(1024 * 1024) // Set max message size to 1MB (adjust as needed)
	go t.readMessages()
}

// readMessages continuously reads messages from the WebSocket
// It handles different types of messages, including PING messages
func (t *TiqsWSClient) readMessages() {
	for {
		// read message ---------------------------------------------------------
		_, message, err := t.socket.ReadMessage()
		if err != nil {
			if e, ok := err.(*websocket.CloseError); ok {
				t.logger(ErrReadingSocketMessage, ". reason:", e.Error())
			} else {
				t.logger(ErrReadingSocketMessage, ". reason:", err)
			}
			// reconnect
			t.closeAndReconnect()
			return
		}

		// decode messages ------------------------------------------------------
		if string(message) == "PING" { // ping from server
			t.lastPingTS = time.Now()
			t.emit("PONG", false)

		} else if isOrderUpdate(string(message)) { // order update
			update, err := decodeOrderMessage(message)
			if err != nil {
				t.logger(ErrDecodingMessage)
				continue
			}
			t.orderChannel <- update

		} else if len(message) == FULLTICK_LENGTH { // tick update
			tick := t.parseTick(message)
			t.tickChannel <- tick

		} else { // unknown message
			t.logger(fmt.Sprintf("Received message with unexpected length: %d", len(message)))

		}
	}
}

func (t *TiqsWSClient) closeAndReconnect() {
	t.CloseConnection()
	t.connectSocket()
}

// emit sends a message through the WebSocket
// If the socket is not connected, it queues the message (unless volatile is true)
func (t *TiqsWSClient) emit(message interface{}, volatile bool) {

	var msg []byte
	var err error

	switch v := message.(type) {
	case string:
		msg = []byte(v)
	case SocketMessage:
		msg, err = json.Marshal(v)
		if err != nil {
			t.logger(ErrMarshlingMsg)
			return
		}
	default:
		t.logger(ErrUnsupportedMsgType)
		return
	}

	if t.socket != nil { // server connected
		err := t.socket.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			t.logger(ErrEmitingToSocket)
			if !volatile {
				t.pendingQueue = append(t.pendingQueue, message)
			}
		} //else {
		// 	t.logger("⬆ Emitted to socket:", string(msg))
		// }
	} else { // server is not connected
		t.logger(ErrSocketNotConnected)
		if !volatile {
			t.pendingQueue = append(t.pendingQueue, message)
		}
	}
}

// startPingChecker initiates a periodic check to ensure the connection is alive
// If no PING is received within 35 seconds, it triggers a reconnection
func (t *TiqsWSClient) startPingChecker() {
	if t.pingCheckerTimer != nil {
		t.pingCheckerTimer.Stop()
	}

	t.lastPingTS = time.Now()

	t.pingCheckerTimer = time.AfterFunc(35*time.Second, func() {
		diff := time.Since(t.lastPingTS)
		if diff > 35*time.Second {
			t.logger(INFO_SOCKET_PING_DIFFERENCE)
			t.closeAndReconnect()
		}
		t.startPingChecker()
	})
}

// processPendingRequests sends any queued messages that couldn't be sent earlier
// due to connection issues
func (t *TiqsWSClient) processPendingRequests() {

	if len(t.pendingQueue) > 0 {
		t.logger(INFO_PROCCESSING_PENDING_REQUESTS)
		for _, request := range t.pendingQueue {
			t.emit(request, false)
		}
		t.pendingQueue = nil
	}
}

// subscribePreviousSubscriptions resubscribes to all previously subscribed topics
// This is useful when reconnecting to ensure all subscriptions are maintained
func (t *TiqsWSClient) subscribePreviousSubscriptions() {
	t.logger(INFO_PROCCESSING_PREVIOUS_SUBSCRIPTION)
	for token := range t.subscriptions {
		t.emit(SocketMessage{
			Code: CODE_SUB,
			Mode: MODE_FULL,
			Full: []int{token},
		}, false)
	}
}

// AddSubscription adds a new subscription to the store
func (t *TiqsWSClient) AddSubscription(token int) {
	t.subscriptions[token] = struct{}{}
	// subscribePreviousSubscriptions()
	t.emit(SocketMessage{
		Code: CODE_SUB,
		Mode: MODE_FULL,
		Full: []int{token},
	}, false)
}

// RemoveSubscription removes a subscription from the store
func (t *TiqsWSClient) RemoveSubscription(token int) {
	delete(t.subscriptions, token)
	t.emit(SocketMessage{
		Code: CODE_UNSUB,
		Mode: MODE_FULL,
		Full: []int{token},
	}, false)
}

// GetSubscriptions returns the current subscriptions
func (t *TiqsWSClient) GetSubscriptions() map[int]struct{} {
	return t.subscriptions
}

// GetDataChannel returns the data channel
func (t *TiqsWSClient) GetDataChannel() <-chan Tick {
	return t.tickChannel
}

// GetOrderChannel returns the order update channel
func (t *TiqsWSClient) GetOrderChannel() <-chan OrderUpdate {
	return t.orderChannel
}

// CloseConnection closes the WebSocket connection
func (t *TiqsWSClient) CloseConnection() {
	if t.socket == nil {
		return
	}

	t.socket.Close()
	t.socket = nil
	t.logger(INFO_CLOSED_WEBSOCKET)

	// Stop timers
	if t.pingCheckerTimer != nil {
		t.pingCheckerTimer.Stop()
		t.pingCheckerTimer = nil
	}

	// Clear pending queue
	t.pendingQueue = nil
}

// bytesToInt32 takes a byte slice as input and parses it into an int32
// The byte slice must be of length 4
// It returns the parsed int32 value
func bytesToInt32(data []byte) int32 {
	// Check if the length of the byte slice is as expected
	if len(data) != 4 {
		// t.logger(ErrInvalidByteSliceLength)
		return 0
	}

	// Parse the byte slice into an int32
	// We use bitwise operations to shift the bytes into their correct positions
	// The result is the parsed int32 value
	value := int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3])

	return value
}

// parseTick takes a byte slice as input and parses it into a Tick struct
// It returns a Tick and a boolean indicating whether the parsing was successful
func (t *TiqsWSClient) parseTick(data []byte) Tick {

	// Create a new Tick struct and fill it with data from the byte slice
	var tick = Tick{
		Token:              bytesToInt32(data[0:4]),               // Token
		LTP:                bytesToInt32(data[4:8]),               // Last traded price
		NetChangeIndicator: int32(data[8]),                        // Net change indicator
		NetChange:          bytesToInt32(data[9:13]),              // Net change
		LTQ:                bytesToInt32(data[13:17]),             // Last traded quantity
		AvgPrice:           bytesToInt32(data[17:21]),             // Average traded price
		TotalBuyQuantity:   bytesToInt32(data[21:25]),             // Total buy quantity
		TotalSellQuantity:  bytesToInt32(data[25:29]),             // Total sell quantity
		Open:               bytesToInt32(data[29:33]),             // Open price
		High:               bytesToInt32(data[33:37]),             // High price
		Close:              bytesToInt32(data[37:41]),             // Close price
		Low:                bytesToInt32(data[41:45]),             // Low price
		Volume:             bytesToInt32(data[45:49]),             // Volume
		LTT:                bytesToInt32(data[49:53]),             // Last traded time
		Time:               bytesToInt32(data[53:57]) + 315513000, // Time
		OI:                 bytesToInt32(data[57:61]),             // Open interest
		OIDayHigh:          bytesToInt32(data[61:65]),             // Open interest day high
		OIDayLow:           bytesToInt32(data[65:69]),             // Open interest day low
		LowerLimit:         bytesToInt32(data[69:73]),             // Lower limit
		UpperLimit:         bytesToInt32(data[73:77]),             // Upper limit
	}

	// Return a Tick
	return tick
}

func (t *TiqsWSClient) logger(msg ...any) {
	if t.enableLog {
		log.Println(msg...)
	}
}

// Check if the keyword 'orderUpdate' exists in the given string
func isOrderUpdate(input string) bool {
	// Use strings.Contains to check for the keyword
	return strings.Contains(input, "orderUpdate")
}

func decodeOrderMessage(message []byte) (OrderUpdate, error) {
	var rawOrder map[string]string
	err := json.Unmarshal(message, &rawOrder)
	if err != nil {
		return OrderUpdate{}, err
	}

	var orderUpdate OrderUpdate

	// Assign string fields directly without checks
	orderUpdate.ID = rawOrder["id"]
	orderUpdate.Type = rawOrder["type"]
	orderUpdate.UserID = rawOrder["userId"]
	orderUpdate.Exchange = rawOrder["exchange"]
	orderUpdate.Symbol = rawOrder["symbol"]
	orderUpdate.Product = rawOrder["product"]
	orderUpdate.Status = rawOrder["status"]
	orderUpdate.ReportType = rawOrder["reportType"]
	orderUpdate.TransactionType = rawOrder["transactionType"]
	orderUpdate.Order = rawOrder["order"]
	orderUpdate.Retention = rawOrder["retention"]
	orderUpdate.Reason = rawOrder["reason"]
	orderUpdate.ExchangeOrderId = rawOrder["exchangeOrderId"]
	orderUpdate.CancelQty = rawOrder["cancelQty"]
	orderUpdate.Tags = rawOrder["tags"]
	orderUpdate.DisclosedQty = rawOrder["disclosedQty"]
	orderUpdate.TriggerPrice = rawOrder["triggerPrice"]

	// Convert Token to int with existence check
	if val, ok := rawOrder["token"]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			orderUpdate.Token = intVal
		}
	}

	// Convert Qty to int with existence check
	if val, ok := rawOrder["qty"]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			orderUpdate.Qty = intVal
		}
	}

	// Convert Price to float64 with existence check
	if val, ok := rawOrder["price"]; ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			orderUpdate.Price = floatVal
		}
	}

	// Convert AvgPrice to float64 with existence check
	if val, ok := rawOrder["avgPrice"]; ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			orderUpdate.AvgPrice = floatVal
		}
	}

	// Convert Timestamp to time.Time with existence check
	if val, ok := rawOrder["timestamp"]; ok {
		if timeVal, err := strconv.Atoi(val); err == nil {
			orderUpdate.Timestamp = time.Unix(int64(timeVal), 0)
		}
	}

	// Convert exchangeTime to time.Time with existence check
	if val, ok := rawOrder["exchangeTime"]; ok {
		if timeVal, err := time.Parse("02-01-2006 15:04:05", val); err == nil {
			orderUpdate.ExchangeTime = timeVal
		}
	}

	return orderUpdate, nil
}
