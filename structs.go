package tiqs

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Order struct {
	Status                   string `json:"status"`
	UserID                   string `json:"userID"`
	AccountID                string `json:"accountID"`
	Exchange                 string `json:"exchange"`
	Symbol                   string `json:"symbol"`
	ID                       string `json:"id"`
	RejectReason             string `json:"rejectReason"`
	Price                    string `json:"price"`
	Quantity                 string `json:"quantity"`
	MarketProtection         string `json:"marketProtection"`
	Product                  string `json:"product"`
	OrderStatus              string `json:"orderStatus"`
	TransactionType          string `json:"transactionType"`
	Order                    string `json:"order"`
	FillShares               string `json:"fillShares"`
	AveragePrice             string `json:"averagePrice"`
	ExchangeOrderID          string `json:"exchangeOrderID"`
	CancelQuantity           string `json:"cancelQuantity"`
	Remarks                  string `json:"remarks"`
	DisclosedQuantity        string `json:"disclosedQuantity"`
	OrderTriggerPrice        string `json:"orderTriggerPrice"`
	Retention                string `json:"retention"`
	BookProfitPrice          string `json:"bookProfitPrice"`
	BookLossPrice            string `json:"bookLossPrice"`
	TrailingPrice            string `json:"trailingPrice"`
	Amo                      string `json:"amo"`
	PricePrecision           string `json:"pricePrecision"`
	TickSize                 string `json:"tickSize"`
	LotSize                  string `json:"lotSize"`
	Token                    string `json:"token"`
	TimeStamp                string `json:"timeStamp"`
	OrderTime                string `json:"orderTime"`
	ExchangeUpdateTime       string `json:"exchangeUpdateTime"`
	SnoOrderDirection        string `json:"snoOrderDirection"`
	SnoOrderID               string `json:"snoOrderID"`
	PriceFactor              string `json:"priceFactor"`
	Multiplier               string `json:"multiplier"`
	DisplayName              string `json:"displayName"`
	RequiredQuantity         string `json:"requiredQuantity"`
	RequiredPrice            string `json:"requiredPrice"`
	RequiredTriggerPrice     string `json:"requiredTriggerPrice"`
	RequiredBookLossPrice    string `json:"requiredBookLossPrice"`
	RequiredOriginalQuantity string `json:"requiredOriginalQuantity"`
	RequiredOriginalPrice    string `json:"requiredOriginalPrice"`
	OriginalTriggerPrice     string `json:"originalTriggerPrice"`
	OriginalBookLossPrice    string `json:"originalBookLossPrice"`
}

type OrderBookResponse struct {
	Data   []Order `json:"data"`
	Status string  `json:"status"`
}

type TradeBookResponse struct {
	Data   []TradeData `json:"data"`
	Status string      `json:"status"`
}

type TradeData struct {
	Status             string `json:"status"`
	UserID             string `json:"userID"`
	AccountID          string `json:"accountID"`
	Exchange           string `json:"exchange"`
	Symbol             string `json:"symbol"`
	ID                 string `json:"id"`
	Quantity           string `json:"quantity"`
	Product            string `json:"product"`
	TransactionType    string `json:"transactionType"`
	Order              string `json:"order"`
	FillShares         string `json:"fillShares"`
	AveragePrice       string `json:"averagePrice"`
	ExchangeOrderID    string `json:"exchangeOrderID"`
	Remarks            string `json:"remarks"`
	Retention          string `json:"retention"`
	PricePrecision     string `json:"pricePrecision"`
	TickSize           string `json:"tickSize"`
	LotSize            string `json:"lotSize"`
	CustomFirm         string `json:"customFirm"`
	FillTime           string `json:"fillTime"`
	FillID             string `json:"fillID"`
	FillPrice          string `json:"fillPrice"`
	FillQuantity       string `json:"fillQuantity"`
	OrderSource        string `json:"orderSource"`
	Token              string `json:"token"`
	TimeStamp          string `json:"timeStamp"`
	ExchangeUpdateTime string `json:"exchangeUpdateTime"`
	SnoOrderDirection  string `json:"snoOrderDirection"`
	SnoOrderID         string `json:"snoOrderID"`
	RequestTime        string `json:"requestTime"`
	ErrorMessage       string `json:"errorMessage"`
}

