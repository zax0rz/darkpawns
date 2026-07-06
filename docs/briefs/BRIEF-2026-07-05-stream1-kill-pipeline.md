# Brief: Stream 1 — Kill Pipeline Correctness — 2026-07-05

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Agent:** Kimi k2.6/k2.7 (primary) or Gemini 3.5-flash (fallback)

---

## Fix 1: DP-942 — Skill/spell kills award zero XP, no kill counter, no events (Urgent)

**File:** `pkg/game/damage_stubs.go` — `DoSpellDamage()` (line 41), `doDamage()` (line 77)

**Problem:**
Both `DoSpellDamage` and `doDamage` route mob deaths through `w.handleMobDeath(v, nil, 303)` — killer is explicitly `nil`, attackType is hardcoded `303` (TypeSlash). This bypasses `HandleDeath` (death.go:112-168) which handles:

- XP award via `AwardMobKillXP` (called from `HandleDeath` → mob kill path)
- Kill counter + `counter_procs` milestones
- `MobKilledEvent` published to event bus
- Memory hooks (narrative memory writes via `fireMobKill`)
- AutoGold/AutoLoot killer logic
- XP split across group members

The result: **every mob killed by a skill/spell (backstab, bash, kick, circle, charge, damage spells, etc.) awards zero XP, zero gold, increments no kill counter, fires no events.** This is the primary gameplay bug in the codebase.

For player victims, `DoSpellDamage`/`doDamage` call `w.rawKill(v, 303)` which calls `w.HandleDeath(victim, nil, attackType)` — killer is `nil` there too, so the player death path runs with no killer name, but the death side-effects (EXP loss, CON roll, corpse) still happen. This is less broken than mob kills but still loses the killer name.

**Fix:**
1. In `DoSpellDamage` (line 60): replace `w.handleMobDeath(v, nil, 303)` with `w.HandleDeath(v, attackerAsCombatant, attackType)`.
2. In `doDamage` (line 104): same replacement.
3. In `DoSpellDamage` (line 53): replace `w.rawKill(v, 303)` with `w.HandleDeath(v, attackerAsCombatant, attackType)`.
4. In `doDamage` (line 97): same replacement.
5. The `attacker` parameter is `interface{}` — it may be `*Player`, `*MobInstance`, or nil. Convert to `combat.Combatant` with a type assertion:
   ```go
   var killer combat.Combatant
   if a, ok := attacker.(combat.Combatant); ok {
       killer = a
   }
   ```
6. For the attackType, derive from the skill name. The `skill` parameter already holds the skill/spell name. Use a map lookup or switch to resolve to the correct TYPE_* constant (e.g., backstab→301, kick→302, etc.). **Falling back to 303 is acceptable** if a clean mapping isn't trivial — the important thing is passing the real killer. Create a helper `skillToAttackType(skill string) int` in `damage_stubs.go`.
7. Fix the misleading DP-901 comment at `pkg/game/skill_commands.go:1560` that claims XP is handled.

**Cite:** C source — `fight.c:die_with_killer()` → `group_gain(ch, victim)` (lines ~1638+). In C, XP flows regardless of what dealt the killing blow. The `damage()` function always calls `die_with_killer(ch, killer)` with the real killer. `fight.c` `raw_kill()` and `die()` always receive the attacker.

**Regression Test:** `pkg/game/damage_stubs_test.go`
- `TestDoDamageAwardsXP`: create a World with a mob, a player attacker, call `doDamage(player, mob, 100, "backstab")` where 100 damage kills the mob. Assert player gained XP (`player.GetExp() > 0`) and kill counter incremented.
- `TestDoSpellDamageAwardsXP`: same for `DoSpellDamage`.
- `TestDoDamagePlayerDeathHasKillerName`: call `doDamage(mob, player, lethal)` and verify `handlePlayerDeath` received a non-empty `killerName`.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-943 — Player death not idempotent — concurrent kills double-punish (Urgent)

**File:** `pkg/game/death.go` — `handlePlayerDeath()` (line 359)

**Problem:**
`handleMobDeath` has a proper guard: `SetAlive(false)` + `delete(w.activeMobs, id)` under `w.mu.Lock()` (lines 226-241). Second caller finds nothing in `activeMobs` and returns.

`handlePlayerDeath` has **no guard**. Two goroutines (combat engine tick + skill/spell kill session goroutine) can both drop the same player to ≤0 HP and both run the full death path:
- Double EXP loss (the already-reduced EXP is divided again)
- Double CON roll (two independent random rolls)
- Two corpses (second is empty since inventory was already cleared)
- Double respawn messages
- Double gold zeroing

The `Player` struct has no `alive` or `dying` field (unlike `MobInstance` which has `alive atomic.Bool` at mob.go:26). There is no check-and-set at the top of `handlePlayerDeath`.

