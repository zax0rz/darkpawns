// boards.go — Thin pkg/game wrappers for the extracted pkg/boards system.
//
// The BoardSystem implementation lives in pkg/boards/. This file keeps the
// pieces that cannot move without creating a circular dependency:
//   - World.FindBoard (needs []*ObjectInstance from GetItemsInRoom)
//   - World.GetOrInitBoards (constructs the board system during boot)
//   - genBoard spec proc (must conform to the pkg/game SpecFunc signature)

package game

import (
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/boards"
)

// FindBoard searches the player's current room for a board object and returns
// the board type index. It stays on *World because GetItemsInRoom returns
// []*ObjectInstance, which cannot be expressed through an interface slice.
func (w *World) FindBoard(ch *Player) int {
	if w.Boards == nil {
		return -1
	}
	roomVNum := ch.GetRoomVNum()
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		for i := 0; i < boards.NumBoards; i++ {
			if obj.VNum == w.Boards.BoardInfo(i).VNum {
				return i
			}
		}
	}
	return -1
}

// GetOrInitBoards ensures the board system is initialized.
func (w *World) GetOrInitBoards(basePath string) *boards.BoardSystem {
	if w.Boards == nil {
		w.Boards = boards.InitBoards(basePath)
	}
	return w.Boards
}

// genBoard is the spec procedure for bulletin board objects.
// It intercepts read/write/remove/look/examine commands when a board is present.
func genBoard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	_ = me // board objects don't have mob instances, but spec signature requires it
	if w.Boards == nil {
		return false
	}
	if ch == nil {
		return false
	}

	boardType := w.FindBoard(ch)
	if boardType == -1 {
		slog.Error("SYSERR: degenerate board!")
		return false
	}

	switch cmd {
	case "write":
		magic := w.Boards.WriteMessage(boardType, ch, arg)
		if magic > 0 {
			// C's Board_write_message enters the ordinary string editor. Keep the
			// board write distinguishable from note/mail editing while exposing
			// the same PLR_WRITING communication gate.
			ch.WriteMagic = magic
			ch.SetPlrFlag(PlrWriting, true)
		}
		return true
	case "look", "examine":
		fields := strings.Fields(arg)
		if len(fields) == 0 || (!strings.EqualFold(fields[0], "board") && !strings.EqualFold(fields[0], "bulletin")) {
			return false
		}
		return w.Boards.ShowBoard(boardType, ch)
	case "read":
		fields := strings.Fields(arg)
		if len(fields) == 1 && (strings.EqualFold(fields[0], "board") || strings.EqualFold(fields[0], "bulletin")) {
			return w.Boards.ShowBoard(boardType, ch)
		}
		return w.Boards.DisplayMsg(boardType, ch, arg)
	case "remove":
		return w.Boards.RemoveMsg(boardType, ch, arg)
	}

	return false
}

func init() {
	// Register gen_board as an object spec procedure
	RegisterSpec("gen_board", genBoard)
}
