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
	Calendar struct {
		Enabled         bool     `json:"enabled"`
		CredentialsFile string   `json:"credentials_file"`
		TokenFile       string   `json:"token_file"`
		Calendars       []string `json:"calendars"`
	} `json:"calendar"`
	MCP struct {
		Tools map[string]bool `json:"tools"`
	} `json:"mcp"`
}

// Load loads the configuration from a JSON file.
// If path is empty, it searches for "bttk-mcp/config.json" in XDG config directories.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = xdg.SearchConfigFile("bttk-mcp/config.json")
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

	// Set defaults for Calendar
	if cfg.Calendar.CredentialsFile == "" {
		cfg.Calendar.CredentialsFile = "credentials.json"
	}
	if cfg.Calendar.TokenFile == "" {
		cfg.Calendar.TokenFile = "token.json"
	}

	if cfg.Calendar.CredentialsFile, errPath = resolve(cfg.Calendar.CredentialsFile); errPath != nil {
		return nil, errPath
	}
	if cfg.Calendar.TokenFile, errPath = resolve(cfg.Calendar.TokenFile); errPath != nil {
		return nil, errPath
	}

	return &cfg, nil
}
