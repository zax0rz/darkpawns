package admin

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newTestWorldForWrite creates a minimal World with one of each entity type
// for testing world write methods.
func newTestWorldForWrite(t *testing.T) *game.World {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Test Room", Zone: 1,
				Flags:  []string{"0", "0", "0", "0"},
				Sector: 0,
				Exits:  map[string]parser.Exit{},
			},
			{
				VNum: 1002, Name: "Second Room", Zone: 1,
				Flags:  []string{"0", "0", "0", "0"},
				Sector: 1,
				Exits:  map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1001},
				},
			},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "a guard", LongDesc: "A guard stands here.", Level: 5, AC: 50, Gold: 10, Exp: 100, Alignment: 0, Position: 0, DefaultPos: 0, Sex: 0},
			{VNum: 2002, ShortDesc: "a merchant", LongDesc: "A merchant eyes you.", THAC0: 10, Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10},
		},
		Objs: []parser.Obj{
			{
				VNum: 3001, Keywords: "sword", ShortDesc: "a steel sword",
				LongDesc: "A steel sword lies here.", TypeFlag: 5,
				Weight: 5, Cost: 100,
				WearFlags: [4]int{1 << 13, 0, 0, 0},
				Values:    [4]int{0, 3, 5, 0},
				ExtraFlags: [4]int{0, 0, 0, 0},
			},
			{
				VNum: 3002, Keywords: "shield", ShortDesc: "a wooden shield",
				LongDesc: "A wooden shield is here.", TypeFlag: 11,
				Weight: 8, Cost: 50,
			},
		},
		Zones: []parser.Zone{
			{Number: 1, Name: "Test Zone", TopRoom: 2000, Lifespan: 15, ResetMode: 1},
		},
	}

	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// ---------------------------------------------------------------------------
// Room write methods
// ---------------------------------------------------------------------------

func TestSetRoomName_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetRoomName(1001, "Updated Name") {
		t.Fatal("SetRoomName returned false for existing room")
	}
	room := w.GetRoomInWorld(1001)
	if room.Name != "Updated Name" {
		t.Errorf("room name = %q, want %q", room.Name, "Updated Name")
	}
}

func TestSetRoomName_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomName(9999, "nope") {
		t.Error("SetRoomName should return false for missing room")
	}
}

func TestSetRoomDescription_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetRoomDescription(1001, "A dark room.") {
		t.Fatal("SetRoomDescription returned false")
	}
	room := w.GetRoomInWorld(1001)
	if room.Description != "A dark room." {
		t.Errorf("got %q, want %q", room.Description, "A dark room.")
	}
}

func TestSetRoomDescription_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomDescription(9999, "desc") {
		t.Error("expected false for missing room")
	}
}

func TestSetRoomFlags_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	flags := []string{"1", "0", "0", "0"}
	if !w.SetRoomFlags(1001, flags) {
		t.Fatal("SetRoomFlags returned false")
	}
	room := w.GetRoomInWorld(1001)
	if len(room.Flags) != 4 || room.Flags[0] != "1" {
		t.Errorf("room flags = %v, want [1 0 0 0]", room.Flags)
	}
}

func TestSetRoomFlags_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomFlags(9999, []string{"0", "0", "0", "0"}) {
		t.Error("expected false for missing room")
	}
}

func TestSetRoomSector_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetRoomSector(1001, 2) {
		t.Fatal("SetRoomSector returned false")
	}
	room := w.GetRoomInWorld(1001)
	if room.Sector != 2 {
		t.Errorf("sector = %d, want 2", room.Sector)
	}
}

func TestSetRoomSector_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomSector(9999, 1) {
		t.Error("expected false for missing room")
	}
}

func TestSetRoomExit_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetRoomExit(1001, "east", 1002, -1) {
		t.Fatal("SetRoomExit returned false")
	}
	room := w.GetRoomInWorld(1001)
	exit, ok := room.Exits["east"]
	if !ok {
		t.Fatal("exit 'east' not found")
	}
	if exit.ToRoom != 1002 {
		t.Errorf("exit to room = %d, want 1002", exit.ToRoom)
	}
}

func TestSetRoomExit_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomExit(9999, "north", 1002, -1) {
		t.Error("expected false for missing room")
	}
}

