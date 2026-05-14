# Hardening — Subagent 2

You are hardening the Dark Pawns Go codebase.
Repository: `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**RULES:**
- Run `go build ./... && go vet ./... && go test ./pkg/game/ ./pkg/session/` after every change
- If a build breaks, revert immediately
- Commit each fix separately with message `fix: DP-X — description`
- Small, surgical changes only. No refactors beyond what's specified.

## Tasks

### DP-58: Unexport Inventory.AddItem/RemoveItem
`pkg/game/inventory.go` has exported `AddItem` and `RemoveItem` methods.
- These should be unexported (`addItem`/`removeItem`) since they're called within the game package
- But check cross-package callers first: `grep -rn "\.AddItem\|\.RemoveItem" --include="*.go" pkg/ | grep -v "inventory.go\|_test\|// "`
- If called from other packages (session, scripting), keep exported versions or add unexported internal versions
- The goal: internal mutation methods should not be part of the public API

### DP-60: Fix mob equipment semantics
`pkg/game/mob.go` line 261: `EquipItem` doesn't check if the slot is already occupied.
- If slot is occupied, the old item should be unequipped first (returned to inventory)
- Player equipment (`equipment.go`) does this correctly — use it as reference
- Add slot-occupied check before equipping

### DP-61: Cleanup
- Find and delete any `.bak` files: `find . -name "*.bak" -not -path "./.git/*" -delete`
- Check for stale lock-related documentation that references old locking patterns

### DP-64: Replace AddItemToRoom with MoveObjectToRoom
`AddItemToRoom` just appends to roomItems without setting Location or detaching from old location.
`MoveObjectToRoom` does it properly (detach + attach + set Location).
- Find all callers: `grep -rn "AddItemToRoom" --include="*.go" pkg/ | grep -v "_test\|// \|interface\|Scriptable\|scripting"`
- For each caller in the game package, check if the object has a current location that needs detaching
- If the object is freshly created (no old location), AddItemToRoom is fine — leave it
- If the object might be moving from somewhere else, replace with MoveObjectToRoom
- Do NOT touch the scripting adapter — that's a different interface
- The comment in world.go says "Prefer MoveObjectToRoom for new code" — that's the guidance

### DP-65: Container cycle prevention
`pkg/game/object.go` line 148: `AddToContainer` adds an object to a container.
- No cycle detection: A can contain B can contain A → infinite recursion
- Add cycle detection: walk the container chain from the target container up to root
- If we encounter `obj` itself in the chain, reject the add (return false)
- Max depth: 10 (practical limit for nested containers in a MUD)
- Only applies to ObjInContainer → ObjInContainer chains

## Final
After all changes, report: what was fixed, what was verified safe, any issues found.
