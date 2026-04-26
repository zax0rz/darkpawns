# QA Report — ObjectLocation Invariant Audit
**Date:** 2026-04-25  
**Branch:** main (4567be9)  
**Scope:** Post-sprint invariant audit after 11-commit ObjectLocation modernisation  

---

## 1. Build & Test Baseline

| Check | Result |
|-------|--------|
| `go build ./...` | **PASS** |
| `go vet ./...` | **PASS** |
| Pre-existing tests (15) | **15/15 PASS** |
| New regression tests (4) | **1/4 PASS — 3 FAIL (bugs confirmed)** |

---

## 2. Invariant Verification

### 2.1 `location.go` — `ObjectLocation.Validate()` covers all states

**Result: PASS with gap**

All six `ObjectLocationKind` values have explicit `case` blocks in `Validate()`. The ObjInRoom and ObjEquipped checks are thorough. However:

**Gap (low severity):** `ObjNowhere` validates `OwnerKind`, `RoomVNum`, `PlayerName`, `MobID` but does **not** check `ContainerObjID` or `ShopVNum` (location.go:127–131). A `LocNowhere()` with a stale non-zero `ContainerObjID` or `ShopVNum` would pass validation silently.

---

### 2.2 `movement.go` — MoveObject always sets Location, detach+attach are symmetric, rollback works

**Result: FAIL on rollback**

- `MoveObject` sets `obj.Location = dst` only on success ✓  
- `detachObjectLocked` is symmetric to `attachObjectLocked` — for every location kind there is a matching remove/add ✓  
- **Rollback bug (movement.go:136–137):** When the primary `attachObjectLocked` fails, MoveObject attempts a best-effort rollback:

  ```go
  w.attachObjectLocked(obj, obj.Location)  // re-attach to old location
  ```

  If this rollback also fails (e.g., old location was a full inventory that had the item force-inserted beyond capacity), `obj.Location` retains the stale old value while the object sits in **neither** the old nor new location. Correct behaviour: set `obj.Location = LocNowhere()` to signal the stranded state.

  **Test confirming this:** `TestMoveObjectRollbackStrandedLocation` — FAIL  
  **File:line:** movement.go:136–137

---

### 2.3 `inventory.go` — unexported methods only modify slices; exported wrappers delegate

**Result: PASS structurally, FAIL in practice**

- Unexported `addItem`/`removeItem`/`clear` only modify the `Items` slice ✓  
- Exported `AddItem`/`RemoveItem`/`Clear` delegate to the unexported versions ✓  

**However:** The exported wrappers are marked "Deprecated: Use MoveObject helpers instead" but are still called from ~20 sites outside `inventory.go`, most of which do **not** update `obj.Location`. See §4 (Stale Bypass Sites) for the full list.

---

### 2.4 `equipment.go` — equip sets Location, unequip sets Location

**Result: PASS for players, PARTIAL FAIL for mobs**

- `equipment.equip()` sets `item.Location = LocEquippedPlayer(eq.OwnerName, slot)` ✓  
- `equipment.unequip()` sets `item.Location = LocInventoryPlayer(eq.OwnerName)` ✓  
- `MobInstance.EquipItem()` sets `obj.Location = LocEquippedMob(id, slot)` ✓  
- `MobInstance.UnequipItem()` sets `obj.Location = LocNowhere()` then immediately overwrites with `LocInventoryMob` inside `AddToInventory` — harmless but redundant (mob.go:245–246)  

**Bug:** `attachObjectLocked` for `ObjEquipped + OwnerMob` (movement.go:92–98) calls `m.AddToInventory(obj)` **before** setting `m.Equipment[slot]`. `AddToInventory` appends to the `Inventory` slice and sets `Location = ObjInInventory`. The later `m.Equipment[slot] = obj` adds to Equipment without removing from Inventory. Result: item is in **both** slices. `obj.Location` ends up correct (overwritten by `obj.Location = dst` at MoveObject line 141) but the double-presence in Inventory+Equipment causes items to be transferred **twice** into the corpse on mob death if the mob ever dynamically equipped via MoveObject.

  **Test confirming this:** `TestMobEquipInEquipmentNotInventory` — FAIL  
  **File:line:** movement.go:92–98

---

### 2.5 `object.go` — ObjectInstance has no old Carrier/EquippedOn/Container/EquipPosition fields

**Result: PASS**

`grep -rn "Carrier\b\|EquippedOn\b\|\.Container\b.*ObjectInstance\|\.EquipPosition\b" pkg/game/*.go` returns zero results (excluding movement.go and location.go). The stale fields are fully gone. ✓