func TestSetRoomExit_ExistingUpdated(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetRoomExit(1002, "north", 1001, -1) {
		t.Fatal("SetRoomExit returned false")
	}
	room := w.GetRoomInWorld(1002)
	if exit, ok := room.Exits["north"]; !ok || exit.ToRoom != 1001 {
		t.Errorf("north exit to room = %d, want 1001", exit.ToRoom)
	}

	// Update with key
	if !w.SetRoomExit(1002, "north", 1001, 3001) {
		t.Fatal("SetRoomExit update returned false")
	}
	room2 := w.GetRoomInWorld(1002)
	if room2.Exits["north"].Key != 3001 {
		t.Errorf("exit key = %d, want 3001", room2.Exits["north"].Key)
	}
}

func TestSetRoomExtraDescs_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	descs := []parser.ExtraDesc{{Keywords: "test", Description: "A test."}}
	if !w.SetRoomExtraDescs(1001, descs) {
		t.Fatal("SetRoomExtraDescs returned false")
	}
	room := w.GetRoomInWorld(1001)
	if len(room.ExtraDescs) != 1 || room.ExtraDescs[0].Keywords != "test" {
		t.Error("extra descs not set correctly")
	}
}

func TestSetRoomExtraDescs_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetRoomExtraDescs(9999, []parser.ExtraDesc{}) {
		t.Error("expected false for missing room")
	}
}

// ---------------------------------------------------------------------------
// Mob write methods
// ---------------------------------------------------------------------------

func TestSetMobShortDesc_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobShortDesc(2001, "a veteran guard") {
		t.Fatal("SetMobShortDesc returned false")
	}
	mob, ok := w.GetMobPrototype(2001)
	if !ok {
		t.Fatal("mob not found")
	}
	if mob.ShortDesc != "a veteran guard" {
		t.Errorf("short desc = %q, want %q", mob.ShortDesc, "a veteran guard")
	}
}

func TestSetMobShortDesc_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetMobShortDesc(9999, "whatever") {
		t.Error("expected false for missing mob")
	}
}

func TestSetMobLongDesc_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobLongDesc(2001, "A veteran guard stands here. He looks tough.") {
		t.Fatal("SetMobLongDesc returned false")
	}
	mob, ok := w.GetMobPrototype(2001)
	if !ok {
		t.Fatal("mob not found")
	}
	if mob.LongDesc != "A veteran guard stands here. He looks tough." {
		t.Errorf("unexpected long desc: %q", mob.LongDesc)
	}
}

func TestSetMobLevel(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobLevel(2001, 99) {
		t.Fatal("SetMobLevel returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.Level != 99 {
		t.Errorf("level = %d, want 99", mob.Level)
	}
}

func TestSetMobLevel_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetMobLevel(9999, 1) {
		t.Error("expected false")
	}
}

func TestSetMobAC(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobAC(2001, -100) {
		t.Fatal("SetMobAC returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.AC != -100 {
		t.Errorf("AC = %d, want -100", mob.AC)
	}
}

func TestSetMobHP(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobHP(2001, 5, 10, 30) {
		t.Fatal("SetMobHP returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.HP.Num != 5 || mob.HP.Sides != 10 || mob.HP.Plus != 30 {
		t.Errorf("HP = %dd%d+%d, want 5d10+30", mob.HP.Num, mob.HP.Sides, mob.HP.Plus)
	}
}

func TestSetMobHP_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetMobHP(9999, 1, 1, 1) {
		t.Error("expected false")
	}
}

func TestSetMobGold_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobGold(2001, 500) {
		t.Fatal("SetMobGold returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.Gold != 500 {
		t.Errorf("gold = %d, want 500", mob.Gold)
	}
}

func TestSetMobGold_ClampsNegative(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobGold(2001, -100) {
		t.Fatal("SetMobGold returned false for valid mob")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.Gold != 0 {
		t.Errorf("negative gold should clamp to 0, got %d", mob.Gold)
	}
}

func TestSetMobGold_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetMobGold(9999, 100) {
		t.Error("expected false")
	}
}

func TestSetMobExp_ClampsNegative(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.SetMobExp(2001, -1)
	mob, _ := w.GetMobPrototype(2001)
	if mob.Exp != 0 {
		t.Errorf("negative exp should clamp to 0, got %d", mob.Exp)
	}
}

func TestSetMobAlignment_ClampsRange(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.SetMobAlignment(2001, 2000)
	mob, _ := w.GetMobPrototype(2001)
	if mob.Alignment != 1000 {
		t.Errorf("alignment should clamp to 1000, got %d", mob.Alignment)
	}

	w.SetMobAlignment(2001, -2000)
	mob, _ = w.GetMobPrototype(2001)
	if mob.Alignment != -1000 {
		t.Errorf("alignment should clamp to -1000, got %d", mob.Alignment)
	}
}

func TestSetMobKeywords(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobKeywords(2001, "guard soldier") {
		t.Fatal("SetMobKeywords returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.Keywords != "guard soldier" {
		t.Errorf("keywords = %q, want %q", mob.Keywords, "guard soldier")
	}
}

func TestSetMobStats(t *testing.T) {
	w := newTestWorldForWrite(t)

	tests := []struct {
		name string
		set  func(vnum, val int) bool
	}{
		{"Str", w.SetMobStr},
		{"Int", w.SetMobInt},
		{"Wis", w.SetMobWis},
		{"Dex", w.SetMobDex},
		{"Con", w.SetMobCon},
		{"Cha", w.SetMobCha},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_valid", func(t *testing.T) {
			if !tt.set(2002, 18) {
				t.Fatal("set returned false")
			}
		})
		t.Run(tt.name+"_missing", func(t *testing.T) {
			if tt.set(9999, 10) {
				t.Error("expected false for missing mob")
			}
		})
	}
}

