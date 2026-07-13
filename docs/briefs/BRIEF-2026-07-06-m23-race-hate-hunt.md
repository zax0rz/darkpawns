# BRIEF — M-23: Port race-hate aggression + double-speed hunting from mobact.c

**Linear:** DP-968 (M-23 remainder: verify mobile_activity scavenger/memory/helper behaviors vs mobact.c)
**Effort:** M
**Agent:** Kimi
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — M-23 (Phase 1E, partially closed)

## Goal

Port two missing `mobile_activity()` behaviors from `src/mobact.c` to Go:

1. **Race-hate aggression** — mobs attack players whose race_hate list matches the mob's race (mobact.c:236-258)
2. **Double-speed hunting** — hunter mobs call `hunt_victim()` twice per tick (mobact.c:74-80)

## Background

April's M-23 flagged `mobile_activity()` as not fully ported. DP-908 fixed wander cadence and SENTINEL/STAY_ZONE. The remaining behaviors were audited on 2026-07-05. Five of six behaviors (scavenger, memory, helper, aggressive/AGGR24, hunting) are already ported in `pkg/game/mobact.go`. Two gaps remain:

| Behavior | Status | C Location |
|---|---|---|
| Race-hate aggression | ❌ Missing | `src/mobact.c:236-258` |
| Double-speed hunt | ⚠️ Half-done | `src/mobact.c:74-80` vs `pkg/game/mobact.go:191-194` |

### What already exists in Go

The **combat damage bonus** for race_hate is already wired in:

- `pkg/combat/callbacks.go:19` — `GetRaceHate func(name string, index int) int` on `GameCallbacks`
- `pkg/combat/fight_core.go:298-304` — iterates 5 slots, adds `ch.GetLevel()` damage per match (matching `src/fight.c:1466-1468`)

What's missing: the **data field**, the **mobact aggression scan**, the **APPLY_RACE_HATE affect handler**, and the **double-hunt call**.

## C Source — Read These Directly

### Race-hate aggression (src/mobact.c:236-258)

```c
/* race hate haters */
if ( GET_MOB_SPEC(ch) != shop_keeper)
 for (found = FALSE,vict = world[ch->in_room].people;
     vict && !found;
     vict = vict->next_in_room)
 {
  int i = 0;
  for (i = 0; i < 5 && !found; i++)
    if (GET_RACE_HATE(vict, i) == GET_RACE(ch))
{
 if (!IS_NPC(vict) && CAN_SEE(ch, vict) &&
    !PRF_FLAGGED(vict, PRF_NOHASSLE) &&
       (!IS_AFFECTED(vict, AFF_PROTECT_EVIL) ||
    (IS_EVIL(ch) && !number(0,5))))
  {
    if (!number(0,5) && can_speak(ch))
          act("'Come to destroy my kin? Die!', exclaims $n.",
          FALSE, ch, 0, 0, TO_ROOM);
    hit(ch, vict, TYPE_UNDEFINED);
    found = TRUE;
  }
}
 }
```

Key semantics:
- Only runs if mob is NOT a shop_keeper spec
- Scans **victims in room** (not the mob itself) for race_hate matches
- The **player's** `race_hate[]` slots are checked against the **mob's** race
- Only targets **non-NPC** players (PCs)
- `CAN_SEE` check (standard visibility)
- `PRF_NOHASSLE` immunity (wizards)
- `AFF_PROTECT_EVIL` blocks attack UNLESS the mob is evil AND fails a 1-in-6 check
- 1-in-6 chance of flavor text (if mob can speak)
- `hit(ch, vict, TYPE_UNDEFINED)` — starts combat
- `found = TRUE` — only attacks one target per tick

### Double-speed hunting (src/mobact.c:74-80)

