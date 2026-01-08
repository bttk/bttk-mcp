package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bttk/bttk-mcp/pkg/config"
	"github.com/bttk/bttk-mcp/pkg/gmail"
	"github.com/bttk/bttk-mcp/pkg/gmailmcp"
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

	// Check for commands
	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "auth":
			runAuth(cfg)
			return
		default:
			log.Fatalf("Unknown command: %s", flag.Arg(0))
		}
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

func runAuth(cfg *config.Config) {
	fmt.Println("Checking Gmail authentication...")
	client, err := gmail.NewClient(cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	fmt.Println("Authentication successful. Verifying API access...")

	// Perform a simple search to verify the token works for API calls
	_, err = client.SearchMessages("label:INBOX")
	if err != nil {
		log.Fatalf("API verification failed: %v", err)
	}
	fmt.Println("Gmail authentication and verification completed successfully!")
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
