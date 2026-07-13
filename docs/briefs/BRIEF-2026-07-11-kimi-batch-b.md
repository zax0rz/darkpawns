# Brief: Kimi Batch B — 2026-07-11

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-1030 — PK bookkeeping missing on live death path (MED)

**File:** `pkg/game/death.go` — `handlePlayerDeath()` (line ~390-530)

**Problem:**
When a player kills another player, the C source at `src/fight.c:1671-1689` runs PK-specific bookkeeping:
1. PK mudlog: `"(PK) %s killed by %s at %s"`
2. Flag killer as OUTLAW if victim wasn't one: `SET_BIT_AR(PLR_FLAGS(ch), PLR_OUTLAW)`
3. Increment killer's PK count: `GET_PKS(ch)++`
4. Increment victim's death count: `GET_DEATHS(victim)++`
5. Record victim's last death time: `GET_LAST_DEATH(victim) = (long)time(0)`

None of this happens in the Go `handlePlayerDeath`. The function already has `killerName string` and `victim (*Player)` available. The PK log and counter increments just need to be wired in.

**C source (fight.c:1671-1689):**
```c
if (!IS_NPC(victim)) {
  if(!IS_NPC(ch) && (ch != victim)) { /* Pkill */
    sprintf(buf2, "(PK) %s killed by %s at %s", GET_NAME(victim),
            GET_NAME(ch), world[victim->in_room].name);
    if(!PLR_FLAGGED(victim, PLR_OUTLAW))
      SET_BIT_AR(PLR_FLAGS(ch), PLR_OUTLAW);
  } else {
    sprintf(buf2, "%s killed by %s at %s", GET_NAME(victim), GET_NAME(ch),
            world[victim->in_room].name);
  }
  mudlog(buf2, BRF, LVL_IMMORT, TRUE);
  if (ch != victim)
    GET_PKS(ch)++;
  GET_DEATHS(victim)++;
  GET_LAST_DEATH(victim) = (long)time(0);
}
```

**Fix:**
In `handlePlayerDeath`, after the idempotent death guard (line ~418) and before the EXP loss section, add the PK bookkeeping block. The function already knows:
- `victim` is a `*Player` (line ~408)
- `killerName` is a `string` (parameter)
- `roomVNum` is the death room (line ~404)

Insert after the `if player.GetHP() > 0 { return }` guard (line ~418):
```c
// PK bookkeeping — fight.c:1671-1689
// Only fires for player-vs-player kills (killerName != victim's own name)
if killerName != "" && killerName != player.GetName() {
    // Find the killer player to check their flags
    if killer := w.GetPlayerByName(killerName); killer != nil && !killer.IsNPC() {
        slog.Info("(PK) "+fmt.Sprintf("%s killed by %s at room %d", player.GetName(), killerName, roomVNum),
            "victim", player.GetName(), "killer", killerName, "room", roomVNum)
        // Flag killer as outlaw if victim wasn't one
        if !killer.HasPlrFlag(PlrOutlaw) {
            killer.SetPlrFlag(PlrOutlaw, true)
        }
        killer.PKs++
    } else {
        slog.Info(fmt.Sprintf("%s killed by %s at room %d", player.GetName(), killerName, roomVNum),
            "victim", player.GetName(), "killer", killerName, "room", roomVNum)
    }
    player.Deaths++
    player.LastDeath = time.Now().Unix()
}
```

**Important:**
- You need to find or define `PlrOutlaw` — the bit number for `PLR_OUTLAW` from `src/structs.h`. Check `pkg/game/constants.go` for a `PlrBitNames` array or similar. If it doesn't exist, define it as a constant (the C value is typically bit 4 or similar — check `src/structs.h`).
- `GetPlayerByName` should exist on World — search for it. If not, iterate `w.players` to find the killer.
- `HasPlrFlag` should exist on Player — if not, use `p.Flags & (1 << uint(bit)) != 0`.
- The non-PK branch (mob kills player) just needs the mudlog, which is already handled elsewhere or can be added simply.

**Cite:** `src/fight.c:1671-1689` — PK bookkeeping block in die_with_killer()

