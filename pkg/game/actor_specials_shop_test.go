package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSpecShopKeeperActorPortsC pins SPECIAL(shop_keeper) for a mobile actor
// (src/shop.c:942-1002). The special is installed at boot by
// assign_the_shopkeepers (src/shop.c:1226-1243), so it is absent from
// MobSpecAssign; the session lane recovers it from the shop index and this test
// pins the body's own gates: shop-room, awake, steal handled, every other
// command declined, and the four shopping arms handled the way C answers TRUE
// for them.
func TestSpecShopKeeperActorPortsC(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Shop"})
	keeper := newSpecProcTestMob(t, w, 1001, 10)
	keeper.SetPosition(combat.PosStanding)
	shops := NewShopManager()
	shop := &Shop{VNum: 1, KeeperVNum: keeper.GetVNum(), Rooms: []int{1001}, OpenHour1: 0, CloseHour1: 23, OpenHour2: 0, CloseHour2: 23}
	shops.AddShop(shop)
	w.SetShopManager(shops)

	// One unsorted stock item so shopping_list's sort_keeper_objs pass runs.
	stockObj := NewObjectInstance(&parser.Obj{
		VNum: 5001, Keywords: "sword long", ShortDesc: "a long sword", Cost: 100,
	}, -1)
	if err := w.MoveObjectToMobInventoryFront(stockObj, keeper); err != nil {
		t.Fatalf("MoveObjectToMobInventoryFront: %v", err)
	}

	pet := newSpecProcTestMob(t, w, 1001, 1)
	ctx := CharacterSpecialContext{Actor: pet}

	if got := specShopKeeperActor(w, ctx, keeper, "emote", " neighs"); got != SpecialDeclined {
		t.Fatalf("emote = %v, want SpecialDeclined", got)
	}
	if got := specShopKeeperActor(w, ctx, keeper, "steal", ""); got != SpecialHandled {
		t.Fatalf("steal = %v, want SpecialHandled", got)
	}
	// C: shopping_list runs is_ok, then sort_keeper_objs, then
	// page_string(ch->desc, …) — which returns on a NULL descriptor
	// (src/modify.c:430-433) — and shop_keeper answers TRUE.
	// slide_obj randomly discards non-produced stock on number(0, 5) == 0
	// (src/shop.c:678-682). Pin the keep branch; an unseeded draw made this
	// assertion fail one time in six even when the sort ran correctly (R3/R5h).
	draws := installDrawRecorder(t, 1)
	if got := specShopKeeperActor(w, ctx, keeper, "list", ""); got != SpecialHandled {
		t.Fatalf("list = %v, want SpecialHandled", got)
	}
	if shop.LastSort != 1 {
		t.Fatalf("shop sort = %d, want 1: the list arm must sort the keeper's stock", shop.LastSort)
	}
	if draws.count() != 1 || draws.lastBounds() != [2]int{0, 5} {
		t.Fatalf("shop sort draws = %v, want one number(0, 5)", draws.calls)
	}
	// A pet with no gold and no stock match still answers TRUE.
	if got := specShopKeeperActor(w, ctx, keeper, "buy", " sword"); got != SpecialHandled {
		t.Fatalf("buy = %v, want SpecialHandled", got)
	}
	if got := specShopKeeperActor(w, ctx, keeper, "value", " sword"); got != SpecialHandled {
		t.Fatalf("value = %v, want SpecialHandled", got)
	}
	if got := specShopKeeperActor(w, ctx, keeper, "sell", " sword"); got != SpecialHandled {
		t.Fatalf("sell = %v, want SpecialHandled", got)
	}

	// C's ok_shop_room gate: outside the shop's rooms the special declines.
	pet.SetRoom(1002)
	if got := specShopKeeperActor(w, ctx, keeper, "steal", ""); got != SpecialDeclined {
		t.Fatalf("steal outside the shop = %v, want SpecialDeclined", got)
	}
	// C's AWAKE(keeper) gate.
	pet.SetRoom(1001)
	keeper.SetPosition(combat.PosSleeping)
	if got := specShopKeeperActor(w, ctx, keeper, "steal", ""); got != SpecialDeclined {
		t.Fatalf("steal beside a sleeping keeper = %v, want SpecialDeclined", got)
	}
}
