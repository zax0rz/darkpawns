package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdPardonMatchesCInnerAuthorization(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Pardongod", 1, 1001)
	target := makeCommandTestSession(t, m, "Pardontarget", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdPardon(actor, nil); err != nil {
		t.Fatalf("cmdPardon unauthorized: %v", err)
	}
	if got, want := readMsgText(t, actor), "Huh?!?"; got != want {
		t.Fatalf("unauthorized response = %q, want %q", got, want)
	}

	actor.player.SetPlrFlag(game.PlrChosen, true)
	target.player.SetPlrFlag(game.PlrOutlaw, true)
	if err := cmdPardon(actor, []string{"Pardontarget"}); err != nil {
		t.Fatalf("cmdPardon chosen: %v", err)
	}
	if got, want := readMsgText(t, actor), "Pardoned.\r\n"; got != want {
		t.Fatalf("chosen actor response = %q, want %q", got, want)
	}
	if got, want := readMsgText(t, target), "You have been pardoned by the Gods!\r\n"; got != want {
		t.Fatalf("chosen target response = %q, want %q", got, want)
	}
	if target.player.GetFlags()&(1<<uint(game.PlrOutlaw)) != 0 {
		t.Fatal("pardon left PLR_OUTLAW set")
	}
}

func TestCmdPardonUsesCOneArgumentAndHigherImmortalGuard(t *testing.T) {
	t.Run("fill word and trailing arguments", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Pardongod", LVL_IMPL, 1001)
		target := makeCommandTestSession(t, m, "Pardontarget", 1, 1001)
		registerInWorld(t, actor)
		registerInWorld(t, target)
		target.player.SetPlrFlag(game.PlrOutlaw, true)

		if err := cmdPardon(actor, []string{"the", "Pardontarget", "trailing", "words"}); err != nil {
			t.Fatalf("cmdPardon fill word: %v", err)
		}
		if got, want := readMsgText(t, actor), "Pardoned.\r\n"; got != want {
			t.Fatalf("fill-word actor response = %q, want %q", got, want)
		}
		if got, want := readMsgText(t, target), "You have been pardoned by the Gods!\r\n"; got != want {
			t.Fatalf("fill-word target response = %q, want %q", got, want)
		}
	})

	t.Run("higher immortal", func(t *testing.T) {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Pardongod", LVL_IMMORT, 1001)
		target := makeCommandTestSession(t, m, "Highergod", LVL_IMPL, 1001)
		registerInWorld(t, actor)
		registerInWorld(t, target)
		target.player.SetPlrFlag(game.PlrOutlaw, true)

		if err := cmdPardon(actor, []string{"Highergod"}); err != nil {
			t.Fatalf("cmdPardon higher immortal: %v", err)
		}
		if got, want := readMsgText(t, actor), "Hmmm...you'd better not.\r\n"; got != want {
			t.Fatalf("higher-immortal response = %q, want %q", got, want)
		}
		if target.player.GetFlags()&(1<<uint(game.PlrOutlaw)) == 0 {
			t.Fatal("higher immortal was pardoned")
		}
	})
}
