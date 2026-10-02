# Phase 4 E3: forced-rent roundtrip

Base: `origin/main` 44bbb7142, after #1756. The entry train is merged. Browser
name routing, identity consumers, and wizard file-editor loaded-inventory
ownership remain explicit frontier items, not completed by the entry PR or this
train. `entry.login-restrictions` still waits for D6.

## Inventory roundtrip

`idle.rent-roundtrip-inventory` is unit-green. The live vehicle
`lifecycle-idle-rent-inventory` also matches C at seed 1. C force-rents before
extraction (`src/limits.c:438-451`); `Crash_rentsave` saves and then frees the
objects (`src/objsave.c:912-956`). Re-entry loads and auto-equips the saved tree
(`src/interpreter.c:2184-2194; src/objsave.c:557,440`). `equip_char` reapplies the
owner's ITEM_TAKE_NAME description (`src/handler.c:720-728`). These source lines
were read while writing this proof.

Two reachable defects were repaired. The idle retirement saved the objects but
left their old instances registered, so reload doubled the world object count.
After the save, cleanup now detaches and frees the concrete retired body's tree.
The store restore also bypassed the live equipment naming step; both modern
item lists and legacy equipment maps now share that step. The existing save
format and its TAKE_NAME override omission are unchanged: auto-equipping
reconstructs the name. An old test expecting the prototype description on an
equipped reload was incorrect and is replaced by the C-backed expectation.

The real SQLite test carries a bag with bread and wears a TAKE_NAME tunic,
force-rents, removes the body, returns through the real menu reload, and checks
the restored topology, slot, exact description and world count. It avoids
name-based object ownership by using canonical object moves in its setup. The
live vehicle retains C's nested inventory and named equipment blocks; an inside
void room isolates D5 weather. The manifest's status reflects the stronger
hidden-state unit proof, rather than claiming that transcript equality proves
absence of leaked objects.

Evidence root:
`~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-switch-idle-proofs/`.
- Cleanup: `inventory-before.log`, `inventory-reverted.log`,
  `inventory-restored.log`: 0 → 1 → 0. Reverting cleanup fails on six world
  objects where the three restored objects are required.
- Naming: `taken-name-before.log`, `taken-name-reverted.log`,
  `taken-name-restored.log`: 0 → 1 → 0. Disabling the restore naming calls fails
  on the equipped owner description in both the store and roundtrip tests.
- Baseline live output: `rent-baseline.txt` shows the tunic description red.
  The fixed two-vehicle run is `dp-1371-rent-repaired`: 2/2 CLEAN.

Other readers: the shared naming helper is used by live equip, modern restore
and legacy restore; carried objects and non-TAKE_NAME overrides keep their
existing behavior. SQLite and the menu consume the retained object data before
retirement cleanup. The cleanup uses the existing concrete-owner discard helper
and also clears equipment references, avoiding same-name lookups after removal.
The D5 room-index/weather differences are still held separately. The legal-quit
path (`pkg/session/cmd_inventory.go:101-107`) also saves a rented tree without
calling `ExtractRentedObjects`; its transient objects need the same ownership
audit in a follow-up. This train claims the forced-rent path only.

## Equipped NORENT roundtrip

`idle.rent-roundtrip-equipped-norent` is unit-green. The separate live vehicle
`lifecycle-idle-rent-norent` matches C at seed 1. Its warmup wears shipped NORENT
cloak 4291 and carries NORENT fruit 4320; after forced-rent and relogin, C has
empty inventory and only the retained named tunic. The SQLite proof also keeps
a rentable nested bag as a control and requires exactly the three rentable
instances in the restored world. This prevents a zero-object fixture from
passing a destructive "fix" that discards everything.

`norent-before.log`, `norent-reverted.log`, `norent-restored.log` retain 0 → 1 → 0:
the negative control keeps NORENT objects through the filter and fails the
restored count (five instead of three). The failure is an assertion, not a
build error or timeout. C filters worn objects through the carried-list
extraction (`src/objsave.c:730-760`) before writing the rent file (:912-956).

Other readers: equipped and carried object locations use the same extraction
API, while nested contents are retained through the existing store tree format.
The preserved tunic guards against filtering all equipment; the inventory case
owns nested restore and the TAKE_NAME boundary. This commit adds proof only,
and changes neither the rental policy nor the oracle/harness.
