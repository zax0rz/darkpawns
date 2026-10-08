package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// TestLuaRawKillMudlogProducer proves scripts.c:1246-1252: the Lua raw_kill
// binding logs a PC victim at BRF/LVL_IMMORT/file FALSE before the kill, with
// C's two payload forms. The port logged the same facts to slog instead.
func TestLuaRawKillMudlogProducer(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withKiller bool
		payload    string
	}{
		{"with killer", true, "%s killed by %s at Reset test room."},
		{"without killer", false, "%s killed at Reset test room."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := newZoneResetTestSpawner(t)
			victim := NewPlayer(4100, "Rawvictim", 100)
			if err := w.AddPlayer(victim); err != nil {
				t.Fatalf("AddPlayer(victim): %v", err)
			}
			killer := NewPlayer(4101, "Rawkiller", 100)
			if err := w.AddPlayer(killer); err != nil {
				t.Fatalf("AddPlayer(killer): %v", err)
			}

			// BRF (1) at LVL_IMMORT (31), file FALSE: a LOG1 observer at the
			// threshold sees it, one level below does not, an observer with no
			// syslog flags at all stays silent (so the type needs at least
			// log_level 1, i.e. exactly BRF), and nothing reaches the file.
			watch := NewPlayer(4102, "Rawwatch", 100)
			watch.SetLevel(LVL_IMMORT)
			watch.SetPlrFlag(PrfLog1, true)
			below := NewPlayer(4103, "Rawbelow", 100)
			below.SetLevel(LVL_IMMORT - 1)
			below.SetPlrFlag(PrfLog1, true)
			off := NewPlayer(4104, "Rawnormal", 100)
			off.SetLevel(40)
			provider := &milestoneProvider{
				players: []*Player{watch, below, off},
				lines:   map[string]string{},
			}
			oldProvider := getImmortalSessionProvider()
			SetImmortalSessionProvider(provider)
			t.Cleanup(func() { SetImmortalSessionProvider(oldProvider) })
			file := &bytes.Buffer{}
			oldWriter := getLogWriter()
			SetLogWriter(file)
			t.Cleanup(func() { SetLogWriter(oldWriter) })

			var payload string
			if tc.withKiller {
				payload = fmt.Sprintf(tc.payload, victim.GetName(), killer.GetName())
			} else {
				payload = fmt.Sprintf(tc.payload, victim.GetName())
			}

			adapter := NewWorldScriptableAdapter(w)
			victimRef := scripting.CharRef{ID: victim.GetID()}
			if tc.withKiller {
				killerRef := scripting.CharRef{ID: killer.GetID()}
				adapter.RawKill(victimRef, &killerRef, combat.TYPE_UNDEFINED)
			} else {
				adapter.RawKill(victimRef, nil, combat.TYPE_UNDEFINED)
			}

			if want := "[ " + payload + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], want) {
				t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
			}
			if strings.Contains(provider.lines[below.Name], payload) ||
				strings.Contains(provider.lines[off.Name], payload) {
				t.Fatal("level or type filter leak")
			}
			if got := strings.Count(file.String(), payload); got != 0 {
				t.Fatalf("file payload count=%d want 0 (C passes file FALSE); log=%q", got, file.String())
			}
		})
	}
}
