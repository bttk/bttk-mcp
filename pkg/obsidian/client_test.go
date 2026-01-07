package obsidian

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Vault_List(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", auth)
		fmt.Fprint(w, `{"files": ["a.md", "b.md"]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	files, err := client.Vault.List(context.Background(), "")
	require.NoError(t, err)

	expected := []string{"a.md", "b.md"}
	assert.Equal(t, expected, files)
}

func TestClient_ActiveFile_Patch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "append", r.Header.Get("Operation"))
		assert.Equal(t, "heading", r.Header.Get("Target-Type"))
		assert.Equal(t, "My Heading", r.Header.Get("Target"))
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.ActiveFile.Patch(context.Background(), PatchAppend, TargetHeading, "My Heading", "new content")
	require.NoError(t, err)
}

func TestClient_Search_Simple(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/simple/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		query := r.URL.Query().Get("query")
		assert.Equal(t, "test", query)
		fmt.Fprint(w, `[{"filename": "a.md", "score": 1.0}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	results, err := client.Search.Simple(context.Background(), "test", 100)
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "a.md", results[0].Filename)
}

func TestClient_ActiveFile_Get(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		fmt.Fprint(w, "active file content")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	content, err := client.ActiveFile.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "active file content", content)
}

func TestClient_ActiveFile_GetNote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/vnd.olrapi.note+json", r.Header.Get("Accept"))
		fmt.Fprint(w, `{"content": "note content", "stat": {"size": 100}}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	note, err := client.ActiveFile.GetNote(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "note content", note.Content)
	assert.Equal(t, float64(100), note.Stat.Size)
}

func TestClient_ActiveFile_Append(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/markdown", r.Header.Get("Content-Type"))
		// We could verify body here too if needed
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.ActiveFile.Append(context.Background(), "appended content")
	require.NoError(t, err)
}

func TestClient_ActiveFile_Delete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.ActiveFile.Delete(context.Background())
	require.NoError(t, err)
}

func TestClient_Vault_Get(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/test.md", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		fmt.Fprint(w, "file content")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	content, err := client.Vault.Get(context.Background(), "test.md")
	require.NoError(t, err)
	assert.Equal(t, "file content", content)
}

func TestClient_Vault_GetNote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/test.md", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/vnd.olrapi.note+json", r.Header.Get("Accept"))
		fmt.Fprint(w, `{"content": "note content", "stat": {"size": 200}}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	note, err := client.Vault.GetNote(context.Background(), "test.md")
	require.NoError(t, err)
	assert.Equal(t, "note content", note.Content)
	assert.Equal(t, float64(200), note.Stat.Size)
}

func TestClient_Vault_Create(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/new.md", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "text/markdown", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.Vault.Create(context.Background(), "new.md", "content")
	require.NoError(t, err)
}

func TestClient_Vault_Delete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/todelete.md", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.Vault.Delete(context.Background(), "todelete.md")
	require.NoError(t, err)
}

func TestClient_Search_JsonLogic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/vnd.olrapi.jsonlogic+json", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"filename": "a.md", "result": true}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	query := map[string]interface{}{"var": "test"}
	results, err := client.Search.JSONLogic(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "a.md", results[0].Filename)
}

func TestClient_Search_Dataview(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/vnd.olrapi.dataview.dql+txt", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"filename": "b.md", "result": "some value"}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	results, err := client.Search.Dataview(context.Background(), "TABLE FROM \"folder\"")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "b.md", results[0].Filename)
}
