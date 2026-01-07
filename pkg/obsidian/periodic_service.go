package obsidian

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PeriodicService handles interaction with periodic notes (daily, weekly, etc.).
type PeriodicService struct {
	client *Client
}

// GetCurrent returns the content of the current periodic note for the specified period.
// period can be "daily", "weekly", "monthly", "quarterly", "yearly".
func (s *PeriodicService) GetCurrent(ctx context.Context, period string) (string, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/", period)})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var content string
	err = s.client.do(req, &content)
	return content, err
}

// GetCurrentNote returns the current periodic note parsed as a Note struct.
func (s *PeriodicService) GetCurrentNote(ctx context.Context, period string) (*Note, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/", period)})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.olrapi.note+json")

	var note Note
	err = s.client.do(req, &note)
	return &note, err
}

// AppendToCurrent appends content to the current periodic note.
func (s *PeriodicService) AppendToCurrent(ctx context.Context, period, content string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/", period)})
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), strings.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/markdown")

	return s.client.do(req, nil)
}

// PatchCurrent updates the current periodic note.
func (s *PeriodicService) PatchCurrent(ctx context.Context, period string, op PatchOperation, targetType TargetType, target string, content string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/", period)})
	req, err := http.NewRequestWithContext(ctx, "PATCH", u.String(), strings.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Operation", string(op))
	req.Header.Set("Target-Type", string(targetType))
	req.Header.Set("Target", target)
	req.Header.Set("Content-Type", "text/markdown")

	return s.client.do(req, nil)
}

// DeleteCurrent deletes the current periodic note.
func (s *PeriodicService) DeleteCurrent(ctx context.Context, period string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/", period)})
	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}

// Get returns the content of a periodic note for a specific date.
func (s *PeriodicService) Get(ctx context.Context, period string, year, month, day int) (string, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("periodic/%s/%d/%d/%d/", period, year, month, day)})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var content string
	err = s.client.do(req, &content)
	return content, err
}

// Note: Additional methods for Append, Patch, Delete for specific dates can be added following similar pattern.
// Implementing subset for brevity as per requirement to design a package, but full client would have them.
