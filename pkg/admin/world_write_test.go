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
				Exits: map[string]parser.Exit{
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
				WearFlags:  [4]int{1 << 13, 0, 0, 0},
				Values:     [4]int{0, 3, 5, 0},
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

// ---------------------------------------------------------------------------
// Mob write methods
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Object write methods
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Zone write methods
// ---------------------------------------------------------------------------
