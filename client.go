package tiqs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

// Client represents a client with an access token
type Client struct {
	// AccessToken is the access token which is used to
	// authenticate the user
	accessToken string

	// AppID is the app ID which is used to generate the access
	// token
	appID string

	// UserID is the user ID which is used to login
	userID string
}

// New returns a new Client with the given parameters
func New(userID, appID, accessToken string) (*Client) {

	// Return a new client with the app ID and access token
	return &Client{
		accessToken: accessToken,
		appID:       appID,
		userID:      userID,
	}
}
