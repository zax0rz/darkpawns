package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// src/act.offensive.c:160 -> src/fight.c:534-580 ->
// src/handler.c:1116-1117,1172-1174. The notice's AWAKE gate is
// src/utils.c:422-423; assert it through real session dispatch and rendering.
func TestInstakillFollowerNoticeBeforeMenuWire(t *testing.T) {
	for _, tc := range []struct {
		name     string
		position int
		notice   string
	}{
		{"awake", combat.PosStanding, "Follower stops following you.\r\n"},
		{"sleeping", combat.PosSleeping, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := makeGateTestManager(t, false)
			m.world.StopAITicker()
			killer := makeGateSession(t, m, 1, "Killer", 40)
			victim := makeGateSession(t, m, 2, "Victim", 10)
			follower := makeGateSession(t, m, 3, "Follower", 10)
			victim.terminalNamed = true
			if err := cmdFollow(follower, []string{"Victim"}); err != nil {
				t.Fatal(err)
			}
			_ = captureWire(killer)
			_ = captureWire(victim)
			_ = captureWire(follower)
			victim.player.SetPosition(tc.position)
			if err := cmdKill(killer, []string{"victim"}); err != nil {
				t.Fatal(err)
			}
			_ = captureWire(victim)
			m.BeginHeartbeatOutput()
			m.ExtractPendingChars()
			m.EndHeartbeatOutput()
			var wire strings.Builder
			for len(victim.send) > 0 {
				frame, ok := victim.RenderTerminalFrame(<-victim.send)
				if ok {
					frame = victim.TrackPrompt(frame)
					switch frame.Kind {
					case FrameText, FramePrompt, FrameEntryPrompt:
						wire.WriteString(frame.Text)
					}
				}
			}
			got := wire.String()
			want := tc.notice + menuText
			if got != want {
				t.Fatalf("victim extraction wire=%q, want %q", got, want)
			}
			if got := captureWire(follower); !strings.Contains(got, "You stop following Victim.\r\n") {
				t.Fatalf("follower wire=%q", got)
			}
		})
	}
}
