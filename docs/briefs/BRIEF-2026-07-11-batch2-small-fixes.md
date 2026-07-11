# Brief: Batch 2 — 2026-07-11

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## CRITICAL: Verify Against C Source Before Fixing

**Do NOT just apply the fix described below.** Read the C source file at the path specified in each `**Cite:**` field FIRST. Confirm the C behavior matches what this brief describes. If the C source says something different from what's written here, STOP and report the discrepancy. Fidelity to C is the entire point.

---

## Fix 1: DP-1035 — Game-hour ticks run 2.1× fast (30s vs 63s); mob AI double-driven (MED)

This fix has two parts.

### Part A: Point update ticker

**File:** `pkg/game/world.go:194` — `StartPointUpdateTicker(30 * time.Second)`

**Problem:**
C's mud hour is 63 seconds (SECS_PER_MUD_HOUR). The point_update/affect_update cycle runs once per mud hour. Go uses 30 seconds — making the game clock run 2.1× too fast. Regen, hunger, thirst, and affect durations all tick at more than double the intended rate.

**Fix:**
Change line 194 from:
```go
w.StartPointUpdateTicker(30 * time.Second)
```
to:
```go
w.StartPointUpdateTicker(63 * time.Second)
```

Update the comment on line 193 accordingly:
```go
// Called every 63 seconds (src/utils.h:135 SECS_PER_MUD_HOUR = 63)
```

**Cite:** `src/utils.h:135` — `#define SECS_PER_MUD_HOUR 63`. `src/comm.c:825-828` — affect_update() and point_update() fire on `pulse % (SECS_PER_MUD_HOUR * PASSES_PER_SEC)`.

### Part B: Mob AI double-driven

**File:** `pkg/game/ai.go:182-195` — `StartAITicker()` AND `pkg/game/world.go:190` — startup call

**Problem:**
Mob AI is driven by TWO mechanisms:
1. The game loop's `OnMobileActivity` fires every `PULSE_MOBILE` (4 seconds) — faithful to C.
2. `StartAITicker()` fires `AITick()` every 10 seconds — an additional driver that doesn't exist in C.

The C source only has `mobile_activity()` called from `heartbeat()` on `PULSE_MOBILE` (4s). There is no second AI tick. The Go `StartAITicker()` duplicates work and causes mobs to process AI 1.5× faster than C.

**Fix:**
Remove the `StartAITicker()` call at `world.go:190` and the `StartAITicker()` function itself at `ai.go:182-195`. The existing `OnMobileActivity` in the game loop is the faithful C driver. Also remove the `aiticker` field from World if it exists.

**⚠️ IMPORTANT:** Before removing, verify that `AITick()` does something `OnMobileActivity` does not. Read both functions carefully. If `AITick()` handles event processing or non-mob AI work that `OnMobileActivity` doesn't, keep `AITick()` but wire it into the existing 4s game loop instead of running its own ticker. The goal is ONE AI driver, not zero.

**Cite:** `src/comm.c:815-818` — `if (!(pulse % PULSE_MOBILE)) mobile_activity();`. `src/structs.h:633` — `#define PULSE_MOBILE (4 RL_SEC)`. There is no second AI driver in C.

**Regression Test:**
- Add `TestPointUpdateTickerInterval`: verify the ticker interval is 63s (not 30s). If StartPointUpdateTicker stores the interval, assert it. If it's hard to test, add a comment explaining why.
- No functional test needed for mob AI removal — existing mob behavior tests cover this path.

---

## Fix 2: DP-1036 — Object/corpse decay driven per-mob-in-room (N mobs = N× decay) (MED)

**File:** `pkg/game/limits_condition.go:284` — inside the NPC loop in `PointUpdate()` (or equivalent tick function)

**Problem:**
`w.decayObjectsInRoom(roomVNum)` is called once per active mob in the room. If 5 mobs are in a room, corpses decay 5× faster than intended. C's `point_update()` iterates the global object_list exactly once per tick — each object's timer is decremented exactly once regardless of how many mobs are in the room.

**Fix:**
Add a room-dedup set before the NPC loop. Track which rooms have already been decayed this tick.

Before the NPC iteration loop, add:
```go
decayedRooms := make(map[int]bool)
```

Inside the NPC loop, change:
```go
w.decayObjectsInRoom(roomVNum)
```
to:
```go
if !decayedRooms[roomVNum] {
    decayedRooms[roomVNum] = true
    w.decayObjectsInRoom(roomVNum)
}
```

**Cite:** `src/limits.c:525-686` — `point_update()` iterates `object_list` (global) exactly once per call. The loop is NOT nested inside the character loop. Go's approach of decaying per-mob is architecturally different from C's global object list iteration.

**Regression Test:**
- `TestDecayNotDoubledForMultipleMobs`: create a room with 2 mobs and a corpse with timer=2. Run PointUpdate once. Assert corpse timer is 1 (not 0). This proves the dedup works.

---

