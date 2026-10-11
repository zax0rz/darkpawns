package game

import (
	"bytes"
	"testing"
)

// TestScriptAdapterLogIsBriefImmortalNoFile pins the type, level and file flag
// of every one of the 57 D7 Lua seam producers at once. The bindings emit
// through Bridge.Log, which carries none of the three (pkg/scripting asserts
// the payload and the file flag there); this adapter is the only place that
// turns Log into a mudlog call, and it must be BRF / LVL_IMMORT / file FALSE.
//
// The three arguments are each discriminated exactly, and the controls script
// mutates each one on its own:
//
//   - type: a LOG1 observer (log_level 1) receives the line, so the type is at
//     most BRF; a syslog-off observer (log_level 0) does not, so the type is at
//     least BRF. Together: BRF, not "at least BRF" — NRM (2) and CMP (3) would
//     silence the LOG1 observer.
//   - level: an observer at exactly LVL_IMMORT (31) receives it and one at 30
//     does not, which pins 31 rather than any higher level.
//   - file: the file sink must stay entirely empty.
func TestScriptAdapterLogIsBriefImmortalNoFile(t *testing.T) {
	const payload = "[Lua] Invalid argument passed to lua_action."

	newObserver := func(id int, name string, level int, log1 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		return p
	}
	atImmort := newObserver(201, "Scriptimmort", LVL_IMMORT, true)
	below := newObserver(202, "Scriptbelow", LVL_IMMORT-1, true)
	quiet := newObserver(203, "Scriptquiet", 40, false)
	provider := &milestoneProvider{
		players: []*Player{atImmort, below, quiet},
		lines:   map[string]string{},
	}
	oldProvider := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(oldProvider) })

	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })

	NewWorldScriptableAdapter(nil).Log(payload)

	// Ordered so each argument's mutation trips its own assertion: a wrong
	// level shows up here first, a wrong type at the observer below, and a
	// wrong file flag last.
	if got := provider.lines[below.Name]; got != "" {
		t.Fatalf("level-%d observer saw %q, want the level filter to hide it", LVL_IMMORT-1, got)
	}
	if want := "[ " + payload + " ]\r\n"; provider.lines[atImmort.Name] != want {
		t.Fatalf("level-%d BRF observer bytes = %q, want %q", LVL_IMMORT, provider.lines[atImmort.Name], want)
	}
	if got := provider.lines[quiet.Name]; got != "" {
		t.Fatalf("syslog-off observer saw %q, so the type is higher than BRF", got)
	}
	if file.Len() != 0 {
		t.Fatalf("file flag must be FALSE, so the file sink must be entirely empty: %q", file.String())
	}
}
