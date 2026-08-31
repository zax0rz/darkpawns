package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestForceRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("force")
	if !ok {
		t.Fatal("force command is not registered")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("force gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosSleeping)
	}
}

func TestCmdForceUsesCRejections(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Forceactor", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Forcetarget", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no arguments", args: nil, want: "Whom do you wish to force do what?\r\n"},
		{name: "no command", args: []string{"Forcetarget"}, want: "Whom do you wish to force do what?\r\n"},
		{name: "missing target", args: []string{"Nobody", "whoami"}, want: noPersonHere},
		{name: "equal level", args: []string{"Forceactor", "whoami"}, want: "No, no, no!\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cmdForce(actor, tt.args); err != nil {
				t.Fatalf("cmdForce returned error: %v", err)
			}
			if got := readMsgText(t, actor); got != tt.want {
				t.Fatalf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCmdForceGodCasterAndVictimNotification(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Forcegod", LVL_GOD, 1001)
	target := makeCommandTestSession(t, m, "Forcetarget", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdForce(actor, []string{"Forcetarget", "whoami"}); err != nil {
		t.Fatalf("cmdForce returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("actor acknowledgement = %q", got)
	}
	if got := readMsgText(t, target); got != "Forcegod has forced you to 'whoami'.\r\n" {
		t.Fatalf("victim notification = %q", got)
	}
	if got := readMsgText(t, target); got != "Forcetarget" {
		t.Fatalf("forced command output = %q", got)
	}
}

func TestCmdForceRoomAndAllFilterByLevelAndRoom(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Forceactor", LVL_IMPL, 1001)
	roomTarget := makeCommandTestSession(t, m, "Forceroom", 1, 1001)
	awayTarget := makeCommandTestSession(t, m, "Forceaway", 1, 1002)
	highTarget := makeCommandTestSession(t, m, "Forcehigh", LVL_IMPL, 1001)
	for _, sess := range []*Session{actor, roomTarget, awayTarget, highTarget} {
		registerInWorld(t, sess)
	}

	if err := cmdForce(actor, []string{"room", "whoami"}); err != nil {
		t.Fatalf("room force returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("room acknowledgement = %q", got)
	}
	if got := readMsgText(t, roomTarget); got != "Forceactor has forced you to 'whoami'.\r\n" {
		t.Fatalf("room target notification = %q", got)
	}
	if got := readMsgText(t, roomTarget); got != "Forceroom" {
		t.Fatalf("room target command output = %q", got)
	}
	assertNoSessionMessage(t, awayTarget)
	assertNoSessionMessage(t, highTarget)

	if err := cmdForce(actor, []string{"all", "whoami"}); err != nil {
		t.Fatalf("all force returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("all acknowledgement = %q", got)
	}
	if got := readMsgText(t, roomTarget); got != "Forceactor has forced you to 'whoami'.\r\n" {
		t.Fatalf("all room target notification = %q", got)
	}
	if got := readMsgText(t, roomTarget); got != "Forceroom" {
		t.Fatalf("all room target command output = %q", got)
	}
	if got := readMsgText(t, awayTarget); got != "Forceactor has forced you to 'whoami'.\r\n" {
		t.Fatalf("all away target notification = %q", got)
	}
	if got := readMsgText(t, awayTarget); got != "Forceaway" {
		t.Fatalf("all away target command output = %q", got)
	}
	assertNoSessionMessage(t, highTarget)
}

func TestCmdForceBelowGreaterGodTreatsAllAsCharacterName(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Forcegod", LVL_GOD, 1001)
	registerInWorld(t, actor)

	if err := cmdForce(actor, []string{"all", "whoami"}); err != nil {
		t.Fatalf("cmdForce returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != noPersonHere {
		t.Fatalf("message = %q, want %q", got, noPersonHere)
	}
}

func assertNoSessionMessage(t *testing.T, s *Session) {
	t.Helper()
	select {
	case msg := <-s.send:
		t.Fatalf("unexpected message for %s: %s", s.player.Name, string(msg))
	default:
	}
}
