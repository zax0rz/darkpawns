// dp-agent — Dark Pawns Agent CLI.
//
// One binary to connect, play, log, and manage AI agents in Dark Pawns.
//
// Usage:
//
//	dp-agent play                  # interactive play mode
//	dp-agent session               # timed session with logging
//	dp-agent dream                 # offline memory consolidation
//	dp-agent config                # view or set config
//	dp-agent keygen                # generate a new agent API key
//	dp-agent whoami                # show agent identity
//	dp-agent exec "north"          # one-shot command
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/pkg/agentcli"
	"github.com/zax0rz/darkpawns/pkg/dreaming"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "play":
		cmdPlay(os.Args[2:])
	case "session":
		cmdSession(os.Args[2:])
	case "config":
		cmdConfig(os.Args[2:])
	case "keygen":
		cmdKeygen(os.Args[2:])
	case "whoami":
		cmdWhoami()
	case "dream":
		cmdDream(os.Args[2:])
	case "exec":
		cmdExec(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `dp-agent — Dark Pawns Agent CLI

Usage:
  dp-agent play          Interactive play mode
  dp-agent session       Timed session with full logging
  dp-agent dream         Offline memory consolidation (dreaming cycle)
  dp-agent config        View or set agent configuration
  dp-agent keygen -name  Generate a new agent API key
  dp-agent whoami        Show agent identity
  dp-agent exec <cmd>    One-shot command

Env: DP_KEY, DP_CONFIG
`)
}

