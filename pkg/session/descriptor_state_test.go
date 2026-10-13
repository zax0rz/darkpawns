package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

type conPlayingFixture struct {
	manager     *Manager
	actor       *Session
	playing     *Session
	mailing     *Session
	noteWrite   *Session
	menu        *Session
	creating    *Session
	editing     *Session
	textEditing *Session
	linkless    *Session
}

func newConPlayingFixture(t *testing.T) conPlayingFixture {
	t.Helper()
	m := newReditTestManager(t)
	m.world.LibTextDir = t.TempDir()
	nextPlayerID := 1
	newSession := func(name string, level int) *Session {
		s := makeCommandTestSession(t, m, name, level, 1001)
		s.player.ID = nextPlayerID
		nextPlayerID++
		s.player.Stats.Int, s.player.Stats.Wis = 10, 10
		s.player.CopyBaseAttributes()
		registerTestSession(t, m, s, name)
		return s
	}

	f := conPlayingFixture{
		manager:     m,
		actor:       newSession("Actor", game.LVL_GOD),
		playing:     newSession("Playing", 10),
		mailing:     newSession("Mailing", 10),
		noteWrite:   newSession("NoteWrite", 10),
		menu:        newSession("Menu", 10),
		creating:    newSession("Creating", 10),
		editing:     newSession("Editing", game.LVL_IMPL),
		textEditing: newSession("TextEditing", game.LVL_GRGOD),
		linkless:    newSession("Linkless", 10),
	}
	f.mailing.player.SetPlrFlag(game.PlrWriting, true)
	f.mailing.player.SetPlrFlag(game.PlrMailing, true)
	f.noteWrite.player.SetPlrFlag(game.PlrWriting, true)
	f.menu.showMainMenu()
	f.creating.startNewCharFlow("Creating")
	if err := ExecuteCommand(f.editing, "redit", nil); err != nil {
		t.Fatalf("enter room editor: %v", err)
	}
	if err := ExecuteCommand(f.textEditing, "tedit", []string{"news"}); err != nil {
		t.Fatalf("enter text editor: %v", err)
	}
	if !f.editing.isRoomEditing() {
		t.Fatal("test setup did not enter the real REDIT editor")
	}
	// OLC entry emits the ordinary room emote. Drain setup output before
	// measuring the descriptor-gated command below.
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		drainSessionText(t, s)
	}
	f.linkless.DetachTransport()
	return f
}

func TestConPlayingEditorMarkersFollowEditorTransitions(t *testing.T) {
	f := newConPlayingFixture(t)
	if f.editing.isConPlaying() || f.textEditing.isConPlaying() {
		t.Fatal("active REDIT/TEDIT descriptors counted as CON_PLAYING")
	}
	f.editing.handleReditInput("q")
	f.textEditing.cancelTextEdit()
	if !f.editing.isConPlaying() || !f.textEditing.isConPlaying() {
		t.Fatal("closed editor states did not return to CON_PLAYING")
	}
}
