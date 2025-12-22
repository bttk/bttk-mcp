package config

import (
	"encoding/json"
	"os"

	"github.com/adrg/xdg"
)

// Config represents the configuration for the Obsidian Local REST API.
type Config struct {
	Obsidian struct {
		URL    string `json:"url"`
		Cert   string `json:"cert"`
		APIKey string `json:"apikey"`
	} `json:"obsidian"`
	MCP struct {
		Tools map[string]bool `json:"tools"`
	} `json:"mcp"`
}

// Load loads the configuration from a JSON file.
// If path is empty, it searches for "bagent/config.json" in XDG config directories.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = xdg.SearchConfigFile("bagent/config.json")
		if err != nil {
			return nil, err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
