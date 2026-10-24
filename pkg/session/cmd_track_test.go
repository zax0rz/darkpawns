package session

// Registration proof for the "track" command (BRIEF-2026-10-24 Fix 1):
// `track` resolves through command_interpreter (the C order table row 447)
// and reaches the game handler. With the registration removed, the same
// input must produce the parser's unrecognized-command line.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTrackCommandDispatch(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Class = game.ClassWarrior
	// World output routes by registered session name; an unregistered
	// fixture player would swallow the handler's message.
	if err := m.Register("Alice", s); err != nil {
		t.Fatalf("register Alice: %v", err)
	}
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	t.Run("reaches the handler, not the parser", func(t *testing.T) {
		// A warrior without the skill: the command itself answers.
		if err := ExecuteCommand(s, "track", []string{"bob"}); err != nil {
			t.Fatalf("ExecuteCommand: %v", err)
		}
		if msg := readSessionText(t, s); msg != "You have no idea how.\r\n" {
			t.Fatalf("track output = %q, want %q", msg, "You have no idea how.\r\n")
		}
	})

	t.Run("full path to the trail message", func(t *testing.T) {
		s.player.SetSkill("track", 75)
		victim := game.NewPlayer(2, "Vic", 1002) // room 1002 has no exits
		victim.SetSex(game.SexFemale)
		if err := m.world.AddPlayer(victim); err != nil {
			t.Fatalf("AddPlayer victim failed: %v", err)
		}

		if err := ExecuteCommand(s, "track", []string{"Vic"}); err != nil {
			t.Fatalf("ExecuteCommand: %v", err)
		}
		if msg := readSessionText(t, s); msg != "You can't sense a trail to her from here.\r\n" {
			t.Fatalf("track output = %q, want %q", msg, "You can't sense a trail to her from here.\r\n")
		}
	})
}
