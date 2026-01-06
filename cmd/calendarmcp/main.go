package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bttk/bttk-mcp/pkg/calendar"
	"github.com/bttk/bttk-mcp/pkg/calendarmcp"
	"github.com/bttk/bttk-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Custom flag usage to support subcommands
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [flags] [subcommand]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSubcommands:\n")
		fmt.Fprintf(os.Stderr, "  list\tList available calendars\n")
		fmt.Fprintf(os.Stderr, "  auth\tAuthenticate with Google Calendar\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Handle subcommands
	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "list":
			runList(*configPath)
			return
		case "auth":
			runAuth(*configPath)
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", flag.Arg(0))
			flag.Usage()
			os.Exit(1)
		}
	}

	// Default behavior: Run MCP Server
	runServer(*configPath)
}

func getCredentialsPaths(cfg *config.Config) (string, string) {
	if cfg != nil {
		return cfg.Calendar.CredentialsFile, cfg.Calendar.TokenFile
	}
	return "credentials.json", "token.json"
}

func loadConfig(path string) (*config.Config, error) {
	// Using pkg/config
	return config.Load(path)
}

func runList(configPath string) {
	client, _ := setup(configPath)

	calendars, err := client.ListCalendars()
	if err != nil {
		log.Fatalf("Failed to list calendars: %v", err)
	}

	fmt.Println("All Available Calendars:")

	for _, cal := range calendars {
		fmt.Printf("- %s (ID: %s) [Primary: %v] [Access: %s]\n", cal.Summary, cal.Id, cal.Primary, cal.AccessRole)
	}
}

func runAuth(configPath string) {
	fmt.Println("Checking Calendar authentication...")
	client, _ := setup(configPath)

	fmt.Println("Authentication successful. Verifying API access...")
	_, err := client.ListCalendars()
	if err != nil {
		log.Fatalf("API verification failed: %v\n(If you have recently changed scopes, try deleting token.json)", err)
	}

	fmt.Println("Calendar authentication and verification completed successfully!")
}

func runServer(configPath string) {
	client, cfg := setup(configPath)

	s := server.NewMCPServer(
		"Calendar MCP",
		"1.0.0",
		server.WithLogging(),
	)

	// Pass config for runtime allowlist checking in tools
	// We need to convert cfg.Calendar.Calendars to map/slice relevant to tools
	// tools.go expects map[string][]string
	toolConfig := map[string][]string{
		"calendars": cfg.Calendar.Calendars,
	}

	calendarmcp.AddTools(s, client, toolConfig)

	if err := serveStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func setup(configPath string) (*calendar.Client, *config.Config) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		// Log warning but continue if just config file missing vs struct error?
		// Usually config is required for some things, but maybe defaults work.
		log.Printf("Warning: error loading config: %v", err)
	}

	// Create client
	// Note: pkg/calendar/client.go NewClient takes (credentialsPath, tokenPath string)
	credPath, tokenPath := getCredentialsPaths(cfg)

	client, err := calendar.NewClient(credPath, tokenPath)
	if err != nil {
		log.Fatalf("Failed to create calendar client: %v", err)
	}
	return client, cfg
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
