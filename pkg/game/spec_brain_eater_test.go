package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func prepareBrainEaterTest(t *testing.T, level int) (*World, *Player, *MobInstance, *ObjectInstance, func() string) {
	t.Helper()
	w, player, transcript := newSpecProcTestWorld(t)
	registerBeheadPrototypes(t, w)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), level)
	mob.Str = 11
	mob.SetPosition(combat.PosStanding)
	corpse := registerBeheadObject(t, w, &parser.Obj{
		VNum: 4017, Keywords: "corpse guard trainee", ShortDesc: "the corpse of a guard trainee",
		LongDesc: "The corpse of a guard trainee is lying here.", TypeFlag: ITEM_CONTAINER,
		Values: [4]int{1, 0, -1, 1},
	})
	transcript()
	return w, player, mob, corpse, transcript
}

func TestSpecBrainEater_BeheadsCorpseAndLevelsInstance(t *testing.T) {
	w, player, mob, original, transcript := prepareBrainEaterTest(t, 22)
	content := registerBeheadObject(t, w, &parser.Obj{
		VNum: 4018, Keywords: "silver ring", ShortDesc: "a silver ring",
		LongDesc: "A silver ring lies here.", TypeFlag: ITEM_OTHER, WearFlags: [4]int{1},
	})
	if err := w.MoveObjectToContainer(content, original); err != nil {
		t.Fatalf("MoveObjectToContainer: %v", err)
	}

	if !specBrainEater(w, nil, mob, "", "") {
		t.Fatal("expected brain_eater to handle a qualifying corpse")
	}

	if got, want := transcript(), "A test mob rips the head off the corpse of a guard trainee!\r\nA test mob pulls the brain out of the head and eats it with a noisy\r\nslurp, blood and drool flying everywhere.\r\n"; got != want {
		t.Errorf("room audience = %q, want %q", got, want)
	}
	if mob.GetLevel() != 23 {
		t.Errorf("mob level = %d, want 23", mob.GetLevel())
	}
	if mob.Prototype.Level != 22 {
		t.Errorf("prototype level = %d, want unchanged 22", mob.Prototype.Level)
	}

	var head, beheaded *ObjectInstance
	for _, obj := range mob.Inventory {
		if obj.GetVNum() == 16 {
			head = obj
		}
	}
	for _, obj := range w.GetItemsInRoom(player.GetRoomVNum()) {
		if obj.GetVNum() == 17 {
			beheaded = obj
		}
	}
	if head == nil {
		t.Fatal("brain_eater did not place the head in mob inventory")
	}
	if head.GetShortDesc() != "a bloody head ripped from the corpse of a guard trainee" {
		t.Errorf("head short description = %q", head.GetShortDesc())
	}
	if beheaded == nil {
		t.Fatal("brain_eater did not leave a beheaded corpse")
	}
	if got := beheaded.GetKeywords(); got != "corpse guard trainee headless beheaded" {
		t.Errorf("beheaded corpse keywords = %q", got)
	}
	if len(beheaded.Contains) != 1 || beheaded.Contains[0] != content {
		t.Fatalf("beheaded corpse contents = %#v, want transferred ring", beheaded.Contains)
	}
	if original.Location != LocNowhere() || content.Location != LocContainer(beheaded.ID) {
		t.Errorf("locations after behead: original=%#v content=%#v", original.Location, content.Location)
	}
}

func TestSpecBrainEater_DamrollGrowthAtLevelThirty(t *testing.T) {
	w, _, mob, _, _ := prepareBrainEaterTest(t, 30)
	if got := mob.GetDamroll(); got != 0 {
		t.Fatalf("initial damroll = %d, want 0", got)
	}

	if !specBrainEater(w, nil, mob, "", "") {
		t.Fatal("expected brain_eater to handle a qualifying corpse")
	}
	if got := mob.GetDamroll(); got != 2 {
		t.Errorf("damroll after brain = %d, want 2", got)
	}
	if got := mob.GetDamageRoll().Plus; got != mob.Prototype.Damage.Plus {
		t.Errorf("damage-roll plus = %d, want prototype plus %d", got, mob.Prototype.Damage.Plus)
	}
}

func TestSpecBrainEater_SkipsNonQualifyingObjects(t *testing.T) {
	tests := []struct {
		name  string
		proto *parser.Obj
	}{
		{
			name:  "not a container",
			proto: &parser.Obj{VNum: 4020, Keywords: "corpse guard", ShortDesc: "a corpse", TypeFlag: ITEM_OTHER, Values: [4]int{0, 0, 0, 1}},
		},
		{
			name:  "zero corpse marker",
			proto: &parser.Obj{VNum: 4021, Keywords: "corpse guard", ShortDesc: "a corpse", TypeFlag: ITEM_CONTAINER},
		},
		{
			name:  "headless",
			proto: &parser.Obj{VNum: 4022, Keywords: "corpse guard headless", ShortDesc: "a headless corpse", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}},
		},
		{
			name:  "missing corpse keyword",
			proto: &parser.Obj{VNum: 4023, Keywords: "guard remains", ShortDesc: "some remains", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, transcript := newSpecProcTestWorld(t)
			registerBeheadPrototypes(t, w)
			mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 22)
			mob.Str = 11
			mob.SetPosition(combat.PosStanding)
			registerBeheadObject(t, w, tt.proto)
			transcript()

			if specBrainEater(w, nil, mob, "", "") {
				t.Fatal("non-qualifying object was handled")
			}
			if strings.Contains(transcript(), "pulls the brain") {
				t.Fatal("non-qualifying object emitted brain-eater output")
			}
		})
	}
}

func TestSpecBrainEater_EntryGates(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Player, *MobInstance)
	}{
		{name: "command", setup: func(_ *Player, _ *MobInstance) {}},
		{name: "fighting", setup: func(player *Player, mob *MobInstance) { mob.SetFighting(player.GetName()) }},
		{name: "sleeping", setup: func(_ *Player, mob *MobInstance) { mob.SetPosition(combat.PosSleeping) }},
		{name: "negative hp", setup: func(_ *Player, mob *MobInstance) { mob.SetHealth(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, mob, _, transcript := prepareBrainEaterTest(t, 22)
			tt.setup(player, mob)
			cmd := ""
			if tt.name == "command" {
				cmd = "look"
			}
			if specBrainEater(w, nil, mob, cmd, "") {
				t.Fatal("entry gate was handled")
			}
			if strings.Contains(transcript(), "pulls the brain") {
				t.Fatal("entry gate emitted brain-eater output")
			}
		})
	}
}