**Fix:**
1. Add a `dying atomic.Bool` field to the `Player` struct (in `pkg/game/player.go`). Initialize to `false` in `NewPlayer`.
2. At the top of `handlePlayerDeath` (death.go:359), before any side-effects:
   ```go
   if !player.dying.CompareAndSwap(false, true) {
       // Already dying — skip
       return
   }
   ```
3. Reset `dying` to `false` on respawn. The respawn happens at lines 462-466. Add:
   ```go
   player.dying.Store(false)
   ```
   after `player.Heal(9999)` at line 465.
4. Add accessor method `IsDying() bool` on Player for any future callers that need to check.
5. Add a test that exercises this guard (see regression tests below).

**Cite:** No direct C equivalent — C's single-threaded model prevents this. In C, `die_with_killer()` can only be called once per character because there's only one thread of execution. The Go port's concurrency creates this class of bug.

**Deviation:** This is a Go-only addition (atomic guard). Not a C fidelity issue — it's a concurrency correctness fix required by Go's multi-goroutine architecture.

**Regression Test:** `pkg/game/death_test.go`
- `TestHandlePlayerDeathIdempotent`: call `handlePlayerDeath` twice for the same player in rapid succession (no delay needed — the atomic guard is synchronous). Assert EXP was deducted exactly once, CON was reduced at most once, exactly one corpse was created.
- `TestHandlePlayerDeathDyingReset`: verify that after `handlePlayerDeath` completes (respawn), `player.IsDying()` returns `false`.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-944 — specDump swallows drop command — award can never fire (Urgent)

**File:** `pkg/game/spec_procs.go` — `specDump()` (line 155)

**Problem:**
C's `spec_dump` (src/spec_procs.c:254-293) does:
1. Pre-clean any items already on the room floor (extract all room contents)
2. If not "drop", return 0
3. **Call `do_drop(ch, argument, cmd, 0)` directly** — this moves items from player inventory to room
4. Re-scan room contents, value them (`MAX(1, MIN(10, cost/10))` per item), extract them, award XP or gold

Go's `specDump` (spec_procs.go:155-174) does:
1. Pre-clean room via `roomCleanup` (line 157)
2. If not "drop", return false (correct)
3. **Never calls the actual drop** — item stays in player inventory
4. Calls `roomCleanup` again (line 163) — room is empty, so `value` is always 0
5. Returns `true` (line 173) — **consumes the command**, so the normal `cmdDrop` handler never runs

Two bugs:
- The `drop` command is swallowed in dump rooms — players can never drop items
- The award can never fire (value is always 0)

**Fix:**
Replace the broken drop path with a call to the real drop logic. `w.doDrop(ch, me, cmd, arg)` exists at `pkg/game/item_transfer.go:241` and performs the actual item transfer + messages.

```go
func specDump(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    roomVNum := ch.GetRoomVNum()
    _ = w.roomCleanup(roomVNum)

    if cmd != "drop" {
        return false
    }

    // Perform the actual drop (mirrors C's do_drop(ch, argument, cmd, 0))
    w.doDrop(ch, me, cmd, arg)

    // Value and remove the dropped items
    value := w.roomCleanup(roomVNum)
    if value > 0 {
        sendToChar(ch, "You are awarded for outstanding performance.")
        w.roomMessage(roomVNum, ch.GetName()+" has been awarded by the gods!")
        if ch.GetLevel() < 3 {
            ch.Exp += value
        } else {
            ch.Gold += value
        }
    }
    return true
}
```

**Important:** The `arg` parameter passed to `specDump` is the full argument string after the command (e.g., if player typed `drop sword`, `cmd="drop"` and `arg="sword"`). Verify this matches what `w.doDrop` expects — it receives `(ch, me, cmd, arg)` and parses `arg` internally. If `arg` is empty, `doDrop` prints "Drop what?" which is correct behavior.

**Cite:** C source — `src/spec_procs.c:254-293` (`spec_dump`). C calls `do_drop(ch, argument, cmd, 0)` directly. The valuation formula is identical between C and Go: `MAX(1, MIN(10, cost / 10))`.

**Regression Test:** `pkg/game/spec_procs_test.go`
- `TestSpecDumpPerformDrop`: create a player with an item in inventory, place them in a dump room, call `specDump(w, player, nil, "drop", "sword")`. Assert the item is no longer in the player's inventory and that `player.Gold > 0` (award fired).
- `TestSpecDumpNonDropPassThrough`: call `specDump(w, player, nil, "look", "")`. Assert it returns `false`.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 4: DP-945 — specSnake victim messaging loses TO_VICT form (Low)

**File:** `pkg/game/spec_procs.go` — `specSnake()` (line 177)

