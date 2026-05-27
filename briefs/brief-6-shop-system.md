# Brief 6: Shop System Overhaul

**Issues:** DP-503, DP-504, DP-505, DP-506, DP-507, DP-508, DP-509
**Priority:** HIGH (3 High, 2 Medium, 2 Low)
**Files:** `pkg/game/systems/shop.go`, `pkg/game/shop.go`, `pkg/game/scripts.go`, `lib/world/scripts/globals.lua`, `lib/world/scripts/mob/blacksmith.lua`
**C Sources:** `src/shop.c`, `src/shop.h`

---

## Problem

The shop system has two separate implementations, missing features, and broken Lua integrations. The active shop (`pkg/game/systems/shop.go`) is a simplified integer-based port that dropped several critical CircleMUD mechanics. The faithful float-based implementation (`pkg/game/shop.go`) exists but is unused. Meanwhile, Lua scripts that interact with shops reference undefined functions.

---

## Issues in This Brief

### DP-509 — Charisma ignored in shop pricing (HIGH)

**Go:** `pkg/game/systems/shop.go:109-140` — `CalculateBuyPrice`/`CalculateSellPrice` use raw integer multipliers with no Charisma input.
**C:** `src/shop.c:462-467` — `buy_price()` applies `price -= price * (GET_CHA(ch) * 0.005)`.
**Unused Go:** `pkg/game/shop.go:130-145` — `BuyPrice`/`SellPrice` have the correct Charisma logic already written.

**Fix:** Port the Charisma scaling from `pkg/game/shop.go` into `pkg/game/systems/shop.go`. The unused file already has the correct implementation — copy the `cha` parameter and the `0.005` multiplier math into the active shop's price functions.

### DP-504 — No shopkeeper gold limits (HIGH)

**Go:** `pkg/game/systems/shop.go` — no `Gold` or `bankAccount` field on the Shop struct.
**C:** `src/shop.h:63` — `int bankAccount;` field on `shop_index`. `src/shop.c:808` — keeper gold replenished from bank on restock.

**Fix:** Add `Gold` and `BankAccount` fields to `systems.Shop`. Decrement keeper gold when players sell items. When gold depleted, keeper refuses to buy (use `missing_cash1` message). Replenish from bank on restock ticks. See `src/shop.c:800-815` for the restock logic.

### DP-507 — No with_who trade constraints (MEDIUM)

**Go:** `pkg/game/systems/shop.go` — no trade restriction checks.
**C:** `src/shop.c:74-101` — `is_ok_char()` checks `NOTRADE_GOOD`, `NOTRADE_EVIL`, `NOTRADE_NEUTRAL`, `NOTRADE_MAGIC_USER`, `NOTRADE_CLERIC`, `NOTRADE_THIEF`, `NOTRADE_WARRIOR` bitflags. `src/shop.h:59` — `int with_who;` field. `src/shop.h:149-155` — NOTRADE macros.

**Fix:** Add `WithWho` int field to `systems.Shop`. Read it from `.shp` files during shop parsing. Before each transaction (buy/sell/steal/list), validate the player against the with_who bitvector. Return appropriate rejection messages (`MSG_NO_SELL_ALIGN`, `MSG_NO_SELL_CLASS`) from `src/shop.c`.

### DP-503 — Delete unused pkg/game/shop.go (LOW)

**File:** `pkg/game/shop.go` — 200+ line faithful implementation, never imported.

**Fix:** After porting Charisma pricing (DP-509) from this file into `systems/shop.go`, delete `pkg/game/shop.go` entirely. Run `go build ./...` to confirm nothing references it.

### DP-506 — dofile path double-prefixes scripts/ (MEDIUM)

**Go:** `pkg/scripting/engine.go:1665` — `luaDofile` joins `scriptsDir` (`lib/world/scripts/`) with the path argument.
**Lua:** `lib/world/scripts/globals.lua:154` — `dofile("scripts/mob/no_move.lua")` produces `lib/world/scripts/scripts/mob/no_move.lua` (double prefix, doesn't exist).

**Fix:** Change `globals.lua:154` from `dofile("scripts/mob/no_move.lua")` to `dofile("mob/no_move.lua")`. The engine already prepends `scriptsDir`.

### DP-505 — foreachi undefined — black armor quest broken (HIGH)

**Lua:** `lib/world/scripts/mob/blacksmith.lua:39` — calls `foreachi(pieces, one_piece)`. This is a Lua 4.x function not available in gopher-lua (Lua 5.1). Not defined in `globals.lua` or `engine.go`.

**Impact:** Black armor quest panics and rolls back when player gives humming black armor to the blacksmith.

**Fix:** Add a `foreachi` compatibility shim to `globals.lua`:
```lua
function foreachi(tbl, fn)
  for i, v in ipairs(tbl) do
    local res = fn(i, v)
    if res then return res end
  end
end
```

### DP-508 — ongive rejection message commented out (LOW)

**Go:** `pkg/game/scripts.go:76-77` — the SendMessage call is commented out:
```go
// ctx.Ch.SendMessage("You can't give that here.\r\n")
```

**Fix:** Uncomment the SendMessage call. When ongive returns false/nil, the player should see "You can't give that here."

---

## Execution Order

1. **DP-506** first — one-line Lua fix, unblocks `no_move` for other scripts
2. **DP-505** second — add `foreachi` shim to `globals.lua`, unblocks black armor quest
3. **DP-509** third — port Charisma pricing from `shop.go` → `systems/shop.go`
4. **DP-504** fourth — add gold limits to Shop struct
5. **DP-507** fifth — add with_who trade constraints
6. **DP-508** sixth — uncomment ongive message
7. **DP-503** last — delete `pkg/game/shop.go` after all ports are done

## Verification

After all fixes:
```bash
cd darkpawns_repo
go build ./...
go vet ./...
go test ./...
```

Manually verify:
- `grep -rn "pkg/game/shop.go"` should return no imports
- `grep -rn "foreachi"` in `globals.lua` should show the shim
- `grep -rn "dofile"` in `globals.lua` should show `mob/no_move.lua` (no `scripts/` prefix)
