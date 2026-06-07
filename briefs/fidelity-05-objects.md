# Fidelity Brief 05: Object System

**Date:** 2026-05-27
**Priority:** HIGH — 1,661 objects, core inventory/combat mechanic
**C source:** `src/act.item.c` (1789 lines)
**Go source:** `pkg/session/eat_cmds.go` (298 lines), `pkg/session/cmd_inventory.go`, various game files

---

## Scope

Objects are the game's physical layer — weapons, armor, containers, food, drink, scrolls, potions, wands, staffs, keys, and 1,661 unique items. This brief covers:

1. **`do_get()` / `do_put()`** — container interaction
2. **`do_drop()` / `do_give()`** — transferring objects
3. **`do_wear()` / `do_wield()` / `do_grab()`** — equipping
4. **`do_remove()`** — unequipping
5. **`do_eat()` / `do_drink()` / `do_pour()`** — consumption
6. **`do_use()`** — using scrolls/potions/wands/staffs
7. **Object types** — weapon, armor, container, drink, food, scroll, wand, staff
8. **Object values** — value[0]-value[3] meaning per type

---

## What to Verify

### 1. `do_get()` — Picking Up Objects

**C source** (act.item.c:346):
```c
ACMD(do_get)
{
  // "get all" / "get all.room" / "get <object>"
  // "get <object> <container>"
  // Check weight: can't carry more than CAN_CARRY_W(ch)
  // Check count: can't carry more than CAN_CARRY_N(ch)
  // Check container: must be open, not locked
  // Messages: "You get $p." / "$n gets $p." / "$n gets $p from $P."
}
```

**Check:**
- Weight limit check
- Item count limit check
- Container interaction (get from open container)
- Messages for get from room vs get from container

### 2. `do_put()` — Putting in Containers

**C source** (act.item.c:77):
```c
ACMD(do_put)
{
  // "put <object> <container>"
  // Container must be open
  // Container must have capacity
  // Object weight + container weight <= container capacity
  // Can't put containers in containers (usually)
  // Messages: "You put $p in $P." / "$n puts $p in $P."
}
```

**Check:**
- Container capacity check
- Nested container restriction
- Messages

### 3. `do_drop()` — Dropping Objects

**C source** (act.item.c:529):
```c
ACMD(do_drop)
{
  // "drop all" / "drop <object>" / "drop gold <amount>"
  // Can't drop while fighting
  // Messages: "You drop $p." / "$n drops $p."
  // Corpse drops on death
}
```

**Check:**
- Fighting restriction
- Gold drop (fractional amounts)
- Messages

### 4. `do_wear()` — Equipping

**C source** (act.item.c:1584):
```c
ACMD(do_wear)
{
  // "wear <object>" / "wear all"
  // Check wear position (WEAR_BODY, WEAR_HEAD, WEAR_ARMS, etc.)
  // Check if position is already occupied
  // Apply object affects (AC, stat bonuses, etc.)
  // Messages: "You wear $p." / "$n wears $p."
  // Armor class calculation
}
```

**Check:**
- Wear position mapping (WIELD, HOLD, WEAR_BODY, WEAR_HEAD, etc.)
- Occupied position handling (swap or deny?)
- Object affect application (AC, stats)
- Messages

### 5. `do_wield()` — Wielding Weapons

**C source** (act.item.c:1661):
```c
ACMD(do_wield)
{
  // Must be weapon type
  // Two-handed check
  // Skill check for weapon type (if applicable)
  // Messages: "You wield $p." / "$n wields $p."
}
```

**Check:**
- Weapon type validation
- Two-handed weapon handling
- Weapon skill check

### 6. `do_eat()` — Consuming Food

**C source** (act.item.c:1035):
```c
ACMD(do_eat)
{
  // Object must be FOOD type
  // Poison check: if food is poisoned, apply poison affect
  // Food value[1] = food value (hunger points restored)
  // Food value[2] = poison chance
  // Messages: "You eat $p." / "$n eats $p."
  // Apply hunger restoration
}
```

**Check:**
- Food type validation
- Poison chance
- Hunger restoration amount
- Messages

### 7. `do_drink()` — Drinking from Containers

**C source** (act.item.c:895):
```c
ACMD(do_drink)
{
  // Object must be DRINK_CON type
  // drink value[1] = liquid amount remaining
  // drink value[2] = liquid type
  // drink value[3] = poison flag
  // Liquid types have different effects (water restores thirst, alcohol intoxicates, etc.)
  // Messages: "You drink from $p." / "$n drinks from $p."
  // Intoxication system
}
```

**Check:**
- Liquid type effects
- Intoxication system
- Poison from drink containers
- Messages

### 8. `do_use()` — Using Items

**C source** (act.item.c:895):
```c
ACMD(do_use)
{
  // use scroll — cast spell from scroll
  // use potion — drink potion (cast spell)
  // use wand — point wand at target (cast spell)
  // use staff — point staff at room (area spell)
  // Check charges remaining
  // Messages: "You use $p." / "$n uses $p."
}
```

**Check:**
- Scroll/potion/wand/staff handling
- Charge tracking
- Spell casting from items
- Messages

---

## Object Value Fields

Each object type uses value[0]-value[3] differently:

| Type | value[0] | value[1] | value[2] | value[3] |
|------|----------|----------|----------|----------|
| WEAPON | damage num | damage size | weapon type | (unused) |
| ARMOR | AC | (unused) | (unused) | (unused) |
| CONTAINER | max weight | flags | key vnum | (unused) |
| DRINK_CON | capacity | liquid remaining | liquid type | poison flag |
| FOOD | food value | (unused) | poison flag | (unused) |
| SCROLL | spell level | spell 1 | spell 2 | spell 3 |
| WAND | spell level | max charges | current charges | spell |
| STAFF | spell level | max charges | current charges | spell |

**Check:** Does the Go code read these fields correctly for each type?

---

## Implementation Notes

- Object data is loaded from `lib/world/obj/` files (CircleMUD format)
- Object affects are applied via `apply_object_affects()` on wear
- Object decay: some objects have timers, decay on ground
- Container flags: CONTAINER_CLOSEABLE, CONTAINER_PICKPROOF, CONTAINER_CLOSED, CONTAINER_LOCKED

---

## Verification

1. Pick up objects from room — verify weight/count limits
2. Put objects in container — verify capacity
3. Drop objects — verify messages
4. Wear armor — verify AC change
5. Wield weapon — verify damage
6. Eat food — verify hunger restoration
7. Drink from drink container — verify thirst/intoxication
8. Use scroll — verify spell casting
9. Test locked container — verify key requirement
10. Run `go test ./pkg/game/...`
