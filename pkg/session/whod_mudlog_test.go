package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhodMudlogProducers(t *testing.T) {
	for _, tc := range []struct {
		name, arg, payload    string
		initial, atLog, after int
	}{
		{"on", "on", "WHOD turned on by Logactor.", game.WhodShowOff, game.WhodShowOn, game.WhodShowOn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, a, _, watch := wizardLogFixture(t)
			watch.player.SetLevel(34)
			watch.player.SetPlrFlag(game.PrfLog2, false)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			below := addObserver(t, m, "BelowWhod", 33, true, false)
			a.player.SetInvisLevel(40)
			m.world.WhodDisplay.Mode = tc.initial
			file := captureMudlogFile(t)
			ready := false
			game.SetLogWriter(&flagProbe{buf: file, when: func() { ready = len(a.send) > 0 && m.world.WhodDisplay.Mode == tc.atLog }})
			if err := executeCommand(a, "whod", []string{tc.arg}, false); err != nil {
				t.Fatal(err)
			}
			acks := map[string]string{"on": "WHOD turned on.\n\r", "off": "WHOD turned off.\n\r", "remove": "name will not be shown on WHOD.\n\r", "add": "name will now be shown on WHOD.\n\r"}
			if got := strings.Join(drainSessionText(t, a), ""); got != acks[tc.name] {
				t.Fatalf("acknowledgement = %q", got)
			}
			if !ready {
				t.Fatal("WHOD must acknowledge before log at C mutation boundary")
			}
			if !strings.Contains(file.String(), tc.payload) {
				t.Fatalf("missing file payload: %q", file.String())
			}
			if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+tc.payload+" ]\r\n" {
				t.Fatalf("observer = %q", got)
			}
			if got := strings.Join(drainSessionText(t, below), ""); got != "" {
				t.Fatalf("below threshold = %q", got)
			}
			if m.world.WhodDisplay.Mode != tc.after {
				t.Fatalf("mode=%d", m.world.WhodDisplay.Mode)
			}
		})
	}
}

func TestWhodMudlogRefusals(t *testing.T) {
	m, a, _, w := wizardLogFixture(t)
	file := captureMudlogFile(t)
	for _, arg := range []string{"", "invalid", "on"} {
		m.world.WhodDisplay.Mode = game.WhodShowOn
		if err := cmdWhod(a, []string{arg}); err != nil {
			t.Fatal(err)
		}
	}
	if file.Len() != 0 || len(w.send) != 0 {
		t.Fatal("WHOD refusal/display must not log")
	}
}
