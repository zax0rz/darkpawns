package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zax0rz/darkpawns/pkg/agentcli"
)

func main() {
	name := flag.String("name", "", "character name (required)")
	configPath := flag.String("config", "", "config file path (default ~/.dp-agent.json)")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required")
		flag.Usage()
		os.Exit(1)
	}

	// Set up logging
	level := slog.LevelInfo
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	// Load config
	var cfg *agentcli.AgentConfig
	var err error
	if *configPath != "" {
		// TODO: load from specific path
		cfg, err = agentcli.LoadConfig()
	} else {
		cfg, err = agentcli.LoadConfig()
	}
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	cfg.PlayerName = *name

	// Validate config
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// Create daemon
	daemon, err := agentcli.NewDaemon(cfg)
	if err != nil {
		slog.Error("create daemon", "error", err)
		os.Exit(1)
	}

	// Set up signal handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()
	}()

	// Start daemon (blocks until context is cancelled)
	slog.Info("starting daemon", "player", cfg.PlayerName)
	if err := daemon.Start(ctx); err != nil {
		slog.Error("daemon error", "error", err)
		os.Exit(1)
	}
}
