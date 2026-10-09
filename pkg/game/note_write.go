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
			ch.SendMessage("Your note was lost. (internal error)\r\n")
			return true
		}
		if state.buffer == "" {
			ch.SendMessage("You have written nothing. Note discarded.\r\n")
			return true
		}

		state.obj.Runtime.NoteText = state.buffer
		ch.SendMessage("Note recorded.\r\n")
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
		ch.SendMessage("Note limit reached. Type '@' on a new line to save.\r\n")
		return false
	}
	noteWriteMu.Unlock()
	return false
}
