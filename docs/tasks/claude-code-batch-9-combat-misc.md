# Claude Code Batch — Run 9: Combat, Visibility & Equipment Fixes

## Overview
4 fidelity issues across different subsystems. Zero file overlap between tasks.

## Issues
- DP-382: Sneaking/invisible characters visible in room list (HIGH)
- DP-437: Equipment class checks hardcoded false (HIGH)
- DP-396: Shoot command truncated — no ranged mechanic (MEDIUM)
- DP-398: Immortal kill/instakill missing (LOW)

---

## Task 1: Fix sneaking/invisible visibility in room lists (DP-382)

**File:** `pkg/game/look.go` — `listCharToChar()` (line 166)

**C source:** `src/act.informative.c` — `list_char_to_char()`:
- For each character in room: if `!CAN_SEE(ch, victim)` → skip (completely invisible)
- `CAN_SEE` checks: blindness, AFF_INVISIBLE (needs detect_invis), AFF_SNEAK (needs detect_sneak or level difference)
- Only shows "You sense a slight presence" for sneaking characters that fail the visibility check

**Current Go (broken):**
```go
func (w *World) listCharToChar(room *parser.Room, ch *Player) {
    for _, m := range room.Mobs {
        if !chCanSee(ch, m) {  // chCanSee only checks blindness!
            continue
        }
        // ... prints mob name
    }
    for _, p := range room.Players {
        if p == ch { continue }
        // NO visibility check for sneak/invis at all!
        // ... prints player name
    }
}
```

**Fix:** Add visibility checks matching the C source:

For **mobs** (line 172): Replace `chCanSee(ch, m)` with a proper check that also tests:
- `m.IsAffected(affInvis) && !ch.IsAffected(affDetectInvis)` → skip
- `m.IsAffected(affSneak) && !ch.IsAffected(affDetectSneak) && ch.GetLevel() < m.GetLevel()+10` → skip (or show "slight presence")

For **players** (around line 185): Add before printing:
```go
if p.IsAffected(affSneak) && !ch.IsAffected(affDetectSneak) && ch.GetLevel() < p.GetLevel()+10 {
    continue
}
if p.IsAffected(affInvis) && !ch.IsAffected(affDetectInvis) {
    continue
}
```

**Check:** What affect constants exist? Look in `pkg/game/affects.go` or `pkg/game/constants.go` for:
- `affSneak`, `affInvis` (or `affInvisible`), `affDetectInvis`, `affDetectSneak`, `affSenseLife`

Also check `chCanSee` in `look.go` — it may need to be enhanced rather than bypassed.

---

## Task 2: Equipment class checks — pass real weapon/shield data (DP-437)

**File:** `pkg/game/equipment.go:193`

**Current Go (broken):**
```go
if InvalidClass(class, uint32(xf), false, false) {
    return true, nil
}
```
Both `isWieldedSlashWeapon` and `isShield` are hardcoded `false`. Clerics can wield slashing weapons, thieves can use shields.

**Fix:** Before the `InvalidClass` call, detect the weapon type and shield bit:

```go
// Detect weapon type and shield status
isSlash := false
isShieldItem := false

if item.Prototype != nil {
    // Check if item is a weapon — Values[3] is damage type
    // TYPE_SLASH = TYPE_HIT(6) + 3 = 9, but we just need the offset
    // Values[3] == 3 means slash damage (relative to TYPE_HIT)
    if item.Prototype.TypeFlag == int(ItemWeaponType) {
        isSlash = item.Prototype.Values[3] == 3 // TYPE_SLASH - TYPE_HIT
    }
    // Check if item has shield wear flag
    for _, flag := range item.Prototype.WearFlags {
        if flag == 9 { // ITEM_WEAR_SHIELD = bit 9
            isShieldItem = true
            break
        }
    }
}

if InvalidClass(class, uint32(xf), isSlash, isShieldItem) {
    return true, nil
}
```

**Verify:**
- `item.Prototype.TypeFlag` is `int` — compare with `int(ItemWeaponType)` (value 5)
- `item.Prototype.Values` is `[16]int` or similar — `Values[3]` is damage type
- `item.Prototype.WearFlags` — check the type (likely `[]int` or `[]uint32`)
- `ItemWeaponType` is defined in `pkg/game/item_helpers.go:18` as `ItemType = 5`

**C source:** `src/act.item.c:1600` — `wear()` function:
```c
if (GET_OBJ_TYPE(obj) == ITEM_WEAPON)
    slash = (GET_OBJ_VAL(obj, 3) == TYPE_SLASH - TYPE_HIT);
if (IS_SET(GET_OBJ_WEAR(obj), ITEM_WEAR_SHIELD))
    shield = TRUE;
```

---

## Task 3: Shoot command — add ranged mechanic (DP-396)

