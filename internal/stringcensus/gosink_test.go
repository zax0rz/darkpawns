package stringcensus

import (
	"testing"
)

// roomMessageFixture is the shape the widening exists for: an invented line
// enters an unlisted output wrapper (World.roomMessage, which forwards to
// sendActRoomMessage) instead of a primitive, so the table is the only thing
// that can see it. The line is specNoMoveDown's invented "blocks your way
// down." (pkg/game/spec_procs3.go:310) in miniature.
const roomMessageFixture = `package game

func (w *World) sendActRoomMessage(roomVNum int, msg string) {
	w.SendMessage(msg)
}

func (w *World) roomMessage(roomVNum int, msg string) {
	w.sendActRoomMessage(roomVNum, msg + "\r\n")
}

func specNoMoveDown(w *World) {
	w.roomMessage(1001, "%s blocks your way down.")
}
`

func fixtureReport(t *testing.T, sinks map[string]goSink) *Report {
	t.Helper()
	src := []sourceFile{{rel: "pkg/game/spec.go", data: roomMessageFixture}}
	cands, unresolved, err := extractGoFilesWithSinks(src, sinks)
	if err != nil {
		t.Fatalf("extractGoFilesWithSinks: %v", err)
	}
	return buildReport(src, nil, nil, cands, nil, unresolved)
}

// TestRoomMessageIsASink is the R5h proof for the widened table: the invented
// line is go-only through roomMessage, and the same fixture reports nothing at
// all once roomMessage is removed from the sink index. Without the roomMessage
// entry the census is blind to it, which is the bug this table change fixes.
func TestRoomMessageIsASink(t *testing.T) {
	report := fixtureReport(t, goSinkIndex())
	const want = "blocks your way down."
	var found *Segment
	for i, s := range report.GoOnlySegments {
		if s.Text == want {
			found = &report.GoOnlySegments[i]
		}
	}
	if found == nil {
		t.Fatalf("go-only = %+v, want %q", report.GoOnlySegments, want)
	}
	if found.Sink != ".roomMessage" {
		t.Fatalf("sink = %q, want .roomMessage", found.Sink)
	}

	without := goSinkIndex()
	delete(without, ".roomMessage")
	delete(without, "roomMessage")
	counterfactual := fixtureReport(t, without)
	if counterfactual.GoOnly != 0 {
		t.Fatalf("without the roomMessage entry go-only = %+v, want none", counterfactual.GoOnlySegments)
	}
}

// TestGoSinkTextArgsSelectsTextArguments pins the per-argument table: a sink
// that mixes message text with a log line must contribute only the text, or
// the census would report server log prose as player-facing.
func TestGoSinkTextArgsSelectsTextArguments(t *testing.T) {
	src := []sourceFile{{rel: "pkg/game/teleport.go", data: `package game

func f(w *World) {
	elementTeleportPerson(w, nil, 10, "You appear in a brilliant flash of light.", "You vanish in a brilliant flash of light.", "You arrive in a brilliant flash of light.", true, false, "element teleport moved a body")
}
`}}
	cands, _, err := extractGoFiles(src)
	if err != nil {
		t.Fatalf("extractGoFiles: %v", err)
	}
	got := literalSet(cands)
	want := []string{
		"You appear in a brilliant flash of light.",
		"You vanish in a brilliant flash of light.",
		"You arrive in a brilliant flash of light.",
	}
	if len(got) != len(want) {
		t.Fatalf("candidates = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %q, want %q", got, want)
		}
	}
	for _, c := range cands {
		if c.raw == "element teleport moved a body" {
			t.Fatalf("log message collected as player text: %+v", c)
		}
	}
}
