package obsidianmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"bttk.dev/agent/pkg/obsidian"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Helper to get arguments map
func getArgs(req mcp.CallToolRequest) map[string]interface{} {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return make(map[string]interface{})
	}
	return args
}

// GetActiveFileTool returns the tool definition
func GetActiveFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_get_active_file",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDescription("Get the content of the currently active file in Obsidian"),
	)
}

// GetActiveFileHandler returns the tool handler
func GetActiveFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := client.ActiveFile.GetNote(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get active file: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	}
}

func RegisterGetActiveFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(GetActiveFileTool(), GetActiveFileHandler(client))
}

// AppendActiveFileTool returns the tool definition
func AppendActiveFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_append_active_file",
		mcp.WithDescription("Append content to the currently active file"),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to append")),
	)
}

// AppendActiveFileHandler returns the tool handler
func AppendActiveFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		content, ok := args["content"].(string)
		if !ok {
			return mcp.NewToolResultError("content must be a string"), nil
		}

		if err := client.ActiveFile.Append(ctx, content); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to append to active file: %v", err)), nil
		}
		return mcp.NewToolResultText("Content appended successfully"), nil
	}
}

func RegisterAppendActiveFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(AppendActiveFileTool(), AppendActiveFileHandler(client))
}

// PatchActiveFileTool returns the tool definition
func PatchActiveFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_patch_active_file",
		mcp.WithDescription("Patch the currently active file"),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: append, prepend, replace")),
		mcp.WithString("target_type", mcp.Required(), mcp.Description("Target type: heading, block, frontmatter")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target selector (e.g., heading name)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to patch")),
	)
}

// PatchActiveFileHandler returns the tool handler
func PatchActiveFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		opStr, _ := args["operation"].(string)
		targetTypeStr, _ := args["target_type"].(string)
		target, _ := args["target"].(string)
		content, _ := args["content"].(string)

		if err := client.ActiveFile.Patch(ctx, obsidian.PatchOperation(opStr), obsidian.TargetType(targetTypeStr), target, content); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to patch active file: %v", err)), nil
		}
		return mcp.NewToolResultText("File patched successfully"), nil
	}
}

func RegisterPatchActiveFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(PatchActiveFileTool(), PatchActiveFileHandler(client))
}

// SearchSimpleTool returns the tool definition
func SearchSimpleTool() mcp.Tool {
	return mcp.NewTool("obsidian_search_simple",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDescription("Search the vault for files matching a query"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("context_length", mcp.Description("Length of context to return")),
	)
}

// SearchSimpleHandler returns the tool handler
func SearchSimpleHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		query, _ := args["query"].(string)
		contextLen, _ := args["context_length"].(float64)

		results, err := client.Search.Simple(ctx, query, int(contextLen))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search: %v", err)), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"results": results,
		})
	}
}

func RegisterSearchSimple(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(SearchSimpleTool(), SearchSimpleHandler(client))
}

// SearchJSONLogicTool returns the tool definition
func SearchJSONLogicTool() mcp.Tool {
	return mcp.NewTool("obsidian_search_json_logic",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDescription("Search the vault using JsonLogic"),
		mcp.WithString("query", mcp.Required(), mcp.Description(`JsonLogic query (as a JSON string), e.g. {
  "or": [
    {
      "===": [
        {
          "var": "frontmatter.url"
        },
        "https://myurl.com/some/path/"
      ]
    },
    {
      "glob": [
        {
          "var": "frontmatter.url-glob"
        },
        "https://myurl.com/some/path/"
      ]
    }
  ]
}`)),
	)
}

// SearchJSONLogicHandler returns the tool handler
func SearchJSONLogicHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		queryStr, _ := args["query"].(string)

		var query interface{}
		if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid JSON logic query: %v", err)), nil
		}

		results, err := client.Search.JsonLogic(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search: %v", err)), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"results": results,
		})
	}
}

func RegisterSearchJSONLogic(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(SearchJSONLogicTool(), SearchJSONLogicHandler(client))
}

// GetDailyNoteTool returns the tool definition
func GetDailyNoteTool() mcp.Tool {
	return mcp.NewTool("obsidian_get_daily_note",
		mcp.WithDescription("Get the content of today's daily note"),
	)
}

// GetDailyNoteHandler returns the tool handler
func GetDailyNoteHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := client.Periodic.GetCurrentNote(ctx, "daily")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get daily note: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	}
}

func RegisterGetDailyNote(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(GetDailyNoteTool(), GetDailyNoteHandler(client))
}

// GetFileTool returns the tool definition
func GetFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_get_file",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDescription("Get the content of a specific file in the vault"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
	)
}

// GetFileHandler returns the tool handler
func GetFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		content, err := client.Vault.GetNote(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get file: %v", err)), nil
		}
		return mcp.NewToolResultJSON(content)
	}
}

func RegisterGetFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(GetFileTool(), GetFileHandler(client))
}

// ListFilesTool returns the tool definition
func ListFilesTool() mcp.Tool {
	return mcp.NewTool("obsidian_list_files",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDescription("List files in a directory"),
		mcp.WithString("path", mcp.Description("Directory path (empty for root)")),
	)
}

// ListFilesHandler returns the tool handler
func ListFilesHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		files, err := client.Vault.List(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list files: %v", err)), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"files": files,
		})
	}
}

func RegisterListFiles(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(ListFilesTool(), ListFilesHandler(client))
}

// CreateOrUpdateFileTool returns the tool definition
func CreateOrUpdateFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_create_or_update_file",
		mcp.WithDescription("Create a new file or update an existing one"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content of the file")),
	)
}

// CreateOrUpdateFileHandler returns the tool handler
func CreateOrUpdateFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)

		err := client.Vault.Create(ctx, path, content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create/update file: %v", err)), nil
		}
		return mcp.NewToolResultText("File created/updated successfully"), nil
	}
}

func RegisterCreateOrUpdateFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(CreateOrUpdateFileTool(), CreateOrUpdateFileHandler(client))
}

// OpenFileTool returns the tool definition
func OpenFileTool() mcp.Tool {
	return mcp.NewTool("obsidian_open_file",
		mcp.WithDescription("Open a file in Obsidian UI"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
		mcp.WithBoolean("new_leaf", mcp.Description("Open in a new leaf (tab)")),
	)
}

// OpenFileHandler returns the tool handler
func OpenFileHandler(client *obsidian.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		path, _ := args["path"].(string)
		newLeaf, _ := args["new_leaf"].(bool)

		err := client.Open.File(ctx, path, newLeaf)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to open file: %v", err)), nil
		}
		return mcp.NewToolResultText("File opened successfully"), nil
	}
}

func RegisterOpenFile(s *server.MCPServer, client *obsidian.Client) {
	s.AddTool(OpenFileTool(), OpenFileHandler(client))
}
