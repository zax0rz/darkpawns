package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Exercise registered buy dispatch, preserving C's leading-count syntax and
// affordable-prefix state (src/shop.c:489-589). Gold and objects are proof.
func TestShopCountedBuyStateThroughDispatch(t *testing.T) {
	for _, tc := range []struct {
		command                string
		gold, count, remaining int
		text                   string
	}{
		{"buy 2 apple", 100, 2, 80, "You now have a red apple (x 2).\n\r"},
		{"buy apple 3", 100, 1, 90, "You now have a red apple.\n\r"},
		{"buy 5 apple", 25, 2, 5, "You can only afford 2."},
		{"buy 0 apple", 100, 0, 100, "What do you want to buy??"},
		{"buy -1 apple", 100, 0, 100, "No such item."},
		{"buy apple", 0, 0, 0, "You're too poor!"},
	} {
		t.Run(tc.command+tc.text, func(t *testing.T) {
			_, manager, keeper, _ := shopOutputFixture(t, 1)
			buyer := addShopOutputSession(t, manager, "Countbuyer")
			peer := addShopOutputSession(t, manager, "Countpeer")
			setShopOutputTestAttributes(buyer.player)
			buyer.player.SetLevel(10)
			buyer.player.Inventory.SetCapacity(10, 0, 10, 10)
			buyer.player.SetGold(tc.gold)
			shopDispatch(t, buyer, tc.command)
			if got := buyer.player.GetGold(); got != tc.remaining {
				t.Errorf("gold=%d want %d", got, tc.remaining)
			}
			if got := buyer.player.Inventory.GetItemCount(); got != tc.count {
				t.Errorf("inventory=%d want %d", got, tc.count)
			}
			if got := keeper.GetGold(); got != 1000+tc.gold-tc.remaining {
				t.Errorf("keeper gold=%d", got)
			}
			if got := renderedOutput(buyer); !strings.Contains(got, tc.text) {
				t.Errorf("output=%q want %q", got, tc.text)
			}
			if tc.count > 0 && !strings.Contains(renderedOutput(peer), "Countbuyer buys a red apple") {
				t.Errorf("peer=%q", renderedOutput(peer))
			}
			if tc.count == 0 && strings.Contains(renderedOutput(peer), "buys") {
				t.Errorf("refused purchase reached peer=%q", renderedOutput(peer))
			}
		})
	}
}

func TestShopCountedBuyStopsAtEachCapacity(t *testing.T) {
	for _, weight := range []int{0, 100} {
		_, manager, keeper, _ := shopOutputFixture(t, weight)
		buyer := addShopOutputSession(t, manager, "Capacitybuyer")
		setShopOutputTestAttributes(buyer.player)
		buyer.player.SetLevel(10)
		buyer.player.Inventory.SetCapacity(10, 0, 10, 10)
		buyer.player.SetGold(10000)
		before := buyer.player.GetGold()
		shopDispatch(t, buyer, "buy 1000 apple")
		count := buyer.player.Inventory.GetItemCount()
		if count < 1 {
			t.Fatal("did not buy affordable prefix")
		}
		if count > buyer.player.MaxCarryItems() || buyer.player.CarriedWeight() > buyer.player.MaxCarryWeight() {
			t.Fatal("purchase exceeded C capacity")
		}
		if buyer.player.GetGold() != before-10*count || keeper.GetGold() != 1000+10*count {
			t.Fatal("coins do not match transferred objects")
		}
		expected := "You can only hold"
		if weight > 0 {
			expected = "You can only carry"
		}
		if !strings.Contains(renderedOutput(buyer), expected) {
			t.Errorf("missing partial refusal %q: %q", expected, renderedOutput(buyer))
		}
	}
}

func shopDispatch(t *testing.T, s *Session, line string) {
	t.Helper()
	fields := strings.Fields(line)
	if err := ExecuteCommand(s, fields[0], fields[1:]); err != nil {
		t.Fatal(err)
	}
}

