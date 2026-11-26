package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStructuredCommandOver255PlainTruncatesWithExactNotice(t *testing.T) {
	s := filterTestSpeaker(t)
	payload, err := json.Marshal(CommandData{
		Command: "say",
		Args:    []string{strings.Repeat("x", 260)},
	})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(ClientMessage{Type: MsgCommand, Data: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleMessage(frame); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}

	got := drainSessionText(t, s)
	wantNotice := "Line too long.  Truncated to:\r\nsay " + strings.Repeat("x", 251) + "\r\n"
	wantSay := "You say '" + strings.Repeat("x", 251) + "'\r\n"
	if len(got) < 2 {
		t.Fatalf("messages = %q, want notice + cut say output", got)
	}
	if got[0] != wantNotice {
		t.Fatalf("notice = %q, want %q", got[0], wantNotice)
	}
	if got[1] != wantSay {
		t.Fatalf("say output = %q, want %q", got[1], wantSay)
	}
}

func TestStructuredCommandOver255WithDollarTruncatesWithDoubledNotice(t *testing.T) {
	s := filterTestSpeaker(t)
	// "say " (4) + 250*'x' (250) + "$" (costs 2 -> budget exhausted) + "y" remaining.
	payload, err := json.Marshal(CommandData{
		Command: "say",
		Args:    []string{strings.Repeat("x", 250) + "$y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(ClientMessage{Type: MsgCommand, Data: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleMessage(frame); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}

	got := drainSessionText(t, s)
	wantNotice := "Line too long.  Truncated to:\r\nsay " + strings.Repeat("x", 250) + "$$\r\n"
	wantSay := "You say '" + strings.Repeat("x", 250) + "$'\r\n"
	if len(got) < 2 {
		t.Fatalf("messages = %q, want notice + cut say output", got)
	}
	if got[0] != wantNotice {
		t.Fatalf("notice = %q, want %q", got[0], wantNotice)
	}
	if got[1] != wantSay {
		t.Fatalf("say output = %q, want %q", got[1], wantSay)
	}
}

func TestTerminalLineOver255EmitsExactlyOneNotice(t *testing.T) {
	s := filterTestSpeaker(t)
	if !s.TerminalLine("say " + strings.Repeat("x", 260)) {
		t.Fatal("TerminalLine closed session unexpectedly")
	}

	got := drainSessionText(t, s)
	wantNotice := "Line too long.  Truncated to:\r\nsay " + strings.Repeat("x", 251) + "\r\n"
	wantSay := "You say '" + strings.Repeat("x", 251) + "'\r\n"
	notices := 0
	sawSay := false
	for _, msg := range got {
		if msg == wantNotice {
			notices++
		}
		if msg == wantSay {
			sawSay = true
		}
	}
	if notices != 1 {
		t.Fatalf("notice count = %d, want 1 (messages=%q)", notices, got)
	}
	if !sawSay {
		t.Fatalf("cut say output %q not found in %q", wantSay, got)
	}
}
