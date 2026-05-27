# Port Fidelity Audit: Module 41 (`objsave.c`)

This audit examines the port fidelity between the legacy C source file `src/objsave.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/objsave.c` (1,250 lines)
- **Functions & Features**:
  - **Object Saving / Rent Storage**: Manages players' inventory and equipment persistence using separate crash/rent files (`.objs` files).
  - **Receptionist & Cryogenicist Spec-Procs**: Provides interactive shopkeepers (`receptionist` and `cryogenicist`) that check rent costs, list active rent prices, and voluntary rent/freeze player items.
  - **Container Nesting Reconstitution**: Reconstructs nested container hierarchies on load using a specialized array stack `cont_row[MAX_BAG_ROW]` (`MAX_BAG_ROW = 5`) to correctly restore nested items inside bags.
  - **Rent Deadlines**: Calculates daily rent limitations via `Crash_rent_deadline` based on combined cash on hand and bank savings.

### Go Port Files
- **Go Implementation**:
  - [pkg/game/objsave.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/objsave.go): Port of the core calculations (`IsUnrentable`, `CalculateRent`, `ExtractObjs`, `AutoEquip`, `OfferRent`, `Idlesave`, `RentSave`, `CryoSave`).
  - [pkg/game/save.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/save.go): Handles JSON player save serialization, containing a flat list of `Inventory` and `Equipment` slots.
  - [pkg/db/convert.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/db/convert.go): Translates player database records into runtime Player structs, restoring raw VNums into the inventory and equipment maps.

---

## 2. High-Fidelity Validation & Critical Gaps

This audit has discovered multiple **critical structural gaps** and a **catastrophic data loss bug** in the Go implementation of the saving/rent system:

### 1. Catastrophic Player-Deletion Data Loss Bug (Idlesave)
- **The Gap**: In `src/objsave.c#L196-L197`, deleting a crash file (`Crash_delete_file`) only deletes the temporary item-save file (`<name>.objs`), leaving the character's core save sheet (`<name>.plr`) completely intact.
- In Go's `Idlesave` (`pkg/game/objsave.go#L559-L563`), if a player idles out with no items in their inventory or equipment, the server tries to delete the temporary crash file:
  ```go
  // If player has nothing left, delete the save file and return.
  if len(p.Inventory.Items) == 0 && len(p.Equipment.Slots) == 0 {
      DeleteCrashFile(p.GetName())
      return
  }
  ```
  However, because player attributes and items are merged into a single `<name>.json` file, `DeleteCrashFile` delegates directly to `DeletePlayer(name)` (`pkg/game/objsave.go#L726`), which **permanently deletes the player's entire account/character JSON file from disk!**
- **Impact**: Any player who idles out with no inventory or equipment will have their character sheet **permanently erased from the game**. The next time they log in, they must recreate their character from scratch.

### 2. Nesting Hierarchy & Custom Object State Discarded (DB Save/Load)
- **The Gap**: CircleMUD uses the `cont_row` array stack to save and restore complex container nesting (e.g., a key inside a pouch inside a backpack).
- In Go, `pkg/db/convert.go#L82-L111` serializes the player's inventory as a flat array of VNums (`[]int`) and equipment as a flat slot-to-vnum map (`map[string]int`).
- **Impact**:
  - **Nesting Lost**: When a player logs out or rents, all nested container relations are flattened. Any items inside containers are dumped directly into the player's general inventory on login.
  - **State Discarded**: Because only VNums are saved, all custom object state (custom name modifiers, descriptions, magic charges, weapon stats, or object values altered during play) is **completely wiped out**, resetting all items back to their base prototypes.

### 3. Receptionist Bank Gold Ignored (Rent Blocker)
- **The Gap**: In `src/objsave.c#L1021`, the receptionist's `Crash_rent_deadline` uses the player's combined cash and bank gold (`GET_GOLD(ch) + GET_BANK_GOLD(ch)`) to calculate the daily limit.
- In Go's `GenReceptionist` (`pkg/game/objsave.go#L740`), the receptionist only verifies `p.Gold` (cash on hand) when offering rent:
  ```go
  cost := OfferRent(p, true, mode)
  if cost > 0 && cost <= p.Gold { ... }
  ```
- **Impact**: Players cannot voluntary rent or freeze items if their wealth is stored in the bank. They are forced to manually walk to a bank, withdraw all their cash, and carry huge, dangerous quantities of gold to the receptionist.

### 4. Dormant/Stubbed Rent Deadline & Missing Unrentable Output
- **The Gap**:
  - In `pkg/game/objsave.go#L693`, `RentDeadline` calculates `days := (ch.Gold) / cost` but assigns it to a discarded format string `_ = fmt.Sprintf(...)`, failing to ever transmit the deadline message to the player.
  - `GenReceptionist` fails to report unrentable items or output warning messages explaining *which* items in their inventory are blocking the rent action, leaving players blind as to why they "can't afford to rent."

---

## 3. Go's Architectural Improvements Over C

Despite the bugs, Go's implementation introduces excellent modernize improvements:
1. **Safety from Pointer Corruption**: Legacy C uses recursive pointer links (`obj->contains`, `obj->next_content`) which are notoriously easy to corrupt or leak during extraction/movement. Go uses typed slices (`[]*ObjectInstance`), preventing dereferencing of raw pointers.
2. **Simplified Weight Calculations**: C manually increments and decrements weight balances up the parent-chain. Go uses clean dynamic mapping (`RestoreWeight` caching weights in `CustomData["restored_weight"]`), preventing integer overflow and container leaks.

---

## 4. Concurrency & Thread Safety

- **Session Locking Issues**: The save loop (`SaveAllPlayers`) runs concurrently with character tickers. While `playerToSaveData` acquires `p.mu.RLock()`, operations within `Idlesave` (unequipping items and deleting inventory contents) lack full mutex guarding on the parent player structure, exposing the character to concurrent combat ticks or commands while their items are being stripped/saved.
- **Garbage Collection Safety**: Slices are safely cleaned via `ExtractObjs` (`p.Inventory.clear()`), allowing Golang's GC to collect unreferenced object instances without requiring dangerous manual `free()` loops.

---

## 5. Summary of Recommended Next Steps

1. **FIX THE IDLESAVE BUG IMMEDIATELY**:
   Change the logic of `DeleteCrashFile` to only clear item inventory arrays in the JSON/DB record rather than calling `DeletePlayer` to erase the entire character save.
2. **Upgrade DB Schema for Item Reconstitution**:
   Migrate `PlayerRecord`'s flat `Inventory` / `Equipment` columns to support JSON strings holding serializable `saveItemData` structures containing item VNums, state, and nesting depths.
3. **Integrate Bank Gold in Receptionist Validation**:
   Modify `GenReceptionist` and `RentDeadline` to check `p.Gold + p.BankGold`, letting players utilize their bank savings.
4. **Fix Discarded Deadline Output**:
   Convert the discarded `_ = fmt.Sprintf(...)` in `RentDeadline` to active `ch.SendMessage` socket writes.
