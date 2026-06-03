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
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/util"
)

const shutdownTimeout = 5 * time.Second

//nolint:gochecknoglobals
var allTools = []struct {
	name        string // name in cfg.MCP.Tools config
	mcpName     string // actual tool name registered
	serviceName string // "obsidian", "calendar", "gmail"
	getTool     func() mcp.Tool
	getHandler  func(
		obsidianClient *obsidian.Client,
		calendarClient *calendar.Client,
		calendarConfig *calendarmcp.CalendarConfig,
		gmailClient *gmail.Client,
	) server.ToolHandlerFunc
}{
	// Obsidian Tools
	{
		name:        "get_active_file",
		mcpName:     "obsidian_get_active_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.GetActiveFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.GetActiveFileHandler(obs)
		},
	},
	{
		name:        "append_active_file",
		mcpName:     "obsidian_append_active_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.AppendActiveFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.AppendActiveFileHandler(obs)
		},
	},
	{
		name:        "patch_active_file",
		mcpName:     "obsidian_patch_active_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.PatchActiveFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.PatchActiveFileHandler(obs)
		},
	},
	{
		name:        "search_simple",
		mcpName:     "obsidian_search_simple",
		serviceName: "obsidian",
		getTool:     obsidianmcp.SearchSimpleTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.SearchSimpleHandler(obs)
		},
	},
	{
		name:        "search_json_logic",
		mcpName:     "obsidian_search_json_logic",
		serviceName: "obsidian",
		getTool:     obsidianmcp.SearchJSONLogicTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.SearchJSONLogicHandler(obs)
		},
	},
	{
		name:        "search_dql",
		mcpName:     "obsidian_search_dql",
		serviceName: "obsidian",
		getTool:     obsidianmcp.SearchDQLTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.SearchDQLHandler(obs)
		},
	},
	{
		name:        "get_daily_note",
		mcpName:     "obsidian_get_daily_note",
		serviceName: "obsidian",
		getTool:     obsidianmcp.GetDailyNoteTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.GetDailyNoteHandler(obs)
		},
	},
	{
		name:        "get_file",
		mcpName:     "obsidian_get_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.GetFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.GetFileHandler(obs)
		},
	},
	{
		name:        "list_files",
		mcpName:     "obsidian_list_files",
		serviceName: "obsidian",
		getTool:     obsidianmcp.ListFilesTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.ListFilesHandler(obs)
		},
	},
	{
		name:        "create_or_update_file",
		mcpName:     "obsidian_create_or_update_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.CreateOrUpdateFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.CreateOrUpdateFileHandler(obs)
		},
	},
	{
		name:        "open_file",
		mcpName:     "obsidian_open_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.OpenFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.OpenFileHandler(obs)
		},
	},
	{
		name:        "move_file",
		mcpName:     "obsidian_move_file",
		serviceName: "obsidian",
		getTool:     obsidianmcp.MoveFileTool,
		getHandler: func(obs *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return obsidianmcp.MoveFileHandler(obs)
		},
	},

	// Calendar Tools
	{
		name:        "calendar_list",
		mcpName:     "calendar_list",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarListTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarListHandler(cal, cc)
		},
	},
	{
		name:        "calendar_list_events",
		mcpName:     "calendar_list_events",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarListEventsTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarListEventsHandler(cal, cc)
		},
	},
	{
		name:        "calendar_create_event",
		mcpName:     "calendar_create_event",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarCreateEventTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarCreateEventHandler(cal, cc)
		},
	},
	{
		name:        "calendar_patch_event",
		mcpName:     "calendar_patch_event",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarPatchEventTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarPatchEventHandler(cal, cc)
		},
	},
	{
		name:        "calendar_delete_event",
		mcpName:     "calendar_delete_event",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarDeleteEventTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarDeleteEventHandler(cal, cc)
		},
	},
	{
		name:        "calendar_move_event",
		mcpName:     "calendar_move_event",
		serviceName: "calendar",
		getTool:     calendarmcp.CalendarMoveEventTool,
		getHandler: func(_ *obsidian.Client, cal *calendar.Client, cc *calendarmcp.CalendarConfig, _ *gmail.Client) server.ToolHandlerFunc {
			return calendarmcp.CalendarMoveEventHandler(cal, cc)
		},
	},

	// Gmail Tools
	{
		name:        "gmail_search",
		mcpName:     "gmail_search",
		serviceName: "gmail",
		getTool:     gmailmcp.GmailSearchTool,
		getHandler: func(_ *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, gm *gmail.Client) server.ToolHandlerFunc {
			return gmailmcp.GmailSearchHandler(gm)
		},
	},
	{
		name:        "gmail_read",
		mcpName:     "gmail_read",
		serviceName: "gmail",
		getTool:     gmailmcp.GmailReadTool,
		getHandler: func(_ *obsidian.Client, _ *calendar.Client, _ *calendarmcp.CalendarConfig, gm *gmail.Client) server.ToolHandlerFunc {
			return gmailmcp.GmailReadHandler(gm)
		},
	},
}

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
	runServer(cfg, *forceStdio, *configPath)
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

