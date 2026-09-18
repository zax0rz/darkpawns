package admin

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newTestWorldForWrite creates a minimal World with one of each entity type
// for testing world write methods.
func newTestWorldForWrite(t *testing.T) *game.World {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Test Room", Zone: 1,
				Flags:  []string{"0", "0", "0", "0"},
				Sector: 0,
				Exits:  map[string]parser.Exit{},
			},
			{
				VNum: 1002, Name: "Second Room", Zone: 1,
				Flags:  []string{"0", "0", "0", "0"},
				Sector: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1001},
				},
			},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "a guard", LongDesc: "A guard stands here.", Level: 5, AC: 50, Gold: 10, Exp: 100, Alignment: 0, Position: 0, DefaultPos: 0, Sex: 0},
			{VNum: 2002, ShortDesc: "a merchant", LongDesc: "A merchant eyes you.", THAC0: 10, Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10},
		},
		Objs: []parser.Obj{
			{
				VNum: 3001, Keywords: "sword", ShortDesc: "a steel sword",
				LongDesc: "A steel sword lies here.", TypeFlag: 5,
				Weight: 5, Cost: 100,
				WearFlags:  [4]int{1 << 13, 0, 0, 0},
				Values:     [4]int{0, 3, 5, 0},
				ExtraFlags: [4]int{0, 0, 0, 0},
			},
			{
				VNum: 3002, Keywords: "shield", ShortDesc: "a wooden shield",
				LongDesc: "A wooden shield is here.", TypeFlag: 11,
				Weight: 8, Cost: 50,
			},
		},
		Zones: []parser.Zone{
			{Number: 1, Name: "Test Zone", TopRoom: 2000, Lifespan: 15, ResetMode: 1},
		},
	}

	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// ---------------------------------------------------------------------------
// Room write methods
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mob write methods
// ---------------------------------------------------------------------------

func TestSetMobTHAC0(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobTHAC0(2002, 5) {
		t.Fatal("SetMobTHAC0 returned false")
	}
	mob, _ := w.GetMobPrototype(2002)
	if mob.THAC0 != 5 {
		t.Errorf("THAC0 = %d, want 5", mob.THAC0)
	}
}

// ---------------------------------------------------------------------------
// Object write methods
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Zone write methods
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Shop write methods — requires a shop manager with a real ShopManager
// ---------------------------------------------------------------------------

// newWorldWithShops creates a world that also has shops configured.
func newWorldWithShops(t *testing.T) *game.World {
	t.Helper()
	w := newTestWorldForWrite(t)

	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{
		KeeperVNum: 2002,
		BuyTypes:   []int{1, 5},
		SellTypes:  []int{3001},
		ProfitBuy:  1.2,
		ProfitSell: 0.8,
		KeeperName: "Merchant",
		RoomVNum:   1001,
	})
	w.SetShopManager(sm)
	return w
}

func TestSetShopBuyTypes_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopBuyTypes(2002, []int{2, 3, 4}) {
		t.Fatal("SetShopBuyTypes returned false")
	}
	shop, ok := w.GetShopByKeeper(2002)
	if !ok {
		t.Fatal("shop not found")
	}
	if len(shop.BuyTypes) != 3 || shop.BuyTypes[0] != 2 {
		t.Errorf("buy types = %v, want [2 3 4]", shop.BuyTypes)
	}
}

func TestSetShopBuyTypes_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopBuyTypes(9999, []int{1}) {
		t.Error("expected false for missing shop keeper")
	}
}

func TestSetShopSellTypes_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopSellTypes(2002, []int{3002}) {
		t.Fatal("SetShopSellTypes returned false")
	}
}

func TestSetShopSellTypes_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopSellTypes(9999, []int{1}) {
		t.Error("expected false")
	}
}

func TestSetShopProfit_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopProfit(2002, 1.5, 0.6) {
		t.Fatal("SetShopProfit returned false")
	}
	shop, ok := w.GetShopByKeeper(2002)
	if !ok {
		t.Fatal("shop not found")
	}
	if shop.ProfitBuy != 1.5 || shop.ProfitSell != 0.6 {
		t.Errorf("profit = %.1f/%.1f, want 1.5/0.6", shop.ProfitBuy, shop.ProfitSell)
	}
}

func TestSetShopProfit_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopProfit(9999, 1.0, 1.0) {
		t.Error("expected false")
	}
}
