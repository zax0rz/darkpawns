package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhisperActMessagesMatchCForAllAudiences(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Actor", 1001, true)
	victim := makeTestSession(t, m, "Victim", 1001, true)
	observer := makeTestSession(t, m, "Observer", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, victim)
	registerInWorld(t, observer)

	if err := cmdWhisper(actor, []string{"Victim", "quiet", "words"}); err != nil {
		t.Fatalf("cmdWhisper: %v", err)
	}

	assertWhisperText(t, actor, "You whisper to Victim, 'quiet words'\r\n")
	assertWhisperText(t, victim, "Actor whispers to you, 'quiet words'\r\n")
	assertWhisperText(t, observer, "Actor whispers something to Victim.\r\n")
}

func TestWhisperRejectsSelfWithCMessage(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Actor", 1001, true)
	registerInWorld(t, actor)

	if err := cmdWhisper(actor, []string{"self", "hello"}); err != nil {
		t.Fatalf("cmdWhisper: %v", err)
	}

	assertWhisperText(t, actor, "You can't get your mouth close enough to your ear...\r\n")
}

func TestWhisperCGuardMessages(t *testing.T) {
	t.Run("usage", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeTestSession(t, m, "Actor", 1001, true)
		registerInWorld(t, actor)

		if err := cmdWhisper(actor, []string{"Victim"}); err != nil {
			t.Fatalf("cmdWhisper: %v", err)
		}
		assertWhisperText(t, actor, "Whom do you want to whisper to.. and what??\r\n")
	})

	t.Run("muted before usage", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeTestSession(t, m, "Actor", 1001, true)
		registerInWorld(t, actor)
		actor.player.Flags |= 1 << uint(game.PlrNoshout)

		if err := cmdWhisper(actor, nil); err != nil {
			t.Fatalf("cmdWhisper: %v", err)
		}
		assertWhisperText(t, actor, "Sorry, you cannot do that.\r\n")
	})
}

func TestWhisperNoRepeatUsesCConfirmation(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Actor", 1001, true)
	victim := makeTestSession(t, m, "Victim", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, victim)
	actor.player.Flags |= 1 << uint(game.PrfNoRepeat)

	if err := cmdWhisper(actor, []string{"Victim", "quiet"}); err != nil {
		t.Fatalf("cmdWhisper: %v", err)
	}

	assertWhisperText(t, actor, "Okay.\r\n")
	assertWhisperText(t, victim, "Actor whispers to you, 'quiet'\r\n")
}

func TestWhisperAcceptsVisibleMobTarget(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	actor := makeTestSession(t, m, "Actor", 1001, true)
	observer := makeTestSession(t, m, "Observer", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, observer)
	registerMob(t, m, 2001, 1001)
	// SpawnMob announces the arrival to room players; isolate the command output.
	_ = drainSendChannel(t, actor)
	_ = drainSendChannel(t, observer)

	if err := cmdWhisper(actor, []string{"guard", "halt"}); err != nil {
		t.Fatalf("cmdWhisper: %v", err)
	}

	assertWhisperText(t, actor, "You whisper to A goblin guard, 'halt'\r\n")
	assertWhisperText(t, observer, "Actor whispers something to A goblin guard.\r\n")
}

func assertWhisperText(t *testing.T, s *Session, want string) {
	t.Helper()
	got := readSessionText(t, s)
	if got != want {
		t.Errorf("whisper text = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, "$\x1b") {
		t.Errorf("whisper text leaked an act token or ANSI escape: %q", got)
	}
}
