package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSocialFallbackUsesCommandGateTable(t *testing.T) {
	t.Run("actor position", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "Actor", 1, 1001)
		s.player.SetPosition(combat.PosResting)
		if err := ExecuteCommand(s, "dance", nil); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, s); got != "Nah... You feel too relaxed to do that.." {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("minimum level", func(t *testing.T) {
		const command = "restrictedtestsocial"
		game.Socials[command] = &game.Social{
			Name:     command,
			Messages: []string{"handler ran", "$n ran a handler", "#"},
		}
		commandGates[command] = commandGate{MinLevel: 2, MinPosition: combat.PosStanding, Source: "test gate"}
		t.Cleanup(func() {
			delete(game.Socials, command)
			delete(commandGates, command)
		})

		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "Actor", 1, 1001)
		s.player.SetPosition(combat.PosStanding)
		if err := ExecuteCommand(s, command, nil); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, s); got != "Huh?!?\r\n" {
			t.Fatalf("output = %q", got)
		}
	})
}

func TestPageUsesOneTargetAndRemainingMessage(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Wizard", LVL_IMMORT, 1001)
	target := makeCommandTestSession(t, m, "Target", 1, 1002)
	m.sessions[actor.player.Name] = actor
	m.sessions[target.player.Name] = target

	if err := ExecuteCommand(actor, "page", []string{"Target", "paging", "you", "now"}); err != nil {
		t.Fatal(err)
	}
	const want = "\x07\x07*Wizard* paging you now"
	if got := readMsgText(t, target); got != want {
		t.Fatalf("target output = %q, want %q", got, want)
	}
	if got := readMsgText(t, actor); got != want {
		t.Fatalf("actor output = %q, want %q", got, want)
	}
}

func TestPageGatesMortalAndPageAll(t *testing.T) {
	t.Run("immortal command gate", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Mortal", 1, 1001)
		if err := ExecuteCommand(actor, "page", []string{"Nobody", "hello"}); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, actor); got != "Huh?!?\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("page all requires above god", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "God", LVL_GOD, 1001)
		if err := ExecuteCommand(actor, "page", []string{"all", "hello", "everyone"}); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, actor); got != "You will never be godly enough to do that!" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("page all fans out above god", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "GreaterGod", LVL_GOD+1, 1001)
		target := makeCommandTestSession(t, m, "Target", 1, 1002)
		m.sessions[actor.player.Name] = actor
		m.sessions[target.player.Name] = target
		if err := ExecuteCommand(actor, "page", []string{"all", "hello", "everyone"}); err != nil {
			t.Fatal(err)
		}
		const want = "\x07\x07*GreaterGod* hello everyone"
		if got := readMsgText(t, actor); got != want {
			t.Fatalf("actor output = %q", got)
		}
		if got := readMsgText(t, target); got != want {
			t.Fatalf("target output = %q", got)
		}
	})
}
