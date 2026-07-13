# Brief: Batch 3 — 2026-07-11

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## CRITICAL: Verify Against C Source Before Fixing

**Do NOT just apply the fix described below.** Read the C source file at the path specified in each `**Cite:**` field FIRST. Confirm the C behavior matches what this brief describes. If the C source says something different from what's written here, STOP and report the discrepancy. Fidelity to C is the entire point.

---

## Fix 1: DP-1032 — Bash: mobs take no wait-state, keep attacking, stay knocked down all fight (MED)

**Files:**
- `pkg/command/skill_commands.go:1628-1633` — WaitTarget wiring
- `pkg/game/skill_combat.go:136-214` — `DoBash()` (already sets `WaitTarget: 2`)
- Mob combat round calculation (needs investigation — see below)

**Problem:**
DoBash correctly sets `WaitTarget: 2` (PULSE_VIOLENCE * 2), but `skill_commands.go:1628-1633` only applies WaitTarget when the target is `*game.Player`:
```go
if result.WaitTarget > 0 && target != nil {
    if p, ok := target.(*game.Player); ok {
        p.SetWaitState(result.WaitTarget)
    }
}
```
Mob targets are silently ignored. Additionally, C's `perform_violence()` (fight.c:1975-1987) has a mob-side mechanism: when `GET_MOB_WAIT(ch) > 0`, the mob gets `attacks = 0` for that round, and when the wait expires mid-combat, the mob stands back up from sitting to fighting position. Go has no equivalent mob wait tracking.

The result: a bashed mob stays at `PosSitting` forever, keeps attacking every round as if nothing happened, and never stands back up.

**Fix:**
This requires two changes:

1. **Wire WaitTarget for mob targets.** In `skill_commands.go:1628-1633`, extend the WaitTarget handling to also apply to mob targets. You'll need to either:
   - Add a `MobWaitUntil time.Time` or `MobWaitTicks int` field to `MobInstance` and a `SetWaitState(ticks int)` method on it, OR
   - Use an existing mob wait mechanism if one exists (search the codebase first)

