# ObjectLocation Refactor — Design Document

> Status: Phase 1 Prep Complete (tests written, bugs documented)
> Based on: `pkg/game/location.go`, `pkg/game/movement.go`, existing tests in `pkg/game/object_movement_test.go`

## 1. Current State (As-Found)

The `ObjectLocation` tagged union **already exists** in `pkg/game/location.go` and `MoveObject` **already exists** in `pkg/game/movement.go`. The `Carrier interface{}` and `EquippedOn interface{}` fields have already been removed from `ObjectInstance`.

What remains is cleanup, centralization, and hardening:

### 1.1 What's Already Done

- `ObjectLocation` struct with `Kind`, `OwnerKind`, `RoomVNum`, `PlayerName`, `MobID`, `ContainerObjID`, `ShopVNum`, `Slot`
- `MoveObject` / `detachObjectLocked` / `attachObjectLocked` — centralized movement with rollback
- `LocRoom`, `LocInventoryPlayer`, `LocInventoryMob`, `LocEquippedPlayer`, `LocEquippedMob`, `LocContainer`, `LocShop`, `LocNowhere` constructors
- Validation (`Validate()`), predicates (`IsInRoom`, `IsInInventory`, `IsEquipped`, etc.)
- Comprehensive test suite in `pkg/game/object_movement_test.go` documenting current behavior

### 1.2 Bugs Documented by Tests (Failing Today)

| Test | Bug | Severity |
|------|-----|----------|
| `TestExtractObjectFromPlayerEquipment` | `ExtractObject` ObjEquipped branch is a no-op for `OwnerPlayer`; item stays in `Equipment.Slots` | **High** |
| `TestExtractObjectFromMobEquipment` | `ExtractObject` ObjEquipped branch is a no-op for `OwnerMob`; item stays in `mob.Equipment` map | **High** |
| `TestSaveLoadLocationRoundTrip` | `AutoEquip` (objsave.go) calls `addItem` for inventory but never sets `Location`; items loaded from disk have `Kind=ObjInRoom` (from `NewObjectInstance`) instead of `ObjInInventory` | **High** |
| `TestContainerCyclePrevention` | `attachObjectLocked(ObjInContainer)` has no cycle check; `MoveObjectToContainer(A, B)` succeeds when B is already inside A | **Medium** |
| `TestMobEquipInEquipmentNotInventory` | `attachObjectLocked(ObjEquipped, OwnerMob)` calls `m.AddToInventory(obj)` before setting `Equipment[slot]`; item ends up in BOTH Inventory and Equipment | **Medium** |
| `TestMoveObjectRollbackStrandedLocation` | When both attach and rollback-attach fail, `obj.Location` retains stale value instead of `LocNowhere()` | **Low** |

### 1.3 Dual Tracking: `RoomVNum` Field on `ObjectInstance`

`ObjectInstance` still has a `RoomVNum int` field alongside `Location.ObjectLocation.RoomVNum`. `SetRoomVNum` updates both. This is redundant and a source of inconsistency.

**Decision:** Remove `RoomVNum` from `ObjectInstance`. The canonical room is `obj.Location.RoomVNum`. Update `SetRoomVNum` to set `obj.Location.RoomVNum` only.

### 1.4 Bypassed Movement Paths

Many code paths mutate inventory/equipment/containers directly without going through `MoveObject`:

- `Inventory.AddItem()` / `RemoveItem()` — direct slice manipulation
- `Equipment.Equip()` / `Unequip()` — direct map manipulation + Location update
- `MobInstance.AddToInventory()` / `RemoveFromInventory()` — direct slice manipulation
- `MobInstance.EquipItem()` / `UnequipItem()` — direct map manipulation
- `ObjectInstance.AddToContainer()` / `RemoveFromContainer()` — direct slice manipulation
- `AutoEquip()` — direct calls to `addItem` / `Equip`

**Decision:** These methods can stay as internal primitives, but ALL external callers (commands, scripts, spec procs, shop system) must use `MoveObject`. A follow-up audit should convert direct callers.

---

## 2. ObjectLocation Struct Definition (Current — No Change Needed)

