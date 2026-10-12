package game

// missing_blank_test.go — PF-063 cluster 5 proofs for the game side: C
// composes these act strings with their own trailing "\r\n" and act()
// appends the second — the blank line the audience sees before the prompt.
// whisper/ask TO_CHAR: act.comm.c:1010-1012; guild_guard's room line:
// spec_procs.c:580; the conjured vanish line: spec_procs3.c:878-882. Exact
// sink bytes; reverting any fix fails (mutation checks).

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newMissingBlankWorld builds a two-message world whose sink captures every
// player's bytes by name.
func newMissingBlankWorld(t *testing.T) (*World, map[string]*string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Study"}},
		Mobs: []parser.Mob{{
			VNum: 2001, Keywords: "guard", ShortDesc: "the guild guard",
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	captured := make(map[string]*string)
	w.MessageSink = func(name string, msg []byte) {
		if captured[name] == nil {
			captured[name] = new(string)
		}
		*captured[name] += string(msg)
	}
	return w, captured
}

// TestWhisperAskSelfReplyCarriesOwnBreak — do_spec_comm's TO_CHAR reply is
// sprintf'd with its own \r\n and act() appends the second (act.comm.c:1010-
// 1012). Covers both verbs.
func TestWhisperAskSelfReplyCarriesOwnBreak(t *testing.T) {
	cases := []struct {
		ask  bool
		want string
	}{
		{false, "You whisper to Watcher, 'quiet words'\r\n\r\n"},
		{true, "You ask Watcher, 'why'\r\n\r\n"},
	}
	for _, tc := range cases {
		w, captured := newMissingBlankWorld(t)
		actor := NewPlayer(1, "Actor", 1001)
		if err := w.AddPlayer(actor); err != nil {
			t.Fatalf("AddPlayer actor: %v", err)
		}
		watcher := NewPlayer(2, "Watcher", 1001)
		if err := w.AddPlayer(watcher); err != nil {
			t.Fatalf("AddPlayer watcher: %v", err)
		}

		if tc.ask {
			w.DoSpecComm(actor, "Watcher why", true)
		} else {
			w.DoSpecComm(actor, "Watcher quiet words", false)
		}
		if got := *captured["Actor"]; got != tc.want {
			t.Fatalf("ask=%v actor bytes = %q, want %q (string's break plus act()'s append)", tc.ask, got, tc.want)
		}
	}
}

// TestGuildGuardRoomActCarriesOwnBreak — spec_procs.c:580's act format ends
// with \r\n; act() appends the second.
func TestGuildGuardRoomActCarriesOwnBreak(t *testing.T) {
	w, captured := newMissingBlankWorld(t)
	mob, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	actor := NewPlayer(1, "Actor", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}

	if !specGuildGuard(w, actor, mob, "flee", "") {
		t.Fatal("guild guard did not intercept flee")
	}
	want := "Actor tries to flee inside the guild but the guard block his way!\r\n\r\n"
	if got := *captured["Watcher"]; got != want {
		t.Fatalf("watcher bytes = %q, want %q (act's own break plus act()'s append)", got, want)
	}
}

// TestConjuredVanishActCarriesOwnBreak — spec_procs3.c:878-882: the say
// line is single-terminated, the vanish line is sprintf'd with its own
// \r\n and act() appends the second.
func TestConjuredVanishActCarriesOwnBreak(t *testing.T) {
	w, captured := newMissingBlankWorld(t)
	mob, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}

	if !specConjured(w, nil, mob, "vanish-probe", "") {
		t.Fatal("conjured spec did not run its vanish arm")
	}
	want := "The guild guard states, 'My work here is done.'\r\n" +
		"The guild guard disappears in a flash of white light!\r\n\r\n"
	if got := *captured["Watcher"]; got != want {
		t.Fatalf("watcher bytes = %q, want %q (say line single, vanish line doubled)", got, want)
	}
}