**Regression Test:**
Add a test in `pkg/game/death_test.go` or `pkg/game/glm_batch_a_test.go`:
- Create two players, put them in the same room
- Kill player B via player A's combat path
- Assert `playerB.Deaths == 1`, `playerA.PKs == 1`, `playerA.HasPlrFlag(PlrOutlaw) == true`

---

## Fix 2: DP-1027 — Death traps don't kill (HIGH)

**File:** `pkg/game/world.go` — `MovePlayer()` (line ~880-965)

**Problem:**
When a player moves into a room with the ROOM_DEATH flag, the C source at `src/act.movement.c:289-302` kills them instantly (unless immortal). The Go `MovePlayer` has no ROOM_DEATH check at all. There IS a dead code path in `act_movement.go:339-343` that does `ch.TakeDamage(ch.GetHP()+1)` with a "You have entered a death trap!" message, but this path is unreachable from the live movement code.

**C source (act.movement.c:289-302):**
```c
if ( (ROOM_FLAGGED(ch->in_room, ROOM_DEATH)) &&
  (GET_LEVEL(ch) < LVL_IMMORT || IS_NPC(ch))  )
{
  log_death_trap(ch);
  death_cry(ch);
  extract_char(ch);
  if (mount) {
    log_death_trap(mount);
    death_cry(mount);
    extract_char(mount);
  }
  return 0;
}
```

**Fix:**
In `MovePlayer`, after the successful move (after `p.RoomVNum = newRoom.VNum` at line ~948, but before `w.mu.Unlock()`), add a ROOM_DEATH check:

```go
// Death trap check — act.movement.c:289-302
// ROOM_DEATH is bit 1 in RoomBitNames (constants.go:135)
if roomHasFlagBit(newRoom.Flags, 1) && p.Level < LVL_IMMORT {
    slog.Info("death trap", "player", p.GetName(), "room", newRoom.VNum)
    // Kill the player — TakeDamage with enough to drop to 0
    p.TakeDamage(p.GetHP() + 1)
    p.SendMessage("You have entered a death trap!\r\n")
    // Broadcast to room
    w.roomMessage(newRoom.VNum, fmt.Sprintf("The sound of a death cry is heard as %s enters the room!\r\n", p.GetName()))
}
```

**Important:**
- `LVL_IMMORT` — find the constant in `pkg/game/limits.go` or similar. C has `LVL_IMMORT 31`.
- `roomHasFlagBit` is in `pkg/game/room_flags.go:38` — it takes `flags []string` and `flagBit int`.
- The ROOM_DEATH flag is `RoomBitNames[1] = "DEATH"` (constants.go:135), so bit number 1.
- The C code checks `GET_LEVEL(ch) < LVL_IMMORT || IS_NPC(ch)` — mobs in death traps also die. For now, just handle players (mobs in DT rooms is a separate concern).
- The `TakeDamage(p.GetHP() + 1)` pattern is what the dead path in act_movement.go already uses — it's the standard way to kill a player without going through combat.

**Cite:** `src/act.movement.c:289-302` — ROOM_DEATH check in do_gen_door/movement

**Regression Test:**
Add a test in `pkg/game/death_test.go`:
- Create a room with ROOM_DEATH flag
- Create a player in an adjacent room
- Move player into the death trap room
- Assert player HP is 0 (or dead)

---

## Fix 3: DP-1028 — Mob spell affects never expire (HIGH)

**File:** `pkg/game/affect_update.go` — `AffectUpdate()` (line ~30-80)

**Problem:**
The C `affect_update()` at `src/magic.c:431-457` iterates `character_list` — which includes BOTH players AND NPCs. The Go `AffectUpdate` only iterates `w.players`. Mob affects are stored in `CustomData["affect_<spellID>"]` (mob.go:974-980) with a comment "the affect tick system will handle duration-based removal", but no ticker ever reads those keys. This means any duration debuff on a mob (blind, curse, poison, sleep, slow) lasts forever.

