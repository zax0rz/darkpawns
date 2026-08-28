package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newCarveDepthWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 12, Keywords: "fish filet", ShortDesc: "a fish filet", TypeFlag: ITEM_FOOD},
			{VNum: 13, Keywords: "bird leg", ShortDesc: "a bird leg", TypeFlag: ITEM_FOOD},
			{VNum: 14, Keywords: "rabbit chops", ShortDesc: "some rabbit chops", TypeFlag: ITEM_FOOD},
			{VNum: 8015, Keywords: "meat", ShortDesc: "a piece of meat", TypeFlag: ITEM_FOOD},
			{VNum: 8010, Keywords: "bread", ShortDesc: "a loaf of bread", TypeFlag: ITEM_FOOD},
			{VNum: 8011, Keywords: "headless corpse", ShortDesc: "a beheaded corpse", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}},
			{VNum: 9000, Keywords: "fish carve_meat corpse", ShortDesc: "the corpse of fish", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}},
			{VNum: 9001, Keywords: "weapon", ShortDesc: "a blunt weapon", TypeFlag: ITEM_WEAPON, Values: [4]int{0, 1, 4, 1}},
			{VNum: 9002, Keywords: "sword", ShortDesc: "a carving sword", TypeFlag: ITEM_WEAPON, Values: [4]int{0, 1, 4, 3}},
			{VNum: 9003, Keywords: "scrap", ShortDesc: "a scrap", TypeFlag: ITEM_OTHER},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	ch := NewPlayer(1, "Carvegod", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, ch
}

func carveDepthCorpse(t *testing.T, w *World, keywords string) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(9000, 1001)
	if err != nil {
		t.Fatalf("SpawnObject corpse: %v", err)
	}
	obj.Runtime.Keywords = keywords
	w.AddItemToRoom(obj, 1001)
	return obj
}

