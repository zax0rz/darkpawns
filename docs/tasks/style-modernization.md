# Style & Modernization — Subagent 3

You are cleaning up style issues and small modernizations in the Dark Pawns Go codebase.
Repository: `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**RULES:**
- Run `go build ./... && go vet ./... && go test ./pkg/engine/ ./pkg/game/` after every change
- If a build breaks, revert immediately
- Commit each fix separately with message `fix: DP-X — description`
- Small, surgical changes only.

## Tasks

### DP-6: Remove duplicate #nosec G404 comments
`pkg/engine/skill.go` — each `#nosec G404` comment appears twice on consecutive lines.
- 6 pairs of duplicates (12 duplicate lines total)
- Remove the duplicate line from each pair (keep the one with the descriptive comment like `// #nosec G404 — game RNG, not cryptographic`)
- Lines to check: 89-90, 106-107, 138-139, 142-143, 173-174, 179-180

### DP-9: Convert nested if/else to type switch
Search for nested if/else patterns that could be cleaner type switches.
- Look in `pkg/game/act_informative.go`, `pkg/game/act_item.go`, `pkg/game/act.go`
- Pattern: `if x, ok := y.(Type); ok { ... } else if x, ok := y.(Type2); ok { ... }`
- Convert to: `switch x := y.(type) { case Type: ... case Type2: ... }`
- Only convert if it genuinely improves readability. Don't force it.

### DP-62: Assess C-shaped file splits (DO NOT ACTUALLY SPLIT)
These files are too large and should eventually be split:
- `pkg/game/act_item.go` — 2021 lines
- `pkg/game/act_informative.go` — check size
- `pkg/game/spec_procs*.go` — check sizes

**DO NOT split any files.** Instead:
1. List each file and what it contains (function groups)
2. Propose a logical split plan (e.g., "act_item.go → act_item_use.go, act_item_trade.go, act_item_wear.go")
3. Write the plan to `docs/reports/file-split-plan.md`
4. This is research, not implementation

### DP-63: Assess CustomData replacement (DO NOT IMPLEMENT)
`pkg/game/object.go` has `CustomData map[string]interface{}`.
- Search for all CustomData usage: `grep -rn "CustomData" --include="*.go" pkg/`
- Assess what types are actually stored in CustomData
- Propose a typed alternative (e.g., a struct with known fields)
- Write the assessment to `docs/reports/customdata-assessment.md`
- This is research, not implementation

## Final
After all changes, report: what was fixed, what was assessed, what was left for later.
