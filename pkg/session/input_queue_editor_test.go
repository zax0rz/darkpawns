package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/boards"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// A real descriptor input path must route writing states before queue admission,
// even while lagged and with an ordinary command already pending (DP-1400).
func TestInputQueueCapLaggedEditorPaste(t *testing.T) {
	for _, kind := range []string{"editor", "board"} {
		t.Run(kind, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeCommandTestSession(t, m, "Builder", game.LVL_IMPL, 1001)
			closed := false
			s.SetCloseFunc(func() { closed = true })
			s.player.SetWaitState(3)
			if !s.tryExecuteNow("whoami", nil) {
				t.Fatal("ordinary input should queue while lagged")
			}
			var stored string
			if kind == "editor" {
				s.startLiveStringEditor(func(text string) { stored = text }, 8192)
			} else {
				m.world.Boards = boards.InitBoards(t.TempDir())
				s.player.WriteMagic = m.world.Boards.WriteMessage(0, s.player, "Paste proof")
				if s.player.WriteMagic == 0 {
					t.Fatal("board writer did not start")
				}
				s.player.SetPlrFlag(game.PlrWriting, true)
			}
			input := func(line string) {
				t.Helper()
				data, err := json.Marshal(CommandData{Command: line, RawLine: line})
				if err != nil {
					t.Fatal(err)
				}
				if err := s.handleCommand(data); err != nil {
					t.Fatalf("paste %q: %v", line, err)
				}
			}
			var expected strings.Builder
			for i := 0; i < 306; i++ {
				line := fmt.Sprintf("line%03d", i)
				input(line)
				expected.WriteString(line + "\r\n")
			}
			if closed {
				t.Fatal("lagged paste disconnected the builder")
			}
			if got := s.queueLen(); got != 1 {
				t.Fatalf("paste entered command queue: depth=%d, want original 1", got)
			}
			input("/s")
			if closed || s.queueLen() != 1 {
				t.Fatal("editor save entered capped command path")
			}
			if kind == "editor" {
				if s.isTextEditing() {
					t.Fatal("editor save was not consumed")
				}
				if stored != expected.String() {
					t.Fatalf("editor stored %d bytes, want all %d pasted bytes", len(stored), expected.Len())
				}
			} else {
				if s.player.WriteMagic != 0 || s.player.GetFlags()&(1<<game.PlrWriting) != 0 {
					t.Fatal("board save was not consumed")
				}
				reader := &queuePasteBoardReader{Player: s.player}
				if !m.world.Boards.DisplayMsg(0, reader, "1") {
					t.Fatal("saved board post cannot be read")
				}
				if !strings.Contains(reader.text.String(), expected.String()) {
					t.Fatalf("board did not retain all 306 lines: %q", reader.text.String())
				}
			}
		})
	}
}

type queuePasteBoardReader struct {
	*game.Player
	text strings.Builder
}

func (p *queuePasteBoardReader) SendMessage(text string) { p.text.WriteString(text) }
