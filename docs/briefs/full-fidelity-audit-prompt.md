# Port Fidelity Audit — Full Codebase Prompt

**Purpose:** Autonomous C-to-Go fidelity audit across the entire Dark Pawns codebase.
**Model:** Gemini 3.5 Flash (or equivalent 1M context window)
**Output:** Per-file-pair audit reports in `darkpawns_repo/audit_reports/`

---

## ROLE & OBJECTIVE

You are an autonomous systems engineering agent executing a port fidelity audit on the "Dark Pawns" codebase located at `/darkpawns_repo/`. Your task is to compare the legacy 1994 C source code against the modern Go port to identify logic drift, missing side effects, type boundary bugs, and structural mismatches.

The C source lives in `src/` (~73,000 lines, 80 files). The Go port lives in `pkg/game/`, `pkg/session/`, `pkg/combat/`, `pkg/spells/`, and related packages (~211 files, ~45,000 lines). This is a CircleMUD 3.0 port — not a rewrite. The Go code should match C behavior, not improve on it.

---

## PREREQUISITE: FILE MAPPING

Before auditing, produce a file mapping table. This is your execution plan.

1. List every `.c` file in `src/` (excluding `svn_version.c`, `version.c`).
2. For each C file, identify the corresponding Go file(s) or package(s).
3. Note the mapping type: `1:1` (direct port), `1:N` (split across packages), `N:1` (merged), or `NONE` (unported).
4. Estimate complexity: `LOW` (<200 lines C), `MEDIUM` (200-1000), `HIGH` (>1000).
5. Write the table to `audit_reports/file_mapping.md`.

**Mapping strategies:**
- `comm.c` → `pkg/session/` (session handling, command dispatch)
- `act.*.c` → `pkg/game/` (game commands — act.comm.go, act.item.go, act.other.go, etc.)
- `fight.c` → `pkg/combat/` (combat engine)
- `magic.c`, `spells.c` → `pkg/spells/` (spell system)
- `db.c` → `pkg/world/` (world loading, zone parsing)
- `spec_procs*.c` → `pkg/game/spec_procs*.go` (special procedures)
- `interpreter.c` → `pkg/session/commands.go` (command registry + dispatch)
- `handler.c` → `pkg/game/handler*.go` (object/character manipulation)
- Files with no Go equivalent → marked `UNPORTED`

**Review the mapping table before proceeding.** If a C file maps to multiple Go packages, list all of them. If a Go file has no C lineage (new infrastructure), mark it `NEW` and skip it during audit.

---

## EXECUTION STRATEGY (ANTI-OVERFLOW PROTOCOL)

To prevent context saturation and hallucination, execute this audit sequentially. Do NOT read the entire repository into context at once.

### Per-File-Pair Loop

```
FOR EACH row in file_mapping.md where mapping != UNPORTED and != NEW:

  1. READ the C source file (full file).
  2. READ the corresponding Go file(s) (full file).
  3. AUDIT using the evaluation criteria below.
  4. WRITE findings to audit_reports/[module_name]_audit.md.
  5. BEFORE moving to next pair: acknowledge you are done with this pair
     and will not reference it again. Do not carry state between pairs.
```

### Context Management Rules

- Process ONE file pair at a time. Do not hold multiple file pairs in working memory.
- After writing the audit report for a pair, do not re-read or re-reference that pair.
- If a finding in the current pair references a function in a DIFFERENT file, note the cross-reference but do not follow it. The other file will be audited when its turn comes.
- If a C file has no corresponding Go file (UNPORTED), write a brief note to the audit report listing what was not ported and move on.
- **Macro Glossary Exception:** You are permitted (and encouraged) to keep `src/structs.h`, `src/utils.h`, and `src/spells.h` open in context at all times to decode C macro definitions encountered during individual file audits.

---

## EVALUATION CRITERIA

For each file pair, hunt down these classic C-to-Go port traps:

### 1. Stub Functions & Partial Ports
Functions that print a message but don't do the actual work, or are only half-finished.
- `sendToChar(ch, "...")` followed by `return true` with no logic between
- Functions with empty bodies or only a TODO comment
- Functions that print "You start X" but never complete the action
- Functions that exist in C but have no Go counterpart (unported)
- **Partial Switch/Conditional Porting:** Switch statements or `if/else` ladders where the C version handles a wide variety of sub-types, positions, or item slots, but the Go port only implements a subset and silently ignores or stubs the remaining branches.

### 2. Silent Simplifications
Search for `// Simplified` in Go code. For each:
- Read the comment explaining what was simplified
- Read the corresponding C function
- Determine if the simplification changes BEHAVIOR (not just removes unused features)
- A simplification that removes a check, skips an edge case, or hardcodes a value is a finding
- A simplification that removes dead code or unused optional features is not a finding

### 3. Missing Side Effects (Hidden State Mutations)
C MUD code modifies global variables and passes pointers that mutate structs deep in call chains.
- Functions that take pointer arguments in C but value arguments in Go
- Global state modifications (combat_list, char_list, zone_table) not present in Go
- Functions that modify object state (action_description, affected[], etc.) differently
- Room/zone state changes that don't propagate

### 4. Type & Boundary Vulnerabilities
C fluidly casts signed/unsigned and truncates. Go's strict typing catches compile-time issues but creates logical bugs.
- `int32` used where C uses `unsigned char` (overflow behavior differs)
- `rand.Intn()` vs C `number()` — check upper/lower bounds match
- `dice()` implementation — verify dice formula matches C exactly
- String truncation: C `snprintf` vs Go `fmt.Sprintf` (buffer overflow semantics differ)
- Nil pointer panics where C would read garbage memory silently

### 5. Control Flow Drift
Verify logic paths match exactly.
- Macro evaluations — C macros that expand to conditions (e.g., `AFF_FLAGGED`, `PLR_FLAGGED`) must have equivalent checks in Go
- Early returns — if C has `if (!x) return;` and Go doesn't, that's a drift
- Loop boundaries — `for` vs `while` translation, off-by-one in range
- Switch/if-else differences — C `switch` with fallthrough vs Go `switch` without
- Random number generation bounds — `number_range(a, b)` vs `rand.Intn(b-a+1)+a`

### 6. Missing Citations
Every function ported from C should have a `// Source: file:line` comment. Functions that look ported but have no source citation are drift candidates — verify they actually match C behavior.

### 7. Concurrency & Mutex Safety
C MUDs are strictly single-threaded, running on a synchronous main loop. The Go engine is concurrent (using separate goroutines for network sessions, combat tickers, etc.).
- Identify where Go code replicates C's global/struct state mutations but fails to guard them with appropriate locks (`sync.Mutex`, `sync.RWMutex`) or channel communication in Go's concurrent model.
- Check if concurrent writes/reads on character attributes, items, or room occupants from connection/ticker loops introduce data race vulnerabilities.

### 8. Unit Test Coverage & Verification
Before concluding that a difference is logic drift, check if the corresponding Go package has a `*_test.go` file (e.g. `location_test.go`).
- Determine if the discrepancy is covered or asserted by existing unit tests.
- Note whether the unit test explicitly guarantees the "incorrect" behavior (which might mean it is an intentional design choice) or if it's missing coverage entirely.

---

## OUTPUT FORMAT

Write one file per audited pair to `audit_reports/[module_name]_audit.md`:

