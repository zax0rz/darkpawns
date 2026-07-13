# Fidelity Brief 06: Shop System

**Date:** 2026-05-27
**Priority:** MEDIUM — 65 shops, economy backbone
**C source:** `src/shop.c` (1445 lines)
**Go source:** scattered across game/session files

---

## Scope

Shops are the economy. 65 shopkeepers buy, sell, and haggle. The C source has specific logic for:
1. Shop data structure (keeper, items, buy/sell types, profit margins)
2. Buy/sell commands
3. Haggling system
4. Shopkeeper responses (greeting, denial, stolen item detection)
5. Shop item lists (what each shopkeeper sells/buys)

This brief covers:

1. **Shop data structure** — `struct shop_data`, loaded from `lib/etc/shops`
2. **`do_buy()`** — purchasing from shopkeeper
3. **`do_sell()`** — selling to shopkeeper
4. **`do_value()`** — asking price
5. **`do_list()`** — listing items for sale
6. **Haggling** — price negotiation
7. **Shopkeeper behavior** — greeting, denial, theft detection

---

## What to Verify

### 1. Shop Data Structure

**C source** (shop.c:68):
```c
struct shop_data *shop_index;
```

Loaded from `lib/etc/shops` file. Each shop has:
- Shopkeeper mob vnum
- Buy types (what the shop buys)
- Sell types (what the shop sells)
- Profit buy percentage (>100% = markup)
- Profit sell percentage (<100% = discount)
- Room vnum
- Opening/closing hours

**Check:** Does the Go code load this file correctly? Are all 65 shops parsed?

### 2. `do_buy()` — Purchasing

**C source** (shop.c):
```c
// Player says "buy <item>"
// Shopkeeper checks if item is in stock
// Price = item cost * profit_buy_percentage
// Player must have enough gold
// Gold deducted, item given to player
// Messages: "You buy $p." / "$n buys $p."
```

**Check:**
- Price calculation formula
- Stock check
- Gold deduction
- Item transfer
- Messages

### 3. `do_sell()` — Selling

**C source** (shop.c):
```c
// Player says "sell <item>"
// Shopkeeper checks if item is in buy types
// Price = item cost * profit_sell_percentage
// Shopkeeper may refuse (stolen item, wrong type)
// Gold given, item taken
// Messages: "You sell $p for %d gold." / "$n sells $p."
```

**Check:**
- Price calculation formula
- Type check (buy types)
- Stolen item detection
- Messages

### 4. Haggling System

**C source** (shop.c):
The haggling system has:
- Player offers a price
- Shopkeeper counter-offers
- Multiple rounds of negotiation
- Each round has a chance to break off
- Shopkeeper's patience decreases

**Check:**
- Is the haggling system implemented in Go?
- What's the maximum haggle rounds?
- What's the shopkeeper's counter-offer formula?

### 5. Shopkeeper Responses

**C source** (shop.c):
Shopkeepers have specific responses:
- Greeting: "Welcome to my shop!" (on enter)
- Denial: "I don't sell that." (wrong item type)
- No gold: "You don't have enough gold."
- Stolen: "I don't buy stolen goods!" (if item has TIMER set or owner check)
- Farewell: "Come back soon!" (on leave)

**Check:** Are these messages present? Do they match the C source?

### 6. Shop Types

**C source** (constants.c):
Shop types define what each shop buys/sells:
```
TYPE_MAGIC    — scrolls, potions, wands, staffs
TYPE_WEAPON   — weapons
TYPE_ARMOR    — armor
TYPE_FOOD     — food and drink
TYPE_CONTAINER — containers
TYPE_POTION   — potions
TYPE_WAND     — wands
TYPE_STAFF    — staffs
TYPE_SCROLL   — scrolls
```

**Check:** Does the Go code have the same type definitions? Are they used correctly in shop data?

---

## Implementation Notes

- Shop data is loaded from `lib/etc/shops` — CircleMUD format
- Shopkeeper mobile scripts handle the buy/sell interaction
- Some shops have custom behavior via spec_procs (e.g., "The Keeper" in zone 8)
- The `ok_shopkeeper()` function checks if the player can buy from/sell to a specific shopkeeper

---

## Verification

1. Visit each of the 65 shops — verify they open and sell items
2. Buy an item — verify price and gold deduction
3. Sell an item — verify price and item removal
4. Try to sell wrong type — verify denial
5. Try to buy with no gold — verify denial
6. Haggle with shopkeeper — verify price negotiation
7. Visit shop outside hours — verify denial
8. Run `go test ./pkg/game/...`