func TestSetMobTHAC0(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobTHAC0(2002, 5) {
		t.Fatal("SetMobTHAC0 returned false")
	}
	mob, _ := w.GetMobPrototype(2002)
	if mob.THAC0 != 5 {
		t.Errorf("THAC0 = %d, want 5", mob.THAC0)
	}
}

func TestSetMobDamage(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobDamage(2002, 2, 6, 3) {
		t.Fatal("SetMobDamage returned false")
	}
	mob, _ := w.GetMobPrototype(2002)
	if mob.Damage.Num != 2 || mob.Damage.Sides != 6 || mob.Damage.Plus != 3 {
		t.Errorf("Damage = %dd%d+%d, want 2d6+3", mob.Damage.Num, mob.Damage.Sides, mob.Damage.Plus)
	}
}

func TestSetMobPosition(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobPosition(2001, 1) {
		t.Fatal("SetMobPosition returned false")
	}
	if w.SetMobPosition(9999, 0) {
		t.Error("expected false")
	}
}

func TestSetMobDefaultPos(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobDefaultPos(2001, 1) {
		t.Fatal("SetMobDefaultPos returned false")
	}
	if w.SetMobDefaultPos(9999, 0) {
		t.Error("expected false")
	}
}

func TestSetMobSex(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobSex(2001, 1) {
		t.Fatal("SetMobSex returned false")
	}
	if w.SetMobSex(9999, 0) {
		t.Error("expected false")
	}
}

func TestSetMobRace(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetMobRace(2001, 3) {
		t.Fatal("SetMobRace returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if mob.Race != 3 {
		t.Errorf("race = %d, want 3", mob.Race)
	}
}

func TestSetMobActionFlags(t *testing.T) {
	w := newTestWorldForWrite(t)
	flags := []string{"SPEC", "SENTINEL"}
	if !w.SetMobActionFlags(2001, flags) {
		t.Fatal("SetMobActionFlags returned false")
	}
	mob, _ := w.GetMobPrototype(2001)
	if len(mob.ActionFlags) != 2 || mob.ActionFlags[0] != "SPEC" {
		t.Errorf("action flags = %v, want [SPEC SENTINEL]", mob.ActionFlags)
	}
}

func TestSetMobAffectFlags(t *testing.T) {
	w := newTestWorldForWrite(t)
	flags := []string{"INFRARED"}
	if !w.SetMobAffectFlags(2001, flags) {
		t.Fatal("SetMobAffectFlags returned false")
	}
}

// ---------------------------------------------------------------------------
// Object write methods
// ---------------------------------------------------------------------------

func TestSetObjShortDesc_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetObjShortDesc(3001, "an iron sword") {
		t.Fatal("SetObjShortDesc returned false")
	}
	obj, ok := w.GetObjPrototype(3001)
	if !ok {
		t.Fatal("obj not found")
	}
	if obj.ShortDesc != "an iron sword" {
		t.Errorf("short desc = %q, want %q", obj.ShortDesc, "an iron sword")
	}
}

func TestSetObjShortDesc_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetObjShortDesc(9999, "nothing") {
		t.Error("expected false")
	}
}

func TestSetObjLongDesc(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetObjLongDesc(3001, "An iron sword lies here.") {
		t.Fatal("SetObjLongDesc returned false")
	}
}

func TestSetObjWeight_ClampsNegative(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.SetObjWeight(3001, -5)
	obj, _ := w.GetObjPrototype(3001)
	if obj.Weight != 0 {
		t.Errorf("negative weight should clamp to 0, got %d", obj.Weight)
	}
}

