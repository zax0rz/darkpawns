# Fidelity Brief 04: Movement System

**Date:** 2026-05-27
**Priority:** HIGH — player moves through the world constantly
**C source:** `src/act.movement.c` (951 lines)
**Go source:** `pkg/game/act_movement.go` (986 lines), `pkg/session/movement_cmds.go` (429 lines)

---

## Scope

Movement is the most frequent player action. The C source handles:
1. Directional movement (n/s/e/w/up/down)
2. Door manipulation (open/close/lock/unlock/pick)
3. Room flags (DEATH, DARK, DEATH, !MAGIC, etc.)
4. Zone transitions
5. Boat/water movement
6. Flying movement
7. Special exits
8. Following (group members follow leader)

This brief covers:

1. **`do_move()` / `perform_move()` / `do_simple_move()`** — the three-layer movement stack
2. **`do_gen_door()`** — door commands (open/close/lock/unlock/pick)
3. **Room flags** — movement restrictions
4. **Follower movement** — group members following
5. **Movement messages** — leave/enter messages, custom zone messages
6. **Movement costs** — move point deduction

---

## What to Verify

### 1. Movement Stack

**C source** (act.movement.c):
```
do_move(ch, dir)          — ACMD entry point, validates input
  perform_move(ch, dir)   — checks special procs, calls do_simple_move
    do_simple_move(ch, dir) — actual room transition, costs move points
```

**Check:** Does the Go code follow this same three-layer structure? Or is it flattened?

### 2. `do_simple_move()` — The Core

**C source** (act.movement.c:95):
```c
int do_simple_move(struct char_data *ch, int dir, int need_specials_check)
{
  // 1. Check if room allows movement
  // 2. Check for closed/locked doors
  // 3. Check for boat (if water sector)
  // 4. Check for flying (if needed)
  // 5. Calculate move cost:
  //    - Indoor/roads: 1 move
  //    - Water: 1 move + swim check
  //    - Desert/mountains: 2 moves
  //    - Underwater: 3 moves
  // 6. Deduct move points
  // 7. Remove AFF_SNEAK/AFF_HIDE if moving
  // 8. Execute act() messages for leave/enter
  // 9. Move player to new room
  // 10. Trigger zone entry procs
  // 11. Execute look() for new room
}
```

**Check each step:**
- Move cost calculation per sector type
- Swim check for water sectors
- AFF_SNEAK/AFF_HIDE removal on movement
- Message sending (leave old room, enter new room)
- Zone entry triggers

### 3. Sector Types and Move Cost

**C source** (constants.c, act.movement.c):
```
SECT_INSIDE:     1 move
SECT_CITY:       1 move
SECT_ROAD:       1 move
SECT_FIELD:      1 move
SECT_FOREST:     1 move
SECT_HILLS:      2 moves
SECT_MOUNTAIN:   2 moves
SECT_WATER_SWIM: 1 move (swim check)
SECT_WATER_NOSWIM: need boat
SECT_UNDERWATER: 3 moves
SECT_DESERT:     2 moves
SECT_SNOW:       2 moves
SECT_TREE:       2 moves
SECT_UNDERGROUND: 1 move
```

**Check:** Does the Go code have these exact costs? Are they in the same order?

### 4. Room Flags

**C source** (act.movement.c):
- `ROOM_DARK` — can't see exits, need light
- `ROOM_DEATH` — death trap, instant death on entry
- `ROOM_NO_MOB` — NPCs can't enter (sometimes PCs too)
- `ROOM_INDOORS` — no weather, no flying
- `ROOM_TUNNEL` — max 2-3 people
- `ROOM_PRIVATE` — max 2 people
- `ROOM_GODROOM` — immortals only

**Check:** Does the Go code handle all room flags? Are the death trap and tunnel limits correct?

### 5. Door Commands

**C source** (act.movement.c:598, `do_gen_door()`):
Commands: OPEN, CLOSE, LOCK, UNLOCK, PICK

```c
ACMD(do_gen_door)
{
  // Find door in room or by name
  // Check for key (for lock/unlock)
  // Check for lock picks (for pick)
  // Check if pickproof
  // Toggle door state
  // Send messages
}
```

**Check:**
- Door state tracking (open/closed/locked)
- Key requirement for lock/unlock
- Lock pick requirement for pick
- Pickproof check
- Messages: "You open $p." / "$n opens $p." etc.

### 6. Follower Movement

**C source** (act.movement.c):
When a leader moves, followers follow:
```c
// After leader moves successfully:
for each follower in room:
  if follower is group member or loyal:
    perform_move(follower, dir)
    // Don't check specials for followers
```

**Check:** Do followers move automatically? Is there a delay? Do they get their own messages?

### 7. Movement Messages

**C source** (act.movement.c):
```c
// Leave message (old room):
act("You leave $F.", FALSE, ch, 0,EXIT(ch, dir)->keyword, TO_CHAR);
act("$n leaves $F.", FALSE, ch, 0,EXIT(ch, dir)->keyword, TO_NOTVICT);

// Enter message (new room):
act("You arrive from $F.", FALSE, ch, 0,EXIT(ch, dir)->keyword, TO_CHAR);
act("$n arrives from $F.", FALSE, ch, 0,EXIT(ch, dir)->keyword, TO_NOTVICT);
```

**Check:** Are these the exact messages? What about custom zone messages (some zones override the default)?

---

## Implementation Notes

- `EXIT(ch, dir)` macro gets the exit data for a direction
- Exit structures have: `to_room`, `key`, `exit_info` (OPEN, CLOSED, LOCKED), `keyword`
- The `special()` function is called on room enter — some rooms have special behavior
- Boat check: `has_boat()` checks for a boat object in inventory

---

## Verification

1. Move through rooms with different sector types — verify move costs
2. Open/close/lock/unlock doors — verify state changes
3. Try to move through locked door — verify denial message
4. Move as group leader — verify followers follow
5. Enter a death trap room — verify instant death
6. Enter a tunnel room with 3 people — verify denial
7. Swim in water sector — verify swim check
8. Run `go test ./pkg/game/...`
