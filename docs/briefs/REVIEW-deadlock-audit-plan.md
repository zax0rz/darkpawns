# Daeron's Review — pkg/game Deadlock Audit Implementation Plan

**Date:** 2026-06-17
**Source:** Gemini's implementation plan (via The Architect)

---

## Overall Assessment

The plan is correct. The four categories (self-deadlocks, lock-ordering, logic inversions, door state) are accurately identified and the proposed fixes are sound. Proceed with confidence on the core fixes.

## Specific Feedback

### 1. ObjectPool (`pkg/optimization/object_pool.go`)

**Correct.** The `getLocked()` extraction is the right pattern. `TryGet()` calling `Get()` while holding the pool mutex is the textbook self-deadlock.

**Test:** Add a unit test for `TryGet()` — call it, verify it returns the object, not deadlocks. 20 lines, catches the pattern forever after. This is the highest-value test in the batch because the ObjectPool is used across the entire codebase.

### 2. World Core deadlocks (`world.go`, `graph.go`, `house_boot.go`, `house_control.go`)

**Correct.** All five functions (`mobOpenDoor`, `HouseBoot`, `HcontrolBuildHouse`, `HcontrolDestroyHouse`, `HcontrolSetKey`) follow the same pattern: write lock → call getter → getter tries to read lock → deadlock. Accessing maps directly under the held lock is the right fix.

**Risk flag:** The `executeMobCommand` refactor in `world.go` is the most dangerous change. It currently holds `w.mu.RLock()` during command dispatch, and the dispatched methods (open, drop, get) try to acquire locks on the same mutex. Releasing `w.mu.RUnlock()` before the switch dispatch is correct, but this changes the locking semantics for every command handler. Make sure the race detector is clean *after this specific change* — not just at the end. Run `go test -race ./pkg/game/...` immediately after this file is modified, before moving to other components.

**Test:** Add a test that dispatches a command (e.g., `drop`) while the world lock is held, and verifies it completes. This is the regression guard for the executeMobCommand pattern.

### 3. ShopManager lock ordering (`pkg/game/systems/shop_manager.go`)

**Correct.** The ABBA pattern is clear: processBuy locks player → shop, processSell locks shop → player. Establishing player → shop order everywhere is the standard fix.

**Additional issue the plan notes:** processSell does player inventory modifications outside the player lock — data race. Wrapping the entire transaction in the player lock is the right fix.

**Test already in plan:** `TestShopTransactionConcurrentDeadlock` — two goroutines, one buying, one selling, race detector catches the lock ordering. This is sufficient. No additional test needed.

### 4. Logic inversions (`pkg/spells/damage_spells.go`)

**Correct.** Six instances of `!randBool(...)` — removing the `!` restores parity with the C source where rare events are actually rare.

**Don't test these.** One-character changes. The build passing and the game not crashing is sufficient. A test for "does fireball rarely deal ambient damage" is too fragile to be useful.

### 5. Door state (`pkg/game/systems/door.go`, `door_manager.go`)

**Correct.** Adding `initialClosed`/`initialLocked` fields and restoring them in `Reset()` is clean. The lint cleanup in `GetDoorsInRoom()` is trivial.

**Test:** Add tests for `Door.Reset()` and `DoorManager.ResetDoors()` — these are already in the plan. Good.

## Scope Recommendation

**Block on:**
- ObjectPool.TryGet() test
- executeMobCommand dispatch test
- ShopManager concurrent test
- Race detector clean after each component (especially world.go)

**Defer:**
- Additional door tests beyond what's in the plan (nice-to-have, not regression-critical)
- Lint cleanup in door_manager.go (trivial, won't regress)
- Any additional spell tests

## One More Thing

The plan correctly notes that `GetPlayersInRoom()` is called recursively within methods that already hold the lock. Inlining these calls is the right fix, but this is a refactor of working code paths — code that currently doesn't deadlock because it doesn't hit the specific execution order. The danger isn't introducing deadlocks; it's introducing subtle behavioral changes in room scanning. Verify the game's room behavior after this change, not just the race detector.

---

*Daeron out.*
