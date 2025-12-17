package obsidian

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClient_Vault_List(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vault/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got '%s'", auth)
		}
		fmt.Fprint(w, `{"files": ["a.md", "b.md"]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	files, err := client.Vault.List(context.Background(), "")
	if err != nil {
		t.Fatalf("Vault.List failed: %v", err)
	}

	expected := []string{"a.md", "b.md"}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("expected %v, got %v", expected, files)
	}
}

func TestClient_ActiveFile_Patch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/active/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.Header.Get("Operation") != "append" {
			t.Errorf("expected Operation header 'append', got '%s'", r.Header.Get("Operation"))
		}
		if r.Header.Get("Target-Type") != "heading" {
			t.Errorf("expected Target-Type header 'heading', got '%s'", r.Header.Get("Target-Type"))
		}
		if r.Header.Get("Target") != "My Heading" {
			t.Errorf("expected Target header 'My Heading', got '%s'", r.Header.Get("Target"))
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ActiveFile.Patch(context.Background(), PatchAppend, TargetHeading, "My Heading", "new content")
	if err != nil {
		t.Fatalf("ActiveFile.Patch failed: %v", err)
	}
}

func TestClient_Search_Simple(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/simple/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		query := r.URL.Query().Get("query")
		if query != "test" {
			t.Errorf("expected query param 'test', got '%s'", query)
		}
		fmt.Fprint(w, `[{"filename": "a.md", "score": 1.0}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	results, err := client.Search.Simple(context.Background(), "test", 100)
	if err != nil {
		t.Fatalf("Search.Simple failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Filename != "a.md" {
		t.Errorf("expected filename 'a.md', got '%s'", results[0].Filename)
	}
}
