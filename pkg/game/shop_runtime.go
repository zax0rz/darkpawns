package game

// shopByVNumLocked returns a live shop. The caller must hold w.mu.
func (w *World) shopByVNumLocked(vnum int) (*Shop, bool) {
	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return nil, false
	}
	shop := sm.GetShopByVNum(vnum)
	return shop, shop != nil
}

// SnapshotShopAccount returns the shop and keeper balances from one world
// snapshot. The keeper may be nil when no matching mobile is loaded.
func (w *World) SnapshotShopAccount(vnum int, keeper *MobInstance) (Shop, int, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	shop, ok := w.shopByVNumLocked(vnum)
	if !ok {
		return Shop{}, 0, false
	}
	coins := 0
	if keeper != nil {
		coins = keeper.GetGold()
	}
	return CloneShop(*shop), coins, true
}

// CreditShopPurchase mirrors shopping_buy's keeper payment and optional
// deposit above MAX_OUTSIDE_BANK (src/shop.c:584-589).
func (w *World) CreditShopPurchase(vnum int, keeper *MobInstance, amount int) {
	if keeper == nil || amount <= 0 {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(vnum)
	if !ok {
		return
	}
	keeper.updateGold(func(coins int) (int, bool) {
		coins += amount
		if shop.Flags&ShopFlagUsesBank != 0 && coins > shopMaxCoinsOutsideBank {
			shop.Bank += coins - shopMaxCoinsOutsideBank
			coins = shopMaxCoinsOutsideBank
		}
		return coins, true
	})
}

// TryPayShopSale mirrors shopping_sell's combined cash/bank affordability
// check and cash subtraction (src/shop.c:763-768).
func (w *World) TryPayShopSale(vnum int, keeper *MobInstance, amount int) bool {
	if keeper == nil || amount < 0 {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(vnum)
	if !ok {
		return false
	}
	return keeper.updateGold(func(coins int) (int, bool) {
		if coins+shop.Bank < amount {
			return coins, false
		}
		return coins - amount, true
	})
}

// RefundShopSale restores keeper coins if the item could not be removed after
// the C-ordered account check.
func (w *World) RefundShopSale(vnum int, keeper *MobInstance, amount int) {
	if keeper == nil || amount <= 0 {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.shopByVNumLocked(vnum); ok {
		keeper.updateGold(func(coins int) (int, bool) { return coins + amount, true })
	}
}

// RebalanceShopBank mirrors shopping_sell's refill after a successful sale
// (src/shop.c:808-812).
func (w *World) RebalanceShopBank(vnum int, keeper *MobInstance) {
	if keeper == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(vnum)
	if !ok {
		return
	}
	keeper.updateGold(func(coins int) (int, bool) {
		if coins >= shopMinCoinsOutsideBank || shop.Bank <= 0 {
			return coins, true
		}
		transfer := shopMaxCoinsOutsideBank - coins
		if transfer > shop.Bank {
			transfer = shop.Bank
		}
		shop.Bank -= transfer
		return coins + transfer, true
	})
}
