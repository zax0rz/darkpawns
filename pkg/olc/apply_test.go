package olc

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestApplyStringCaps(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		apply func(string) string
	}{
		{
			name:  "room name preserves C's MAX_ROOM_NAME minus NUL",
			limit: MaxRoomName - 1,
			apply: func(value string) string {
				room := parser.Room{}
				if err := Apply(Operation{Kind: OpSetRoomName, Room: &room, Text: value}); err != nil {
					t.Fatal(err)
				}
				return room.Name
			},
		},
		{
			name:  "room description",
			limit: MaxRoomDesc,
			apply: func(value string) string {
				room := parser.Room{}
				if err := Apply(Operation{Kind: OpSetRoomDescription, Room: &room, Text: value}); err != nil {
					t.Fatal(err)
				}
				return room.Description
			},
		},
		{
			name:  "exit description",
			limit: MaxExitDesc,
			apply: func(value string) string {
				exit := parser.Exit{}
				if err := Apply(Operation{Kind: OpSetExitDescription, Exit: &exit, Text: value}); err != nil {
					t.Fatal(err)
				}
				return exit.Description
			},
		},
		{
			name:  "extra description",
			limit: MaxExtraDesc,
			apply: func(value string) string {
				extra := parser.ExtraDesc{}
				if err := Apply(Operation{Kind: OpSetExtraDescription, Extra: &extra, Text: value}); err != nil {
					t.Fatal(err)
				}
				return extra.Description
			},
		},
		{
			name:  "mob name",
			limit: MaxMobName,
			apply: func(value string) string {
				mob := parser.Mob{}
				if err := Apply(Operation{Kind: OpSetMobKeywords, Mob: &mob, Text: value}); err != nil {
					t.Fatal(err)
				}
				return mob.Keywords
			},
		},
		{
			name:  "object name",
			limit: MaxObjName,
			apply: func(value string) string {
				obj := parser.Obj{}
				if err := Apply(Operation{Kind: OpSetObjKeywords, Obj: &obj, Text: value}); err != nil {
					t.Fatal(err)
				}
				return obj.Keywords
			},
		},
		{
			name:  "object action description",
			limit: MaxMessage,
			apply: func(value string) string {
				obj := parser.Obj{}
				if err := Apply(Operation{Kind: OpSetObjActionDescription, Obj: &obj, Text: value}); err != nil {
					t.Fatal(err)
				}
				return obj.ActionDesc
			},
		},
		{
			name:  "object extra description",
			limit: MaxExtraDesc,
			apply: func(value string) string {
				obj := parser.Obj{ExtraDescs: []parser.ExtraDesc{{}}}
				if err := Apply(Operation{Kind: OpSetObjExtraDescription, Extra: &obj.ExtraDescs[0], Text: value}); err != nil {
					t.Fatal(err)
				}
				return obj.ExtraDescs[0].Description
			},
		},
		{
			name:  "mob description",
			limit: MaxMobDesc,
			apply: func(value string) string {
				mob := parser.Mob{}
				if err := Apply(Operation{Kind: OpSetMobDetailedDescription, Mob: &mob, Text: value}); err != nil {
					t.Fatal(err)
				}
				return mob.DetailedDesc
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			within := strings.Repeat("x", test.limit)
			if got := test.apply(within); len(got) != test.limit {
				t.Fatalf("boundary length = %d, want %d", len(got), test.limit)
			}
			over := within + "x"
			if got := test.apply(over); len(got) != test.limit {
				t.Fatalf("over-boundary length = %d, want %d", len(got), test.limit)
			}
		})
	}

	var clamped int
	for _, input := range []int{-1, 0, 10, 11} {
		if err := Apply(Operation{Kind: OpClampInt, Value: input, Low: 0, High: 10, Result: &clamped}); err != nil {
			t.Fatal(err)
		}
		want := input
		if want < 0 {
			want = 0
		}
		if want > 10 {
			want = 10
		}
		if clamped != want {
			t.Fatalf("generic clamp input %d: got %d, want %d", input, clamped, want)
		}
	}
}

func TestApplyNumericClampBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		kind  OperationKind
		low   int
		high  int
		read  func(parser.Mob) int
		value func(int) int
	}{
		{"sex", OpSetMobSex, 0, 2, func(m parser.Mob) int { return m.Sex }, func(v int) int { return v }},
		{"hitroll", OpSetMobHitroll, 0, 127, func(m parser.Mob) int { return m.THAC0 }, func(v int) int { return 20 - v }},
		{"damroll", OpSetMobDamroll, 0, 127, func(m parser.Mob) int { return m.Damage.Plus }, func(v int) int { return v }},
		{"damage num", OpSetMobNumDamageDice, 0, 127, func(m parser.Mob) int { return m.Damage.Num }, func(v int) int { return v }},
		{"damage sides", OpSetMobSizeDamageDice, 0, 127, func(m parser.Mob) int { return m.Damage.Sides }, func(v int) int { return v }},
		{"hp num", OpSetMobNumHPDice, 0, 50, func(m parser.Mob) int { return m.HP.Num }, func(v int) int { return v }},
		{"hp sides", OpSetMobSizeHPDice, 0, 3000, func(m parser.Mob) int { return m.HP.Sides }, func(v int) int { return v }},
		{"hp plus", OpSetMobAddHP, 0, 30000, func(m parser.Mob) int { return m.HP.Plus }, func(v int) int { return v }},
		{"ac", OpSetMobAC, -200, 200, func(m parser.Mob) int { return m.AC }, func(v int) int { return v }},
		{"position", OpSetMobPosition, 0, 14, func(m parser.Mob) int { return m.Position }, func(v int) int { return v }},
		{"default position", OpSetMobDefaultPosition, 0, 14, func(m parser.Mob) int { return m.DefaultPos }, func(v int) int { return v }},
		{"attack", OpSetMobAttack, 0, 14, func(m parser.Mob) int { return m.BareHandAttack }, func(v int) int { return v }},
		{"alignment", OpSetMobAlignment, -1000, 1000, func(m parser.Mob) int { return m.Alignment }, func(v int) int { return v }},
		{"race", OpSetMobRace, 0, 30, func(m parser.Mob) int { return m.Race }, func(v int) int { return v }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, input := range []int{test.low - 1, test.low, test.high, test.high + 1} {
				mob := parser.Mob{}
				if err := Apply(Operation{Kind: test.kind, Mob: &mob, Value: input}); err != nil {
					t.Fatal(err)
				}
				effective := input
				if effective < test.low {
					effective = test.low
				}
				if effective > test.high {
					effective = test.high
				}
				if got, want := test.read(mob), test.value(effective); got != want {
					t.Fatalf("input %d: got %d, want %d", input, got, want)
				}
			}
		})
	}

	for _, test := range []struct {
		name string
		kind OperationKind
		read func(parser.ShopProto) int
	}{
		{"open1", OpSetShopOpenHour1, func(s parser.ShopProto) int { return s.OpenHour1 }},
		{"open2", OpSetShopOpenHour2, func(s parser.ShopProto) int { return s.OpenHour2 }},
		{"close1", OpSetShopCloseHour1, func(s parser.ShopProto) int { return s.CloseHour1 }},
		{"close2", OpSetShopCloseHour2, func(s parser.ShopProto) int { return s.CloseHour2 }},
	} {
		t.Run("shop "+test.name, func(t *testing.T) {
			for _, input := range []int{-1, 0, 28, 29} {
				shop := parser.ShopProto{}
				if err := Apply(Operation{Kind: test.kind, Shop: &shop, Value: input}); err != nil {
					t.Fatal(err)
				}
				want := input
				if want < 0 {
					want = 0
				}
				if want > 28 {
					want = 28
				}
				if got := test.read(shop); got != want {
					t.Fatalf("input %d: got %d, want %d", input, got, want)
				}
			}
		})
	}

	for _, test := range []struct {
		name string
		kind OperationKind
		read func(parser.Obj) int
		low  int
		high int
	}{
		{"value3", OpSetObjValue3, func(o parser.Obj) int { return o.Values[2] }, -10, 20},
		{"value4", OpSetObjValue4, func(o parser.Obj) int { return o.Values[3] }, 1, 103},
	} {
		t.Run("object "+test.name, func(t *testing.T) {
			for _, input := range []int{test.low - 1, test.low, test.high, test.high + 1} {
				obj := parser.Obj{}
				if err := Apply(Operation{Kind: test.kind, Obj: &obj, Value: input, Low: test.low, High: test.high}); err != nil {
					t.Fatal(err)
				}
				want := input
				if want < test.low {
					want = test.low
				}
				if want > test.high {
					want = test.high
				}
				if got := test.read(obj); got != want {
					t.Fatalf("input %d: got %d, want %d", input, got, want)
				}
			}
		})
	}

	for _, input := range []int{2999, 3000, 3099, 3100} {
		zone := parser.Zone{Number: 30, TopRoom: 3050}
		if err := Apply(Operation{Kind: OpSetZoneTopRoom, Zone: &zone, Value: input, Low: 3000, High: 3099}); err != nil {
			t.Fatal(err)
		}
		want := input
		if want < 3000 {
			want = 3000
		}
		if want > 3099 {
			want = 3099
		}
		if zone.TopRoom != want {
			t.Fatalf("zone top input %d: got %d, want %d", input, zone.TopRoom, want)
		}
	}
}