type PositionBookResponse struct {
	Data   []PositionBookData `json:"data"`
	Status string             `json:"status"`
}

type PositionBookData struct {
	AvgPrice                 string `json:"avgPrice"`
	BreakEvenPrice           string `json:"breakEvenPrice"`
	CarryForwardAvgPrice     string `json:"carrtForwardAvgPrice"`
	CarryForwardBuyAmount    string `json:"carryForwardBuyAmount"`
	CarryForwardBuyAvgPrice  string `json:"carryForwardBuyAvgPrice"`
	CarryForwardBuyQty       string `json:"carryForwardBuyQty"`
	CarryForwardSellAmount   string `json:"carryForwardSellAmount"`
	CarryForwardSellAvgPrice string `json:"carryForwardSellAvgPrice"`
	CarryForwardSellQty      string `json:"carryForwardSellQty"`
	DayBuyAmount             string `json:"dayBuyAmount"`
	DayBuyAvgPrice           string `json:"dayBuyAvgPrice"`
	DayBuyQty                string `json:"dayBuyQty"`
	DaySellAmount            string `json:"daySellAmount"`
	DaySellAvgPrice          string `json:"daySellAvgPrice"`
	DaySellQty               string `json:"daySellQty"`
	Exchange                 string `json:"exchange"`
	LotSize                  string `json:"lotSize"`
	LTP                      string `json:"ltp"`
	Multiplier               string `json:"multiplier"`
	NetUploadPrice           string `json:"netUploadPrice"`
	OpenBuyAmount            string `json:"openBuyAmount"`
	OpenBuyAvgPrice          string `json:"openBuyAvgPrice"`
	OpenBuyQty               string `json:"openBuyQty"`
	OpenSellAmount           string `json:"openSellAmount"`
	OpenSellAvgPrice         string `json:"openSellAvgPrice"`
	OpenSellQty              string `json:"openSellQty"`
	PriceFactor              string `json:"priceFactor"`
	PricePrecision           string `json:"pricePrecision"`
	Product                  string `json:"product"`
	Qty                      string `json:"qty"`
	RealisedPnL              string `json:"realisedPnL"`
	Symbol                   string `json:"symbol"`
	TickSize                 string `json:"tickSize"`
	Token                    string `json:"token"`
	UnrealisedMarkToMarket   string `json:"unrealisedMarkToMarket"`
	UploadPrice              string `json:"uploadPrice"`
}

type OrderStatusResponse struct {
	Data   []orderStatus `json:"data"`
	Status string        `json:"status"`
}

type MarginRequest struct {
	Exchange        string `json:"exchange"`
	Token           string `json:"token"`
	Quantity        string `json:"quantity"`
	Price           string `json:"price"`
	TriggerPrice    string `json:"triggerPrice"`
	Product         string `json:"product"`
	TransactionType string `json:"transactionType"`
	Order           string `json:"order"`
}

type BasketMarginResponse struct {
	Data struct {
		MarginUsed           string `json:"marginUsed"`
		MarginUsedAfterTrade string `json:"marginUsedAfterTrade"`
	} `json:"data"`
	Status string `json:"status"`
}

type MarginDetailResponse struct {
	Data struct {
		Cash   string `json:"cash"`
		Charge struct {
			Brokerage      float64 `json:"brokerage"`
			SebiCharges    float64 `json:"sebiCharges"`
			ExchangeTxnFee float64 `json:"exchangeTxnFee"`
			StampDuty      float64 `json:"stampDuty"`
			Ipft           float64 `json:"ipft"`
			TransactionTax float64 `json:"transactionTax"`
			Gst            struct {
				CGST  float64 `json:"cgst"`
				SGST  float64 `json:"sgst"`
				IGST  float64 `json:"igst"`
				Total float64 `json:"total"`
			} `json:"gst"`
			Total float64 `json:"total"`
		} `json:"charge"`
		Margin     string `json:"margin"`
		MarginUsed string `json:"marginUsed"`
	} `json:"data"`
	Status string `json:"status"`
}

