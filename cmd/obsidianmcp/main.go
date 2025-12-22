package main

import (
	"context"
	"flag"
	"io"
	stdlog "log"
	"log/syslog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bttk.dev/agent/pkg/config"
	"bttk.dev/agent/pkg/obsidian"
	"bttk.dev/agent/pkg/obsidianmcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var configPath string
	var verbose bool
	flag.StringVar(&configPath, "config", "", "path to config file (default: ~/.config/bagent/config.json)")
	flag.BoolVar(&verbose, "v", false, "enable verbose logging of input/output")
	flag.Parse()

	setupLogger()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to load %s", configPath)
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
		log.Fatal().Err(err).Msg("failed to create client")
	}

	// Create MCP Server
	s := server.NewMCPServer(
		"Obsidian MCP Server",
		"1.0.0",
	)

	// Tool registry map
	toolRegistry := map[string]func(*server.MCPServer, *obsidian.Client){
		"get_active_file":       obsidianmcp.RegisterGetActiveFile,
		"append_active_file":    obsidianmcp.RegisterAppendActiveFile,
		"patch_active_file":     obsidianmcp.RegisterPatchActiveFile,
		"search_simple":         obsidianmcp.RegisterSearchSimple,
		"search_json_logic":     obsidianmcp.RegisterSearchJSONLogic,
		"search_dql":            obsidianmcp.RegisterSearchDQL,
		"get_daily_note":        obsidianmcp.RegisterGetDailyNote,
		"get_file":              obsidianmcp.RegisterGetFile,
		"list_files":            obsidianmcp.RegisterListFiles,
		"create_or_update_file": obsidianmcp.RegisterCreateOrUpdateFile,
		"open_file":             obsidianmcp.RegisterOpenFile,
	}

	// Register tools based on config
	for name, registerFunc := range toolRegistry {
		if enabled, ok := cfg.MCP.Tools[name]; ok && enabled {
			log.Info().Msgf("Registering tool %s", name)
			registerFunc(s, client)
		} else if !ok {
			// If not in config, we can decide to enable by default or skip.
			// Given the user request, skipping seems safer if we want strict control.
			log.Warn().Msgf("Tool %s not found in config, skipping", name)
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
		log.Fatal().Err(err).Msg("Server error")
	}
}

type loggingReader struct {
	r io.Reader
}

func (lr *loggingReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		log.Info().Msgf("IN: %q", p[:n])
	}
	return n, err
}

type loggingWriter struct {
	w io.Writer
}

func (lw *loggingWriter) Write(p []byte) (n int, err error) {
	if len(p) < 50 {
		log.Info().Msgf("OUT: %q", p)
	} else {
		log.Info().Msgf("OUT: %q...", p[:50])
	}
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

func setupLogger() {
	syslogger, err := syslog.New(stdlog.LstdFlags, "obsidianmcp")
	if err != nil {
		panic(err)
	}
	log.Logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.SyslogLevelWriter(syslogger),
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		}))
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
