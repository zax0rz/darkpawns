# Subagent Brief: handler.c + db.c Fidelity Fixes

**Objective:** Fix 12 port fidelity issues from the handler.c and db.c C audit. 7 HIGH, 3 MEDIUM, 2 LOW.

**Working directory:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**Before committing:** Run `go build ./... && go vet ./...` to verify every fix.

**Source of truth:** Each fix references C source line numbers. Read the C code before modifying Go — the C behavior is canonical.

---

## HIGH Priority (fix first)

### Fix 1: DP-365 — ExtractObject Missing Container Recursion

**File:** `pkg/game/world_object.go` — `ExtractObject()`
**C source:** `src/handler.c:1020-1024`

**Problem:** When a container (bag, corpse, etc.) is destroyed, C recursively extracts all contents. Go only removes the container itself — objects inside stay orphaned in `w.objectInstances` forever.

**Fix:** Before removing the container from the global map, iterate `obj.Contains` and call `ExtractObject` on each child:

```go
// Recursively extract contents — handler.c:1020-1024
for child := obj.Contains; child != nil; child = child.NextContent {
    w.ExtractObject(child)
}
```

**Verify:** Check that `obj.Contains` field exists on the object type. If the linked list uses a different field name (e.g., `Contents`, `Carrying`), use that. Read the struct definition first.

---

### Fix 2: DP-368 — CharTransfer Bypasses Room Light Tracking

**File:** `pkg/game/world_movement.go` — `CharTransfer()`
**C source:** `src/handler.c:520-521, 541-543`

**Problem:** Every teleport, gate, recall, or summon bypasses room light counter tracking. Rooms permanently drift dark/light.

**Fix:** In `CharTransfer`, before moving the character:
1. Check if character has a lit light source equipped (check equipment for light-producing items)
2. If so, decrement source room's light counter
3. After moving, increment destination room's light counter

Look at how `MovePlayer` in `pkg/game/world.go` handles light tracking — it already does this for walking. Port the same logic to `CharTransfer`.

---

### Fix 3: DP-366 — ExtractPendingChars Doesn't Drop Inventory/Gear

**File:** `pkg/game/char_mgmt.go` — `ExtractPendingChars()`
**C source:** `src/handler.c` — `extract_char_final()`

**Problem:** When players are extracted, only their light source is unequipped. All other equipment and carried items are lost (never dropped to room floor). Mobs are also not processed by the delayed extraction queue.

**Fix:**
1. In `ExtractPendingChars`, when processing a player: unequip ALL equipment slots (not just light), then move all carried objects to the room via the equivalent of `obj_to_room`
2. Add mob extraction support: check for mobs with an extraction flag and process them similarly

Read the existing `ExtractPendingChars` function carefully to understand the current flow before modifying.

---

### Fix 4: DP-369 — Equipment Anti-Zap Bypass

**File:** `pkg/game/equipment.go` — `equip()`
**C source:** `src/handler.c` — `equip_char()`

**Problem:** Go's `equip()` has no alignment or class validation. Players/mobs can equip restricted items without being zapped.

**Fix:** At the top of `equip()`, before equipping, add:
1. Alignment check: if item has `ITEM_ANTI_EVIL` and player is evil, or `ITEM_ANTI_GOOD` and player is good, or `ITEM_ANTI_NEUTRAL` and player is neutral → zap (send message, return item to inventory, return)
2. Class check: if item has class restriction and player's class doesn't match → zap

Look for constants like `ItemAntiEvil`, `ItemAntiGood`, `ItemAntiNeutral` and the alignment check functions. The C source shows the exact logic at `src/handler.c`.

---

### Fix 5: DP-373 — Spawner 'R' Command Leaks Orphaned Instances

**File:** `pkg/game/spawner.go` — `ExecuteZoneReset()` case "R"
**C source:** `src/db.c` — zone reset 'R' command

**Problem:** The 'R' (remove) command in zone resets only removes items/mobs from the spawner's local tracking. It never calls `ExtractObject` or `ExtractMob`, leaving orphaned entries in global maps.

**Fix:** In case "R", after removing from local tracking, also call `w.ExtractObject(obj)` or `w.ExtractMob(mob)` to clean up global state. This requires access to the World instance from the spawner — check how other spawner methods access it.

---

### Fix 6: DP-375 — High-Level Mob Stat Boosts Missing

**File:** `pkg/game/mob.go` — `NewMob()`
**C source:** `src/db.c:1053-1062`

**Problem:** In C, mobs above level 15 get random stat boosts (+0 to +7 per stat). Go sets stats strictly from prototype — high-level mobs are significantly weaker.

**Fix:** After setting base attributes in `NewMob()`, add:

