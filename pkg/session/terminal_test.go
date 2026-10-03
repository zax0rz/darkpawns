package session

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cGreetingsFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/c_greetings.txt")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\n", "\r\n")
}

func TestGreetingsLogoMatchesCFixture(t *testing.T) {
	if want := cGreetingsFixture(t); GreetingsLogo != want {
		t.Fatalf("GreetingsLogo differs from C fixture\ngot:  %q\nwant: %q", GreetingsLogo, want)
	}
}

// Every frame type a session queues renders the way the telnet writer always
// has; the browser's terminal mode now shares these bytes.
func TestRenderTerminalFrame(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  string
		want TerminalFrame
		ok   bool
	}{
		{"event adds a missing line end", `{"type":"event","data":{"type":"text","text":"You hit it."}}`, TerminalFrame{Kind: FrameText, Text: "You hit it.\r\n"}, true},
		{"event keeps C's LFCR as one break", `{"type":"event","data":{"type":"text","text":"a\n\rb\n\r"}}`, TerminalFrame{Kind: FrameText, Text: "a\r\nb\r\n"}, true},
		{"raw event is untouched", `{"type":"event","data":{"type":"raw","text":"\u001b[1;24r"}}`, TerminalFrame{Kind: FrameText, Text: "\x1b[1;24r"}, true},
		{"prompt defaults", `{"type":"prompt","data":{}}`, TerminalFrame{Kind: FramePrompt, Text: "> "}, true},
		{"prompt text", `{"type":"prompt","data":{"text":"\r\n<20hp 5m 30mv> "}}`, TerminalFrame{Kind: FramePrompt, Text: "\r\n<20hp 5m 30mv> "}, true},
		{"raw pager prompt keeps lone carriage return", `{"type":"prompt","data":{"text":"\r[ Return to continue ]","raw":true}}`, TerminalFrame{Kind: FramePrompt, Text: "\r[ Return to continue ]"}, true},
		{"entry prompt keeps its bytes", `{"type":"char_create","data":{"prompt":"\n\rMake your choice: ","secret":false}}`, TerminalFrame{Kind: FrameEntryPrompt, Text: "\n\rMake your choice: "}, true},
		{"secret entry prompt", `{"type":"char_create","data":{"prompt":"Password: ","secret":true}}`, TerminalFrame{Kind: FrameEntryPrompt, Text: "Password: ", Secret: true}, true},
		{"error", `{"type":"error","data":{"message":"nope"}}`, TerminalFrame{Kind: FrameText, Text: "\r\n!! nope\r\n"}, true},
		{"gmcp", `{"type":"gmcp","data":{"package":"Char.Vitals","json":"{}"}}`, TerminalFrame{Kind: FrameGMCP, GMCPPackage: "Char.Vitals", GMCPPayload: "{}"}, true},
		{"vars are not text", `{"type":"vars","data":{"HEALTH":5}}`, TerminalFrame{}, false},
		{"state is not text", `{"type":"state","data":{}}`, TerminalFrame{}, false},
		{"a rotated token is not text", `{"type":"token_refresh","data":{"token":"x"}}`, TerminalFrame{}, false},
	} {
		got, ok := RenderTerminalFrame([]byte(tc.msg))
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: got %+v, %v; want %+v, %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// src/act.comm.c:846-864; src/comm.c:1620-1643: reset follows the
// act line, then exactly one noncompact flush CRLF precedes the prompt.
func TestTerminalFramePreservesLineEndingBeforeANSIReset(t *testing.T) {
	msg := []byte(`{"type":"event","data":{"type":"text","text":"\u001b[37mGroup line\r\n\u001b[0m"}}`)
	got, ok := RenderTerminalFrame(msg)
	want := "\x1b[37mGroup line\r\n\x1b[0m"
	if !ok || got.Text != want {
		t.Fatalf("colored act bytes %q, want %q", got.Text, want)
	}
}

// src/comm.c:1083-1121: C_CMP colors are green >=75%, yellow >=33%,
// red otherwise, independently for hit points, mana and movement.
func TestPromptVitalsUseCCompleteColors(t *testing.T) {
	p := game.NewPlayer(1, "Prompt", 1001)
	p.Health, p.MaxHealth = 100, 100
	p.Mana, p.MaxMana = 50, 100
	p.Move, p.MaxMove = 32, 100
	p.CopyBaseAttributes()
	for _, flag := range []int{int(game.PrfDisphp), int(game.PrfDispmmana), int(game.PrfDispmove), int(game.PrfColor1), int(game.PrfColor2)} {
		p.SetPlrFlag(flag, true)
	}
	s := &Session{player: p}
	want := "\x1b[32m100\x1b[0mH \x1b[33m50\x1b[0mM \x1b[31m32\x1b[0mV > "
	if got := s.promptText(); got != want {
		t.Fatalf("complete vitals %q, want %q", got, want)
	}
	for _, tc := range []struct {
		current, maximum int
		shade            string
	}{
		{32, 100, "31"},
		{33, 100, "33"},
		{74, 100, "33"},
		{75, 100, "32"},
		{131, 400, "31"},
		{132, 400, "33"},
		{299, 400, "33"},
		{300, 400, "32"},
		{0, 0, "31"},
		{1, 0, "32"},
		{-1, 0, "31"},
	} {
		want := fmt.Sprintf("\x1b[%sm%d\x1b[0mH ", tc.shade, tc.current)
		if got := promptVital(tc.current, tc.maximum, "H", true); got != want {
			t.Fatalf("vital %d/%d %q, want %q", tc.current, tc.maximum, got, want)
		}
	}
	p.SetPlrFlag(int(game.PrfColor1), false)
	if got := s.promptText(); got != "100H 50M 32V > " {
		t.Fatalf("normal color should not color vitals: %q", got)
	}
}
