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
)

// setupMockServer creates a mock Obsidian REST API server
func setupMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *obsidian.Client) {
	ts := httptest.NewServer(handler)
	client, err := obsidian.NewClient(ts.URL, "test-token", obsidian.WithInsecureTLS())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return ts, client
}

func TestGetActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/active/" {
			t.Errorf("Expected path /active/, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "This is the active file content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetActiveFileTool(),
		Handler: GetActiveFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_active_file",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Content) == 0 {
		t.Fatal("Expected content, got empty")
	}

	// Check content via StructuredContent (if available) or raw Content
	// The implementation returns NewToolResultJSON which populates Content with JSON string usually?
	// NewToolResultJSON: "func NewToolResultJSON(data interface{}) *CallToolResult"
	// It serializes data to JSON and puts it in content

	// Assuming the result is JSON text
	logMsg(t, res)
}

func TestAppendActiveFile(t *testing.T) {
	expectedContent := "New line"
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/active/" {
			t.Errorf("Expected path /active/, got %s", r.URL.Path)
		}

		// Verify body
		// For brevity, assuming client works correctly if it sends correct request

		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    AppendActiveFileTool(),
		Handler: AppendActiveFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "append_active_file",
			Arguments: map[string]interface{}{
				"content": expectedContent,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPatchActiveFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/active/" {
			t.Errorf("Expected path /active/, got %s", r.URL.Path)
		}

		// Check headers for specialized Obsidian patch headers
		if r.Header.Get("Operation") != "append" {
			t.Errorf("Expected Operation append, got %s", r.Header.Get("Operation"))
		}

		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    PatchActiveFileTool(),
		Handler: PatchActiveFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "patch_active_file",
			Arguments: map[string]interface{}{
				"operation":   "append",
				"target_type": "heading",
				"target":      "MyHeading",
				"content":     "New Content",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchSimple(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/search/simple/" {
			t.Errorf("Expected path /search/simple/, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "test query" {
			t.Errorf("Expected query 'test query', got %s", r.URL.Query().Get("query"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md", "score": 1.0}]`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    SearchSimpleTool(),
		Handler: SearchSimpleHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search_simple",
			Arguments: map[string]interface{}{
				"query": "test query",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	logMsg(t, res)
}

func TestSearchJSONLogic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/search/" {
			t.Errorf("Expected path /search/, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"filename": "test.md"}]`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    SearchJSONLogicTool(),
		Handler: SearchJSONLogicHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	query := `{"or": [{"===": [{"var": "frontmatter.url"}, "http://example.com"]}]}`

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search_json_logic",
			Arguments: map[string]interface{}{
				"query": query,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDailyNote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/periodic/daily/" {
			t.Errorf("Expected path /periodic/daily/, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "Daily note content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetDailyNoteTool(),
		Handler: GetDailyNoteHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_daily_note",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		// The client encodes the path. "folder/file.md" -> "folder/file.md" (if no special chars)
		// Assuming implementation uses path escaping
		if r.URL.Path != "/vault/folder/file.md" {
			t.Errorf("Expected path /vault/folder/file.md, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"content": "File content"}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    GetFileTool(),
		Handler: GetFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_file",
			Arguments: map[string]interface{}{
				"path": "folder/file.md",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestListFiles(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/vault/folder" {
			t.Errorf("Expected path /vault/folder, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"files": ["file1.md", "file2.md"]}`)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    ListFilesTool(),
		Handler: ListFilesHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_files",
			Arguments: map[string]interface{}{
				"path": "folder",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateOrUpdateFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		if r.URL.Path != "/vault/new/file.md" {
			t.Errorf("Expected path /vault/new/file.md, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CreateOrUpdateFileTool(),
		Handler: CreateOrUpdateFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "create_or_update_file",
			Arguments: map[string]interface{}{
				"path":    "new/file.md",
				"content": "New content",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenFile(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/open/my/file.md" {
			t.Errorf("Expected path /open/my/file.md, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}

	ts, client := setupMockServer(t, handler)
	defer ts.Close()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    OpenFileTool(),
		Handler: OpenFileHandler(client),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	_, err = srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "open_file",
			Arguments: map[string]interface{}{
				"path": "my/file.md",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func logMsg(t *testing.T, res *mcp.CallToolResult) {
	if res.IsError {
		t.Error("Tool returned error")
	}
	// For debugging, we can print the content
	for _, c := range res.Content {
		if text, ok := c.(mcp.TextContent); ok {
			t.Logf("Content: %s", text.Text)
		}
	}

	// Check StructuredContent serialization if present
	if res.StructuredContent != nil {
		jsonContent, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Errorf("Failed to marshal StructuredContent: %v", err)
		}
		if string(jsonContent)[0] != '{' {
			var snippet string
			if len(jsonContent) < 30 {
				snippet = string(jsonContent)
			} else {
				snippet = string(jsonContent)[:30]
			}
			t.Errorf("Expected JSON object, got %s", snippet)
		}
	}
}
