package obsidianmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bttk.dev/agent/pkg/obsidian"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockServer creates a mock Obsidian REST API server
func setupMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *obsidian.Client) {
	ts := httptest.NewServer(handler)
	client, err := obsidian.NewClient(ts.URL, "test-token", obsidian.WithInsecureTLS())
	require.NoError(t, err, "Failed to create client")
	return ts, client
}

func TestGetActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/active/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "This is the active file content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetActiveFileTool(),
		Handler: GetActiveFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_get_active_file",
		},
	})
	require.NoError(t, err)

	assert.NotEmpty(t, res.Content, "Expected content, got empty")

	logMsg(t, res)
}

func TestAppendActiveFile(t *testing.T) {
	expectedContent := "New line"
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/active/", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    AppendActiveFileTool(),
		Handler: AppendActiveFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_append_active_file",
			Arguments: map[string]interface{}{
				"content": expectedContent,
			},
		},
	})
	require.NoError(t, err)
}

func TestPatchActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/active/", r.URL.Path)
		assert.Equal(t, "append", r.Header.Get("Operation"))
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    PatchActiveFileTool(),
		Handler: PatchActiveFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_patch_active_file",
			Arguments: map[string]interface{}{
				"operation":   "append",
				"target_type": "heading",
				"target":      "MyHeading",
				"content":     "New Content",
			},
		},
	})
	require.NoError(t, err)
}

func TestSearchSimple(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/search/simple/", r.URL.Path)
		assert.Equal(t, "test query", r.URL.Query().Get("query"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md", "score": 1.0}]`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    SearchSimpleTool(),
		Handler: SearchSimpleHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_search_simple",
			Arguments: map[string]interface{}{
				"query": "test query",
			},
		},
	})
	require.NoError(t, err)

	logMsg(t, res)
}

func TestSearchJSONLogic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/search/", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md"}]`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    SearchJSONLogicTool(),
		Handler: SearchJSONLogicHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	query := `{"or": [{"===": [{"var": "frontmatter.url"}, "http://example.com"]}]}`

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_search_json_logic",
			Arguments: map[string]interface{}{
				"query": query,
			},
		},
	})
	require.NoError(t, err)
}

func TestGetDailyNote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/periodic/daily/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "Daily note content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetDailyNoteTool(),
		Handler: GetDailyNoteHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_get_daily_note",
		},
	})
	require.NoError(t, err)
}

func TestGetFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		// Assuming implementation uses path escaping of some sort, or direct concatenation
		assert.Equal(t, "/vault/folder/file.md", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "File content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetFileTool(),
		Handler: GetFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_get_file",
			Arguments: map[string]interface{}{
				"path": "folder/file.md",
			},
		},
	})
	require.NoError(t, err)
}

func TestListFiles(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/vault/folder", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"files": ["file1.md", "file2.md"]}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    ListFilesTool(),
		Handler: ListFilesHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_list_files",
			Arguments: map[string]interface{}{
				"path": "folder",
			},
		},
	})
	require.NoError(t, err)
}

func TestCreateOrUpdateFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/vault/new/file.md", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CreateOrUpdateFileTool(),
		Handler: CreateOrUpdateFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_create_or_update_file",
			Arguments: map[string]interface{}{
				"path":    "new/file.md",
				"content": "New content",
			},
		},
	})
	require.NoError(t, err)
}

func TestOpenFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/open/my/file.md", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    OpenFileTool(),
		Handler: OpenFileHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "obsidian_open_file",
			Arguments: map[string]interface{}{
				"path": "my/file.md",
			},
		},
	})
	require.NoError(t, err)
}

func logMsg(t *testing.T, res *mcp.CallToolResult) {
	assert.False(t, res.IsError, "Tool returned error")

	// For debugging, we can print the content
	for _, c := range res.Content {
		if text, ok := c.(mcp.TextContent); ok {
			t.Logf("Content: %s", text.Text)
		}
	}

	// Check StructuredContent serialization if present
	if res.StructuredContent != nil {
		jsonContent, err := json.Marshal(res.StructuredContent)
		require.NoError(t, err, "Failed to marshal StructuredContent")

		// Verify it matches expected JSON object structure
		assert.Equal(t, byte('{'), jsonContent[0], "Expected JSON object")
	}
}
