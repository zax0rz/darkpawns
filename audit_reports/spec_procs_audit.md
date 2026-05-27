# Port Fidelity Audit: Module 50 (`spec_procs.c`)

This audit examines the port fidelity between the legacy C source file `src/spec_procs.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/spec_procs.c` (2,420 lines)
- **Functions & Features**:
  - **Core Special Procedures**: Implements legacy Diku behaviors for guildmasters (`guild`), death dumps (`dump`), cityguards (`cityguard`), postmasters (`postmaster`), random gossipers (`puff`), scavenger dogs (`fido`), town sweepers (`janitor`), and wandering mayors (`mayor`).
  - **Complex Tactical Combat**: Special procedures utilize combat routines (fighters, clerics, mages) to execute spells, bash, disarm, or headbutt opponents dynamically.

### Go Port Files
- **Go Implementation**:
  - [pkg/game/spec_procs.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_procs.go): Contains Go implementations of the core CircleMUD special procedures.
  - [pkg/game/postmaster.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/postmaster.go): Manages in-game mail routing spec-procs.

---

## 2. High-Fidelity Validation & Critical Gaps

This audit has discovered **devastating player gear deletion bugs** and **severe behavioral regressions** in the Go implementation of core special procedures:

### 1. Critical Scavenger Bug: Permanent Gear Deletion on Death (`specFido`)
- **The Gap**: In legacy C (`src/spec_procs.c#L735-L741`), when a scavenger dog (`specFido`) devours a player's corpse container, it recursively extracts all player gear, inventory, and gold from the corpse and spills them safely onto the room floor (`obj_to_room`) before extracting the empty corpse.
- In Go's `specFido` (`pkg/game/spec_procs.go#L483`), devouring a corpse simply calls:
  ```go
  if err := w.MoveObjectToNowhere(obj); err != nil { ... }
  ```
- **Impact**: Because the corpse's nested contents are never unloaded first, **all of the player's equipped weapons, armor, quest items, and cash stored inside their corpse are permanently deleted from the database**. Any player who dies in a room where a dog wanders will have their entire character gear wiped out.

### 2. Janitor Trash Erasure (Items Deleted on Sweep)
- **The Gap**: In legacy C, the `janitor` sweeps trash from the floor and places it inside their own inventory (`obj_to_char`). Players who accidentally dropped valuable items could kill the janitor to retrieve them.
- In Go, `specJanitor` (`pkg/game/spec_procs.go#L501`) calls `RemoveItemFromRoom(obj)` without transferring it to the janitor's inventory.
- **Impact**: Dropped items are immediately erased from existence upon sweep, preventing any retrieval.

### 3. Extremely Inactive Town Cityguards (Dumbed-Down Spec)
- **The Gap**: In legacy C, `specCityguard` dynamically protects players by searching the room, identifying combatants, and intervening in fights to attack evil NPCs/players while protecting good citizens. It also automatically attacks wanted criminals (`PLR_OUTLAW`) and calls standard `fighter` combat skills (bash, parry, headbutt).
- In Go, `specCityguard` only checks if a player moves and blocks their exit if they are an Outlaw. It **never attacks outlaws**, **never protects good players**, **never attacks evil combatants**, and **never executes combat skills**.
- **Impact**: Towns are completely undefended. Players and monsters can fight inside safe cities without guards ever intervening.

### 4. Bypassed Mayor Walking Routine & Cuchi Easter Egg
- **The Gap**:
  - In C, `specMayor` walks a complex morning and night path, automatically unlocking and opening the bazaar gates at 6 AM and locking them at 8 PM. In Go, the mayor stands static in a single room, spitting out generic citizen greetings.
  - In C, patting the cute Easter egg mob `specCuchi` awards players 10 gold, or promotes the creator "Orodreth" to Implementor level (`LVL_IMPL`). In Go, `specCuchi` is fully stubbed out as a basic citizen.

---

## 3. Go's Architectural Improvements Over C

- **Mutex Isolation**: Go's spec-procs operate cleanly inside the thread-safe `World` structure, locking resources safely to prevent stack corruption.
- **RNG Safety**: Dice rolls are cleanly mapped to random distributions without relying on legacy C array-shifting loops.

---

## 5. Summary of Recommended Next Steps

1. **FIX THE CORPSE GEAR DELETION BUG IMMEDIATELY**:
   Modify `specFido` in `pkg/game/spec_procs.go` to extract all nested items inside the corpse container and place them back in the room before moving the corpse to nowhere.
2. **Restore Janitor Inventory Transfer**:
   Update `specJanitor` to add the swept item to the janitor's inventory using `me.Inventory.addItem(obj)` instead of deleting it.
3. **Upgrade Cityguards to Intervene in Combat**:
   Implement dynamic combat scanning in `specCityguard` to check for active fights, automatically trigger attacks against outlaw players, and participate using the `fighter` combat skill methods.
4. **Restore Mayor Gateway Locking**:
   Incorporate game time checks in `specMayor` to automatically open, close, unlock, and lock bazaar gateway exits at scheduled hours.
