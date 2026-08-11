package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdColorPersistsLevel verifies cmdColor actually sets PrfColor1/2
// instead of just printing a message — it used to be a placebo that never
// touched player state, so "toggle" always reported color as off regardless
// of what "color on"/"color complete" said.
func TestCmdColorPersistsLevel(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdColor(s, nil); err != nil {
		t.Fatalf("cmdColor: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "color level is off") {
		t.Errorf("initial color level: got %q", msg)
	}

	if err := cmdColor(s, []string{"complete"}); err != nil {
		t.Fatalf("cmdColor: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "now complete") {
		t.Errorf("color complete message: got %q", msg)
	}
	if s.player.Flags&(1<<game.PrfColor1) == 0 || s.player.Flags&(1<<game.PrfColor2) == 0 {
		t.Errorf("color complete should set both PrfColor1 and PrfColor2")
	}

	if err := cmdColor(s, []string{"sparse"}); err != nil {
		t.Fatalf("cmdColor: %v", err)
	}
	readSessionText(t, s)
	if s.player.Flags&(1<<game.PrfColor1) == 0 {
		t.Errorf("color sparse should set PrfColor1")
	}
	if s.player.Flags&(1<<game.PrfColor2) != 0 {
		t.Errorf("color sparse should clear PrfColor2")
	}

	// "on" is a shorthand for "complete" — src/act.informative.c do_color().
	if err := cmdColor(s, []string{"on"}); err != nil {
		t.Fatalf("cmdColor: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "now complete") {
		t.Errorf("color on should behave like complete: got %q", msg)
	}

	if err := cmdColor(s, []string{"bogus"}); err != nil {
		t.Fatalf("cmdColor: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "Usage") {
		t.Errorf("invalid color level: got %q", msg)
	}
}

// TestCmdAutoExitTogglesPersistently verifies cmdAutoExit actually flips
// Player.AutoExit instead of just printing a fixed "Auto-exit toggled."
// message every time regardless of state.
func TestCmdAutoExitTogglesPersistently(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if !s.player.GetAutoExit() {
		t.Fatalf("NewPlayer should default AutoExit to true")
	}

	if err := cmdAutoExit(s, nil); err != nil {
		t.Fatalf("cmdAutoExit: %v", err)
	}
	if s.player.GetAutoExit() {
		t.Errorf("first toggle should disable autoexit")
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "disabled") {
		t.Errorf("autoexit-disabled message: got %q", msg)
	}

	if err := cmdAutoExit(s, nil); err != nil {
		t.Fatalf("cmdAutoExit: %v", err)
	}
	if !s.player.GetAutoExit() {
		t.Errorf("second toggle should re-enable autoexit")
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "enabled") {
		t.Errorf("autoexit-enabled message: got %q", msg)
	}
}

// TestCmdToggleIgnoresArguments verifies cmdToggle always shows the full
// preference grid regardless of its argument — src/act.informative.c
// do_toggle() never branches on its argument at all. It used to special-case
// "toggle autoexit" as the only working sub-toggle and reject every other
// name; individual toggles now have their own top-level commands instead
// (see commands.go's "Preference toggles" block).
//
// The grid is the C-verbatim sprintf from act.informative.c:2513-2579 — the
// header-less 8-row layout beginning "Hit Pnt Display:". This assertion would
// have failed against the prior Go re-skin (an invented "Toggles:\r\n  hitpoint
// : OFF ..." table).
func TestCmdToggleIgnoresArguments(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdToggle(s, nil); err != nil {
		t.Fatalf("cmdToggle: %v", err)
	}
	noArgMsg := readSessionText(t, s)
	if !strings.HasPrefix(noArgMsg, "Hit Pnt Display:") {
		t.Errorf("no-arg toggle should show the C grid: got %q", noArgMsg)
	}
	if !strings.Contains(noArgMsg, "Broadcasts:") {
		t.Errorf("grid should include the final Broadcasts column: got %q", noArgMsg)
	}

	if err := cmdToggle(s, []string{"brief"}); err != nil {
		t.Fatalf("cmdToggle: %v", err)
	}
	argMsg := readSessionText(t, s)
	if !strings.HasPrefix(argMsg, "Hit Pnt Display:") {
		t.Errorf("toggle with an argument should still show the C grid: got %q", argMsg)
	}
	if strings.Contains(argMsg, "Unknown toggle") {
		t.Errorf("toggle should never reject an argument: got %q", argMsg)
	}
}
