package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"bttk.dev/agent/pkg/obsidian"
	"bttk.dev/agent/pkg/obsidian/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.json", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("failed to load %s: %v", configPath, err)
		os.Exit(1)
	}

	// Initialize Obsidian Client
	client, err := obsidian.NewClient(cfg.Obsidian.URL, cfg.Obsidian.APIKey, obsidian.WithInsecureTLS())
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// Create MCP Server
	s := server.NewMCPServer(
		"Obsidian MCP Server",
		"1.0.0",
	)

	// Helper to get arguments map
	getArgs := func(req mcp.CallToolRequest) map[string]interface{} {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return make(map[string]interface{})
		}
		return args
	}

	// Tool: get_active_file
	s.AddTool(mcp.NewTool("get_active_file",
		mcp.WithDescription("Get the content of the currently active file in Obsidian"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := client.ActiveFile.GetNote(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get active file: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	})

	// Tool: append_active_file
	s.AddTool(mcp.NewTool("append_active_file",
		mcp.WithDescription("Append content to the currently active file"),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to append")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		content, ok := args["content"].(string)
		if !ok {
			return mcp.NewToolResultError("content must be a string"), nil
		}

		if err := client.ActiveFile.Append(ctx, content); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to append to active file: %v", err)), nil
		}
		return mcp.NewToolResultText("Content appended successfully"), nil
	})

	// Tool: patch_active_file
	s.AddTool(mcp.NewTool("patch_active_file",
		mcp.WithDescription("Patch the currently active file"),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: append, prepend, replace")),
		mcp.WithString("target_type", mcp.Required(), mcp.Description("Target type: heading, block, frontmatter")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target selector (e.g., heading name)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to patch")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		opStr, _ := args["operation"].(string)
		targetTypeStr, _ := args["target_type"].(string)
		target, _ := args["target"].(string)
		content, _ := args["content"].(string)

		if err := client.ActiveFile.Patch(ctx, obsidian.PatchOperation(opStr), obsidian.TargetType(targetTypeStr), target, content); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to patch active file: %v", err)), nil
		}
		return mcp.NewToolResultText("File patched successfully"), nil
	})

	// Tool: search
	s.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search the vault for files matching a query"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("context_length", mcp.Description("Length of context to return")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		query, _ := args["query"].(string)
		contextLen, _ := args["context_length"].(float64)

		results, err := client.Search.Simple(ctx, query, int(contextLen))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search: %v", err)), nil
		}

		var resultStr string
		for _, r := range results {
			resultStr += fmt.Sprintf("File: %s (Score: %.2f)\n", r.Filename, r.Score)
			for _, m := range r.Matches {
				resultStr += fmt.Sprintf("  Context: %s\n", m.Context)
			}
			resultStr += "\n"
		}
		if resultStr == "" {
			resultStr = "No matches found."
		}

		return mcp.NewToolResultText(resultStr), nil
	})

	// Tool: get_daily_note
	s.AddTool(mcp.NewTool("get_daily_note",
		mcp.WithDescription("Get the content of today's daily note"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := client.Periodic.GetCurrentNote(ctx, "daily")
		if err != nil {
			// Daily note might not exist, but let's report error
			return mcp.NewToolResultError(fmt.Sprintf("failed to get daily note: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	})

	// Tool: get_file
	s.AddTool(mcp.NewTool("get_file",
		mcp.WithDescription("Get the content of a specific file in the vault"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		content, err := client.Vault.GetNote(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get file: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	})

	// Tool: list_files
	s.AddTool(mcp.NewTool("list_files",
		mcp.WithDescription("List files in a directory"),
		mcp.WithString("path", mcp.Description("Directory path (empty for root)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		files, err := client.Vault.List(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list files: %v", err)), nil
		}

		var resultStr string
		for _, f := range files {
			resultStr += f + "\n"
		}
		return mcp.NewToolResultText(resultStr), nil
	})

	// Tool: create_or_update_file
	s.AddTool(mcp.NewTool("create_or_update_file",
		mcp.WithDescription("Create a new file or update an existing one"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content of the file")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)

		err := client.Vault.Create(ctx, path, content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create/update file: %v", err)), nil
		}
		return mcp.NewToolResultText("File created/updated successfully"), nil
	})

	// Tool: open_file
	s.AddTool(mcp.NewTool("open_file",
		mcp.WithDescription("Open a file in Obsidian UI"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithBoolean("new_leaf", mcp.Description("Open in a new leaf (tab)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		newLeaf, _ := args["new_leaf"].(bool)

		err := client.Open.File(ctx, path, newLeaf)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to open file: %v", err)), nil
		}
		return mcp.NewToolResultText("File opened successfully"), nil
	})

	// Start the server using Stdio
	if err := server.ServeStdio(s); err != nil {
		log.Printf("Server error: %v", err)
	}
}