```c
/* hunt two steps at a time to do it faster */
if ( (GET_POS(ch) == POS_STANDING) && (MOB_FLAGGED(ch, MOB_HUNTER))
    && (ch->fighting == NULL))
    hunt_victim(ch);
if ( (GET_POS(ch) == POS_STANDING) && (MOB_FLAGGED(ch, MOB_HUNTER))
    && (ch->fighting == NULL))
    hunt_victim(ch);
```

The exact same block is duplicated — called twice per tick. Go only calls once at `pkg/game/mobact.go:191-194`.

### Race-hate data (src/structs.h:951)

```c
long race_hate[5];  /* 5 races you're allowed to hate :) */
```

Initialized to -1 for all 5 slots in `src/db.c:1114`, `2456`, `2966`.

### APPLY_RACE_HATE affect (src/handler.c:205-238)

This is how race_hate slots get populated at runtime — via object affects with `location = APPLY_RACE_HATE` (constant 25). The handler:

- `mod < 0`: removes matching race_hate slot (cancels hatred)
- `mod == 0`: toggles human (race 0) hatred (special case — only one human slot)
- `mod > 0`: adds race to first empty (-1) slot

This is called when items with `APPLY_RACE_HATE` are equipped/unequipped.

## Fix

### 1. Add `RaceHates` field to `MobInstance` and `Player`

```go
// In MobInstance (pkg/game/mob.go):
RaceHates [5]int // race_hate from mobact.c/structs.h — 5 race slots, -1 = empty

// In Player (pkg/game/player.go):
RaceHates [5]int // race_hate — populated via APPLY_RACE_HATE affects
```

Initialize all slots to -1 in `NewMobInstance` / `Spawner`, and in player creation/creation code.

### 2. Add `ApplyRaceHate` constant

In `pkg/game/deferred_fight_fns.go`, add to the APPLY_* constants:

```go
ApplyRaceHate = 25
```

### 3. Wire `GameCallbacks.GetRaceHate` implementation

The callback is already defined in `pkg/combat/callbacks.go:19` but has no game-package implementation. When setting up `GameCallbacks`, add:

```go
GetRaceHate: func(name string, index int) int {
    // Look up player or mob by name, return RaceHates[index]
    // Return -1 if not found or index out of range
},
```

This requires access to World state to look up by name. Follow the existing callback wiring pattern (see how other GameCallbacks are populated in the server init code).

### 4. Add race-hate aggression to `mobile_activity()` (mobact.go)

Insert the race-hate scan block **after the aggressive/AGGR24 blocks** and **before the memory block** (matching C order in mobact.c). Add it around line ~260 in `pkg/game/mobact.go`, before the `// -- Mob Memory --` comment.

```go
// -- Race-hate aggression (mobact.c:236-258) --
if ch.GetSpec() != "shop_keeper" {
    players := w.GetPlayersInRoom(ch.GetRoom())
    for _, vict := range players {
        attacked := false
        for i := 0; i < 5 && !attacked; i++ {
            if vict.RaceHates[i] != mob.GetRace() {
                continue
            }
            // Only targets PCs (vict is a Player, so always non-NPC)
            if !w.canSee(mob, vict) || vict.HasNoHassle() {
                continue
            }
            if vict.HasAffect(AFF_PROTECT_EVIL) && !(mob.IsEvil() && rand.IntN(6) == 0) {
                continue
            }
            if rand.IntN(6) == 0 && canSpeak(mob) {
                w.RoomEcho(ch.GetRoom(), "'Come to destroy my kin? Die!', exclaims %s.", ch.GetName())
            }
            if err := w.StartCombat(mob, vict); err != nil {
                slog.Warn("StartCombat failed in race-hate mob", "mob", ch.GetName(), "error", err)
            }
            attacked = true
        }
        if attacked {
            break // Only one victim per tick
        }
    }
}
```

**Adapt as needed** — the exact Go API may differ from the pseudocode above. Check the existing mobact.go patterns for how `canSee`, `StartCombat`, `RoomEcho`, etc. are called. Key things to match from C:

