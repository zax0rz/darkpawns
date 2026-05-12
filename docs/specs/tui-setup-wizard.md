# Dark Pawns Setup Wizard — Specification

**Status:** Draft
**Author:** Daeron (scoping), The Architect (direction)
**Date:** 2026-05-09
**Type:** First feature — medium lift

---

## 1. What This Is

A setup wizard for Dark Pawns that runs in the terminal, generates a config file, and gets the server running. It replaces the current flag-based startup with a guided first-run experience.

**Not in scope:** A persistent dashboard TUI, live server monitoring, admin panels. Those are future projects. This wizard runs once, does its job, and exits.

---

## 2. Current State (What Exists Today)

### Server Startup (`cmd/server/main.go`)

The server takes 5 CLI flags:

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-world` | `""` | **Yes** | Path to world lib directory |
| `-scripts` | `""` | No | Path to Lua scripts (defaults to `world/lib/scripts`) |
| `-port` | `"4350"` | No | TCP listen port |
| `-db` | `postgres://postgres:postgres@localhost/darkpawns?sslmode=disable` | No | PostgreSQL URL |
| `-web` | `""` | No | Path to web client files |

Additional env vars read at runtime:
- `USE_TLS` — enable TLS (`"true"`)
- `TLS_CERT_FILE` — cert path
- `TLS_KEY_FILE` — key path
- `JWT_SECRET` — signing key (read in `pkg/auth/jwt.go`)

**No config file exists.** Everything is flags + env vars. There is no config loading code anywhere in the server.

### World Data Structure

The parser expects a `libDir` with this layout:

```
libDir/
  wld/    # Room files (*.wld)
  mob/    # Mob files (*.mob)
  obj/    # Object files (*.obj)
  zon/    # Zone files (*.zon)
  shp/    # Shop files (*.shp)
  scripts/  # Lua mob scripts
  etc/    # Player data, clans, mail
  text/   # Help files
  misc/   # Socials database
```

The repo ships world data in `lib/`. The server binary does NOT embed it — it reads from a filesystem path at runtime.

### Database

PostgreSQL is optional. If the connection fails, the server continues without persistence (no player saves, no moderation, no mail). The docker-compose includes Postgres + Redis. Redis is only used by the AI agent container, not the core server.

### Build Artifacts

- Binary: `go build -o darkpawns ./cmd/server`
- Docker: `Dockerfile` builds a multi-stage image (Go builder → Python builder → Alpine runtime)
- Compose: `docker-compose.yml` includes server, postgres, redis, ai-agent

---

## 3. What the Wizard Produces

One artifact: `darkpawns.yaml` — a config file that the server reads at startup.

```yaml
# darkpawns.yaml — Server Configuration
server:
  port: 4350
  host: "0.0.0.0"          # optional, default 0.0.0.0

world:
  path: "./lib"             # path to world lib directory

scripts:
  path: "./lib/scripts"     # path to Lua scripts (default: world.path/scripts)

database:
  url: "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable"
  enabled: true             # false = run without persistence

tls:
  enabled: false
  cert_file: ""
  key_file: ""

web:
  enabled: false
  path: ""                  # path to web client files

auth:
  jwt_secret: ""            # auto-generated if empty
```

The wizard validates every value before writing. The config file is the single source of truth — after setup, no flags are needed.

---

## 4. Wizard Flow (7 Screens)

### Screen 0: Pre-Flight Check

Runs before the TUI renders. Checks:
- **Terminal size:** minimum 80×24. If smaller, degrade gracefully (no ASCII art, no borders).
- **Go version:** `go version` — warn if <1.21.
- **Port availability:** attempt `net.Listen("tcp", ":PORT")` for the proposed port. If blocked, warn and suggest an alternative.
- **Root check:** if `os.Getuid() == 0`, print a warning. Don't refuse — some Docker flows run as root. But warn loudly.
- **Existing config:** if `darkpawns.yaml` exists, offer to edit it (rehydrate the TUI with current values) or overwrite.

### Screen 1: Welcome

