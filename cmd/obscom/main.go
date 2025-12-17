package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"bttk.dev/agent/pkg/obsidian"
	"bttk.dev/agent/pkg/obsidian/config"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "command" || os.Args[2] != "list" {
		fmt.Println("Usage: obscom command list")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("obscom", flag.ExitOnError)
	configPath := fs.String("config", "config.json", "path to the configuration file")
	fs.Parse(os.Args[3:])

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var opts []obsidian.Option
	if cfg.Obsidian.Cert != "" {
		opts = append(opts, obsidian.WithCertificate(cfg.Obsidian.Cert))
	} else {
		opts = append(opts, obsidian.WithInsecureTLS())
	}
	client, err := obsidian.NewClient(cfg.Obsidian.URL, cfg.Obsidian.APIKey, opts...)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	commands, err := client.Commands.List(context.Background())
	if err != nil {
		log.Fatalf("failed to list commands: %v", err)
	}

	fmt.Printf("%-30s %s\n", "NAME", "ID")
	fmt.Printf("%-30s %s\n", "----", "--")
	for _, cmd := range commands {
		fmt.Printf("%-30s %s\n", cmd.Name, cmd.ID)
	}
}
