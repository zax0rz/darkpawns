package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// setupNoteWriter registers a player in note-write mode targeting paper and
// returns the player's sent-message sink.
func setupNoteWriter(t *testing.T, id int, paper *ObjectInstance) (*Player, *[]string) {
	t.Helper()
	var sent []string
	player := NewPlayer(id, "NoteWriter", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	noteWriteMu.Lock()
	noteWriteEntries[player.ID] = &noteWriteEntry{obj: paper}
	noteWriteMu.Unlock()
	t.Cleanup(func() {
		noteWriteMu.Lock()
		delete(noteWriteEntries, player.ID)
		noteWriteMu.Unlock()
	})
	player.SetPlrFlag(PlrWriting, true)
	return player, &sent
}

// TestNoteCommitIsSilentAndReadable pins C's do_write/string_add contract for
// notes (DP-1522): '@' prints nothing — playing_string_cleanup has no note
// branch (src/modify.c:218-236) — and the one per-object field carries the
// text with C's per-line CRLF. Fails if the invented "Note recorded." or the
// blank-read regression returns (R1/R4/R5h).
func TestNoteCommitIsSilentAndReadable(t *testing.T) {
	paper := &ObjectInstance{}
	player, sent := setupNoteWriter(t, 51, paper)

	if HandleNoteInput(player, "the avian sun rises") {
		t.Fatal("note editor exited on a text line")
	}
	if !HandleNoteInput(player, "@") {
		t.Fatal("@ did not complete the note editor")
	}
	if len(*sent) != 0 {
		t.Fatalf("note commit printed %q, want C's silence", *sent)
	}
	const want = "the avian sun rises\r\n"
	if got := paper.Runtime.NoteText; got != want {
		t.Fatalf("stored note = %q, want %q", got, want)
	}
	if got := paper.ActionDescription(); got != want {
		t.Fatalf("ActionDescription() = %q, want %q", got, want)
	}
	if player.GetFlags()&(1<<PlrWriting) != 0 {
		t.Fatal("PLR_WRITING still set after @")
	}
}

// TestNoteInstancesDoNotShareText pins that writing one copy leaves another
// copy of the same note blank: C gives each instance its own buffer on the
// first string_add line, so the text never lands in the shared prototype.
func TestNoteInstancesDoNotShareText(t *testing.T) {
	proto := &parser.Obj{VNum: 19510, Keywords: "tome", TypeFlag: ITEM_NOTE}
	first := &ObjectInstance{Prototype: proto}
	second := &ObjectInstance{Prototype: proto}

	player, _ := setupNoteWriter(t, 53, first)
	if HandleNoteInput(player, "only on the first copy") {
		t.Fatal("note editor exited on a text line")
	}
	if !HandleNoteInput(player, "@") {
		t.Fatal("@ did not complete the note editor")
	}
	if got := first.ActionDescription(); got != "only on the first copy\r\n" {
		t.Fatalf("first copy = %q, want the written text", got)
	}
	if got := second.ActionDescription(); got != "" {
		t.Fatalf("second copy shares text %q, want blank", got)
	}
}

// TestEmptyNoteStaysBlank pins string_add's '@' path with no lines: C never
// allocates the buffer, action_description stays NULL, and nothing is
// announced — no "Note discarded." exists in src/ (R4).
func TestEmptyNoteStaysBlank(t *testing.T) {
	paper := &ObjectInstance{}
	player, sent := setupNoteWriter(t, 54, paper)

	if !HandleNoteInput(player, "@") {
		t.Fatal("@ did not complete the empty note")
	}
	if len(*sent) != 0 {
		t.Fatalf("empty note commit printed %q, want C's silence", *sent)
	}
	if got := paper.ActionDescription(); got != "" {
		t.Fatalf("empty note stored %q, want blank", got)
	}
}

// TestNoteGateReadsWrittenTextAndPrototypeText pins the predicate do_write's
// "already written" gate reads (src/act.comm.c:1094): non-empty after a write
// (second write must refuse) and non-empty when the prototype carries an
// action description, C's non-NULL fresh-instance pointer.
func TestNoteGateReadsWrittenTextAndPrototypeText(t *testing.T) {
	proto := &parser.Obj{VNum: 19511, Keywords: "tablet", TypeFlag: ITEM_NOTE, ActionDesc: "an inscription"}
	fresh := &ObjectInstance{Prototype: proto}
	if got := fresh.ActionDescription(); got != "an inscription" {
		t.Fatalf("fresh instance text = %q, want prototype's", got)
	}

	written := &ObjectInstance{Prototype: &parser.Obj{VNum: 19510, Keywords: "tome", TypeFlag: ITEM_NOTE}}
	player, _ := setupNoteWriter(t, 55, written)
	if HandleNoteInput(player, "draft") {
		t.Fatal("note editor exited on a text line")
	}
	if !HandleNoteInput(player, "@") {
		t.Fatal("@ did not complete the note editor")
	}
	if written.ActionDescription() == "" {
		t.Fatal("written note reports blank action description; second write would be allowed")
	}
}