func TestApplyMobLevelCascadeWritesCFields(t *testing.T) {
	mob := parser.Mob{
		Level:  99,
		Exp:    777,
		Gold:   888,
		HP:     parser.DiceRoll{Num: 1, Sides: 2, Plus: 3},
		Damage: parser.DiceRoll{Num: 4, Sides: 5, Plus: 6},
		AC:     7,
		THAC0:  8,
	}
	if err := Apply(Operation{Kind: OpSetMobLevel, Mob: &mob, Value: 30}); err != nil {
		t.Fatal(err)
	}
	if got, want := mob.Level, 30; got != want {
		t.Errorf("level = %d, want %d", got, want)
	}
	if got, want := mob.Exp, 60000; got != want {
		t.Errorf("exp = %d, want %d", got, want)
	}
	if got, want := mob.Damage, (parser.DiceRoll{Num: 20, Sides: 4, Plus: 20}); got != want {
		t.Errorf("damage = %#v, want %#v", got, want)
	}
	if got, want := mob.HP, (parser.DiceRoll{Num: 30, Sides: 5, Plus: 414}); got != want {
		t.Errorf("hp = %#v, want %#v", got, want)
	}
	if got, want := mob.AC, -200; got != want {
		t.Errorf("ac = %d, want %d", got, want)
	}
	if got, want := mob.THAC0, -10; got != want {
		t.Errorf("thac0 = %d, want %d", got, want)
	}
	if got, want := mob.Gold, 888; got != want {
		t.Errorf("unrelated gold = %d, want %d", got, want)
	}

	zero := parser.Mob{}
	if err := Apply(Operation{Kind: OpSetMobLevel, Mob: &zero, Value: 0}); err != nil {
		t.Fatal(err)
	}
	if got, want := zero.Level, 0; got != want {
		t.Errorf("zero level = %d, want %d", got, want)
	}
	if got, want := zero.Exp, 100; got != want {
		t.Errorf("zero-level cascade exp = %d, want %d", got, want)
	}
}

func TestApplyRoomOperations(t *testing.T) {
	room := parser.Room{
		VNum: 1001,
		Name: "old",
		Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 1002},
		},
		ExtraDescs: []parser.ExtraDesc{{Keywords: "old", Description: "old"}},
	}
	if err := Apply(Operation{Kind: OpSetRoomFlag, Room: &room, Bit: 0, Value: 1}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetRoomFlag, Room: &room, Bit: 27, Value: 1}); err != nil {
		t.Fatal(err)
	}
	if got := room.Flags[0]; got != "134217729" {
		t.Fatalf("room flag word = %q, want bits 0 and 27 set", got)
	}
	if err := Apply(Operation{Kind: OpSetRoomSector, Room: &room, Value: 15}); err != nil {
		t.Fatal(err)
	}
	if room.Sector != 15 {
		t.Fatalf("sector = %d, want 15", room.Sector)
	}
	if err := Apply(Operation{Kind: OpSetRoomSector, Room: &room, Value: 16}); err == nil {
		t.Fatal("invalid sector was accepted")
	}

	if err := Apply(Operation{
		Kind:       OpSetExitTarget,
		Room:       &room,
		Direction:  "east",
		Value:      1003,
		RoomExists: func(vnum int) bool { return vnum == 1002 },
	}); err == nil {
		t.Fatal("nonexistent exit target was accepted")
	}
	if err := Apply(Operation{
		Kind:       OpSetExitTarget,
		Room:       &room,
		Direction:  "east",
		Value:      1002,
		RoomExists: func(vnum int) bool { return vnum == 1002 },
	}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetExitKeywords, Room: &room, Direction: "east", Text: "door"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetExitKey, Room: &room, Direction: "east", Value: 3001}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetExitDoorFlags, Room: &room, Direction: "east", Value: 2}); err != nil {
		t.Fatal(err)
	}
	if got := room.Exits["east"]; got.ToRoom != 1002 || got.Keywords != "door" || got.Key != 3001 || got.DoorState != 2 || got.ExitInfo != parser.ExitIsDoor|parser.ExitPickproof {
		t.Fatalf("east exit = %#v", got)
	}
	if err := Apply(Operation{Kind: OpSetExitDescription, Room: &room, Direction: "east", Text: strings.Repeat("x", MaxExitDesc+1)}); err != nil {
		t.Fatal(err)
	}
	if got := len(room.Exits["east"].Description); got != MaxExitDesc {
		t.Fatalf("exit description length = %d, want %d", got, MaxExitDesc)
	}

	if err := Apply(Operation{
		Kind:  OpAddExtraDescription,
		Room:  &room,
		Index: -1,
		Extra: &parser.ExtraDesc{Keywords: "new", Description: "new"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetExtraKeywords, Room: &room, Index: 1, Text: "changed"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpSetExtraDescription, Room: &room, Index: 1, Text: "changed description"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(Operation{Kind: OpRemoveExtraDescription, Room: &room, Index: 0}); err != nil {
		t.Fatal(err)
	}
	if len(room.ExtraDescs) != 1 || room.ExtraDescs[0].Keywords != "changed" {
		t.Fatalf("extra descriptions = %#v", room.ExtraDescs)
	}

	source := parser.Room{Name: "copied", Description: "copied description"}
	if err := Apply(Operation{Kind: OpCopyRoom, Room: &room, Source: &source}); err != nil {
		t.Fatal(err)
	}
	if room.Name != source.Name || room.Description != source.Description {
		t.Fatalf("copy result = %q / %q", room.Name, room.Description)
	}
	if err := Apply(Operation{Kind: OpPurgeExit, Room: &room, Direction: "east"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := room.Exits["east"]; ok {
		t.Fatal("purged exit remains")
	}
}
