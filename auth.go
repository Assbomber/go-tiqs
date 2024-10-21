package tiqs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pquerna/otp/totp"
)

// API endpoints
const (
	baseURLLogin        = "https://api.tiqs.in/auth/login"
	uRLVerifyTOTP       = "https://api.tiqs.in/auth/validate-2fa"
	authGenerateToken   = "https://api.tiqs.in/auth/app/generate-token"
	authenticationToken = "https://api.tiqs.trading/auth/app/authenticate-token"
)

// sendLogin sends a login request
func sendLogin(client ClientParams) (string, error) {
	payload := map[string]interface{}{
		"userId":       client.UserID,
		"password":     client.Password,
		"captchaValue": "",
		"captchaId":    nil,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(baseURLLogin, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	requestKey := result["data"].(map[string]interface{})["requestId"].(string)
	return requestKey, nil
}

// generateTOTP generates a TOTP code
func generateTOTP(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// verifyTOTP verifies the TOTP
func verifyTOTP(client ClientParams, requestKey, totpCode string) (string, string, error) {
	payload := map[string]string{
		"code":      totpCode,
		"requestId": requestKey,
		"userId":    client.UserID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	resp, err := http.Post(uRLVerifyTOTP, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", "", err
	}

	session := result["data"].(map[string]interface{})["session"].(string)
	token := result["data"].(map[string]interface{})["token"].(string)

	return session, token, nil
}

// authTokenAPI authenticates the token
func authTokenAPI(sessionKey, tokenKey, appID string) (string, error) {
	payload := map[string]string{
		"apiKey": appID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", authGenerateToken, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Session", sessionKey)
	req.Header.Set("Token", tokenKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	redirectURL := result["data"].(map[string]interface{})["redirectUrl"].(string)
	return redirectURL, nil
}

// authenticateToken authenticates the token
func authenticateToken(checksum, token, appID string) (string, string, error) {
	payload := map[string]string{
		"checkSum": checksum,
		"token":    token,
		"appId":    appID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	resp, err := http.Post(authenticationToken, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", "", err
	}

	name := result["data"].(map[string]interface{})["name"].(string)
	token = result["data"].(map[string]interface{})["token"].(string)

	return name, token, nil
}

// extractRequestToken extracts the request token from the URL
func extractRequestToken(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	values, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return "", err
	}

	return values.Get("request-token"), nil
}

// hashKey creates a SHA256 hash of the key
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ClientParams represents the client parameters
type ClientParams struct {
	// UserID is the user id which is used to login
	UserID string `validate:"required"`

	// Password is the user password
	Password string `validate:"required"`

	// TOTPKey is the TOTP key which is used to generate the TOTP code
	TOTPKey string `validate:"required"`

	// AppID is the app id which is used to generate the access token
	AppID string `validate:"required"`

	// AppSecret is the app secret which is used to generate the access token
	AppSecret string `validate:"required"`
}

// Generates the access token for user using app ID and secret
func GenerateAccessToken(client ClientParams) (string, error) {
	// Validate the client info
	err := validate.Struct(client)
	if err != nil {
		return "", err
	}

	// Step 1 - Retrieve request_key from send_login_otp API
	requestKey, err := sendLogin(client)
	if err != nil {
		return "", fmt.Errorf("send_login_otp failure - %v", err)
	}

	// Step 2 - Generate totp
	totpCode, err := generateTOTP(client.TOTPKey)
	if err != nil {
		return "", fmt.Errorf("generate_totp failure - %v", err)
	}

	// Step 3 - Verify totp and get access token
	session, accessToken, err := verifyTOTP(client, requestKey, totpCode)
	if err != nil {
		return "", fmt.Errorf("verify_totp_result failure - %v", err)
	}

	// Step 4 - Using both we will hit auth API to get the request-token
	redirectURL, err := authTokenAPI(session, accessToken, client.AppID)
	if err != nil {
		return "", fmt.Errorf("auth_tokenAPI failure - %v", err)
	}

	// Step 5 - Extract the request-token from redirectURL
	requestToken, err := extractRequestToken(redirectURL)
	if err != nil {
		return "", fmt.Errorf("extract_request_token failure - %v", err)
	}

	// Step 6 - Making sha256 of appId:appSecret:requestToken
	key := client.AppID + ":" + client.AppSecret + ":" + requestToken
	checkSum := hashKey(key)

	// Step 7 - To create token hit the authenticate API
	_, token, err := authenticateToken(checkSum, requestToken, client.AppID)
	if err != nil {
		return "", fmt.Errorf("authenticate_token failure - %v", err)
	}
	return token, nil
}
