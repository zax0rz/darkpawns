package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSyntheticMoneyIsTakeable — C's create_money zeroes the wear flags and
// then sets ITEM_WEAR_TAKE (src/handler.c:1484-1486), so CAN_WEAR(obj,
// ITEM_WEAR_TAKE) is true for every money pile. The port's prototype-less
// pile carried no wear flags at all, so IsTakeable() reported false and both
// the mob get (npcGetDirect) and the scavenger gate (canGetObject) refused it.
func TestSyntheticMoneyIsTakeable(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Coin proof", Zone: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	for _, amount := range []int{1, 5, 250} {
		money := w.createMoneyObject(amount)
		if money == nil {
			t.Fatalf("createMoneyObject(%d) = nil", amount)
		}
		if !money.IsTakeable() {
			t.Fatalf("createMoneyObject(%d).IsTakeable() = false; create_money sets ITEM_WEAR_TAKE", amount)
		}
	}
}

// TestMobGetsCorpseGoldFromAttitudeLoot — attitude_loot runs
// do_get(ch, "all corpse") (src/fight.c:1120); perform_get_from_container
// passes can_take_obj and the room sees "$n gets $p from $P."
// (src/act.item.c:200-212) before get_check_money converts the pile to gold.
// The port's mob-get path skipped the coins because the synthetic pile was not
// takeable, so the room saw nothing and the mob kept no gold.
func TestMobGetsCorpseGoldFromAttitudeLoot(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Loot gold", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 4209, Keywords: "dragon", ShortDesc: "Kaerdein the Ice Dragon", Level: 80}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	victim := NewPlayer(1, "Goldpeer", 1001)
	observer := NewPlayer(2, "Goldwatch", 1001)
	for _, p := range []*Player{victim, observer} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	dragon, err := w.SpawnMob(4209, 1001)
	if err != nil {
		t.Fatal(err)
	}

	// C make_corpse moves GET_GOLD(ch) into a create_money pile inside the
	// corpse (src/fight.c:406-413); 5 coins renders as "a tiny pile of gold
	// coins" (src/handler.c:1397-1425).
	corpse := w.makeCorpse(victim.Name, victim.GetSex(), nil, nil, 1001, combat.TYPE_UNDEFINED, 5, false)
	if err := w.MoveObjectToRoomFront(corpse, 1001); err != nil {
		t.Fatal(err)
	}
	if len(corpse.Contains) != 1 {
		t.Fatalf("corpse contains %d objects; want the money pile", len(corpse.Contains))
	}
	if !corpse.Contains[0].IsTakeable() {
		t.Fatal("corpse gold pile is not takeable; create_money sets ITEM_WEAR_TAKE")
	}

	var got strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == observer.Name {
			got.Write(msg)
		}
	}
	w.attitudeLootMob(dragon, victim)

	want := "Kaerdein the Ice Dragon gets a tiny pile of gold coins from the corpse of Goldpeer.\r\n"
	if got.String() != want {
		t.Fatalf("loot transcript = %q; want %q", got.String(), want)
	}
	if dragon.GetGold() != 5 {
		t.Fatalf("dragon gold = %d; want 5 (get_check_money credits the pile)", dragon.GetGold())
	}
	if len(corpse.Contains) != 0 {
		t.Fatalf("corpse still holds %d objects; the pile was extracted", len(corpse.Contains))
	}
}
