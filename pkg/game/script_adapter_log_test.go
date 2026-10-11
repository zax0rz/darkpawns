package game

import (
	"bytes"
	"strings"
	"testing"
)

// TestScriptAdapterLogIsBriefImmortalNoFile pins the type, level and file flag
// of every one of the 57 D7 Lua seam producers at once. The bindings emit
// through Bridge.Log, which carries none of the three (pkg/scripting asserts
// the payload and the file flag there); this adapter is the only place that
// turns Log into a mudlog call, and it must be BRF / LVL_IMMORT / file FALSE.
//
// BRF from higher verbosity: BRF is the lowest type (1), so the discriminator
// is an observer whose syslog is off (log_level 0). It is silent, which proves
// the type is not NRM (2) or CMP (3); the LOG1 observer receiving the line
// proves the type is at most 1. Together: BRF, not "at least BRF".
func TestScriptAdapterLogIsBriefImmortalNoFile(t *testing.T) {
	const payload = "[Lua] Invalid argument passed to lua_action."

	newObserver := func(id int, name string, level int, log1 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		return p
	}
	brfOnly := newObserver(201, "Scriptbrf", 40, true)
	below := newObserver(202, "Scriptbelow", LVL_IMMORT-1, true)
	quiet := newObserver(203, "Scriptquiet", 40, false)
	provider := &milestoneProvider{
		players: []*Player{brfOnly, below, quiet},
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

	if want := "[ " + payload + " ]\r\n"; provider.lines[brfOnly.Name] != want {
		t.Fatalf("BRF observer bytes = %q, want %q", provider.lines[brfOnly.Name], want)
	}
	if got := provider.lines[below.Name]; got != "" {
		t.Fatalf("level-%d observer saw %q, want the level filter to hide it", LVL_IMMORT-1, got)
	}
	if got := provider.lines[quiet.Name]; got != "" {
		t.Fatalf("syslog-off observer saw %q, so the type is higher than BRF", got)
	}
	if strings.Contains(file.String(), payload) {
		t.Fatalf("file flag must be FALSE: %q", file.String())
	}
}