```go
if mob.Level > 15 {
    statmod := mob.Level - 15
    // number(0, statmod) in C = rand.Intn(statmod + 1) in Go
    mob.RealAbils.Str += min(rand.Intn(statmod+1), 7)
    mob.RealAbils.Intel += min(rand.Intn(statmod+1), 7)
    mob.RealAbils.Wis += min(rand.Intn(statmod+1), 7)
    mob.RealAbils.Dex += min(rand.Intn(statmod+1), 7)
    mob.RealAbils.Con += min(rand.Intn(statmod+1), 7)
    mob.RealAbils.Cha += min(rand.Intn(statmod+1), 7)
}
```

Check the actual struct field names — they might be `Str`/`Int`/`Wis` etc. or might use different naming. Read the struct first.

---

### Fix 7: DP-376 — ITEM_RARE Affect Mutations Missing

**File:** `pkg/game/spawner.go` — `SpawnObject()`
**C source:** `src/db.c:1899-1925` — `init_rare()`

**Problem:** Rare items (flagged `ITEM_RARE`) should have random stat variance. Go ignores the flag entirely.

**Fix:** After spawning an object instance, check if the prototype has the `ITEM_RARE` flag. If so, iterate the object's applies and with 20% probability modify:
- `APPLY_DAMROLL` / `APPLY_HITROLL`: +/- 1
- `APPLY_AC`: +/- 5

Port the `init_rare` logic from C. Look for how applies are stored on objects in Go (likely a slice of Apply structs with Type and Value fields).

---

## MEDIUM Priority

### Fix 8: DP-367 — Summoning Circle Check Unported

**File:** `pkg/game/world_movement.go`
**C source:** `src/handler.c:527` — `circle_check()`

**Problem:** Portal summoning circles (VNum 9) never decay. Portal spells bypass gameplay constraints.

**Fix:** Port `circle_check` — search room contents for an object with VNum 9 (summoning circle), decrement its timer, return whether it's active. Call this during room transitions for relevant characters.

This is lower priority because it affects a niche mechanic. Skip if it requires significant new infrastructure.

---

### Fix 9: DP-370 — Synchronous Mob Extraction Concurrency Risk

**File:** `pkg/game/world.go` — `ExtractMob()`
**C source:** `src/handler.c` — delayed extraction queue

**Problem:** Mob extraction happens synchronously, which can cause race conditions during game tick iteration.

**Fix:** Route mob extraction through the same delayed pending queue used for players. Set an extraction flag on the mob, process it on the next tick. This matches C's approach.

This requires understanding the existing extraction queue infrastructure. Read `ExtractPendingChars` and the mob flag system first.

---

### Fix 10: DP-371 — Mob Gold Spawns No Variance

**File:** `pkg/game/mob.go` — `NewMob()`
**C source:** `src/db.c:1766-1775`

**Problem:** Mob gold is set exactly to prototype value. C randomizes by +/- 20%.

**Fix:** After setting gold, if > 0:
```go
if mob.Gold > 0 {
    if rand.Intn(2) == 0 {
        mob.Gold += rand.Intn(20) * mob.Gold / 100
    } else {
        mob.Gold -= rand.Intn(20) * mob.Gold / 100
    }
}
```

---

## LOW Priority (skip if time-constrained)

### Fix 11: DP-374 — Zone Command Boundary Checks

**File:** `pkg/game/spawner.go`
**C source:** `src/db.c` — `renum_zone_table()`

Defensive hardening only. Add boot-time VNum validation. Skip if it requires major refactoring.

---

### Fix 12: DP-372 — Lock Ordering Audit

**File:** `pkg/game/save.go` — `DeserializeWorld()`

Audit only — verify lock order is `w.mu → mob.mu → obj.mu`. Add a comment documenting the canonical order. No behavior change expected.

---

## Execution Order

1. **DP-365** (container recursion) — memory leak, highest impact
2. **DP-373** (spawner 'R' leak) — same class of bug, related code
3. **DP-375** (mob stat boosts) — combat balance
4. **DP-376** (rare item variance) — loot economy
5. **DP-369** (anti-zap bypass) — equipment validation
6. **DP-368** (light tracking) — room state drift
7. **DP-366** (inventory drop) — item persistence
8. **DP-371** (gold variance) — simple one-liner
9. **DP-367** (summoning circle) — niche mechanic
10. **DP-370** (mob extraction queue) — concurrency
11. **DP-374** (boundary checks) — defensive
12. **DP-372** (lock audit) — documentation

## After All Fixes

Run the full verification suite:
```bash
go build ./... && go vet ./... && go test ./...
```

If any test fails, fix it before committing. Each fix should be a separate commit with a message referencing the DP issue (e.g., "fix: recursive container extraction in ExtractObject (DP-365)").
