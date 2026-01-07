package obsidian

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// SearchService handles searching in the vault.
type SearchService struct {
	client *Client
}

// SearchResult represents a single match in Simple search.
type SearchResult struct {
	Filename string  `json:"filename"`
	Score    float64 `json:"score"`
	Matches  []struct {
		Context string `json:"context"`
		Match   struct {
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"match"`
	} `json:"matches"`
}

// Simple performs a simple text search.
func (s *SearchService) Simple(ctx context.Context, query string, contextLength int) ([]SearchResult, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "search/simple/"})
	q := u.Query()
	q.Set("query", query)
	if contextLength > 0 {
		q.Set("contextLength", strconv.Itoa(contextLength))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	err = s.client.do(req, &results)
	return results, err
}

// JsonLogicResult represents a result from a JsonLogic or Dataview search.
type JsonLogicResult struct {
	Filename string      `json:"filename"`
	Result   interface{} `json:"result"`
}

// JsonLogic performs a structured search using JsonLogic.
func (s *SearchService) JsonLogic(ctx context.Context, query interface{}) ([]JsonLogicResult, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "search/"})

	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/vnd.olrapi.jsonlogic+json")

	var results []JsonLogicResult
	err = s.client.do(req, &results)
	return results, err
}

// Dataview performs a search using Dataview Query Language (DQL).
func (s *SearchService) Dataview(ctx context.Context, dql string) ([]JsonLogicResult, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "search/"})
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader([]byte(dql)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/vnd.olrapi.dataview.dql+txt")

	var results []JsonLogicResult
	err = s.client.do(req, &results)
	return results, err
}
