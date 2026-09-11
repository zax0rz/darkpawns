# Shop live-inventory gate — 2026-09-10

Branch: `glm/depth-shop-live-inventory`, based on `main` at `6c054d5c9`.

## Disposition

The `shops.live-inventory-gate` D4 debt is converted to
`oracle-green-multiseed`. Two companion rows make the proof boundaries
explicit: a stocked control and a cost/visibility filter control. No shop
consolidation or other modernization work is included.

## Actual call paths

C dispatches `list` from `src/interpreter.c:947` through `special()` at
`src/interpreter.c:1407-1440`, which invokes the room mob's
`SPECIAL(shop_keeper)`. `src/shop.c:942-982` resolves the keeper/shop and
room, then `shopping_list()` at `src/shop.c:877-925` calls `is_ok()` before
walking `keeper->carrying`. `is_ok()` at `src/shop.c:111-137` performs the
open-hours and `is_ok_char()` checks; `is_ok_char()` at `src/shop.c:74-108`
performs keeper visibility, God, alignment, NPC, and class gates. The list
loop admits only `CAN_SEE_OBJ(ch,obj)` and `obj->obj_flags.cost > 0` at
`src/shop.c:896-917`, groups with `same_obj()` (`src/shop.c:293-320`), and
formats rows with `list_object()` (`src/shop.c:845-875`). C prepends each G
object through `obj_to_char()` at `src/handler.c:559-566`.

The corresponding Go command is registered at `pkg/session/commands.go:165-167`
and executes `cmdList()` in `pkg/session/shop_cmds.go`. Keeper association is
resolved by `findShopKeeperAndMobInRoom()` using
`World.GetMobsInRoom()`/`GetShopByKeeper()`. The corrected path performs the
open/visibility/trade gate, then walks `MobInstance.Inventory`, applies cost
and `game.CanSeeObject()`, groups equivalent live objects, applies C keyword
matching, and formats the C header/rows. Static `Shop.SellTypes` is used only
to identify `shop_producing()`/`Unlimited`, never as the inventory source.

R5e found a sibling defect in the shared Go `canSeeObject()` helper: C's
`CAN_SEE_OBJ` is `MORT_CAN_SEE_OBJ || PRF_HOLYLIGHT` (`src/utils.h:538-542`),
not an immortal-level bypass. The bypass was removed and the holy-light
boundary is covered by `TestFidelityCanSeeObjectInvisibility`. The C integer
order in `buy_price()`/`sell_price()` (`src/shop.c:460-468,638-651`) was also
corrected in `Shop.BuyPrice`/`Shop.SellPrice`, with
`TestShopPriceUsesCIntegerOrder`.

## Fixtures and proof

`shop-live-inventory-gate.txt` quiets zone 121's authored resets and appends
`M 0 12100 1 12133`, leaving the parsed shop 12105 associated with an awake
keeper carrying nothing. Its `look` block proves room/keeper reachability;
its `list` block proves the empty carried-inventory result. The stocked
control retains the authored M/G reset and proves the live branch with all
four products plus `list pepper`. The filter vehicle uses the new disposable
`give-object` zone fixture, adding cost-zero object 1210, positive-cost
ITEM_INVISIBLE 16005 (loaded by `add-obj-index 160.obj`), and stocked 12132;
only 12132 is listed. These are disposable copies; neither C source tree nor
authoritative world files are edited.

Before the correction, the preserved reports are in
`docs/fidelity/depth/evidence/2026-09-10-shop-live-inventory/before-fix.txt`.
After correction, the target, control, and filter scenarios matched C at
seeds 1 and 2 with `--show-oracle`; the raw post-fix runs are stored beside
that file.

## Reproduction record

The target and companion runs used the isolated checkout at commit
`02d72c305` with `PATH=/usr/local/go/bin:$PATH` and
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-gate --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-gate --seed 2 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-control --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-control --seed 2 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-filters --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-live-inventory-filters --seed 2 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-stack-list-live --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario shop-stack-list-live --seed 2 --show-oracle
make fidelity-depth
make expected-divergences-check
```

Each differential report says `result: no normalized divergence`; the
`--show-oracle` blocks are preserved in the corresponding seed log. The
expected-divergence check required regenerating the existing manifest-backed
ledger's missing `character-creation-name-retry / entry.motd-color` row; no
pin or exclusion was created. The full `make oracle-regression` census is
recorded below.

## Final gates and disposition

The final code/evidence candidate was commit `e4c9df54e`. The exact full
census command was:

```text
set -o pipefail
export PATH=/usr/local/go/bin:$PATH
export DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
export ORACLE_REGRESSION_GO=/usr/local/go/bin/go
export ORACLE_REGRESSION_TIMEOUT=240s
export ORACLE_REGRESSION_SEED=1
export ORACLE_REGRESSION_JOBS=4
make oracle-regression
```

The complete output is preserved in
`docs/fidelity/depth/evidence/2026-09-10-shop-live-inventory/oracle-regression-full-final.txt`.
Its final tally is `scenarios=938 passed=928 expected=9 unpinnable=1
stale=0 failed=0 infra=0 timed_out=0`; the exit status is 2 solely for the
previously human-cleared `accuse-noarg-depth` baseline. The four shop
scenarios are PASS in that census. Four transient C listener collisions were
manually inspected and recovered on the worker's bounded retry; the inspection
is preserved in
`cutthroat-peaceful-infra-inspection.txt`, and none filed as INFRA.

`make fidelity-depth` passed with 4794 total cases, 4678 proven/delegated, 65
blocked, and 51 excluded. The final-candidate `make expected-divergences-check`
does not pass: its generator requires the unrelated
`character-creation-name-retry / entry.motd-color` row from the still-blocked
`docs/fidelity/depth/entry.tsv`, while the census proves that scenario PASS and
therefore reports the generated row stale. The exact failed output is in
`expected-divergences-final.txt`. The generated row was not committed, and the
unrelated entry debt was not repaired, pinned, or excluded.

Disposition: B (blocked). The shop debt itself is proven at D4 with C-vs-Go
fixtures and the isolated Go correction, but the requested final gate set is
not simultaneously satisfiable without an unrelated entry-manifest decision.
Smallest next action: reconcile `entry.motd-color` against its existing
scenario/evidence, then rerun `make expected-divergences-check` and the full
census. Reviewable PR: [#1437](https://github.com/zax0rz/darkpawns/pull/1437).
It is intentionally left unmerged and explicitly records the unrelated gate
blocker.

## Remaining boundary

This milestone proves `list` at the live keeper-inventory gate, stocked
format/order/price behavior, and the cost/visibility filters. The broader
shop transaction surface (all buy/sell failure and state branches, multiple
shop rooms, and all special-procedure shops) remains separately blocked in
the surface inventory and is not claimed here.
