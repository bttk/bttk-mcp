package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bttk/bttk-mcp/pkg/calendar"
	"github.com/bttk/bttk-mcp/pkg/calendarmcp"
	"github.com/bttk/bttk-mcp/pkg/config"
	"github.com/bttk/bttk-mcp/pkg/gmail"
	"github.com/bttk/bttk-mcp/pkg/gmailmcp"
	"github.com/bttk/bttk-mcp/pkg/obsidian"
	"github.com/bttk/bttk-mcp/pkg/obsidianmcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/util"
)

const shutdownTimeout = 5 * time.Second

func main() {
	// Custom flag usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [flags] [subcommand]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSubcommands:\n")
		fmt.Fprintf(os.Stderr, "  auth\tAuthenticate and verify Calendar and Gmail APIs\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	configPath := flag.String("config", "", "Path to configuration file")
	forceStdio := flag.Bool("s", false, "Force communication over standard input/output (stdio)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Handle subcommand
	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "auth":
			runAuth(cfg)
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", flag.Arg(0))
			flag.Usage()
			os.Exit(1)
		}
	}

	// Default behavior: run MCP Server
	runServer(cfg, *forceStdio)
}

func runAuth(cfg *config.Config) {
	if cfg.Calendar.Enabled {
		fmt.Println("Checking Calendar authentication...")
		credPath, tokenPath := cfg.Calendar.CredentialsFile, cfg.Calendar.TokenFile
		client, err := calendar.NewClient(credPath, tokenPath)
		if err != nil {
			log.Fatalf("Failed to create Calendar client: %v", err)
		}
		fmt.Println("Authentication successful. Verifying API access...")
		_, err = client.ListCalendars()
		if err != nil {
			log.Fatalf("Calendar API verification failed: %v\n(If you have recently changed scopes, try deleting token.json)", err)
		}
		fmt.Println("Calendar authentication and verification completed successfully!")
	} else {
		fmt.Println("Calendar is not enabled in config, skipping.")
	}

	if cfg.Gmail.Enabled {
		fmt.Println("Checking Gmail authentication...")
		client, err := gmail.NewClient(cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
		if err != nil {
			log.Fatalf("Failed to create Gmail client: %v", err)
		}
		fmt.Println("Authentication successful. Verifying API access...")
		_, err = client.SearchMessages("label:INBOX", 1)
		if err != nil {
			log.Fatalf("Gmail API verification failed: %v", err)
		}
		fmt.Println("Gmail authentication and verification completed successfully!")
	} else {
		fmt.Println("Gmail is not enabled in config, skipping.")
	}
}

func runServer(cfg *config.Config, forceStdio bool) {
	s := server.NewMCPServer(
		"BTTK Combined MCP Server",
		"1.0.0",
		server.WithLogging(),
	)

	// Helper for tools filtering based on cfg.MCP.Tools config
	registerTool := func(name string, registerFunc func()) {
		if cfg.MCP.Tools != nil {
			if enabled, ok := cfg.MCP.Tools[name]; ok && !enabled {
				log.Printf("Tool %s explicitly disabled in config, skipping", name)
				return
			}
		}
		registerFunc()
	}

	setupObsidian(s, cfg, registerTool)
	setupCalendar(s, cfg, registerTool)
	setupGmail(s, cfg, registerTool)

	// Determine transport mode
	listenAddr := cfg.MCP.Address
	if forceStdio {
		if err := serveStdio(s); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		if listenAddr == "" {
			log.Fatal("Error: listen address must be specified in config under mcp.address (e.g. \"localhost:2885\"), or force stdio mode with -s")
		}
		if err := serveStreamableHTTP(s, listenAddr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func setupObsidian(s *server.MCPServer, cfg *config.Config, registerTool func(string, func())) {
	if !cfg.Obsidian.Enabled {
		log.Println("Warning: Obsidian not enabled in config. Skipping Obsidian tools.")
		return
	}
	if cfg.Obsidian.URL == "" {
		log.Println("Warning: Obsidian url not configured. Skipping Obsidian tools.")
		return
	}

	log.Println("Initializing Obsidian service client...")
	var opts []obsidian.Option
	if cfg.Obsidian.Cert != "" {
		opts = append(opts, obsidian.WithCertificate(cfg.Obsidian.Cert))
	} else {
		opts = append(opts, obsidian.WithInsecureTLS())
	}
	client, err := obsidian.NewClient(cfg.Obsidian.URL, cfg.Obsidian.APIKey, opts...)
	if err != nil {
		log.Fatalf("Failed to create Obsidian client: %v", err)
	}

	registerTool("get_active_file", func() {
		s.AddTool(obsidianmcp.GetActiveFileTool(), obsidianmcp.GetActiveFileHandler(client))
	})
	registerTool("append_active_file", func() {
		s.AddTool(obsidianmcp.AppendActiveFileTool(), obsidianmcp.AppendActiveFileHandler(client))
	})
	registerTool("patch_active_file", func() {
		s.AddTool(obsidianmcp.PatchActiveFileTool(), obsidianmcp.PatchActiveFileHandler(client))
	})
	registerTool("search_simple", func() {
		s.AddTool(obsidianmcp.SearchSimpleTool(), obsidianmcp.SearchSimpleHandler(client))
	})
	registerTool("search_json_logic", func() {
		s.AddTool(obsidianmcp.SearchJSONLogicTool(), obsidianmcp.SearchJSONLogicHandler(client))
	})
	registerTool("search_dql", func() {
		s.AddTool(obsidianmcp.SearchDQLTool(), obsidianmcp.SearchDQLHandler(client))
	})
	registerTool("get_daily_note", func() {
		s.AddTool(obsidianmcp.GetDailyNoteTool(), obsidianmcp.GetDailyNoteHandler(client))
	})
	registerTool("get_file", func() {
		s.AddTool(obsidianmcp.GetFileTool(), obsidianmcp.GetFileHandler(client))
	})
	registerTool("list_files", func() {
		s.AddTool(obsidianmcp.ListFilesTool(), obsidianmcp.ListFilesHandler(client))
	})
	registerTool("create_or_update_file", func() {
		s.AddTool(obsidianmcp.CreateOrUpdateFileTool(), obsidianmcp.CreateOrUpdateFileHandler(client))
	})
	registerTool("open_file", func() {
		s.AddTool(obsidianmcp.OpenFileTool(), obsidianmcp.OpenFileHandler(client))
	})
}

func setupCalendar(s *server.MCPServer, cfg *config.Config, registerTool func(string, func())) {
	if !cfg.Calendar.Enabled {
		log.Println("Warning: Google Calendar not enabled in config. Skipping Calendar tools.")
		return
	}

	log.Println("Initializing Google Calendar service client...")
	credPath, tokenPath := cfg.Calendar.CredentialsFile, cfg.Calendar.TokenFile
	client, err := calendar.NewClient(credPath, tokenPath)
	if err != nil {
		log.Fatalf("Failed to create Google Calendar client: %v", err)
	}

	toolConfig := map[string][]string{
		"calendars": cfg.Calendar.Calendars,
	}

	registerTool("calendar_list", func() {
		s.AddTool(calendarmcp.CalendarListTool(), calendarmcp.CalendarListHandler(client, toolConfig))
	})
	registerTool("calendar_list_events", func() {
		s.AddTool(calendarmcp.CalendarListEventsTool(), calendarmcp.CalendarListEventsHandler(client, toolConfig))
	})
	registerTool("calendar_create_event", func() {
		s.AddTool(calendarmcp.CalendarCreateEventTool(), calendarmcp.CalendarCreateEventHandler(client, toolConfig))
	})
	registerTool("calendar_patch_event", func() {
		s.AddTool(calendarmcp.CalendarPatchEventTool(), calendarmcp.CalendarPatchEventHandler(client, toolConfig))
	})
	registerTool("calendar_delete_event", func() {
		s.AddTool(calendarmcp.CalendarDeleteEventTool(), calendarmcp.CalendarDeleteEventHandler(client, toolConfig))
	})
	registerTool("calendar_move_event", func() {
		s.AddTool(calendarmcp.CalendarMoveEventTool(), calendarmcp.CalendarMoveEventHandler(client, toolConfig))
	})
}

func setupGmail(s *server.MCPServer, cfg *config.Config, registerTool func(string, func())) {
	if !cfg.Gmail.Enabled {
		log.Println("Warning: Gmail not enabled in config. Skipping Gmail tools.")
		return
	}

	log.Println("Initializing Gmail service client...")
	client, err := gmail.NewClient(cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
	if err != nil {
		log.Fatalf("Failed to create Gmail client: %v", err)
	}

	registerTool("gmail_search", func() {
		s.AddTool(gmailmcp.GmailSearchTool(), gmailmcp.GmailSearchHandler(client))
	})
	registerTool("gmail_read", func() {
		s.AddTool(gmailmcp.GmailReadTool(), gmailmcp.GmailReadHandler(client))
	})
}

func serveStdio(srv *server.MCPServer) error {
	log.Println("Starting MCP server in stdio mode...")
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

func serveStreamableHTTP(srv *server.MCPServer, addr string) error {
	httpServer := server.NewStreamableHTTPServer(srv,
		// server.WithEndpointPath(addr),
		server.WithLogger(util.DefaultLogger()),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		log.Println("Shutting down HTTP/SSE server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP/SSE shutdown error: %v", err)
		}
		cancel()
	}()

	log.Printf("Starting MCP server in Streamable HTTP, listening on %s...", addr)
	err := httpServer.Start(addr)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-ctx.Done()
	return nil
}
