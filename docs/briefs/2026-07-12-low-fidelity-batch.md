# DP-1062: LOW Batch — Decay, Instakill, PK Counters, Message Nits, Scavenger, XP Nits, Wear-Off, Parry Vestiges

**Target:** Multiple files (see each item)
**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/low-fidelity-batch`
**After fixing:** `go build ./... && go vet ./... && go test ./...`

This is a batch of 8 small fidelity fixes. Do them all in one PR. They're independent — different files, different systems.

---

## 1. Decay Coverage — Objects in rooms without mobs never decay

**Files:** `pkg/game/limits_condition.go`
**Current:** `decayObjectsInRoom()` is called only for rooms that have a mob in them (inside the mob loop at line ~291, guarded by `decayedRooms` map). Corpses in cleared rooms (no mobs left) and corpses carried by players never decay.
**C source:** `point_update()` in `src/limits.c:529+` iterates the **global object list**, checking each object's timer independently of what room has mobs.
**Fix:** After the character/mob loop in `PointUpdate()`, add a second pass that iterates all rooms (not just mob rooms) and calls `decayObjectsInRoom()`. OR iterate the global object list like C. The `decayedRooms` dedup map is still good — just expand its scope.
**Note:** Carried objects (in inventory) should NOT decay — only room objects. C's global iteration skips objects with `obj->carried_by` set.

---

## 2. Instakill Routing — Go awards XP for immortal instakill, C uses raw_kill

**File:** `pkg/game/death.go`
**Current:** The `kill` command routes through `HandleDeath(victim, killer)` even for immortal instakills (equal or lower level victim). This means the killer gets XP, PK counters, OUTLAW flag, and the victim loses exp/37.
**C source:** `do_kill()` in `src/act.offensive.c:152-160` calls `raw_kill(ch, victim)` for instakills — no award, no bookkeeping, no penalty. `raw_kill` just extracts the character and resets them.
**Fix:** In the command handler for `kill` (wherever the immortal check is), when the killer is an immortal and the victim's level is <= killer's level, call a `rawKill()` path instead of `HandleDeath()`. The `rawKill` path should: extract character from room, reset HP to max, move to start room, send "You have been killed!" message. No XP, no counters, no alignment shift.
**Reference:** Check how `raw_kill` is already implemented in `pkg/scripting/engine.go:1468-1469` (lua binding). There may already be a `rawKill` function you can call.

---

## 3. PK Kill Counters — Kills/Deaths not tracked for player-vs-player kills

**File:** `pkg/game/death.go`
**Current:** `GET_KILLS(ch)++` / `counter_procs(ch)` only fires in the mob branch (inside `HandleDeath` when `killer != nil`). When a player kills another player, the kill counter doesn't increment. Also `GET_DEATHS(victim)++` is skipped when killer is self or empty.
**C source:** `src/fight.c:1689-1690` — `GET_KILLS(ch)++` and `counter_procs(ch)` run **unconditionally** for ALL kills including PK (outside the NPC-victim branch). `GET_DEATHS(victim)++` is also unconditional.
**Fix:** Move the `Kills++` block to fire for ALL kills, not just mob kills. Move `Deaths++` on the victim to fire unconditionally (not guarded by `killer != nil`). Watch for the existing `kp, ok := w.GetPlayer(killerName)` guard — if killer is a player in a PK, this should still work since killerName is set.

---

## 4. Damage Message Variants — Drop invented flavor, use only C-verbatim text

**File:** `pkg/combat/fight_core.go` (around line 596-625)
**Current:** Each damage tier has 3-4 variants. One per tier matches C's `dam_weapons[]` verbatim. The others are invented "flavor" that C doesn't have. C is deterministic — same message every time for a given tier.
**Fix:** Keep only the C-verbatim variant in each tier. Remove the additional flavor variants. The comment at line 624 already notes which variant is verbatim — keep that one, delete the rest.
**Reference:** C source is `src/constants.c` — the `dam_weapons[]` array. Each tier has exactly one message string.

---

## 5. Scavenger Mob Messages — Pickup is silent in Go, C announces to room

**File:** `pkg/game/mobact.go` (around line 215)
**Current:** When a scavenger mob picks up an item, no message is sent to the room.
**C source:** `src/mobact.c:115` — `act("$n gets $p.", FALSE, ch, obj, 0, TO_ROOM)`. The room sees "A beggar gets a long sword."
**Fix:** After the scavenger picks up the item (wherever the `obj.MoveTo(mob)` or equivalent call is), add a room broadcast: `m.SendToRoom(fmt.Sprintf("%s gets %s.\r\n", m.GetName(), obj.GetShortDesc()))`. Use whatever room-send method the mob system already uses (check `mobact.go` for existing `SendToRoom` or `BroadcastRoom` patterns).
**Note:** Also fix the carry-weight clamp issue here — scavengers use `mobMaxCarryWeight(m)` at `mobact.go:62` which also ignores StrAdd. This is related to DP-1056 but can be fixed independently with the same pattern.

---

## 6. XP Share Nits — Four small formula/text discrepancies

**File:** `pkg/game/party.go` (around lines 250-265)
**Fix these four:**

a. **Group base truncation:** Line ~252: `base -= base / 100` (integer div). C uses `base -= base * .01` which truncates differently for some values (e.g., 250 → Go 248 / C 247). Change to: `base -= int(float64(base) * 0.01)` to match C's double-truncation behavior. Actually, C's `base * .01` is `int(base * 0.01)` which truncates toward zero. Use that.

b. **Solo 1-XP message:** Line ~229 says `"measly little"` for solo kills of 1 XP. C's solo message is `"one lousy experience point."` (from `src/fight.c`). The `"measly little"` string is actually C's **group** share message. Fix solo to say `"one lousy experience point."`.

c. **Zero-exp mobs:** If `base` computes to 0, the current `if base < 1 { base = 1 }` guard saves it. But verify the message still fires — C always awards `MAX(1, ...)` and sends the corresponding message. This may already be correct; just verify.

d. **Solo player in group:** A grouped player alone in the room should get the full solo XP minus the 2-point slack (C's `is_in_group` checks same-room). Verify Go's group detection does the same — if it counts anyone in the party regardless of room, that's wrong.

---

## 7. Mob Wear-Off Broadcast — Go shows wear-off to room, C sends to mob (invisible)

**File:** `pkg/spells/affect_spells.go` (around `MagUnaffects` or wherever spell effects expire)
**Current:** When a spell wears off a mob, Go broadcasts the message to the room. Players see "The haste spell wears off on a cityguard."
**C source:** `src/magic.c:449-452` — `send_to_char` to the mob itself. Since mobs don't have a terminal, nothing is displayed to anyone. C's wear-off on mobs is SILENT.
**Fix:** When the target is a mob (`target.IsNPC()`), suppress the wear-off message entirely. Only send it when the target is a player.

---

## 8. Parry Vestiges — Dead code cleanup

**File:** `pkg/game/skill_c10_combat.go`
**Current:** `DoParry()` (line 314), `CheckParry()` (line 328), and `IsParrying()` still exist but parry was never fully wired into melee (it's a stance toggle with no combat integration). There's also a dead `awake` guard on parry that C doesn't have.
**Fix:** Delete `DoParry`, `CheckParry`, `IsParrying` and the parry stance field from the Player struct. Remove any references to these functions in command dispatch or skill registration. This is pure dead code removal — no behavior change since nothing calls them in combat.

---

## Commit message

`fix: LOW fidelity batch — decay, instakill, PK counters, messages, scavenger, XP nits, wear-off, parry cleanup (DP-1062)`