**Problem:**
C's snake spec (src/spec_procs.c) sends:
- TO_CHAR: (none — mob doesn't send to itself)
- TO_VICT: `act("$n bites you!", FALSE, ch, 0, victim, TO_VICT)`
- TO_ROOM: `act("$n bites $N!", FALSE, ch, 0, victim, TO_ROOM)`

Go's `specSnake` (spec_procs.go:188) uses `w.roomMessage(me.RoomVNum, me.GetName()+" bites "+melee.GetName()+"!")` which sends the **exact same third-person string to everyone in the room, including the victim**. The victim sees "Snake bites VictimName!" instead of "Snake bites you!".

The `roomMessage` helper (spec_procs.go:48-53) broadcasts one string to all players with no audience distinction.

**Fix:**
Replace the `roomMessage` call with audience-aware messaging. The `ActMessage` helper exists at `pkg/game/skills.go:297-313` but requires `PronounData` structs. For spec procs, a simpler approach:

1. Add a helper `sendToVictim` or use the existing `sendToChar` + modify `roomMessage` to exclude a specific player. The simplest fix:

```go
// In specSnake, replace line 188:
// Before:
w.roomMessage(me.RoomVNum, me.GetName()+" bites "+melee.GetName()+"!")
// After:
w.actMessage(me.RoomVNum, me, melee,
    "",  // TO_CHAR (mob doesn't need a message)
    me.GetName()+" bites you!",           // TO_VICT
    me.GetName()+" bites "+melee.GetName()+"!", // TO_ROOM
)
```

2. Implement `actMessage` on World (or expand the existing helpers) to send per-audience messages. This pattern likely repeats in other spec procs, so the helper should be reusable. A minimal implementation:

```go
func (w *World) actMessage(roomVNum int, actor *MobInstance, victim combat.Combatant, toChar, toVict, toRoom string) {
    players := w.GetPlayersInRoom(roomVNum)
    for _, p := range players {
        if victim != nil && p.GetName() == victim.GetName() {
            if toVict != "" {
                p.SendMessage(toVict + "\r\n")
            }
        } else {
            if toRoom != "" {
                p.SendMessage(toRoom + "\r\n")
            }
        }
    }
}
```

3. If the victim is a `*MobInstance` (not a player), there's no TO_VICT target — just send toRoom to all players. The current code only calls this when `melee` (a `*MobInstance`) is non-nil. For mob-vs-mob combat, there are no players to send TO_VICT to. **Focus on the player-in-room case** — check if `melee` is a `*Player` and send TO_VICT accordingly, or if `melee` is a mob, check if any player in the room has that mob as their fight target.

**Simplify:** The most impactful fix is to just handle the `*Player` victim case. Check `mobMeleeTarget` — if it returns a player, send TO_VICT to that player. The current `mobMeleeTarget` function (spec_procs.go:92-110) returns `*MobInstance`, so if the melee target is a player, this won't be hit. **Verify who `mobMeleeTarget` can return.** If it always returns mob targets, this fix is lower priority — it only matters when mobs fight players and the player is the melee target of the snake mob.

**Cite:** C source — `src/spec_procs.c` snake spec. Uses `act()` with `TO_VICT` and `TO_ROOM` separately. The `$n` / `$N` pronoun system routes different strings to different audiences.

**Regression Test:** `pkg/game/spec_procs_test.go`
- `TestSpecSnakeVictimMessage`: if `mobMeleeTarget` can return a player, verify the victim player receives a second-person message ("bites you!") and other room players receive third-person ("bites VictimName!").

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. **Fix 2 (DP-943)** — Player death idempotency. Smallest change, adds atomic guard. No dependencies on other fixes.
2. **Fix 1 (DP-942)** — Route through `HandleDeath`. Depends on Fix 2 being in place so the HandleDeath path has the idempotency guard for player victims.
3. **Fix 3 (DP-944)** — specDump. Independent of Fix 1/2. Pure spec proc fix.
4. **Fix 4 (DP-945)** — specSnake messaging. Independent. Can be deferred if `mobMeleeTarget` only returns mob targets.

**Suggested batch order:** 2 → 1 → 3 → 4

---

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace/darkpawns_repo
git checkout -b fix/stream1-kill-pipeline
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: kill pipeline correctness — XP awards, death idempotency, specDump, specSnake (DP-942, DP-943, DP-944, DP-945)"
git push -u origin fix/stream1-kill-pipeline
```

Wait for review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-942: Add comment "Fixed — DoSpellDamage/doDamage now route through HandleDeath with real killer", commit <hash>, move to Done
- DP-943: Add comment "Fixed — added dying atomic.Bool guard on Player for idempotent death", commit <hash>, move to Done
- DP-944: Add comment "Fixed — specDump now calls doDrop before valuing room contents", commit <hash>, move to Done
- DP-945: Add comment "Fixed — specSnake sends TO_VICT message to victim", commit <hash>, move to Done
