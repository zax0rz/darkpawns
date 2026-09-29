package game

import (
	"fmt"
	"log/slog"
)

// ---------------------------------------------------------------------------
// do_save
// ---------------------------------------------------------------------------

func (w *World) doSave(ch *Player, me *MobInstance, cmd string, arg string) bool {
	// C do_save returns for NPCs and link-dead characters before any save
	// or message (act.other.c:188: !ch->desc).
	if isPlayerNPC(ch, me) || ch.IsLinkless() {
		return true
	}

	jsonErr := SavePlayer(ch)
	// The store of record is what login reads, so the save command must
	// write it: C's do_save is save_char(ch, NOWHERE) + Crash_crashsave
	// (act.other.c:198-199). The JSON file stays for the wizard paths that
	// read it (DP-1359); neither write substitutes for the other, and a
	// JSON failure must not prevent the store write.
	if res := w.SavePlayerRecord(ch, "save", LoadRoomNowhere, SaveCrash); res == SaveFailed {
		slog.Error("store-of-record save failed for save command", "player", ch.Name)
	}
	if jsonErr != nil {
		ch.SendMessage("Could not save your data. Contact an admin!\r\n")
		return true
	}

	// C do_save (act.other.c): "Saving %s.\r\n" with GET_NAME(ch).
	ch.SendMessage(fmt.Sprintf("Saving %s.\r\n", ch.Name))
	return true
}
