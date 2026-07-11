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

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/grapevine"
	"github.com/zax0rz/darkpawns/pkg/metrics"
	"github.com/zax0rz/darkpawns/pkg/moderation"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/telnet"
	"github.com/zax0rz/darkpawns/web"
)

func main() {
	var (
		worldDir   = flag.String("world", "", "Path to world files (lib directory)")
		scriptsDir = flag.String("scripts", "", "Path to Lua scripts (defaults to world/lib/scripts)")
		port       = flag.String("port", "4350", "Server port")
		dbURL      = flag.String("db", "", "Database URL (falls back to DATABASE_URL env var)")
		webDir     = flag.String("web", "", "Path to web client files (index.html, client.js, style.css)")
		hugoDir    = flag.String("hugo", "", "Path to Hugo static site (served as root)")
		telnetPort = flag.Int("telnet-port", 7777, "Telnet port (0 to disable)")
	)
	flag.Parse()

	if *worldDir == "" {
		slog.Error("Usage: server -world <path-to-lib>")
		os.Exit(1)
	}
	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
	}
	if *dbURL == "" {
		slog.Error("Database URL is required; pass -db or set DATABASE_URL")
		os.Exit(1)
	}

	// Validate JWT signing secret at boot. A sub-32-char secret silently breaks
	// token issuance for the whole process lifetime (DP-910): GenerateJWT/
	// ValidateJWT return an error that call sites log-and-continue, so WS agent
	// clients get empty tokens and CI never notices because telnet play doesn't
	// need a token. Fail loud at startup instead.
	//   - production: refuse to start with a clear message.
	//   - development: derive an ephemeral 32-byte secret so local boot works.
	if err := auth.ValidateJWTSecret(); err != nil {
		if os.Getenv("ENVIRONMENT") != "development" {
			slog.Error("JWT_SECRET invalid; refusing to start in production",
				"error", err,
				"hint", "set JWT_SECRET to a >=32 char value, e.g. openssl rand -hex 32")
			os.Exit(1)
		}
		ephemeral, gerr := generateEphemeralJWTSecret()
		if gerr != nil {
			slog.Error("failed to generate ephemeral dev JWT secret", "error", gerr)
			os.Exit(1)
		}
		_ = os.Setenv("JWT_SECRET", ephemeral)
		slog.Warn("JWT_SECRET missing/short in development; generated an ephemeral secret",
			"hint", "issued tokens are invalid across restarts; set JWT_SECRET for stable dev")
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
	gameWorld.WorldPath = *worldDir

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

	// Set ban/xnames file paths relative to world directory (DP-421)
	game.SetBanFilePaths(*worldDir)

	// Create session manager.
	// Pass a true-nil db.Database interface when there is no database: a nil
	// *db.DB stored in an interface is itself non-nil, which would defeat the
	// nil checks downstream and panic. (DP-589)
	var dbIface db.Database
	if database != nil {
		dbIface = database
	}
	manager := session.NewManager(gameWorld, dbIface)
	gameWorld.SetShopManager(manager.GetShopManager()) // Wire shop system to world
	game.SetWeatherWorld(gameWorld)                    // Wire world for weather broadcasts
	manager.SetCombatBroadcastFunc()                   // Enable combat messages to rooms
	manager.SetDeathFunc()                             // Enable death/respawn handling
	manager.SetFleeHooks()                             // Wire wimpy auto-flee (DP-389)
	manager.RegisterMemoryHooks()                      // Enable narrative memory writes on kill/death
	manager.SetDamageFunc()                            // Enable HEALTH dirty-tracking for agents
	manager.SetDreamingDir("data/dreaming")            // Dreaming layer output (memory summaries)

	// Decision capture (DP-213) — enabled when database is available and
	// partitions can be ensured. If partition creation fails, leave the writer
	// unset so the manager falls back to its no-op behavior and records are not
	// silently dropped during flush.
	var decisionLogWriter *db.DecisionLogWriter
	if database != nil {
		if err := database.EnsureDecisionLogPartitions(); err != nil {
			slog.Warn("failed to create decision log partitions; decision capture disabled", "error", err)
		} else {
			decisionLogWriter = database.NewDecisionLogWriter()
			manager.SetDecisionLog(decisionLogWriter)
			slog.Info("decision capture enabled")
		}
	}
	manager.SetScriptFightFunc()                         // Enable mob fight scripts after each combat round
	manager.SetScriptDeathFunc()                         // Enable mob death scripts on kill
	manager.SetOnRoundEnd()                              // Decrement wait states each combat round
	manager.SetCommandExecFunc()                         // Wire doOrder command dispatch for charmed followers
	gameWorld.SetCombatEngine(manager.GetCombatEngine()) // Enable AI to use combat
	manager.WireCombatCallbacks()                        // Wire PR2/PR3 character-state hooks into combat engine
	manager.SetCombatMessageFunc()                       // Wire DamMessage() and GameCallbacks for live combat

	// Verify critical combat hooks are wired (DP-952).
	// This check is placed before any long-lived resource with a defer so that
	// an early exit does not skip cleanup.
	if err := manager.GetCombatEngine().ValidateCallbacks(); err != nil {
		fatal("critical combat hook not wired — refusing to start: %v", err)
	}

	// Initialize scripting engine
	if *scriptsDir == "" {
		*scriptsDir = *worldDir + "/scripts"
	}
	slog.Info("Loading scripts", "path", *scriptsDir)
	worldAdapter := game.NewWorldScriptableAdapter(gameWorld)
	scriptEngine := scripting.NewEngine(*scriptsDir, worldAdapter)
	defer scriptEngine.Close()
	game.ScriptEngine = scriptEngine

	// Initialize and start Grapevine WebSocket Client in background
	gvClient := grapevine.NewClient(gameWorld)
	gvClient.Start()
	defer gvClient.Stop()

	// Wire moderation: mute, ban, word filter, spam detection
	if database != nil {
		modManager := moderation.NewManager(database.SQLDB())
		modAdapter := session.NewModerationAdapter(modManager)
		manager.SetModerationChecker(modAdapter)
		slog.Info("Moderation manager wired with database backend")
	} else {
		slog.Warn("No database — moderation disabled (mute/ban/spam filters unavailable)")
	}

	// Initialize board system (DP-422)
	gameWorld.GetOrInitBoards(*worldDir)
	gameWorld.Boards.SetWorld(gameWorld)

	// Start event queue (MobProg delayed events, etc.).
	// Mob AI is driven by the game loop's OnMobileActivity below (PULSE_MOBILE
	// = 4s), faithful to C's mobile_activity() (DP-1035).
	gameWorld.StartEventQueue()

	// Start game loop (heartbeat, mobile activity, combat ticks).
	// PointUpdate is driven by World's standalone 63s ticker, not this loop.
	gameLoop := engine.NewGameLoop(engine.GameLoopCallbacks{
		OnPerformViolence: func() {
			// Combat engine handles its own 2s tick via CombatEngine.Start()
		},
		OnMobileActivity: func() {
			gameWorld.MobileActivity()
		},
		OnWeatherAndTime: func() {
			game.WeatherAndTime(true, manager.SendToOutdoor)
		},
		OnAffectUpdate: func() {
			gameWorld.AffectUpdate()
		},
		OnCheckIdlePasswords: func() {
			manager.CheckIdlePasswords()
		},
		OnReapLinkdeadSessions: func() {
			manager.ReapLinkdeadSessions()
		},
	})
	// loopCtx ties the heartbeat to the server lifetime: canceling it drains the
	// loop the same way gameLoop.Stop() does (DP-892). Stop() remains the primary
	// signal-driven shutdown path below; loopCancel is the belt-and-suspenders.
	loopCtx, loopCancel := context.WithCancel(context.Background())
	defer loopCancel()
	gameLoop.Start(loopCtx)
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
	// Serve Hugo static site if -hugo flag provided
	// Falls back to -web flag for legacy web client, then plain text index
	if *hugoDir != "" {
		fs := http.FileServer(http.Dir(*hugoDir))
		http.Handle("/", fs)
		slog.Info("Serving Hugo site", "path", *hugoDir)
	} else if *webDir != "" {
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

	// Admin routes — JWT-protected, role-gated
	auditLogger, err := audit.NewAuditLogger("logs/audit.log")
	if err != nil {
		slog.Warn("Failed to create audit logger, admin audit trail disabled", "error", err)
	}

	// Log buffer for admin operations panel — captures slog output in-memory
	logBuffer := admin.NewLogBuffer(1000)
	// Wire slog to also write to the buffer.
	// We use a fresh TextHandler writing to os.Stderr as the base, NOT
	// slog.Default().Handler(), because SetDefault with a wrapping handler
	// that references the old default creates a recursive lock in Go 1.26+.
	baseHandler := slog.NewTextHandler(os.Stderr, nil)
	logHandler := admin.NewSlogHandler(baseHandler, logBuffer)
	slog.SetDefault(slog.New(logHandler))

	adminRouter, err := admin.NewRouter(gameWorld, auditLogger, logBuffer, database, manager)
	if err != nil {
		fatal("failed to init admin router: %v", err)
	}
	// Health endpoint is unauthenticated — registered before the auth-wrapped catch-all
	http.HandleFunc("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			slog.Warn("admin health write failed", "error", err)
		}
	})
	http.Handle("/admin/", adminRouter)

	// Serve admin UI static assets (compiled React app)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("admin-ui-dist/assets"))))

	// Track zone reset goroutine for graceful shutdown
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
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

		// Build initial spec-room cache now that mobs/items are in place.
		gameWorld.RebuildSpecRooms()

		gameWorld.StartPeriodicResets(60 * time.Second)
	}()

	// Start server
	addr := ":" + *port
	slog.Info("Server listening", "address", addr)
	slog.Info("WebSocket endpoint", "url", "ws://localhost"+addr+"/ws")

	// Start telnet listener
	if *telnetPort > 0 {
		if err := telnet.Listen(*telnetPort, manager); err != nil {
			slog.Error("Telnet listener failed", "error", err)
		} else {
			slog.Info("Telnet listening", "port", *telnetPort)
		}
	}

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create handler with security middleware
	handler := web.SecurityHeaders(http.DefaultServeMux)

	// TLS configuration:
	//   - Cert + key both set → TLS enabled automatically
	//   - USE_TLS=true but certs missing → fatal (explicit intent, broken config)
	//   - No certs → plaintext with warning
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	useTLS := os.Getenv("USE_TLS") == "true"

	if useTLS && (certFile == "" || keyFile == "") {
		slog.Error("USE_TLS=true but TLS_CERT_FILE and TLS_KEY_FILE are not set")
		gameLoop.Stop()
		os.Exit(1) //nolint:gocritic // exitAfterDefer: gameLoop.Stop() called explicitly above
	}
	haveCerts := certFile != "" && keyFile != ""

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	errChan := make(chan error, 1)

	go func() {
		if haveCerts {
			slog.Info("Starting HTTPS server", "address", addr)
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		} else {
			slog.Warn("TLS disabled — WebSocket and API traffic is unencrypted. Set TLS_CERT_FILE and TLS_KEY_FILE for production.", "address", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}
	}()

	select {
	case <-sigChan:
		slog.Info("Received shutdown signal")
	case err := <-errChan:
		slog.Error("Server error, shutting down gracefully", "error", err)
	}
	slog.Info("Shutting down gracefully...")

	// 1. Stop heartbeat callbacks before draining sessions or saving world state.
	gameLoop.Stop()

	// 1a. Stop standalone world tickers so they cannot mutate state during save.
	// The AI ticker and point update ticker share the World's done channel;
	// StopAITicker closes it and stops both. StopPeriodicResets ends the
	// zone-reset goroutine started in the boot goroutine below.
	gameWorld.StopAITicker()
	gameWorld.StopPeriodicResets()

	// 2. Stop telnet listener (accepting new TCP connections)
	telnet.Stop()

	// 3. Stop HTTP/WebSocket server (accepting new WebSocket connections)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	// 4. Drain active player sessions (stops combat, broadcasts leave, saves profiles, closes connections)
	manager.ShutdownGracefully(5 * time.Second)

	// 5. Flush buffered decision/combat records before closing the database.
	if decisionLogWriter != nil {
		decisionLogWriter.Stop()
	}

	// Wait for zone resets to finish before saving — prevents concurrent
	// writes to world state from corrupting the save file.
	wg.Wait()

	// Save dynamic world state before exit.
	if err := game.SaveWorld(gameWorld); err != nil {
		slog.Error("Failed to save world state", "error", err)
	}
	slog.Info("Shutdown complete. Farewell.")
}

// generateEphemeralJWTSecret returns a random hex-encoded 32-byte secret for
// development boots where JWT_SECRET is unset or too short. The result is 64
// hex chars (>= auth.MinJWTSecretLength). Used only when ENVIRONMENT=development.
func generateEphemeralJWTSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// fatal logs an error message and exits the process. Kept in a helper so that
// early validation failures in main do not trip gocritic's exitAfterDefer check.
func fatal(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
