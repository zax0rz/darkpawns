package game

// note_write.go — note-writing state machine for ITEM_NOTE objects.
// Ports C's do_write + string_add for notes: the text lands on the object's
// own action_description (Runtime.NoteText; see ObjectInstance.ActionDescription)
// and the editor commits silently — C's playing_string_cleanup has no note
// branch, so '@' prints nothing (src/modify.c:218-236; witnessed in
// deepseek-dp1502-note-text-gap).
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
	obj         *ObjectInstance
	buffer      string
	editorBytes int
	overflowed  bool
}

// HandleNoteInput processes one line of input from a player in note-write mode.
// Returns true when writing is complete (line == "@"), false while buffering.
// Clears PLR_WRITING on completion or error.
// Called from session_login.go PLR_WRITING intercept when PLR_MAILING is unset.
//
// '@' commits with no message: for a note, C's string_add takes the
// STRINGADD_SAVE path through playing_string_cleanup, which only handles mail
// and board posts (src/modify.c:127-156,218-236). An empty note leaves C's
// action_description NULL, so the object simply stays blank — there is no
// "discarded" message anywhere in src/. Accepted lines are stored each with a
// trailing CRLF, exactly as string_add appends "\r\n" after every line
// (src/modify.c:216-217).
func HandleNoteInput(ch *Player, line string) bool {
	if line == "@" {
		noteWriteMu.Lock()
		state, ok := noteWriteEntries[ch.ID]
		delete(noteWriteEntries, ch.ID)
		noteWriteMu.Unlock()

		ch.SetPlrFlag(PlrWriting, false)

		if ok && state.buffer != "" {
			text := state.buffer
			if text[len(text)-1] != '\n' {
				// First-line truncation stores the raw prefix; C's SAVE path
				// still leaves the "\r\n" terminator on the stored string.
				text += "\r\n"
			}
			state.obj.Runtime.NoteText = text
		}
		// With no entry or no text, C leaves paper->action_description NULL:
		// nothing is stored and nothing is printed.
		return true
	}

	noteWriteMu.Lock()
	state, ok := noteWriteEntries[ch.ID]
	if !ok {
		noteWriteMu.Unlock()
		ch.SetPlrFlag(PlrWriting, false)
		return true
	}
	if len(line)+state.editorBytes+3 > maxNoteLength {
		notice := "String too long.  Last line skipped.\r\n"
		if state.editorBytes == 0 {
			state.buffer = line[:maxNoteLength-3]
			state.editorBytes = maxNoteLength - 1 // truncated text plus C's CRLF
			notice = "String too long - Truncated.\r\n"
		}
		state.overflowed = true
		noteWriteMu.Unlock()
		ch.SendMessage(notice)
		return false
	}
	state.buffer += line + "\r\n"
	state.editorBytes += len(line) + 2 // string_add stores every accepted line with CRLF.
	noteWriteMu.Unlock()
	return false
}
