package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Switched-body coverage for the Exec*-bridged act.other commands (DP-1516
// remaining half, found in the earlier stack). In C, command_interpreter runs
// a resolved command as d->character, which for a switched descriptor is the
// mobile. The display NPC refusal goes through that body's descriptor
// (src/act.other.c:1024-1034); the old bridge sent it to the detached owner,
// so the manager dropped it.

// TestCmdTransformSwitchedMobUntransformable pins do_transform's untransformable
// refusal for a switched mobile (src/act.other.c:1417): a mobile carries no
// PLR flags, so C answers the refusal where the old bridge ran the god's arms.
func TestCmdTransformSwitchedMobUntransformable(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-transform-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdTransform(characterCommandContextForSession(s), []string{}); err != nil {
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
	m.world.MessageSink = func(playerName string, msg []byte) {
		if s.isSwitched && playerName == s.player.Name {
			return
		}
		got = append(got, string(msg))
	}
	m.world.MobileDescriptorOwner = func(body *game.MobInstance) *game.Player {
		if s.isSwitched && body == s.switchedMob {
			return s.player
		}
		return nil
	}
	m.world.MobileMessageSink = func(body *game.MobInstance, msg []byte) {
		if s.isSwitched && body == s.switchedMob {
			got = append(got, string(msg))
		}
	}
	return s, &got
}

func TestCmdDisplaySwitchedMobRefusesAsNpc(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-display-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdDisplay(characterCommandContextForSession(s), []string{"on"}); err != nil {
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
// arm (src/act.other.c:1733), which speaks for NPCs too; the port's old
// owner-targeted send disappeared because the switched owner has no descriptor.
// A level-not-own-body god (>5) and the mobile converge on the same bytes.
func TestCmdRecallSwitchedMobSpeaksExperience(t *testing.T) {
	s, got := switchedDisplaySession(t, "switch-recall-god")
	s.isSwitched = true
	s.switchedMob = registerMob(t, s.manager, 2001, 1001)

	if err := cmdRecall(characterCommandContextForSession(s), []string{}); err != nil {
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

	if err := cmdPrompt(characterCommandContextForSession(s), []string{"all"}); err != nil {
		t.Fatalf("cmdPrompt returned error: %v", err)
	}
	if len(*got) != 1 || (*got)[0] != "Monsters don't need displays.  Go away.\r\n" {
		t.Fatalf("switched-mob prompt output = %q, want C NPC refusal", *got)
	}
}
