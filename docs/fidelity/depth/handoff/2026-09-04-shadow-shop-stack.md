# 2026-09-04 — shadow shop stack adjudication

## Scope and ordering

This handoff is committed before the shop evidence and source edits. It is
the Phase 0.3 owner record for the shadow-shop-stack ticket, after the Phase
0 regression harness handoff. The authoritative modeled corpus is 4,758
cases: 4,653 proven/delegated, 54 blocked, and 51 excluded. The separate
surface inventory is 70 rows and 4,926 weighted units: 8 proven-already, 61
blocked, and 1 excluded-with-C-reason.

## Confirmed live boundary

The production startup path creates `systems.ShopManager` in
`pkg/session/manager.go` and passes it to `World.SetShopManager` from
`cmd/server/main.go`. The active session command path in
`pkg/session/shop_cmds.go` calls `World.GetShopByKeeper`. That world helper
only type-asserts the legacy `*game.ShopManager`, so a real keeper in a
parsed `.shp` file is recognized by `shopKeepers` for combat gates but is not
resolved by `list`, `buy`, or `sell`. The result is a live-broken shadow
stack, not a fidelity exclusion (R1/R2/R4/R5e).

The systems manager also has no startup call that loads the authoritative
Circle shop records. The parser currently retains only the keeper and
behavior bitvector, which is sufficient for the existing combat gate but not
for the economy command surface. `data/shops.json` is absent in the checked
out world, so persistence cannot mask the missing boot path.

## Next proof and fix boundary

The next slice will add a C-first keeper vehicle in room 8162 using mob 12100
and probe the live `list` path with `--show-oracle`. If the expected C shop
listing is confirmed against the Go fallback, the fix will be isolated as a
bugfix before any modernization deletion or refactor. It must preserve the
C `.shp` field order, existing save format, and the parser-derived combat
shopkeeper bitvector. Full oracle regression and all repository gates are
required before the shop fix can be considered complete. The broader shop
transaction rows remain blocked until their own C-byte and state proof is
complete; this ticket does not invent unproven economy behavior.