**C source (magic.c:431-457):**
```c
void affect_update(void) {
  static struct master_affected_type *af, *next;
  static struct char_data *i;
  for (i = character_list; i; i = i->next)
    for (af = i->affected; af; af = next) {
      next = af->next;
      if (af->duration >= 1)
        af->duration--;
      else if (af->duration == -1)
        af->duration = -1;    /* permanent */
      else {
        /* expires */
        if (*spell_wear_off_msg[af->type])
          send_to_char(spell_wear_off_msg[af->type], i);
        affect_remove(i, af);
      }
    }
}
```

**Fix:**
Extend `AffectUpdate` to also iterate active mobs. After the player loop, add a mob loop:

```go
// Mob affect expiration — magic.c:431-457
// C iterates character_list which includes both players and NPCs.
mobs := w.GetAllMobs()
for _, mob := range mobs {
    if !mob.IsAlive() {
        continue
    }
    mob.mu.Lock()
    // Collect expired affect keys
    var expiredKeys []string
    for key, val := range mob.CustomData {
        if !strings.HasPrefix(key, "affect_") {
            continue
        }
        aff, ok := val.(*engine.Affect)
        if !ok {
            continue
        }
        if aff.Duration == -1 {
            continue // permanent
        }
        if aff.Duration >= 1 {
            aff.Duration--
            continue
        }
        // Duration == 0 — expires this tick
        expiredKeys = append(expiredKeys, key)
        // Send wear-off message to the room (mobs don't have a SendMessage)
        if msg := SpellWearOffMsg(aff.SpellID); msg != "" {
            w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s %s\r\n", mob.GetShortDesc(), msg))
        }
    }
    // Remove expired affects
    for _, key := range expiredKeys {
        delete(mob.CustomData, key)
        // Parse spell number from key ("affect_<N>")
        spellNum := 0
        if _, err := fmt.Sscanf(key, "affect_%d", &spellNum); err == nil {
            mob.RemoveAffectBySpell(spellNum)
        }
    }
    mob.mu.Unlock()
}
```

**Important:**
- `GetAllMobs()` is at `pkg/game/world.go:1188` — returns `[]*MobInstance`
- `RemoveAffectBySpell(spellNum int)` exists at `pkg/game/mob.go:993` — clears the CustomData key AND removes AFF_* bitmask bits
- `SpellWearOffMsg(spellID int)` exists in the spells package — returns the wear-off string
- The mob affect system uses `CustomData["affect_<spellID>"]` keys (mob.go:974-980). The pattern is `fmt.Sprintf("affect_%d", aff.SpellID)`
- Mobs don't have `SendMessage` — broadcast the wear-off message to the room via `w.roomMessage`
- Lock `mob.mu` before accessing `CustomData` — other goroutines may read mob state
- The `engine.Affect` struct has `Duration int` and `SpellID int` fields

**Cite:** `src/magic.c:431-457` — affect_update() iterates character_list (players + NPCs)

**Regression Test:**
Add a test in `pkg/game/glm_batch_a_test.go` or a new `pkg/game/affect_update_test.go`:
- Create a mob, apply a poison affect with Duration=1 via CustomData
- Run one AffectUpdate tick
- Assert the affect key is removed from CustomData
- Assert the AFF_* bitmask bit is cleared

---

## Execution Order

1. **Fix 1 (DP-1030)** — PK bookkeeping. Single function edit in death.go.
2. **Fix 2 (DP-1027)** — Death traps. Single function edit in world.go.
3. **Fix 3 (DP-1028)** — Mob affects. Single function edit in affect_update.go.

All three are independent — no file overlap. Can be implemented in any order.

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add -A
git commit -m "fix: PK bookkeeping, death trap kills, mob affect expiration (DP-1030, DP-1027, DP-1028)"
git push -u origin fix/kimi-batch-b
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-1030: Add comment "Fixed — PK bookkeeping wired into handlePlayerDeath (death.go)", commit hash, move to Done
- DP-1027: Add comment "Fixed — ROOM_DEATH check added to MovePlayer (world.go)", commit hash, move to Done
- DP-1028: Add comment "Fixed — mob affects expire in AffectUpdate (affect_update.go)", commit hash, move to Done
