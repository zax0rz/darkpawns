# Phase 1.5 — Fix Known Gaps in mobact.go + mobprogs.go

Per v4-pro-port-plan.md, Phase 1 = 1a (mobprog+mobact) + 1b (spec batch 1) + 1c (spec batch 2).
Phase 1a is done (committed). Phase 1.5 = fix the stubs in 1a before moving to 1b.

## Context

**C src:** `src/mobprog.c` (646 lines), `src/mobact.c` (408 lines)
**Go files:** `pkg/game/mobprogs.go` (545 lines), `pkg/game/mobact.go` (356 lines)
**Branch:** main, commit 2db73a5 (but rebased, may have different hash now)
**Build:** `go build ./... && go vet ./...` — verify first

**Key Go interfaces used in these files:**
- `Player` has: `Gold` (int), `Level` (int), `SendMessage(msg string)`, `GetName() string`, `Inventory.AddItem(obj *ObjectInstance)`, `Inventory` (check for `Remove`)
- `MobInstance` has: `RemoveFromInventory(obj *ObjectInstance)`, `SetStatus(s string)`, `GetFighting()`, `SetFighting()`, `StopFighting()`, `GetPosition() int`, `GetShortDesc() string`, `HasFlag(flag string) bool` (currently returns false)
- `ObjectInstance` has: `GetTypeFlag() int` (returns int, use literals: 15=ITEM_FOOD, 17=ITEM_DRINKCON), `GetWeight() int`, `GetCost() int`, `GetShortDesc() string`
- `World` has: `Rooms map[int]*parser.Room`

## Fix List (in order, build+veting after each)

### 1. `HasFlag(string) bool` in pkg/game/mob.go — wire it
Current: `return false`
Need: iterate `m.Prototype.ActionFlags` ([]string), return true if any match.
Flags called in mobact.go (grep for them): AGGRESSIVE, SENTINEL, SCAVENGER, MEMORY, HELPER, AGGR24, STAY_ZONE, HUNTER, MOB_AGGR_EVIL, MOB_AGGR_NEUTRAL, MOB_AGGR_GOOD
Also: SPEC, NPC (but only SPEC and NPC are used for non-string checks — check context)

### 2. `GetAllMobs()` in pkg/game/mobact.go — implement
Current: returns nil
Need: iterate w.Rooms, collect all mobs from each room into []*MobInstance
Check if there's `w.GetMobsInRoom(vnum)` or similar. Or iterate rooms directly.

### 3. `extractObj` Player carrier path in pkg/game/mobprogs.go — fix
Current: TODO comment for Player case
Need: if obj.Carrier is *Player, check for `Inventory.Remove()` or `Inventory.RemoveItem()`. If not available, add a simple method to Player for removing objects from inventory.

### 4. Verify stc (mobprogs.go) works
`stc(msg, ch)` calls `ch.SendMessage(msg)` — verify Player.SendMessage exists. If not, change to `w.sendToChar(ch, msg)` or similar.

### 5. Check `scanForMob` in mobact.go if used
May need a room-scoped helper to find a player/mob in the room.

## Commit Rule
One commit per fix. `go build ./... && go vet ./...` after each. Push at end.

## Verify This First
Before fixing: `go build ./... && go vet ./...` — confirm the files compile as-is after the rebase.
