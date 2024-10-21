package tiqs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Allows the user to place single trade
func (c *Client) placeOrder(order OrderRequest) (*OrderResponse, error) {

	client := http.DefaultClient
	jsonData, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", placeOrderEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response OrderResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("%w, response: %+v", ErrOrderPlacementFailed, response)
	}
	return &response, nil
}

// CancelOrder sends a DELETE request to cancel an order
func (c *Client) cancelOrder(tiqsID string) (*cancelResponse, error) {
	// Construct the URL
	url := fmt.Sprintf("https://api.tiqs.trading/order/regular/%s", tiqsID)

	// Create a new request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("appId", c.appID)
	req.Header.Set("token", c.accessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the JSON response
	var response cancelResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %v", err)
	}

	// Check if the status code is not 200 OK
	if response.Status != "success" {
		return nil, fmt.Errorf("failed to cancel order. body: %v", response)
	}

	return &response, nil
}
