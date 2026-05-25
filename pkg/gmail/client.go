package gmail

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

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

// Message represents a simplified Gmail message.
type Message struct {
	ID       string `json:"id,omitempty"`
	ThreadID string `json:"threadId,omitempty"`
	Snippet  string `json:"snippet,omitempty"`
	Date     string `json:"date,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Cc       string `json:"cc,omitempty"`
}

// API defines the interface for interacting with Gmail.
// This allows for mocking in tests.
type API interface {
	SearchMessages(query string, maxResults int64) ([]*Message, error)
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
func (c *Client) SearchMessages(query string, maxResults int64) ([]*Message, error) {
	user := "me"
	r, err := c.Service.Users.Messages.List(user).Q(query).MaxResults(maxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListMessages, err)
	}

	messages := make([]*Message, 0, len(r.Messages))
	for _, msg := range r.Messages {
		m, err := c.Service.Users.Messages.Get(user, msg.Id).
			Format("metadata").
			MetadataHeaders("To", "From", "Subject", "Date", "Cc").
			Fields("id", "threadId", "snippet", "payload(headers)").
			Do()
		if err != nil {
			log.Printf("failed to get message details for ID %s: %v", msg.Id, err)
			continue
		}

		var date, from, to, subject, cc string
		if m.Payload != nil {
			for _, h := range m.Payload.Headers {
				switch strings.ToLower(h.Name) {
				case "date":
					date = h.Value
				case "from":
					from = h.Value
				case "to":
					to = h.Value
				case "subject":
					subject = h.Value
				case "cc":
					cc = h.Value
				}
			}
		}

		messages = append(messages, &Message{
			ID:       m.Id,
			ThreadID: m.ThreadId,
			Snippet:  m.Snippet,
			Date:     date,
			From:     from,
			To:       to,
			Subject:  subject,
			Cc:       cc,
		})
	}
	return messages, nil
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
