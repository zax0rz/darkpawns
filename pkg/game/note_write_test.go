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
	return player, &sent
}

// TestNoteCommitIsSilentAndReadable pins C's do_write/string_add contract for
// notes (DP-1522): '@' prints nothing — playing_string_cleanup has no note
// branch (src/modify.c:218-236) — and do_write targets the note's own
// action_description (src/act.comm.c:1089-1102); string_add stores its lines
// with CRLF (src/modify.c:132-155). Fails if "Note recorded." or blank reads
// return (R1/R4/R5h).
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
	if IsWritingNote(player) {
		t.Fatal("note editor still active after @")
	}
	if player.GetFlags()&(1<<PlrWriting) != 0 {
		t.Fatal("PLR_WRITING set during/after paper write")
	}
}

// TestNoteInstancesDoNotShareText pins that writing one copy leaves another
// copy of the same note blank: read_object copies the prototype into each
// instance (src/db.c:1947-1953), and C stores action_description on that
// instance (src/structs.h:743-754; src/modify.c:132-144).
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

// TestNoteGateReadsWrittenTextAndPrototypeText pins do_write's "already
// written" refusal (src/act.comm.c:1089-1095): a prototype description is
// copied to a fresh object, and text committed to one instance also blocks a
// later write to that instance.
func TestNoteGateReadsWrittenTextAndPrototypeText(t *testing.T) {
	w, player, lastMsg := newDonateTestWorld(t)
	penProto := &parser.Obj{VNum: 19512, Keywords: "pen", ShortDesc: "a pen", TypeFlag: ITEM_PEN}
	pen := w.NewObjectFromProto(penProto, roomNowhere)
	if err := w.MoveObjectToPlayerInventory(pen, player); err != nil {
		t.Fatal(err)
	}

	prototypeNote := w.NewObjectFromProto(&parser.Obj{
		VNum: 19511, Keywords: "tablet", ShortDesc: "a tablet", TypeFlag: ITEM_NOTE, ActionDesc: "an inscription",
	}, roomNowhere)
	if err := w.MoveObjectToPlayerInventory(prototypeNote, player); err != nil {
		t.Fatal(err)
	}
	w.DoWrite(player, "tablet pen")
	if got, want := lastMsg(), "There's something written on it already.\r\n"; got != want {
		t.Fatalf("prototype text write gate = %q, want %q", got, want)
	}
	if IsWritingNote(player) || player.GetFlags()&(1<<PlrWriting) != 0 {
		t.Fatal("prototype text entered the note editor")
	}

	written := w.NewObjectFromProto(&parser.Obj{
		VNum: 19510, Keywords: "tome", ShortDesc: "a tome", TypeFlag: ITEM_NOTE,
	}, roomNowhere)
	if err := w.MoveObjectToPlayerInventory(written, player); err != nil {
		t.Fatal(err)
	}
	w.DoWrite(player, "tome pen")
	if got, want := lastMsg(), "Write your note.  End with '@' on a new line.\r\n"; got != want {
		t.Fatalf("initial note write = %q, want %q", got, want)
	}
	if !IsWritingNote(player) {
		t.Fatal("paper write did not enter the note editor")
	}
	if player.GetFlags()&(1<<PlrWriting) != 0 {
		t.Fatal("paper write must not set PLR_WRITING")
	}
	if HandleNoteInput(player, "draft") {
		t.Fatal("note editor exited on a text line")
	}
	if !HandleNoteInput(player, "@") {
		t.Fatal("@ did not complete the note editor")
	}
	lastMsg()
	w.DoWrite(player, "tome pen")
	if got, want := lastMsg(), "There's something written on it already.\r\n"; got != want {
		t.Fatalf("written text write gate = %q, want %q", got, want)
	}
	if IsWritingNote(player) || player.GetFlags()&(1<<PlrWriting) != 0 {
		t.Fatal("existing text entered a second note editor")
	}
}
