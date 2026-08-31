package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGsayRegistrationUsesCEntryGates(t *testing.T) {
	for _, name := range []string{"gsay", "gtell"} {
		t.Run(name, func(t *testing.T) {
			entry, ok := commandGates[name]
			if !ok {
				t.Fatalf("%s command has no C gate", name)
			}
			if entry.MinLevel != 0 || entry.MinPosition != combat.PosSleeping {
				t.Fatalf("%s gate = level %d position %d, want level 0 position %d", name, entry.MinLevel, entry.MinPosition, combat.PosSleeping)
			}
		})
	}

	entry, ok := cmdRegistry.Lookup("gsay")
	if !ok {
		t.Fatal("gsay command is not registered")
	}
	alias, ok := cmdRegistry.Lookup("gtell")
	if !ok {
		t.Fatal("gtell command is not registered")
	}
	if entry != alias {
		t.Fatal("gsay and gtell do not share the C do_gsay handler")
	}
}

func TestGsayRawMessageAndNoRepeatMatchC(t *testing.T) {
	m := makeTestManager(t)
	leader := makeCommandTestSession(t, m, "Leader", 1, 1001)
	member := makeCommandTestSession(t, m, "Member", 1, 1001)
	leader.player.SetInGroup(true)
	leader.player.SetPlrFlag(game.PrfNoRepeat, true)
	member.player.SetInGroup(true)
	member.player.SetFollowing(leader.player.Name)
	if err := m.world.AddPlayer(leader.player); err != nil {
		t.Fatal(err)
	}
	if err := m.world.AddPlayer(member.player); err != nil {
		t.Fatal(err)
	}
	m.sessions[leader.player.Name] = leader
	m.sessions[member.player.Name] = member

	if err := cmdGtellText(leader, "multiple    &Rred&n"); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, leader); got != "Okay." {
		t.Fatalf("leader output = %q, want C no-repeat confirmation", got)
	}
	if got := readMsgText(t, member); got != "Leader tells the group, 'multiple    Rredn'\r\n" {
		t.Fatalf("member output = %q, want raw-space and ANSI-deleted group message", got)
	}
}
