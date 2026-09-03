package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSnoopRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("snoop")
	if !ok {
		t.Fatal("snoop command is not registered")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("snoop gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosDead)
	}
}

func TestCmdSnoopStateGatesUseCBranches(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Session, *Session)
		args  []string
		want  string
	}{
		{
			name: "busy self",
			setup: func(actor, target *Session) {
				actor.snooping = target
				target.snoopBy = actor
			},
			args: []string{"Snooptarget"},
			want: "Busy already. \r\n",
		},
		{
			name: "dont be stupid",
			setup: func(actor, target *Session) {
				target.snooping = actor
			},
			args: []string{"Snooptarget"},
			want: "Don't be stupid.\r\n",
		},
		{
			name: "level boundary",
			setup: func(actor, target *Session) {
				target.player.Level = LVL_IMPL
			},
			args: []string{"Snooptarget"},
			want: "You can't.\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeTestManager(t)
			actor := makeCommandTestSession(t, m, "Snoopactor", LVL_IMPL, 1001)
			target := makeCommandTestSession(t, m, "Snooptarget", 1, 1001)
			registerInWorld(t, actor)
			registerInWorld(t, target)
			tt.setup(actor, target)

			if err := cmdSnoop(actor, tt.args); err != nil {
				t.Fatalf("cmdSnoop returned error: %v", err)
			}
			if got := readMsgText(t, actor); got != tt.want {
				t.Fatalf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCmdSnoopStopAndRelayMatchC(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Snoopactor", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Snooptarget", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdSnoop(actor, []string{"Snooptarget"}); err != nil {
		t.Fatalf("starting snoop returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("start message = %q, want C OK", got)
	}
	if actor.snooping != target || target.snoopBy != actor {
		t.Fatal("snoop start did not install both descriptor links")
	}

	target.SendMessage("You say 'snooped'.\r\n")
	if got := readMsgText(t, actor); got != "% You say 'snooped'.\r\n%%" {
		t.Fatalf("snooped output = %q, want C descriptor delimiters", got)
	}
	_ = readMsgText(t, target)

	if err := cmdSnoop(actor, nil); err != nil {
		t.Fatalf("stopping snoop returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "You stop snooping.\r\n" {
		t.Fatalf("stop message = %q, want C stop message", got)
	}
	if actor.snooping != nil || target.snoopBy != nil {
		t.Fatal("snoop stop did not clear both descriptor links")
	}

	if err := cmdSnoop(actor, nil); err != nil {
		t.Fatalf("repeated stop returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "You aren't snooping anyone.\r\n" {
		t.Fatalf("repeated stop message = %q, want C no-snoop message", got)
	}
}

func TestCmdSnoopUsesSwitchedOriginalLevel(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Snoopactor", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Snooptarget", 1, 1001)
	target.isSwitched = true
	target.switchedOriginalLevel = LVL_IMPL
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdSnoop(actor, []string{"Snooptarget"}); err != nil {
		t.Fatalf("cmdSnoop returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "You can't.\r\n" {
		t.Fatalf("switched-original message = %q, want C level gate", got)
	}
}

func TestSnoopInputRelayUsesCFraming(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Snoopactor", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Snooptarget", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	target.snoopBy = actor
	target.forwardSnoopInput("say", "hello  world", nil)
	if got := readMsgText(t, actor); got != "% say hello  world\r\n" {
		t.Fatalf("snooped input = %q, want C percent prefix and CRLF", got)
	}
}
