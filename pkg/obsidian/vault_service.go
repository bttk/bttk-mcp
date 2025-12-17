package obsidian

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// VaultService handles interaction with files in the vault.
type VaultService struct {
	client *Client
}

// List lists files in the root directory (if path is empty) or a specified directory.
func (s *VaultService) List(ctx context.Context, path string) ([]string, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "vault/" + path})
	// If path is a directory, it ensures trailing slash usually, but API might be flexible.
	// The API doc says /vault/{pathToDirectory}/ for directory listing.
	// If path is empty, it uses /vault/

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Files []string `json:"files"`
	}
	err = s.client.do(req, &resp)
	return resp.Files, err
}

// Get returns the content of a file in the vault.
func (s *VaultService) Get(ctx context.Context, path string) (string, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "vault/" + path})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var content string
	err = s.client.do(req, &content)
	return content, err
}

// GetNote returns the file parsed as a Note struct.
func (s *VaultService) GetNote(ctx context.Context, path string) (*Note, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "vault/" + path})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.olrapi.note+json")

	var note Note
	err = s.client.do(req, &note)
	return &note, err
}

// Create creates a new file or updates an existing one with the given content.
func (s *VaultService) Create(ctx context.Context, path, content string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "vault/" + path})
	req, err := http.NewRequestWithContext(ctx, "PUT", u.String(), strings.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/markdown")

	return s.client.do(req, nil)
}

// Delete deletes a file in the vault.
func (s *VaultService) Delete(ctx context.Context, path string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "vault/" + path})
	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}
