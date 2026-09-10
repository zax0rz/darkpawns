# Port Fidelity Audit: Module 27 (`gate.c`)

This audit examines the port fidelity between the legacy C source file `src/gate.c` (moongates, night portal phase ticking, gate spells, and portal teleportation) and its Go implementations in `pkg/game/gates.go` and `pkg/game/spec_procs_missing.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/gate.c` (398 lines)
- **Core Functions**:
  - `load_night_gate` (spawns blue portals in rooms 4001–4008 at night based on moon phase)
  - `remove_night_gate` (extracts night portals at daybreak)
  - `moon_gate` (special procedure that teleports players entering night portals to random destinations and relocates their mounts/riders)
  - `spell_gate` (creates a red portal in rooms 4001–4008 that teleports to room 4000, or triggers portal collision death if another portal exists in the room)

### Go Port Files
- `pkg/game/gates.go` (implements blue/red portal constants, the `gatePhases` table, `LoadNightGate()`, `RemoveNightGate()`, and `SpellGate()`)
- `pkg/game/spec_procs_missing.go` (implements the `specMoonGate` special procedure registered to portals in `spec_assign.go`)
- `pkg/game/limits_condition.go` (handles object decay/tick updates hourly; completely misses portal timers)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Dysfunctional Red Portal Decay (Infinite Red Gates)
- **Source Context**: `pkg/game/gates.go#L137-L141`, `pkg/game/limits_condition.go#L315-L400`
- **Fidelity Bug**: In legacy C, red portals spawned by the `gate` spell last for 2 ticks (or 3 if cast at level 30) and are decremented during the hourly `point_update` loop, after which they fade out of existence and are deleted.
  In Go, `SpellGate` sets the red portal's timer correctly (`redGate.SetTimer(timer)`). However, **`pkg/game/limits_condition.go` contains no case for `RedPortalVNum` (4002)**.
- **Impact**: Red portals spawned by casting the `gate` spell will remain on the server **forever**, permanently clogging rooms. This poses a major hazard because casting the `gate` spell in a room that already contains a portal triggers the portal collision code, instantly killing the caster (`RawKill`).

### 2. Severe Fidelity Deficit in `specMoonGate` (The "MortalStartRoom" Shortcut)
- **Source Context**: `pkg/game/spec_procs_missing.go#L160-L177` (`specMoonGate`)
- **Fidelity Bug**: In legacy C, when a player enters a moon gate, the code scans the 16 gates in `gate_phase` to find which room the player is in, chooses randomly between two exit destinations (`gate_exit_room` and `gate_exit_room2` with a 50% chance), and teleports the character.
  In Go, `specMoonGate` **completely ignores the `gatePhases` table, the moon phase cycles, and the random destinations**. Instead, it hardcodes all entries directly to MortalStartRoom (room 8004)!
  ```go
  ch.SetRoom(MortalStartRoom)
  w.actToRoom(ch, "$n arrives through a shimmering moon gate!", nil, nil)
  ```
- **Impact**: Entering *any* moon gate across the world acts as an instant, free town-portal shortcut directly back to the temple, completely bypassing the complex astronomical transit matrix designed in the original game.

### 3. Mount and Rider Relocation Omitted
- **Source Context**: `pkg/game/spec_procs_missing.go#L160-L177`
- **Fidelity Bug**: In C, when a player enters a moon gate, the code checks if they are mounted. If so, their mount is teleported alongside them. If a mount enters, its rider is teleported.
  Go's `specMoonGate` contains **no mount or rider transfer logic**.
- **Impact**: Players stepping through portals will leave their mounts stranded behind in the original room.

### 4. Portal Look Intercept Missing
- **Source Context**: `src/gate.c#L307-L325`
- **Fidelity Bug**: C intercepted `look` commands targeting `"gate"`, `"moongate"`, or `"portal"` and printed a flavor message: `"You see a shimmering light... you could ENTER the gate.\r\n"`.
  Go's `specMoonGate` checks if command is `"enter"`; if not, it returns false. The `look` command is completely unhandled.
- **Impact**: Looking at a gate does not print the custom portal flavor text.

---

## 3. Go Improvements Over C

### 1. Safer Collision Code
- **Fidelity Improvement**: In legacy C, the portal collision code called `raw_kill(ch, TYPE_BLAST)` and immediately extracted the object, which could result in dangling pointers if not handled carefully in OLC sweeps. Go's wrapper handles this safely in memory, logging failures elegantly.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **Add Red Portal Timer Decrement in `limits_condition.go`**:
   Add a check for `RedPortalVNum` in the hourly item updater:
   ```go
   if objVNum == RedPortalVNum {
       if obj.GetTimer() > 0 {
           obj.SetTimer(obj.GetTimer() - 1)
       }
       if obj.GetTimer() == 0 {
           w.SendToRoom(roomVNum, "The shimmering red portal of light fades out of existence.\r\n")
           w.ExtractObject(obj, roomVNum)
           continue
       }
   }
   ```

2. **Restore Moon Gate Transit Matrix**:
   Refactor `specMoonGate` in `spec_procs_missing.go` to scan `gatePhases` and choose the exit room randomly matching the C logic instead of hardcoding `MortalStartRoom`.

3. **Restore Mount/Rider Relocations**:
   Integrate mounting checks (`ch.IsMounted()`) inside `specMoonGate` to transfer the mount when a player enters, matching the behavior in `pkg/game/deferred_fight_fns.go`.
