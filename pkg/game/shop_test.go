package game

import "testing"

func TestShopPriceUsesCIntegerOrder(t *testing.T) {
	shop := &Shop{ProfitBuy: 1.5, ProfitSell: 0.5}

	// C truncates the markup before applying charisma, then truncates the
	// compound assignment. Applying charisma to the float markup would return
	// 4 for this case instead of C's 3.
	if got := shop.BuyPrice(3, 10); got != 3 {
		t.Fatalf("BuyPrice(3, 10) = %d, want 3", got)
	}
	if got := shop.SellPrice(3, 10); got != 1 {
		t.Fatalf("SellPrice(3, 10) = %d, want 1", got)
	}
}
