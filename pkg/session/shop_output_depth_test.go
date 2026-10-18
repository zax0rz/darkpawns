package session

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func shopOutputFixture(t *testing.T, itemWeight int) (*game.World, *Manager, *game.MobInstance, *game.Shop) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Market", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant",
			Level: 10, Gold: 1000, HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{
			{VNum: 6001, Keywords: "apple red", ShortDesc: "a red apple", TypeFlag: int(game.ItemOther), Cost: 10, Weight: itemWeight},
			{VNum: 6002, Keywords: "pebble", ShortDesc: "a pebble", TypeFlag: int(game.ItemOther), Cost: 0, Weight: 0},
			{VNum: 6003, Keywords: "manual", ShortDesc: "a field manual", TypeFlag: int(game.ItemOther), Cost: 15, Weight: 1},
		},
		Shops: []parser.ShopProto{{
			VNum: 77, KeeperVNum: 5001, Rooms: []int{1001}, Products: []int{6001},
			BuyTypes: []int{int(game.ItemOther)}, BuyProfit: 1, SellProfit: 1,
			Messages: [7]string{"%s No such item.", "%s You don't have that.", "%s I don't buy that.", "%s I can't afford that!", "%s You're too poor!", "%s That'll be %d coins, thanks.", "%s I'll give you %d coins for that."},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	keeper, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	keeper.SetGold(1000)
	addShopKeeperStock(t, w, keeper, 6001)
	return w, m, keeper, w.GetAllShops()[0]
}

type shopStockRoller struct{}

func (shopStockRoller) Number(_, _ int) int { return 1 }
func (shopStockRoller) Dice(num, _ int) int { return num }
func (shopStockRoller) IntN(_ int) int      { return 1 }

func TestShopMovesFiniteStockOnBuyAndRetainsSale(t *testing.T) {
	w, manager, keeper, shop := shopOutputFixture(t, 1)
	seller := addShopOutputSession(t, manager, "Stockbuyer")
	seller.player.SetGold(100)
	seller.player.Inventory.MaxWeight = 100
	manual := addShopKeeperStock(t, w, keeper, 6003)

	oldRoller := combat.GetRoller()
	combat.SetRoller(shopStockRoller{})
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	if err := ExecuteCommand(seller, "buy", []string{"manual"}); err != nil {
		t.Fatalf("buy finite stock: %v", err)
	}
	if got := seller.player.Inventory.FindItems("manual"); len(got) != 1 || got[0] != manual {
		t.Fatalf("bought finite stock = %v, want the keeper's original object %d", got, manual.ID)
	}
	if manual.Location.Kind != game.ObjInInventory || manual.Location.OwnerKind != game.OwnerPlayer || manual.Location.PlayerName != seller.player.Name {
		t.Fatalf("bought object's location = %+v, want buyer inventory", manual.Location)
	}
	for _, stock := range keeper.InventorySnapshot() {
		if stock == manual {
			t.Fatal("finite stock remained with keeper after purchase")
		}
	}

	if err := ExecuteCommand(seller, "sell", []string{"manual"}); err != nil {
		t.Fatalf("sell finite stock: %v", err)
	}
	if got := seller.player.Inventory.FindItems("manual"); len(got) != 0 {
		t.Fatalf("seller still carries the sold item: %v", got)
	}
	if manual.Location.Kind != game.ObjInInventory || manual.Location.OwnerKind != game.OwnerMob || manual.Location.MobID != keeper.ID {
		t.Fatalf("sold object's location = %+v, want keeper inventory", manual.Location)
	}
	found := false
	for _, stock := range keeper.InventorySnapshot() {
		found = found || stock == manual
	}
	if !found {
		t.Fatalf("sold item %d was not retained in keeper inventory", manual.ID)
	}
	if _, _, ok := w.SnapshotShopAccount(shop.VNum, keeper); !ok {
		t.Fatal("shop account disappeared after stock transfer")
	}
}

func addShopKeeperStock(t *testing.T, w *game.World, keeper *game.MobInstance, vnum int) *game.ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject shop stock %d: %v", vnum, err)
	}
	if err := w.MoveObjectToMobInventoryFront(obj, keeper); err != nil {
		t.Fatalf("MoveObjectToMobInventoryFront shop stock %d: %v", vnum, err)
	}
	return obj
}

func addShopOutputSession(t *testing.T, manager *Manager, name string) *Session {
	t.Helper()
	s := makeCommandTestSession(t, manager, name, 1, 1001)
	setShopOutputTestAttributes(s.player)
	registerTestSession(t, manager, s, name)
	return s
}

