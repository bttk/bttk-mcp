package obsidian

import (
	"context"
	"net/http"
	"net/url"
)

// CommandService handles interaction with Obsidian commands.
type CommandService struct {
	client *Client
}

// Command represents an available command.
type Command struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// List returns a list of available commands.
func (s *CommandService) List(ctx context.Context) ([]Command, error) {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "commands/"})
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Commands []Command `json:"commands"`
	}
	err = s.client.do(req, &resp)
	return resp.Commands, err
}

// Execute executes a command by its ID.
func (s *CommandService) Execute(ctx context.Context, commandID string) error {
	u := s.client.baseURL.ResolveReference(&url.URL{Path: "commands/" + commandID + "/"})
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}