func runAgent(cfg *agentcli.AgentConfig, ctx context.Context) {
	client := agentcli.NewAgentClient(cfg)
	if err := client.Connect(ctx); err != nil {
		slog.Error("connect", "error", err)
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()
	slog.Info("connected")
	if err := client.RunDecisionLoop(ctx); err != nil {
		slog.Warn("done", "error", err)
	}
}

func loadConfig() *agentcli.AgentConfig {
	cfg, err := agentcli.LoadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	return cfg
}

// ─── play ─────────────────────────────────────────────────────────────────────

func cmdPlay(args []string) {
	fs := flag.NewFlagSet("play", flag.ExitOnError)
	duration := fs.Duration("duration", 0, "max duration (15m, 1h)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if *duration > 0 {
		dlCtx, dlCancel := context.WithTimeout(ctx, *duration)
		defer dlCancel()
		runAgent(cfg, dlCtx)
	} else {
		runAgent(cfg, ctx)
	}
}

// ─── session ──────────────────────────────────────────────────────────────────

func cmdSession(args []string) {
	fs := flag.NewFlagSet("session", flag.ExitOnError)
	duration := fs.Duration("duration", 15*time.Minute, "session duration")
	temp := fs.Float64("temp", -1, "LLM temperature override (default: config value)")
	valence := fs.Bool("valence", true, "enable emotional valence in memory")
	logDir := fs.String("log-dir", "", "log output directory (default: config value)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	cfg := loadConfig()
	if *temp >= 0 {
		cfg.Temperature = *temp
	}
	cfg.Valence = *valence
	if *logDir != "" {
		cfg.LogDir = *logDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	sigCtx, sigCancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()

	slog.Info("session", "duration", *duration)
	runAgent(cfg, sigCtx)
}

// ─── config ───────────────────────────────────────────────────────────────────

func cmdConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	key := fs.String("key", "", "set API key")
	playerName := fs.String("player-name", "", "set character name")
	model := fs.String("model", "", "set model")
	tier := fs.String("tier", "", "set tier (small/medium/large/unlimited)")
	temp := fs.Float64("temp", -1, "set LLM temperature")
	logDir := fs.String("log-dir", "", "set log output directory")
	valence := fs.String("valence", "", "enable emotional valence (true/false)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	cfg, err := agentcli.LoadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	changed := false
	if *key != "" {
		cfg.Key = *key
		changed = true
	}
	if *playerName != "" {
		cfg.PlayerName = *playerName
		changed = true
	}
	if *model != "" {
		cfg.ModelFast = *model
		changed = true
	}
	if *tier != "" {
		cfg.Tier = *tier
		changed = true
	}
	if *temp >= 0 {
		cfg.Temperature = *temp
		changed = true
	}
	if *logDir != "" {
		cfg.LogDir = *logDir
		changed = true
	}
	if *valence != "" {
		cfg.Valence = *valence == "true"
		changed = true
	}
	if changed {
		if err := agentcli.SaveConfig(cfg); err != nil {
			slog.Error("save config", "error", err)
			os.Exit(1)
		}
		fmt.Println("config saved to", agentcli.ConfigPath())
	}

	fmt.Printf("Config: %s\n", agentcli.ConfigPath())
	fmt.Printf("  Key:     %s\n", maskKey(cfg.Key))
	fmt.Printf("  Player:  %s\n", cfg.PlayerName)
	fmt.Printf("  Model:   %s\n", cfg.ModelFast)
	fmt.Printf("  Tier:    %s\n", cfg.Tier)
	fmt.Printf("  Game:    %s:%d\n", cfg.GameHost, cfg.GamePort)
}

// ─── keygen ───────────────────────────────────────────────────────────────────

func cmdKeygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	name := fs.String("name", "", "character name")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: -name is required")
		fs.Usage()
		os.Exit(1)
	}

	// Single-quote the name for the suggested shell command. Go's %q is not
	// shell-safe (e.g. a name containing $(...) would execute if pasted inside
	// double quotes); single-quote escaping is.
	fmt.Fprintf(os.Stderr, "run: go run ./cmd/agentkeygen -name %s -db \"$DB_DSN\"\n", shellSingleQuote(*name))
	fmt.Fprintf(os.Stderr, "(keygen integration in next pass)\n")
}

// shellSingleQuote wraps s in single quotes for safe shell paste, escaping any
// embedded single quotes via the '\” idiom.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ─── whoami ───────────────────────────────────────────────────────────────────

func cmdWhoami() {
	cfg, err := agentcli.LoadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	fmt.Printf("Key:   %s\n", maskKey(cfg.EffectiveKey()))
	fmt.Printf("Tier:  %s\n", cfg.Tier)
	fmt.Printf("Model: %s\n", cfg.ModelFast)
	fmt.Printf("Game:  %s:%d\n", cfg.GameHost, cfg.GamePort)
}

// ─── dream ────────────────────────────────────────────────────────────────────

func cmdDream(args []string) {
	fs := flag.NewFlagSet("dream", flag.ExitOnError)
	sessionDir := fs.String("sessions", "data/sessions", "session log directory")
	outputDir := fs.String("output", "data/memory", "output directory for memory graph")
	agentID := fs.String("agent", "", "agent ID to process (required)")
	dryRun := fs.Bool("dry-run", false, "print what would happen without writing")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *agentID == "" {
		fmt.Fprintln(os.Stderr, "error: -agent is required")
		fs.Usage()
		os.Exit(1)
	}

	cfg := dreaming.DreamConfig{
		SessionDir:  *sessionDir,
		OutputDir:   *outputDir,
		AgentID:     *agentID,
		GraphConfig: dreaming.DefaultGraphConfig(),
		DryRun:      *dryRun,
	}

	slog.Info("dream", "agent", *agentID, "sessions", *sessionDir, "output", *outputDir)
	result, err := dreaming.RunDream(cfg)
	if err != nil {
		slog.Error("dream failed", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Dream complete:\n")
	fmt.Printf("  Agent:           %s\n", result.AgentID)
	fmt.Printf("  Sessions read:   %d\n", result.SessionFiles)
	fmt.Printf("  Events extracted: %d\n", result.EventsExtracted)
	fmt.Printf("  Nodes before:    %d\n", result.NodesBefore)
	fmt.Printf("  Nodes after:     %d\n", result.NodesAfter)
	fmt.Printf("  Pruned:          %d\n", result.Pruned)
	fmt.Printf("  Summary tokens:  %d\n", result.SummaryTokens)
}

// ─── exec ─────────────────────────────────────────────────────────────────────

func cmdExec(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dp-agent exec <command> [args...]")
		os.Exit(1)
	}

	cfg := loadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := agentcli.NewAgentClient(cfg)
	if err := client.Connect(ctx); err != nil {
		slog.Error("connect", "error", err)
		cancel()
		os.Exit(1) //nolint:gocritic // exitAfterDefer: cancel() called explicitly above
	}
	defer func() { _ = client.Close() }()

	action := &agentcli.LLMResponse{
		ActionType: args[0],
		Args:       args[1:],
	}

	if err := client.PushCommand(ctx, action); err != nil {
		slog.Error("exec", "error", err)
		cancel()
		os.Exit(1) //nolint:gocritic // exitAfterDefer: cancel() called explicitly above
	}

	// Read and print the server response
	for {
		var msg struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := client.ReadJSON(&msg); err != nil {
			break
		}
		slog.Debug("exec recv", "type", msg.Type)
		if msg.Type == "state" {
			// Decode into the canonical client state model. The bespoke struct
			// this replaced declared Events as []string, but the server sends
			// structured Event objects — the mismatch made json.Unmarshal fail
			// and the whole state (room + HP) get silently dropped.
			var state agentcli.GameState
			if err := json.Unmarshal(msg.Data, &state); err != nil {
				slog.Debug("exec parse state", "error", err)
				continue
			}
			if state.Room.Name != "" {
				fmt.Printf("%s (HP: %d/%d)\n", state.Room.Name, state.Player.Health, state.Player.MaxHealth)
			}
			for _, e := range state.Events {
				fmt.Printf("%v\n", e.Data)
			}
			return // got our response, exit
		}
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func maskKey(key string) string {
	if len(key) < 8 {
		return "***"
	}
	return key[:4] + "…" + key[len(key)-4:]
}