**File:** `pkg/game/skill_c10_combat.go:110-136` — `DoShoot()`

**C source:** `src/act.offensive.c` — `do_shoot()`:
1. Check shooter has a ranged weapon wielded (bow/sling — check weapon type)
2. Parse direction: `shoot <target> <direction>` or `shoot <direction> <target>`
3. Find target in adjacent room in that direction
4. Roll hit check
5. On success: damage the target, then `char_from_room(target)` + `char_to_room(target, shooter_room)` — drags target into shooter's room
6. On fail: "Your shot misses wildly."

**Current Go (broken):** Restricted to same room — no direction parsing, no exit traversal.

**Fix — simplified ranged mechanic:**

The full C implementation is complex (projectile objects, ammo tracking). For fidelity, implement the core mechanic:

1. Parse direction from args: `shoot <target> <direction>` or `shoot <direction> <target>`
2. Validate direction is a valid exit from shooter's room
3. Find target mob in the adjacent room
4. Roll hit (use existing combat hit check)
5. On hit: deal damage, drag target to shooter's room
6. On miss: "Your shot misses wildly."

```go
// After skill check succeeds:
// Parse direction from args
parts := strings.Fields(arg)
var targetName, direction string
for _, dir := range []string{"north","south","east","west","up","down","n","s","e","w","u","d"} {
    for _, part := range parts {
        if strings.EqualFold(part, dir) {
            direction = dir
            targetName = strings.ReplaceAll(arg, dir, "")
            targetName = strings.TrimSpace(targetName)
            break
        }
    }
}

if direction == "" {
    ch.SendMessage("Shoot in which direction?\r\n")
    return
}

// Find exit
room := w.GetRoomInWorld(ch.GetRoomVNum())
exit, ok := room.Exits[direction]
if !ok {
    ch.SendMessage("There is no exit in that direction.\r\n")
    return
}

// Find target in adjacent room
// ... search mobs in exit.ToRoom
```

**Note:** This is the most complex task in this batch. If the full ranged mechanic is too large for one subagent, implement a simplified version:
- Accept direction arg
- Check exit exists
- Send "You shoot into the <direction>!" message
- Skip the actual damage/drag for now
- Mark as partial fix with a TODO for the full implementation

---

## Task 4: Immortal kill/instakill (DP-398)

**File:** `pkg/session/commands.go` — command registry

**C source:** `src/act.offensive.c` — `do_kill()`:
```c
ACMD(do_kill) {
    if (IS_NPC(ch)) { ... }
    if ((GET_LEVEL(ch) >= LVL_IMMORT) && !*arg) {
        send_to_char("Kill who?\r\n", ch);
        return;
    }
    if (*arg) {
        // Find target, check same room
        // If immortal: instant kill (set health to 0, call die())
        // If mortal: delegate to do_hit()
    }
}
```

**Current Go:** `kill` is a simple alias to `hit` for all levels.

**Fix:** In the command registry or in the hit function, check if the player is immortal:

Look at where `kill` is registered — it's likely `cmdRegistry.Register("kill", wrapArgs(cmdHit), ...)`.

**Option A — modify cmdHit:**
Add at the top of `cmdHit` (or a new `cmdKill` wrapper):
```go
func cmdKill(s *Session, args []string) error {
    if s.player.GetLevel() >= LVL_IMMORT && len(args) > 0 {
        // Instakill — find target, set HP to 0, call death
        target := findTargetInRoom(s, args[0])
        if target != nil {
            target.SetHP(0)
            // Trigger death processing
            s.manager.world.ProcessDeath(target, s.player)
            s.Send(fmt.Sprintf("You INSTAKILL %s!\r\n", target.Name))
            return nil
        }
        s.Send("They aren't here.\r\n")
        return nil
    }
    // Fall through to normal hit for mortals
    return cmdHit(s, args)
}
```

**Option B — separate command:**
Register `kill` as `cmdKill` (immortal instakill) and keep `hit` as the mortal attack:
```go
cmdRegistry.Register("kill", wrapArgs(cmdKill), "Instakill a target (immortal).", LVL_IMMORT, 0)
cmdRegistry.Register("hit", wrapArgs(cmdHit), "Attack a target.", 0, 0)
```

**Check:** How does the Go code handle death? Look for `ProcessDeath`, `die()`, `raw_kill()` in `pkg/game/`. The death system was fixed in an earlier batch — use the same path.

---

## Execution Order
All 4 tasks are independent. Recommended order:
1. Task 2 (equipment class check) — smallest, ~10 lines
2. Task 1 (sneak/invis visibility) — medium, affect constants
3. Task 4 (immortal kill) — small, needs death system path
4. Task 3 (shoot ranged) — largest, may be partial

## Verification
1. `go build ./...` — must pass after each task
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
