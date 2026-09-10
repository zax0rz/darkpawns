# Audit Report: act.item.c vs Go Item Interaction Engines

**C file:** `src/act.item.c` (1,789 lines)
**Go file(s):** `pkg/game/act_item_stubs.go` (649 bytes), `pkg/game/inventory.go` (6,488 bytes), `pkg/game/item_consumable.go` (6,486 bytes), `pkg/game/item_container.go` (3,158 bytes), `pkg/game/item_door.go` (5,497 bytes), `pkg/game/item_equipment.go` (8,974 bytes), `pkg/game/item_helpers.go` (17,694 bytes), `pkg/game/item_transfer.go` (12,398 bytes), `pkg/session/cmd_inventory.go` (9,567 bytes), `pkg/session/eat_cmds.go` (8,231 bytes), `pkg/session/use_cmds.go` (6,420 bytes)
**Mapping type:** 1:N
**Functions audited:** 11 C commands / ~15 Go command entrypoints & helpers

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Severe Scrambled Item Type Mappings in Examine Command
- **Location:** `pkg/session/examine.go` inside `itemTypeString()`.
- **C behavior:** In C, `examine` is simply a shortcut that prints description details and automatically opens containers or drink vessels.
- **Go behavior:** Go's detailed `examine` implementation prints full statistics, including an explicit item type label derived from `itemTypeString(item.GetTypeFlag())`. However, the mappings in `itemTypeString()` are completely scrambled and mismatch the internal MUD constants:
  - Internal `ITEM_DRINKCON` (`17`) -> displays text `"trash"` (internal `ITEM_TRASH` is 17 in C, but drink containers are 17 in Go).
  - Internal `ITEM_FOUNTAIN` (`23`) -> displays text `"clock"`.
  - Internal `ITEM_FOOD` (`19`) -> displays text `"jewelry"`.
  - Internal `ITEM_POTION` (`10`) -> displays text `"food"`.
- **Discrepancy:** Massive display discrepancy. If a player examines a loaf of bread, it displays `Type: jewelry`. If they examine a magic potion, it displays `Type: food`. If they examine a dynamic water container, it displays `Type: trash`. If they examine a municipal fountain, it displays `Type: clock`. This completely breaks the usability of the examine output.
- **Severity:** HIGH
- **Type:** DRIFT / BUG

### [FINDING-002]: Omitted Liquid Pouring (`pour`) Command
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/game/item_consumable.go:155` (`doPour()`).
- **C behavior:** In C `act.item.c:1159` (`do_pour`), players can pour liquid out of containers (`pour out`) or transfer liquid between two drink containers, which also correctly carries over any poisoned flags.
- **Go behavior:** The Go game layer correctly ports the entire logic in `doPour()`. However, the command is **completely dead and un-wired**; it is never registered in `pkg/session/commands.go`.
- **Discrepancy:** Players have no way to empty liquid containers, clear out poisoned water, or transfer dynamic liquids, locking away a legacy immersive game mechanic.
- **Severity:** HIGH
- **Type:** STUB

---

## Type & Boundary Vulnerabilities

### [FINDING-003]: Invalid Food Constant Check in `cmdEat`
- **Location:** `pkg/session/eat_cmds.go:31` inside `cmdEat()`.
- **C behavior:** Only objects flagged with `ITEM_FOOD` are edible.
- **Go behavior:** Go's `cmdEat` contains a hardcoded constant check:
  `if item.GetTypeFlag() != 19 { // ITEM_FOOD ... Send("You can't eat THAT!") }`
  As identified in `examine.go`, type flag `19` represents `ITEM_FOOD` internally, but `examine.go:itemTypeString` associates it with `jewelry`. If these types are ever aligned to traditional Diku numbers, this hardcoded check will collapse.
- **Severity:** LOW
- **Type:** DRIFT

---

## Control Flow & Mathematical Fidelity

### [FINDING-004]: Rollback/Corruption Risk on Take Weight Calculations
- **Location:** `pkg/game/item_transfer.go:17` in `canTakeObj()`.
- **C behavior:** In C `act.item.c`, a player cannot pick up an item if it exceeds their carry capacity (`CAN_CARRY_W(ch)` vs `IS_CARRYING_W(ch)`).
- **Go behavior:** Go translates this check to:
  ```go
  if ch.Inventory.GetWeight()+obj.GetWeight() > ch.Inventory.Capacity * 10 {
      w.actToChar(ch, "$p: you can't carry that much weight.", obj, nil)
      return false
  }
  ```
- **Risk:** The scale factor `Capacity * 10` is an arbitrary linear weight estimation rather than referencing the player's true strength attribute carry caps (`CAN_CARRY_W` macro in C uses a strength-indexed table in `constants.c`). This lets weak players carry excessive weights and heavy players get artificially blocked.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Concurrency & Mutex Safety

### [FINDING-005]: Concurrency Data Race in Item Transfers (`performGive`)
- **Location:** `pkg/game/item_transfer.go:303` inside `performGive()`.
- **C behavior:** Strictly single-threaded loop; thread-safe.
- **Go behavior:** When a player gives an item, `performGive` executes checks on the recipient player's inventory properties:
  ```go
  if len(vict.Inventory.Items) >= vict.Inventory.Capacity { ... }
  if obj.GetWeight()+vict.Inventory.GetWeight() > vict.Inventory.Capacity * 10 { ... }
  ```
  These fields are read concurrently on different player session goroutines without acquiring the recipient's lock (`vict.mu`). Because these structures are concurrently modified by combat tickers, other drop/get commands, or regens, this constitutes a classic read/write data race.
- **Severity:** HIGH
- **Type:** CONCURRENCY

---

## Unported Functions

All core item interaction functions from `act.item.c` were successfully ported to Go (either in `pkg/game/` or `pkg/session/`), with the exception of the `pour` command mapping.

---

## Summary

- **Total findings:** 5
- **Critical:** 0
- **High:** 3
- **Medium:** 1
- **Low:** 1
- **Unported functions:** 0

---

## Verification Plan

### Automated Verification
Verify the compilation safety of the item interaction package:
```bash
go build ./pkg/session/...
go build ./pkg/game/...
```
