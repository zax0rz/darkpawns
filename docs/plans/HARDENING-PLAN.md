# Phase 7 — MoveObject Hardening (GPT-5.5 Post-Review)

Based on GPT-5.5 post-modernization review. All 6 modernization phases complete; this pass seals the invariant enforcement.

## 7A. Fix MoveObject rollback (movement.go)

**Issue:** On attach failure, rollback re-attach discards errors. Object can end up half-detached in limbo.
**Fix:** If rollback attach also fails, log the error and set Location to ObjNowhere. Don't silently swallow.

## 7B. Unexport Inventory.AddItem/RemoveItem + Equipment.Equip/Unequip

**Issue:** Direct call sites bypass Location invariants.
**Fix:** Rename to `addItem/removeItem` (unexported). All callers go through MoveObject or World helpers.
**Affected:** world.go (giveItem), mail.go, skills.go, spec_procs2.go, objsave.go, item_equipment.go

## 7C. Replace AddItemToRoom() with MoveObjectToRoom()

**Issue:** AddItemToRoom() just appends to roomItems, doesn't set Location or detach.
**Fix:** Replace all AddItemToRoom() call sites with MoveObjectToRoom(). Remove or deprecate AddItemToRoom.
**Affected:** death.go (corpse/ash placement, ~3 sites), world.go

## 7D. Container cycle prevention (movement.go)

**Issue:** A can contain B can contain A.
**Fix:** In attachObjectLocked for ObjInContainer, walk the container chain (container → container.Container) and reject if cycle detected. Max depth ~10 to prevent infinite loops.

## 7E. Fix mob equipment semantics (movement.go)

**Issue:** Mob equipment path manually does AddToInventory + Equipment[slot] = obj, leaving items in BOTH inventory and equipment. Player path uses Equipment.Equip() which handles swap/unequip.
**Fix:** Either: (a) make mob equipment use the same pattern as players with proper remove-from-inventory-on-equip, or (b) accept that mobs hold equipped items in inventory too (C behavior) but document it. Option (a) is cleaner.

## 7F. Cleanup

- Delete `pkg/game/act_item.go.bak`
- Document lock ordering: `w.mu → Equipment.mu → Inventory.mu`
- Tighten Validate() to reject impossible field combinations per state
