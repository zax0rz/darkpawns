package stringcensus

import (
	"strings"
	"testing"
)

func TestLexCSkipsCommentsAndKeepsStrings(t *testing.T) {
	src := "/* \"not a string\" */\n// \"neither is this\"\nchar *s = \"real\";\n"
	var got []string
	for _, g := range cLiteralGroups(lexC(src)) {
		got = append(got, g.text)
	}
	want := []string{"real"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("literals = %#v, want %#v", got, want)
	}
}

func TestLexCConjoinsAdjacentLiterals(t *testing.T) {
	src := "send_to_char(\"Return to the temple and QUIT to leave the game \"\n" +
		"             /* wrap */ \"and keep your equipment.\\r\\n\", ch);\n"
	got := extractCFile("src/x.c", src)
	if len(got) != 1 {
		t.Fatalf("candidates = %d (%+v), want 1", len(got), got)
	}
	if want := "Return to the temple and QUIT to leave the game and keep your equipment.\r\n"; got[0].raw != want {
		t.Fatalf("raw = %q, want %q", got[0].raw, want)
	}
}

func TestLexCEscapes(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`"a\"b"`, `a"b`},
		{`"a\\b"`, `a\b`},
		{`"a\nb"`, "a\nb"},
		{`"\x41\101"`, "AA"},
		{`"tab\there"`, "tab\there"},
		{`"unknown \q escape"`, "unknown q escape"},
	}
	for _, tc := range cases {
		got, ok := unescapeC(tc.raw)
		if !ok {
			t.Errorf("unescapeC(%q) failed", tc.raw)
			continue
		}
		if got != tc.want {
			t.Errorf("unescapeC(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestLexCEscapedQuoteEndsNeitherLiteralNorChar(t *testing.T) {
	src := "char c = '\"'; char *s = \"a\\\"b\" \"c\";\n"
	got := cLiteralGroups(lexC(src))
	if len(got) != 1 {
		t.Fatalf("groups = %d, want 1", len(got))
	}
	if want := `a"bc`; got[0].text != want {
		t.Fatalf("text = %q, want %q", got[0].text, want)
	}
}

func TestLexCSkipsPreprocessorDirectives(t *testing.T) {
	src := "#define GREETINGS \"macro text that is never passed by a call\" \\\n" +
		"    \"continued on the next line\"\n" +
		"send_to_char(\"a real message here\", ch);\n"
	got := extractCFile("src/x.c", src)
	if len(got) != 1 || got[0].raw != "a real message here" {
		t.Fatalf("candidates = %+v, want only the real message", got)
	}
}

func TestExtractCSinkAndFunction(t *testing.T) {
	src := `
static void helper(struct char_data *ch)
{
  sprintf(buf, "%s has quit the game.", GET_NAME(ch));
  send_to_char("Goodbye, friend.. Come back soon!\r\n", ch);
}

ACMD(do_not_here)
{
  stc("Sorry, but you cannot do that here!\r\n", ch);
}

SPECIAL(cityguard)
{
  act("$n says, 'Move along now.'", TRUE, ch, 0, 0, TO_ROOM);
}
`
	got := extractCFile("src/act.other.c", src)
	type row struct{ sink, fn, raw string }
	var rows []row
	for _, c := range got {
		rows = append(rows, row{sink: c.sink, fn: c.fn, raw: c.raw})
	}
	want := []row{
		{sink: "buffered:sprintf", fn: "helper", raw: "%s has quit the game."},
		{sink: "send_to_char", fn: "helper", raw: "Goodbye, friend.. Come back soon!\r\n"},
		{sink: "send_to_char", fn: "do_not_here", raw: "Sorry, but you cannot do that here!\r\n"},
		{sink: "act", fn: "cityguard", raw: "$n says, 'Move along now.'"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d candidates %+v, want %d", len(rows), rows, len(want))
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("candidate %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
}

func TestExtractCFunctionTracking(t *testing.T) {
	src := "int first(void) {\n send_to_char(\"inside the first function\", ch);\n}\n" +
		"int second(void)\n{\n send_to_char(\"inside the second function\", ch);\n}\n"
	got := extractCFile("src/x.c", src)
	if len(got) != 2 {
		t.Fatalf("candidates = %+v, want 2", got)
	}
	if got[0].fn != "first" || got[1].fn != "second" {
		t.Fatalf("functions = %q, %q; want first, second", got[0].fn, got[1].fn)
	}
}

func TestExtractCTableFilesKeepEveryLiteral(t *testing.T) {
	src := "const char *phases[] = {\n \"one-quarter full(waxing)\",\n \"three-quarters full(waxing)\"\n};\n"
	got := extractCFiles([]sourceFile{{rel: "src/constants.c", data: src}})
	if len(got) != 2 {
		t.Fatalf("candidates = %+v, want 2", got)
	}
	if got[0].sink != "table" {
		t.Fatalf("sink = %q, want table", got[0].sink)
	}
}

func TestExtractCReportsLineNumbers(t *testing.T) {
	src := "int f(void)\n{\n\n  send_to_char(\"a line of player text\", ch);\n}\n"
	got := extractCFile("src/x.c", src)
	if len(got) != 1 || got[0].line != 4 {
		t.Fatalf("candidates = %+v, want one at line 4", got)
	}
}
