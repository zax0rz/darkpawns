# Port Fidelity Audit Brief (DP-192)

**Objective:** Systematically compare every ported C function against its Go counterpart in `pkg/game/` to find stubs, silent simplifications, behavioral differences, and missing edge cases.

## Context

Dark Pawns was ported from ~73,000 lines of CircleMUD C to ~211 Go files. The C source lives in `src/`. The Go port lives in `pkg/game/` and related packages. Most functions are faithfully ported, but some were simplified, stubbed, or skipped entirely. This audit finds them all.

## What to Search For

### 1. Stub Functions
Functions that print a message but don't do the actual work. Look for:
- `sendToChar(ch, "...")` followed by `return true` with no logic between
- Functions with empty bodies or only a TODO comment
- Functions that print "You start X" but never complete the action

### 2. "Simplified" Comments
Search for `// Simplified` in the Go code. For each one:
- Read the comment explaining what was simplified
- Read the corresponding C function in `src/`
- Determine if the simplification changes behavior (not just removes unused features)
- Note the file, line, and what's different

### 3. Missing Edge Cases
Compare Go function signatures and argument counts against C:
- Functions that take fewer arguments in Go than C (hardcoded defaults?)
- Functions that skip error-checking paths the C version handles
- Functions that don't check conditions the C version checks (e.g., PLR_FLAGGED checks, IS_NPC checks, level checks)

### 4. Unported Functions
C functions that have no Go counterpart at all. Focus on:
- Functions in `src/act.comm.c`, `src/act.item.c`, `src/act.other.c`, `src/act.offensive.c`, `src/act.movement.c` — these are the core gameplay commands
- Functions in `src/spec_procs.c`, `src/spec_procs2.c`, `src/spec_procs3.c` — special procedures
- Functions in `src/boards.c`, `src/mail.c` — note/mail system
- Functions in `src/house.c` — housing system (known to be partially stubbed)

### 5. TODO/FIXME Near Ported Code
Search for `TODO` and `FIXME` within 5 lines of functions that have C source citations.

## How to Compare

For each C function:
1. Find the `// Source: <file>:<line>` citation in the Go code
2. Read the C source at that location
3. Compare the logic flow, not just the surface behavior
4. Note any `// Simplified:` or `// Extended:` comments and evaluate them

If a function has NO source citation, assume drift until proven otherwise.

## Output Format

For each finding, produce:

```
### FINDING: [short title]
- **Go file:** pkg/game/X.go:LINE
- **C source:** src/X.c:LINE
- **Severity:** CRITICAL | HIGH | MEDIUM | LOW
- **Type:** STUB | SIMPLIFICATION | MISSING_EDGE_CASE | UNPORTED | DRIFT
- **Description:** What's different and why it matters
- **Impact:** Who is affected (players, builders, admins) and how
- **Recommendation:** Fix, document, or defer
```

## Priority Order

1. **Core gameplay commands** (`act.comm.c`, `act.item.c`, `act.other.c`, `act.offensive.c`, `act.movement.c`) — these affect every player
2. **Special procedures** (`spec_procs*.c`) — mob behavior, quests
3. **Boards/mail** (`boards.c`, `mail.c`) — social features
4. **Housing** (`house.c`) — player housing (known partial)
5. **Wizard commands** (`act.wizard.c`) — admin tools (lower priority, fewer users)

## Out of Scope

- **OLC/Builder commands** — The C OLC system (medit, oedit, redit, zedit, sedit, tedit, cedit, luaedit) was intentionally not ported. All world editing now lives in the web admin panel at `/admin`. The Go server has zero OLC commands registered. Do not flag missing OLC functions.
- **`damage_stubs.go`** — Intentional dual-path combat design, documented
- **Functions with no C lineage** (new Go infrastructure: admin API, moderation, optimization)
- **Build/CI/tooling code**
- **Test files**


## Reference

- C source: `src/` directory in the repo root
- Go port: `pkg/game/` and related packages
- Linear project: DP team, "Port Fidelity" label
- Prior audit results: see RESEARCH-LOG.md sessions 44-47

## Known Stubs (Starting Points)

These are already identified — verify they're real and look for more:

1. `houses.go:56` — Player ID ↔ Name lookup stubs
2. `houses.go:94` — Obj_from_store / Obj_to_store stubs  
3. `house_save.go:32` — House object load/save stubs
4. `spec_procs4.go:290` — do_look placeholder
5. `other_economy.go:172` — Item type routing "TBD"
6. `skill_stealth.go:63` — Steal simplified
7. `skills2.go:199` — Link minds simplified
8. `skills2.go:400` — DoDig loot table is placeholder strings
