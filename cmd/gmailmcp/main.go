package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bttk.dev/agent/pkg/config"
	"bttk.dev/agent/pkg/gmail"
	"bttk.dev/agent/pkg/gmailmcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Gmail.CredentialsFile == "" || cfg.Gmail.TokenFile == "" {
		log.Fatal("Gmail credentials_file and token_file must be specified in config")
	}

	client, err := gmail.NewClient(cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
	if err != nil {
		log.Fatalf("Failed to create Gmail client: %v", err)
	}

	s := server.NewMCPServer("gmailmcp", "1.0.0")

	gmailmcp.AddTools(s, client)

	if err := serveStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func serveStdio(srv *server.MCPServer) error {
	stdioServer := server.NewStdioServer(srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()

	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
