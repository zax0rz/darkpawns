# Dead Code Cleanup — Subagent 1

You are cleaning up dead code in the Dark Pawns Go codebase.
Repository: `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**RULES:**
- Run `go build ./... && go vet ./... && go test ./pkg/engine/ ./pkg/game/` after every change
- If a build breaks, revert immediately
- Commit each fix separately with message `fix: DP-X — description`
- Never delete code that has callers. Always verify with `grep -rn "FunctionName" --include="*.go" pkg/`

## Tasks

### DP-3: Delete comm_infra.go
`pkg/engine/comm_infra.go` — 408 lines. Header says DEPRECATED, zero callers.
- Verify zero callers first: `grep -rn "nonblock\|set_sendbuf\|get_from_q\|timediff\|timeadd\|perform_subst\|perform_alias\|make_prompt\|setup_log\|open_logfile" --include="*.go" pkg/ | grep -v "_test\|// \|^Binary"`
- If confirmed zero callers, delete the file
- Build + test

### DP-4: Delete example_integration.go
`pkg/engine/example_integration.go` — 202 lines. Entire file is commented-out example code.
- Verify it's all comments (no executable code)
- Delete the file
- Build + test

### DP-11: Assess CrashLoad
`pkg/game/save.go` line 674 defines `CrashLoad`. Line 651 defines `CleanCrashFile`.
- Check: `grep -rn "CrashLoad\|CleanCrashFile" --include="*.go" pkg/ | grep -v "func \|slog\.\|// \|_test"` — are these called from anywhere besides each other?
- If CrashLoad is only called by CleanCrashFile and CleanCrashFile is never called from outside its own file, both are dead code → delete both functions
- If they ARE called, leave them alone and note it

### DP-7: Assess generateAffectID
`pkg/engine/affect.go` line 173 defines `generateAffectID()`. It creates IDs like `aff_20260102150405_abcdefgh`.
- Check: `grep -rn "generateAffectID\|\.ID ==\|\.ID !=\|RemoveAffect.*ID" --include="*.go" pkg/ | grep -v "func \|// \|_test"` — is the ID actually used for matching?
- The concern: "IDs never referenced" — maybe RemoveAffect uses the ID but is itself never called in a way that matters
- If the affect ID system is actively used (RemoveAffect is called with string IDs from combat/spells), leave it alone and note "actively used"
- If it's dead or only used by dead code, delete `generateAffectID()` and simplify `Affect.ID` to not generate random strings

## Final
After all changes, report: what was deleted, what was kept, why.