func setShopOutputTestAttributes(player *game.Player) {
	player.Stats = game.CharStats{Str: 10, Dex: 10, Int: 10, Wis: 10, Con: 10, Cha: 10}
	player.CopyBaseAttributes()
}

func TestShopBuyCapacityRefusalsThroughDispatch(t *testing.T) {
	t.Run("item count", func(t *testing.T) {
		w, manager, _, _ := shopOutputFixture(t, 1)
		buyer := addShopOutputSession(t, manager, "Buyer")
		buyer.player.SetGold(100)
		proto, ok := w.GetObjPrototype(6002)
		if !ok {
			t.Fatal("pebble prototype missing")
		}
		for range buyer.player.Inventory.GetCapacity() {
			item := w.NewObjectFromProto(proto, -1)
			if err := buyer.player.Inventory.AddItem(item); err != nil {
				t.Fatalf("fill inventory: %v", err)
			}
		}
		if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
			t.Fatalf("buy dispatch: %v", err)
		}
		if got, want := renderedOutput(buyer), "apple: You can't carry any more items.\n\r"; got != want {
			t.Fatalf("count refusal = %q, want %q", got, want)
		}
		if got := buyer.player.GetGold(); got != 100 {
			t.Fatalf("gold after count refusal = %d, want 100", got)
		}
		if got, want := buyer.player.Inventory.GetItemCount(), buyer.player.Inventory.GetCapacity(); got != want {
			t.Fatalf("inventory count after count refusal = %d, want unchanged at %d", got, want)
		}
	})

	t.Run("carry weight", func(t *testing.T) {
		_, manager, _, _ := shopOutputFixture(t, 1000)
		buyer := addShopOutputSession(t, manager, "Buyer")
		buyer.player.SetGold(100)
		if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
			t.Fatalf("buy dispatch: %v", err)
		}
		if got, want := renderedOutput(buyer), "apple: You can't carry that much weight.\n\r"; got != want {
			t.Fatalf("weight refusal = %q, want %q", got, want)
		}
		if got := buyer.player.GetGold(); got != 100 {
			t.Fatalf("gold after weight refusal = %d, want 100", got)
		}
		if got := buyer.player.Inventory.GetItemCount(); got != 0 {
			t.Fatalf("inventory count after weight refusal = %d, want 0", got)
		}
	})

	t.Run("available capacity control", func(t *testing.T) {
		_, manager, _, _ := shopOutputFixture(t, 1)
		buyer := addShopOutputSession(t, manager, "Buyer")
		buyer.player.SetGold(100)
		buyer.player.Inventory.MaxWeight = 100
		if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
			t.Fatalf("buy dispatch: %v", err)
		}
		if got := buyer.player.Inventory.GetItemCount(); got != 1 {
			t.Fatalf("inventory count with available capacity = %d, want 1", got)
		}
		if got := buyer.player.GetGold(); got != 90 {
			t.Fatalf("gold with available capacity = %d, want 90", got)
		}
	})
}

func TestShopBuySuccessHasCTransactionAudiencesThroughDispatch(t *testing.T) {
	w, manager, _, _ := shopOutputFixture(t, 1)
	buyer := addShopOutputSession(t, manager, "Buyer")
	observer := addShopOutputSession(t, manager, "Observer")
	buyer.player.SetGold(100)
	buyer.player.Inventory.MaxWeight = 100
	if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
		t.Fatalf("buy dispatch: %v", err)
	}
	if got, want := renderedOutput(observer), "Buyer buys a red apple.\r\n"; got != want {
		t.Fatalf("room audience = %q, want %q; buyer output=%q gold=%d carry=%d/%d weight=%d/%d", got, want, renderedOutput(buyer), buyer.player.GetGold(), buyer.player.Inventory.GetItemCount(), buyer.player.Inventory.GetCapacity(), buyer.player.CarriedWeight(), buyer.player.Inventory.MaxWeight)
	}
	if got, want := renderedOutput(buyer), "A merchant tells you, 'That'll be 10 coins, thanks.'\r\nYou now have a red apple.\n\r"; got != want {
		t.Fatalf("buyer output = %q, want %q", got, want)
	}
	if got := buyer.player.GetGold(); got != 90 {
		t.Fatalf("buyer coins = %d, want 90", got)
	}
	if got := w.GetAllObjects(); len(got) < 1 {
		t.Fatal("successful buy did not create a registered object")
	}
}

