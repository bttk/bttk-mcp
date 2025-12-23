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
