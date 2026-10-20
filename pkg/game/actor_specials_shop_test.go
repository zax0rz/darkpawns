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
// command declined so the pet's command continues, and list/buy/sell/value as
// an explicit descriptor gap.
func TestSpecShopKeeperActorPortsC(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Shop"})
	keeper := newSpecProcTestMob(t, w, 1001, 10)
	shops := NewShopManager()
	shops.AddShop(&Shop{VNum: 1, KeeperVNum: keeper.GetVNum(), Rooms: []int{1001}})
	w.SetShopManager(shops)

	pet := newSpecProcTestMob(t, w, 1001, 1)
	ctx := CharacterSpecialContext{Actor: pet}

	if got := specShopKeeperActor(w, ctx, keeper, "emote", ""); got != SpecialDeclined {
		t.Fatalf("emote = %v, want SpecialDeclined", got)
	}
	if got := specShopKeeperActor(w, ctx, keeper, "steal", ""); got != SpecialHandled {
		t.Fatalf("steal = %v, want SpecialHandled", got)
	}
	if got := specShopKeeperActor(w, ctx, keeper, "list", ""); got != SpecialUnsupported {
		t.Fatalf("list = %v, want SpecialUnsupported (page_string needs a descriptor)", got)
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
