package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShowShopsBootOrderAndColumns(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 5, ShortDesc: "keeper"}}}
	for i := range 20 {
		parsed.Shops = append(parsed.Shops, parser.ShopProto{VNum: 900 - i, Rooms: []int{1001}, KeeperVNum: 5, BuyProfit: 1.234, SellProfit: 0.567, WithWho: 3})
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
	s.wantsStructuredData = true
	if err := cmdShow(s, []string{"shops"}); err != nil {
		t.Fatal(err)
	}
	got := renderedOutput(s)
	const header = " ##   Virtual   Where    Keeper    Buy   Sell   Customers\n\r---------------------------------------------------------\n\r"
	if !strings.HasPrefix(got, "\n\r"+header+"  1      900     1001         5   0.57   1.23    __NMCTW\n\r") {
		t.Fatalf("first row/columns/order: %q", got)
	}
	if strings.Count(got, header) != 2 {
		t.Fatalf("19-row header cadence: %q", got)
	}
	if !strings.Contains(got, " 19      882     1001         5   0.57   1.23    __NMCTW\n\r"+header+" 20      881") {
		t.Fatalf("header boundary/order: %q", got)
	}
	if !strings.HasSuffix(got, " 20      881     1001         5   0.57   1.23    __NMCTW\n\r") {
		t.Fatalf("last row: %q", got)
	}
}

