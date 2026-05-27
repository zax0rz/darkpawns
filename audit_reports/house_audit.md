# Port Fidelity Audit: Module 29 (`house.c`)

This audit examines the port fidelity between the legacy C source file `src/house.c` (player housing administrative commands, OLC building, guest lists, and crash-saving room inventories) and its Go implementations in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/house.c` (745 lines)
- **Core Functions**:
  - `House_load`, `House_save`, `House_restore_weight`, `House_crashsave` (processes recursive serialization and weight adjustments of room/container items to `.house` files)
  - `House_boot` (loads house control records from `HCONTROL_FILE` binary structure, validates directories, and applies flags)
  - `hcontrol_build_house`, `hcontrol_destroy_house`, `hcontrol_pay_house`, `hcontrol_list_houses` (admin commands to manage real-estate)
  - `do_house` (player guest lists and ownership transfer command)
  - `House_can_enter` (validates player ID against private house guest lists)

### Go Port Files
- `pkg/game/houses.go` (core `HouseControl` struct, stubs, and file location resolvers)
- `pkg/game/house_boot.go` (handles JSON parsing, room boundary validations, and startup boot operations)
- `pkg/game/house_control.go` (implements the `hcontrol` command suite and `HouseCanEnter()` permission checks)
- `pkg/game/house_player.go` (implements the user-facing `doHouse` guest-list and transfer handlers)
- `pkg/game/house_rent.go` (displays stored items inside housing files)
- `pkg/game/house_save.go` (implements JSON house serialization, room item collections, and recursive weight logic)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Critical Container Weight Corruption (Global Shared Prototype Mutation)
- **Source Context**: `pkg/game/house_save.go#L180-L190` (`collectHouseItems`), `pkg/game/house_save.go#L208-L212` (`HouseRestoreWeight`)
- **Fidelity Bug**: In the original C code, container weight was adjusted directly on the container's *specific instance weight* (`GET_OBJ_WEIGHT(tmp)`) during saving.
  In Go, `ObjectInstance` does not have an instance weight field; instead, it delegates reading weight directly to its shared prototype (`o.Prototype.Weight`).
  During house saving, `collectHouseItems` mutates the weight directly on this shared prototype:
  ```go
  container.Prototype.Weight -= obj.Prototype.Weight
  ```
  And `HouseRestoreWeight` restores it:
  ```go
  container.Prototype.Weight += obj.Prototype.Weight
  ```
- **Impact**: Any change made here permanently alters the global weight of **all boxes/chests of that VNum in the MUD** for all players! If a player saves a house containing a container, the weight of that container type changes globally on the live server, leading to severe concurrency races and global items state corruption.

### 2. Clamping Math Bug (Spiraling Container Weights to Infinity)
- **Source Context**: `pkg/game/house_save.go#L185-L188` (`collectHouseItems`), `pkg/game/house_save.go#L210-L211` (`HouseRestoreWeight`)
- **Fidelity Bug**: In `collectHouseItems`, if the items in the container are heavier than the container itself, the subtraction drops below 1 and is clamped:
  ```go
  container.Prototype.Weight -= obj.Prototype.Weight
  if container.Prototype.Weight < 1 {
      container.Prototype.Weight = 1
  }
  ```
  However, in `HouseRestoreWeight`, the addition is performed unconditionally:
  ```go
  container.Prototype.Weight += obj.Prototype.Weight
  ```
  If a container prototype weighs 5, and it contains items of weight 10, the subtraction drops to -5, which is clamped to 1. During restoration, the full 10 is added back: `1 + 10 = 11`.
- **Impact**: The box's prototype weight permanently increases from **5 to 11**. After every crash-save tick, the weight of boxes on the server will spiral upward, eventually making them so heavy that no player can lift them.

### 3. Redundant Weight Modification Code
- **Source Context**: `pkg/game/house_save.go#L137-L151` (`ObjToStore`)
- **Fidelity Bug**: In Go, `ObjToStore` serializes objects to JSON (`houseSaveItem`), which only stores `VNum`, `ContainerID`, and `State`. The object's weight is **never written to the JSON file**; when the house is loaded, the item's weight is dynamically fetched from its static prototype.
- **Impact**: Subtracting and adding weights during saving in Go is entirely redundant and serves no serialization purpose, meaning this highly dangerous global-prototype weight corruption bug was introduced to implement logic that is completely useless in the Go port!

---

## 3. Go Improvements Over C

### 1. High-Fidelity JSON Serialization
- **Fidelity Improvement**: Go replaces C's fragile, machine-dependent binary files (`struct obj_file_elem`) with standard, human-readable, and robust JSON files (`house_control.json` and `<vnum>.house`). This prevents endianness issues, structure padding drift, and file corruptions.

### 2. Concurrency Thread Safety
- **Fidelity Improvement**: C's house administration ran on the single thread. Go guards all house control changes and guest list mutations with a read/write mutex (`w.mu.Lock()`), ensuring clean thread-safety.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **Delete all Weight Adjustments in `house_save.go`**:
   Since the JSON serializer (`ObjToStore`) does not store the weight, all weight modification and restoration loops inside `houseCrashsave`, `collectHouseItems`, and `HouseRestoreWeight` should be **completely removed**. This immediately deletes the global prototype corruption and spiraling container weight bugs, and simplifies the codebase.
