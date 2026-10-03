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

// The C wear act precedes equip_char's anti-alignment refusal, on both loot
// passes. Raw standalone mobiles exercise the World-aware caller directly.
func TestAttitudeLootWearBeforeAlignmentRefusal(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Objs: []parser.Obj{{VNum: 200, Keywords: "armor", ShortDesc: "an evil ward", TypeFlag: ITEM_ARMOR, WearFlags: [4]int{1 | 1<<3}, ExtraFlags: [4]int{FlagAntiEvil}, Cost: 200, Values: [4]int{10}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	killer := NewMob(&parser.Mob{VNum: 300, Keywords: "looter", ShortDesc: "a looter", Alignment: -500, Position: 8, DefaultPos: 8, Str: 11, Dex: 11, Int: 11, Wis: 11, Con: 11, Cha: 11, AC: 100}, 1001)
	killer.ID = 300
	observer := NewPlayer(1, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	item, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	w.activeMobs[killer.ID] = killer
	w.mu.Unlock()
	if err := w.MoveObjectToMobInventoryFront(item, killer); err != nil {
		t.Fatal(err)
	}
	// This caller must supply its World even for a mobile constructed directly.
	var room, actor strings.Builder
	w.MessageSink = func(name string, b []byte) {
		if name == observer.Name {
			room.Write(b)
		}
	}
	w.MobileMessageSink = func(body *MobInstance, b []byte) {
		if body == killer {
			actor.Write(b)
		}
	}
	victim := NewPlayer(2, "Victim", 1001)
	w.attitudeLootMob(killer, victim)
	wantRoom := "A looter wears an evil ward on its body.\r\nA looter is zapped by an evil ward and instantly lets go of it.\r\n"
	if room.String() != wantRoom+wantRoom {
		t.Fatalf("loot wear/refusal room order = %q", room.String())
	}
	wantActor := "You wear an evil ward on your body.\r\nYou are zapped by an evil ward and instantly let go of it.\r\n"
	if actor.String() != wantActor+wantActor {
		t.Fatalf("loot wear/refusal actor order = %q", actor.String())
	}
	if killer.Equipped(mobWearBody) != nil || len(killer.Inventory) != 1 || killer.Inventory[0] != item || item.Location != LocInventoryMob(killer.ID) || killer.GetAC() != 100 {
		t.Fatal("anti-alignment loot did not preserve inventory and armor")
	}
}

func TestMobileWearNPCGates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		where     int
		configure func(*testing.T, *World, *MobInstance, *ObjectInstance)
		want      string
		slot      int
	}{
		{"wear-bit", eqWearBody, func(_ *testing.T, _ *World, _ *MobInstance, o *ObjectInstance) { o.Prototype.WearFlags = [4]int{} }, "You can't wear a shield there.\r\n", -1},
		{"occupied", eqWearBody, func(t *testing.T, w *World, m *MobInstance, _ *ObjectInstance) {
			o, e := w.SpawnObject(201, -1)
			if e != nil {
				t.Fatal(e)
			}
			if e = w.EquipMobileObject(m, o, eqWearBody); e != nil {
				t.Fatal(e)
			}
		}, alreadyWearing[eqWearBody], -1},
		{"flesh-alter", eqWearWield, func(_ *testing.T, _ *World, m *MobInstance, o *ObjectInstance) {
			o.Prototype.WearFlags = [4]int{1 | 1<<13}
			m.SetAffected(affFleshAlter)
		}, "Your flesh is altered, you can't wield anything!\r\n", -1},
		{"heavy", eqWearWield, func(_ *testing.T, _ *World, _ *MobInstance, o *ObjectInstance) {
			o.Prototype.WearFlags = [4]int{1 | 1<<13}
			o.Prototype.Weight = 12
		}, "It is too heavy for you to use.\r\n", -1},
		{"two-handed", eqWearWield, func(t *testing.T, w *World, m *MobInstance, o *ObjectInstance) {
			o.Prototype.WearFlags = [4]int{1 | 1<<13}
			o.Prototype.ExtraFlags[0] |= 1 << extraFlagTwoHanded
			held, e := w.SpawnObject(201, -1)
			if e != nil {
				t.Fatal(e)
			}
			if e = w.EquipMobileObject(m, held, eqWearHold); e != nil {
				t.Fatal(e)
			}
		}, "Both hands must be free to wield that.\r\n", -1},
		{"shield-conflict", eqWearShield, func(t *testing.T, w *World, m *MobInstance, o *ObjectInstance) {
			o.Prototype.WearFlags = [4]int{1 | 1<<9}
			weapon, e := w.SpawnObject(200, -1)
			if e != nil {
				t.Fatal(e)
			}
			proto := *weapon.Prototype
			proto.ExtraFlags[0] |= 1 << extraFlagTwoHanded
			weapon.Prototype = &proto
			if e = w.EquipMobileObject(m, weapon, eqWearWield); e != nil {
				t.Fatal(e)
			}
		}, "Both your hands are occupied with your weapon at the moment.\r\n", -1},
		{"paired-finger", eqWearFingerR, func(t *testing.T, w *World, m *MobInstance, o *ObjectInstance) {
			o.Prototype.WearFlags = [4]int{1 | 1<<1}
			ring, e := w.SpawnObject(201, -1)
			if e != nil {
				t.Fatal(e)
			}
			if e = w.EquipMobileObject(m, ring, eqWearFingerR); e != nil {
				t.Fatal(e)
			}
		}, "You slide a shield on to your left ring finger.\r\n", eqWearFingerL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, m, _ := zoneArmedMob(t)
			m.UnequipItem(mobWearWield)
			o, e := w.SpawnObject(201, -1)
			if e != nil {
				t.Fatal(e)
			}
			proto := *o.Prototype
			proto.ShortDesc = "a shield"
			proto.WearFlags = [4]int{1 | 1<<3}
			o.Prototype = &proto
			if e = w.MoveObjectToMobInventoryFront(o, m); e != nil {
				t.Fatal(e)
			}
			tc.configure(t, w, m, o)
			var actor strings.Builder
			w.MobileMessageSink = func(body *MobInstance, b []byte) {
				if body == m {
					actor.Write(b)
				}
			}
			w.performMobileWear(m, o, tc.where)
			if actor.String() != tc.want {
				t.Fatalf("NPC wear gate=%q, want %q", actor.String(), tc.want)
			}
			if tc.slot < 0 && o.Location != LocInventoryMob(m.ID) {
				t.Fatal("refused wear detached item")
			}
			if tc.slot >= 0 && m.Equipped(tc.slot) != o {
				t.Fatal("paired slot not used")
			}
		})
	}
}
