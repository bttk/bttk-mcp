package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/bttk/bttk-mcp/pkg/config"
	"github.com/bttk/bttk-mcp/pkg/obsidian"
	"github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "command" || os.Args[2] != "list" {
		fmt.Println("Usage: obscom command list")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("obscom", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the configuration file (default: ~/.config/bttk-mcp/config.json)")
	_ = fs.Parse(os.Args[3:])

	// Initialize zerolog
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	var opts []obsidian.Option
	if cfg.Obsidian.Cert != "" {
		opts = append(opts, obsidian.WithCertificate(cfg.Obsidian.Cert))
	} else {
		opts = append(opts, obsidian.WithInsecureTLS())
	}
	client, err := obsidian.NewClient(cfg.Obsidian.URL, cfg.Obsidian.APIKey, opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create client")
	}

	commands, err := client.Commands.List(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list commands")
	}

	fmt.Printf("%-30s %s\n", "NAME", "ID")
	fmt.Printf("%-30s %s\n", "----", "--")
	for _, cmd := range commands {
		fmt.Printf("%-30s %s\n", cmd.Name, cmd.ID)
	}
}
