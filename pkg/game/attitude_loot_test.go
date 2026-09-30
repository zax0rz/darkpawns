package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C preserves carrying in make_corpse (fight.c:399-402), then attitude_loot
// calls do_get("all corpse") (fight.c:1120), which walks forward
// (act.item.c:246-253). Test both boundaries so two reversals cannot cancel.
func TestAttitudeLootCorpseOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Loot proof", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 4209, Keywords: "dragon", ShortDesc: "Kaerdein the Ice Dragon", Level: 80}},
		Objs: []parser.Obj{
			{VNum: 8037, Keywords: "sword", ShortDesc: "a short sword", TypeFlag: ITEM_WEAPON, WearFlags: [4]int{1}},
			{VNum: 8019, Keywords: "tunic", ShortDesc: "a frayed tunic", TypeFlag: ITEM_ARMOR, WearFlags: [4]int{1}},
			{VNum: 8020, Keywords: "backpack", ShortDesc: "a backpack", TypeFlag: ITEM_CONTAINER, WearFlags: [4]int{1}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	victim := NewPlayer(1, "Dragonpeer", 1001)
	observer := NewPlayer(2, "Observer", 1001)
	for _, p := range []*Player{victim, observer} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	dragon, err := w.SpawnMob(4209, 1001)
	if err != nil {
		t.Fatal(err)
	}
	var objects []*ObjectInstance
	// Creation gives sword, tunic, backpack; obj_to_char prepends each.
	for _, vnum := range []int{8037, 8019, 8020} {
		obj, err := w.SpawnObject(vnum, -1)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
			t.Fatal(err)
		}
		objects = append(objects, obj)
	}
	corpse := w.makeCorpse(victim.Name, victim.GetSex(), victim.Inventory.FindItems(""), nil, 1001, combat.TYPE_UNDEFINED, 0, false)
	if err := w.MoveObjectToRoomFront(corpse, 1001); err != nil {
		t.Fatal(err)
	}
	if len(corpse.Contains) != 3 || corpse.Contains[0] != objects[2] || corpse.Contains[1] != objects[1] || corpse.Contains[2] != objects[0] {
		t.Fatalf("corpse order = %v; want backpack, tunic, sword", corpse.Contains)
	}
	var got strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == observer.Name {
			got.Write(msg)
		}
	}
	w.attitudeLootMob(dragon, victim)
	want := "Kaerdein the Ice Dragon gets a backpack from the corpse of Dragonpeer.\r\n" +
		"Kaerdein the Ice Dragon gets a frayed tunic from the corpse of Dragonpeer.\r\n" +
		"Kaerdein the Ice Dragon gets a short sword from the corpse of Dragonpeer.\r\n" +
		"Kaerdein the Ice Dragon junks a short sword. It vanishes in a puff of smoke!\r\n" +
		"Kaerdein the Ice Dragon junks a frayed tunic. It vanishes in a puff of smoke!\r\n"
	if got.String() != want {
		t.Fatalf("loot transcript = %q; want %q", got.String(), want)
	}
	if len(corpse.Contains) != 0 || len(dragon.Inventory) != 1 || dragon.Inventory[0] != objects[2] {
		t.Fatalf("after loot: corpse=%v inventory=%v; want empty corpse and retained backpack", corpse.Contains, dragon.Inventory)
	}
}
