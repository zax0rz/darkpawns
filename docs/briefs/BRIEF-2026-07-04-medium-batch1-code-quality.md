# Brief: Medium Batch 1 — Code Quality (Kimi)

Owner: Kimi (has Linear access)
Scope: Ship focused code-quality fixes for these 5 issues, update Linear with results, and keep behavioral changes faithful to the C source.

## Workflow

1. For each issue below, read the linked source and C source.
2. Make the smallest safe fix that matches the C behavior/intent.
3. Add targeted tests where possible.
4. Run:
   - `go build ./...`
   - `go vet ./...`
   - `go test ./...`
5. Commit per logical fix group or per issue.
6. Update Linear:
   - move issue to In Review or Done
   - add comment with commit hash + what changed
   - add related PR if created

## Issues

### 1) DP-670 — Movement panic on out-of-range sector values
Goal: prevent panic/index-out-of-range when room sector is malformed.

Expected behavior:
- parser validates sector value on load
- movement code defensively handles bad sector value
- if C source has the same inherited bug, note it and fix the Go side anyway

Source hints:
- Go: `pkg/game/act_movement.go`
- Go: `pkg/parser/wld.go`
- C source: `src/act.movement.c:135-136`

Acceptance:
- bad sector value does not panic
- valid sector values still work
- add at least one test for invalid/out-of-range sector handling

---

### 2) DP-699 — Combat hit/miss messages missing `\r\n`
Goal: fix telnet/display corruption from inconsistent line terminators.

Expected behavior:
- all combat hit/miss messages use the same terminator convention as the rest of the combat code
- no behavioral change beyond clean output formatting

Source hints:
- Go: `pkg/combat/engine.go`
- C source: `src/fight.c`

Acceptance:
- hit/miss messages match codebase convention
- add a focused test or assertion if practical
- confirm no accidental combat logic change

---

### 3) DP-700 — Word-filter censoring is case-sensitive after case-insensitive match
Goal: make censoring consistent with matching semantics.

Expected behavior:
- if matching is case-insensitive, censoring should also handle case-insensitive matches correctly
- preserve original message structure as much as possible

Source hints:
- Go: `pkg/command/admin_commands.go`
- C source: `src/act.comm.c`

Acceptance:
- lowercase, mixed-case, and uppercase variants are all censored when configured
- add tests for exact and regex censor behavior
- no unrelated chat command regressions

---

### 4) DP-797 — Buffered decision logs not flushed on shutdown
Goal: preserve buffered decision/combat records on shutdown.

Expected behavior:
- retain writer reference
- call `Stop` / flush during graceful shutdown
- no goroutine leak and no panic on double-close/shutdown race

Source hints:
- Go: `cmd/server/main.go`
- related path: `DecisionLogWriter`

Acceptance:
- shutdown path flushes buffered records
- add focused shutdown test if feasible
- confirm server still starts cleanly

---

### 5) DP-528 — Affect stack removal skips flag reference counting
Goal: fix missing flag refcount decrement in stacked affect removal paths.

Expected behavior:
- `removeOldestStack` and `removeAffectsByStackID` should mirror the flag/refcount behavior of the normal remove path
- stacked affect replacement should not leave stale status flags set

Source hints:
- Go: `pkg/engine/affect_manager.go`
- related path: `RemoveAffect`

Acceptance:
- refcount decreases correctly on stacked removal
- status flag clears when refcount hits zero
- add tests for stacked affect replacement/refcount behavior
- confirm no obvious combat/affect regression

---

## Non-goals

- no broad refactor
- no feature work
- no unrelated cleanup

## Required output

For each issue, provide:
- what changed
- file list
- commit hash
- Linear comment content
- whether it needs follow-up
