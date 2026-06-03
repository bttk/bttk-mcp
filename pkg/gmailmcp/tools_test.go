package gmailmcp

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/bttk/bttk-mcp/pkg/gmail"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	gmailv1 "google.golang.org/api/gmail/v1"
)

var errMessageNotFound = errors.New("message not found")

// MockGmailClient is a mock implementation of pkg_gmail.GmailAPI
type MockGmailClient struct {
	SearchMessagesFunc func(ctx context.Context, query string, maxResults int64) ([]*gmail.Message, error)
	GetMessageFunc     func(ctx context.Context, id string) (*gmailv1.Message, error)
}

func (m *MockGmailClient) SearchMessages(ctx context.Context, query string, maxResults int64) ([]*gmail.Message, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(ctx, query, maxResults)
	}
	return nil, nil
}

func (m *MockGmailClient) GetMessage(ctx context.Context, id string) (*gmailv1.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, id)
	}
	return nil, nil
}

func TestGmailSearch(t *testing.T) {
	mockClient := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, query string, _ int64) ([]*gmail.Message, error) {
			if query == "test" {
				return []*gmail.Message{
					{
						ID:       "123",
						ThreadID: "t123",
						Snippet:  "Verification code...",
						To:       "me@example.com",
						From:     "noreply@google.com",
					},
					{ID: "124", ThreadID: "t124"},
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
	assert.Contains(t, text.Text, `"snippet":"Verification code..."`)
	assert.Contains(t, text.Text, `"to":"me@example.com"`)
	assert.Contains(t, text.Text, `"from":"noreply@google.com"`)
}

func TestGmailRead(t *testing.T) {
	mockClient := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, id string) (*gmailv1.Message, error) {
			if id == "123" {
				return &gmailv1.Message{
					Id:       "123",
					ThreadId: "t123",
					Snippet:  "Hello world",
					Payload: &gmailv1.MessagePart{
						MimeType: "text/plain",
						Headers: []*gmailv1.MessagePartHeader{
							{Name: "Subject", Value: "Test Email"},
							{Name: "From", Value: "sender@example.com"},
						},
						Body: &gmailv1.MessagePartBody{
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
