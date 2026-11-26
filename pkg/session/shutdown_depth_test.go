package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShutdownAbbreviationKeepsCGuardBytes(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Shutdownactor", game.LVL_IMPL, 1001)

	if err := cmdShutdowStub(s); err != nil {
		t.Fatalf("cmdShutdowStub: %v", err)
	}
	if got, want := readMsgText(t, s), "If you want to shut something down, say so!\r\n"; got != want {
		t.Fatalf("guard output = %q, want %q", got, want)
	}
}

func TestShutdownPlanUsesCGlobalMessages(t *testing.T) {
	tests := []struct {
		name      string
		option    string
		broadcast string
		marker    string
	}{
		{name: "shutdown", broadcast: "Shutting down.\r\n"},
		{name: "reboot", option: "reboot", broadcast: "Rebooting.. come back in a minute or two.\r\n", marker: ".fastboot"},
		{name: "die", option: "die", broadcast: "Shutting down for maintenance.\r\n", marker: ".killscript"},
		{name: "pause", option: "pause", broadcast: "Shutting down for maintenance.\r\n", marker: "pause"},
		{name: "mixed-case reboot", option: "ReBoOt", broadcast: "Rebooting.. come back in a minute or two.\r\n", marker: ".fastboot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, ok := shutdownPlanFor(tt.option)
			if !ok {
				t.Fatal("shutdown option rejected")
			}
			if plan.broadcast != tt.broadcast || plan.marker != tt.marker {
				t.Fatalf("plan = %#v, want broadcast %q marker %q", plan, tt.broadcast, tt.marker)
			}
		})
	}

	if _, ok := shutdownPlanFor("unknown"); ok {
		t.Fatal("unknown shutdown option accepted")
	}
}

func TestShutdownCommandBroadcastsCTextAndRequestsLifecycle(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Shutdownactor", game.LVL_IMPL, 1001)
	m.sessions[actor.player.Name] = actor

	if err := cmdShutdown(actor, []string{"ReBoOt"}); err != nil {
		t.Fatalf("cmdShutdown: %v", err)
	}
	if got, want := readMsgText(t, actor), "Rebooting.. come back in a minute or two.\r\n"; got != want {
		t.Fatalf("broadcast = %q, want %q", got, want)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("all-save acknowledgement = %q, want %q", got, want)
	}
	select {
	case request := <-m.ShutdownRequests():
		if request.Marker != ".fastboot" {
			t.Fatalf("shutdown marker = %q, want .fastboot", request.Marker)
		}
	default:
		t.Fatal("shutdown request was not queued")
	}
}
