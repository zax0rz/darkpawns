# Brief: processSell check-first ordering — DP-1224 — 2026-08-11

**Executor:** DeepSeek-v4-flash (0731) via opencode. **Delivered by Zach.**
**Repo/base:** `git@github-darkpawns:zax0rz/darkpawns.git`, branch **from `main`**.
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS. Also `gofumpt -w` any file you touch.
**Not oracle-gated:** shop sell has no `cmd/dp-oracle-diff` scenario; prove this fix with the unit test below, not the oracle.

---

## Fix: DP-1224 — processSell removes the item before verifying the shop can buy it (MED)

**File:** `pkg/game/systems/shop_manager.go` — `(*ShopManager).processSell()` (starts line ~215)

**Problem:**
`processSell` calls `player.Inventory.RemoveItem(gameItem)` **first** (line ~235), *then* checks whether the shop can pay (`shop.TryDeductGold`, ~245) and whether the shop has space (`len(shop.GetInventory()) >= shop.MaxItems`, ~253). Every rejection path therefore has to put the item back with `player.Inventory.AddItem(...)`, and **if that restore fails** (inventory capacity/weight edge) the item is **orphaned** — gone from the player, never added to the shop, only a `slog.Error`. That is permanent item loss on a path the player didn't cause.

**C authority — check-first.** `src/shop.c` `shopping_sell()` (line 763) verifies affordability **before any transfer**:
```c
if (GET_GOLD(keeper) + SHOP_BANK(shop_nr) < sell_price(ch, obj, shop_nr)) {
  /* ... keeper-can't-afford message ... */
  return;
}
```
The item leaves the player only after the shop is confirmed able to buy it.

**Cite:** `src/shop.c:763` (`shopping_sell`, affordability gate before transfer). No behavior change to the *success* path or to the existing concurrency design (keep `player.Lock()` held throughout, and keep `shop.TryDeductGold` as the atomic check-and-deduct — DP-660/DP-757).

**Fix — reorder so nothing mutates the player until gold + space are confirmed.** Move the space check and the gold deduction **ahead of** `RemoveItem`; refund the gold if `RemoveItem` then reports the player doesn't have the item. Concretely, the body becomes (after the existing trade-restriction and `CanBuyType` checks and the `gameItem, ok := item.(*game.ObjectInstance)` assertion):

```go
	price := shop.CalculateBuyPrice(item, player.Stats.Cha)

	// C shop.c:763 shopping_sell — verify the shop can buy BEFORE the item
	// leaves the player, so no rejection path can orphan it.

	// 1. Space check (no mutation).
	if len(shop.GetInventory()) >= shop.MaxItems {
		return false, "The shop's inventory is full."
	}

	// 2. Gold: atomic check-and-deduct (no item mutation yet).
	if !shop.TryDeductGold(price) {
		return false, "Sorry, I don't have enough cash to buy that."
	}

	// 3. Only now remove from the player. If the item isn't actually there,
	//    refund the gold we just deducted.
	if !player.Inventory.RemoveItem(gameItem) {
		shop.AddGold(price)
		return false, "You don't have that item."
	}

	// 4. Pay the player and hand the item to the shop.
	player.Gold += price
	gameItem.Location = game.LocNowhere()
	if !shop.AddItem(item) {
		// Rollback: un-pay, refund shop, restore item (just removed, so this
		// restore is safe).
		player.Gold -= price
		shop.AddGold(price)
		if err := player.Inventory.AddItem(gameItem); err != nil {
			slog.Error("shop sell rollback: failed shop add, failed to restore item",
				"player", player.Name, "obj_vnum", gameItem.VNum, "error", err)
		}
		return false, "The shopkeeper can't take that right now."
	}
	// ... any existing post-transfer bookkeeping (logging, restock timers) stays ...
	return true, "..." // keep the existing success message/return verbatim
```

Notes for the executor:
- Delete the two old rollback `AddItem` blocks that belonged to the *pre-remove* gold/space checks — those paths now `return` **before** `RemoveItem`, so there is nothing to roll back.
- Do **not** change `TryDeductGold` / `AddGold` / `AddItem` signatures or the `player.Lock()`/`defer player.Unlock()` bracket.
- Keep the existing success-path return value/message exactly as it is today (read the current tail of the function and preserve it).
- The only remaining inventory-restore rollback is the `shop.AddItem` failure in step 4, which restores an item removed microseconds earlier under the same held lock — the orphan window is closed on every player-visible rejection path.

**Regression test:** `pkg/game/systems/shop_manager_test.go`
- `TestProcessSell_ShopCannotPay_ItemStaysWithPlayer`: shop with gold < price; call `processSell`; assert it returns `false`, the item is **still in `player.Inventory`**, `player.Gold` unchanged, shop gold unchanged.
- `TestProcessSell_ShopFull_ItemStaysWithPlayer`: shop at `MaxItems`; assert `false`, item still with player, no gold moved on either side.
- `TestProcessSell_Success`: solvent shop with space; assert `true`, item moved to shop, `player.Gold += price`, shop gold `-= price` (guards the reorder didn't break the happy path).
- Reuse whatever shop/player construction the existing `shop_manager_test.go` uses.

**Verification:** `go build ./... && go vet ./... && go test ./... && gofumpt -l pkg/game/systems`

---

## Git / delivery
- Branch from `main` (e.g. `ds/dp-1224-shop-sell-order`). Edit + tests only, **no merge**.
- Commit: `fix: processSell verifies shop gold/space before removing item (DP-1224)`.
- Open a PR (or hand the branch back). Claude reviews the C-fidelity + runs the build gate before merge.
- After merge: DP-1224 → Done with the commit hash.
