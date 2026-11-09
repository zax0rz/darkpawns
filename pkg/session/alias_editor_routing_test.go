package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/boards"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1516 item 9: C's perform_complex_alias leaves the expansion on the
// descriptor input queue (interpreter.c:1055-1097), so a trailing command is
// consumed by an editor an earlier command opened (comm.c:614-627). Go
// dispatched every expanded command inline, so the trailing lines ran as
// commands ("Huh?!?") instead of reaching string_add. Here the alias's
// "body"/"@" lines must reach the active board editor: "body" appends the
// message text and "@" saves it.
func TestComplexAliasRoutesTrailingLinesToEditor(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Poster", game.LVL_IMPL, 1001)
	m.world.Boards = boards.InitBoards(t.TempDir())

	magic := m.world.Boards.WriteMessage(0, s.player, "headline")
	if magic == 0 {
		t.Fatal("board writer did not start")
	}
	s.player.WriteMagic = magic
	s.player.SetPlrFlag(game.PlrWriting, true)

	s.player.Aliases = []game.Alias{{Alias: "p", Replacement: " body;@", Type: game.AliasComplex}}

	if err := ExecuteCommand(s, "p", nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if s.player.WriteMagic != 0 || s.player.GetFlags()&(1<<game.PlrWriting) != 0 {
		t.Fatal("board post was not consumed by the descriptor editor")
	}
	reader := &queuePasteBoardReader{Player: s.player}
	if !m.world.Boards.DisplayMsg(0, reader, "1") {
		t.Fatal("saved board post cannot be read")
	}
	if !strings.Contains(reader.text.String(), "body") {
		t.Fatalf("board post missing editor body: %q", reader.text.String())
	}
}