func TestShopKeeperTellUsesCRoomOrderForAbbreviatedTargetThroughDispatch(t *testing.T) {
	_, manager, _, _ := shopOutputFixture(t, 1)
	buyer := addShopOutputSession(t, manager, "Buyer")
	peer := addShopOutputSession(t, manager, "BuyerPeer")
	buyer.player.SetGold(100)
	buyer.player.Inventory.MaxWeight = 100
	if buyer.player.GetRoomEntrySequence() >= peer.player.GetRoomEntrySequence() {
		t.Fatal("test precondition: peer must be first in C's room.people order")
	}
	if err := ExecuteCommand(buyer, "buy", []string{"apple"}); err != nil {
		t.Fatalf("buy dispatch: %v", err)
	}
	if got, want := renderedOutput(buyer), "You now have a red apple.\n\r"; got != want {
		t.Fatalf("nonselected buyer output = %q, want %q", got, want)
	}
	if got, want := renderedOutput(peer), "Buyer buys a red apple.\r\nA merchant tells you, 'That'll be 10 coins, thanks.'\r\n"; got != want {
		t.Fatalf("C abbreviation target output = %q, want %q", got, want)
	}
	if got := buyer.player.Inventory.GetItemCount(); got != 1 {
		t.Fatalf("buyer inventory count = %d, want 1", got)
	}
}

func TestShopSellSuccessHasCTransactionAudiencesThroughDispatch(t *testing.T) {
	w, manager, keeper, shop := shopOutputFixture(t, 1)
	seller := addShopOutputSession(t, manager, "Seller")
	observer := addShopOutputSession(t, manager, "Observer")
	proto, ok := w.GetObjPrototype(6001)
	if !ok {
		t.Fatal("apple prototype missing")
	}
	item, err := w.SpawnObject(proto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject sale item: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(item, seller.player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory sale item: %v", err)
	}
	price := shop.SellPrice(item.GetCost(), seller.player.GetCha())
	if err := ExecuteCommand(seller, "sell", []string{"apple"}); err != nil {
		t.Fatalf("sell dispatch: %v", err)
	}
	if got, want := renderedOutput(observer), "Seller sells a red apple.\r\n"; got != want {
		t.Fatalf("room audience = %q, want %q", got, want)
	}
	want := "A merchant tells you, 'I'll give you " + itoaShopOutput(price) + " coins for that.'\r\nThe shopkeeper now has a red apple.\n\r"
	if got := renderedOutput(seller); got != want {
		t.Fatalf("seller output = %q, want %q", got, want)
	}
	if got := seller.player.GetGold(); got != price {
		t.Fatalf("seller coins = %d, want %d", got, price)
	}
	if got := keeper.GetGold(); got != 1000-price {
		t.Fatalf("keeper coins = %d, want %d", got, 1000-price)
	}
}

func TestShopKeeperStealSpecialRunsBeforeCommandThroughDispatch(t *testing.T) {
	_, manager, _, _ := shopOutputFixture(t, 1)
	actor := addShopOutputSession(t, manager, "Thief")
	observer := addShopOutputSession(t, manager, "Observer")
	if err := ExecuteCommand(actor, "steal", []string{"merchant"}); err != nil {
		t.Fatalf("steal dispatch: %v", err)
	}
	if got, want := renderedOutput(actor), "You are slapped by a merchant.\r\nA merchant shouts 'Thief is a bloody thief!'\r\n"; got != want {
		t.Fatalf("shopkeeper actor response = %q, want %q", got, want)
	}
	if got, want := renderedOutput(observer), "A merchant slaps Thief.\r\n"; got != want {
		t.Fatalf("room audience = %q, want %q", got, want)
	}
}

func TestShopKeeperStealSocialUsesCNewestPrefixMatchThroughDispatch(t *testing.T) {
	_, manager, _, _ := shopOutputFixture(t, 1)
	actor := addShopOutputSession(t, manager, "Shopoutput")
	peer := addShopOutputSession(t, manager, "Shopoutputpeer")
	if err := ExecuteCommand(actor, "steal", []string{"merchant"}); err != nil {
		t.Fatalf("steal dispatch: %v", err)
	}
	if got, want := renderedOutput(actor), "A merchant slaps Shopoutputpeer.\r\nA merchant shouts 'Shopoutput is a bloody thief!'\r\n"; got != want {
		t.Fatalf("shopkeeper actor response = %q, want %q", got, want)
	}
	if got, want := renderedOutput(peer), "You are slapped by a merchant.\r\n"; got != want {
		t.Fatalf("newest prefix-match target response = %q, want %q", got, want)
	}
}

func itoaShopOutput(value int) string {
	return strconv.Itoa(value)
}
