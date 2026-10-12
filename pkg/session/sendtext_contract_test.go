package session

// sendtext_contract_test.go — the MsgText render contract (PR #1923 review):
// C's send_to_char writes the string verbatim (src/comm.c:2335 →
// write_to_output :1348) and process_output adds exactly one noncompact
// frame (:1630-1632). A terminated value through sendText must render
// [message, blank, prompt] — the old unconditional append printed three
// CRLFs where C prints two. A bare value still gains exactly one line end.
// Reverting the render to text + "\r\n" fails the terminated rows
// (mutation check).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestSendTextContractTerminatedAndBareValues drives sendText → SendPrompt
// → RenderTerminalFrame in both compact modes for both value shapes.
func TestSendTextContractTerminatedAndBareValues(t *testing.T) {
	cases := []struct {
		name        string
		value       string
		wantFull    string
		wantCompact string
	}{
		{
			name:        "terminated value renders verbatim plus one frame",
			value:       "terminated value\r\n",
			wantFull:    "terminated value\r\n\r\n> ",
			wantCompact: "terminated value\r\n> ",
		},
		{
			name:        "bare value gains exactly one line end",
			value:       "bare value",
			wantFull:    "bare value\r\n\r\n> ",
			wantCompact: "bare value\r\n> ",
		},
	}
	for _, tc := range cases {
		for _, compact := range []bool{false, true} {
			m := &Manager{}
			s := m.NewSession()
			s.player = game.NewPlayer(1, "Tester", 1001)
			if compact {
				s.player.Flags |= 1 << uint(game.PrfCompact)
			}
			s.sendText(tc.value)
			s.SendPrompt()
			want := tc.wantFull
			if compact {
				want = tc.wantCompact
			}
			var got strings.Builder
			for len(s.send) > 0 {
				frame, ok := RenderTerminalFrame(<-s.send)
				if ok {
					got.WriteString(frame.Text)
				}
			}
			if got.String() != want {
				t.Errorf("%s (compact=%v): wire = %q, want %q", tc.name, compact, got.String(), want)
			}
		}
	}
}

// TestCmdWhereTerminatedListRendersOneFrame — the real caller from the
// review: cmdWhere's argument arm builds a \r\n-terminated list and sends it
// through sendText; the rendered wire is the list verbatim, one blank, the
// prompt.
func TestCmdWhereTerminatedListRendersOneFrame(t *testing.T) {
	for _, compact := range []bool{false, true} {
		w, s := newBareSendItemHarness(t)
		setCompact(s, compact)
		// A second registered session gives the argument arm one match.
		other := makeTestSession(t, s.manager, "Quaffertoo", 1001, true)
		s.manager.mu.Lock()
		s.manager.sessions["quaffertoo"] = other
		s.manager.mu.Unlock()
		if err := w.AddPlayer(other.player); err != nil {
			t.Fatalf("AddPlayer other: %v", err)
		}

		if err := cmdWhere(s, []string{"Qua"}); err != nil {
			t.Fatalf("cmdWhere: %v", err)
		}
		s.SendPrompt()
		got := renderAllOutput(s)
		// Both sessions match "Qua"; the most recently connected sorts
		// first (C iterates descriptor_list, newest prepended).
		wantRows := "M  1. Quaffertoo                - [ 1001]Study\r\n" +
			"M  2. Quaffer                   - [ 1001]Study\r\n"
		if compact {
			if got != wantRows+"> " {
				t.Fatalf("compact where wire = %q, want %q", got, wantRows+"> ")
			}
			continue
		}
		if got != wantRows+"\r\n> " {
			t.Fatalf("where wire = %q, want %q", got, wantRows+"\r\n> ")
		}
	}
}
