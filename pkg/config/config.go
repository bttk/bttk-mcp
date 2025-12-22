package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// Config represents the configuration for the Obsidian Local REST API.
type Config struct {
	Obsidian struct {
		URL    string `json:"url"`
		Cert   string `json:"cert"`
		APIKey string `json:"apikey"`
	} `json:"obsidian"`
	Gmail struct {
		Enabled         bool   `json:"enabled"`
		CredentialsFile string `json:"credentials_file"`
		TokenFile       string `json:"token_file"`
	} `json:"gmail"`
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

	configDir := filepath.Dir(path)
	resolve := func(p string) (string, error) {
		if p == "" || filepath.IsAbs(p) {
			return p, nil
		}
		fullPath := filepath.Join(configDir, p)
		return filepath.Abs(fullPath)
	}

	// Set defaults for Gmail
	if cfg.Gmail.CredentialsFile == "" {
		cfg.Gmail.CredentialsFile = "credentials.json"
	}
	if cfg.Gmail.TokenFile == "" {
		cfg.Gmail.TokenFile = "token.json"
	}

	var errPath error
	if cfg.Obsidian.Cert, errPath = resolve(cfg.Obsidian.Cert); errPath != nil {
		return nil, errPath
	}
	if cfg.Gmail.CredentialsFile, errPath = resolve(cfg.Gmail.CredentialsFile); errPath != nil {
		return nil, errPath
	}
	if cfg.Gmail.TokenFile, errPath = resolve(cfg.Gmail.TokenFile); errPath != nil {
		return nil, errPath
	}

	return &cfg, nil
}