func TestShowShopDetailedReportAndLiveBankThroughDispatch(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Market"}, {VNum: 1002, Name: "Back room"}},
		Mobs: []parser.Mob{{
			VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant",
			Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{{
			VNum: 6001, Keywords: "apple", ShortDesc: "a red apple",
			LongDesc: "A red apple is here.", TypeFlag: int(game.ItemOther), Cost: 100, Weight: 1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	keeper, err := world.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	keeper.SetGold(5000)
	newestInRoom, err := world.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	newestInRoom.SetGold(8888)
	newestKeeper, err := world.SpawnMob(5001, 1002)
	if err != nil {
		t.Fatal(err)
	}
	newestKeeper.SetGold(7777)
	shops := game.NewShopManager()
	shops.AddShop(&game.Shop{
		VNum: 77, KeeperVNum: 5001, Flags: game.ShopFlagUsesBank, Bank: 1000,
		Rooms: []int{1001}, SellTypes: []int{6001}, BuyTypes: []int{int(game.ItemOther)},
		ProfitBuy: 1, ProfitSell: 1,
	})
	world.SetShopManager(shops)

	manager := newTestManager(t, world, nil)
	s := makeCommandTestSession(t, manager, "Showgod", game.LVL_IMPL, 1001)
	proto, ok := world.GetObjPrototype(6001)
	if !ok {
		t.Fatal("apple prototype missing")
	}
	item := world.NewObjectFromProto(proto, -1)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatal(err)
	}
	price := shops.GetShopByVNum(77).SellPrice(item.GetCost(), s.player.GetCha())
	if err := ExecuteCommand(s, "sell", []string{"apple"}); err != nil {
		t.Fatalf("real sell dispatch: %v", err)
	}
	wantCoins := 8888 - price
	if got := newestInRoom.GetGold(); got != wantCoins {
		t.Fatalf("room-list keeper coins after sale = %d, want %d", got, wantCoins)
	}
	if got := keeper.GetGold(); got != 5000 {
		t.Fatalf("older room keeper coins after sale = %d, want 5000", got)
	}
	shop, coins, ok := world.SnapshotShopAccount(77, newestInRoom)
	if !ok || coins != wantCoins || shop.Bank != 1000 {
		t.Fatalf("shop account after sale: found=%v coins=%d bank=%d, want true/%d/1000", ok, coins, shop.Bank, wantCoins)
	}
	_ = renderedOutput(s)

	if err := ExecuteCommand(s, "show", []string{"shops", "1"}); err != nil {
		t.Fatalf("real show dispatch: %v", err)
	}
	want := fmt.Sprintf(
		"Vnum:       [   77], Rnum: [    1]\n\r"+
			"Rooms:      Market (#1001)\n\r"+
			"Shopkeeper: a merchant (#5001), Special Function: NO\n\r"+
			"Coins:      [%9d], Bank: [%9d] (Total: %d)\n\r"+
			"Customers:  Good, Evil, Neutral, Magic User, Cleric, Thief, Warrior\n\r"+
			"Produces:   a red apple (#6001)\n\r"+
			"Buys:       OTHER (#12) [all]\n\r"+
			"Buy at:     [1.00], Sell at: [1.00], Open: [0-0, 0-0]\n\r"+
			"Bits:       USES_BANK \n\r", newestKeeper.GetGold(), shop.Bank, newestKeeper.GetGold()+shop.Bank)
	if got := renderedOutput(s); got != want {
		t.Fatalf("detailed report bytes\ngot:  %q\nwant: %q", got, want)
	}

	if got := showDetailedShop(world, "bad", 1001); got != "Illegal shop number.\n\r" {
		t.Fatalf("invalid selector = %q", got)
	}
	if got := showDetailedShop(world, ".", 1001); got != want {
		t.Fatalf("room selector bytes\ngot:  %q\nwant: %q", got, want)
	}
}

func TestDetailedShopListWrapsAtCByteBoundary(t *testing.T) {
	entries := []string{strings.Repeat("a", 60), strings.Repeat("b", 20)}
	got := detailedShopList("Produces:   ", entries, "Nothing!")
	want := "Produces:   " + strings.Repeat("a", 60) + ", \n\r" +
		"            " + strings.Repeat("b", 20) + "\n\r"
	if got != want {
		t.Fatalf("wrapped bytes = %q, want %q", got, want)
	}
}

func TestShowShopDetailedEmptyAndInvalidSelectors(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}},
		Shops: []parser.ShopProto{{VNum: 1, KeeperVNum: -1, WithWho: 127}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	const want = "Vnum:       [    1], Rnum: [    1]\n\r" +
		"Rooms:      None!\n\r" +
		"Shopkeeper: <NONE>\n\r" +
		"Customers:  None\n\r" +
		"Produces:   Nothing!\n\r" +
		"Buys:       Nothing!\n\r" +
		"Buy at:     [0.00], Sell at: [0.00], Open: [0-0, 0-0]\n\r" +
		"Bits:       NOBITS \n\r"
	if got := showDetailedShop(w, "1", 1001); got != want {
		t.Fatalf("empty report bytes\ngot:  %q\nwant: %q", got, want)
	}
	for _, selector := range []string{"bad", "0", "2"} {
		if got := showDetailedShop(w, selector, 1001); got != "Illegal shop number.\n\r" {
			t.Errorf("selector %q = %q", selector, got)
		}
	}
	if got := showDetailedShop(w, ".", 1001); got != "This isn't a shop!\n\r" {
		t.Fatalf("room with no shop = %q", got)
	}

	// VNUM 1 has the C puff special; assign_the_shopkeepers preserves it as
	// SHOP_FUNC when it attaches the ordinary shopkeeper procedure.
	specialWorld, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}},
		Mobs:  []parser.Mob{{VNum: 1, Keywords: "Puff", ShortDesc: "Puff"}},
		Shops: []parser.ShopProto{{VNum: 2, KeeperVNum: 1, WithWho: 127}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(specialWorld.StopAITicker)
	wantSpecial := "Vnum:       [    2], Rnum: [    1]\n\r" +
		"Rooms:      None!\n\r" +
		"Shopkeeper: Puff (#1), Special Function: YES\n\r" +
		"Customers:  None\n\r" +
		"Produces:   Nothing!\n\r" +
		"Buys:       Nothing!\n\r" +
		"Buy at:     [0.00], Sell at: [0.00], Open: [0-0, 0-0]\n\r" +
		"Bits:       NOBITS \n\r"
	if got := showDetailedShop(specialWorld, "1", 1001); got != wantSpecial {
		t.Fatalf("special keeper bytes\ngot:  %q\nwant: %q", got, wantSpecial)
	}
}

func TestShopBuyCreditsKeeperBankThroughDispatch(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Market"}},
		Mobs: []parser.Mob{{
			VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant",
			Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{{
			VNum: 6001, Keywords: "apple", ShortDesc: "a red apple",
			LongDesc: "A red apple is here.", TypeFlag: int(game.ItemOther), Cost: 100, Weight: 1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	keeper, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	keeper.SetGold(14995)
	shops := game.NewShopManager()
	shops.AddShop(&game.Shop{
		VNum: 77, KeeperVNum: 5001, Flags: game.ShopFlagUsesBank,
		Rooms: []int{1001}, SellTypes: []int{6001}, BuyTypes: []int{int(game.ItemOther)},
		ProfitBuy: 1, ProfitSell: 1,
	})
	w.SetShopManager(shops)
	manager := newTestManager(t, w, nil)
	buyer := makeCommandTestSession(t, manager, "Buyer", 1, 1001)
	buyer.player.SetGold(1000)
	price := shops.GetShopByVNum(77).BuyPrice(100, buyer.player.GetCha())
	if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
		t.Fatalf("real buy dispatch: %v", err)
	}
	if got := buyer.player.GetGold(); got != 1000-price {
		t.Fatalf("buyer coins after purchase = %d, want %d", got, 1000-price)
	}
	shop, coins, ok := w.SnapshotShopAccount(77, keeper)
	wantBank := 14995 + price - 15000
	if !ok || coins != 15000 || shop.Bank != wantBank {
		t.Fatalf("shop account after purchase: found=%v coins=%d bank=%d, want true/15000/%d", ok, coins, shop.Bank, wantBank)
	}

	wizard := makeCommandTestSession(t, manager, "Showgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(wizard, "show", []string{"shops", "1"}); err != nil {
		t.Fatalf("real detail dispatch: %v", err)
	}
	wantBalance := fmt.Sprintf("Coins:      [%9d], Bank: [%9d] (Total: %d)\n\r", coins, shop.Bank, coins+shop.Bank)
	if got := renderedOutput(wizard); !strings.Contains(got, wantBalance) {
		t.Fatalf("report omitted live post-purchase balance %q: %q", wantBalance, got)
	}
}

func TestShowShopsEmptyAndMissingKeeper(t *testing.T) {
	for _, shops := range [][]parser.ShopProto{nil, {{VNum: 1, KeeperVNum: -1, BuyProfit: 2, SellProfit: 0.5, WithWho: 127}}} {
		w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Shops: shops})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(w.StopAITicker)
		s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
		if err := cmdShow(s, []string{"shops"}); err != nil {
			t.Fatal(err)
		}
		got := renderedOutput(s)
		if len(shops) == 0 {
			if got != "\n\r" {
				t.Fatalf("empty %q", got)
			}
		} else if !strings.Contains(got, "  1        1       -1    <NONE>   0.50   2.00    _______\n\r") {
			t.Fatalf("missing keeper/room %q", got)
		}
	}
}

// Literal paging opts in narrowly: normal paging retains its existing bytes.
func TestShopLiteralPagerDeliveryAndSnoop(t *testing.T) {
	for _, heartbeat := range []bool{false, true} {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Actor", 40, 1001)
		spy := makeCommandTestSession(t, m, "Spy", 40, 1001)
		actor.snoopBy = spy
		if heartbeat {
			m.BeginHeartbeatOutput()
		}
		pageLiteralString(actor, "first\n\rsecond\n\r")
		if heartbeat {
			if len(actor.send) != 0 || len(spy.send) != 0 {
				t.Fatal("literal page escaped heartbeat")
			}
			m.EndHeartbeatOutput()
		}
		if got := renderedOutput(actor); got != "first\n\rsecond\n\r" {
			t.Fatalf("literal actor heartbeat=%v: %q", heartbeat, got)
		}
		if got := renderedOutput(spy); got != "% first\n\rsecond\n\r%%" {
			t.Fatalf("literal snoop heartbeat=%v: %q", heartbeat, got)
		}
		if actor.pagerLiteral || actor.IsPaging() {
			t.Fatal("single page retained literal mode")
		}
		// C's write path emits the string's spelling verbatim (comm.c:2335,
		// :1348, :1380), so ordinary paging renders LFCR pairs as written.
		PageString(actor, "normal\n\r")
		if got := renderedOutput(actor); got != "normal\n\r" {
			t.Fatalf("ordinary paging changed: %q", got)
		}
		_ = renderedOutput(spy)
		pageLiteralString(actor, strings.Repeat("row\n\r", 30))
		if !actor.IsPaging() || !actor.pagerLiteral {
			t.Fatal("literal multipage inactive")
		}
		if got := renderedOutput(actor); !strings.Contains(got, "row\n\r") || strings.Contains(got, "row\r\n") {
			t.Fatalf("literal page boundaries: %q", got)
		}
		_ = renderedOutput(spy)
		actor.navigatePager("q")
		if actor.pagerLiteral || actor.IsPaging() {
			t.Fatal("quit retained literal mode")
		}
		PageString(actor, "ordinary\n\r")
		if got := renderedOutput(actor); got != "ordinary\n\r" {
			t.Fatalf("after quit: %q", got)
		}
	}
}

func TestShopLiteralPagerMixedHeartbeatSnoop(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Actor", 40, 1001)
	spy := makeCommandTestSession(t, m, "Spy", 40, 1001)
	actor.snoopBy = spy
	m.BeginHeartbeatOutput()
	actor.Send("normal\n")
	pageLiteralString(actor, "literal\n\r")
	m.EndHeartbeatOutput()
	if got := renderedOutput(actor); got != "normal\r\nliteral\n\r" {
		t.Fatalf("mixed actor: %q", got)
	}
	if got := renderedOutput(spy); got != "% normal\r\nliteral\n\r%%" {
		t.Fatalf("mixed snoop: %q", got)
	}
}
