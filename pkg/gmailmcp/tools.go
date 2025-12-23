package gmailmcp

import (
	"context"
	"fmt"

	"bttk.dev/agent/pkg/gmail"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddTools registers Gmail tools to the MCP server.
func AddTools(s *server.MCPServer, client gmail.GmailAPI) {
	s.AddTool(GmailSearchTool(), GmailSearchHandler(client))
	s.AddTool(GmailReadTool(), GmailReadHandler(client))
}

func GmailSearchTool() mcp.Tool {
	return mcp.NewTool("gmail_search",
		mcp.WithDescription("Search for Gmail messages using a query string."),
		mcp.WithString("query", mcp.Required(), mcp.Description("The search query (e.g., 'from:user@example.com', 'subject:meeting').")),
	)
}

func GmailSearchHandler(client gmail.GmailAPI) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		query, ok := args["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query argument must be a string"), nil
		}

		msgs, err := client.SearchMessages(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search messages: %v", err)), nil
		}

		if len(msgs) == 0 {
			return mcp.NewToolResultText("No messages found matching the query."), nil
		}

		// Simplified summary of results
		var summary string
		for _, msg := range msgs {
			summary += fmt.Sprintf("- ID: %s, ThreadID: %s\n", msg.Id, msg.ThreadId)
		}
		summary += fmt.Sprintf("\nFound %d messages. Use gmail_read with an ID to see content.", len(msgs))

		return mcp.NewToolResultText(summary), nil
	}
}

func GmailReadTool() mcp.Tool {
	return mcp.NewTool("gmail_read",
		mcp.WithDescription("Read the content of a specific Gmail message by ID."),
		mcp.WithString("message_id", mcp.Required(), mcp.Description("The ID of the message to read.")),
	)
}

func GmailReadHandler(client gmail.GmailAPI) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		id, ok := args["message_id"].(string)
		if !ok {
			return mcp.NewToolResultError("message_id argument must be a string"), nil
		}

		msg, err := client.GetMessage(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
		}

		// Extract meaningful content (simplified)
		var content string
		content += fmt.Sprintf("ID: %s\nThreadID: %s\nSnippet: %s\n", msg.Id, msg.ThreadId, msg.Snippet)

		// Headers
		for _, h := range msg.Payload.Headers {
			if h.Name == "Subject" || h.Name == "From" || h.Name == "To" || h.Name == "Date" {
				content += fmt.Sprintf("%s: %s\n", h.Name, h.Value)
			}
		}

		// Body (very basic extraction of text/plain)
		// Accessing parts can be complex in Gmail API, this is a best-effort simple text grab.
		if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
			content += "\n--- Body ---\n" + msg.Payload.Body.Data // This is base64url encoded usually, need to decode if raw
		} else if len(msg.Payload.Parts) > 0 {
			content += "\n--- Parts ---\n"
			for _, p := range msg.Payload.Parts {
				if p.MimeType == "text/plain" && p.Body.Data != "" {
					content += p.Body.Data // Base64 encoded
				}
			}
		}

		return mcp.NewToolResultText(content), nil
	}
}
