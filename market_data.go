package tiqs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetOrderBook returns the order book of the user.
func (c *Client) GetOrderBook() (*OrderBookResponse, error) {
	// Create a new request
	client := http.DefaultClient
	req, err := http.NewRequest("GET", orderBookEndpoint, nil)
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var response OrderBookResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrOrderBookFailed, response)
	}

	// Return the response
	return &response, nil
}

// GetTradeBook returns the trade book of the user.
func (c *Client) GetTradeBook() (*TradeBookResponse, error) {
	// Create a new request
	client := http.DefaultClient
	req, err := http.NewRequest("GET", tradeBookEndpoint, nil)
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var response TradeBookResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrTradeBookFailed, response)
	}

	// Return the response
	return &response, nil
}

// GetPositionBook returns the position book of the user.
func (c *Client) GetPositionBook() (*PositionBookResponse, error) {
	// Create a new request
	client := http.DefaultClient
	req, err := http.NewRequest("GET", positionBookEndpoint, nil)
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var response PositionBookResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrPositionBookFailed, response)
	}

	// Return the response
	return &response, nil
}

// GetOrderStatus returns the status of an order
func (c *Client) GetOrderStatus(orderID string) (string, error) {
	// Create a new request
	client := http.DefaultClient
	url := fmt.Sprintf("%s/%s", getOrderStatusEndpoint, orderID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Decode the response
	var response OrderStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return "", fmt.Errorf("%w, response: %+v", ErrGetOrderStatusFailed, response)
	}

	// we will fetch the order status from first element because it is latest updated
	orderStatus := response.Data[0].OrderStatus
	return orderStatus, nil
}

// GetOptionChain fetches the option chain details for the given parameters.
func (c *Client) GetOptionChain(optionChainreq OptionChainRequest) (*OptionChainResponse, error) {
	// Marshal the request body
	jsonData, err := json.Marshal(optionChainreq)
	if err != nil {
		return nil, err
	}

	// Create a new request
	req, err := http.NewRequest("POST", getOptionChainEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var response OptionChainResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrOptionChainFailed, response)
	}

	// Return the response
	return &response, nil
}

// GetOrderMargin sends a POST request to the /margin/order endpoint to get the order margin.
// If the request is not successful, it returns an error.
func (c *Client) GetOrderMargin(marginReq MarginRequest) (*MarginDetailResponse, error) {

	client := http.DefaultClient
	jsonData, err := json.Marshal(marginReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", getMarginEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var response MarginDetailResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrMarginFailed, response)
	}

	// Return the response
	return &response, nil
}

// GetBasketMargin sends a POST request to the /margin/basket endpoint to get the basket margin.
// If the request is not successful, it returns an error.
func (c *Client) GetBasketMargin(marginReq []MarginRequest) (*BasketMarginResponse, error) {

	// Create a new request
	client := http.DefaultClient

	// Marshal the request body
	jsonData, err := json.Marshal(marginReq)
	if err != nil {
		return nil, err
	}

	// Create the request
	req, err := http.NewRequest("POST", getBasketMarginEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Close the response
	defer resp.Body.Close()

	// Decode the response
	var response BasketMarginResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	// Check if the request was successful
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrBasketMarginFailed, response)
	}

	// Return the response
	return &response, nil
}

// This function returns ltp of a symbol in Paisa
func (c *Client) GetLTPFromAPI(dataToken int) (int, error) {

	client := http.DefaultClient
	// Create JSON payload
	payload := fmt.Sprintf(`{
			"token": %d
		}`, dataToken)

	req, err := http.NewRequest("POST", getLTPEndpoint, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var response QuoteResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}
	if response.Status != "success" {
		return 0, fmt.Errorf("%w, response: %+v", ErrGettingLTP, response)
	}
	// this ltp is in Paisa
	return response.Data.LTP, nil
}

// GetExpiryDates returns the list of expiry dates for famous INDICES
//
//	GET /market-data/option-expiry-dates
func (c *Client) GetExpiryDates() (*ExpiryDateResponse, error) {
	client := http.DefaultClient
	req, err := http.NewRequest("GET", getExpriyDatesEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response ExpiryDateResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrGettingExpiryDates, response)
	}
	return &response, nil
}
