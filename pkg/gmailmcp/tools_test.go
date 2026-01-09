package gmailmcp

import (
	"context"
	"encoding/base64"
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
	SearchMessagesFunc func(query string, maxResults int64) ([]*gmail.Message, error)
	GetMessageFunc     func(id string) (*gmail.Message, error)
}

func (m *MockGmailClient) SearchMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(query, maxResults)
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
		SearchMessagesFunc: func(query string, _ int64) ([]*gmail.Message, error) {
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
	assert.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)
	assert.Contains(t, text.Text, `"count":2`)
	assert.Contains(t, text.Text, `"id":"123"`)
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
						MimeType: "text/plain",
						Headers: []*gmail.MessagePartHeader{
							{Name: "Subject", Value: "Test Email"},
							{Name: "From", Value: "sender@example.com"},
						},
						Body: &gmail.MessagePartBody{
							Data: base64.URLEncoding.EncodeToString([]byte("This is the decoded body content.")),
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
				"messageId": "123",
			},
		},
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError, "Tool result should not be an error")

	// Check output
	assert.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	assert.True(t, ok)

	// Quick checks
	checks := []string{`"id":"123"`, `"snippet":"Hello world"`, `"Test Email"`, `"sender@example.com"`, `"This is the decoded body content."`}
	for _, check := range checks {
		assert.Contains(t, text.Text, check)
	}

	// Test truncation
	res, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "gmail_read",
			Arguments: map[string]interface{}{
				"messageId":    "123",
				"maxBodyBytes": 10,
			},
		},
	})
	assert.NoError(t, err)
	text, _ = res.Content[0].(mcp.TextContent)
	assert.Contains(t, text.Text, "This is th... [TRUNCATED]")
}
