package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const clientUnknown = "unknown"

func initLogger() {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

func setupHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddOnRegisterSession(func(_ context.Context, session server.ClientSession) {
		slog.Info("Client session registered",
			slog.String("session_id", session.SessionID()),
		)
	})

	hooks.AddOnUnregisterSession(func(_ context.Context, session server.ClientSession) {
		slog.Info("Client session unregistered",
			slog.String("session_id", session.SessionID()),
		)
	})

	hooks.AddAfterInitialize(func(ctx context.Context, _ any, message *mcp.InitializeRequest, _ *mcp.InitializeResult) {
		sessionID := ""
		if session := server.ClientSessionFromContext(ctx); session != nil {
			sessionID = session.SessionID()
		}
		slog.Info("Client initialized",
			slog.String("session_id", sessionID),
			slog.String("client_name", message.Params.ClientInfo.Name),
			slog.String("client_version", message.Params.ClientInfo.Version),
			slog.String("protocol_version", message.Params.ProtocolVersion),
		)
	})

	hooks.AddBeforeCallTool(func(ctx context.Context, _ any, message *mcp.CallToolRequest) {
		sessionID := ""
		clientName := clientUnknown
		clientVersion := clientUnknown
		if session := server.ClientSessionFromContext(ctx); session != nil {
			sessionID = session.SessionID()
			if clientInfoSession, ok := session.(server.SessionWithClientInfo); ok {
				info := clientInfoSession.GetClientInfo()
				clientName = info.Name
				clientVersion = info.Version
			}
		}
		slog.Info("Calling tool",
			slog.String("session_id", sessionID),
			slog.String("client_name", clientName),
			slog.String("client_version", clientVersion),
			slog.String("tool_name", message.Params.Name),
			slog.Any("arguments", message.Params.Arguments),
		)
	})

	hooks.AddAfterCallTool(func(ctx context.Context, _ any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		sessionID := ""
		clientName := clientUnknown
		clientVersion := clientUnknown
		if session := server.ClientSessionFromContext(ctx); session != nil {
			sessionID = session.SessionID()
			if clientInfoSession, ok := session.(server.SessionWithClientInfo); ok {
				info := clientInfoSession.GetClientInfo()
				clientName = info.Name
				clientVersion = info.Version
			}
		}

		status := "success"
		if result != nil && result.IsError {
			status = "error"
		}
		slog.Info("Tool call finished",
			slog.String("session_id", sessionID),
			slog.String("client_name", clientName),
			slog.String("client_version", clientVersion),
			slog.String("tool_name", message.Params.Name),
			slog.String("status", status),
		)
	})

	hooks.AddOnError(func(ctx context.Context, _ any, method mcp.MCPMethod, message any, err error) {
		sessionID := ""
		if session := server.ClientSessionFromContext(ctx); session != nil {
			sessionID = session.SessionID()
		}
		slog.Error("MCP error occurred",
			slog.String("session_id", sessionID),
			slog.String("method", string(method)),
			slog.Any("message", message),
			slog.String("error", err.Error()),
		)
	})

	return hooks
}
