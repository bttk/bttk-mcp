package obsidian

import (
	"context"
	"net/http"
	"net/url"
)

// OpenService handles opening files in the Obsidian UI.
type OpenService struct {
	client *Client
}

// File opens the specified file in Obsidian.
// If newLeaf is true, the file will be opened in a new leaf (tab).
func (s *OpenService) File(ctx context.Context, filename string, newLeaf bool) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "open/" + filename})
	if newLeaf {
		q := u.Query()
		q.Set("newLeaf", "true")
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}
