package session

import (
	"bytes"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func fileEditorLogFixture(t *testing.T) (*Manager, *Session, *Session, *Session, *Session, *bytes.Buffer) {
	t.Helper()
	m := makeTestManager(t)
	m.world.ScriptsDir = t.TempDir()
	m.world.LibTextDir = t.TempDir()
	a := makeCommandTestSession(t, m, "Fileactor", 40, 1001)
	a.player.SetInvisLevel(40)
	a.player.SetPlrFlag(game.PrfLog1, true)
	a.player.SetPlrFlag(game.PrfLog2, true)
	watch := makeCommandTestSession(t, m, "Filewatch", 40, 1002)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	below := makeCommandTestSession(t, m, "Filebelow", 33, 1002)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	normal := makeCommandTestSession(t, m, "Filenormal", 40, 1002)
	normal.player.SetPlrFlag(game.PrfLog2, true)
	for _, s := range []*Session{a, watch, below, normal} {
		registerTestSession(t, m, s, s.player.Name)
	}
	file := captureMudlogFile(t)
	game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: "", onError: func() {
		if a.textEdit == nil || a.player.GetFlags()&(1<<game.PlrWriting) == 0 || len(a.send) != 0 {
			t.Error("file producer must precede ack and common cleanup")
		}
	}})
	return m, a, watch, below, normal, file
}
