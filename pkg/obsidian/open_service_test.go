package obsidian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Open_File(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/open/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Check path
		if r.URL.Path != "/open/my file.md" {
			t.Errorf("expected path /open/my file.md, got %s", r.URL.Path)
		}

		// Check query params
		if r.URL.Query().Get("newLeaf") != "true" {
			t.Errorf("expected newLeaf=true, got %s", r.URL.Query().Get("newLeaf"))
		}

		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Open.File(context.Background(), "my file.md", true)
	if err != nil {
		t.Fatalf("Open.File failed: %v", err)
	}
}
