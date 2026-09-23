package stringcensus

import (
	"strings"
	"testing"
)

// extractGoSource runs the Go extractor over one synthetic file.
func extractGoSource(t *testing.T, src string) ([]candidate, []unresolvedSite) {
	t.Helper()
	cands, unresolved, err := extractGoFiles([]sourceFile{{rel: "pkg/game/x.go", data: src}})
	if err != nil {
		t.Fatalf("extractGoFiles: %v", err)
	}
	return cands, unresolved
}

func literalSet(cands []candidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.raw)
	}
	return out
}

func TestExtractGoDirectLiteral(t *testing.T) {
	src := `package game

func f(p *Player) {
	p.SendMessage("You feel much better now.\r\n")
}
`
	got, _ := extractGoSource(t, src)
	if len(got) != 1 || got[0].raw != "You feel much better now.\r\n" || got[0].sink != ".SendMessage" {
		t.Fatalf("candidates = %+v", got)
	}
	if got[0].line != 4 {
		t.Fatalf("line = %d, want 4", got[0].line)
	}
}

func TestExtractGoSprintfArguments(t *testing.T) {
	src := `package game

import "fmt"

func f(p *Player, who string) {
	p.Send(fmt.Sprintf("You strike %s and deal a lot of damage.", who))
}
`
	got, _ := extractGoSource(t, src)
	want := []string{"You strike %s and deal a lot of damage."}
	if strings.Join(literalSet(got), "|") != strings.Join(want, "|") {
		t.Fatalf("candidates = %+v", got)
	}
}

func TestExtractGoConcatenatedLiterals(t *testing.T) {
	src := `package game

func f(p *Player) {
	p.SendMessage("Return to the temple and QUIT to leave" +
		" the game and keep your equipment.")
}
`
	got, _ := extractGoSource(t, src)
	want := "Return to the temple and QUIT to leave the game and keep your equipment."
	if len(got) != 1 || got[0].raw != want {
		t.Fatalf("candidates = %+v, want %q", got, want)
	}
}

func TestExtractGoResolvesSingleAssignmentVariable(t *testing.T) {
	src := `package game

func f(p *Player) {
	msg := "You are already standing on your feet."
	p.SendMessage(msg)
}
`
	got, _ := extractGoSource(t, src)
	if len(got) != 1 {
		t.Fatalf("candidates = %+v, want 1", got)
	}
	if got[0].raw != "You are already standing on your feet." {
		t.Fatalf("raw = %q", got[0].raw)
	}
	if want := ".SendMessage(via msg)"; got[0].sink != want {
		t.Fatalf("sink = %q, want %q", got[0].sink, want)
	}
}

func TestExtractGoReportsAmbiguousVariable(t *testing.T) {
	src := `package game

func f(p *Player, cond bool) {
	msg := "One of two messages that might be sent."
	if cond {
		msg = "The other message that might be sent."
	}
	p.SendMessage(msg)
}
`
	got, unresolved := extractGoSource(t, src)
	if len(got) != 0 {
		t.Fatalf("candidates = %+v, want none: the value is ambiguous", got)
	}
	if len(unresolved) != 1 || unresolved[0].expr != "msg" {
		t.Fatalf("unresolved = %+v, want one row for msg", unresolved)
	}
}

func TestExtractGoReportsBuiltStringFromLocal(t *testing.T) {
	src := `package game

import "strings"

func f(p *Player, lines []string) {
	body := strings.Join(lines, "\r\n")
	p.SendMessage(body)
}
`
	got, unresolved := extractGoSource(t, src)
	if len(got) != 0 {
		t.Fatalf("candidates = %+v, want none", got)
	}
	if len(unresolved) != 1 || unresolved[0].expr != "body" {
		t.Fatalf("unresolved = %+v, want one row for body", unresolved)
	}
}

func TestExtractGoIgnoresParametersAndUnknownCalls(t *testing.T) {
	src := `package game

func f(p *Player, message string) {
	p.SendMessage(message)
	p.AnythingElse("a literal that is not player output")
	other.Send("a literal on a different receiver is still a sink")
}
`
	got, unresolved := extractGoSource(t, src)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %+v, want none: a parameter is not a lead", unresolved)
	}
	// Only the .Send call qualifies: .SendMessage's argument is a parameter and
	// .AnythingElse is not a sink. The receiver name does not matter, because
	// the census does not type-check.
	if len(got) != 1 || got[0].sink != ".Send" {
		t.Fatalf("candidates = %+v, want only the .Send literal", got)
	}
}

func TestExtractGoActAndFunctionSinks(t *testing.T) {
	src := `package game

func f(w *World, ch, vict Actor) {
	Act(w, true, ch, vict, nil, nil, "The guard notices you and frowns.", "", ToRoom)
	SendToRoom(w, ch, "You hear a distant rumble underground.")
	SendToViolation(w, ch, "not a sink at all, very long string here")
}
`
	got, _ := extractGoSource(t, src)
	if len(got) != 2 {
		t.Fatalf("candidates = %+v, want 2", got)
	}
	if got[0].sink != "Act" || got[1].sink != "SendToRoom" {
		t.Fatalf("sinks = %q, %q; want Act, SendToRoom (the call spelling)", got[0].sink, got[1].sink)
	}
}

func TestExtractGoSkipsTestFiles(t *testing.T) {
	files := []sourceFile{
		{rel: "pkg/game/x.go", data: "package game\nfunc f(p *Player) { p.Send(\"a real message here\") }\n"},
		{rel: "pkg/game/x_test.go", data: "package game\nfunc TestX(t *testing.T) { Send(\"a test-only message\") }\n"},
	}
	filtered := make([]sourceFile, 0, len(files))
	for _, f := range files {
		if isGoSource(f.rel, nil) {
			filtered = append(filtered, f)
		}
	}
	cands, _, err := extractGoFiles(filtered)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].file != "pkg/game/x.go" {
		t.Fatalf("candidates = %+v, want only the non-test file", cands)
	}
}
