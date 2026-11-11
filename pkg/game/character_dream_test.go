package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Switched-mobile witnesses for the character-lane ports: do_dream
// (src/act.social.c:294-304) reads the body's position, not the owning
// player's, so a sleeping switched mobile dreams where the old bridge (which
// ran command bodies as the god) daydreamed.

func TestCharacterDreamSwitchedMobSleeping(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8004, Name: "Dream Room", Zone: 8}},
		Mobs: []parser.Mob{
			{VNum: 2001, Keywords: "sleeper", ShortDesc: "a sleeper", Level: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	mob, err := w.SpawnMob(2001, 8004)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosSleeping)

	var got []string
	ctx := CharacterCommandContext{
		Actor:         mob,
		World:         w,
		PrivateOutput: func(text string) { got = append(got, text) },
	}
	if res := CharacterDream(ctx); res != CharacterCommandHandled {
		t.Fatal("CharacterDream did not handle the switched body")
	}
	if len(got) != 1 || got[0] != "You dream of running naked through a field of tulips.\r\n" {
		t.Fatalf("sleeping switched-mob dream output = %q, want the tulip line", got)
	}
}

func TestCharacterDreamSwitchedMobStanding(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8004, Name: "Dream Room", Zone: 8}},
		Mobs: []parser.Mob{
			{VNum: 2001, Keywords: "sleeper", ShortDesc: "a sleeper", Level: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	mob, err := w.SpawnMob(2001, 8004)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosStanding)

	got := ""
	ctx := CharacterCommandContext{
		Actor:         mob,
		World:         w,
		PrivateOutput: func(text string) { got += text },
	}
	if res := CharacterDream(ctx); res != CharacterCommandHandled {
		t.Fatal("CharacterDream did not handle the switched body")
	}
	if got != "You daydream about better times.\r\n" {
		t.Fatalf("standing switched-mob dream output = %q, want the daydream line", got)
	}
	if strings.Contains(got, "tulips") {
		t.Fatalf("standing switched-mob output = %q, want no dream line", got)
	}
}
