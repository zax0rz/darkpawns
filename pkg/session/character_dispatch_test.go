package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCharacterCommandContextKeepsActiveBodyAndRealSession(t *testing.T) {
	manager := &Manager{}
	owner := game.NewPlayer(1, "Owner", 1001)
	mob := game.NewMob(&parser.Mob{VNum: 2001, Level: 20}, 7)
	switched := &Session{manager: manager, player: owner, isSwitched: true, switchedMob: mob}

	ctx := characterCommandContextForSession(switched)
	if ctx.actor != mob {
		t.Fatalf("acting body = %T %v, want switched mob %p", ctx.actor, ctx.actor, mob)
	}
	if ctx.session != switched {
		t.Fatalf("descriptor session = %p, want original live session %p", ctx.session, switched)
	}
}

func TestCharacterCommandContextKeepsSwitchedPlayerBody(t *testing.T) {
	manager := &Manager{}
	owner := game.NewPlayer(1, "Owner", 1001)
	borrowed := game.NewPlayer(2, "Borrowed", 1001)
	switched := &Session{manager: manager, player: borrowed, isSwitched: true, switchedOriginal: owner, switchedPlayer: borrowed}

	ctx := characterCommandContextForSession(switched)
	if ctx.actor != borrowed || ctx.session != switched {
		t.Fatalf("context body/session = %p/%p, want %p/%p", ctx.actor, ctx.session, borrowed, switched)
	}
}

func TestResolveCharacterCommandUsesExistingCRegistry(t *testing.T) {
	got := resolveCharacterCommand("for", LVL_IMPL)
	want, ok := cmdRegistry.Lookup("force")
	if !ok {
		t.Fatal("force is missing from the existing registry")
	}
	if !got.Found || got.Name != "force" || got.Entry != want {
		t.Fatalf("resolution = (%q, %p, %v), want existing force entry %p", got.Name, got.Entry, got.Found, want)
	}
}

func TestResolveCharacterCommandFiltersBeforeChoosingPrefix(t *testing.T) {
	mortal := resolveCharacterCommand("go", 1)
	wantGossip, ok := cmdRegistry.Lookup("gossip")
	if !ok {
		t.Fatal("gossip is missing from the existing registry")
	}
	if !mortal.Found || mortal.Name != "gossip" || mortal.Entry != wantGossip {
		t.Fatalf("mortal go resolution = (%q, %p, %v), want gossip entry %p", mortal.Name, mortal.Entry, mortal.Found, wantGossip)
	}

	immortal := resolveCharacterCommand("go", LVL_IMMORT)
	wantGoto, ok := cmdRegistry.Lookup("goto")
	if !ok {
		t.Fatal("goto is missing from the existing registry")
	}
	if !immortal.Found || immortal.Name != "goto" || immortal.Entry != wantGoto {
		t.Fatalf("immortal go resolution = (%q, %p, %v), want goto entry %p", immortal.Name, immortal.Entry, immortal.Found, wantGoto)
	}
}

func TestPrepareCharacterCommandDrawsBeforeEmptyAndUnknownResults(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		drawResult int
		wantEmpty  bool
		wantFound  bool
	}{
		{name: "empty line zero draw", line: " \t", drawResult: 0, wantEmpty: true},
		{name: "empty line nonzero draw", line: "\t ", drawResult: 3, wantEmpty: true},
		{name: "unknown command zero draw", line: "zzunknown  arg ", drawResult: 0},
		{name: "unknown command nonzero draw", line: "zzunknown  arg ", drawResult: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor := game.NewPlayer(1, "Owner", 1001)
			actor.SetAffect(game.AffHide, true)
			calls := 0
			prepared := prepareCharacterCommand(characterCommandContext{actor: actor}, test.line, func(min, max int) int {
				calls++
				if min != 0 || max != 3 {
					t.Fatalf("draw bounds = %d..%d, want 0..3", min, max)
				}
				return test.drawResult
			})
			if calls != 1 {
				t.Fatalf("leading hide draw count = %d, want 1", calls)
			}
			if prepared.Empty != test.wantEmpty || prepared.Command.Found != test.wantFound {
				t.Fatalf("preparation = %+v, want Empty=%v Found=%v", prepared, test.wantEmpty, test.wantFound)
			}
			if wantHidden := test.drawResult != 0; actor.IsAffected(game.AffHide) != wantHidden {
				t.Errorf("hide after draw = %v, want %v", actor.IsAffected(game.AffHide), wantHidden)
			}
		})
	}
}

func TestPrepareCharacterCommandPreservesCArgumentPointerRemainder(t *testing.T) {
	actor := game.NewPlayer(1, "Owner", 1001)
	tests := []struct {
		name, line, wantCommand, wantRaw string
	}{
		{name: "word command retains delimiter and whitespace", line: "say   x  ", wantCommand: "say", wantRaw: "   x  "},
		{name: "punctuation retains everything after its byte", line: "'  hi", wantCommand: "'", wantRaw: "  hi"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepared := prepareCharacterCommand(characterCommandContext{actor: actor}, test.line, func(int, int) int { return 1 })
			if prepared.Empty || prepared.Command.Name != test.wantCommand {
				t.Fatalf("command preparation = %+v, want command %q", prepared, test.wantCommand)
			}
			if prepared.RawArgs != test.wantRaw {
				t.Errorf("raw remainder = %q, want exact C line pointer remainder %q", prepared.RawArgs, test.wantRaw)
			}
		})
	}
}

func TestPrepareCharacterCommandClearsHideOnZeroForPlayerAndMob(t *testing.T) {
	type affectBody interface {
		combat.Combatant
		IsAffected(int) bool
	}
	actors := []affectBody{
		game.NewPlayer(1, "Owner", 1001),
		game.NewMob(&parser.Mob{VNum: 2001}, 7),
	}
	for _, actor := range actors {
		if mob, ok := actor.(*game.MobInstance); ok {
			mob.SetAffected(game.AffHide)
		} else {
			actor.(*game.Player).SetAffect(game.AffHide, true)
		}
		prepared := prepareCharacterCommand(characterCommandContext{actor: actor}, "look", func(int, int) int { return 0 })
		if prepared.Empty {
			t.Fatalf("non-empty command unexpectedly prepared empty for %T", actor)
		}
		if actor.IsAffected(game.AffHide) {
			t.Errorf("hide remained set on %T after zero draw", actor)
		}
	}
}
