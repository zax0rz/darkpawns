# Brief 2: Combat Cleanup

**Filed by:** Daeron
**Date:** 2026-06-17
**Priority:** Medium
**Category:** Bug / Test
**Estimated scope:** 3 files, ~15 lines changed, 2 tests

## Summary

Four findings in the combat subsystem. Three are bugs (nil panic, missing data, display corruption), one is a flaky test. All are core game — no AI dependency.

**Deferred:** FSMDecision dead code — in `pkg/agentcli`, AI-dependent.

---

## Fix 1: broadcastCombatMsg nil player panic

**File:** `pkg/session/act_offensive.go` (lines 13-18)

**Bug:** `broadcastCombatMsg` accesses `s.player.Name` at line 16 without checking if `s.player` is nil. A nil player causes a panic. Defensive nil guards exist elsewhere in the codebase — this one was missed.

**Fix:** Add `if s.player == nil { return }` at the start of the function. Match the defensive pattern used in other session methods.

**Test:** Add `TestBroadcastCombatMsg_NilPlayer` — call `broadcastCombatMsg` with a session where `player` is nil, verify no panic. This is a regression guard.

---

## Fix 2: Combat hit/miss messages missing \r\n

**File:** `pkg/combat/engine.go` (lines 362, 365, 381, 382)

**Bug:** `sendHitMessage` and `sendMissMessage` format strings don't include trailing `\r\n`. Every other `SendMessage` call in the same file and in `fight_core.go` includes `\r\n`. Missing terminators cause subsequent MUD output to concatenate on the same telnet line — display corruption.

**Fix:** Append `\r\n` to all four format strings:
- `'You hit %s for %d damage!\r\n'`
- `'%s hits you for %d damage!\r\n'`
- `'You miss %s!\r\n'`
- `'%s misses you!\r\n'`

**Test:** Add `TestCombatMessages_HaveNewlines` — call `sendHitMessage` and `sendMissMessage` with a mock writer, verify the written bytes end with `\r\n`. This catches the pattern if anyone strips newlines in a future refactor.

---

## Fix 3: CombatPair.LastAttackType never written

**File:** `pkg/combat/engine.go` (lines 21, 321, 434-442)

**Bug:** `CombatPair.LastAttackType` is documented as tracking the attack type that killed the victim, but `processCombatPair` never assigns to it. `handleDeath` reads it at lines 436-440 and always gets `0` (TYPE_UNDEFINED). Death messages, corpse creation, and kill tracking lose the attack type information.

**Fix:** Set `pair.LastAttackType` to the appropriate attack type (e.g., `AttackNormal` for engine melee) at the point where damage is calculated in `processCombatPair`, before the damage is applied. Check the existing attack type resolution logic in `fight_core.go` to match the correct type.

**Test:** Add `TestHandleDeath_PassesAttackType` — set up a `CombatPair`, simulate a kill via `processCombatPair`, verify `DeathFunc` receives the correct attack type (not 0).

---

## Fix 4: CheckSavingThrow flaky test

**File:** `pkg/spells/saving_throws_test.go` (lines 196-225)

**Bug:** `TestCheckSavingThrow_HighLevelSavesOften` and `TestCheckSavingThrow_Level0SavesRarely` use probabilistic assertions with only 100 iterations. While the current thresholds are technically safe, 100 iterations is small and could produce borderline results under non-uniform RNG.

**Fix:** Increase iteration count to 1000 and adjust thresholds proportionally. Alternatively, mock the RNG for deterministic output — but that's a larger change. The 1000-iteration approach is simpler and sufficient.

**Test:** This IS the test fix. Update the existing test, don't add a new one.

---

## Verification Plan

```bash
go build ./...
go vet ./...
go test ./pkg/session/... -run TestBroadcastCombatMsg -v
go test ./pkg/combat/... -run "TestCombatMessages|TestHandleDeath" -v
go test ./pkg/spells/... -run TestCheckSavingThrow -v
go test -race ./pkg/combat/...
```

## What "done" looks like

- 3 bugs fixed, 1 flaky test hardened
- 3 new tests + 1 updated test, all passing
- `go build ./... && go vet ./... && go test ./...` clean
- One commit, pushed to branch