type OptionData struct {
	Exchange       string `json:"exchange"`
	Symbol         string `json:"symbol"`
	Token          string `json:"token"`
	OptionType     string `json:"optionType"`
	StrikePrice    string `json:"strikePrice"`
	PricePrecision string `json:"pricePrecision"`
	TickSize       string `json:"tickSize"`
	LotSize        string `json:"lotSize"`
}

type OptionChainResponse struct {
	Data   []OptionData `json:"data"`
	Status string       `json:"status"`
}

type OptionChainRequest struct {
	Token    string `json:"token"`
	Exchange string `json:"exchange"`
	Count    string `json:"count"`
	Expiry   string `json:"expiry"`
}

type orderStatus struct {
	Status             string `json:"status"`
	Exchange           string `json:"exchange"`
	Symbol             string `json:"symbol"`
	ID                 string `json:"id"`
	Price              string `json:"price"`
	Quantity           string `json:"quantity"`
	Product            string `json:"product"`
	OrderStatus        string `json:"orderStatus"`
	ReportType         string `json:"reportType"`
	TransactionType    string `json:"transactionType"`
	Order              string `json:"order"`
	FillShares         string `json:"fillShares"`
	AveragePrice       string `json:"averagePrice"`
	RejectReason       string `json:"rejectReason"`
	ExchangeOrderID    string `json:"exchangeOrderID"`
	CancelQuantity     string `json:"cancelQuantity"`
	Remarks            string `json:"remarks"`
	DisclosedQuantity  string `json:"disclosedQuantity"`
	OrderTriggerPrice  string `json:"orderTriggerPrice"`
	Retention          string `json:"retention"`
	BookProfitPrice    string `json:"bookProfitPrice"`
	BookLossPrice      string `json:"bookLossPrice"`
	TrailingPrice      string `json:"trailingPrice"`
	Amo                string `json:"amo"`
	PricePrecision     string `json:"pricePrecision"`
	TickSize           string `json:"tickSize"`
	LotSize            string `json:"lotSize"`
	Token              string `json:"token"`
	TimeStamp          string `json:"timeStamp"`
	OrderTime          string `json:"orderTime"`
	ExchangeUpdateTime string `json:"exchangeUpdateTime"`
	RequestTime        string `json:"requestTime"`
	ErrorMessage       string `json:"errorMessage"`
}

type OptionExpiryDate struct {
	BANKNIFTY   []string `json:"BANKNIFTY"`
	FINNIFTY    []string `json:"FINNIFTY"`
	MIDCPNIFTY  []string `json:"MIDCPNIFTY"`
	NIFTY       []string `json:"NIFTY"`
	NiftyNext50 []string `json:"NIFTYNXT50"`
}

type ExpiryDateResponse struct {
	Data   OptionExpiryDate `json:"data"`
	Status string           `json:"status"`
}

type QuoteResponse struct {
	Data struct {
		Close int `json:"close"`
		LTP   int `json:"ltp"`
		Token int `json:"token"`
	} `json:"data"`
	Status string `json:"status"`
}

type Direction string
type PositionStatus int

const (
	EntryPending PositionStatus = iota
	EntryOpen
	EntryComplete
	ExitPending
	ExitOpen
	ExitPartial
	ExitComplete
)

const (
	Long  Direction = "long"
	Short Direction = "short"
)

type IStrategy interface {

	/*
		It is a command to enter market position.
		If an order with the same ID is already pending, it is possible to modify the order.
		If there is no order with the specified ID, a new order is placed.
		To deactivate an entry order, the command strategy.Cancel or strategy.Cancel_all should be used.
	*/
	Entry(orderID string, opts EntryOpts) error
	/*
		It is a command to exit either a specific entry.
		If an order with the same ID is already pending, it is possible to modify the order.
		If an entry order was not filled, but an exit order is generated,
		the exit order will wait till entry order is filled and then the exit order is placed.
		To deactivate an exit order, the command strategy.Cancel or strategy.Cancel_all should be used.
	*/
	Exit(orderID string, opts ExitOpts) error

	// !UNIMPLEMENTED
	// It is a command to cancel/deactivate pending orders by referencing their orderID
	Cancel(orderID string)

	// It returns the trader reference which this strategy belongs to
	GetTrader() *AutoTrader

	// It returns the closed position if available
	GetClosedPositionsByOrderID(orderID string) []Position

	// It returns the open position if available
	GetOpenPositionByOrderID(orderID string) *Position

	// Removes the strategy from traders account.
	// Note: It does not closes any previous existing positions
	Unplug()

	// Returns the symbol name
	GetSymbol() string
}

