// ARCHITECTURAL NOTE [M-07]: Manual wiring with no lifecycle management
//
// This main.go constructs and wires all dependencies in init-order-dependent
// imperative code. Initialization must happen in exact sequence:
//   1. Parse world files
//   2. Create game world
//   3. Init scripting engine (depends on world)
//   4. Connect to database (optional, graceful fallback)
//   5. Create session manager (depends on world + db)
//   6. Register manager hooks: combat broadcast, death, memory, damage, scripts, parry/dodge
//   7. Setup HTTP routes
//   8. Start zone reset goroutine
//   9. Start HTTP server
// 10. Block on signal for shutdown
//
// Problems:
//   - Init order is implicit and fragile — reordering breaks at runtime.
//   - No graceful shutdown of in-flight connections or goroutines.
//   - No centralized error handling for partial-init failures.
//   - Hook registration is scattered across multiple Set*Func() calls.
//
// Suggested improvement: App struct with explicit Start/Stop lifecycle.
//   type App struct {
//       world    *game.World
//       db       *db.DB
//       manager  *session.Manager
//       script   *scripting.Engine
//       server   *http.Server
//   }
//   func (a *App) Start(ctx context.Context) error  // init all, start serving
//   func (a *App) Stop(ctx context.Context) error    // graceful drain + cleanup
//
// Deferred to future refactor. See RESEARCH-LOG.md [DESIGN].

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
	"github.com/zax0rz/darkpawns/pkg/engine"
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
		webDir     = flag.String("web", "", "Path to web client files (index.html, client.js, style.css)")
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
		defer func() { _ = database.Close() }()
		slog.Info("Database connected.")
	}

	// Create session manager
	manager := session.NewManager(gameWorld, database)
	gameWorld.SetShopManager(manager.GetShopManager()) // Wire shop system to world
	game.SetWeatherWorld(gameWorld)                     // Wire world for weather broadcasts
	manager.SetCombatBroadcastFunc()                  // Enable combat messages to rooms
	manager.SetDeathFunc()                            // Enable death/respawn handling
	manager.RegisterMemoryHooks()                     // Enable narrative memory writes on kill/death
	manager.SetDamageFunc()                           // Enable HEALTH dirty-tracking for agents
	manager.SetScriptFightFunc()                      // Enable mob fight scripts after each combat round
	manager.SetParryDodgeFuncs()                      // Enable parry/dodge checks (C-11)
	game.SetAICombatEngine(manager.GetCombatEngine()) // Enable AI to use combat

	// Start game loop (heartbeat, point_update, mobile activity, combat ticks)
	gameLoop := engine.NewGameLoop(engine.GameLoopCallbacks{
		OnPointUpdate: func() {
			gameWorld.PointUpdate()
		},
		OnPerformViolence: func() {
			// Combat engine handles its own 2s tick via CombatEngine.Start()
		},
		OnMobileActivity: func() {
			// Future: mob AI wandering, speech triggers
		},
	})
	gameLoop.Start()
	defer gameLoop.Stop()

	// Setup HTTP routes
	http.HandleFunc("/ws", manager.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK\n")); err != nil {
			slog.Warn("health check write failed", "error", err)
		}
	})
	http.HandleFunc("/metrics", metrics.Handler().ServeHTTP)
	// Serve web client if -web flag provided, otherwise plain text index
	if *webDir != "" {
		fs := http.FileServer(http.Dir(*webDir))
		http.Handle("/", fs)
		slog.Info("Serving web client", "path", *webDir)
	} else {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			if _, err := w.Write([]byte("Dark Pawns Server\nWebSocket: ws://" + r.Host + "/ws\n")); err != nil {
				slog.Warn("index page write failed", "error", err)
			}
		})
	}

	// Setup API handler chain: Auth → ContentNegotiation
	// The ContentNegotiationMiddleware serves OpenAPI spec and JSON responses.
	// AuthMiddleware protects all /api/ endpoints with JWT bearer tokens.
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/api/openapi.json")
	})
	apiMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"error": "API endpoint not found", "docs": "/api/openapi.json"}`)); err != nil {
			slog.Warn("API 404 write failed", "error", err)
		}
	})
	http.Handle("/api/", web.AuthMiddleware(apiMux))

	// Start zone resets in background (initial + periodic every 60s)
	go func() {
		slog.Info("Starting zone resets...")
		if err := gameWorld.StartZoneResets(); err != nil {
			slog.Error("Zone reset error", "error", err)
		} else {
			slog.Info("Zone resets complete")
		}

		// Restore dynamic world state (door states, mob positions, room items, gossip)
		// AFTER zone resets have spawned mobs.
		if err := game.LoadWorld(gameWorld); err != nil {
			slog.Error("Failed to load world state", "error", err)
		} else {
			slog.Info("World state restored")
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
			srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
				slog.Error("Server error", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Warn("Starting HTTP server (not secure for production)", "address", addr)
			srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
			if err := srv.ListenAndServe(); err != nil {
				slog.Error("Server error", "error", err)
				os.Exit(1)
			}
		}
	}()

	<-sigChan
	slog.Info("Shutting down...")

	// Save dynamic world state before exit.
	if err := game.SaveWorld(gameWorld); err != nil {
		slog.Error("Failed to save world state", "error", err)
	}
}