```
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║   ██████╗ ██████╗ ██╗ ██████╗███████╗                    ║
║   ██╔══██╗██╔══██╗██║██╔════╝██╔════╝                    ║
║   ██║  ██║██████╔╝██║██║     █████╗                      ║
║   ██║  ██║██╔══██╗██║██║     ██╔══╝                      ║
║   ██████╔╝██║  ██║██║╚██████╗███████╗                    ║
║   ╚═════╝ ╚═╝  ╚═╝╚═╝ ╚═════╝╚══════╝                   ║
║                                                          ║
║          P  A  W  N  S                                   ║
║                                                          ║
║   Setup Wizard                                           ║
║                                                          ║
║   Welcome to Dark Pawns. This wizard will configure      ║
║   your server and generate a config file.                ║
║                                                          ║
║   [Enter] Continue    [Ctrl+C] Quit                      ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
```

**If terminal is too narrow for the box:** fall back to a single-line title: `Dark Pawns Setup — Step 1/6`

### Screen 2: World Data Path

```
  World Data Location

  Path to your Dark Pawns world files (lib directory):
  Contains wld/, mob/, obj/, zon/ subdirectories.

  > /Users/zach/darkpawns/lib              [_______________]

  [Enter] Continue    [Esc] Back    [Tab] Browse

  ✓ Found: 10,057 rooms, 1,319 mobs, 1,661 objects, 95 zones
```

**Validation:**
- Path must exist
- Must contain `wld/` and `mob/` subdirectories (minimum)
- Run `parser.ParseWorld()` on the path — show world stats if successful
- If parse fails, show the error and let the user fix the path
- Default: `./lib` (relative to where the wizard is run)

**Browse mode:** Tab key opens a simple directory picker (Bubbles list component). Not essential for MVP — could ship without it.

### Screen 3: Server Port

```
  Server Port

  TCP port for player connections:

  > 4350                          [_______________]

  [Enter] Continue    [Esc] Back

  Default: 4350 (standard Dark Pawns port)
  Players connect via: telnet://localhost:4350
```

**Validation:**
- Must be a valid port number (1–65535)
- Attempt `net.Listen("tcp", ":PORT")` — if it fails, the port is in use
- Show the bind error immediately, don't wait for execution
- Default: `4350`

### Screen 4: Database

```
  Database Configuration

  PostgreSQL provides player saves, moderation, and mail.

  > Enable database?  [Yes] / No

  Connection URL:
  > postgres://postgres:postgres@localhost/darkpawns?sslmode=disable
                                                          [_______________]

  [Enter] Continue    [Esc] Back

  ✓ Connection successful — database "darkpawns" ready
  OR
  ✗ Connection failed — server will run without persistence
```

**Validation:**
- If enabled, attempt `sql.Open("postgres", url)` with a 3-second timeout
- Show success/failure immediately
- If failure: offer to continue without DB (server supports this), or go back and fix the URL
- Default: enabled, with the current default URL
- If the user chooses "No database": set `database.enabled: false` in the config

### Screen 5: Optional Services

```
  Optional Services

  Web Client:
  > Enable web client?  Yes / [No]
  Path to web files: [_______________]

  TLS:
  > Enable TLS?  Yes / [No]
  Cert file: [_______________]
  Key file:  [_______________]

  [Enter] Continue    [Esc] Back

  These can be configured later in darkpawns.yaml
```

**Defaults:** all disabled. This screen is mostly "press Enter to skip." The point is to let power users set these up now if they want to, without needing to edit YAML later.

### Screen 6: Review & Generate

```
  Configuration Summary

  World:    /Users/zach/darkpawns/lib (10,057 rooms)
  Port:     4350
  Database: postgres://localhost/darkpawns ✓ connected
  Web:      disabled
  TLS:      disabled

  Config file: ./darkpawns.yaml

  [Enter] Generate & Start    [Esc] Back    [S] Save only (don't start)
```

**Actions:**
- Write `darkpawns.yaml` to the current directory
- Generate a random JWT secret if none was provided (32-byte hex string)
- If "Generate & Start": immediately start the server using the generated config
- If "Save only": write the config and exit with a success message

### Screen 7: Execution