func TestDoCarveCEntryBranches(t *testing.T) {
	tests := []struct {
		name string
		call func(*World, *Player)
		want string
	}{
		{
			name: "missing object",
			want: "You can't seem to find a xyzzy to carve!",
		},
		{
			name: "non corpse",
			call: func(w *World, _ *Player) {
				obj, err := w.SpawnObject(8010, 1001)
				if err != nil {
					t.Fatalf("SpawnObject bread: %v", err)
				}
				w.AddItemToRoom(obj, 1001)
			},
			want: "Your initials are about all you can carve in that!",
		},
		{
			name: "non carvable corpse",
			call: func(w *World, _ *Player) { carveDepthCorpse(t, w, "headless corpse") },
			want: "There's no way you could ever eat THAT!!!",
		},
		{
			name: "no wield",
			call: func(w *World, _ *Player) { carveDepthCorpse(t, w, "fish carve_meat corpse") },
			want: "You don't have anything to carve with!",
		},
		{
			name: "wrong wield",
			call: func(w *World, ch *Player) {
				carveDepthCorpse(t, w, "fish carve_meat corpse")
				weapon, err := w.SpawnObject(9001, -1)
				if err != nil {
					t.Fatalf("SpawnObject weapon: %v", err)
				}
				if err := ch.Equipment.SetSlot(SlotWield, weapon); err != nil {
					t.Fatalf("SetSlot: %v", err)
				}
			},
			want: "You can't carve with that!",
		},
		{
			name: "full arms",
			call: func(w *World, ch *Player) {
				carveDepthCorpse(t, w, "fish carve_meat corpse")
				for i := 0; i < ch.MaxCarryItems(); i++ {
					filler, err := w.SpawnObject(9003, -1)
					if err != nil {
						t.Fatalf("SpawnObject filler: %v", err)
					}
					if err := w.MoveObjectToPlayerInventory(filler, ch); err != nil {
						t.Fatalf("MoveObjectToPlayerInventory filler: %v", err)
					}
				}
			},
			want: "Your arms are already full!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ch := newCarveDepthWorld(t)
			if tt.call != nil {
				tt.call(w, ch)
			}
			got := DoCarve(ch, map[string]string{
				"missing object":      "xyzzy",
				"non corpse":          "bread",
				"non carvable corpse": "headless",
				"no wield":            "corpse",
				"wrong wield":         "corpse",
				"full arms":           "corpse",
			}[tt.name], w).MessageToCh
			if got != tt.want {
				t.Fatalf("MessageToCh = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDoCarveSuccessUsesCFoodAndRoomContract(t *testing.T) {
	w, ch := newCarveDepthWorld(t)
	corpse := carveDepthCorpse(t, w, "fish carve_meat corpse")
	child, err := w.SpawnObject(9003, -1)
	if err != nil {
		t.Fatalf("SpawnObject child: %v", err)
	}
	if err := w.MoveObjectToContainer(child, corpse); err != nil {
		t.Fatalf("MoveObjectToContainer: %v", err)
	}
	weapon, err := w.SpawnObject(9002, -1)
	if err != nil {
		t.Fatalf("SpawnObject weapon: %v", err)
	}
	if err := ch.Equipment.SetSlot(SlotWield, weapon); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}

	result := DoCarve(ch, "corpse", w)
	if !result.Success {
		t.Fatal("DoCarve reported failure")
	}
	if result.MessageToCh != "You carve up some meat from the the corpse of fish." {
		t.Fatalf("MessageToCh = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Carvegod carves up some meat from the the corpse of fish." {
		t.Fatalf("MessageToRoom = %q", result.MessageToRoom)
	}
	if _, ok := ch.Inventory.FindItem("meat"); !ok {
		t.Fatal("carved food was not added to inventory")
	}
	if _, ok := w.ResolveObjectInRoom(ch, "corpse"); ok {
		t.Fatal("corpse remained in the room")
	}
	if _, ok := w.ResolveObjectInRoom(ch, "scrap"); !ok {
		t.Fatal("corpse contents were not dumped into the room")
	}
}

func TestDoCarveFoodMappings(t *testing.T) {
	for _, tt := range []struct {
		name     string
		keywords string
		wantVNum int
	}{
		{name: "meat", keywords: "fish carve_meat corpse", wantVNum: 8015},
		{name: "fish", keywords: "fish carve_fish corpse", wantVNum: 12},
		{name: "bird", keywords: "bird carve_bird corpse", wantVNum: 13},
		{name: "rabbit", keywords: "rabbit carve_rabbit corpse", wantVNum: 14},
		{name: "fallback", keywords: "strange carve_thing corpse", wantVNum: 8015},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, ch := newCarveDepthWorld(t)
			carveDepthCorpse(t, w, tt.keywords)
			weapon, err := w.SpawnObject(9002, -1)
			if err != nil {
				t.Fatalf("SpawnObject weapon: %v", err)
			}
			if err := ch.Equipment.SetSlot(SlotWield, weapon); err != nil {
				t.Fatalf("SetSlot: %v", err)
			}
			if result := DoCarve(ch, "corpse", w); !result.Success {
				t.Fatal("DoCarve reported failure")
			}
			if got := ch.Inventory.Items[len(ch.Inventory.Items)-1].GetVNum(); got != tt.wantVNum {
				t.Fatalf("food vnum = %d, want %d", got, tt.wantVNum)
			}
		})
	}
}

func TestMakeCorpseUsesMobKeywordList(t *testing.T) {
	w, _ := newCarveDepthWorld(t)
	corpse := w.makeCorpse("fish", 0, nil, nil, 1001, -1, 0, true, "fish carve_meat")
	if got := corpse.GetKeywords(); got != "fish carve_meat corpse" {
		t.Fatalf("corpse keywords = %q, want %q", got, "fish carve_meat corpse")
	}
	if !strings.Contains(corpse.GetShortDesc(), "corpse of fish") {
		t.Fatalf("corpse short description = %q", corpse.GetShortDesc())
	}
}

func TestHandleMobDeathPreservesMobKeywordListOnCorpse(t *testing.T) {
	w, ch := newCarveDepthWorld(t)
	w.mobs[4350] = &parser.Mob{
		VNum: 4350, Keywords: "fish carve_meat", ShortDesc: "fish",
		HP: parser.DiceRoll{Num: 1, Sides: 1},
	}
	mob, err := w.SpawnMob(4350, ch.GetRoom())
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	w.handleMobDeath(mob, nil, -1)
	corpse, ok := w.ResolveObjectInRoom(ch, "corpse")
	if !ok {
		t.Fatal("mob death did not create a corpse")
	}
	if got := corpse.GetKeywords(); got != "fish carve_meat corpse" {
		t.Fatalf("corpse keywords = %q, want %q", got, "fish carve_meat corpse")
	}
}
