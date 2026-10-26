package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShopListKeywordMatchesCIsNameBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		keywords string
		want     bool
	}{
		{name: "complete token", keyword: "pepper", keywords: "black pepper spice", want: true},
		{name: "middle of token is not abbreviation", keyword: "pep", keywords: "black pepper spice", want: false},
		{name: "final token remains exact", keyword: "cinnamon", keywords: "cinnamon spice", want: true},
		{name: "case insensitive", keyword: "PEPPER", keywords: "black pepper spice", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shopListKeywordMatches(tt.keyword, tt.keywords); got != tt.want {
				t.Fatalf("shopListKeywordMatches(%q, %q) = %t, want %t", tt.keyword, tt.keywords, got, tt.want)
			}
		})
	}
}

func TestShopHashPurchaseSelectorIgnoresTrailingWord(t *testing.T) {
	w, _, _, _ := shopOutputFixture(t, 1)
	appleProto, ok := w.GetObjPrototype(6001)
	if !ok {
		t.Fatal("apple prototype missing")
	}
	manualProto, ok := w.GetObjPrototype(6003)
	if !ok {
		t.Fatal("manual prototype missing")
	}
	apple := game.NewObjectInstance(appleProto, -1)
	manual := game.NewObjectInstance(manualProto, -1)
	viewer := game.NewPlayer(1, "Buyer", 1001)

	if got := findShopPurchaseObject("#2 manual", []*game.ObjectInstance{apple, manual}, viewer); got != manual {
		t.Fatalf("hash selector with trailing word = %v, want second stock group %d", got, manual.ID)
	}
}

func TestCmdBuyExtractsSelectedZeroCostKeeperStock(t *testing.T) {
	w, manager, keeper, _ := shopOutputFixture(t, 1)
	stock := addShopKeeperStock(t, w, keeper, 6002)
	buyer := addShopOutputSession(t, manager, "ZeroCostBuyer")
	buyer.player.SetGold(100)

	oldRoller := combat.GetRoller()
	combat.SetRoller(shopStockRoller{})
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	if err := cmdBuy(buyer, []string{"pebble"}); err != nil {
		t.Fatalf("cmdBuy zero-cost stock: %v", err)
	}
	for _, carried := range keeper.InventorySnapshot() {
		if carried == stock {
			t.Fatal("selected zero-cost stock remained in keeper inventory")
		}
	}
	for _, registered := range w.GetAllObjects() {
		if registered == stock {
			t.Fatal("selected zero-cost stock remained registered")
		}
	}
	if got := renderedOutput(buyer); got != "A merchant tells you, 'No such item.'\r\n" {
		t.Fatalf("zero-cost buy response = %q", got)
	}
}

type shopCountingRoller struct{ numberCalls int }

func (r *shopCountingRoller) Number(_, _ int) int { r.numberCalls++; return 1 }
func (*shopCountingRoller) Dice(num, _ int) int   { return num }
func (*shopCountingRoller) IntN(_ int) int        { return 1 }

func TestCmdBuyWithoutArgumentsSortsKeeperStockBeforePrompt(t *testing.T) {
	_, manager, keeper, _ := shopOutputFixture(t, 1)
	addShopKeeperStock(t, manager.world, keeper, 6003)
	buyer := addShopOutputSession(t, manager, "PromptBuyer")
	roller := &shopCountingRoller{}
	oldRoller := combat.GetRoller()
	combat.SetRoller(roller)
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	if err := cmdBuy(buyer, nil); err != nil {
		t.Fatalf("cmdBuy without arguments: %v", err)
	}
	if roller.numberCalls != 1 {
		t.Fatalf("sort draws before empty-argument response = %d, want 1", roller.numberCalls)
	}
	if got := renderedOutput(buyer); got != "A merchant tells you, 'What do you want to buy??'\r\n" {
		t.Fatalf("empty-argument response = %q, want C keeper prompt", got)
	}
}

func TestShopBuyAndSellCheckOpenHours(t *testing.T) {
	_, manager, _, shop := shopOutputFixture(t, 1)
	shop.OpenHour1 = 29
	shop.OpenHour2 = 29

	buyer := addShopOutputSession(t, manager, "ClosedBuyer")
	if err := cmdBuy(buyer, []string{"apple"}); err != nil {
		t.Fatalf("cmdBuy at closed shop: %v", err)
	}
	if got := renderedOutput(buyer); got != "A merchant exclaims, 'Come back later!'\r\n" {
		t.Fatalf("closed-shop buy response = %q", got)
	}

	seller := addShopOutputSession(t, manager, "ClosedSeller")
	if err := cmdSell(seller, []string{"apple"}); err != nil {
		t.Fatalf("cmdSell at closed shop: %v", err)
	}
	if got := renderedOutput(seller); got != "A merchant exclaims, 'Come back later!'\r\n" {
		t.Fatalf("closed-shop sell response = %q", got)
	}
}

func TestSwitchedShopKeeperResetsSortOnCommand(t *testing.T) {
	_, manager, keeper, shop := shopOutputFixture(t, 1)
	actor := addShopOutputSession(t, manager, "Switcher")
	actor.isSwitched = true
	actor.switchedMob = keeper
	shop.LastSort = 9

	if handled := runSpecials(actor, "look", nil, ""); handled {
		t.Fatal("shopkeeper special unexpectedly handled look")
	}
	if shop.LastSort != 0 {
		t.Fatalf("SHOP_SORT after switched keeper command = %d, want 0", shop.LastSort)
	}
}
