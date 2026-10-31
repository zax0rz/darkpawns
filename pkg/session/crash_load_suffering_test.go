package session

// crash_load_suffering_test.go — the DEX arm's TYPE_SUFFERING line at entry.
// check_for_bad_stats' DEX arm calls damage(ch, ch, 40, TYPE_SUFFERING)
// (src/handler.c:665-672), and TYPE_SUFFERING is not IS_WEAPON, so damage()
// always calls skill_message (src/fight.c:1534-1535). Its set-399 hit victim
// line goes to ch->desc: the entering descriptor, while the character is still
// inside Crash_load and has no registered session. Zach chose (2026-10-15) to
// deliver it through the entry sinks rather than record a divergence.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// lib/misc/messages set 399 has two records, so dice(1,2) picks one hit line.
var sufferingHitLines = []string{
	"You lie on the ground suffering from your terrible wounds...",
	"You lie on the ground suffering from bleeding wounds...",
}

func enterWithZeroDex(t *testing.T, name string, complete bool) string {
	t.Helper()
	m, _, s := crashBadStatsEnv(t, name, 200, game.ApplyDex, []savedWornItem{{201, crashWearHead}})
	m.SetCombatBroadcastFunc()
	m.WireCombatCallbacks()
	if err := m.SetCombatMessageFunc(); err != nil {
		t.Fatalf("SetCombatMessageFunc: %v", err)
	}
	if complete {
		s.player.SetPlrFlag(game.PrfColor1, true)
		s.player.SetPlrFlag(game.PrfColor2, true)
	}
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	return strings.Join(drainSessionTextAndRaw(t, s), "")
}

func TestCrashLoadDexSufferingLineReachesTheEnteringDescriptor(t *testing.T) {
	got := enterWithZeroDex(t, "Sufferer", false)
	trip := "You trip over your own feet and hit your head!\n\r"
	if !strings.HasPrefix(got, trip) {
		t.Fatalf("entry bytes = %q, want the trip line first", got)
	}
	rest := got[len(trip):]
	found := false
	for _, line := range sufferingHitLines {
		if strings.HasPrefix(rest, line+"\r\n") {
			found = true
		}
	}
	if !found {
		t.Fatalf("after the trip line = %q, want a set-399 hit victim line", rest)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("below complete color the line carries no color bytes: %q", got)
	}
}

func TestCrashLoadDexSufferingLineIsRedAtCompleteColor(t *testing.T) {
	got := enterWithZeroDex(t, "Redsuffer", true)
	found := false
	for _, line := range sufferingHitLines {
		if strings.Contains(got, "\x1b[31m"+line+"\r\n\x1b[0m") {
			found = true
		}
	}
	if !found {
		t.Fatalf("entry bytes = %q, want CCRED + victim line + CCNRM (skill_message at C_CMP)", got)
	}
}

// drainSessionTextAndRaw renders each queued message the way the terminal
// receives it (text events gain their line ending; raw events don't) and keeps
// raw events, C's bare color sends, in arrival order.
func drainSessionTextAndRaw(t *testing.T, s *Session) []string {
	t.Helper()
	var out []string
	for {
		select {
		case msg := <-s.send:
			if frame, ok := RenderTerminalFrame(msg); ok && frame.Kind == FrameText {
				out = append(out, frame.Text)
			}
		default:
			return out
		}
	}
}
