package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// zoneErrorObservers stages the contract of log_zone_error's first producer
// (src/db.c:2052-2053): NRM (2), LVL_GOD (34), file TRUE.
//
//   - watch: level 34 with LOG2 only (log_level 2) sees NRM.
//   - below: level 33 differs by exactly one level.
//   - briefOnly: level 40 with LOG1 only (log_level 1) is silent, so the type is
//     not BRF (1).
//   - complete: level 40 with LOG1+LOG2 (log_level 3) sees it, so the type is
//     not CMP (3).
func zoneErrorObservers(t *testing.T) (*milestoneProvider, *bytes.Buffer, *Player, *Player, *Player, *Player) {
	t.Helper()
	newObserver := func(id int, name string, level int, log1, log2 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		p.SetPlrFlag(PrfLog2, log2)
		return p
	}
	watch := newObserver(201, "Zonewatch", LVL_GOD, false, true)
	below := newObserver(202, "Zonebelow", LVL_GOD-1, false, true)
	briefOnly := newObserver(203, "Zonebrief", 40, true, false)
	complete := newObserver(204, "Zonecomplete", 40, true, true)
	provider := &milestoneProvider{
		players: []*Player{watch, below, briefOnly, complete},
		lines:   map[string]string{},
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	return provider, file, watch, below, briefOnly, complete
}

// TestZoneErrorMudlogBranches proves all six C ZONE_ERROR branches
// (src/db.c:2174,2187,2202,2207,2250,2279) reach log_zone_error's first
// producer: exact payload, NRM/LVL_GOD/file TRUE, once per occurrence. The
// door branch's one C condition is the port's two arms (direction out of range,
// or no such exit); each is exercised separately and neither can fire twice.
func TestZoneErrorMudlogBranches(t *testing.T) {
	for _, tc := range []struct {
		name     string
		commands []parser.ZoneCommand
		message  string
	}{
		{"P target obj not found", []parser.ZoneCommand{{Command: "P", Arg1: 200, Arg2: 10, Arg3: 999}}, "target obj not found"},
		{"G non-existant mob", []parser.ZoneCommand{{Command: "G", Arg1: 200, Arg2: 10}}, "attempt to give obj to non-existant mob"},
		{"E non-existant mob", []parser.ZoneCommand{{Command: "E", Arg1: 200, Arg2: 10}}, "trying to equip non-existant mob"},
		{"E invalid equipment pos", []parser.ZoneCommand{
			{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100},
			{Command: "E", IfFlag: 1, Arg1: 200, Arg2: 10, Arg3: 99},
		}, "invalid equipment pos number"},
		{"D direction out of range", []parser.ZoneCommand{{Command: "D", Arg1: 100, Arg2: 9, Arg3: 1}}, "door does not exist"},
		{"D no such exit", []parser.ZoneCommand{{Command: "D", Arg1: 100, Arg2: 0, Arg3: 1}}, "door does not exist"},
		{"unknown command disabled", []parser.ZoneCommand{{Command: "Q", Arg1: 1}}, "unknown cmd in reset table; cmd disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			// C records the 1-based zone-file line in ZCMD.line (src/db.c:1597)
			// and prints it in the second ordered diagnostic.
			for i := range tc.commands {
				tc.commands[i].Line = 5 + i
			}
			zone := parser.Zone{Number: 1, TopRoom: 199, Commands: tc.commands}
			w.parsedData = nil
			w.zones[1] = &zone
			provider, file, watch, below, briefOnly, complete := zoneErrorObservers(t)
			payload := "SYSERR: error in zone file: " + tc.message
			file2 := &milestoneWriter{payload: payload, at: func() {
				// src/db.c:2280 sets ZCMD.command = '*' after the producer.
				if current, ok := w.GetZone(1); ok && current.Commands[0].Command == "*" {
					t.Error("producer must precede the reset table mutation")
				}
			}}
			SetLogWriter(file2)
			if err := s.ExecuteZoneReset(&zone); err != nil {
				t.Fatalf("reset failed: %v", err)
			}
			SetLogWriter(file)
			if got := strings.Count(file2.String(), payload); got != 1 {
				t.Fatalf("file payload count=%d want 1; log=%q", got, file2.String())
			}
			want := "[ " + payload + " ]\r\n"
			if !strings.Contains(provider.lines[watch.Name], want) {
				t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
			}
			// The second ordered line, C's "offending cmd" diagnostic, carrying
			// the command character, the zone number and ZCMD.line.
			second := fmt.Sprintf("SYSERR: ...offending cmd: '%c' cmd in zone #1, line %d",
				tc.commands[len(tc.commands)-1].Command[0], 5+len(tc.commands)-1)
			if got := strings.Count(file2.String(), second); got != 1 {
				t.Fatalf("second line count=%d want 1; log=%q", got, file2.String())
			}
			if wantSecond := "[ " + second + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], wantSecond) {
				t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], wantSecond)
			}
			if !strings.Contains(provider.lines[complete.Name], want) {
				t.Fatal("complete-syslog observer missed the NRM line")
			}
			for name, line := range map[string]string{
				below.Name:     provider.lines[below.Name],
				briefOnly.Name: provider.lines[briefOnly.Name],
			} {
				if strings.Contains(line, payload) {
					t.Fatalf("%s received a filtered line %q", name, line)
				}
			}
		})
	}
}
