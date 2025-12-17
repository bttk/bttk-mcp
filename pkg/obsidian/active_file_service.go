package obsidian

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// ActiveFileService handles interaction with the currently active file in Obsidian.
type ActiveFileService struct {
	client *Client
}

// Get returns the content of the currently active file as a string.
func (s *ActiveFileService) Get(ctx context.Context) (string, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "active/"})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var content string
	err = s.client.do(req, &content)
	return content, err
}

// GetNote returns the active file parsed as a Note struct (including frontmatter and stats).
// This sends the Accept: application/vnd.olrapi.note+json header.
func (s *ActiveFileService) GetNote(ctx context.Context) (*Note, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "active/"})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.olrapi.note+json")

	var note Note
	err = s.client.do(req, &note)
	return &note, err
}

// Append appends content to the end of the currently active file.
func (s *ActiveFileService) Append(ctx context.Context, content string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "active/"})
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), strings.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/markdown")

	return s.client.do(req, nil)
}

// Delete deletes the currently active file.
func (s *ActiveFileService) Delete(ctx context.Context) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "active/"})
	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}

// Patch updates the active file.
func (s *ActiveFileService) Patch(ctx context.Context, op PatchOperation, targetType TargetType, target string, content string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "active/"})
	req, err := http.NewRequestWithContext(ctx, "PATCH", u.String(), strings.NewReader(content))
	if err != nil {
		return err
	}
	
	req.Header.Set("Operation", string(op))
	req.Header.Set("Target-Type", string(targetType))
	req.Header.Set("Target", target)
	// Default to markdown content type for simplicity in this helper, 
	// though the API supports JSON for table rows etc.
	// We might want to make this flexible later.
	req.Header.Set("Content-Type", "text/markdown")

	return s.client.do(req, nil)
}
