package tiqs

const baseURL = "https://api.tiqs.trading"

const placeOrderEndpoint = baseURL + "/order/regular"
const orderBookEndpoint = baseURL + "/user/orders"
const tradeBookEndpoint = baseURL + "/user/trades"
const positionBookEndpoint = baseURL + "/user/positions"
const getLTPEndpoint = baseURL + "/info/quote/ltp"
const getMarginEndpoint = baseURL + "/margin/order"
const getBasketMarginEndpoint = baseURL + "/margin/basket"
const getOptionChainEndpoint = baseURL + "/info/option-chain"
const getOrderStatusEndpoint = baseURL + "/order"
const getExpriyDatesEndpoint = baseURL + "/info/option-chain-symbols"
