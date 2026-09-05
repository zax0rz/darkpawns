# Dark Pawns — AI Agent Instructions

## Prime Directive — Fidelity Law

Dark Pawns is a **1:1 faithful port**: the Go server must emit the *same player-facing bytes* as the original C. "The game is the game." Before you change any player-observable behavior, read the port bible:

**[`docs/fidelity/RULEBOOK.md`](docs/fidelity/RULEBOOK.md)** — the C→Go translation law (R1–R5).

**Documentation map** — read these before touching the matching surface:

| Doc | Governs |
|---|---|
| [`docs/fidelity/RULEBOOK.md`](docs/fidelity/RULEBOOK.md) | C→Go port fidelity law (R1–R5) |
| [`DEPLOYMENT.md`](DEPLOYMENT.md) | Server topology, systemd, deploy procedure |
| [`docs/brand-voice.md`](docs/brand-voice.md) | Site prose voice — three-layer framework; public site uses Layer 3 |
| [`website-astro/DESIGN.md`](website-astro/DESIGN.md) | Site design system ("Haunted Paperback"); machine-readable twin in `website-astro/.impeccable/design.json` |
| [`website-astro/PRODUCT.md`](website-astro/PRODUCT.md) | Site audience and product decisions |
| [`website-astro/ARCHIVE-POLICY.md`](website-astro/ARCHIVE-POLICY.md) | What recovered community material may be published |
| [`website-astro/CONTENT-AUDIT.md`](website-astro/CONTENT-AUDIT.md) | Post-migration content review queue |
| [`website-astro/SPEC-AUDIT.md`](website-astro/SPEC-AUDIT.md) | specification.website release checklist status |

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

The Dark Pawns website is a static site built using **Astro** and served via Caddy.

### Core Codebase Location
* Authored website source lives in **`website-astro/`**. Generated map, database, and public discovery assets remain in **`website/static/`** and are shared through Astro's public directory configuration.
* **NEVER** edit or build website source on the production server. `/srv/hugo/` is only the deployed document root; its name is historical.

### Design Aesthetics & Philosophy
* **"Haunted Paperback" style** (see `website-astro/DESIGN.md`): worn cream paper backgrounds, charcoal ink text, one oxblood accent, flat ink, serif typography.
* **Asset Pipeline**: Authored JavaScript must be imported by Astro so Vite fingerprints it. Shared generated assets under `website/static/` keep stable public URLs.

### Automated Deployment Pipeline
To prevent compiling from the wrong directory or syncing outdated files, **always use the Makefile target** in the root of the repository:

```bash
make deploy-site
```

This target automatically executes the complete, secure deployment sequence:
1. Regenerates map and database assets from the authoritative world files.
2. Runs voice, content-inventory, and Astro build checks.
3. Generates the Caddy permanent-redirect table.
4. Syncs `website-astro/dist/` to the configured production document root. `DEPLOY_USER` and `DEPLOY_HOST` are required explicitly.

**Always dry-run first and read the deletions.** The sync uses `--delete`, so anything production serves that your branch does not build is removed:

```bash
rsync -azn --delete --itemize-changes \
  website-astro/dist/ root@192.168.1.121:/srv/hugo/ | grep '^\*deleting'
```

Only stale build artifacts should appear: superseded `_astro/*.css` hashes, Markdown twins for routes that no longer emit them, old `.bak` files. If a real page or image is listed, stop: its source is missing from the branch you are deploying. Commit the source and dry-run again rather than dropping `--delete`. This check caught a published blog post and its illustrations on 2026-09-04, whose only copy was untracked files in another worktree.
