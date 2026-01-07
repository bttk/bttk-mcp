package gmailmcp

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

var errMessageNotFound = errors.New("message not found")

// MockGmailClient is a mock implementation of pkg_gmail.GmailAPI
type MockGmailClient struct {
	SearchMessagesFunc func(query string) ([]*gmail.Message, error)
	GetMessageFunc     func(id string) (*gmail.Message, error)
}

func (m *MockGmailClient) SearchMessages(query string) ([]*gmail.Message, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(query)
	}
	return nil, nil
}

func (m *MockGmailClient) GetMessage(id string) (*gmail.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(id)
	}
	return nil, nil
}

func TestGmailSearch(t *testing.T) {
	mockClient := &MockGmailClient{
		SearchMessagesFunc: func(query string) ([]*gmail.Message, error) {
			if query == "test" {
				return []*gmail.Message{
					{Id: "123", ThreadId: "t123"},
					{Id: "124", ThreadId: "t124"},
				}, nil
			}
			return []*gmail.Message{}, nil
		},
	}

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GmailSearchTool(),
		Handler: GmailSearchHandler(mockClient),
	})
	assert.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "gmail_search",
			Arguments: map[string]interface{}{
				"query": "test",
			},
		},
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError, "Tool result should not be an error")

	// Check output
	var content string
	for _, c := range res.Content {
		if text, ok := c.(mcp.TextContent); ok {
			content += text.Text
		}
	}

	expectedSnippet := "Found 2 messages"
	assert.NotEmpty(t, content)
	assert.Contains(t, content, expectedSnippet)
}

func TestGmailRead(t *testing.T) {
	mockClient := &MockGmailClient{
		GetMessageFunc: func(id string) (*gmail.Message, error) {
			if id == "123" {
				return &gmail.Message{
					Id:       "123",
					ThreadId: "t123",
					Snippet:  "Hello world",
					Payload: &gmail.MessagePart{
						Headers: []*gmail.MessagePartHeader{
							{Name: "Subject", Value: "Test Email"},
							{Name: "From", Value: "sender@example.com"},
						},
						Body: &gmail.MessagePartBody{
							Data: "This is the body",
						},
					},
				}, nil
			}
			return nil, errMessageNotFound
		},
	}

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GmailReadTool(),
		Handler: GmailReadHandler(mockClient),
	})
	assert.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "gmail_read",
			Arguments: map[string]interface{}{
				"message_id": "123",
			},
		},
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError, "Tool result should not be an error")

	// Check output
	var content string
	for _, c := range res.Content {
		if text, ok := c.(mcp.TextContent); ok {
			content += text.Text
		}
	}

	// Quick checks
	checks := []string{"Subject: Test Email", "From: sender@example.com", "This is the body"}
	for _, check := range checks {
		assert.Contains(t, content, check)
	}
}
