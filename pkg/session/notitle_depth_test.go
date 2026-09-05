package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNotitleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["notitle"]
	if !ok {
		t.Fatal("notitle command has no C gate")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("notitle gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosDead)
	}
}

func TestCmdNotitleMatchesCBranchesAndToggle(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Notitlegod", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Notitlevictim", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no argument", want: "Yes, but for whom?!?\r\n"},
		{name: "missing target", args: []string{"Nobody"}, want: "There is no such player.\r\n"},
		{name: "skip leading fill word and ignore trailing", args: []string{"the", "Notitlevictim", "ignored"}, want: "(GC) Notitle ON for Notitlevictim by Notitlegod.\r\n"},
		{name: "toggle off", args: []string{"Notitlevictim"}, want: "(GC) Notitle OFF for Notitlevictim by Notitlegod.\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cmdNotitle(actor, tt.args); err != nil {
				t.Fatal(err)
			}
			if got := readMsgText(t, actor); got != tt.want {
				t.Fatalf("response = %q, want %q", got, tt.want)
			}
		})
	}
	if target.player.GetFlags()&(1<<uint(game.PlrNotitle)) != 0 {
		t.Fatal("paired toggles left PLR_NOTITLE enabled")
	}
}
