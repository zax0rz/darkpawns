package session

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func colorFrame(t *testing.T, kind string, data interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(ServerMessage{Type: kind, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTerminalColorCodes(t *testing.T) {
	s := makeTestSession(t, makeTestManager(t), "Viewer", 1001, true)
	s.player.SetPlrFlag(game.PrfColor1, true)
	codes := []string{"&&", "&n", "&d", "&b", "&g", "&c", "&r", "&m", "&y", "&w", "&D", "&B", "&G", "&C", "&R", "&M", "&Y", "&W", "&z", "&", "&&r", "\x1b[31m"}
	values := []string{"&", "\x1b[0m", "\x1b[0;30m", "\x1b[0;34m", "\x1b[0;32m", "\x1b[0;36m", "\x1b[0;31m", "\x1b[0;35m", "\x1b[0;33m", "\x1b[0;37m", "\x1b[1;30m", "\x1b[1;34m", "\x1b[1;32m", "\x1b[1;36m", "\x1b[1;31m", "\x1b[1;35m", "\x1b[1;33m", "\x1b[1;37m", "&z", "&", "&r", "\x1b[31m"}
	for i, code := range codes {
		f, ok := s.RenderTerminalFrame(colorFrame(t, MsgEvent, EventData{Type: "raw", Text: code}))
		if !ok || f.Text != values[i] {
			t.Fatalf("%q => %q, want %q", code, f.Text, values[i])
		}
	}
}

func TestTerminalColorPerRecipient(t *testing.T) {
	for _, flag := range []int{game.PrfColor1, game.PrfColor2} {
		s := makeTestSession(t, makeTestManager(t), "Viewer", 1001, true)
		msg := colorFrame(t, MsgEvent, EventData{Type: "text", Text: "&RRed&n &&r &z &"})
		off, _ := s.RenderTerminalFrame(msg)
		if off.Text != "&RRed&n &&r &z &\r\n" {
			t.Fatalf("off bytes %q", off.Text)
		}
		s.player.SetPlrFlag(flag, true)
		on, _ := s.RenderTerminalFrame(msg)
		if on.Text != "\x1b[1;31mRed\x1b[0m &r &z &\r\n" {
			t.Fatalf("flag %d bytes %q", flag, on.Text)
		}
	}
}

func TestTerminalColorEntryOnce(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.charColor = true
	s.sendCharCreatePrompt("motd", "&&r &RRed&n", nil)
	msg := drainMsg(t, s)
	var raw ServerMessage
	if err := json.Unmarshal(msg, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Data.(map[string]interface{})["prompt"] != "&&r &RRed&n" {
		t.Fatal("entry expanded before terminal funnel")
	}
	f, _ := s.RenderTerminalFrame(msg)
	if f.Text != "&r \x1b[1;31mRed\x1b[0m" {
		t.Fatalf("entry expanded twice or not at all: %q", f.Text)
	}
}

func TestTerminalColorBrowserAndPrompts(t *testing.T) {
	s := makeTestSession(t, makeTestManager(t), "Viewer", 1001, true)
	s.player.SetPlrFlag(game.PrfColor2, true)
	for _, kind := range []string{MsgEvent, MsgPrompt} {
		msg := colorFrame(t, kind, map[string]interface{}{"type": "raw", "raw": true, "text": "&gGreen&n"})
		rendered, ok := renderForBrowserTerminalTracked(s, msg)
		if !ok {
			t.Fatal("browser did not render")
		}
		var result struct {
			Data terminalOut `json:"data"`
		}
		if err := json.Unmarshal(rendered, &result); err != nil {
			t.Fatal(err)
		}
		if result.Data.Text != "\x1b[0;32mGreen\x1b[0m" {
			t.Fatalf("browser %s bytes %q", kind, result.Data.Text)
		}
	}
	gmcp := colorFrame(t, "gmcp", map[string]interface{}{"package": "Comm.Test", "json": "&r"})
	f, _ := s.RenderTerminalFrame(gmcp)
	if f.GMCPPayload != "&r" {
		t.Fatal("GMCP metadata expanded")
	}
	s.isSwitched = true
	s.switchedMob = &game.MobInstance{}
	msg := colorFrame(t, MsgPrompt, map[string]interface{}{"raw": true, "text": "&rNPC"})
	f, _ = s.RenderTerminalFrame(msg)
	if f.Text != "&rNPC" {
		t.Fatal("NPC used original PC color preference")
	}
}