```
  Starting Dark Pawns...

  [spinner] Loading world files...
  [✓] 10,057 rooms loaded
  [✓] 1,319 mobs loaded
  [✓] 1,661 objects loaded
  [✓] 95 zones loaded
  [spinner] Initializing scripting engine...
  [✓] 1 Lua script loaded
  [spinner] Connecting to database...
  [✓] Database connected
  [spinner] Starting zone resets...
  [✓] Zone resets complete
  [spinner] Listening on port 4350...

  ┌─────────────────────────────────────────┐
  │  Dark Pawns is running!                  │
  │                                          │
  │  Connect:  telnet://localhost:4350        │
  │  Web:      http://localhost:4350          │
  │  Health:   http://localhost:4350/health   │
  │                                          │
  │  [Ctrl+C] to stop                        │
  └─────────────────────────────────────────┘
```

**Error handling:** If any step fails, show the error inline and offer retry or quit. Don't crash the TUI.

---

## 5. Architecture

### Package Layout

```
cmd/
  server/main.go          # existing — add config file loading
  setup/main.go           # NEW — entry point for the wizard
pkg/
  config/
    config.go             # NEW — config struct + YAML loading
    defaults.go           # NEW — default values
  setup/
    wizard.go             # NEW — Bubble Tea model + step orchestration
    steps/
      welcome.go          # Screen 1
      worldpath.go        # Screen 2
      port.go             # Screen 3
      database.go         # Screen 4
      optional.go         # Screen 5
      review.go           # Screen 6
      execute.go          # Screen 7
    validate.go           # NEW — pre-flight + per-step validation
```

### The Config Package (`pkg/config/`)

This is the critical dependency. The server must be able to read `darkpawns.yaml` at startup.

```go
// config.go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    World    WorldConfig    `yaml:"world"`
    Scripts  ScriptsConfig  `yaml:"scripts"`
    Database DatabaseConfig `yaml:"database"`
    TLS      TLSConfig      `yaml:"tls"`
    Web      WebConfig      `yaml:"web"`
    Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
    Port int    `yaml:"port"`
    Host string `yaml:"host"`
}

type WorldConfig struct {
    Path string `yaml:"path"`
}

type ScriptsConfig struct {
    Path string `yaml:"path"`
}

type DatabaseConfig struct {
    URL     string `yaml:"url"`
    Enabled bool   `yaml:"enabled"`
}

type TLSConfig struct {
    Enabled  bool   `yaml:"enabled"`
    CertFile string `yaml:"cert_file"`
    KeyFile  string `yaml:"key_file"`
}

type WebConfig struct {
    Enabled bool   `yaml:"enabled"`
    Path    string `yaml:"path"`
}

type AuthConfig struct {
    JWTSecret string `yaml:"jwt_secret"`
}
```

**Loading logic:**
1. Check for `darkpawns.yaml` in the current directory
2. If found, unmarshal and return
3. If not found, fall back to CLI flags (backward compatible)
4. CLI flags override config file values (flag > env > config > defaults)

### Server Changes (`cmd/server/main.go`)

Minimal changes to support config file loading:

```
1. Before flag.Parse(), check for darkpawns.yaml
2. If config file exists, load it as defaults
3. CLI flags override config values
4. Env vars (USE_TLS, JWT_SECRET) override both
```

The server does NOT depend on the setup package. Config loading is in `pkg/config/`, which both `cmd/server` and `cmd/setup` import.

### The Setup Package (`pkg/setup/`)

**No dependency on the server's game packages.** The setup wizard imports:
- `pkg/config` — to read/write the config file
- `pkg/parser` — only to validate the world data path (call `ParseWorld` and show stats)

It does NOT import `pkg/game`, `pkg/session`, `pkg/combat`, etc. The wizard validates data; it doesn't run the server.

**Exception:** Screen 7 (Execution) needs to start the server. This is handled by calling the server's `Start()` function or by spawning the server binary as a subprocess. The subprocess approach is simpler and keeps the boundary clean:

```go
// In execute.go — after config is written:
cmd := exec.Command(os.Args[0], "-world", config.World.Path, "-port", strconv.Itoa(config.Server.Port))
cmd.Stdout = logViewport  // Bubbles viewport for live output
cmd.Stderr = logViewport
```

