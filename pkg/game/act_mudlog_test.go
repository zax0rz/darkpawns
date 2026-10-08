package game

import (
	"bytes"
	"strings"
	"testing"
)

// actMudlogObservers stages the C producer's contract for comm.c:2525, which
// is CMP (3), LVL_IMMORT, file TRUE.
//
//   - watch: level 31 with a complete syslog (LOG1+LOG2 => log_level 3) sees it.
//   - below: level 30 with a complete syslog is under the level threshold.
//   - briefOnly: level 40, LOG1 only (log_level 1) — silent, so the type is not
//     BRF (1).
//   - normalOnly: level 40, LOG2 only (log_level 2) — silent, so the type is not
//     NRM (2). Only a complete syslog accepts the line, which is CMP.
func actMudlogObservers(t *testing.T) (*milestoneProvider, *bytes.Buffer, *Player, *Player, *Player, *Player) {
	t.Helper()
	newObserver := func(id int, name string, level int, log1, log2 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		p.SetPlrFlag(PrfLog2, log2)
		return p
	}
	watch := newObserver(101, "Actwatch", LVL_IMMORT, true, true)
	below := newObserver(102, "Actbelow", LVL_IMMORT-1, true, true)
	briefOnly := newObserver(103, "Actbrief", 40, true, false)
	normalOnly := newObserver(104, "Actnormal", 40, false, true)
	provider := &milestoneProvider{
		players: []*Player{watch, below, briefOnly, normalOnly},
		lines:   map[string]string{},
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	return provider, file, watch, below, briefOnly, normalOnly
}

// TestActNoValidTargetMudlog proves comm.c:2523-2526's producer at both of the
// port's "no valid target" arms: the C arm (neither ch nor obj holds a room)
// and the port's defensive nil-world arm (C's world is a global array and
// cannot be nil). C logs and returns before any audience is computed; the port
// never delivers a line on either arm.
func TestActNoValidTargetMudlog(t *testing.T) {
	const payload = "SYSERR: no valid target to act()!"
	w, _ := newZoneResetTestSpawner(t)
	for _, tc := range []struct {
		name  string
		world *World
	}{
		{"no valid target", w},
		{"nil world defensive arm", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, file, watch, below, briefOnly, normalOnly := actMudlogObservers(t)
			Act(tc.world, true, nil, nil, nil, nil, "$n waves.", "", ToRoom)
			if got := strings.Count(file.String(), payload); got != 1 {
				t.Fatalf("file payload count=%d want 1; log=%q", got, file.String())
			}
			if want := "[ " + payload + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], want) {
				t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
			}
			for name, line := range map[string]string{
				below.Name:      provider.lines[below.Name],
				briefOnly.Name:  provider.lines[briefOnly.Name],
				normalOnly.Name: provider.lines[normalOnly.Name],
			} {
				if strings.Contains(line, payload) {
					t.Fatalf("%s received a filtered line %q", name, line)
				}
			}
		})
	}
}
