package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newConsiderTestWorld(t *testing.T) (*World, map[string]string) {
	t.Helper()
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Consider Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	messages := make(map[string]string)
	world.MessageSink = func(playerName string, message []byte) {
		messages[playerName] += string(message)
	}
	return world, messages
}

func addConsiderPlayer(t *testing.T, world *World, id int, name string) *Player {
	t.Helper()
	player := NewPlayer(id, name, 1001)
	player.Stats.Str = 10
	if err := world.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	return player
}

func TestDoConsiderNotFoundAndSelf(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		world, messages := newConsiderTestWorld(t)
		actor := addConsiderPlayer(t, world, 1, "Actor")

		world.DoConsider(actor, "Nobody")
		if got := messages[actor.Name]; got != considerNotFound {
			t.Fatalf("not-found message = %q, want %q", got, considerNotFound)
		}
	})

	for _, target := range []string{"self", "Actor", "Act"} {
		t.Run("self "+target, func(t *testing.T) {
			world, messages := newConsiderTestWorld(t)
			actor := addConsiderPlayer(t, world, 1, "Actor")

			world.DoConsider(actor, target)
			if got := messages[actor.Name]; got != considerSelf {
				t.Fatalf("self message = %q, want %q", got, considerSelf)
			}
		})
	}
}

func TestDoConsiderIsPrivate(t *testing.T) {
	world, messages := newConsiderTestWorld(t)
	actor := addConsiderPlayer(t, world, 1, "Actor")
	target := addConsiderPlayer(t, world, 2, "Target")
	observer := addConsiderPlayer(t, world, 3, "Observer")

	world.DoConsider(actor, target.Name)

	want := "Target looks like a fair fight, looks to be\r\n" +
		"in about the same physical shape as you, and seems about as confident in\r\n" +
		"battle as you do.\r\n"
	if got := messages[actor.Name]; got != want {
		t.Fatalf("actor message = %q, want %q", got, want)
	}
	if got := messages[target.Name]; got != "" {
		t.Errorf("target received private evaluation: %q", got)
	}
	if got := messages[observer.Name]; got != "" {
		t.Errorf("observer received private evaluation: %q", got)
	}
}

func TestConsiderThresholdLadders(t *testing.T) {
	t.Run("damage", func(t *testing.T) {
		tests := []struct {
			diff int
			want string
		}{
			{21, "$N looks like $E could eat you for lunch, "},
			{20, "$N looks like $E could tear you up in a fight, "},
			{11, "$N looks like $E could tear you up in a fight, "},
			{10, "$N looks like $E could hurt you in a fight, "},
			{6, "$N looks like $E could hurt you in a fight, "},
			{5, "$N looks like a fair fight, "},
			{-2, "$N looks like a fair fight, "},
			{-3, "$N looks like an easy kill, "},
			{-4, "$N looks like an easy kill, "},
			{-5, "$N looks like a very easy kill, "},
			{-9, "$N looks like a very easy kill, "},
			{-10, "$N might not even be worth the effort to kill, "},
		}
		for _, test := range tests {
			if got := considerDamageAssessment(test.diff); got != test.want {
				t.Errorf("damage diff %d = %q, want %q", test.diff, got, test.want)
			}
		}
	})

	t.Run("health", func(t *testing.T) {
		const actorHP = 10
		tests := []struct {
			diff int
			want string
		}{
			{41, "looks to be\r\nin much better physical shape than you, "},
			{40, "looks to be\r\nin a lot better physical shape than you, "},
			{21, "looks to be\r\nin a lot better physical shape than you, "},
			{20, "looks to be\r\nin better physical shape than you, "},
			{11, "looks to be\r\nin better physical shape than you, "},
			{10, "looks to be\r\nin about the same physical shape as you, "},
			{0, "looks to be\r\nin about the same physical shape as you, "},
			{-1, "looks to be\r\nin a little worse physical shape as you, "},
			{-24, "looks to be\r\nin a little worse physical shape as you, "},
			{-25, "looks to be\r\nin a worse physical shape than you, "},
			{-49, "looks to be\r\nin a worse physical shape than you, "},
			{-50, "looks to be\r\nin a lot worse physical shape than you, "},
		}
		for _, test := range tests {
			if got := considerHealthAssessment(test.diff, actorHP); got != test.want {
				t.Errorf("health diff %d = %q, want %q", test.diff, got, test.want)
			}
		}
	})

	t.Run("level", func(t *testing.T) {
		const actorLevel = 10
		tests := []struct {
			diff int
			want string
		}{
			{11, "and moves with an ease telling\r\nof many won battles."},
			{10, "and seems to know $S opponent."},
			{6, "and seems to know $S opponent."},
			{5, "and seems about as confident in\r\nbattle as you do."},
			{0, "and seems about as confident in\r\nbattle as you do."},
			{-1, "and seems less confident in\r\nbattle than you do."},
			{-2, "and seems less confident in\r\nbattle than you do."},
			{-3, "and seems much less confident in\r\nbattle than you do."},
			{-4, "and seems much less confident in\r\nbattle than you do."},
			{-5, "and seems ready to run from a\r\nfight."},
			{-6, "and seems ready to run from a\r\nfight."},
			{-7, "and seems like $E's never been\r\nin battle before."},
		}
		for _, test := range tests {
			if got := considerLevelAssessment(test.diff, actorLevel); got != test.want {
				t.Errorf("level diff %d = %q, want %q", test.diff, got, test.want)
			}
		}
	})
}

func TestDoConsiderUsesCanonicalVictimPronouns(t *testing.T) {
	tests := []struct {
		sex        int
		subject    string
		possessive string
	}{
		{sex: 0, subject: "he", possessive: "his"},
		{sex: 1, subject: "she", possessive: "her"},
		{sex: 2, subject: "it", possessive: "its"},
	}

	for _, test := range tests {
		t.Run(test.subject, func(t *testing.T) {
			world, messages := newConsiderTestWorld(t)
			actor := addConsiderPlayer(t, world, 1, "Actor")
			actor.SetLevel(10)
			target := addConsiderPlayer(t, world, 2, "Target")
			target.SetLevel(16)
			target.Stats.Str = 19 // str_app todam 7: selects the $E damage band
			target.Sex = test.sex

			combat.WithRoller(combat.NewScriptedRoller([]int{0, 0}), func() {
				world.DoConsider(actor, target.Name)
			})

			want := "Target looks like " + test.subject + " could hurt you in a fight, " +
				"looks to be\r\nin about the same physical shape as you, " +
				"and seems to know " + test.possessive + " opponent.\r\n"
			if got := messages[actor.Name]; got != want {
				t.Fatalf("sex %d message = %q, want %q", test.sex, got, want)
			}
		})
	}
}