Wait — if we're generating a config file, the server should read it. So the execution step is just:

```go
cmd := exec.Command(os.Args[0]) // darkpawns-server reads darkpawns.yaml automatically
cmd.Stdout = logViewport
cmd.Stderr = logViewport
```

But this only works after the server is updated to read config files. Until then, pass the flags explicitly.

### The Setup Entry Point (`cmd/setup/main.go`)

```go
//go:build !web

package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/zax0rz/darkpawns/pkg/setup"
)

func main() {
    // Non-TUI fallback: if stdin is not a terminal, run plain prompts
    if !isTerminal(os.Stdin) {
        setup.RunPlain(os.Stdin, os.Stdout)
        return
    }

    p := tea.NewProgram(setup.NewWizard(), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Non-TUI Fallback

The plain-interactive mode reuses the same step logic but prints questions to stdout and reads answers from stdin. Same validation, same config output, no Bubble Tea dependency. This is critical for:
- Docker builds (`docker run -it darkpawns/setup`)
- CI/CD pipelines
- SSH sessions with no ANSI support
- Users who just prefer answering prompts

Implementation: each step exposes a `PromptPlain(r io.Reader, w io.Writer) error` method alongside the Bubble Tea `Update/View`.

---

## 6. Dependencies

### New Go Dependencies

| Package | Purpose | Size |
|---------|---------|------|
| `github.com/charmbracelet/bubbletea` | TUI framework | ~200KB |
| `github.com/charmbracelet/lipgloss` | Styling/layout | ~100KB |
| `github.com/charmbracelet/bubbles` | UI components (text input, viewport, spinner) | ~150KB |
| `gopkg.in/yaml.v3` | YAML config read/write | ~100KB |

**Total new dependency weight:** ~550KB. All are well-maintained, widely used Go libraries. Charm's ecosystem is the de facto standard for Go TUIs.

### Existing Dependencies Leveraged

| Package | Used For |
|---------|----------|
| `pkg/parser` | World data validation (Screen 2) |
| `net` (stdlib) | Port bind test (Screen 3) |
| `database/sql` + `github.com/lib/pq` | Database connectivity test (Screen 4) — already in go.mod |
| `os/exec` | Spawning server process (Screen 7) |

### No New Dependencies Required For

- Config file reading (stdlib `os.ReadFile` + yaml.v3)
- Terminal detection (stdlib `golang.org/x/term` — already transitive dep of bubbletea)
- Random secret generation (stdlib `crypto/rand`)

---

## 7. What the Server Needs (Minimal Changes)

The server currently has **zero config file loading.** Here's the minimal change:

### Add to `cmd/server/main.go`:

```go
// Before flag.Parse(), attempt to load config file
cfg, configErr := config.Load("darkpawns.yaml")

// After flag.Parse(), merge: config < flags < env
if configErr == nil {
    // Config file loaded — use as defaults
    if *worldDir == "" && cfg.World.Path != "" {
        worldDir = &cfg.World.Path
    }
    // ... same for other flags
}
```

This is backward compatible. If no config file exists, the server works exactly as it does today. If a config file exists, it fills in the defaults that flags don't override.

### Add `pkg/config/` package

~150 lines of code. Config struct, YAML loading, defaults, validation. No game logic.

---

## 8. Edge Cases & Error Handling

| Scenario | Behavior |
|----------|----------|
| Config file exists, wizard rerun | Rehydrate forms with current values. Offer "edit" or "overwrite." |
| Config file exists, server started without flags | Server reads config. Works. |
| Config file exists, server started WITH flags | Flags override config. Works. |
| No config file, no flags | Server errors: "Usage: server -world <path>" (current behavior) |
| World path is valid but wrong format | Parser returns error, wizard shows it. |
| Database URL is wrong | Connection test fails, wizard shows error, offers "continue without DB." |
| Port is in use | Bind test fails, wizard shows which process is using it (if possible). |
| Terminal too narrow | Drop ASCII art, use single-line title, simplify layout. |
| Wizard interrupted (Ctrl+C) | Write partial config? No — don't write until Screen 6 confirmation. |
| Server crashes during Screen 7 execution | Show the error, offer "View logs" or "Quit." |

---

## 9. Build Integration

### Makefile Additions

```makefile
# Setup wizard
setup:
	go build -o darkpawns-setup ./cmd/setup