`CustomData` is intentionally kept as migration bridge per `MigrateCustomData()`. ✓

---

### 2.6 `death.go` — corpse uses MoveObjectToContainer, dust uses MoveObjectToRoom

**Result: PASS**

- `makeCorpse` calls `MoveObjectToContainer(item, corpse)` for each inventory and equipment item ✓  
- `makeDust` calls `MoveObjectToRoom(item, roomVNum)` for each item ✓  
- The corpse is registered in `w.objectInstances` before `MoveObjectToContainer` is called (death.go:397–401) so detach can find the container ✓  

**Note:** Player death pre-clears `Inventory` and `Equipment` before the MoveObject loop (death.go:221–235). This is a fragile pattern — `detachObjectLocked` silently no-ops when it can't find the item, making the move succeed. It works today but creates a hidden dependency on detach being a no-op for already-cleared collections.

---

### 2.7 `runtime_state.go` — typed fields, no external CustomData usage

**Result: PASS**

- `ObjectRuntimeState` and `MobRuntimeState` have typed fields ✓  
- New Go code does not add keys to `CustomData` or `Script`; the `// New Go code should NOT add keys here` comment is present ✓  
- `MigrateCustomData()` cleanly migrates known keys and deletes them ✓

---

## 3. Stale-Reference Search Results

```
grep -rn "Carrier\b|EquippedOn\b|\.Container\b.*ObjectInstance|\.EquipPosition\b" pkg/game/*.go
```
**→ 0 results** ✓

---

## 4. Direct addItem/removeItem Bypassing Location

The following sites call `addItem`/`removeItem` directly without updating `obj.Location`. Items at these sites will have stale Location values that `MoveObject`'s `detachObjectLocked` will silently mishandle.

| File | Line(s) | Context | Severity |
|------|---------|---------|----------|
| `skills.go` | 778–779 | Steal: removes from victim, adds to thief. No Location update. If victim later uses MoveObject on any item, detach may walk wrong player's inventory. | **HIGH** |
| `spec_procs2.go` | 545, 605, 745, 1094 | `SpawnObject(…, room)` then `addItem` — Location stays `ObjInRoom` but item is in inventory | **HIGH** |
| `spec_procs2.go` | 576 | `SpawnObject` → `addItem` for a different player | **HIGH** |
| `spec_procs2.go` | 637, 918 | `removeItem` to destroy items; Location stays `ObjInInventory` | **MEDIUM** |
| `objsave.go` | 147, 153, 166, 174, 178, 241 | Load path: items loaded from DB via `addItem`; Location never set | **HIGH** (affects every login) |
| `world.go` | 975 | `RemoveItemFromCharByVNum`: removes from slice, sets `item.RoomVNum = -1` but Location stays `ObjInInventory` | **MEDIUM** |
| `world.go` | 994 | `GiveItemToCharScriptable`: `addItem` without Location update | **MEDIUM** |
| `world.go` | 953–963 | `RemoveItemFromRoomByVNum`: removes from roomItems, sets `item.RoomVNum = -1`, Location stays `ObjInRoom` | **MEDIUM** |
| `item_equipment.go` | 143 | `removeItem(obj)` before `EquipItem(ch, obj, where)`. Location update delegated to `Equipment.equip()` which sets it — but only on success; if equip fails silently, Location is stale | **LOW** |
| `objsave.go` | (AutoEquip path) | `AutoEquip` calls `p.Inventory.addItem(obj)` on alignment/wear-flag failures without setting Location | **MEDIUM** |

**Total: ~22 call sites that bypass Location.**

---

## 5. Concurrency Check — w.mu.Lock() → MoveObject Deadlock Risk

`MoveObject` acquires `w.mu.Lock()` at movement.go:125. A deadlock would occur if any caller holds `w.mu` and then calls `MoveObject`.

Checked all 22 `w.mu.Lock()` sites:

| File | Line | Context | Risk |
|------|------|---------|------|
| `death.go` | 122 | Searches `activeMobs` map; unlocks at 132 before `makeCorpse`/`MoveObject` calls | **NONE** |
| `death.go` | 172 | `delete(activeMobs, id)`; unlocks at 174 before any MoveObject call | **NONE** |
| `death.go` | 397 | Assigns `nextObjID`; unlocks at 401 before `MoveObjectToContainer` loop | **NONE** |
| `world.go` | 403 | `AddItemToRoom` (deprecated) — no MoveObject call inside | **NONE** |
| `world.go` | 411 | `ExtractObject` — uses internal `removeItemFromRoomLocked`, no MoveObject | **NONE** |
| `houses.go` | 203, 522, 627, 684, 722, 824, 926 | None of these call MoveObject | **NONE** |
| `movement.go` | 125 | MoveObject itself | N/A |
| All others | — | No MoveObject call inside lock scope | **NONE** |

