//go:build !web

package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/metrics"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/web"
)

func main() {
	var (
		worldDir   = flag.String("world", "", "Path to world files (lib directory)")
		scriptsDir = flag.String("scripts", "", "Path to Lua scripts (defaults to world/lib/scripts)")
		port       = flag.String("port", "8080", "Server port")
		dbURL      = flag.String("db", "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable", "Database URL")
	)
	flag.Parse()

	if *worldDir == "" {
		slog.Error("Usage: server -world <path-to-lib>")
		os.Exit(1)
	}

	slog.Info("Dark Pawns Phase 1 Server Starting...")

	// Parse world files
	slog.Info("Loading world", "path", *worldDir)
	parsedWorld, err := parser.ParseWorld(*worldDir)
	if err != nil {
		slog.Error("Failed to parse world", "error", err)
		os.Exit(1)
	}
	slog.Info(parsedWorld.Stats())

	// Create game world
	gameWorld, err := game.NewWorld(parsedWorld)
	if err != nil {
		slog.Error("Failed to create game world", "error", err)
		os.Exit(1)
	}

	// Initialize scripting engine
	if *scriptsDir == "" {
		*scriptsDir = *worldDir + "/scripts"
	}
	slog.Info("Loading scripts", "path", *scriptsDir)
	worldAdapter := game.NewWorldScriptableAdapter(gameWorld)
	scriptEngine := scripting.NewEngine(*scriptsDir, worldAdapter)
	game.ScriptEngine = scriptEngine

	// Connect to database
	slog.Info("Connecting to database...")
	database, err := db.New(*dbURL)
	if err != nil {
		slog.Warn("Database connection failed, continuing without persistence", "error", err)
		database = nil
	} else {
		defer database.Close()
		slog.Info("Database connected.")
	}

	// Create session manager
	manager := session.NewManager(gameWorld, database)
	manager.SetCombatBroadcastFunc()                  // Enable combat messages to rooms
	manager.SetDeathFunc()                            // Enable death/respawn handling
	manager.RegisterMemoryHooks()                     // Enable narrative memory writes on kill/death
	manager.SetDamageFunc()                           // Enable HEALTH dirty-tracking for agents
	manager.SetScriptFightFunc()                      // Enable mob fight scripts after each combat round
	game.SetAICombatEngine(manager.GetCombatEngine()) // Enable AI to use combat

	// Setup HTTP routes
	http.HandleFunc("/ws", manager.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	})
	http.HandleFunc("/metrics", metrics.Handler().ServeHTTP)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`Dark Pawns Phase 1 Server

Connect via WebSocket: ws://` + r.Host + `/ws

Endpoints:
  /health - Health check
  /metrics - Prometheus metrics

Protocol:
  {"type":"login","data":{"player_name":"YourName"}}
  {"type":"command","data":{"command":"look"}}
  {"type":"command","data":{"command":"north"}}
  {"type":"command","data":{"command":"say","args":["hello"]}}
`))
	})

	// Start zone resets in background (initial + periodic every 60s)
	go func() {
		slog.Info("Starting zone resets...")
		if err := gameWorld.StartZoneResets(); err != nil {
			slog.Error("Zone reset error", "error", err)
		} else {
			slog.Info("Zone resets complete")
		}
		gameWorld.StartPeriodicResets(60 * time.Second)
	}()

	// Start server
	addr := ":" + *port
	slog.Info("Server listening", "address", addr)
	slog.Info("WebSocket endpoint", "url", "ws://localhost"+addr+"/ws")

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create handler with security middleware
	handler := web.SecurityHeaders(http.DefaultServeMux)

	// Check if TLS should be used
	useTLS := os.Getenv("USE_TLS") == "true"
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	go func() {
		if useTLS {
			if certFile == "" || keyFile == "" {
				slog.Error("TLS_CERT_FILE and TLS_KEY_FILE environment variables must be set for TLS")
				os.Exit(1)
			}
			slog.Info("Starting HTTPS server", "address", addr)
			if err := http.ListenAndServeTLS(addr, certFile, keyFile, handler); err != nil {
				slog.Error("Server error", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Warn("Starting HTTP server (not secure for production)", "address", addr)
			if err := http.ListenAndServe(addr, handler); err != nil {
				slog.Error("Server error", "error", err)
				os.Exit(1)
			}
		}
	}()

	<-sigChan
	slog.Info("Shutting down...")
}
