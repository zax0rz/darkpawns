# 2026-07-03 Late Session Context (for new session)

## What happened tonight

Massive sprint session. Three coding agents worked in parallel (Claude Code, GLM-5.2 via ZCode, Daeron orchestrating).

### PRs Merged
- **PR #41** — CI gofumpt fix (unblocked 15 consecutive failing CI runs)
- **PR #42** — DP-903: Script load failure caching (86K log lines → 1)
- **PR #43** — DP-905: MakeHit deletion + DP-904: 6 U1000 suppressions retired + ratchet test
- **PR #44** — Sprint 2 MED/LOW: 7 fixes (DP-907/908/909/910/911/912/913)

### Issues Closed
- DP-903 (script cache), DP-904 (U1000 cleanup), DP-905 (MakeHit dead code)
- DP-907 (target resolver), DP-908 (wander cadence), DP-909 (char creation menus)
- DP-910 (JWT secret validation), DP-911 (save error logging), DP-912 (login idle timeout + guest names)
- DP-913 (cosmetic string bugs)

### Key Decisions
- **MakeHit killed, fight_core.go kept** — TakeDamage/DamMessage/BackstabMult are live. MakeHit was dead AND redundant on every axis.
- **Node.js not a prod runtime** — don't chase local/prod/CI Node parity. Keep Node only for CI tooling and OpenClaw local stack.
- **U1000 ratchet test** — `internal/lintguard/u1000_ratchet_test.go` prevents suppression count from growing. Baseline: 40 files. Retire file-by-file as dead code burns down.

### What's Left (Sprint 2)
- **DP-902** — Infra brief needed: session wedge + ghost players (linkdead reaper). The Architect said "infra, not fidelity." Needs a proper brief written.
- **DP-900/901** — Combat reciprocity + skill damage pipeline. BRIEF-2026-07-03-sprint2-critical.md exists. Needs execution.
- **DP-906** — Backstab C gates. BRIEF-2026-07-03-sprint2-backstab.md exists. Needs execution.
- **Remaining U1000 dead code** — 154 symbols across 40 files. Inventory at `docs/briefs/BRIEF-2026-07-03-dp904-u1000-inventory.md`.

### Known Issues
- **TestDoCharge_MountedBonusDamage** — pre-existing RNG-flaky test, fails intermittently under full-suite ordering, passes in isolation. Not from tonight's changes.
- **CI test failure on PR #43** — likely the same charge flake. Architect merged anyway.
- **Exec rendering as images** — transient OpenClaw gateway bug. All exec/read/write outputs rendered as image content. Gateway restart didn't fix. Might clear on new session.

### Files to Update in Linear
- DP-903: Done (fixed, commit a049114)
- DP-904: Done (fixed, commit from PR #43)
- DP-905: Done (fixed, commit from PR #43)
- DP-907 through DP-913: Done (fixed, commits from PR #44)

### Workflow Established
- **Daeron writes briefs → Coding agent executes → Daeron reviews PR → Merge → Linear update**
- Briefs live in `darkpawns_repo/docs/briefs/`
- Claude Code handles complex multi-file work (DP-905 MakeHit deletion)
- GLM-5.2 via ZCode handles batch quick fixes (DP-907-913)
- Both can work in parallel if files don't overlap

### Agent Notes
- **GLM-5.2** is proven tonight as an execution model via ZCode. Good at batch fixes.
- **Claude Code** is good for complex refactors with design decisions.
- **The Architect** merged all PRs tonight despite CI flake on #43. Trust the workflow.

### Session Notes
- Written to `darkpawns_repo/docs/session-notes/2026-07-03.md`
- Briefs at `darkpawns_repo/docs/briefs/`
