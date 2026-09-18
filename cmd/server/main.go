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
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"
	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/contact"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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

// Defaults for a checkout run from the repository root, so that `./server` with
// no flags does something sane. lib/world is the directory the parser expects
// (it reads wld/, mob/, obj/, zon/ and shp/ out of the path it is handed);
// passing lib/ has never worked but was the obvious guess.
const (
	defaultWorldDir = "lib/world"
	defaultWebDir   = "web/public"
)

// usage is the operator's first stop after a boot refusal, so it leads with a
// command that works in a fresh checkout and only then lists the flags. The
// checkout command needs nothing external: the server boots against an
// embedded SQLite database by default, and DATABASE_URL only matters when the
// operator opts into PostgreSQL.
func usage() {
	out := flag.CommandLine.Output()
	// Header and footer are checked separately because flag.PrintDefaults has to
	// run between them; a failure to write usage is worth a log line, not an exit.
	if _, err := fmt.Fprint(out, "Dark Pawns server\n\n"+
		"Usage:\n  server [flags]\n\n"+
		"In a checkout, from the repository root:\n\n"+
		"  export JWT_SECRET=\"$(openssl rand -hex 32)\"\n"+
		"  ./server\n\n"+
		"Boots against an embedded SQLite database in data/ by default. Set\n"+
		"DATABASE_URL (or pass -db) to use PostgreSQL instead.\n\n"+
		"Flags:\n"); err != nil {
		slog.Warn("writing usage failed", "error", err)
		return
	}
	flag.PrintDefaults()
	if _, err := fmt.Fprint(out, "\nSee DEPLOYMENT.md for database setup and the supported deployment model,\n"+
		"and `go run ./cmd/server` for a build-free local run.\n"); err != nil {
		slog.Warn("writing usage failed", "error", err)
	}
}

