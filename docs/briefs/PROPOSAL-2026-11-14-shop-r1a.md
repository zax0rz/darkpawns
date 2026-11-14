# Proposal: one standing R1a ruling for the shop integer/identity branches

2026-11-14, DeepSeek, for Zach. This is a **proposal**; the rulebook and this
text are yours to change. Five blocked branches in
`docs/fidelity/depth/surface/shops-behavior.tsv` depend on C undefined
behaviour. No oracle scenario can settle them, and a unit test would have to
invent a semantic the C server never defined. They are the only reason the
shops families cannot reach zero.

## The five branches

| Row | C site | Why it is undefined |
|---|---|---|
| `buy_price` out-of-range floating conversion | `src/shop.c:460-468` | `(int)(GET_OBJ_COST(obj) * SHOP_BUYPROFIT(shop_nr))`: an out-of-range float-to-int conversion is UB; the result is compiler- and platform-dependent. |
| `shopping_buy` signed transaction sum or keeper gold overflow | `src/shop.c:548-550` | Signed `int` additions (`goldamt`, keeper gold/bank) with no range check; overflow is UB. |
| `shopping_sell` signed sale sum or player gold overflow | `src/shop.c:773-774` | Same class on the sell side. |
| `pet_shops` buy player from adjacent roster | `src/spec_procs.c:1878-1880` | `read_mobile(GET_MOB_RNUM(pet), REAL)` is called without an `IS_NPC` guard; a player in the roster supplies an invalid mob index, so C reads out of bounds. |
| `bank` signed carried gold or bank account overflow | `src/spec_procs.c:2369-2392` | Signed deposit/withdraw sums without range checks; overflow is UB. |

Bounded versions of the same paths are already proven: the shop transaction,
bank, and pet-shop scenarios cover ordinary prices, transfers, and the
`shops-pets-roster-player` listing. Only the overflow/out-of-bounds moment is
open.

## Proposed standing text for Zach (not yet in the rulebook)

> **Shop and special-procedure arithmetic/identity branches that depend on
> undefined behaviour are settled by exclusion, not by a test.** The branches
> are `buy_price`'s float-to-int conversion when the product leaves `int` range
> (`src/shop.c:460-468`), the signed transaction/gold sums in `shopping_buy`
> and `shopping_sell` (`src/shop.c:548-550`, `:773-774`), `pet_shops` buying a
> player from the roster through `read_mobile(GET_MOB_RNUM(pet), REAL)`
> (`src/spec_procs.c:1878-1880`), and the signed bank sums in the bank special
> (`src/spec_procs.c:2369-2392`). The port keeps bounded, defined behaviour,
> invents no player-facing bytes, and never reproduces an out-of-bounds read or
> a signed overflow. Each row names its C line.

This is the same shape as the existing long-shot and undefined-behaviour
exceptions: one class, one ruling, each row naming its C line and its odds or
its UB.

## What stays blocked, and why (not R1a)

These rows are fixture gaps, not undefined behaviour:

- **Broke-temper smoke / other-temper silent** (`src/shop.c:505-518`). The code
  path exists in both player and NPC buy paths (`pkg/session/shop_cmds.go`,
  `pkg/game/shop_npc.go`: temper 0 pukes, 1 emotes "smokes on his joint.",
  default silent). The missing piece is an isolating vehicle: a `sedit` temper
  change plus a poor mortal peer in one disposable run.
- **Stop on a different stock group** (`src/shop.c:534-557`). Needs a
  heterogenous stock fixture: two groups with the same keyword but nonidentical
  identity fields.
- **Secondary special first / assign secondary** (`src/shop.c:942-1002`,
  `:1224-1240`). Needs a keeper with a pre-existing special function and a shop
  assignment that must preserve it.
- **`ok_damage_shopkeeper` kill-enabled keeper proceeds / ordinary mob
  proceeds** (`src/shop.c:1006-1025`). The protected-keeper arm is proven by
  `hit.shopkeeper-gate`; the two proceed arms are combat-owned and need routing
  to a combat manifest, not a new shop proof.

## Asks

1. Approve (or rewrite) the standing text above; then the five rows become
   `excluded` with the ruling cite and the two shops families drop to the six
   fixture rows listed here.
2. Confirm the routing of the two `ok_damage_shopkeeper` rows to combat.
