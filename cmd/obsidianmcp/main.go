package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bttk.dev/agent/pkg/obsidian"
	"bttk.dev/agent/pkg/obsidian/config"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var configPath string
	var verbose bool
	flag.StringVar(&configPath, "config", "config.json", "path to config file")
	flag.BoolVar(&verbose, "v", false, "enable verbose logging of input/output")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("failed to load %s: %v", configPath, err)
		os.Exit(1)
	}

	// Initialize Obsidian Client
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

	// Create MCP Server
	s := server.NewMCPServer(
		"Obsidian MCP Server",
		"1.0.0",
	)

	// Tool registry map
	toolRegistry := map[string]func(*server.MCPServer, *obsidian.Client){
		"get_active_file":       registerGetActiveFile,
		"append_active_file":    registerAppendActiveFile,
		"patch_active_file":     registerPatchActiveFile,
		"search_simple":         registerSearchSimple,
		"search_json_logic":     registerSearchJSONLogic,
		"get_daily_note":        registerGetDailyNote,
		"get_file":              registerGetFile,
		"list_files":            registerListFiles,
		"create_or_update_file": registerCreateOrUpdateFile,
		"open_file":             registerOpenFile,
	}

	// Register tools based on config
	for name, registerFunc := range toolRegistry {
		if enabled, ok := cfg.MCP.Tools[name]; ok && enabled {
			registerFunc(s, client)
		} else if !ok {
			// If not in config, we can decide to enable by default or skip.
			// Given the user request, skipping seems safer if we want strict control.
			log.Printf("Tool %s not found in config, skipping", name)
		}
	}

	// Start the server using Stdio
	var in io.Reader = os.Stdin
	var out io.Writer = os.Stdout
	if verbose {
		in = &loggingReader{os.Stdin}
		out = &loggingWriter{os.Stdout}
	}
	if err := ServeStdio(s, in, out); err != nil {
		log.Printf("Server error: %v", err)
		os.Exit(1)
	}
}

type loggingReader struct {
	r io.Reader
}

func (lr *loggingReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		log.Printf("IN: %s", p[:n])
	}
	return n, err
}

type loggingWriter struct {
	w io.Writer
}

func (lw *loggingWriter) Write(p []byte) (n int, err error) {
	log.Printf("OUT: %s", p)
	return lw.w.Write(p)
}

func ServeStdio(srv *server.MCPServer, in io.Reader, out io.Writer) error {
	s := server.NewStdioServer(srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()

	return s.Listen(ctx, in, out)
}
