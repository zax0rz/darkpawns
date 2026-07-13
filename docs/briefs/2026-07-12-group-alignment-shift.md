# DP-1061: Group Kills Only Shift the Killer's Alignment — C Shifts Every Member

**Target:** `pkg/game/death.go`, `pkg/game/party.go`
**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/group-alignment-shift`
**After fixing:** `go build ./... && go vet ./... && go test ./...`

---

## Problem

The solo-path alignment shift in `pkg/game/death.go` (lines ~178-198 in the old code, now refactored in PR 238) is correct — formula, `>>4` arithmetic shift on negatives, ±1000 clamp, ±350 neutral gate all match C's `change_alignment()` in `src/fight.c:484-502`.

**But:** C calls `change_alignment(member, victim)` for **every group member** inside `perform_group_gain()` (`src/fight.c:688-705`). Go's group loop in `pkg/game/party.go:259-267` awards XP to each member but **never shifts their alignment**. Only the killer gets shifted — and that shift happens in `HandleDeath`, not in the group award path.

## C Source Flow

**Solo kill** (`src/fight.c` around line 1667):
1. `die_with_killer()` → calls `change_alignment(ch, victim)` directly on the killer

**Group kill** (`src/fight.c` around line 1686):
1. `die_with_killer()` → calls `group_gain(ch, victim)`
2. `group_gain()` → calls `perform_group_gain(member, base, victim)` for each in-room member
3. `perform_group_gain()` (lines 688-705):
```c
void perform_group_gain(struct char_data *ch, int base, struct char_data *victim) {
   int share = calc_level_diff(ch, victim, base);
   // ... XP message and gain ...
   if (!IS_NPC(ch))
      gain_exp(ch, share);
   change_alignment(ch, victim);  // ← THIS: every group member shifts
}
```

Key: `change_alignment` is called INSIDE the per-member loop. Every grouped PC gets the shift, not just the killer.

## Go's Current Flow

**`HandleDeath` in `pkg/game/death.go`:**
- Calls `AwardMobKillXP(killer, ...)` 
- Then shifts the killer's alignment (the big block with `(-victimAlign-pk.GetAlignment())>>4`)

**`AwardMobKillXP` in `pkg/game/party.go`:**
- Solo branch: awards XP to killer
- Group branch: loops `inRoom` members, awards XP to each — **no alignment shift**

Result: grouped non-killer members never shift alignment from kills.

## The Fix

The cleanest approach (matching C's structure):

1. **Extract the alignment shift logic** from `HandleDeath` into a reusable function:
```go
// changeAlignment shifts a player's alignment based on victim alignment.
// Source: fight.c:484-502 change_alignment(ch, victim)
func changeAlignment(player *Player, victimAlign int) {
    if victimAlign > -350 && victimAlign < 350 {
        return // neutral victims don't shift
    }
    newAlign := player.GetAlignment() + (-victimAlign-player.GetAlignment())>>4
    if newAlign > 1000 {
        newAlign = 1000
    }
    if newAlign < -1000 {
        newAlign = -1000
    }
    player.SetAlignment(newAlign)
}
```

2. **In `HandleDeath`'s solo path:** call `changeAlignment(killer, victimAlign)` where the killer is known and is a player. This replaces the existing inline block.

3. **In `AwardMobKillXP`'s group loop:** add `changeAlignment` for each member:
```go
for _, m := range inRoom {
    xp := calcKillXPShare(m.GetLevel(), victimLevel, base, true)
    w.GainExp(m, xp)
    // ... XP messages ...
    
    // Alignment shift for every group member — fight.c:704
    if !m.IsNPC() {
        if p, ok := m.(*Player); ok {
            changeAlignment(p, victimAlign)
        }
    }
}
```

4. **Remove the duplicate killer shift** from `HandleDeath` when the kill goes through the group path. The alignment shift should happen in the award paths (solo branch shifts killer, group branch shifts each member) — NOT after `AwardMobKillXP` returns. This is where C does it.

5. **`AwardMobKillXP` needs victim alignment passed in.** Either:
   - Add a `victimAlign int` parameter to `AwardMobKillXP`, or
   - Have it extract from the victim Combatant (check if `GetAlignment()` exists on the interface — if not, add the parameter)

## Alignment Shift Formula (for reference)

From `src/fight.c:484-502`:
```c
if (IS_NPC(ch)) return;           // mobs don't shift
if (IS_NEUTRAL(victim)) return;   // neutral victims don't affect alignment
GET_ALIGNMENT(ch) += (-GET_ALIGNMENT(victim) - GET_ALIGNMENT(ch)) >> 4;
if (GET_ALIGNMENT(ch) > 1000) GET_ALIGNMENT(ch) = 1000;
if (GET_ALIGNMENT(ch) < -1000) GET_ALIGNMENT(ch) = -1000;
```

The `>>4` is arithmetic right shift (Go matches this for signed ints). The formula pulls the killer toward the opposite of the victim's alignment by 1/16th.

## Files to Modify

- `pkg/game/death.go` — extract `changeAlignment()`, remove post-award killer shift for group path
- `pkg/game/party.go` — add alignment shift in group loop, possibly in solo branch too

## Test

Add a unit test in `pkg/game/party_test.go` (or wherever award tests live) that:
1. Creates 3 grouped players in the same room
2. Creates a mob victim with strong evil alignment (e.g., -1000)
3. Calls `AwardMobKillXP` for the group
4. Verifies ALL players' alignment shifted toward +, not just the killer

**Commit message:** `fix: shift alignment for every group member on kill, not just killer (DP-1061)`
