package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Switched-body coverage for the Exec*-bridged act.other commands (DP-1516
// remaining half, found in PR #2474). In C, command_interpreter runs every
// resolved command as d->character, which for a switched descriptor is the
// mobile; `display` answers with its IS_NPC refusal (src/act.other.c:1028)
// and never touches the god's flags or prints Okay. The port ran it as the
// owning player, printing Okay and setting the god's PRF_DISP* bits.

// TestCmdTransformSwitchedMobUntransformable pins do_transform's untransformable
// refusal for a switched mobile (src/act.other.c:1417): a mobile carries no
// PLR flags, so C answers the refusal where the old bridge ran the god's arms.
func TestCmdTransformSwitchedMobUntransformable(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-transform-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdTransform(s, []string{}); err != nil {
		t.Fatalf("cmdTransform returned error: %v", err)
	}
	if len(*got) != 1 || (*got)[0] != "You aren't transformable!\n\r" {
		t.Fatalf("switched-mob transform output = %q, want C refusal", *got)
	}
}

func switchedDisplaySession(t *testing.T, name string) (*Session, *[]string) {
	t.Helper()
	m := makeTestManagerWithMobs(t)
	s := makeCommandTestSession(t, m, name, LVL_IMPL, 1001)
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var got []string
	m.world.MessageSink = func(playerName string, msg []byte) { got = append(got, string(msg)) }
	return s, &got
}

func TestCmdDisplaySwitchedMobRefusesAsNpc(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-display-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdDisplay(s, []string{"on"}); err != nil {
		t.Fatalf("cmdDisplay returned error: %v", err)
	}
	if len(*got) != 1 || (*got)[0] != "Monsters don't need displays.  Go away.\r\n" {
		t.Fatalf("switched-mob display output = %q, want C NPC refusal", *got)
	}
	if s.player.GetFlags()&(1<<game.PrfDisphp) != 0 {
		t.Error("switched display set the god's PRF_DISPHP bit")
	}
}

// TestCmdRecallSwitchedMobSpeaksExperience pins do_recall's shared level/NPC
// arm (src/act.other.c:1733), which speaks for NPCs too; the port's was a
// silent return, and switched recall through the old bridge ran as the god.
// A level-not-own-body god (>5) and the mobile converge on the same bytes.
func TestCmdRecallSwitchedMobSpeaksExperience(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-recall-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdRecall(s, []string{}); err != nil {
		t.Fatalf("cmdRecall returned error: %v", err)
	}
	if len(*got) != 1 || (*got)[0] != "This command is not available for someone of your experience!\r\n" {
		t.Fatalf("switched-mob recall output = %q, want C experience line", *got)
	}
}

// TestCmdPromptSwitchedMobRefusesAsNpc pins the prompt alias (same do_display
// handler, src/interpreter.c:619) through the same switched route.
func TestCmdPromptSwitchedMobRefusesAsNpc(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-prompt-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdPrompt(s, []string{"all"}); err != nil {
		t.Fatalf("cmdPrompt returned error: %v", err)
	}
	if len(*got) != 1 || (*got)[0] != "Monsters don't need displays.  Go away.\r\n" {
		t.Fatalf("switched-mob prompt output = %q, want C NPC refusal", *got)
	}
}
