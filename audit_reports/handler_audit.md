# Audit Report: handler.c vs pkg/game/ & pkg/engine/

**C file:** `src/handler.c` (1,617 lines)
**Go file(s):** `pkg/game/inventory.go` (232 lines), `pkg/game/location.go` (237 lines), `pkg/game/char_mgmt.go` (158 lines), `pkg/game/world.go` (1,376 lines), `pkg/game/world_movement.go` (193 lines), `pkg/game/world_object.go` (137 lines), `pkg/game/equipment.go` (469 lines), `pkg/engine/affect.go` (377 lines), `pkg/engine/affect_helpers.go` (344 lines), `pkg/engine/affect_manager.go` (736 lines)
**Mapping type:** 1:N
**Functions audited:** ~25

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Complete Absence of Container Recursion in `ExtractObject` (Severe Memory/Resource Leak)
- **Location:** `pkg/game/world_object.go:37` in `ExtractObject()`.
- **C behavior:** In `handler.c:1020-1024`, the `extract_obj` function recursively extracts and deletes all contents within a container being destroyed:
  ```c
  /* Get rid of the contents of the object, as well. */
  for (o = obj->contains; o; o = temp) {
    temp = o->next_content;
    extract_obj(o);
  }
  ```
  This ensures that all inner objects are deleted from the global `object_list`, their indexes decremented, and memory freed.
- **Go behavior:** The Go implementation of `ExtractObject` only removes `obj` from its own location and deletes its ID from the global `w.objectInstances` map. It completely ignores `obj.Contains`!
- **Discrepancy:** The objects inside the container are never recursively extracted or deleted from the global `w.objectInstances` map. When a player's corpse decays, a container is junked, or a bag is deleted, all objects nested inside them remain permanently inside the global map `w.objectInstances` in memory. This is a severe resource and memory leak that degrades server performance over time as thousands of orphaned items accumulate in the global lookup map.
- **Severity:** HIGH
- **Type:** BUG / DRIFT

### [FINDING-002]: Centralized Player Teleportation (`CharTransfer`) Bypasses Room Light Tracking
- **Location:** `pkg/game/world_movement.go:7` in `CharTransfer()`.
- **C behavior:** In `handler.c:520-521` (`char_from_room`) and `541-543` (`char_to_room`), the game checks if the character has a lit light source equipped (`has_light(ch)`). If so, it adjusts the room light counter accordingly:
  ```c
  if (has_light(ch))
    world[room].light++;
  ```
- **Go behavior:** While the normal walking command `MovePlayer` in `pkg/game/world.go:885` manually tracks equipped lights, the centralized `CharTransfer` helper (which handles all portal summoning, gate jumps, recalls, teleport spells, and wiz transfers) completely ignores light sources and does not decrement/increment room light levels.
- **Discrepancy:** Every time a character carrying an active light source teleports, gate-jumps, or is summoned/recalled, the room light levels permanently drift. Over time, old rooms remain illuminated forever, and new rooms are left permanently dark.
- **Severity:** HIGH
- **Type:** DRIFT

### [FINDING-003]: Summoning Circle Exclusions (`circle_check` / `COC_VNUM` Omitted)
- **Location:** `pkg/game/world_movement.go:7` in `CharTransfer()` / `src/handler.c:527`.
- **C behavior:** In `handler.c:527` and `553`, room exits and entries call `circle_check(ch)` to see if there is an active summoning circle (`COC_VNUM`, VNum 9) in the room contents, decrementing its timer and returning a boolean that spells check to validate summon/portal gates.
- **Go behavior:** The `circle_check` function is completely unported, and no room transitions in `CharTransfer` or movement packages perform summoning circle timer ticks or validations.
- **Discrepancy:** Portal summoning circles never decay during room movement, and portal spells bypass crucial gameplay constraints on active circle requirements.
- **Severity:** MEDIUM
- **Type:** STUB

