package obsidian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Open_File(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/open/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		// Check path
		assert.Equal(t, "/open/my file.md", r.URL.Path)

		// Check query params
		assert.Equal(t, "true", r.URL.Query().Get("newLeaf"))

		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	require.NoError(t, err)

	err = client.Open.File(context.Background(), "my file.md", true)
	require.NoError(t, err)
}
