# Brief: Quick Wins — 6 One-Line Fixes

**Filed by:** Daeron
**Date:** 2026-06-17
**Priority:** High
**Category:** Bug
**Estimated scope:** 6 files, ~12 lines changed total

## Summary

Six confirmed bugs from clawpatch's 95-finding batch. Each is a single-line or near-single-line fix with clear evidence and low risk. All have testable behavior.

**Deferred from this batch:** DP-611 (LiteLLM endpoint path) — we're decommissioning LiteLLM.

---

## Fix 1: HasFlag bounds check

**File:** `pkg/parser/wld.go` (line ~41)
**Bug:** `Room.HasFlag(bit)` computes `word = bit / 16` and accesses `r.Bounds[word]` without checking `word < 4`. For `bit >= 64`, this panics with index-out-of-range.
**Fix:** Add bounds check before array access: `if word < 0 || word >= 4 { return false }`
**Test:** Add `TestRoom_HasFlagOutOfBounds` — call `HasFlag(64)` and `HasFlag(100)`, verify both return false without panicking.

## Fix 2: GetAlignment nil guard

**File:** `pkg/combat/fight_core.go` (lines 263, 264, 437, 441)
**Bug:** `GetAlignment` is a global function pointer called at 4 sites without nil check. Every other function pointer in the file is nil-checked before use. A nil `GetAlignment` panics the combat goroutine.
**Fix:** Add `if GetAlignment != nil` guard before each call site. Match the pattern used by `HasAffect`, `BroadcastMessage`, `RemoveAffect`, etc.
**Test:** Add `TestChangeAlignment_NilGetAlignment` — call `ChangeAlignment` with `GetAlignment = nil`, verify no panic.

## Fix 3: GetRace nil guard

**File:** `pkg/combat/fight_core.go` (line 426)
**Bug:** Inside `TakeDamage`, the block guarded by `if GetRaceHate != nil` calls `GetRace(victimName)` without its own nil check. `GetRace` is a separate function pointer — it can be nil even when `GetRaceHate` is set.
**Fix:** Add `if GetRace != nil` inside the existing block.
**Test:** Add `TestTakeDamage_NilGetRace` — set `GetRaceHate` to non-nil, `GetRace` to nil, call `TakeDamage`, verify no panic.

## Fix 4: LoginAttemptTracker.Stop() double-close guard

**File:** `pkg/auth/ratelimit.go` (lines 204-206)
**Bug:** `LoginAttemptTracker.Stop()` calls `close(t.stop)` directly. A double call panics. `IPRateLimiter.Stop()` already uses `sync.Once` — this should match.
**Fix:** Add `sync.Once` guard around `close(t.stop)`, matching the pattern at line 114-116.
**Test:** Add `TestLoginAttemptTracker_StopTwice` — create tracker, call `Stop()` twice, verify no panic.

## Fix 5: cmdQcomm PRF_QUEST filter

**File:** `pkg/session/act_comm.go` (line ~52)
**Bug:** `cmdQcomm` broadcasts to ALL online players, but the comment says "question asked to all questing players." Non-questing players receive messages they shouldn't see.
**Fix:** Add `if sess.player.GetFlags()&(1<<uint(game.PrfQuest)) == 0 { continue }` in the broadcast loop.
**Test:** Add `TestCmdQcomm_NonQuestPlayerFiltered` — create two sessions (one with PRF_QUEST, one without), send qcomm from the questing player, verify the non-questing player does not receive it.

## Fix 6: do_split.py hardcoded build path

**File:** `scripts/do_split.py` (line 56)
**Bug:** `fix_imports()` passes `cwd='/home/zach/darkpawns'` — a developer-specific path that doesn't exist on any other machine.
**Fix:** Replace with `cwd=os.path.dirname(os.path.dirname(os.path.abspath(__file__)))` to derive project root from script location.
**Test:** Not unit-testable in a meaningful way. Manual verification: run `python scripts/do_split.py` from a different directory, confirm it doesn't fail with directory-not-found.

---

## Verification Plan

```bash
go build ./...
go vet ./...
go test ./pkg/parser/... -run TestRoom_HasFlag -v
go test ./pkg/combat/... -run "TestChangeAlignment_Nil|TestTakeDamage_Nil" -v
go test ./pkg/auth/... -run TestLoginAttemptTracker_Stop -v
go test ./pkg/session/... -run TestCmdQcomm -v
```

## What "done" looks like

- All 6 fixes applied
- 5 new tests written and passing
- `go build ./... && go vet ./... && go test ./...` clean
- One commit, pushed to branch, PR ready
