# Brief: Stream 4b v2 — Combat Hook Cleanup (F8)

## Context

DP-952 from the Fable Audit 2026-07-05 (originally C-02 from April review).
57 package-level function hooks in `pkg/combat/fight_core.go:14-77`.

**Previous attempt (PR #99, reverted):** A prior brief classified 51 hooks as
"dead" and instructed deletion. The agent (Gemini) gutted ~15 live functions
to empty stubs after deleting their hooks, introducing severe regressions
(RawKill no longer creates corpses, GroupGain broken, DeathCry no longer
broadcasts to adjacent rooms, TakeDamage lost shopkeeper/peaceful room checks,
engine.go haste/slow hardcoded false). Reverted as PR #100.

**Root cause of failure:** The original triage was wrong. Only 3 hooks are truly
dead (never referenced in production code). The other 54 are all on live code
paths — mostly through `TakeDamage()`, which is the central death pipeline called
from `engine.go`, `spec_procs3.go`, and `damage_spells.go`. Deleting the hooks
without providing an alternative pathway breaks the combat→game bridge.

**Correct scope for this brief:** Small, safe, verified cleanup — delete the 3
truly dead hooks + 6 dead wrapper functions, add the GetExp nil guard, add
construction-time validation. The larger migration (Combatant interface
expansion or GameCallbacks struct) is a separate future effort.

**Linear:** DP-952 (F8)
**Branch:** `fix/stream4b-combat-hooks-v2`
**Agent:** Kimi

## Problem

57 package-level `var` hooks in `fight_core.go`. Characteristics:
- **Zero synchronization** — write at boot, read at runtime, no mutex/atomic
- **1 hook lacks a nil guard** (`GetExp`) — nil-panic risk
- **3 hooks are truly dead** — declared but never referenced in any production code
- **6 wrapper functions are dead** — defined but never called from anywhere
- **48 hooks are live** — used by fight_core functions on live code paths

### The 3 Truly Dead Hooks

These are declared in the var block but have zero references in production code:

| Hook | Declared | Why Dead |
|---|---|---|
| `BuildTHAC0` | line 58 | THAC0 calculation done via `Combatant.GetTHAC0()` interface method |
| `RunFightScript` | line 31 | Fight scripts handled via `CombatEngine.OnCombatAction` callback |
| `IsInRoom` | line 71 | Location checked via `Combatant.GetRoom()` + room VNum comparison |

### The 6 Dead Wrapper Functions

These are public functions in fight_core.go that wrap a single hook call but
are never called from any external package:

| Function | Wraps | Why Dead |
|---|---|---|
| `Die(ch Combatant)` | `GainExp` + `RawKill` | `TakeDamage` uses `DieWithKiller` instead |
| `MakeCorpse(victim, attackType)` | `MakeCorpseFunc` | `RawKill` calls the hook directly |
| `MakeDust(victim, attackType)` | `MakeDustFunc` | `RawKill` calls the hook directly |
| `SkillMessage(dam, ch, vict, ...)` | `SkillMessageFunc` | `TakeDamage` calls the hook directly |
| `Appear(ch Combatant)` | `HasAffect` + `RemoveAffect` + `BroadcastMessage` | No callers anywhere in codebase |
| `fleshAlteredType(level)` | (none — no hooks) | Unexported, no callers |

### The 1 Dangerous Hook (Missing Nil Guard)

`GetExp` — called at `fight_core.go:565`, `fight_core.go:1088`, and
`fight_core.go:1153` without nil check. If `GetExp` is nil, any combat that
triggers group gain, death XP calculation, or PK death penalty will panic.
Note: line 1184 is inside `Die()` (being deleted), so it's out of scope.

## Fix Strategy — Surgical Cleanup Only

### 1. Delete the 3 dead hooks

Remove `BuildTHAC0`, `RunFightScript`, and `IsInRoom` from the var block.
These have zero references — removing them is pure dead code elimination.

### 2. Delete the 6 dead wrapper functions

Remove `Die()`, `MakeCorpse()`, `MakeDust()`, `SkillMessage()`, `Appear()`,
and `fleshAlteredType()`. Verify each has no callers across all of `pkg/`
(excluding test files) before deleting.

**Tests to update:**
- `pkg/combat/round_test.go` — remove `TestAppear_Basic`, `TestAppear_ImmortalLevel`,
  and any `RunFightScript` assignment scaffolding
- `pkg/combat/fight_core_test.go` — remove `TestFleshAlteredType`

**IMPORTANT:** Do NOT touch any other functions. Do NOT modify `TakeDamage()`,
`RawKill()`, `DieWithKiller()`, `DeathCry()`, `GroupGain()`, `CounterProcs()`,
`AttitudeLoot()`, `ChangeAlignment()`, `IsInGroup()`, or `DamMessage()`.
These are all live functions with real logic.

### 3. Add nil guard on GetExp

At three call sites — `fight_core.go:565`, `fight_core.go:1088`, and
`fight_core.go:1153` — add nil guard:

```go
// Line 565 (GroupGain):
victimExp := 0
if GetExp != nil {
    victimExp = GetExp(victim.GetName())
}

// Line 1088 (PerformGroupGain):
share := 0
if GetExp != nil {
    share = GetExp(victim.GetName())
}

// Line 1153 (DieWithKiller — inside GainExp != nil block):
// Must guard GetExp even when GainExp is non-nil:
if GainExp != nil {
    loss := 0
    if GetExp != nil {
        loss = GetExp(ch.GetName()) / 37
    }
    GainExp(ch.GetName(), -loss)
}
```

### 4. Add construction-time validation

After wiring hooks in `cmd/server/main.go`, add a validation block:
```go
// Verify critical combat hooks are wired (DP-952)
if combat.BroadcastMessage == nil || combat.SendToCharFunc == nil {
    slog.Error("critical combat hook not wired — combat messages will be silent")
}
```

This doesn't block startup but makes partial init visible in logs.

### 5. Document the remaining hooks

Add a comment block above the remaining 54 hooks explaining:
- They are the bridge between the combat package (Combatant interface) and
  the game layer (Player/Mob structs with direct state access)
- Only 4 are wired in production: `BroadcastMessage`, `SendToCharFunc`,
  `DoFlee`, `DoRetreat` (all in `pkg/session/manager.go`)
- The other ~50 are referenced on live code paths but nil in production,
  relying on existing nil guards
- The primary combat path is CombatEngine struct callbacks; these hooks
  serve the legacy fight_core path
- Future work should migrate these to a GameCallbacks struct (see v3 brief)

## Files

- **`pkg/combat/fight_core.go:14-77`** — delete 3 dead hooks, delete 6 dead functions (MODIFY)
- **`pkg/combat/fight_core.go:565,1088,1153`** — add GetExp nil guard (MODIFY)
- **`pkg/combat/round_test.go`** — remove TestAppear_* tests and RunFightScript scaffolding (MODIFY)
- **`pkg/combat/fight_core_test.go`** — remove TestFleshAlteredType (MODIFY)
- **`cmd/server/main.go`** — add construction-time validation (MODIFY)

### Note: Latent bug in affect_spells.go

`pkg/spells/affect_spells.go:~2807` has a type assertion `killer interface{ Die() }`
that is never satisfied — no type implements `Die()`. This is a pre-existing
fidelity bug (gate spell should kill the caster). Out of scope for this brief
but worth tracking separately.

## Do NOT Modify

- `TakeDamage()` — live, called from engine.go + spec_procs + damage_spells
- `RawKill()` — live, called from skill_combat.go + affect_spells.go
- `DieWithKiller()` — live, called from TakeDamage on kill
- `DeathCry()` — live, called from RawKill
- `GroupGain()` / `PerformGroupGain()` — live, called from TakeDamage
- `CounterProcs()` — live, called from TakeDamage
- `AttitudeLoot()` / `BragMessage()` — live, called from TakeDamage
- `ChangeAlignment()` — live, called from PerformGroupGain
- `IsInGroup()` — live, called from CalcLevelDiff + TakeDamage
- `DamMessage()` — live, called from TakeDamage + manager.go
- `CheckParry()` / `CheckDodge()` — live, called from engine.go
- `engine.go processCombatPair()` — do NOT hardcode haste/slow to false
- `formulas.go` — do NOT remove GetWeaponInfo usage in CheckParry
- `skill_messages.go` — do NOT remove GetCharacterSex usage

## Regression Test

```go
func TestGetExpNilGuard_GroupGain(t *testing.T) {
    // With GetExp = nil, call GroupGain. Verify no panic, share defaults to 0.
}

func TestGetExpNilGuard_DieWithKiller(t *testing.T) {
    // With GetExp = nil but GainExp wired, call DieWithKiller.
    // Verify no panic on the GainExp(chName, -(GetExp(chName)/37)) path.
}
```

## Build Gate

Must match CI (`ci.yml`):

```bash
go build ./... \
  && go vet ./... \
  && go test -race $(go list ./... | grep -v /tests/unit) -v -timeout 120s \
  && gofumpt -l . \
  && golangci-lint run ./...
```

All must pass before committing.

## Commit

```
refactor: delete 3 dead combat hooks + 6 dead wrappers, add GetExp nil guard (DP-952)
```