func TestSetObjCost_ClampsNegative(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.SetObjCost(3001, -1)
	obj, _ := w.GetObjPrototype(3001)
	if obj.Cost != 0 {
		t.Errorf("negative cost should clamp to 0, got %d", obj.Cost)
	}
}

func TestSetObjKeywords(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetObjKeywords(3001, "sword iron blade") {
		t.Fatal("SetObjKeywords returned false")
	}
}

func TestSetObjTypeFlag(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetObjTypeFlag(3001, 7) {
		t.Fatal("SetObjTypeFlag returned false")
	}
	if w.SetObjTypeFlag(9999, 1) {
		t.Error("expected false")
	}
}

func TestSetObjValues(t *testing.T) {
	w := newTestWorldForWrite(t)
	vals := [4]int{1, 2, 3, 4}
	if !w.SetObjValues(3001, vals) {
		t.Fatal("SetObjValues returned false")
	}
	obj, _ := w.GetObjPrototype(3001)
	if obj.Values != vals {
		t.Errorf("values = %v, want %v", obj.Values, vals)
	}
}

func TestSetObjWearFlags(t *testing.T) {
	w := newTestWorldForWrite(t)
	flags := [4]int{1, 0, 0, 0}
	if !w.SetObjWearFlags(3001, flags) {
		t.Fatal("SetObjWearFlags returned false")
	}
}

func TestSetObjExtraFlags(t *testing.T) {
	w := newTestWorldForWrite(t)
	flags := [4]int{0, 1, 0, 0}
	if !w.SetObjExtraFlags(3001, flags) {
		t.Fatal("SetObjExtraFlags returned false")
	}
}

func TestSetObjAffects(t *testing.T) {
	w := newTestWorldForWrite(t)
	affects := []parser.ObjAffect{{Location: 1, Modifier: 5}}
	if !w.SetObjAffects(3001, affects) {
		t.Fatal("SetObjAffects returned false")
	}
}

func TestSetObjExtraDescs(t *testing.T) {
	w := newTestWorldForWrite(t)
	descs := []parser.ExtraDesc{{Keywords: "rusty", Description: "It's a bit rusty."}}
	if !w.SetObjExtraDescs(3001, descs) {
		t.Fatal("SetObjExtraDescs returned false")
	}
}

func TestSetObjMissingReturnsFalse(t *testing.T) {
	w := newTestWorldForWrite(t)
	tests := []struct {
		name string
		call func() bool
	}{
		{"SetObjKeywords", func() bool { return w.SetObjKeywords(9999, "x") }},
		{"SetObjTypeFlag", func() bool { return w.SetObjTypeFlag(9999, 1) }},
		{"SetObjValues", func() bool { return w.SetObjValues(9999, [4]int{}) }},
		{"SetObjWearFlags", func() bool { return w.SetObjWearFlags(9999, [4]int{}) }},
		{"SetObjExtraFlags", func() bool { return w.SetObjExtraFlags(9999, [4]int{}) }},
		{"SetObjAffects", func() bool { return w.SetObjAffects(9999, nil) }},
		{"SetObjExtraDescs", func() bool { return w.SetObjExtraDescs(9999, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.call() {
				t.Error("expected false for missing object")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Zone write methods
// ---------------------------------------------------------------------------

func TestSetZoneLifespan_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetZoneLifespan(1, 30) {
		t.Fatal("SetZoneLifespan returned false")
	}
	zone, ok := w.GetZone(1)
	if !ok {
		t.Fatal("zone not found")
	}
	if zone.Lifespan != 30 {
		t.Errorf("lifespan = %d, want 30", zone.Lifespan)
	}
}

func TestSetZoneLifespan_ClampsNegative(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.SetZoneLifespan(1, -5)
	zone, _ := w.GetZone(1)
	if zone.Lifespan != 0 {
		t.Errorf("negative lifespan should clamp to 0, got %d", zone.Lifespan)
	}
}

func TestSetZoneLifespan_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetZoneLifespan(99, 10) {
		t.Error("expected false for missing zone")
	}
}

func TestSetZoneResetMode_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if !w.SetZoneResetMode(1, 2) {
		t.Fatal("SetZoneResetMode returned false")
	}
	zone, _ := w.GetZone(1)
	if zone.ResetMode != 2 {
		t.Errorf("reset mode = %d, want 2", zone.ResetMode)
	}
}

func TestSetZoneResetMode_Invalid(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetZoneResetMode(1, 3) {
		t.Error("reset mode 3 should be invalid, returned true")
	}
	if w.SetZoneResetMode(1, -1) {
		t.Error("reset mode -1 should be invalid, returned true")
	}
}

func TestSetZoneResetMode_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.SetZoneResetMode(99, 1) {
		t.Error("expected false for missing zone")
	}
}

func TestAddZoneCommand_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	cmd := parser.ZoneCommand{Command: "M", Arg1: 2001, Arg2: 1, Arg3: 1001}
	if !w.AddZoneCommand(1, cmd) {
		t.Fatal("AddZoneCommand returned false")
	}
	zone, _ := w.GetZone(1)
	if len(zone.Commands) != 1 {
		t.Errorf("commands = %d, want 1", len(zone.Commands))
	}
}

func TestAddZoneCommand_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.AddZoneCommand(99, parser.ZoneCommand{}) {
		t.Error("expected false for missing zone")
	}
}