func main() {
	// Every path has a real default so that `./server`, run from a checkout
	// root, does something. The world tree is lib/world, not lib/: the parser
	// reads wld/, mob/, obj/ and zon/ directly out of the directory it is
	// handed, and handing it lib/ fails with "parse rooms: open lib/wld".
	var (
		worldDir   = flag.String("world", defaultWorldDir, "World data directory: the one holding wld/, mob/, obj/, zon/ and shp/")
		scriptsDir = flag.String("scripts", "", "Lua script directory (defaults to <world>/scripts)")
		port       = flag.String("port", "4350", "HTTP and WebSocket port")
		dbURL      = flag.String("db", "", "Database URL or SQLite path (falls back to DATABASE_URL env var; default: embedded SQLite beside the world data)")
		webDir     = flag.String("web", defaultWebDir, "Browser client directory served at / (index.html, client.js, style.css)")
		staticDir  = flag.String("static", "", "Static site directory served at /, takes precedence over -web")
		hugoDir    = flag.String("hugo", "", "Deprecated alias for -static; still works, warns")
		telnetPort = flag.Int("telnet-port", 7777, "Telnet port (0 to disable)")
	)
	flag.Usage = usage
	flag.Parse()
	// Hugo was removed from this repository (no config, no content, no themes);
	// the flag has been a plain file server for a while. Scripts and systemd
	// units still pass it, so keep honoring it and say once per boot that the
	// spelling moved on.
	if *hugoDir != "" {
		if *staticDir == "" {
			*staticDir = *hugoDir
		}
		slog.Warn("-hugo is deprecated; use -static instead", "hugo", *hugoDir, "static", *staticDir)
	}
	seed, err := dprng.ConfigureFromEnvironment()
	if err != nil {
		slog.Error("Invalid DP_SEED", "error", err)
		os.Exit(1)
	}
	slog.Info("Random stream initialized", "seed", seed, "deterministic", os.Getenv("DP_SEED") != "")
	if dpclock.Frozen() {
		slog.Info("DP_CLOCK enabled; real-time game pulses are frozen")
	}
	// DP_FIXED_TIME pins the Unix instant reset_time() derives the calendar
	// from, so sunlight/weather are stable independent of wall-clock MUD hour.
	// Separate from DP_CLOCK (which freezes pulses); tests needing real-time
	// pulses plus a pinned daytime clock use DP_FIXED_TIME.
	if ts, ok := game.ConfigureNowFromEnv(); ok {
		slog.Info("DP_FIXED_TIME pinned", "now", ts)
	}

	// Refuse a world directory that cannot be parsed here, where the message can
	// still name the path the operator meant. Deeper in, the parser says
	// "parse rooms: open lib/wld: no such file" — true, but it does not say
	// that -world takes lib/world and not its parent. Start dir must exist,
	// must be a directory, and must hold wld/.
	if err := validateWorldDir(*worldDir); err != nil {
		slog.Error("world directory unusable; refusing to start",
			"path", *worldDir,
			"error", err,
			"hint", "from the repository root run: ./server -world ./lib/world -web ./web/public",
			"layout", "the -world directory holds wld/, mob/, obj/, zon/ and shp/; in this checkout that is lib/world")
		os.Exit(1)
	}
	// Decide what / serves before the world parse spends seconds on files.
	// -static (a built site) wins over -web (the bundled browser client), and
	// neither is required: /ws and /api/* work without a front door.
	switch {
	case *staticDir != "":
		if err := validateDir(*staticDir); err != nil {
			slog.Error("static site directory unusable; refusing to start",
				"path", *staticDir,
				"error", err,
				"hint", "pass -static <dir>, or drop the flag to serve the bundled client")
			os.Exit(1)
		}
	case *webDir != "":
		if err := validateDir(*webDir); err != nil {
			if *webDir != defaultWebDir {
				slog.Error("web client directory unusable; refusing to start",
					"path", *webDir,
					"error", err,
					"hint", "pass -web <dir>, or -static <dir> to serve a built site instead")
				os.Exit(1)
			}
			// An installed deployment may carry the binary without this
			// checkout's web/ tree, and the operator never asked for this path,
			// so warn rather than refuse. The front door goes dark but the
			// game does not: /ws, /api/*, telnet and /health all still answer.
			slog.Warn("bundled web client not found; serving the plain-text index only",
				"path", *webDir,
				"error", err,
				"hint", "pass -web <dir> to serve the browser client")
			*webDir = ""
		}
	}
	// Anchor the process to the game root (parent of the world/lib dir)
	// before anything reads or writes a relative data/ path (DP-1193).
	// Every persistence path — data/shops.json, data/aliases/, data/mail,
	// data/bugs.txt, admin store — is CWD-relative; a daemon started from
	// the wrong directory would silently fork the data set. Relative flag
	// paths are resolved against the original CWD first.
	absWorld, err := filepath.Abs(*worldDir)
	if err != nil {
		slog.Error("resolving world path", "error", err)
		os.Exit(1)
	}
	for _, d := range []*string{worldDir, scriptsDir, webDir, staticDir} {
		if *d != "" {
			if abs, aerr := filepath.Abs(*d); aerr == nil {
				*d = abs
			}
		}
	}
	gameRoot := filepath.Dir(absWorld)
	if err := os.Chdir(gameRoot); err != nil {
		slog.Error("chdir to game root", "dir", gameRoot, "error", err)
		os.Exit(1)
	}
	slog.Info("Working directory anchored to game root", "dir", gameRoot)
	// Pin the shops persistence path explicitly (DP-1193) — belt and
	// suspenders with the chdir above.
	if err := os.Setenv("DARKPAWNS_DATA_DIR", filepath.Join(gameRoot, "data")); err != nil {
		slog.Error("failed to set persistence data directory", "error", err)
		os.Exit(1)
	}
	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
	}
	if *dbURL == "" {
		if os.Getenv("DP_ALLOW_NO_DB") == "1" {
			// The explicitly allowed no-persistence path: the empty DSN falls
			// through to db.New below, which fails, and boot continues without
			// a store. cmd/dp-oracle-diff sets this and does not guarantee a
			// DATABASE_URL in the environment it builds. Honoured here and not
			// only on connection failure so the defaulting branch cannot
			// resurrect persistence against the operator's stated intent.
		} else {
			// Boot with no external services: default to embedded SQLite beside
			// the world data. The path is anchored the same way as
			// DARKPAWNS_DATA_DIR (DP-1193): the directory holding the world
			// dir's sibling data. PostgreSQL remains the documented choice for
			// scaled deployments, not a requirement to start.
			defaultPath := filepath.Clean(filepath.Join(*worldDir, "..", "data", "darkpawns.db"))
			if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
				slog.Error("default database directory unusable; refusing to start",
					"dir", filepath.Dir(defaultPath), "error", err)
				os.Exit(1)
			}
			*dbURL = "sqlite://" + defaultPath
			slog.Info("no database configured; defaulting to embedded SQLite", "path", defaultPath)
		}
	}

	// Validate JWT signing secret at boot. A sub-32-char secret silently breaks
	// token issuance for the whole process lifetime (DP-910): GenerateJWT/
	// ValidateJWT return an error that call sites log-and-continue, so WS agent
	// clients get empty tokens and CI never notices because telnet play doesn't
	// need a token. Fail loud at startup instead, and name the command.
	//   - production: refuse to start with the export line to paste.
	//   - development: derive an ephemeral 32-byte secret so local boot works.
	if err := auth.ValidateJWTSecret(); err != nil {
		if os.Getenv("ENVIRONMENT") != "development" {
			slog.Error("JWT_SECRET invalid; refusing to start outside development",
				"error", err,
				"hint", `export JWT_SECRET="$(openssl rand -hex 32)"`,
				"development", "ENVIRONMENT=development generates an ephemeral secret instead, and issued tokens do not survive a restart")
			os.Exit(1)
		}
		ephemeral, gerr := generateEphemeralJWTSecret()
		if gerr != nil {
			slog.Error("failed to generate ephemeral dev JWT secret", "error", gerr)
			os.Exit(1)
		}
		if err := os.Setenv("JWT_SECRET", ephemeral); err != nil {
			slog.Error("failed to install ephemeral dev JWT secret", "error", err)
			os.Exit(1)
		}
		slog.Warn("JWT_SECRET missing/short in development; generated an ephemeral secret",
			"hint", "issued tokens are invalid across restarts; set JWT_SECRET for stable dev")
	}

	slog.Info("Dark Pawns Phase 1 Server Starting...")

	// C reset_time() (db.c:415-451) is the first thing boot_db does: it derives
	// the game calendar from the real-time epoch (beginning_of_time), then
	// derives sunlight, moon phase, and initial barometric pressure. That
	// pressure dice roll is the first draw from the process-wide stream,
	// ahead of mob prototypes and zone resets, so ResetTime must run before
	// the world is parsed.
	game.ResetTime()

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
	gameWorld.PostInit()

	// Connect to database
	slog.Info("Connecting to database...")
	database, err := db.New(*dbURL)
	if err != nil {
		if os.Getenv("DP_ALLOW_NO_DB") != "1" {
			slog.Error("Database initialization failed; refusing logins without persistence", "error", err)
			os.Exit(1)
		}
		slog.Warn("Database connection failed, explicitly running without persistence", "error", err)
		database = nil
	} else {
		defer func() {
			if err := database.Close(); err != nil {
				slog.Error("database close failed during server shutdown", "error", err)
			}
		}()
		slog.Info("Database connected.")
	}

	// Set ban/xnames file paths relative to world directory (DP-421)
	game.SetBanFilePaths(*worldDir)

	// Create session manager.
	// gameStore is the player-facing store; researchStore is the decision-capture
	// telemetry surface (decision_log/combat_log). Both are true-nil interfaces
	// when there is no database: a nil *db.DB stored in an interface is itself
	// non-nil, which would defeat the nil checks downstream and panic. (DP-589)
	var gameStore db.GameStore
	var researchStore db.ResearchStore
	if database != nil {
		gameStore = database
		researchStore = database
	}
	manager := session.NewManager(gameWorld, gameStore)
	if database == nil {
		// The no-DB mode is an explicitly non-persistent development/oracle
		// configuration. Its process-local IDs cannot safely address mail
		// across an offline login and restart, so do not wire an incomplete
		// online-only identity lookup.
		slog.Warn("Mail disabled: persistent player identity requires a database")
	} else if err := initializePersistentMail(gameStore); err != nil {
		// C's boot_db() sets no_mail and continues when scan_file() fails.
		// Mail is optional; do not make an unavailable mail store take down
		// the world, listener, or unrelated player sessions.
		slog.Error("Mail disabled; continuing server boot", "error", err)
	} else {
		slog.Info("Mail system initialized with persistent player identity")
	}
	gameWorld.SetShopManager(manager.GetShopManager()) // Wire shop system to world
	game.SetWeatherWorld(gameWorld)                    // Wire world for weather broadcasts
	manager.SetCombatBroadcastFunc()                   // Enable combat messages to rooms
	manager.SetDeathFunc()                             // Enable death/respawn handling
	manager.RegisterMemoryHooks()                      // Enable narrative memory writes on kill/death
	manager.SetDamageFunc()                            // Enable HEALTH dirty-tracking for agents
	manager.SetDreamingDir("data/dreaming")            // Dreaming layer output (memory summaries)

	// Decision capture (DP-213) — enabled when database is available and
	// partitions can be ensured. If partition creation fails, leave the writer
	// unset so the manager falls back to its no-op behavior and records are not
	// silently dropped during flush.
	var decisionLogWriter *db.DecisionLogWriter
	if researchStore != nil {
		if err := researchStore.EnsureDecisionLogPartitions(); err != nil {
			slog.Warn("failed to create decision log partitions; decision capture disabled", "error", err)
		} else {
			decisionLogWriter = researchStore.NewDecisionLogWriter()
			manager.SetDecisionLog(decisionLogWriter)

			// decision_log keeps raw_input, the literal line a player typed,
			// which in a MUD carries tells, says and gossip. Say so at boot
			// rather than leaving an operator to discover it from the schema.
			retain := 0
			if v := os.Getenv("DP_LOG_RETENTION_MONTHS"); v != "" {
				parsed, perr := strconv.Atoi(v)
				if perr != nil || parsed < 0 {
					slog.Error("DP_LOG_RETENTION_MONTHS must be a non-negative whole number of months; refusing to guess",
						"value", v)
					os.Exit(1)
				}
				retain = parsed
			}
			if retain > 0 {
				dropped, derr := database.DropExpiredLogPartitions(retain)
				if derr != nil {
					slog.Warn("could not drop expired log partitions", "error", derr)
				}
				if len(dropped) > 0 {
					slog.Info("dropped expired log partitions", "partitions", dropped)
				}
				slog.Info("decision capture enabled", "records", "command text including tells and says",
					"retention_months", retain)
			} else {
				slog.Info("decision capture enabled", "records", "command text including tells and says",
					"retention", "unlimited",
					"hint", "set DP_LOG_RETENTION_MONTHS to expire whole monthly partitions")
			}
			if db.UsingLegacySalt() {
				slog.Warn("decision log pseudonyms use the salt that ships in the source, so they are reversible from a player list",
					"hint", `set DP_LOG_SALT to a value of your own; it renames every pseudonym, so choose before collecting data you intend to keep`)
			}
		}
	}
	manager.SetScriptFightFunc()                         // Enable mob fight scripts after each combat round
	manager.SetMobSpecialFunc()                          // Enable combat-time native mob specials
	manager.SetScriptDeathFunc()                         // Enable mob death scripts on kill
	manager.SetOnRoundEnd()                              // Decrement wait states each combat round
	manager.SetCommandExecFunc()                         // Wire doOrder command dispatch for charmed followers
	gameWorld.SetCombatEngine(manager.GetCombatEngine()) // Enable AI to use combat
	manager.WireCombatCallbacks()                        // Wire PR2/PR3 character-state hooks into combat engine
	manager.SetCombatMessageFunc()                       // Wire DamMessage() and GameCallbacks for live combat
	manager.SetFleeHooks()                               // Wire wimpy auto-flee after callback replacement (DP-389)

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
		OnDrainInput: func() {
			// DP-1201: per-pulse command drain (comm.c:603). Drains one queued
			// command per session when its wait reaches 0. Runs every tick, at
			// the top of heartbeat before OnPerformViolence.
			manager.DrainInputQueues()
		},
		OnEventProcess: func() {
			gameWorld.EventQueue.Process(context.Background())
		},
		OnExtractPending: func() {
			manager.ExtractPendingChars()
		},
		OnPerformViolence: func() {
			// Production combat keeps its standalone ticker until the Phase 2
			// unification. Pumped DP_CLOCK heartbeats must dispatch C's
			// perform_violence slot, which is a no-op before combat begins.
			if dpclock.Frozen() {
				manager.GetCombatEngine().PerformRound()
			}
		},
		OnMobileActivity: func() {
			gameWorld.MobileActivity()
		},
		// comm.c:690 room_activity — FLAMING/UNDERWATER/WATER_NOSWIM fixed
		// self-damage, pulse-time room specs, and FLYING-sector falls, in C's
		// heartbeat position right after mobile_activity.
		OnRoomActivity: func() {
			gameWorld.RoomActivity()
		},
		// object_activity remains an explicit no-op seam.
		OnObjectActivity: func() {},
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
	manager.SetPulsePump(gameLoop.PumpPulses)
	// loopCtx ties the heartbeat to the server lifetime: canceling it drains the
	// loop the same way gameLoop.Stop() does (DP-892). Stop() remains the primary
	// signal-driven shutdown path below; loopCancel is the belt-and-suspenders.
	loopCtx, loopCancel := context.WithCancel(context.Background())
	defer loopCancel()
	gameLoop.Start(loopCtx)

	// Setup HTTP routes
	http.HandleFunc("/ws", manager.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK\n")); err != nil {
			slog.Warn("health check write failed", "error", err)
		}
	})
	// Gauges describe state, not events, so they are sampled rather than
	// maintained. Tracking every mutation means finding every mutation, and one
	// missed path leaves the gauge wrong until restart; re-reading the truth on
	// a timer cannot drift. Ten seconds is well inside a normal scrape interval.
	//
	// Until this existed, /metrics answered 200 with every gauge reading zero:
	// the collectors were registered and nothing ever wrote them, so a scraper
	// saw a healthy target reporting an empty world. Silence that looks like
	// data is worse than an endpoint that is honestly absent.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				metrics.SetPlayersOnline(manager.SessionCount())
				metrics.SetRoomsActive(len(gameWorld.Rooms()))
				metrics.SetMobsActive(len(gameWorld.GetAllMobs()))
			}
		}
	}()
	contactHandler, err := contact.NewFromEnvironment()
	if err != nil {
		slog.Warn("Website contact form disabled", "error", err)
		http.Handle("/api/contact", contact.UnavailableHandler())
	} else {
		http.Handle("/api/contact", contactHandler)
	}
	// Serve the front door: -static wins, then the bundled browser client, then
	// a plain-text index. Both directories were validated at startup, so the
	// only way here with an empty field is a deliberate omission.
	if *staticDir != "" {
		fs := revalidated(http.FileServer(http.Dir(*staticDir)))
		http.Handle("/", fs)
		slog.Info("Serving static site", "path", *staticDir)
	} else if *webDir != "" {
		fs := revalidated(http.FileServer(http.Dir(*webDir)))
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

	// Publish the API contract without authentication so clients can discover
	// how to authenticate before making a protected request.
	openAPIHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		http.ServeFile(w, r, "web/api/openapi.json")
	}
	http.HandleFunc("/openapi.json", openAPIHandler)
	http.HandleFunc("/api/openapi.json", openAPIHandler)

	// All other /api/ endpoints require a JWT bearer token.
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		web.WriteJSONError(w, http.StatusNotFound, "ENDPOINT_NOT_FOUND", "The requested API endpoint does not exist.", "Consult /openapi.json for supported endpoints.")
	})
	http.Handle("/api/", web.AuthMiddleware(apiMux))

	// Admin routes — JWT-protected, role-gated
	// The log belongs beside the instance's other runtime state, under the game
	// root the process anchored to above (lib/ in a checkout). Naming it
	// absolutely keeps the warning below pointing at the file it means, instead
	// of at a logs/ directory that only exists relative to a CWD nobody chose.
	auditLogPath := filepath.Join(gameRoot, "logs", "audit.log")
	auditLogger, err := audit.NewAuditLogger(auditLogPath)
	if err != nil {
		// Non-fatal by design. NewAuditLogger creates the directory itself, so
		// what is left is a real permissions problem, and refusing to boot the
		// game over an admin-only log would be disproportionate: telnet, /ws,
		// /health and the world do not need it. Name what stops being recorded,
		// because a silent audit trail is indistinguishable from a clean one.
		slog.Warn("Admin audit trail disabled: world edits through /admin/ will not be recorded",
			"error", err, "path", auditLogPath)
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
	http.Handle("/assets/", http.StripPrefix("/assets/", fingerprinted(http.FileServer(http.Dir("admin-ui-dist/assets")))))

	// Track the production zone reset goroutine for graceful shutdown.
	// DP_CLOCK performs this initial population synchronously so the harness's
	// readiness marker guarantees fixtures exist before character setup.
	var wg sync.WaitGroup
	initializeWorld := func() {
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
	}
	if dpclock.Frozen() {
		initializeWorld()
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			initializeWorld()
		}()
	}

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

	shutdownFromCommand := false
	select {
	case <-sigChan:
		slog.Info("Received shutdown signal")
	case request := <-manager.ShutdownRequests():
		if request.Marker != "" {
			markerPath := filepath.Join("..", request.Marker)
			file, markerErr := os.OpenFile(filepath.Clean(markerPath), os.O_CREATE|os.O_WRONLY, 0o640)
			if markerErr != nil {
				slog.Error("failed to write shutdown marker", "path", markerPath, "error", markerErr)
			} else if closeErr := file.Close(); closeErr != nil {
				slog.Error("failed to close shutdown marker", "path", markerPath, "error", closeErr)
			}
		}
		slog.Info("Shutdown requested by command", "marker", request.Marker)
		// The command already sent C's global text and all-save output.
		shutdownFromCommand = true
	case err := <-errChan:
		slog.Error("Server error, shutting down gracefully", "error", err)
	}
	slog.Info("Shutting down gracefully...")

	// 1. Stop standalone world tickers, then stop heartbeat callbacks. A
	// heartbeat callback can be doing slow world work, so bound the wait well
	// below systemd's stop timeout instead of allowing SIGKILL to decide.
	// The AI ticker and point update ticker share the World's done channel;
	// StopAITicker closes it and stops both. StopPeriodicResets ends the
	// zone-reset goroutine started in the boot goroutine below.
	gameWorld.StopAITicker()
	gameWorld.StopPeriodicResets()
	loopCancel()
	heartbeatCtx, heartbeatCancel := context.WithTimeout(context.Background(), 10*time.Second)
	heartbeatErr := gameLoop.StopContext(heartbeatCtx)
	heartbeatCancel()
	heartbeatStopped := heartbeatErr == nil
	if heartbeatErr != nil {
		slog.Error("Heartbeat did not stop before shutdown deadline; world snapshot will be skipped", "error", heartbeatErr)
	}

	// 2. Stop telnet listener (accepting new TCP connections)
	telnet.Stop()

	// 3. Stop HTTP/WebSocket server (accepting new WebSocket connections)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	// 4. Drain active player sessions (stops combat, broadcasts leave, saves profiles, closes connections)
	if shutdownFromCommand {
		manager.ShutdownGracefullyWithoutNotice(5 * time.Second)
	} else {
		manager.ShutdownGracefully(5 * time.Second)
	}

	// 5. Flush buffered decision/combat records before closing the database.
	if decisionLogWriter != nil {
		decisionLogWriter.Stop()
	}

	// Wait for zone resets to finish before saving — prevents concurrent
	// writes to world state from corrupting the save file.
	wg.Wait()

	// Save dynamic world state only after the heartbeat is proven quiescent.
	// Saving concurrently with a stuck callback risks a corrupt snapshot; player
	// profiles have already been drained independently above.
	if heartbeatStopped {
		if err := game.SaveWorld(gameWorld); err != nil {
			slog.Error("Failed to save world state", "error", err)
		}
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

// validateDir reports, in operator terms, why a directory cannot be served.
func validateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

// validateWorldDir checks the two mistakes that cost the most time to diagnose:
// a -world path that does not exist, and a -world path one level too high
// (lib/ instead of lib/world), which the room parser reports only as a missing
// lib/wld file several steps later.
func validateWorldDir(dir string) error {
	if err := validateDir(dir); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "wld")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no wld/ subdirectory: -world wants the directory holding wld/, mob/, obj/, zon/ and shp/, not its parent")
		}
		return err
	}
	return nil
}

// fatal logs an error message and exits the process. Kept in a helper so that
// early validation failures in main do not trip gocritic's exitAfterDefer check.
func fatal(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Cache policy for static files, which differs by whether the filename carries
// a build hash.
//
// The front door ships unversioned names — style.css, client.js — and sent no
// Cache-Control at all, leaving browsers to cache it heuristically. A browser
// that had stored the pre-restructure stylesheet went on using it, so the page
// rendered new markup against the old rules: the wordmark lost its .brand rule
// and fell back to the user-agent's link blue. "no-cache" does not forbid
// storing, only using a stored copy without asking, so Last-Modified still
// yields a 304 and the body is sent again only when it has actually changed.
func revalidated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

// Vite fingerprints the admin bundle, so a changed file is a changed URL and
// the old one is never requested again. Those can be cached hard.
func fingerprinted(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.ServeHTTP(w, r)
	})
}