func TestShopCountedSaleStateThroughDispatch(t *testing.T) {
	for _, tc := range []struct {
		command    string
		cash, sold int
		text       string
	}{
		{"sell 2 apple", 1000, 2, "The shopkeeper now has a red apple (x 2).\n\r"},
		{"sell 5 apple", 1000, 3, "You only have 3 of those."},
		{"sell 3 apple", 15, 1, "I can only afford to buy 1 of those."},
		{"sell apple", 0, 0, "I can't afford that!"},
		{"sell all", 1000, 0, "You don't have that."},
		{"sell 0 apple", 1000, 0, "What do you want to sell??"},
		{"sell", 1000, 0, "What do you want to sell??"},
		{"sell -1 apple", 1000, 0, "You don't have that."},
		{"sell 2.apple ignored", 1000, 1, "The shopkeeper now has a red apple.\n\r"},
	} {
		t.Run(tc.command+tc.text, func(t *testing.T) {
			w, manager, keeper, _ := shopOutputFixture(t, 1)
			seller := addShopOutputSession(t, manager, "Countseller")
			peer := addShopOutputSession(t, manager, "Salepeer")
			seller.player.SetGold(5)
			keeper.SetGold(tc.cash)
			for i := 0; i < 3; i++ {
				item, err := w.SpawnObject(6001, -1)
				if err != nil {
					t.Fatal(err)
				}
				if err := w.MoveObjectToPlayerInventory(item, seller.player); err != nil {
					t.Fatal(err)
				}
			}
			shopDispatch(t, seller, tc.command)
			if got := seller.player.GetGold(); got != 5+10*tc.sold {
				t.Errorf("gold=%d want %d", got, 5+10*tc.sold)
			}
			if got := seller.player.Inventory.GetItemCount(); got != 3-tc.sold {
				t.Errorf("inventory=%d", got)
			}
			if got := keeper.GetGold(); got != tc.cash-10*tc.sold {
				t.Errorf("keeper cash=%d", got)
			}
			if !strings.Contains(renderedOutput(seller), tc.text) {
				t.Errorf("output=%q want %q", renderedOutput(seller), tc.text)
			}
			if tc.sold > 0 && !strings.Contains(renderedOutput(peer), "Countseller sells a red apple") {
				t.Errorf("peer=%q", renderedOutput(peer))
			}
			if tc.sold == 0 && renderedOutput(peer) != "" {
				t.Errorf("refused sale leaked=%q", renderedOutput(peer))
			}
		})
	}
}

func TestShopValueQuotesWithoutMutationThroughDispatch(t *testing.T) {
	w, manager, keeper, _ := shopOutputFixture(t, 1)
	seller := addShopOutputSession(t, manager, "Valuer")
	peer := addShopOutputSession(t, manager, "Valuepeer")
	seller.player.SetGold(17)
	obj, err := w.SpawnObject(6001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, seller.player); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ line, want string }{
		{"value", "What do you want me to valuate??"},
		{"value apple ignored", "I'll give you 10 gold coins for that!"},
		{"value nonexistent", "You don't have that."},
	} {
		shopDispatch(t, seller, tc.line)
		if !strings.Contains(renderedOutput(seller), tc.want) {
			t.Errorf("output lacks %q", tc.want)
		}
		if seller.player.GetGold() != 17 || keeper.GetGold() != 1000 || seller.player.Inventory.GetItemCount() != 1 {
			t.Fatal("valuation mutated money or inventory")
		}
	}
	if renderedOutput(peer) != "" {
		t.Fatal("valuation leaked to room")
	}
}

func TestShopKeeperGatesThroughDispatch(t *testing.T) {
	for _, tc := range []string{"closed", "invisible", "sleeping", "wrong room", "god still invisible"} {
		t.Run(tc, func(t *testing.T) {
			_, manager, keeper, shop := shopOutputFixture(t, 1)
			buyer := addShopOutputSession(t, manager, "Gatebuyer")
			peer := addShopOutputSession(t, manager, "Gatepeer")
			buyer.player.SetGold(100)
			want := ""
			switch tc {
			case "closed":
				shop.OpenHour1 = 29
				want = "A merchant exclaims, 'Come back later!'\r\n"
			case "invisible", "god still invisible":
				buyer.player.SetInvisLevel(33)
				want = "A merchant exclaims, 'I don't trade with someone I can't see!'\r\n"
				if tc == "god still invisible" {
					buyer.player.SetLevel(33)
				}
			case "sleeping":
				keeper.SetPosition(4)
			case "wrong room":
				shop.Rooms = []int{9999}
			}
			shopDispatch(t, buyer, "buy apple")
			got := renderedOutput(buyer)
			if want == "" {
				if got != "Sorry, but you cannot do that here!\r\n" {
					t.Errorf("fallback=%q", got)
				}
				if renderedOutput(peer) != "" {
					t.Errorf("fallback leaked=%q", renderedOutput(peer))
				}
			} else {
				if got != want || renderedOutput(peer) != want {
					t.Errorf("actor=%q peer=%q want=%q", got, renderedOutput(peer), want)
				}
			}
			if buyer.player.GetGold() != 100 || buyer.player.Inventory.GetItemCount() != 0 || keeper.GetGold() != 1000 {
				t.Fatal("keeper refusal mutated transaction state")
			}
		})
	}
}

