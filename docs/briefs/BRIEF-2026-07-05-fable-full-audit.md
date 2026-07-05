# Code Review Brief — Dark Pawns Top-to-Bottom Audit

**Model:** Claude Fable 5 (above-Opus tier)
**Scope:** Full codebase review with roadmap output
**Date:** 2026-07-05
**Codebase stats:** ~104K lines non-test Go across 27 `pkg/` packages + `cmd/server`. `pkg/game/` alone is ~41.5K non-test lines in 130 non-test files (172 counting tests) — and it is one flat Go package, not a tree of sub-packages.

---

## Instructions

You are performing a top-to-bottom code review of a Go MUD server ported from C (DikuMUD/Merc 2.2 lineage). The Go port is complete — C files in `src/` are reference only. Your output will be a `docs/reports/REVIEW-2026-07-05-full-audit.md` document that becomes the project roadmap.

**Working directory:** `/Users/zach/.openclaw/workspace/darkpawns_repo`

### Project Structure

```
cmd/server/          — MUD server entrypoint (~400 lines)
pkg/game/            — Core game logic (~41.5K non-test lines, 130 non-test files, ONE flat package;
                       only subdirs are game/systems and game/data)
pkg/session/         — Player session handling, commands, wizard commands (16,344 lines)
pkg/combat/          — Combat formulas and damage calculation (3,596 lines)
pkg/spells/          — Spell system — saving throws, damage, affect spells (4,557 lines)
pkg/scripting/       — Lua scripting engine for NPC/room/object scripts (3,415 lines)
pkg/command/         — Command router/dispatcher (3,513 lines)
pkg/engine/          — Core engine abstractions (1,683 lines)
pkg/telnet/          — Telnet protocol handling (1,145 lines)
pkg/db/              — SQLite persistence, narrative memory (1,479 lines)
pkg/agent/           — AI agent hooks (414 lines)
pkg/parser/          — Zone/obj/mob/world file parsing (Diku format)
pkg/common/          — Shared interfaces
pkg/admin/           — HTTP admin API
pkg/audit/           — Audit logging
pkg/auth/            — JWT authentication
pkg/metrics/         — Prometheus metrics
pkg/moderation/      — Player moderation
pkg/dreaming/        — Dreaming layer (background narrative)
pkg/events/          — Event system
pkg/grapevine/       — Grapevine inter-MUD chat client
pkg/agentcli/        — Agent CLI
pkg/optimization/    — Optimization helpers
pkg/privacy/         — Privacy controls
pkg/secrets/         — Secrets management
pkg/storage/         — Storage abstractions
pkg/validation/      — Input validation
pkg/testutil/        — Test utilities
src/                  — C reference files (DO NOT modify, DO NOT re-port)
```

(Enumerate with `ls pkg/` — the review must cover ALL packages, including the small ones not called out above.)

### Conventions (from AGENTS.md)

- `ObjectLocation` tagged union — use constructors, never raw `Location` fields
- `ObjectRuntimeState`/`MobRuntimeState` replace `CustomData` where possible
- Equipment uses `Equip()`/`Unequip()` — never manipulate `Slots` map directly
- Player-facing ops MUST check error returns — no `#nosec G104` on new code
- `fmt.Fprintf` to `io.Writer` is MUD output, not logs — do NOT convert to slog
- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

### Test Coverage (current)

| Package | Coverage |
|---------|----------|
| **Total** | 29.7% |
| `pkg/combat` | 59.7% |
| `pkg/game` | 21.4% |
| `pkg/spells` | 36.1% |
| `pkg/session` | 24.1% |
| `pkg/command` | 8.8% |
| `pkg/scripting` | 24.1% |

Golden tests exist for: THAC0, str_app, dex_app, find_exp, regen, combat transcript, spell metadata (108 spells), spell levels (267 tuples), spell damage (24 formulas), spell healing (12 formulas), spell affects (30), backstab mult, AC reduction (23 thresholds), damage messages, position multipliers, attack rounds, con/int/wis_app, practice params, 10 offensive skill formulas.

---

## Phase 0 — Evidence Run (before reading any code)

This project has previously had green builds and passing tests while the game itself didn't work. Ground the review in runtime evidence first:

1. `go build ./...` — must be clean
2. `go vet ./...` — note anything it flags
3. `go test -race ./...` — full suite under the race detector; any race report is an automatic CRITICAL finding with the stack pasted into the report
4. If time permits, run the e2e smoke harness (`scripts/smoke_test_2b.py`) against a locally started server and note whether login → look → move → kill → save works end to end

