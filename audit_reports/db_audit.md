# Audit Report: db.c vs pkg/game/ & pkg/parser/

**C file:** `src/db.c` (3,220 lines)
**Go file(s):** `pkg/game/world.go` (1,376 lines), `pkg/game/world_zone.go` (172 lines), `pkg/game/spawner.go` (674 lines), `pkg/game/serialize.go` (108 lines), `pkg/game/save.go` (697 lines), `pkg/game/mob.go` (830 lines)
**Mapping type:** 1:N
**Functions audited:** ~30

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Total Omission of `ITEM_RARE` Affect Mutations (High Severity)
- **Location:** `pkg/game/spawner.go` in `SpawnObject()` / `src/db.c:1899-1925` / `init_rare()`.
- **C behavior:** When spawning an object from a prototype, if the object is flagged as rare (`IS_OBJ_STAT(obj, ITEM_RARE)`), C's `read_object()` invokes `init_rare(obj)`. This function loops through all active item affects and, with a 20% probability, applies mechanical modifications:
  - `APPLY_DAMROLL` / `APPLY_HITROLL`: increased/decreased by `1`.
  - `APPLY_AC`: increased/decreased by `5`.
- **Go behavior:** Go's object spawning completely ignores the `ITEM_RARE` flag (bit 24) and never invokes any equivalent of `init_rare`.
- **Discrepancy:** Rare items spawned in the Go MUD have 100% identical stats to common items. They completely lack the mechanical and attribute variance designed in legacy specifications, degrading the MUD's loot and reward economy.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-002]: Omission of Mob Level-Scaled Attribute Boosts
- **Location:** `pkg/game/mob.go:75` in `NewMob()` / `src/db.c:1053-1062` / `parse_simple_mob()`.
- **C behavior:** When loading a simple mobile template, if the mob's level is greater than 15, its base attributes (Str, Int, Wis, Dex, Con, Cha) are dynamically incremented using level-scaled random offsets:
  ```c
  if (GET_LEVEL(mob_proto + i)>15) {
     int statmod = GET_LEVEL(mob_proto + i)-15;
     mob_proto[i].real_abils.str += MIN(number(0,statmod), 7);
     mob_proto[i].real_abils.intel += MIN(number(0,statmod), 7);
     ...
  }
  ```
- **Go behavior:** Go's `NewMob()` sets a mob's attributes strictly based on prototype/defaults, completely omitting the level-scaled boosts for high-level monsters.
- **Discrepancy:** High-level mobs (level > 15) in the Go port are significantly weaker than designed in legacy specifications, completely disrupting combat balance and encounter difficulty.
- **Severity:** HIGH
- **Type:** DRIFT

### [FINDING-003]: Spawner 'R' Command Resource Leak & Orphaned Instances
- **Location:** `pkg/game/spawner.go:440` in `ExecuteZoneReset()` case "R".
- **C behavior:** In legacy MUD resets, the 'R' command removes a specified object or mobile from a room AND completely extracts it from the world (freeing it and recursively extracting its carried/equipped gear).
- **Go behavior:** The Go spawner case "R" only invokes:
  ```go
  s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
  // or
  s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
  ```
  These methods only delete the items/mobs from the spawner's local tracking slices (`s.roomObjects`, `s.objInstances`, etc.). They **never** call `ExtractObject` or `ExtractMob` on them!
- **Discrepancy:** The deleted mobs and items are left orphaned in the global `w.objectInstances` and `w.activeMobs` maps, alongside their inventories and equipment. This creates a permanent, severe resource and memory leak on every zone reset.
- **Severity:** HIGH
- **Type:** BUG

### [FINDING-004]: Predictable Mob Gold Spawns (No Random Gold Variance)
- **Location:** `pkg/game/mob.go:75` in `NewMob()` / `src/db.c:1766-1775`.
- **C behavior:** Mobs spawned with gold have their carrying amount randomized by +/- 20%:
  ```c
  if (GET_GOLD(mob)) {
     if (!number(0,1))
      GET_GOLD(mob)+=(number(1,20)*GET_GOLD(mob))/100;
     else
      GET_GOLD(mob)-=(number(1,20)*GET_GOLD(mob))/100;
  }
  ```
- **Go behavior:** Go's `NewMob` sets gold exactly to the prototype value.
- **Discrepancy:** Mob gold drops are completely predictable, removing the organic variance of the MUD economy.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Type & Boundary Vulnerabilities

### [FINDING-005]: Unprotected Zone Command Boundary Checks
- **Location:** `pkg/common/` / `pkg/parser/` and `pkg/game/spawner.go:374` inside loop cases.
- **C behavior:** Zone command loader in `renum_zone_table()` validates room/mob/object VNums immediately at boot, replacing invalid command target indices with `'*'` to disable them and prevent run-time segmentation faults.
- **Go behavior:** The Go spawner relies on interface checks and prototype mapping lookups (`s.world.GetObjPrototype(cmd.Arg1)`) during runtime zone resets.
- **Risk:** If a zone command points to a deleted or corrupt VNum, it logs warnings but continues. If any pointer assertion or downstream function expects valid indices, it can trigger run-time nil pointer dereference panics.
- **Severity:** LOW

---

## Concurrency & Mutex Safety

### [FINDING-006]: Concurrency Safety in Dynamic World State Restores
- **Location:** `pkg/game/save.go:490` in `DeserializeWorld()`.
- **C behavior:** Synchronous single-threaded boot/load.
- **Go behavior:** `DeserializeWorld` locks `w.mu` to restore doors, mob HP/max HP, and ground items.
- **Impact:** Good. The Go port properly locks `w.mu` and `mob.mu` to synchronize state changes, avoiding concurrent write issues. However, since zone dispatcher tickers or players can connect during boot/dispatcher startups, ensuring lock ordering is strictly maintained is crucial to prevent deadlocks.
- **Severity:** LOW

---

## Unported Functions / Systems

The following legacy C functions from `db.c` have no Go counterparts:

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| DNS Cache | 1780-1881 (db.c) | `boot_dns`, `save_dns_cache`, `get_host_from_cache`, and `add_dns_host` managed a hash table IP-to-hostname resolution file to speed up MUD syslog outputs. | NO (handled via standard sockets/telnet listeners in modern engines) |
| check_dst | 3177 (db.c) | Determines if machine knows daylight saving time and prints system logs on DST issues. | NO (handled natively by Go `time` package) |

---

## Summary

- **Total findings:** 6
- **Critical:** 0
- **High:** 3
- **Medium:** 1
- **Low:** 2
- **Unported features:** 2
