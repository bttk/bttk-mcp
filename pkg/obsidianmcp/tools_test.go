package obsidianmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bttk/bttk-mcp/pkg/obsidian"
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

func testTool(t *testing.T, tool mcp.Tool, createHandler func(*obsidian.Client) server.ToolHandlerFunc, toolName string, args map[string]interface{}, mockHandler http.HandlerFunc) *mcp.CallToolResult {
	ts, client := setupMockServer(t, mockHandler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    tool,
		Handler: createHandler(client),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	require.NoError(t, err)
	return res
}

func TestGetActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/active/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "This is the active file content"}`)
	}

	res := testTool(t, GetActiveFileTool(), GetActiveFileHandler, "obsidian_get_active_file", nil, handler)
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

	testTool(t, AppendActiveFileTool(), AppendActiveFileHandler, "obsidian_append_active_file", map[string]interface{}{
		"content": expectedContent,
	}, handler)
}

func TestPatchActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/active/", r.URL.Path)
		assert.Equal(t, "append", r.Header.Get("Operation"))
		w.WriteHeader(http.StatusOK)
	}

	testTool(t, PatchActiveFileTool(), PatchActiveFileHandler, "obsidian_patch_active_file", map[string]interface{}{
		"operation":   "append",
		"target_type": "heading",
		"target":      "MyHeading",
		"content":     "New Content",
	}, handler)
}

func TestSearchSimple(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/search/simple/", r.URL.Path)
		assert.Equal(t, "test query", r.URL.Query().Get("query"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md", "score": 1.0}]`)
	}

	res := testTool(t, SearchSimpleTool(), SearchSimpleHandler, "obsidian_search_simple", map[string]interface{}{
		"query": "test query",
	}, handler)
	logMsg(t, res)
}

func TestSearchJSONLogic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/search/", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md"}]`)
	}

	query := `{"or": [{"===": [{"var": "frontmatter.url"}, "http://example.com"]}]}`

	testTool(t, SearchJSONLogicTool(), SearchJSONLogicHandler, "obsidian_search_json_logic", map[string]interface{}{
		"query": query,
	}, handler)
}

func TestGetDailyNote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/periodic/daily/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "Daily note content"}`)
	}

	testTool(t, GetDailyNoteTool(), GetDailyNoteHandler, "obsidian_get_daily_note", nil, handler)
}

func TestGetFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		// Assuming implementation uses path escaping of some sort, or direct concatenation
		assert.Equal(t, "/vault/folder/file.md", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "File content"}`)
	}

	testTool(t, GetFileTool(), GetFileHandler, "obsidian_get_file", map[string]interface{}{
		"path": "folder/file.md",
	}, handler)
}

func TestListFiles(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/vault/folder", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"files": ["file1.md", "file2.md"]}`)
	}

	testTool(t, ListFilesTool(), ListFilesHandler, "obsidian_list_files", map[string]interface{}{
		"path": "folder",
	}, handler)
}

func TestCreateOrUpdateFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/vault/new/file.md", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}

	testTool(t, CreateOrUpdateFileTool(), CreateOrUpdateFileHandler, "obsidian_create_or_update_file", map[string]interface{}{
		"path":    "new/file.md",
		"content": "New content",
	}, handler)
}

func TestOpenFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/open/my/file.md", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}

	testTool(t, OpenFileTool(), OpenFileHandler, "obsidian_open_file", map[string]interface{}{
		"path": "my/file.md",
	}, handler)
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