**No deadlock risk found.** ✓

---

## 6. Container Cycle Detection

`attachObjectLocked` for `ObjInContainer` (movement.go:100–109):

```go
case ObjInContainer:
    if dst.ContainerObjID > 0 {
        if container, ok := w.objectInstances[dst.ContainerObjID]; ok {
            if !container.AddToContainer(obj) {
                return fmt.Errorf("container %d cannot hold object", dst.ContainerObjID)
            }
        } else {
            return fmt.Errorf("container object %d not found", dst.ContainerObjID)
        }
    }
```

`AddToContainer` only checks `o.IsContainer()` (TypeFlag == 1). **There is no cycle detection.**

If containerA (which holds containerB) is moved inside containerB, `MoveObjectToContainer(A, B)` succeeds. The result is:
- `containerA.Contains` includes `containerB`  
- `containerB.Contains` includes `containerA`  

This corrupts `GetTotalWeight()` (infinite recursion), `GetContents()` traversals, and any corpse dump logic.

**Test confirming this:** `TestContainerCyclePrevention` — FAIL  
**File:line:** movement.go:100–109 and object.go:141–148

---

## 7. New Regression Tests Added

File: `pkg/game/object_movement_test.go`

| Test | Status | Documents |
|------|--------|-----------|
| `TestContainerCyclePrevention` | **FAIL** | Bug: no cycle guard in `attachObjectLocked` |
| `TestMobEquipInEquipmentNotInventory` | **FAIL** | Bug: mob equip via MoveObject adds to both Inventory and Equipment |
| `TestMobUnequipToInventory` | **PASS** | Mob unequip via MoveObject correctly removes from Equipment and adds to Inventory |
| `TestMoveObjectRollbackStrandedLocation` | **FAIL** | Bug: stranded object retains stale Location after double-failed MoveObject |

---

## 8. Findings Summary

### Bugs (requires fix)

| # | Severity | File:Line | Description |
|---|----------|-----------|-------------|
| B1 | **CRITICAL** | movement.go:100–109, object.go:141 | No cycle detection in container attach — corrupts object graph |
| B2 | **HIGH** | movement.go:92–98 | Mob equip via MoveObject adds item to both `Inventory` and `Equipment` slices |
| B3 | **HIGH** | objsave.go:147–241 | Load path uses `addItem` without setting `obj.Location` — all loaded items have stale Location on login |
| B4 | **HIGH** | skills.go:778–779, spec_procs2.go (multiple) | ~20 bypass sites add/remove items without Location update |
| B5 | **MEDIUM** | movement.go:136–137 | Rollback failure leaves `obj.Location` stale (should be `LocNowhere`) |
| B6 | **LOW** | location.go:127–131 | `Validate()` for `ObjNowhere` doesn't check `ContainerObjID`/`ShopVNum` |

### Non-bugs / Notes

| # | File | Note |
|---|------|------|
| N1 | death.go:221–235 | Player death pre-clears inventory before MoveObject loop — fragile but functional |
| N2 | mob.go:245–246 | `UnequipItem` sets `LocNowhere` then immediately overwrites with `LocInventoryMob` — harmless |
| N3 | world.go:402 | `AddItemToRoom` deprecated but still used in tests; safe only with SpawnObject's LocRoom pre-set |

---

## 9. Recommendations

1. **Fix B1 first** — add a cycle check to `attachObjectLocked` before calling `AddToContainer`. Walk `obj.Contains` recursively to check if `dst.ContainerObjID` is already a descendant.

2. **Fix B2** — replace the mob-equip block in `attachObjectLocked` (movement.go:92–98) with a call to `m.EquipItem(obj, int(dst.Slot))` which correctly handles Inventory+Equipment separately and sets Location.

3. **Fix B3** — audit `objsave.go` load path to set `obj.Location` correctly after each `addItem` (use `LocInventoryPlayer` or `LocEquippedPlayer` based on where the item was saved).

4. **Fix B4** — work through the ~20 bypass sites and either switch them to `MoveObject` helpers or add explicit `obj.Location = ...` assignments. The steal path in `skills.go` and spec_proc item grants in `spec_procs2.go` are the highest-traffic cases.

5. **Fix B5** — in `MoveObject`, after a failed rollback-attach, set `obj.Location = LocNowhere()` before returning the error.

6. **Fix B6** — add `ContainerObjID` and `ShopVNum` checks to the `ObjNowhere` case in `Validate()`.