func updateObsidian(client *obsidian.Client, cfg *config.Config) {
	if !cfg.Obsidian.Enabled {
		log.Println("Obsidian is disabled in config")
		_ = client.Update("http://localhost", "")
		return
	}
	if cfg.Obsidian.URL == "" {
		log.Println("Warning: Obsidian url not configured")
		_ = client.Update("http://localhost", "")
		return
	}

	log.Println("Initializing/Updating Obsidian service client...")
	var opts []obsidian.Option
	if cfg.Obsidian.Cert != "" {
		opts = append(opts, obsidian.WithCertificate(cfg.Obsidian.Cert))
	} else {
		opts = append(opts, obsidian.WithInsecureTLS())
	}
	err := client.Update(cfg.Obsidian.URL, cfg.Obsidian.APIKey, opts...)
	if err != nil {
		log.Printf("Failed to update Obsidian client: %v", err)
	}
}

func updateCalendar(client *calendar.Client, cfg *config.Config) {
	if !cfg.Calendar.Enabled {
		log.Println("Google Calendar is disabled in config")
		client.Disable()
		return
	}

	log.Println("Initializing/Updating Google Calendar service client...")
	credPath, tokenPath := cfg.Calendar.CredentialsFile, cfg.Calendar.TokenFile
	err := client.Update(credPath, tokenPath)
	if err != nil {
		log.Printf("Failed to update Google Calendar client: %v", err)
	}
}

func updateGmail(client *gmail.Client, cfg *config.Config) {
	if !cfg.Gmail.Enabled {
		log.Println("Gmail is disabled in config")
		client.Disable()
		return
	}

	log.Println("Initializing/Updating Gmail service client...")
	err := client.Update(cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
	if err != nil {
		log.Printf("Failed to update Gmail client: %v", err)
	}
}

func reconcileTools(
	s *server.MCPServer,
	cfg *config.Config,
	obsClient *obsidian.Client,
	calClient *calendar.Client,
	calConfig *calendarmcp.CalendarConfig,
	gmClient *gmail.Client,
) {
	currentTools := s.ListTools()

	for _, t := range allTools {
		serviceEnabled := false
		switch t.serviceName {
		case "obsidian":
			serviceEnabled = cfg.Obsidian.Enabled && cfg.Obsidian.URL != ""
		case "calendar":
			serviceEnabled = cfg.Calendar.Enabled
		case "gmail":
			serviceEnabled = cfg.Gmail.Enabled
		}

		toolEnabled := true
		if cfg.MCP.Tools != nil {
			if enabled, ok := cfg.MCP.Tools[t.name]; ok {
				toolEnabled = enabled
			}
		}

		shouldRegister := serviceEnabled && toolEnabled

		_, currentlyRegistered := currentTools[t.mcpName]

		if shouldRegister && !currentlyRegistered {
			log.Printf("Registering tool %s", t.mcpName)
			s.AddTool(t.getTool(), t.getHandler(obsClient, calClient, calConfig, gmClient))
		} else if !shouldRegister && currentlyRegistered {
			log.Printf("Unregistering tool %s", t.mcpName)
			s.DeleteTools(t.mcpName)
		}
	}
}

func runServer(cfg *config.Config, forceStdio bool, configPath string) {
	s := server.NewMCPServer(
		"BTTK Combined MCP Server",
		"1.0.0",
		server.WithLogging(),
	)

	// Initialize reloadable client instances
	obsidianClient, err := obsidian.NewClient("http://localhost", "")
	if err != nil {
		log.Fatalf("Failed to initialize Obsidian client structure: %v", err)
	}
	calendarClient := &calendar.Client{}
	gmailClient := &gmail.Client{}
	calendarConfig := &calendarmcp.CalendarConfig{}

	// Update clients initially
	updateObsidian(obsidianClient, cfg)
	updateCalendar(calendarClient, cfg)
	updateGmail(gmailClient, cfg)
	calendarConfig.SetAllowedCalendars(cfg.Calendar.Calendars)

	// Reconcile and register tools initially
	reconcileTools(s, cfg, obsidianClient, calendarClient, calendarConfig, gmailClient)

	// Signal handling for SIGHUP
	initialAddress := cfg.MCP.Address
	sigHupChan := make(chan os.Signal, 1)
	signal.Notify(sigHupChan, syscall.SIGHUP)
	go func() {
		for range sigHupChan {
			log.Println("Received SIGHUP, reloading configuration...")
			newCfg, err := config.Load(configPath)
			if err != nil {
				log.Printf("Error reloading configuration from %s: %v (retaining current config)", configPath, err)
				continue
			}

			if newCfg.MCP.Address != initialAddress {
				log.Printf("Warning: Listen address changed from %s to %s. A server restart is required for this change to take effect.", initialAddress, newCfg.MCP.Address)
			}

			updateObsidian(obsidianClient, newCfg)
			updateCalendar(calendarClient, newCfg)
			updateGmail(gmailClient, newCfg)
			calendarConfig.SetAllowedCalendars(newCfg.Calendar.Calendars)

			reconcileTools(s, newCfg, obsidianClient, calendarClient, calendarConfig, gmailClient)
			log.Println("Configuration reloaded successfully!")
		}
	}()

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
