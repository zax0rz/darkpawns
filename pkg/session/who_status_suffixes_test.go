package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhoStatusForSessionInvisAndEditorAndSwitchOrder(t *testing.T) {
	m := makeTestManager(t)
	immViewer := game.NewPlayer(1, "ImmViewer", 1001)
	immViewer.SetLevel(game.LVL_IMMORT)
	mortalViewer := game.NewPlayer(2, "MortalViewer", 1001)
	mortalViewer.SetLevel(10)

	target := game.NewPlayer(3, "Target", 1001)
	target.SetLevel(game.LVL_GOD)
	target.SetInvisLevel(35)
	target.SetAffect(game.AffInvisible, true)
	target.SetPlrFlag(game.PlrMailing, true)
	target.SetPlrFlag(game.PrfAFK, true)

	sess := &Session{
		manager:    m,
		player:     target,
		isSwitched: true,
		oedit:      &oeditState{},
		textEdit:   &textEditState{}, // active string edit inside oedit must not add (Text Edit)
	}

	// Immortal viewer sees (i35) (not (invis)) in C's exact order:
	// (i35) -> (mailing) -> (out of body) -> (Object Edit) -> (AFK)
	gotImm := whoStatusForSession(immViewer, target, sess, true)
	wantImm := " (i35) (mailing) (out of body) (Object Edit) (AFK)"
	if gotImm != wantImm {
		t.Fatalf("immortal viewer status = %q, want %q", gotImm, wantImm)
	}

	// Mortal viewer sees (invis) instead of (i35) (src/act.informative.c:1880-1883).
	gotMortal := whoStatusForSession(mortalViewer, target, sess, true)
	wantMortal := " (invis) (mailing) (out of body) (Object Edit) (AFK)"
	if gotMortal != wantMortal {
		t.Fatalf("mortal viewer status = %q, want %q", gotMortal, wantMortal)
	}

	// All six OLC/tedit editor states in C's exact order; playingEditor (do_string)
	// stays in CON_PLAYING and must not emit (Text Edit).
	editorCases := []struct {
		name  string
		setup func(*Session)
		want  string
	}{
		{name: "oedit", setup: func(s *Session) { s.oedit = &oeditState{} }, want: " (Object Edit)"},
		{name: "medit", setup: func(s *Session) { s.mobEdit = &meditState{} }, want: " (Mobile Edit)"},
		{name: "zedit", setup: func(s *Session) { s.zedit = &zeditState{} }, want: " (Zone Edit)"},
		{name: "sedit", setup: func(s *Session) { s.sedit = &seditState{} }, want: " (Shop Edit)"},
		{name: "redit", setup: func(s *Session) { s.roomEdit = &reditState{} }, want: " (Room Edit)"},
		{name: "tedit", setup: func(s *Session) { s.textEdit = &textEditState{} }, want: " (Text Edit)"},
		{name: "do_string", setup: func(s *Session) { s.textEdit = &textEditState{playingEditor: true} }, want: ""},
	}
	plain := game.NewPlayer(4, "Plain", 1001)
	plain.SetLevel(game.LVL_IMPL)
	for _, tc := range editorCases {
		t.Run(tc.name, func(t *testing.T) {
			es := &Session{manager: m, player: plain}
			tc.setup(es)
			if got := whoStatusForSession(immViewer, plain, es, false); got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCmdWhoListsOriginalWithOutOfBodyWhenPCSwitched(t *testing.T) {
	m, sw, h, original := switchedPCFixture(t)
	if original.GetLevel() < game.LVL_IMPL {
		original.SetLevel(game.LVL_IMPL)
	}
	// In production, a linkless target has no active descriptor in m.sessions.
	m.mu.Lock()
	for k, sess := range m.sessions {
		if sess == h {
			delete(m.sessions, k)
		}
	}
	m.mu.Unlock()

	viewer := makeCommandTestSession(t, m, "Viewer", game.LVL_IMMORT, 1001)
	game.SetTitle(viewer.player, "the Watcher")

	if err := cmdWho(viewer, nil); err != nil {
		t.Fatalf("cmdWho failed: %v", err)
	}
	out := renderedOutput(viewer)
	if !strings.Contains(out, original.GetName()) || !strings.Contains(out, "(out of body)") {
		t.Fatalf("who output missing original or (out of body):\n%s", out)
	}
	if strings.Contains(out, h.player.GetName()) {
		t.Fatalf("who output leaked borrowed body %q:\n%s", h.player.GetName(), out)
	}
	_ = sw
}