```go
type ObjectLocationKind uint8

const (
    ObjNowhere ObjectLocationKind = iota
    ObjInRoom
    ObjInInventory
    ObjEquipped
    ObjInContainer
    ObjInShop
)

type ObjectOwnerKind uint8

const (
    OwnerNone ObjectOwnerKind = iota
    OwnerPlayer
    OwnerMob
)

type ObjectLocation struct {
    Kind           ObjectLocationKind
    OwnerKind      ObjectOwnerKind
    RoomVNum       int
    PlayerName     string
    MobID          int
    Slot           EquipmentSlot
    ContainerObjID int
    ShopVNum       int
}
```

This design is correct. The "tagged union" pattern with explicit `Kind` + `OwnerKind` is clearer than an interface hierarchy and serializes cleanly.

---

## 3. MoveObject Function Signature (Current — No Change Needed)

```go
func (w *World) MoveObject(obj *ObjectInstance, dst ObjectLocation) error
func (w *World) MoveObjectToNowhere(obj *ObjectInstance) error
func (w *World) MoveObjectToRoom(obj *ObjectInstance, roomVNum int) error
func (w *World) MoveObjectToPlayerInventory(obj *ObjectInstance, player *Player) error
func (w *World) MoveObjectToMobInventory(obj *ObjectInstance, mob *MobInstance) error
func (w *World) MoveObjectToContainer(obj *ObjectInstance, container *ObjectInstance) error
```

The helpers are fine. The core `MoveObject` handles all cases.

---

## 4. Migration Plan (Incremental — Phases A–D)

### Phase A — Fix ExtractObject Equipped Branches (High Priority)

**Files:** `pkg/game/movement.go`

**Changes:**
1. In `ExtractObject`, replace the `ObjEquipped` no-op comment with actual unequip logic:
   - If `OwnerPlayer`: look up player by name, call `p.Equipment.unequip(obj)` or `p.Equipment.RemoveItem(obj)`
   - If `OwnerMob`: look up mob by ID, `delete(mob.Equipment, slot)`
2. Ensure `obj.Location = LocNowhere()` is set after removal.

**Risk:** Low. This fixes a real bug. Existing callers that didn't hit this branch (because they manually unequip before extract) are unaffected.

### Phase B — Fix Save/Load Location Preservation (High Priority)

**Files:** `pkg/game/objsave.go`, `pkg/game/save.go`

**Changes:**
1. Add `Location ObjectLocation` field to `saveItemData`
2. In `playerToSaveData`: serialize `item.Location` into `saveItemData.Location`
3. In `AutoEquip` / load path: after placing item into inventory or equipment, set `obj.Location` to the correct `ObjectLocation`
4. Backward compatibility: if `Location` is missing in old save files, fall back to current behavior (infer from `Locate`)

**Risk:** Medium. Save format change requires backward compat.

### Phase C — Remove Dual `RoomVNum` Tracking (Medium Priority)

**Files:** `pkg/game/object.go`, `pkg/game/location.go`, callers

**Changes:**
1. Remove `RoomVNum int` field from `ObjectInstance`
2. Update `SetRoomVNum` to set `obj.Location.RoomVNum`
3. Update `GetRoomVNum` to read `obj.Location.RoomVNum`
4. Audit all `obj.RoomVNum` direct reads and redirect to `obj.Location.RoomVNum`

**Risk:** Low. `RoomVNum` is already synced with `Location` in all `MoveObject` paths.

### Phase D — Harden MoveObject Edge Cases (Medium Priority)

**Files:** `pkg/game/movement.go`

**Changes:**
1. **Container cycle prevention:** In `attachObjectLocked(ObjInContainer)`, walk up `container.Location` chain to verify the target container is not `obj` itself or inside `obj`
2. **Mob equip duplication fix:** In `attachObjectLocked(ObjEquipped, OwnerMob)`, remove the `m.AddToInventory(obj)` call; equipment-only placement is correct
3. **Stranded location fix:** In `MoveObject`, if both attach and rollback fail, set `obj.Location = LocNowhere()` before returning error

**Risk:** Medium. These change behavior in edge cases. Tests already document expected behavior.

---

## 5. Compatibility Shims

During transition, keep these helpers for code that hasn't been migrated yet:

```go
// InInventoryOf returns true if the object is in the named player's inventory.
func (o *ObjectInstance) InInventoryOf(playerName string) bool {
    return o.Location.Kind == ObjInInventory &&
        o.Location.OwnerKind == OwnerPlayer &&
        o.Location.PlayerName == playerName
}

// InInventoryOfMob returns true if the object is in the mob's inventory.
func (o *ObjectInstance) InInventoryOfMob(mobID int) bool {
    return o.Location.Kind == ObjInInventory &&
        o.Location.OwnerKind == OwnerMob &&
        o.Location.MobID == mobID
}

// EquippedBy returns true if the object is equipped by the named player.
func (o *ObjectInstance) EquippedBy(playerName string) bool {
    return o.Location.Kind == ObjEquipped &&
        o.Location.OwnerKind == OwnerPlayer &&
        o.Location.PlayerName == playerName
}

// EquippedByMob returns true if the object is equipped by the mob.
func (o *ObjectInstance) EquippedByMob(mobID int) bool {
    return o.Location.Kind == ObjEquipped &&
        o.Location.OwnerKind == OwnerMob &&
        o.Location.MobID == mobID
}
```

These already exist as methods on `ObjectLocation` (`IsInInventory`, `IsEquipped`, `OwnerIsPlayer`, `OwnerIsMob`). Just ensure callers use them.

---

## 6. Risk Assessment

| Code Path | Risk | Mitigation |
|-----------|------|------------|
| `ExtractObject` with equipped items | **High** — currently silently fails to remove from equipment | Phase A fixes this; test coverage exists |
| Save/load of player items | **High** — Location not preserved across sessions | Phase B adds Location to save format; backward compat fallback |
| Container cycles | **Medium** — can corrupt weight calculations | Phase D adds cycle check; test exists |
| Mob equip via `MoveObject` | **Medium** — duplicates into inventory | Phase D fixes; test exists |
| `RoomVNum` removal | **Low** — already synced with Location | Phase C; grep for direct field access |
| Shop system (`systems/shop.go`) | **Low** — sets `SetRoomVNum(-1)` for shop items | Verify it uses `LocShop` or equivalent after refactor |

---

## 7. Save/Load Migration

### Current Format (`saveItemData`)

```go
type saveItemData struct {
    VNum   int                    `json:"vnum"`
    Count  int                    `json:"count"`
    Locate int                    `json:"locate"` // 0=inventory, 1+=wear slot
    State  map[string]interface{} `json:"state,omitempty"`
}
```

### New Format

```go
type saveItemData struct {
    VNum     int                    `json:"vnum"`
    Count    int                    `json:"count"`
    Locate   int                    `json:"locate"`   // deprecated, kept for compat
    Location *ObjectLocation        `json:"location,omitempty"`
    State    map[string]interface{} `json:"state,omitempty"`
}
```

### Load Logic

```go
func reconstructItem(data saveItemData) *ObjectInstance {
    obj := NewObjectInstance(proto, 0)
    if data.Location != nil {
        obj.Location = *data.Location
    } else {
        // backward compat: infer from Locate
        if data.Locate == 0 {
            obj.Location = LocInventoryPlayer("") // name filled in by caller
        } else if data.Locate > 0 {
            slot, _ := cWearPosToGoSlot(data.Locate - 1)
            obj.Location = LocEquippedPlayer("", slot) // name filled in by caller
        }
    }
    return obj
}
```

---

## 8. Recommended Implementation Order

1. **Phase A** — Fix `ExtractObject` equipped branches (smallest change, fixes real bug)
2. **Phase D** — Harden `MoveObject` edge cases (container cycle, mob equip duplication, stranded location)
3. **Run full test suite** — verify all object movement tests pass
4. **Phase C** — Remove dual `RoomVNum` tracking
5. **Phase B** — Save/load Location preservation (touches disk format, do last)
6. **Audit** — Search for remaining direct `Inventory.AddItem` / `Equipment.Equip` calls from non-test code and migrate to `MoveObject`

---

## 9. Test Checklist

After each phase, run:

```bash
cd /home/zach/darkpawns && go test ./pkg/game/... -v -run TestObjectMovement -count=1
```

Or the full object movement test suite:

```bash
cd /home/zach/darkpawns && go test ./pkg/game -v -run "TestRoom|TestPlayer|TestContainer|TestExtract|TestInventory|TestMoveObject|TestMob|TestLocation|TestSaveLoad" -count=1
```

All tests should pass after Phases A + D. Phase B requires updating the save/load test expectations.
