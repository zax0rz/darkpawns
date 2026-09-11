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

## Remaining boundary

This milestone proves `list` at the live keeper-inventory gate, stocked
format/order/price behavior, and the cost/visibility filters. The broader
shop transaction surface (all buy/sell failure and state branches, multiple
shop rooms, and all special-procedure shops) remains separately blocked in
the surface inventory and is not claimed here.
