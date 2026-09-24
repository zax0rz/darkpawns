package session

import (
	"os"
	"strings"
	"testing"
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