## Fix 3: DP-1033 — Backstab: multiplier not int-truncated, and no to-hit roll (MED)

This fix has two parts.

### Part A: Backstab multiplier int truncation

**File:** `pkg/combat/formulas.go` — `BackstabMult(level int) float64` (or wherever it's defined)

**Problem:**
C's `backstab_mult()` returns `int`. The expression `(level*.2)+1` produces a float that gets truncated: level 14 → 3.8 → 3, level 19 → 4.8 → 4. Go returns `float64` and the caller does `dam = int(float64(dam) * mult)` — this multiplies by the untruncated float, producing different results at non-multiple-of-5 levels.

- Level 14: C returns 3, Go returns 3.8 → dam is 27% higher
- Level 15: C returns 4, Go returns 4.0 → same (no truncation needed)
- Level 19: C returns 4, Go returns 4.8 → dam is 20% higher

**Fix:**
Change `BackstabMult` to return `int` (matching C), OR truncate the float before use. The simplest fix:

In `BackstabMult`, change:
```go
return (float64(level)*0.2 + 1)
```
to:
```go
return float64(int(float64(level)*0.2 + 1))  // C int truncation
```

This makes the return value match C's `int backstab_mult()`. Update callers if the return type changes.

**Cite:** `src/class.c:719-728` — `int backstab_mult(int level){ return ((level*.2)+1); }` — returns `int`, so the float arithmetic truncates.

### Part B: Backstab missing to-hit roll

**File:** `pkg/game/skill_combat.go:88-112` — `doBackstab()` function

**Problem:**
C's `do_backstab()` calls `hit(ch, vict, SKILL_BACKSTAB)` on success, which runs the full THAC0 d20 to-hit calculation (fight.c:1825-1830). A successful skill roll means "you get to attempt the backstab," NOT "you automatically hit." The to-hit roll can still miss.

Go's backstab calculates damage and returns it directly — no to-hit roll. Every successful backstab lands.

**Fix:**
After the skill success check and before calculating damage, add a THAC0 to-hit roll. The existing `CheckToHit` or equivalent function in the combat package should be used. If the roll misses, return a miss message instead of damage:

```go
// After skill check passes (line ~93), before damage calculation:
if !combat.CheckToHit(ch, victim, combat.AttackBackstab) {
    improveSkill(ch, SkillBackstab)
    return SkillResult{
        Success:       false,
        MessageToCh:   "You try to backstab, but miss!",
        MessageToVict: ActMessage("$n tries to backstab you, but misses!", chPronouns, &victPronouns, ""),
        MessageToRoom: ActMessage("$n tries to backstab $N, but misses.", chPronouns, &victPronouns, ""),
        WaitCh:        1,
    }
}
```

**Cite:** `src/act.offensive.c:224-230` — on a passed skill roll, C calls `hit(ch, vict, SKILL_BACKSTAB)` which runs the full THAC0 hit() path. `src/fight.c:1825-1830` — the to-hit check inside hit() can return 0 (miss).

**Regression Test:**
- `TestBackstabMultTruncation`: assert BackstabMult(14) == 3, BackstabMult(19) == 4 (int values, not float)
- `TestBackstabCanMiss`: if CheckToHit is mockable, test that a low-dex mob can dodge a backstab. If not mockable, add a comment.

---

## Fix 4: DP-1042 — Scavenger mobs ignore CAN_GET_OBJ / cost floor (LOW)

**File:** `pkg/game/mobact.go:176-190` — scavenger block

**Problem:**
Go's scavenger picks the highest-cost item in the room with NO takeable check and NO cost floor. C's scavenger checks `CAN_GET_OBJ(ch, obj)` (which requires ITEM_WEAR_TAKE flag + carry weight/count headroom) AND `GET_OBJ_COST(obj) > max` starting with `max=1` (so items costing 0 or 1 are never picked up).

This means Go scavengers can:
1. Pick up non-takeable items (corpses, furniture, environmental objects)
2. Pick up items with 0 cost (junk, quest items)
3. Pick up items that exceed the mob's carry capacity

**Fix:**
Replace the scavenger block (lines ~176-190) with:

```go
if hasMobFlag(ch, "scavenger") && rand.IntN(11) == 0 {
    items := w.GetItemsInRoom(ch.RoomVNum)
    if len(items) > 0 {
        maxCost := 1  // C: max = 1; items must cost > 1 to be picked up
        var best *Object
        for _, obj := range items {
            if !obj.HasWearFlag(ITEM_WEAR_TAKE) {
                continue
            }
            if obj.GetCost() <= maxCost {
                continue
            }
            // Check carry capacity (weight + count)
            if ch.GetInventoryWeight()+obj.GetWeight() > ch.GetCarryWeight() {
                continue
            }
            if len(ch.GetInventory()) >= ch.GetCarryCount() {
                continue
            }
            best = obj
            maxCost = obj.GetCost()
        }
        if best != nil {
            w.RemoveItemFromRoom(best, ch.RoomVNum)
            ch.AddToInventory(best)
        }
    }
}
```

**Cite:** `src/mobact.c:103-117` — `if (MOB_FLAGGED(ch, MOB_SCAVENGER))` → checks `CAN_GET_OBJ(ch, obj)` (which is `CAN_WEAR(obj, ITEM_WEAR_TAKE) && CAN_CARRY_OBJ(ch,obj) && CAN_SEE_OBJ(ch,obj)`) AND `GET_OBJ_COST(obj) > max` starting with `max=1`. Note: `CAN_SEE_OBJ` is less relevant for mobs (they don't have eyes in the same way), but `ITEM_WEAR_TAKE` and carry limits are critical.

**Regression Test:**
- `TestScavengerSkipsNonTakeable`: create a room with a corpse (no TAKE flag) and a valuable weapon. Scavenger should take the weapon, not the corpse.
- `TestScavengerRespectsCostFloor`: create a room with a 0-cost item and a 100-cost item. Scavenger should ignore the 0-cost item.

---

## Fix 5: DP-1041 — Immortal instakill threshold too low, missing equal-level guard (LOW)

**File:** `pkg/session/combat_cmds.go:21-40` — `doKill()` function

**Problem:**
C gates instakill to `GET_LEVEL(ch) < LVL_IMPL-1` (level 39+). Go gates it to `GetLevel() >= LVL_IMMORT` (level 31+). Every immortal has implementor-grade instakill power. Additionally, C refuses when `GET_LEVEL(vict) == GET_LEVEL(ch)` ("No can do, buddy.."), and sends "$N chops you to pieces!" to the victim. Go sends neither the equal-level check nor the victim message.

**Fix:**
Change the gate from:
```go
if s.player.GetLevel() >= LVL_IMMORT {
```
to:
```go
if s.player.GetLevel() >= LVL_IMPL-1 {
```

Add the equal-level guard after target resolution:
```go
if tgt.Player != nil && tgt.Player.GetLevel() == s.player.GetLevel() {
    s.Send("No can do, buddy..\r\n")
    return nil
}
if tgt.Mob != nil && tgt.Mob.GetLevel() == s.player.GetLevel() {
    s.Send("No can do, buddy..\r\n")
    return nil
}
```

Add the victim message (send to the victim before they die):
```go
case tgt.Player != nil:
    tgt.Player.SendMessage(fmt.Sprintf("%s chops you to pieces!\r\n", s.player.GetName()))
    s.manager.world.HandleDeath(tgt.Player, s.player, 0)
    s.Send(fmt.Sprintf("You chop %s to pieces! Ah! The blood!", tgt.Player.Name))
```

**Cite:** `src/act.offensive.c:138-154` — `if ((GET_LEVEL(ch) < LVL_IMPL-1) || IS_NPC(ch))` gates to level 39+. Line 149: `else if (GET_LEVEL(vict) == GET_LEVEL(ch)) send_to_char("No can do, buddy.. \r\n", ch);`. Line 152-154: act() sends "$N chops you to pieces!" to victim.

Note: Go's LVL_IMPL=40, LVL_IMMORT=31 (from `pkg/game/limits.go:21-23`). C matches: LVL_IMPL=40, LVL_IMMORT=31 (src/structs.h:610-620). So `LVL_IMPL-1 = 39` is correct in both.

**Regression Test:**
- `TestInstakillRequiresImplLevel`: level 35 immortal tries `kill` on a mob → should NOT instakill (should try hit() instead). Level 39 → should instakill.
- `TestInstakillSameLevelBlocked`: level 40 immortal tries `kill` on a level 40 mob → "No can do, buddy.."

---

## Execution Order

1. Fix 1A (tick speed) — isolated
2. Fix 1B (mob AI) — verify AITick vs OnMobileActivity overlap first
3. Fix 2 (decay dedup) — isolated
4. Fix 3A (backstab truncation) — isolated
5. Fix 3B (backstab to-hit) — depends on 3A being clean
6. Fix 4 (scavenger) — isolated
7. Fix 5 (instakill) — isolated
8. Run build gate

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/game/world.go pkg/game/ai.go pkg/game/limits_condition.go pkg/combat/formulas.go pkg/game/skill_combat.go pkg/game/mobact.go pkg/session/combat_cmds.go
git add -u  # pick up any test files
git commit -m "fix: tick speed, decay dedup, backstab fidelity, scavenger checks, instakill gate (DP-1035, DP-1036, DP-1033, DP-1042, DP-1041)"
git push -u origin fix/batch2-small-fixes
gh pr create --title "fix: tick speed, decay dedup, backstab fidelity, scavenger checks, instakill gate (DP-1035, DP-1036, DP-1033, DP-1042, DP-1041)" --body "Fixes DP-1035, DP-1036, DP-1033, DP-1042, DP-1041. See docs/briefs/BRIEF-2026-07-11-batch2-small-fixes.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.