func TestRemoveZoneCommand_Valid(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.AddZoneCommand(1, parser.ZoneCommand{Command: "M"})
	w.AddZoneCommand(1, parser.ZoneCommand{Command: "O"})
	w.AddZoneCommand(1, parser.ZoneCommand{Command: "G"})

	if !w.RemoveZoneCommand(1, 1) {
		t.Fatal("RemoveZoneCommand returned false")
	}
	zone, _ := w.GetZone(1)
	if len(zone.Commands) != 2 || zone.Commands[1].Command != "G" {
		t.Errorf("after removal: commands = %v, want [M G]", zone.Commands)
	}
}

func TestRemoveZoneCommand_InvalidIndex(t *testing.T) {
	w := newTestWorldForWrite(t)
	w.AddZoneCommand(1, parser.ZoneCommand{Command: "M"})

	if w.RemoveZoneCommand(1, -1) {
		t.Error("expected false for negative index")
	}
	if w.RemoveZoneCommand(1, 10) {
		t.Error("expected false for out-of-range index")
	}
}

func TestRemoveZoneCommand_Missing(t *testing.T) {
	w := newTestWorldForWrite(t)
	if w.RemoveZoneCommand(99, 0) {
		t.Error("expected false for missing zone")
	}
}

// ---------------------------------------------------------------------------
// Shop write methods — requires a shop manager with a real ShopManager
// ---------------------------------------------------------------------------

// newWorldWithShops creates a world that also has shops configured.
func newWorldWithShops(t *testing.T) *game.World {
	t.Helper()
	w := newTestWorldForWrite(t)

	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{
		KeeperVNum: 2002,
		BuyTypes:   []int{1, 5},
		SellTypes:  []int{3001},
		ProfitBuy:  1.2,
		ProfitSell: 0.8,
		KeeperName: "Merchant",
		RoomVNum:   1001,
	})
	w.SetShopManager(sm)
	return w
}

func TestSetShopBuyTypes_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopBuyTypes(2002, []int{2, 3, 4}) {
		t.Fatal("SetShopBuyTypes returned false")
	}
	shop, ok := w.GetShopByKeeper(2002)
	if !ok {
		t.Fatal("shop not found")
	}
	if len(shop.BuyTypes) != 3 || shop.BuyTypes[0] != 2 {
		t.Errorf("buy types = %v, want [2 3 4]", shop.BuyTypes)
	}
}

func TestSetShopBuyTypes_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopBuyTypes(9999, []int{1}) {
		t.Error("expected false for missing shop keeper")
	}
}

func TestSetShopSellTypes_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopSellTypes(2002, []int{3002}) {
		t.Fatal("SetShopSellTypes returned false")
	}
}

func TestSetShopSellTypes_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopSellTypes(9999, []int{1}) {
		t.Error("expected false")
	}
}

func TestSetShopProfit_Valid(t *testing.T) {
	w := newWorldWithShops(t)
	if !w.SetShopProfit(2002, 1.5, 0.6) {
		t.Fatal("SetShopProfit returned false")
	}
	shop, ok := w.GetShopByKeeper(2002)
	if !ok {
		t.Fatal("shop not found")
	}
	if shop.ProfitBuy != 1.5 || shop.ProfitSell != 0.6 {
		t.Errorf("profit = %.1f/%.1f, want 1.5/0.6", shop.ProfitBuy, shop.ProfitSell)
	}
}

func TestSetShopProfit_Missing(t *testing.T) {
	w := newWorldWithShops(t)
	if w.SetShopProfit(9999, 1.0, 1.0) {
		t.Error("expected false")
	}
}
