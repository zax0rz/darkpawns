# Dark Pawns — AI Agent Instructions

## Prime Directive — Fidelity Law

Dark Pawns is a **1:1 faithful port**: the Go server must emit the *same player-facing bytes* as the original C. "The game is the game." Before you change any player-observable behavior, read the port bible:

**[`docs/fidelity/RULEBOOK.md`](docs/fidelity/RULEBOOK.md)** — the C→Go translation law (R1–R5).

- **R1** player-facing bytes are law · **R2** the command surface is part of the game · **R3** determinism & draw parity · **R4** no invention · **R5** process rules (find-one-find-the-class; verify the call path).
- **Cite rules by number** in commits, PRs, reviews, and Linear — "violates R4" is a complete verdict.
- `src/` and `darkpawns-c-oracle/` are the **read-only oracle** (ground truth). Never edit them; diff against them with `cmd/dp-oracle-diff`.
- When a byte is in question, **the C source wins** (R5e — verify the actual call path, don't trust a summary). A repeated failure indicts the rule, not the file: amend the rulebook + audit the whole class (R5b/R5c).

### Fidelity Work: Start Here

Before extending oracle coverage or declaring a command complete, read
**[`docs/fidelity/DEPTH_TESTING.md`](docs/fidelity/DEPTH_TESTING.md)**. It explains the current
breadth-to-depth strategy, proof levels, scenario fixtures, manifests, and the dated handoff frontier.
Breadth coverage proves that a command can match once; it does **not** prove the port is complete.

### Research Continuity

Dark Pawns is also an open, ongoing research artifact. Before making paper-level
claims or running a research-writing task, read **[`docs/research/README.md`](docs/research/README.md)**
and update **[`docs/research/EVIDENCE_LEDGER.tsv`](docs/research/EVIDENCE_LEDGER.tsv)**. Treat cron and
agent prose as field notes until its citations survive contact with the repository, oracle output,
Git history, or another named primary artifact.

## Build & Verify

```bash
go build ./...          # Full build — must pass before committing
go vet ./...            # Static analysis — must pass before committing
go test ./...           # All tests — must pass before committing
go test ./pkg/game/...  # Game package tests specifically
golangci-lint run ./... # Full lint (uses .golangci.yml)
```

**NEVER commit without running all four.** Subagents that self-report passing builds have lied before.

## Project Overview

Dark Pawns is a Go MUD server, ported from C (DikuMUD/Merc 2.2 lineage). ~114K lines of Go, ~66 C files remaining for reference only. The Go port is COMPLETE — do not re-port C files.

### Architecture

- `cmd/server/` — MUD server entrypoint
- `pkg/game/` — Core game logic (combat, skills, objects, rooms, mobs, world state)
- `pkg/session/` — Player session handling, commands, wizard commands
- `pkg/parser/` — Zone/obj/mob/world file parsing (Diku format)
- `pkg/combat/` — Combat formulas and damage calculation
- `pkg/spells/` — Spell system (saving throws, damage, affect spells)
- `pkg/telnet/` — Telnet protocol handling
- `pkg/db/` — PostgreSQL persistence, narrative memory for AI agents
- `pkg/agent/` — AI agent hooks (BRENDA agent integration)
- `pkg/session/memory_hooks.go` — Go→Python memory system bridge

### Key Conventions

- **ObjectLocation system**: Objects track location via tagged union `ObjectLocation` in `location.go`. Use `LocInventoryPlayer()`, `LocEquipped()`, `LocNowhere()` constructors. Never manipulate `Location` fields directly.
- **Runtime state**: `ObjectRuntimeState` and `MobRuntimeState` replace `CustomData map[string]interface{}`. Use typed fields. `CustomData` is the escape hatch for truly dynamic keys.
- **Equipment**: `Equipment` struct with typed `EquipmentSlot` constants. Use `Equip()`/`Unequip()`/`UnequipItem()` — never manipulate `Slots` map directly.
- **Error handling**: Player-facing operations (item transfer, equipment, shop) MUST check error returns. Use `slog.Error()` on rollback failures. Never use `#nosec G104` to suppress error checks in player-facing code.
- **`fmt.Fprintf` to `io.Writer`**: These are user-facing output (MUD text). Do NOT convert to `slog` — they stay as formatted writes.
- **Formatting is gofumpt, not gofmt**: CI runs `gofumpt -l .` and fails on any diff. gofumpt is a strict superset of gofmt, so it's the *only* formatter to run — `gofmt` alone will pass locally but still fail CI. Run `make fmt` before committing, and `make hooks` once per clone to install the pre-push hook that enforces it.

### What NOT to Do

- Do not re-port C files in `src/`
- Do not convert `fmt.Fprintf(player.Writer, ...)` to slog
- Do not remove `CustomData` field (it's the escape hatch)
- Do not change save file format
- Do not use `#nosec G104` on new code — handle the error

### #nosec Annotations

The codebase has ~600 `#nosec` annotations. Most are:
- **G404 (weak random)** — ~394 intentional `math/rand` for dice rolls, combat, MUD mechanics. Leave these alone.
- **G104 (errcheck)** — ~135 remaining. These are the cleanup targets. Handle the error or log it.
- **G115 (integer overflow)** — ~26. Add explicit casts or range checks.
- **G304/G306 (file path/perms)** — ~24. Use `filepath.Clean()` and explicit permissions.
- Other G-codes — small counts, mostly intentional.

## Commit Messages

Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

## Testing

Tests live alongside source files as `*_test.go`. Run `go test ./...` after any change.
The game package has object movement tests that validate the ObjectLocation system — keep those green.

## Website Development & Deployment

The Dark Pawns website is a static site built using **Hugo** and served via Caddy. 

### Core Codebase Location
* The website source files live exclusively inside the **`website/`** subdirectory of the main `darkpawns` repository.
* **NEVER** edit files or run builds on the server inside `/opt/darkpawns/darkpawns-site/` or `/opt/darkpawns/darkpawns-site.deprecated/` — these are deprecated, outdated standalone clones and will completely wipe out new features (like `/map` and xterm client integrations) if compiled.

### Design Aesthetics & Philosophy
* **Stephen King Paperback Style**: Clean ivory/cream backgrounds, charcoal ink text, and dark oxblood highlights.
* **Asset Pipeline**: Static JavaScript (like `/js/client.js`) MUST be managed through the Hugo assets pipeline (`resources.Get` + `fingerprint` in Hugo templates) for cache-busting and SRI integrity checks. **Do not** link raw `/js/client.js` in templates.

### Automated Deployment Pipeline
To prevent compiling from the wrong directory or syncing outdated files, **always use the Makefile target** in the root of the repository:

```bash
make deploy-site
```

This target automatically executes the complete, secure deployment sequence:
1. **`python3 website/scripts/parse_world.py`** — Parses the authoritative MUD room files (`lib/world/`) and compiles a fresh `world.json` for the interactive D3 map page.
2. **`cd website && hugo --minify`** — Runs the Hugo compilation in the correct subdirectory context.
3. **`rsync -avz --delete website/public/ root@192.168.1.15:/opt/darkpawns/hugo-site/`** — Syncs the newly compiled static assets directly to the live server web root.
