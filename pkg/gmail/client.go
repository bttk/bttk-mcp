package gmail

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bttk/bttk-mcp/internal/googleapi"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var (
	// ErrReadSecret is returned when the client secret file cannot be read.
	ErrReadSecret = errors.New("unable to read client secret file")
	// ErrParseConfig is returned when the client secret file cannot be parsed.
	ErrParseConfig = errors.New("unable to parse client secret file to config")
	// ErrClientRetrieve is returned when the Gmail client cannot be retrieved.
	ErrClientRetrieve = errors.New("unable to retrieve Gmail client")
	// ErrListMessages is returned when the messages cannot be listed.
	ErrListMessages = errors.New("unable to list messages")
	// ErrGetMessage is returned when a message cannot be retrieved.
	ErrGetMessage = errors.New("unable to get message")
)

// Client is a wrapper around the Gmail API service.
type Client struct {
	Service *gmail.Service
}

// API defines the interface for interacting with Gmail.
// This allows for mocking in tests.
type API interface {
	SearchMessages(query string) ([]*gmail.Message, error)
	GetMessage(id string) (*gmail.Message, error)
}

// NewClient creates a new Gmail client.
// It handles the OAuth2 flow if a valid token is not found.
func NewClient(credentialsPath, tokenPath string) (*Client, error) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadSecret, err)
	}

	client, err := googleapi.GetClient(b, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseConfig, err)
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClientRetrieve, err)
	}

	return &Client{Service: srv}, nil
}

// SearchMessages searches for messages matching the query.
// It returns a list of simplified message details.
func (c *Client) SearchMessages(query string) ([]*gmail.Message, error) {
	user := "me"
	r, err := c.Service.Users.Messages.List(user).Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListMessages, err)
	}
	return r.Messages, nil
}

// GetMessage retrieves the details of a specific message.
func (c *Client) GetMessage(id string) (*gmail.Message, error) {
	user := "me"
	msg, err := c.Service.Users.Messages.Get(user, id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetMessage, err)
	}
	return msg, nil
}
