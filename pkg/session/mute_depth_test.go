package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMuteRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("mute")
	if !ok {
		t.Fatal("mute command is not registered")
	}
	if entry.MinLevel != 1 || entry.MinPosition != combat.PosDead {
		t.Fatalf("mute gate = (%d, %d), want (1, %d)", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}

func TestCmdMuteMatchesDoWizutilTargetBranches(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Mutegod", LVL_IMPL, 1001)
	victim := makeCommandTestSession(t, m, "Mutevictim", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, victim)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no argument", want: "Yes, but for whom?!?\r\n"},
		{name: "missing target", args: []string{"Nobody"}, want: "There is no such player.\r\n"},
		{name: "toggle on", args: []string{"Mutevictim", "ignored"}, want: "(GC) Squelch ON for Mutevictim by Mutegod.\r\n"},
		{name: "toggle off", args: []string{"Mutevictim"}, want: "(GC) Squelch OFF for Mutevictim by Mutegod.\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cmdMute(actor, tt.args); err != nil {
				t.Fatal(err)
			}
			if got := readMsgText(t, actor); got != tt.want {
				t.Fatalf("response = %q, want %q", got, tt.want)
			}
		})
	}

	if victim.player.GetFlags()&(1<<uint(game.PlrNoshout)) != 0 {
		t.Fatal("paired toggles left PLR_NOSHOUT enabled")
	}
	select {
	case msg := <-victim.send:
		t.Fatalf("mute sent unexpected victim output: %s", msg)
	default:
	}
}

func TestCmdMuteRejectsVisibleMob(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	actor := makeCommandTestSession(t, m, "Mutegod", LVL_IMPL, 1001)
	registerInWorld(t, actor)
	registerMob(t, m, 2001, 1001)
	_ = readMsgText(t, actor) // mob-entry announcement

	if err := cmdMute(actor, []string{"guard"}); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, actor), "You can't do that to a mob!\r\n"; got != want {
		t.Fatalf("mob response = %q, want %q", got, want)
	}
}

func TestCmdMuteRejectsHigherImmortal(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Mutegod", LVL_IMMORT, 1001)
	actor.player.SetPlrFlag(game.PlrChosen, true)
	target := makeCommandTestSession(t, m, "Highergod", LVL_IMPL, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdMute(actor, []string{"Highergod"}); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, actor), "Hmmm...you'd better not.\r\n"; got != want {
		t.Fatalf("higher-level response = %q, want %q", got, want)
	}
	if target.player.GetFlags()&(1<<uint(game.PlrNoshout)) != 0 {
		t.Fatal("higher-level target was toggled")
	}
}

func TestCmdMuteMortalFailsInnerDoWizutilAuth(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Mutemortal", 1, 1001)
	registerInWorld(t, actor)

	if err := cmdMute(actor, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, actor), "Huh?!?"; got != want {
		t.Fatalf("mortal no-argument response = %q, want %q", got, want)
	}

	if err := ExecuteCommand(actor, "mut", []string{"Nobody"}); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, actor), "Huh?!?"; got != want {
		t.Fatalf("mortal prefix response = %q, want %q", got, want)
	}
}