// The function defenition that will be called when a new tick is received on a implemented strategy
// strategy: working strategy
// tick: most recent tick
// closeSeries: most recent <=1500 close values.
type OnTickFn func(strategy IStrategy, tick Tick, closeSeries []float64)

// strategy represents a trading strategy
type strategy struct {
	// trader this strategy belongs to
	at *AutoTrader
	// Name of the strategy
	name string
	// Symbol for which the strategy is being deployed
	symbol string
	// Function to be called when a new tick is received
	onTick OnTickFn
	// Open positions mapped by order ID
	openPosLock *sync.RWMutex
	openPos     map[string]*Position
	// Stores tiqs Order Id to local order Id values
	tiqsOrderIdToLocalOrderIdLock *sync.RWMutex
	tiqsOrderIdToLocalOrderId     map[string]string
	// Entry orders mapped by order ID
	ordEntryLock *sync.RWMutex
	ordEntry     map[string]EntryOpts
	// Exit orders mapped by order ID
	ordExitLock *sync.RWMutex
	ordExit     map[string]ExitOpts
	// closed positions mapped by order ID
	closedPosLock *sync.RWMutex
	closedPos     map[string][]Position

	// positions to be cancelled
	ordCancelLock *sync.RWMutex
	ordCancel     map[string]bool

	// incomming ticksChan channel
	ticksChan chan Tick
	// order updates channel
	ordUpdatesChan chan OrderUpdate
	// historical bars
	bars []float64

	// indicates whether to remove this strategy from trader's account.
	unplug bool

	// To track the Profit and Loss of the strategy
	strategyPnL float64
	// stop tick listener signal channel
	stopTickListenerSig chan bool
}

// Position represents a market position
type Position struct {
	// Symnbol name
	Symbol string
	// EntryPx represents the entry price of the position
	EntryPx float64
	// ExitPx represents the exit price of the position
	ExitPx float64
	// EntryTime represents the time when the position was opened
	EntryTime time.Time
	// ExitTime represents the time when the position was closed
	ExitTime time.Time
	// Number of contracts/shares/lots/units to trade
	Qty int
	// Direction represents the market position direction. long or short
	Direction Direction
	// OrdID represents the order identifier for the position
	OrdID string
	// TiqsEntryOrdID represents the tiqs.in order identifier for the position entry
	TiqsEntryOrdID string
	// TiqsExitOrdID represents the tiqs.in order identifier for the position exit
	TiqsExitOrdID string
	// Reason for close
	Reason string
	// Status represents the status of the position
	Status PositionStatus
	// PnL represents the profit or loss for this specific position
	PnL float64
}

// Strategy Entry options
type EntryOpts struct {
	// Required. The order identifier. It is possible to cancel or modify an order by referencing its identifier
	OrderID string `validate:"required"`
	//  Required. Market position direction. long or short
	Direction Direction `validate:"required"`
	// Required. Number of contracts/shares/lots/units to trade
	Qty int `validate:"required,gt=0"`
	// Optional. Limit price of the order
	Limit float64 `validate:"omitempty,gt=0"`
	// Optional. Stop price of the order
	Stop float64 `validate:"omitempty,gt=0"`
	// Optional. Comment for the order
	Comment string
}

// Strategy Exit options
type ExitOpts struct {
	// Required. The order identifier. It is possible to cancel or modify an order by referencing its identifier
	OrderID string `validate:"required"`
	// Required. Number of contracts/shares/lots/units to exit a trade with
	Qty int `validate:"required,gt=0"`
	// Optional. Profit target (requires a specific price).
	// If it is specified, a limit order is placed to exit market position at the specified price
	Limit float64 `validate:"omitempty,gt=0"`
	// Optional. Stop loss (requires a specific price).
	// If it is specified, a stop order is placed to exit market position at the specified price (or worse)
	Stop float64 `validate:"omitempty,gt=0"`
	// Optional. Comment for the order
	Comment string
}