```markdown
# Audit Report: [C_File] vs [Go_File(s)]

**C file:** src/X.c (N lines)
**Go file(s):** pkg/game/X.go (N lines)
**Mapping type:** 1:1 | 1:N | N:1
**Functions audited:** N

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: [Short Title]
- **Location:** [Function/Method, file:line]
- **C behavior:** [Exact state change or condition]
- **Go behavior:** [What Go is doing]
- **Discrepancy:** [Why this alters game state or behavior]
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW
- **Type:** STUB | SIMPLIFICATION | MISSING_SIDE_EFFECT | DRIFT | CONCURRENCY

---

## Type & Boundary Vulnerabilities

### [FINDING-002]: [Short Title]
- **Location:** [Code snippet line/block]
- **Risk:** [Integer overflow, slice bounds, macro translation, RNG bounds]
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW

---

## Control Flow & Mathematical Fidelity

### [FINDING-003]: [Short Title]
- **Issue:** [RNG boundaries, loop conditions, early returns, macro checks]
- **Impact:** [How it alters gameplay stability or determinism]
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW

---

## Concurrency & Mutex Safety

### [FINDING-004]: [Short Title]
- **Location:** [Code snippet line/block]
- **Issue:** [Data race, missing lock, unprotected global state]
- **Impact:** [How it affects thread safety or server stability]
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW

---

## Unported Functions

List any C functions in this file with no Go counterpart. Include function name, C line number, and brief description of what it does.

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| func_name  | 123  | Does X     | NO      |

---

## Summary

- Total findings: N
- Critical: N | High: N | Medium: N | Low: N
- Unported functions: N
```

If a section has zero findings, write "None detected." Do not include conversational commentary in the saved reports.

---

## SEVERITY DEFINITIONS

- **CRITICAL:** Breaks a core gameplay system. Multiple players affected. No workaround. (e.g., combat system broken, all consumables non-functional, server crash)
- **HIGH:** Breaks a specific feature or class. Workaround exists but is painful. (e.g., steal doesn't work on mobs, invisibility doesn't work, a specific spell is broken)
- **MEDIUM:** Behavioral difference that most players won't notice. Cosmetic or edge-case impact. (e.g., hunger/thirst not tracked, room description not shown after teleport)
- **LOW:** Minor deviation from C. No gameplay impact. (e.g., different error message text, unused variable, cosmetic formatting)

---

## WHAT NOT TO FLAG

- **New infrastructure** with no C lineage (admin API, moderation, optimization packages, Docker/deploy code)
- **Test files** (`*_test.go`)
- **Build/tooling** (Makefile, scripts, CI config)
- **Intentional design decisions** documented in comments (e.g., `damage_stubs.go` dual-path combat)
- **`pkg/game/spec_procs_missing.go`** — newly implemented spec procs (session 67)

---

## KNOWN ISSUES (DO NOT RE-REPORT)

These have already been found and fixed. Skip them:

- doUse stub (DP-337) — now routes by item type
- canSee simplified (DP-338) — now has full visibility matrix
- Spec proc pipeline (DP-342) — now wired into command dispatch
- DoSteal mob targets (DP-332) — now supports MobInstance
- DoMindlink mana transfer (DP-334) — now works via MobInstance.GetMana
- DoDig loot table (DP-335) — now spawns real objects
- "use" command routing (DP-341) — now checks inventory first
- write command stub (DP-340) — now wired to correct implementation
- doDrink/doEat hunger/thirst (DP-336) — now calls GainCondition
- Portal room description (DP-339) — now calls doLook after teleport
- House player lookups (DP-333) — now wired
- Postmaster spec proc (DP-343) — now registered
- 5 legacy spec procs (DP-344) — now registered

---

## RECOMMENDED AUDIT ORDER

Start with high-complexity, high-impact files:

1. `fight.c` → `pkg/combat/` — combat formulas, damage calculation, death handling
2. `magic.c` / `spells.c` → `pkg/spells/` — spell effects, saving throws, affect application
3. `handler.c` → `pkg/game/handler*.go` — object/character manipulation, affects, inventory
4. `db.c` → `pkg/world/` — world loading, zone parsing, object/mob instantiation
5. `interpreter.c` → `pkg/session/` — command parsing, alias system, input handling
6. `act.wizard.c` → `pkg/session/wizard_cmds.go` — admin commands
7. `spec_procs*.c` → `pkg/game/spec_procs*.go` — special procedures (already partially audited)
8. Remaining `act.*.c` files — player commands
9. Everything else

---

## FINAL NOTE

This audit is adversarial by design. You are looking for bugs, not confirming correctness. Every finding should include enough detail for a developer to fix it without re-reading the C source. If you're unsure whether something is a bug, flag it as LOW and explain the uncertainty — a human triager will decide.