Record results in a "## Phase 0 — Evidence Run" section. If the race detector or smoke test fails, that failure jumps the Phase 1 priority queue.

---

## Phase 1 — Sweep (30,000 ft)

**Goal:** Survey every package's public API surface. Classify the overall health of the codebase.

### 1A. Architecture Assessment

For every package under `pkg/` (27 as of this writing — enumerate, don't trust this number) plus `cmd/server`, read the primary exported types and functions. Classify each package as:

- **STABLE** — Well-structured, correct interface boundaries, idiomatic Go
- **FRAGILE** — Works but has coupling issues, missing error handling, or unclear contracts
- **DEAD/LAGGING** — Mostly stubs, unused code, or features not yet ported from C

Pay special attention to:

- **`pkg/game/` is ~41.5K non-test lines across 130 files in ONE flat package.** This is by far the largest package and it has no internal package boundaries — files like `act_comm.go`, `combat_helpers.go`, `skill_combat.go`, `damage_stubs.go` are C-file splits sharing one namespace, so everything can touch everything. Is it a god package? Which file clusters have coherent enough boundaries that they could be extracted into real sub-packages or top-level packages, and which are inherently entangled with world state?

- **Package dependency graph:** `pkg/game/` imports `pkg/combat`, `pkg/engine`, `pkg/parser`; `pkg/session/` imports `pkg/game`. Map the actual import graph (`go list -deps` or grep imports). Are there any import cycles lurking (or interfaces that exist only to break one)? Is the dependency direction clean?

- **Global state:** Count and catalog all package-level `var` declarations. The codebase uses singletons (`gameWorld`, `globalLogger`, `shopManager`). How many exist? Are they necessary? Do they make testing harder?

- **`CustomData map[string]interface{}`** — Used in 74 places. This is the "escape hatch" for dynamic state. Where is it used? Should any of these be typed fields instead?

### 1B. C Port Completeness

The C source has 41 `SPECIAL()` functions in `src/spec_procs.c`. The Go port has them split across `spec_procs.go`, `spec_procs2.go`, `spec_procs3.go`, `spec_procs4.go`, and `spec_procs_missing.go` (which contains Go-original specs: no_get, recharger, beholder, zen_master, moon_gate).

Check: Are all 41 C spec procs ported? List any that are missing or stubbed.

Also check:
- Are there commands registered in Go that have stub/empty handlers?
- Are there skills or spells defined in tables but with no `Do*` implementation?
- The `spec_procs.go:944` comment says "Wave 4a: remaining functions from spec_procs.c will be added in Wave 5" — what's still outstanding?

### 1C. Concurrency Model

The codebase has 77 mutex usages across the codebase. The C MUD was single-threaded; Go is concurrent. This is the #1 architectural risk.

Survey:
- Goroutine inventory: What long-running goroutines exist? (zone dispatcher, dreaming layer, grapevine client, AI agent hooks, etc.)
- Shutdown: Is there graceful shutdown? `SIGINT` handler? Do goroutines drain cleanly?
- Lock ordering: Is there a defined lock hierarchy? (e.g., `World.mu` → `Player.mu` → `Mob.mu`). Any risk of deadlock?
- `sync.Mutex` vs `sync.RWMutex` choices — are read-heavy paths using RLock?

### 1D. Security Surface

Previous rounds fixed: JWT issuer enforcement, admin router self-auth, DB credentials, CORS dev guard, moderation audit trail.

Remaining attack surface:
- **Lua sandbox:** `pkg/scripting/engine.go` nils out `dofile`, `loadfile`, `load`, `loadstring`, and `os.execute`. Are there other escape vectors? Can scripts access the filesystem through any remaining global?
- **Input sanitization:** Player names, descriptions, custom messages, clan names — any injection or buffer overflow vectors?
- **Privilege escalation:** Wizard commands, immortal powers — can a mortal player escalate? Are permission checks on every command?
- **Float64 arithmetic:** Shop profit multipliers, item costs — any overflow or precision issues exploitable for infinite gold?
- **Telnet protocol:** Buffer handling, length checks, any missing bounds?

### 1E. Prior Findings Regression Check

A lot of work has shipped since the last full review. Do NOT rediscover known issues from scratch — first verify whether the previous rounds' findings are actually closed:

- `REVIEWS/01-c-to-go-fidelity.md` through `REVIEWS/07-*`, plus `REVIEWS/pass2-concurrency.md`, `pass3-security.md`, `pass4-fidelity.md`, and `REVIEWS/BACKLOG.md`
- The Sprint 2 briefs in `docs/briefs/BRIEF-2026-07-03-*` (these were the fix work driven by the last Fable review: flag-pipeline corruption, one-sided combat, skill damage path skipping death handling, session wedge/ghost cleanup, dead-code layers)

For each previously-reported CRITICAL/HIGH finding: verify the fix landed in the current code (cite file:line), mark it **CLOSED / REGRESSED / NEVER FIXED**. Anything REGRESSED or NEVER FIXED carries its original severity into Phase 3.

### Phase 1 Output

A section titled "## Phase 1 — Sweep" containing:
1. Package health classification table (all packages × status)
2. Architecture risk list (top 10)
3. C port completeness checklist (spec procs, skills, commands)
4. Concurrency model assessment
5. Security surface map
6. Prior-findings regression table (CLOSED / REGRESSED / NEVER FIXED)
7. **Top 5 areas requiring Phase 2 deep dive**, ranked by: player-facing impact, crash risk, security exposure

---

## Phase 2 — Deep Dive (1,000 ft) — DEPTH IS ASPIRATIONAL, COMPLETION IS NOT

**Goal:** Deep-read the riskiest areas and produce specific, actionable findings with `file:line` references.

**Single-session scoping:** the audit MUST finish in one session with Phases 0, 1, and 3 complete. Phase 2 depth is the adjustable variable. Work areas in priority order (union of the Phase 1 top-5 and the mandatory dives B1–B5 below — they usually overlap). For each area, record the depth actually achieved: **FULL** (every line read) / **PARTIAL** (key paths read) / **SKIMMED** / **NOT REACHED**. Anything below FULL gets a corresponding "future deep-dive" entry in the Phase 3 roadmap. Never sacrifice Phase 3 to buy more Phase 2 depth — a shallow finding list with a roadmap beats a deep one without.

### 2A. For each risk area (in priority order, as time allows):

1. Read the files — fully if the area is small, key code paths first if not
2. Document each finding as:
   ```
   **[SEVERITY: CRITICAL/HIGH/MEDIUM/LOW] [CATEGORY] Title**
   - File: `path/to/file.go:line`
   - Description: What's wrong and why it matters
   - C reference: If applicable, cite the C source that shows the expected behavior
   - Suggested fix: Concrete approach (not just "fix this")
   - Effort: S/M/L
   ```
3. Categories: `fidelity`, `concurrency`, `security`, `idiom`, `performance`, `architecture`, `error-handling`, `dead-code`

### 2B. Priority Deep Dives (regardless of Phase 1 ranking)

These areas should be reviewed in Phase 2 even if they don't make the top 5 — at whatever depth time allows, with depth recorded honestly:

#### B1. Combat Data Race Potential
- Read `pkg/combat/fight_core.go` fully
- Read `pkg/game/skill_combat.go` fully
- Read `pkg/game/player_combat.go` fully
- Check: When two players attack the same mob simultaneously, is damage application safe? Are HP reads/writes protected? What about loot distribution on death?

#### B2. Lua Script Sandbox Integrity
- Read `pkg/scripting/engine.go` fully
- Check: Complete list of nil'd globals. Any remaining unsafe globals? Can scripts call Go functions that have side effects? Is there a CPU/memory limit on script execution? Script timeout mechanism?

#### B3. Save/Load Roundtrip Fidelity
- Read `pkg/game/save.go` and `pkg/game/serialize.go` fully
- Check: Can a character be saved and loaded without data loss? Are all fields persisted? Any field drift between save and load? What happens on version mismatch?

#### B4. Error Handling Quality
- The codebase historically had ~135 `#nosec G104` errcheck suppressions (now reportedly cleaned up)
- Search for any remaining unchecked error returns in player-facing code paths:
  - Item transfer (give, drop, put, get)
  - Equipment operations (wear, remove)
  - Shop transactions (buy, sell, value, list)
  - Combat actions (kill, flee, rescue)
- Flag any `err` that is assigned but never checked

#### B5. Spec Proc Fidelity
- Read `pkg/game/spec_procs.go`, `spec_procs2.go`, `spec_procs3.go`, `spec_procs4.go` fully
- Cross-reference against `src/spec_procs.c` for the 41 SPECIAL() functions
- For each spec proc: Does the Go version match C behavior? Are any edge cases missing?

### Phase 2 Output

A section titled "## Phase 2 — Deep Dive" containing:
1. A depth-achieved table (area × FULL/PARTIAL/SKIMMED/NOT REACHED)
2. All findings gathered (specific, actionable, with file:line)
3. Each finding tagged with severity, category, effort

---

## Phase 3 — Roadmap

**Goal:** Transform findings into an ordered action plan.

### 3A. Prioritization Matrix

Create a table of ALL findings from Phase 2, sorted by priority. Columns:

| Priority | Finding | Category | Severity | Effort | Package | Description |
|----------|---------|----------|----------|--------|---------|-------------|

Priority ordering criteria:
1. **Player-facing crash risk** (data races, nil panics, deadlock)
2. **Security exposure** (privilege escalation, injection, sandbox escape)
3. **Gameplay fidelity** (wrong formulas, missing features, save/load corruption)
4. **Architecture debt** (god packages, coupling, dead code that confuses new contributors)
5. **Performance** (hot-path allocations, O(n) lookups, unnecessary locking)
6. **Idiom/quality** (non-idiomatic Go patterns that increase maintenance burden)

### 3B. Recommended Work Streams

Group findings into 3-5 work streams that can be tackled independently:

1. **Stream name** — Theme, estimated effort, affected packages, list of findings
2. Dependencies between streams
3. Quick wins that can be done in a single session

### 3C. Coverage Roadmap

Based on the architecture review, recommend the next round of test investments:
- Which packages need functional/integration tests (not just golden)?
- Which concurrency patterns need race-detector-tested goroutine tests?
- Which security surfaces need adversarial/fuzz tests?
- Target: realistic coverage goals for next 30 days

### Phase 3 Output

A section titled "## Phase 3 — Roadmap" containing:
1. Prioritized findings table
2. Work stream definitions
3. Coverage roadmap
4. **One-paragraph executive summary** of the codebase health

---

## Output Format

Write the complete review to: `docs/reports/REVIEW-2026-07-05-full-audit.md`

Structure:
```markdown
# Dark Pawns — Full Code Review
**Date:** 2026-07-05
**Reviewer:** Claude Fable 5
**Scope:** Full codebase (~104K lines non-test Go)

## Phase 0 — Evidence Run

## Phase 1 — Sweep
### 1A. Architecture Assessment
### 1B. C Port Completeness
### 1C. Concurrency Model
### 1D. Security Surface
### 1E. Prior Findings Regression Check
### Phase 1 Prioritization

## Phase 2 — Deep Dive
### 2A. Top Risk Areas
### 2B. Mandatory Deep Dives
  - B1. Combat Data Race Potential
  - B2. Lua Script Sandbox Integrity
  - B3. Save/Load Roundtrip Fidelity
  - B4. Error Handling Quality
  - B5. Spec Proc Fidelity

## Phase 3 — Roadmap
### 3A. Prioritization Matrix
### 3B. Recommended Work Streams
### 3C. Coverage Roadmap
### Executive Summary
```

---

## Key Reference Files

| Area | Files | Lines |
|------|-------|-------|
| Game core | `pkg/game/*.go` (172 files) | 43,350 |
| Session/commands | `pkg/session/*.go` | 16,344 |
| Combat | `pkg/combat/*.go` | 3,596 |
| Spells | `pkg/spells/*.go` | 4,557 |
| Scripting/Lua | `pkg/scripting/*.go` | 3,415 |
| Command router | `pkg/command/*.go` | 3,513 |
| C spec procs | `src/spec_procs.c` | 41 functions |
| C combat | `src/fight.c` | ~2,000 |
| C magic | `src/magic.c` | ~1,800 |
| C constants | `src/constants.c` | ~1,200 |
| C class | `src/class.c` | ~1,100 |

---

## Constraints

- DO NOT modify any source files. The ONLY file you write is the report (`docs/reports/REVIEW-2026-07-05-full-audit.md`). Running builds and tests (Phase 0) is allowed and expected.
- **Checkpoint as you go:** write the report incrementally — flush each completed phase/section to the report file before starting the next. The audit is too large for one sitting; a checkpointed report survives context compaction, an unwritten one doesn't.
- DO NOT re-port C files. They are reference only.
- DO NOT suggest replacing `fmt.Fprintf` with `slog` — those are MUD output.
- DO NOT suggest removing `CustomData` — it's the escape hatch for dynamic state.
- When judging fidelity, **read the C function in full from `src/*.c`** — do not trust ticket text, prior-review paraphrases, or Go-side comments about what the C does. (Past examples of paraphrase-induced errors: WAIT_STATE *overwrites* rather than adds; spells declared "fabricated" that were actually reachable via the CallMagic dispatch.)
- DO cite specific `file:line` references for every finding.
- DO distinguish between "C behavior we should match" and "C behavior we should improve upon."
- DO flag anything that could cause a production crash or data loss as CRITICAL regardless of category.