type action string

const (
	Sell action = "Sell"
	Buy  action = "Buy"
)

// Place order request params
// TODO: add comments for each property
type OrderRequest struct {
	Exchange        string `json:"exchange"`
	Token           string `json:"token"`
	Quantity        string `json:"quantity"`
	DisclosedQty    string `json:"disclosedQty"`
	Product         string `json:"product"`
	Symbol          string `json:"symbol"`
	TransactionType string `json:"transactionType"`
	Order           string `json:"order"`
	Price           string `json:"price"`
	Validity        string `json:"validity"`
	Tags            string `json:"tags"`
	AMO             bool   `json:"amo"`
	TriggerPrice    string `json:"triggerPrice"`
}

// Place order response params
// TODO: add comments for each property
type OrderResponse struct {
	Message string `json:"message"`
	Data    struct {
		OrderNo     string `json:"orderNo"`
		RequestTime string `json:"requestTime"`
	} `json:"data"`
	Status string `json:"status"`
}

type OptionSymbol struct {
	PE string
	CE string
}

// TiqsWSClient represents the tiqs Websocket client
type TiqsWSClient struct {
	*Client
	appID               string
	accessToken         string
	socket              *websocket.Conn
	pingCheckerTimer    *time.Timer
	lastPingTS          time.Time
	pendingQueue        []interface{}
	wsURL               string
	enableLog           bool
	stopReadMessagesSig chan bool
	stopPingListenerSig chan bool
	subscriptions       map[int]struct{} // All active subscriptions
	tickChannel         chan Tick        // data channel where data will come
	orderChannel        chan OrderUpdate // data channel where order update will come

}

// Tick represents the structure of a tick
type Tick struct {
	// Token
	Token int32
	// Last traded price
	LTP int32
	// Net change indicator
	NetChangeIndicator int32
	// Net change
	NetChange int32
	// Last traded quantity
	LTQ int32
	// Average traded price
	AvgPrice int32
	// Total buy quantity
	TotalBuyQuantity int32
	// Total sell quantity
	TotalSellQuantity int32
	// Open price
	Open int32
	// High price
	High int32
	// Close price
	Close int32
	// Low price
	Low int32
	// Volume
	Volume int32
	// Last traded time
	LTT int32
	// Time
	Time int32
	// Open interest
	OI int32
	// Open interest day high
	OIDayHigh int32
	// Open interest day low
	OIDayLow int32
	// Lower limit
	LowerLimit int32
	// Upper limit
	UpperLimit int32
}

// SocketMessage represents the structure of a socket message : which we are going to send to the websocket
type SocketMessage struct {
	Code string `json:"code"`
	Mode string `json:"mode"`
	Full []int  `json:"full"`
}

// Define the structure to match the incoming JSON message
type OrderUpdate struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	UserID          string    `json:"userId"`
	Exchange        string    `json:"exchange"`
	Symbol          string    `json:"symbol"`
	Token           int       `json:"token"`
	Qty             int       `json:"qty"`
	Price           float64   `json:"price"`
	Product         string    `json:"product"`
	Status          string    `json:"status"`
	ReportType      string    `json:"reportType"`
	TransactionType string    `json:"transactionType"`
	Order           string    `json:"order"`
	Retention       string    `json:"retention"`
	AvgPrice        float64   `json:"avgPrice"`
	Reason          string    `json:"reason"`
	ExchangeOrderId string    `json:"exchangeOrderId"`
	CancelQty       string    `json:"cancelQty"`
	Tags            string    `json:"tags"`
	DisclosedQty    string    `json:"disclosedQty"`
	TriggerPrice    string    `json:"triggerPrice"`
	ExchangeTime    time.Time `json:"exchangeTime"`
	Timestamp       time.Time `json:"timestamp"`
}