- Skip if mob is shop_keeper spec
- Only attack PCs (players in room)
- Check CAN_SEE and PRF_NOHASSLE
- AFF_PROTECT_EVIL blocks unless mob is evil AND `number(0,5)` (1-in-6 pass)
- 1-in-6 flavor text (if can speak)
- Only one victim per tick

### 5. Double-speed hunting fix (mobact.go:191-194)

Duplicate the hunt block. Currently:

```go
if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
    w.huntVictim(ch)
}
```

Change to (matching C's two identical calls):

```go
if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
    w.huntVictim(ch)
}
if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
    w.huntVictim(ch)
}
```

Yes, this is intentional duplication — the C source duplicates it too (mobact.c:74-80). The first call may change position/fighting state, so the second check re-evaluates.

### 6. APPLY_RACE_HATE affect handler (lower priority)

The `APPLY_RACE_HATE` affect handler from `src/handler.c:205-238` controls how equipping items modifies race_hate slots. This is triggered when items with `apply location 25` are equipped/unequipped.

**Assessment needed**: Check if any objects in the world files actually have `APPLY_RACE_HATE` affects. If none exist, this handler can be stubbed with a TODO comment and filed as a follow-up. If objects do use it, port the full handler from handler.c:205-238.

The handler logic (for reference):
- `mod < 0`: find slot matching `-mod`, clear to -1
- `mod == 0`: toggle human (race 0) — special case
- `mod > 0`: find first empty (-1) slot, set to mod

## Files

| File | Change |
|---|---|
| `pkg/game/mob.go` | Add `RaceHates [5]int` to `MobInstance` |
| `pkg/game/player.go` | Add `RaceHates [5]int` to `Player` |
| `pkg/game/deferred_fight_fns.go` | Add `ApplyRaceHate = 25` constant |
| `pkg/game/mobact.go` | Add race-hate aggression scan + double-speed hunt |
| `pkg/game/spawner.go` or init code | Initialize RaceHates to [-1, -1, -1, -1, -1] |
| Callback wiring location | Implement `GetRaceHate` callback |
| `pkg/game/mobact_test.go` | Tests for race-hate and double-hunt |

## Tests

- `TestRaceHateAggression_MobAttacksHater` — player with RaceHates[i] matching mob's race → mob attacks
- `TestRaceHateAggression_ShopKeeperSkips` — shop_keeper spec mob does NOT race-hate attack
- `TestRaceHateAggression_ProtectEvilBlocks` — player with AFF_PROTECT_EVIL blocks non-evil mob
- `TestRaceHateAggression_ProtectEvilEvilPasses` — evil mob has 1-in-6 chance to bypass protection
- `TestDoubleSpeedHunt_CalledTwice` — verify huntVictim is called twice for hunter mob
- `TestCombatRaceHateDamage` — verify damage bonus in fight_core.go when RaceHate matches (may already exist)

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **C fidelity is paramount.** Read the C source directly. The race-hate scan semantics (protection bypass, NOHASSLE immunity, shop_keeper skip, one victim per tick) must match exactly.
2. **Do NOT change existing combat damage bonus logic** in `pkg/combat/fight_core.go:298-304`. That's already correct.
3. **Do NOT port the full handler.c APPLY_RACE_HATE handler unless objects exist that use it.** Check first. If no objects use it, add a TODO and note it as a follow-up.
4. **Double-speed hunt is intentional duplication.** Do not refactor into a loop — match the C structure.
5. **Use the existing mobact.go patterns** for combat initiation (`w.StartCombat`, `slog.Warn` on error, etc.).
6. Single PR.

## C Fidelity Notes

- `src/mobact.c:236-258` — race-hate aggression (primary source)
- `src/mobact.c:74-80` — double-speed hunting
- `src/structs.h:951` — `long race_hate[5]` field
- `src/db.c:1114` — initialization to -1
- `src/handler.c:205-238` — APPLY_RACE_HATE affect handler (port if needed)
- `src/fight.c:1466-1468` — combat damage bonus (already ported)