### [FINDING-004]: Incomplete Delayed Player Extraction (`ExtractPendingChars`)
- **Location:** `pkg/game/char_mgmt.go:97` in `ExtractPendingChars()`.
- **C behavior:** C's delayed extraction ticker loop cleans up both players and mobs (`MOB_EXTRACT`/`PLR_EXTRACT`), and `extract_char_final` drops ALL carried inventory and equipped gear to the floor room on final extraction:
  ```c
  /* transfer objects to room, if any */
  while (ch->carrying) {
    obj = ch->carrying;
    obj_from_char(obj);
    obj_to_room(obj, ch->in_room);
  }
  ```
- **Go behavior:** Go's `ExtractPendingChars` only processes `w.players`, completely ignoring active mobs. Crucially, when extracting players, it only unequips their light source, but does *not* unequip other slots or drop their inventory items onto the ground.
- **Discrepancy:** Carried items and gear of extracted players are either locked in memory or saved to disk invisibly instead of being dumped to the room floor as designed. Mobs are extracted synchronously instead of in the delayed tick, creating concurrency and loop iteration risks.
- **Severity:** HIGH
- **Type:** DRIFT / STUB

---

## Type & Boundary Vulnerabilities

### [FINDING-005]: Equipment Alignment and Class Anti-Zap Bypass
- **Location:** `pkg/game/equipment.go:180` in `equip()`.
- **C behavior:** The legacy `equip_char` function serves as the ultimate gatekeeper for character equipment. If an item has alignment flags matching the character (`ITEM_ANTI_EVIL` with `IS_EVIL`, etc.) or fails class validation (`invalid_class`), the item is zapped and dropped back into the inventory:
  ```c
  if ((IS_OBJ_STAT(obj, ITEM_ANTI_EVIL) && IS_EVIL(ch)) || ... ) {
      act("You are zapped by $p and instantly let go of it.", ...);
      obj_to_char(obj, ch);
      return;
  }
  ```
- **Go behavior:** The Go port's low-level `equip()` implementation completely bypasses all alignment and class checks. It equips the item directly onto the slot without any validation.
- **Discrepancy:** Mobs or players can equip illegal or restricted items through automated loads, scripting, or bypass command routes without being zapped or penalized.
- **Severity:** HIGH
- **Type:** DRIFT

---

## Concurrency & Mutex Safety

### [FINDING-006]: Concurrency Risks in Synchronous Mob Extraction
- **Location:** `pkg/game/world.go:1028` in `ExtractMob()`.
- **C behavior:** Safe delayed queue loop (`extract_char`/`extract_pending_chars`) prevents modifying lists while iterating.
- **Go behavior:** Go's `ExtractMob` immediately locks `w.mu` and deletes the mob from `w.activeMobs` map synchronously.
- **Impact:** While Go maps are protected under `w.mu.Lock()`, modifying the `w.activeMobs` map while the main game ticks are iterating can lead to race conditions or skipped executions if ticks try to perform updates or run actions on combat targets that disappear mid-iteration without cascading cleanup.
- **Severity:** MEDIUM
- **Type:** CONCURRENCY

---

## Unported Functions / Systems

The following legacy C functions from `handler.c` have no Go counterparts:

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| `ITEM_TAKE_NAME` renaming | 720 (handler.c) | Dynamically renames items with `ITEM_TAKE_NAME` flags to match the wearer's name (e.g., "Zach's sword") and resets on unequip. | NO |
| `check_for_bad_stats` | 640 (handler.c) | Triggers MUD events if stats drop to 0: strength makes you too weak, intelligence/wisdom makes you dumb, dexterity causes you to trip and take damage, and charisma makes the world hate you (summoning Pestilence). | NO |

---

## Summary

- **Total findings:** 6
- **Critical:** 0
- **High:** 4
- **Medium:** 2
- **Low:** 0
- **Unported features:** 2