2. **Enforce mob wait in combat.** In the mob's combat round calculation (similar to C's `perform_violence` at fight.c:1975-1987), when the mob has an active wait:
   - Set attacks to 0 for this round (mob can't attack while stunned)
   - When wait expires and mob is below `PosFighting`, stand them back up ("$n scrambles to $s feet!")

**Cite:** `src/act.offensive.c:489-495` — successful bash sets `GET_POS(vict) = POS_SITTING; WAIT_STATE(vict, PULSE_VIOLENCE * 2)`. `src/fight.c:1975-1987` — `perform_violence()`: if `GET_MOB_WAIT(ch) > 0`, decrement by PULSE_VIOLENCE, set `attacks = 0`; if mob is below `POS_FIGHTING` and wait expired, stand to `POS_FIGHTING` with message.

**Regression Test:**
- `TestBashAppliesWaitToMob`: bash a mob → assert mob has an active wait state
- `TestBashedMobSkipsAttack`: bashed mob in combat → assert attacks = 0 while wait is active
- `TestBashedMobStandsWhenWaitExpires`: simulate wait expiry → assert mob returns to PosFighting

---

## Fix 2: DP-1037 — Flee XP penalty gated to level>10; no max_exp_loss cap (MED)

**File:** `pkg/session/movement_cmds.go:248-256`

**Problem:**
Two bugs in one:

**Bug A — Wrong level gate:** The entire XP loss calculation sits inside `if level > 10`:
```go
if level > 10 {
    xpLoss += int(500 * (float64(level) / 2.6))
    s.player.LoseExp(xpLoss)
    ...
}
```
In C, the base loss (`loss = (MAX_HIT - HIT) * LEVEL`) is computed at ALL levels. Only the bonus `500*(level/2.6)` is gated to `level > 10`. The Go code puts both inside the gate, so low-level characters flee for free.

**Bug B — No max_exp_loss cap:** C's `gain_exp()` (limits.c:319) caps any single XP loss at `max_exp_loss` (500,000). Go's `LoseExp()` likely has no such cap.

**Fix for Bug A:**
Restructure the XP loss to match C's logic:
```go
// Base loss from fight strength difference — applies at ALL levels.
// C: loss = (MAX_HIT(FIGHTING) - HIT(FIGHTING)) * LEVEL(FIGHTING)
loss := 0
if fighting := s.manager.combatEngine.GetFighter(s.player.Name); fighting != "" {
    // Get the opponent's stats — may need to resolve from combat engine
    // C source: act.offensive.c:362-368
    loss = (opponentMaxHP - opponentCurrentHP) * opponentLevel
}

level := s.player.GetLevel()
// Bonus loss for level > 10 — C: loss += 500*(GET_LEVEL(ch)/2.6)
if level > 10 {
    loss += int(500 * (float64(level) / 2.6))
}

if loss > 0 {
    // Cap at max_exp_loss — C: limits.c:319 gain = MAX(-max_exp_loss, gain)
    if loss > maxExpLoss {
        loss = maxExpLoss
    }
    s.player.LoseExp(loss)
    s.Send(fmt.Sprintf("You lose %d experience points for fleeing.", loss))
}
```

**⚠️ IMPORTANT:** The exact mechanism for getting the opponent's current/max HP and level during flee depends on how the combat engine exposes fighter data. Read the combat engine interface to find the right API. If the data isn't easily accessible during flee, note what's missing and implement what you can.

**Fix for Bug B:**
Verify that `LoseExp()` or its call chain applies the `maxExpLoss` cap (defined in `pkg/game/limits.go:35` as 500000). If it doesn't, add the cap.

**Cite:** `src/act.offensive.c:362-384` — base loss computed at all levels, bonus gated to level>10. `src/limits.c:319` — `gain = MAX(-max_exp_loss, gain)` caps per-death loss.

**Regression Test:**
- `TestFleeXPLossAppliesAtAllLevels`: level 5 character flees → assert XP is lost (currently 0)
- `TestFleeXPLossCap`: character with huge level → assert XP loss capped at maxExpLoss

---

## Fix 3: DP-1031 — Alignment never changes from kills (MED)

**File:** `pkg/game/death.go` — `handlePlayerDeath()` or `HandleDeath()` / `AwardMobKillXP()`

**Problem:**
`combat.ChangeAlignment(killer, victim)` exists and is correct (fight_core.go:262-275), but it's only called from unreachable code paths (the old TakeDamage death block at fight_core.go:541, and the legacy GroupGain at fight_core.go:979). The live death path (`HandleDeath` → `handlePlayerDeath` → `AwardMobKillXP`) never calls it.

As a result, player alignment never changes from combat. A paladin who kills 10,000 evil demons stays at the same alignment forever.

**Fix:**
Call `ChangeAlignment` from the appropriate point in the live death path. The C call is at fight.c:1667, right after the autogold/loot block and before the PK bookkeeping.

The fix is likely one line added to `AwardMobKillXP()` or `handlePlayerDeath()`:
```go
// After XP award, before PK bookkeeping
if killer != nil && !killer.IsNPC() {
    combat.ChangeAlignment(killer, victim)
}
```

**Verify before implementing:** Read the live death path to find the exact call site. The C order at fight.c:1660-1670 is: (1) autogold, (2) autoloot, (3) `change_alignment(ch, victim)`, (4) PK bookkeeping. Match this order.

**Cite:** `src/fight.c:484-502` — `change_alignment()` implementation. `src/fight.c:1667` — call site in the death block, after autoloot. The function: `GET_ALIGNMENT(ch) += (-GET_ALIGNMENT(victim) - GET_ALIGNMENT(ch)) >> 4`, clamped to [-1000, 1000]. Only affects PCs killing non-neutral NPCs.

**Regression Test:**
- `TestKillingEvilMobShiftsAlignmentGood`: evil-aligned mob killed by neutral player → player alignment shifts positive
- `TestKillingNeutralMobNoShift`: neutral mob killed → alignment unchanged (C: `IS_NEUTRAL(victim)` early return)
- `TestKillingGoodMobShiftsAlignmentEvil`: good mob killed by neutral player → alignment shifts negative

---

## Execution Order

1. Fix 3 (alignment) — smallest, one call site to wire
2. Fix 1 (bash mob wait) — needs investigation of mob wait mechanism
3. Fix 2 (flee XP) — needs combat engine API investigation
4. Run build gate

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/game/death.go pkg/command/skill_commands.go pkg/game/skill_combat.go pkg/session/movement_cmds.go pkg/game/mob.go
git add -u  # pick up any test files
git commit -m "fix: alignment from kills, bash mob wait-state, flee XP penalty (DP-1031, DP-1032, DP-1037)"
git push -u origin fix/batch3-crosscutting
gh pr create --title "fix: alignment from kills, bash mob wait-state, flee XP penalty (DP-1031, DP-1032, DP-1037)" --body "Fixes DP-1031, DP-1032, DP-1037. See docs/briefs/BRIEF-2026-07-11-batch3-crosscutting.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.
