package gmailmcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bttk/bttk-mcp/pkg/gmail"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gmailv1 "google.golang.org/api/gmail/v1"
)

const (
	defaultMaxResults   = 50
	defaultMaxBodyBytes = 10000
)

// AddTools registers Gmail tools to the MCP server.
func AddTools(s *server.MCPServer, client gmail.API) {
	s.AddTool(GmailSearchTool(), GmailSearchHandler(client))
	s.AddTool(GmailReadTool(), GmailReadHandler(client))
}

func GmailSearchTool() mcp.Tool {
	return mcp.NewTool("gmail_search",
		mcp.WithDescription("Search for Gmail messages using a query string."),
		mcp.WithString("query", mcp.Required(), mcp.Description("The search query (e.g., 'from:user@example.com', 'subject:meeting').")),
		mcp.WithNumber("maxResults", mcp.Description("Maximum number of results to return (default 50).")),
	)
}

func GmailSearchHandler(client gmail.API) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		query, ok := args["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query argument must be a string"), nil
		}

		maxResults := int64(defaultMaxResults)
		if mr, ok := args["maxResults"].(float64); ok {
			maxResults = int64(mr)
		}

		msgs, err := client.SearchMessages(query, maxResults)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search messages: %v", err)), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"messages": msgs,
			"count":    len(msgs),
		})
	}
}

func GmailReadTool() mcp.Tool {
	return mcp.NewTool("gmail_read",
		mcp.WithDescription("Read the content of a specific Gmail message by ID."),
		mcp.WithString("messageId", mcp.Required(), mcp.Description("The ID of the message to read.")),
		mcp.WithNumber("maxBodyBytes", mcp.Description("Maximum bytes of body content to return (default 10000).")),
	)
}

func GmailReadHandler(client gmail.API) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		id, ok := args["messageId"].(string)
		if !ok {
			return mcp.NewToolResultError("messageId argument must be a string"), nil
		}

		maxBodyBytes := defaultMaxBodyBytes
		if mbb, ok := args["maxBodyBytes"].(float64); ok {
			maxBodyBytes = int(mbb)
		}

		msg, err := client.GetMessage(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
		}

		currentBytes := 0
		processPart(msg.Payload, maxBodyBytes, &currentBytes)

		return mcp.NewToolResultJSON(msg)
	}
}

func processPart(part *gmailv1.MessagePart, maxBytes int, currentBytes *int) {
	if part == nil {
		return
	}

	if part.Body != nil && part.Body.Data != "" {
		processPartBody(part, maxBytes, currentBytes)
	}

	for _, p := range part.Parts {
		processPart(p, maxBytes, currentBytes)
	}
}

func processPartBody(part *gmailv1.MessagePart, maxBytes int, currentBytes *int) {
	isText := strings.Contains(part.MimeType, "text/plain") || strings.Contains(part.MimeType, "text/html")
	if !isText {
		// Non-text message parts: remove data but leave metadata
		part.Body.Data = ""
		return
	}

	// Gmail uses base64url encoding
	data, err := base64.URLEncoding.DecodeString(part.Body.Data)
	if err != nil {
		// Try raw if standard fails (sometimes padding is missing)
		data, err = base64.RawURLEncoding.DecodeString(part.Body.Data)
	}
	if err != nil {
		return
	}

	remaining := maxBytes - *currentBytes
	switch {
	case remaining <= 0:
		part.Body.Data = ""
	case len(data) > remaining:
		part.Body.Data = string(data[:remaining]) + "... [TRUNCATED]"
		*currentBytes = maxBytes
	default:
		part.Body.Data = string(data)
		*currentBytes += len(data)
	}
}