func TestShopTradeExpressionRefusesSaleAndValueThroughDispatch(t *testing.T) {
	w, manager, keeper, shop := shopOutputFixture(t, 1)
	seller := addShopOutputSession(t, manager, "Exprseller")
	shop.BuyWords = []string{"MAGIC"}
	item, err := w.SpawnObject(6001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(item, seller.player); err != nil {
		t.Fatal(err)
	}
	seller.player.SetGold(17)
	for _, cmd := range []string{"sell apple", "value apple"} {
		shopDispatch(t, seller, cmd)
		if !strings.Contains(renderedOutput(seller), "I don't buy that.") {
			t.Errorf("%s did not refuse", cmd)
		}
		if seller.player.GetGold() != 17 || seller.player.Inventory.GetItemCount() != 1 || keeper.GetGold() != 1000 {
			t.Fatal("expression refusal mutated state")
		}
	}
}

func TestBankHiddenTransactionsThroughDispatch(t *testing.T) {
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Objs: []parser.Obj{{VNum: 8034, Keywords: "atm", ShortDesc: "an ATM"}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	atm, err := w.SpawnObject(8034, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoom(atm, 1001); err != nil {
		t.Fatal(err)
	}
	actor := addShopOutputSession(t, manager, "Banker")
	peer := addShopOutputSession(t, manager, "Bankpeer")
	actor.player.SetGold(100)
	actor.player.SetLevel(33)
	actor.player.SetInvisLevel(33)
	shopDispatch(t, actor, "deposit 20")
	if actor.player.GetGold() != 80 || actor.player.GetBankGold() != 20 {
		t.Fatal("deposit state")
	}
	shopDispatch(t, actor, "withdraw 10")
	if actor.player.GetGold() != 90 || actor.player.GetBankGold() != 10 {
		t.Fatal("withdraw state")
	}
	if got := renderedOutput(peer); got != "" {
		t.Errorf("hidden transactions leaked=%q", got)
	}
	actor.player.SetInvisLevel(0)
	shopDispatch(t, actor, "deposit 10")
	if got := renderedOutput(peer); got != "Banker makes a bank transaction.\r\n" {
		t.Errorf("visible transaction control=%q", got)
	}
	if actor.player.GetGold() != 80 || actor.player.GetBankGold() != 20 {
		t.Fatal("visible deposit state")
	}
}

func TestShopListMagicAnnotationsAndLineEndingsThroughDispatch(t *testing.T) {
	for _, kind := range []int{game.ITEM_WAND, game.ITEM_STAFF} {
		for _, charges := range []int{1, 3} {
			w, manager, _, _ := shopOutputFixture(t, 1)
			proto, ok := w.GetObjPrototype(6001)
			if !ok {
				t.Fatal("missing prototype")
			}
			proto.TypeFlag = kind
			proto.Values = [4]int{1, 3, charges, 0}
			viewer := addShopOutputSession(t, manager, "Listreader")
			shopDispatch(t, viewer, "list")
			got := renderedOutput(viewer)
			if strings.Contains(got, "\r\n") || !strings.HasPrefix(got, " ##   Available   Item                                               Cost\n\r") {
				t.Errorf("C list line endings=%q", got)
			}
			if strings.Contains(got, "(partially used)") != (charges < 3) {
				t.Errorf("type=%d charges=%d list=%q", kind, charges, got)
			}
		}
	}
}

func TestShopListUsesPagerThroughDispatch(t *testing.T) {
	w, manager, keeper, _ := shopOutputFixture(t, 1)
	old := combat.GetRoller()
	combat.SetRoller(shopStockRoller{})
	t.Cleanup(func() { combat.SetRoller(old) })
	for i := 0; i < 30; i++ {
		obj := game.NewObjectInstance(&parser.Obj{VNum: 7000 + i, Keywords: fmt.Sprintf("stock%d", i), ShortDesc: fmt.Sprintf("a stock object %02d", i), Cost: 10}, -1)
		if err := w.MoveObjectToMobInventoryFront(obj, keeper); err != nil {
			t.Fatal(err)
		}
	}
	viewer := addShopOutputSession(t, manager, "Pagereader")
	shopDispatch(t, viewer, "list")
	if !viewer.IsPaging() || !viewer.pagerLiteral {
		t.Fatal("long list did not enter C literal page_string")
	}
	first := renderedOutput(viewer)
	if !strings.HasSuffix(first, "\n") || strings.Contains(first, "\r\n") {
		t.Errorf("first C page boundary=%q", first)
	}
	if strings.Count(first, "stock object") >= 30 {
		t.Fatal("first page leaked the entire list")
	}
	viewer.navigatePager("")
	if rest := renderedOutput(viewer); !strings.HasPrefix(rest, "\r") || !strings.Contains(rest, "stock object") {
		t.Errorf("second page=%q", rest)
	}
}
