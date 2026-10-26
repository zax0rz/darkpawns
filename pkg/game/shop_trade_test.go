package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShopTradeExpressionMatchesCStackSemantics(t *testing.T) {
	item := NewObjectInstance(&parser.Obj{Keywords: "gold ring", TypeFlag: 13, Cost: 20, ExtraFlags: [4]int{1 << 6}}, 1)
	for _, tc := range []struct {
		expression string
		want       bool
	}{
		{"", true},
		{"(absent)", true},
		{"^gold", true},
		{" gold", true},
		{"gold", true},
		{"gol", false},
		{"absent", false},
		{"MAGIC", true},
		{"glow", false},
		{"gold & MAGIC", true},
		{"gold * MAGIC", true},
		{"absent | gold", false},
		{"gold | absent", true},
		{"gold + absent", true},
		{"gold | MAGIC", false},
		{"gold & absent", false},
		{"absent & gold", false},
		{"absent ^", true},
		{"gold & (MAGIC)", true},
		{"gold & {MAGIC}", true},
		{"gold & [MAGIC]", true},
		{"gold )", false},
		{"gold ring", false},
	} {
		if got := shopEvaluateExpression(item, tc.expression); got != tc.want {
			t.Errorf("expr=%q got=%t want=%t", tc.expression, got, tc.want)
		}
	}
}

func TestShopTradeWithMatchesCRefusalOrder(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		kind, cost, charges, flags int
		types                      []int
		words                      []string
		want                       ShopTradeResult
	}{
		{"zero value", ITEM_WAND, 0, 0, 0, []int{ITEM_WAND}, nil, ShopTradeNotOK},
		{"nosell before dead", ITEM_WAND, 10, 0, 1 << 16, []int{ITEM_WAND}, nil, ShopTradeNotOK},
		{"wrong type before dead", ITEM_WAND, 10, 0, 0, []int{13}, nil, ShopTradeNotOK},
		{"dead wand before expression", ITEM_WAND, 10, 0, 0, []int{ITEM_WAND}, []string{"absent"}, ShopTradeDead},
		{"dead staff", ITEM_STAFF, 10, 0, 0, []int{ITEM_STAFF}, nil, ShopTradeDead},
		{"charged wand", ITEM_WAND, 10, 1, 0, []int{ITEM_WAND}, nil, ShopTradeOK},
		{"keyword refusal", 13, 10, 0, 0, []int{13}, []string{"silver"}, ShopTradeNotOK},
		{"later matching rule", 13, 10, 0, 0, []int{13, 13}, []string{"silver", "gold"}, ShopTradeOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shop := &Shop{BuyTypes: tc.types, BuyWords: tc.words}
			item := NewObjectInstance(&parser.Obj{TypeFlag: tc.kind, Keywords: "gold ring", Cost: tc.cost, Values: [4]int{0, 0, tc.charges, 0}, ExtraFlags: [4]int{tc.flags}}, 1)
			if got := shop.TradeWith(item); got != tc.want {
				t.Errorf("trade=%d want=%d", got, tc.want)
			}
		})
	}
}

func TestShopTransactionAmountUsesDigitOnlyLeadingCount(t *testing.T) {
	for _, tc := range []struct {
		arg   string
		count int
		name  string
	}{
		{"", 1, ""},
		{"2 gold", 2, "gold"},
		{"0 gold", 0, "gold"},
		{"gold 3", 1, "gold 3"},
		{"-1 gold", 1, "-1 gold"},
		{"+2 gold", 1, "+2 gold"},
		{"0002 gold", 2, "gold"},
	} {
		if n, name := shopNpcTransactionAmt(tc.arg); n != tc.count || name != tc.name {
			t.Errorf("NPC amount(%q)=%d,%q", tc.arg, n, name)
		}
		if n, name := ShopTransactionAmount(tc.arg); n != tc.count || name != tc.name {
			t.Errorf("amount(%q)=%d,%q", tc.arg, n, name)
		}
	}
}

func TestShopNPCGodLevelPaysAndPrefixSale(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}},
		Mobs:  []parser.Mob{{VNum: 5001, Keywords: "keeper", ShortDesc: "a keeper", Level: 10}, {VNum: 5002, Keywords: "customer", ShortDesc: "a customer", Level: LVL_GOD, Str: 10, Dex: 10, Cha: 10}},
		Objs:  []parser.Obj{{VNum: 6001, Keywords: "apple", ShortDesc: "an apple", Cost: 20, TypeFlag: 13, Weight: 1}},
		Shops: []parser.ShopProto{{VNum: 1, KeeperVNum: 5001, Rooms: []int{1001}, Products: []int{6001}, BuyTypes: []int{13}, BuyProfit: 1, SellProfit: 1, OpenHour1: 0, CloseHour1: 28, OpenHour2: 0, CloseHour2: 28}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	keeper, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	customer, err := w.SpawnMob(5002, 1001)
	if err != nil {
		t.Fatal(err)
	}
	keeper.SetPosition(8)
	customer.SetPosition(8)
	keeper.SetGold(1000)
	customer.SetGold(100)
	obj, err := w.SpawnObject(6001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToMobInventoryFront(obj, keeper); err != nil {
		t.Fatal(err)
	}
	ctx := CharacterSpecialContext{Actor: customer}
	if got := DispatchCharacterSpecial("shop_keeper", w, ctx, keeper, "buy", "2 apple"); got != SpecialHandled {
		t.Fatalf("buy dispatch=%v", got)
	}
	if customer.GetGold() != 60 || keeper.GetGold() != 1040 || len(customer.InventorySnapshot()) != 2 {
		t.Fatalf("NPC buy: coins=%d keeper=%d items=%d", customer.GetGold(), keeper.GetGold(), len(customer.InventorySnapshot()))
	}
	if got := DispatchCharacterSpecial("shop_keeper", w, ctx, keeper, "sell", "ap"); got != SpecialHandled {
		t.Fatalf("sale dispatch=%v", got)
	}
	if customer.GetGold() != 80 || keeper.GetGold() != 1020 || len(customer.InventorySnapshot()) != 1 {
		t.Fatalf("NPC prefix sale: coins=%d keeper=%d items=%d", customer.GetGold(), keeper.GetGold(), len(customer.InventorySnapshot()))
	}
}
