package game

// note_write.go — note-writing state machine for ITEM_NOTE objects.
// Mirrors the mail writing subsystem (mail.go) but stores text directly
// on the object's Runtime.NoteText rather than writing to a mail file.
//
// Source: act.comm.c do_write() — ch->desc->str = &paper->action_description
// In C, string_add() appended lines until '@' was entered alone on a line.
// Go equivalent: HandleNoteInput() buffers lines and commits on '@'.
//
// Session integration: session_login.go PLR_WRITING intercept routes here
// when PLR_MAILING is NOT set (note write, not mail write).

import (
	"sync"
)

// maxNoteLength is declared in act_comm.go — do not redeclare here.

var (
	noteWriteMu      sync.Mutex
	noteWriteEntries = make(map[int]*noteWriteEntry)
)

// noteWriteEntry holds the in-progress note text and the target object.
type noteWriteEntry struct {
	obj    *ObjectInstance
	buffer string
}

// StartNoteWriting registers the player as actively writing on obj.
// Sets PLR_WRITING so the session intercept routes input here.
// Called from doWrite after all validation passes.
func StartNoteWriting(ch *Player, obj *ObjectInstance) {
	noteWriteMu.Lock()
	noteWriteEntries[ch.ID] = &noteWriteEntry{obj: obj}
	noteWriteMu.Unlock()

	ch.SetPlrFlag(PlrWriting, true)
	// PLR_MAILING is intentionally NOT set — that's the discriminator in the
	// session intercept to tell note writes from mail writes.
}

// HandleNoteInput processes one line of input from a player in note-write mode.
// Returns true when writing is complete (line == "@"), false while buffering.
// Clears PLR_WRITING on completion or error.
// Called from session_login.go PLR_WRITING intercept when PLR_MAILING is unset.
func HandleNoteInput(ch *Player, line string) bool {
	if line == "@" {
		noteWriteMu.Lock()
		state, ok := noteWriteEntries[ch.ID]
		delete(noteWriteEntries, ch.ID)
		noteWriteMu.Unlock()

		ch.SetPlrFlag(PlrWriting, false)

		if !ok {
			sendToChar(ch, "Your note was lost. (internal error)\r\n")
			return true
		}
		if state.buffer == "" {
			sendToChar(ch, "You have written nothing. Note discarded.\r\n")
			return true
		}

		state.obj.Runtime.NoteText = state.buffer
		sendToChar(ch, "Note recorded.\r\n")
		return true
	}

	noteWriteMu.Lock()
	state, ok := noteWriteEntries[ch.ID]
	if !ok {
		noteWriteMu.Unlock()
		ch.SetPlrFlag(PlrWriting, false)
		return true
	}
	if state.buffer != "" {
		state.buffer += "\r\n"
	}
	state.buffer += line
	if len(state.buffer) > maxNoteLength {
		state.buffer = state.buffer[:maxNoteLength]
		// Notify and flush — C does this silently but a message is friendlier
		noteWriteMu.Unlock()
		sendToChar(ch, "Note limit reached. Type '@' on a new line to save.\r\n")
		return false
	}
	noteWriteMu.Unlock()
	return false
}

// CancelNoteWriting cleans up note writing state for a player (e.g. on disconnect).
func CancelNoteWriting(playerID int) {
	noteWriteMu.Lock()
	delete(noteWriteEntries, playerID)
	noteWriteMu.Unlock()
}

// IsWritingNote returns true if the player is in note-write mode (not mail mode).
func IsWritingNote(ch *Player) bool {
	if ch.GetFlags()&(1<<PlrWriting) == 0 {
		return false
	}
	if ch.GetFlags()&(1<<PlrMailing) != 0 {
		return false // that's mail
	}
	return true
}
