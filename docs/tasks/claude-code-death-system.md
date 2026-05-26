# Claude Code Task: Fix Death System (DP-439, DP-440, DP-441)

## Scope
Three tightly-coupled bugs in the corpse/death system. All stem from `makeCorpse` in `pkg/game/death.go` creating corpses that the rest of the codebase doesn't recognize.

## Bug 1: IsContainer() is wrong (DP-439) — THE ROOT CAUSE

**File:** `pkg/game/object.go:113`

`ObjectInstance.IsContainer()` checks `o.GetTypeFlag() == 1` but `ITEM_CONTAINER = 15` (in `pkg/game/item_helpers.go:69`). This means **no container in the game works** — not just corpses. Bags, chests, shop inventories, all broken.

**Fix:** Change the check to use the `ItemContainer` constant:
```go
func (o *ObjectInstance) IsContainer() bool {
    return o.GetTypeFlag() == ItemContainer  // was: == 1
}
```

Note: `ItemType.IsContainer()` at `item_helpers.go:94` already does this correctly. Only `ObjectInstance.IsContainer()` is wrong.

## Bug 2: Corpses don't set ITEM_CONTAINER type (DP-439)

**File:** `pkg/game/death.go:698-720` (`makeCorpse`)

`makeCorpse` creates a corpse with `Prototype: nil`. Since `GetTypeFlag()` returns 0 when Prototype is nil, the corpse isn't recognized as a container even after Bug 1 is fixed.

**C source (`src/fight.c`):** `GET_OBJ_TYPE(corpse) = ITEM_CONTAINER`

**Fix:** After creating the corpse, set the type flag. Since `Prototype` is nil, you need to either:
- Option A: Set `corpse.TypeFlagOverride = ITEM_CONTAINER` (if such a field exists), or
- Option B: Create a minimal prototype for corpses, or  
- Option C: Add a `typeFlag` field to `ObjectInstance` that `GetTypeFlag()` checks when Prototype is nil

Check how `ExtraFlagsOverride` works on `ObjectInstance` — there's already a pattern for instance-level overrides. Apply the same pattern for type flag.

## Bug 3: Corpses don't get a decay timer (DP-440)

**File:** `pkg/game/death.go:698-720` (`makeCorpse`)

C source sets `GET_OBJ_TIMER(corpse) = max_npc_corpse_time` (5 ticks) or `max_pc_corpse_time` (10 ticks). Go never sets the timer.

**Fix:** Add corpse timer constants and set the timer in `makeCorpse`:
```go
// In pkg/game/limits.go or constants — from src/config.c:85-86
const (
    MaxNPCCorpseTime = 5
    MaxPCCorpseTime  = 10
)
```

In `makeCorpse`, after creating the corpse:
```go
// Whether the victim is an NPC determines corpse decay rate
// src/fight.c: if (IS_NPC(ch)) GET_OBJ_TIMER(corpse) = max_npc_corpse_time
```

Note: `makeCorpse` doesn't currently receive an `isNPC` parameter. You'll need to add one, or determine NPC status from the inventory/equipment passed in. Check the callers of `makeCorpse` to see what context is available.

## Bug 4: IsCorpse field never set (DP-441)

**File:** `pkg/game/death.go:698-720` (`makeCorpse`)

`CustomData["is_corpse"]` is set to `true`, but `ObjectInstance.IsCorpse` (the typed bool field) is never set. Auto-loot and mortician check the typed field.

**Fix:** Set `corpse.IsCorpse = true` in `makeCorpse`. Also check `GetValue(3)` — C sets `GET_OBJ_VAL(corpse, 3) = 1` as the corpse identifier, which the decay loop checks. Set this too.

## What NOT to touch
- Don't modify the decay loop in `limits_condition.go` — it already checks the right conditions, it just never fires because corpses never match
- Don't change the `moveObjectToContainer` logic — it will work once IsContainer() is fixed
- Don't touch any other `IsContainer()` call sites — they'll all work once the method is fixed

## Verification
After all fixes:
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
4. Verify `IsContainer()` returns true for objects with `TypeFlag == 15`
5. Verify `makeCorpse` produces objects with type=ITEM_CONTAINER, timer set, IsCorpse=true

## C Source Citations
- `src/fight.c — make_corpse()` — corpse creation, type flag, timer
- `src/structs.h:100` — `ITEM_CONTAINER = 15`
- `src/utils.h — IS_CORPSE()` — `GET_OBJ_TYPE(obj) == ITEM_CONTAINER && GET_OBJ_VAL(obj, 3) == 1`