# Combined: setup then run
init: setup
	./darkpawns-setup

# Config file
config: build
	./darkpawns -generate-config  # optional: generate a default config without TUI
```

### Docker

The Dockerfile stays the same. The setup wizard is a separate binary — users run it on the host before `docker compose up`, or inside a container with `-it`:

```bash
docker run -it -v $(pwd)/data:/app darkpawns/setup
```

Or the wizard generates `darkpawns.yaml` on the host, which is mounted into the container.

---

## 10. Implementation Order

### Phase 1: Config Foundation (no TUI yet)
1. Create `pkg/config/` — struct, YAML loading, defaults, validation
2. Update `cmd/server/main.go` — load config file before flag parsing
3. Add `darkpawns.yaml` to `.gitignore`
4. Test: server reads config, flags override, backward compatible

### Phase 2: Setup Logic (no TUI yet)
1. Create `pkg/setup/` — step interfaces, validation functions
2. Implement each step as a plain `PromptPlain(r, w)` function
3. Create `cmd/setup/main.go` — non-TUI entry point (plain prompts)
4. Test: `darkpawns-setup` works in a terminal with plain prompts

### Phase 3: TUI Layer
1. Add Bubble Tea + Lip Gloss + Bubbles dependencies
2. Create `pkg/setup/wizard.go` — Bubble Tea model wrapping step logic
3. Implement each step's `Init/Update/View`
4. Wire terminal detection in `cmd/setup/main.go`
5. Test: full TUI flow works

### Phase 4: Execution & Polish
1. Screen 7: spawn server process, pipe output to viewport
2. Graceful shutdown handling (Ctrl+C in Screen 7 → kill server process)
3. ASCII art refinement (the Dark Pawns logo in the box)
4. Edge case testing: narrow terminal, bad paths, port conflicts

---

## 11. Testing Strategy

- **Unit tests:** `pkg/config/` — YAML round-trip, defaults, override precedence
- **Unit tests:** `pkg/setup/validate.go` — port bind, world path, DB connection
- **Integration test:** `cmd/setup/main.go` in plain mode — pipe inputs to stdin, verify config output
- **Manual test:** TUI flow in various terminal sizes
- **Manual test:** Server reads generated config, starts correctly
- **Manual test:** Backward compatibility — server with flags, no config file

---

## 12. What This Enables Later

The wizard is the foundation for everything else:

- **Config file** → enables Docker without flag soup
- **`pkg/config/`** → every future feature reads config from one place
- **`cmd/setup/`** → can be extended with post-install steps (create admin account, import legacy data)
- **Non-TUI fallback** → scriptable, CI/CD friendly
- **TUI framework** → Bubble Tea + Lip Gloss skills transfer to future TUIs (log viewer, player dashboard)

But those are later. Right now: config file, wizard, server reads config. Ship it.

---

## 13. Open Questions

1. **Config file location:** Should the server search multiple paths (`./darkpawns.yaml`, `~/.config/darkpawns/config.yaml`, `/etc/darkpawns/config.yaml`)? Or just `./darkpawns.yaml`? Recommendation: start with CWD only. Add search paths later.

2. **Admin account creation:** Should the wizard create the first admin/wizard account? The server has character creation via the game, but there's no bootstrap mechanism for admin access. This would be useful but might be scope creep for the first version.

3. **`darkpawns.yaml` in the repo:** Should we ship a `darkpawns.example.yaml` in the repo root? Yes — it serves as documentation for manual setup.

4. **The `web` build tag:** `cmd/server/main.go` has `//go:build !web`. There's presumably a `cmd/server/web.go` with a `//go:build web` variant. Does the setup wizard need a similar split? Probably not — the wizard doesn't serve HTTP.

---

*"A config generator with UX." That's the whole thing. Everything else is polish.*
