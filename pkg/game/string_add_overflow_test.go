package game

import (
	"strings"
	"testing"
)

func requireNewStringAddNotice(t *testing.T, sent []string, previous int, want string) {
	t.Helper()
	if len(sent) != previous+1 || sent[previous] != want {
		t.Fatalf("new string_add notice = %q, want one message %q", sent[previous:], want)
	}
}

func TestNoteStringAddOverflowSkipsLineAndKeepsWriting(t *testing.T) {
	var sent []string
	player := NewPlayer(41, "NoteWriter", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	paper := &ObjectInstance{}
	noteWriteMu.Lock()
	noteWriteEntries[player.ID] = &noteWriteEntry{obj: paper}
	noteWriteMu.Unlock()
	t.Cleanup(func() {
		noteWriteMu.Lock()
		delete(noteWriteEntries, player.ID)
		noteWriteMu.Unlock()
	})
	player.SetPlrFlag(PlrWriting, true)

	line := strings.Repeat("n", 240)
	for range 4 {
		if HandleNoteInput(player, line) {
			t.Fatal("note editor exited while accepting a line")
		}
	}
	state := noteWriteEntries[player.ID]
	accepted := state.buffer

	previousMessages := len(sent)
	if HandleNoteInput(player, strings.Repeat("x", 30)) {
		t.Fatal("note editor exited after skipping an overflowing line")
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long.  Last line skipped.\r\n")
	if state.buffer != accepted {
		t.Fatalf("overflow changed accepted text: buffer=%d want buffer=%d", len(state.buffer), len(accepted))
	}
	if player.GetFlags()&(1<<PlrWriting) == 0 || noteWriteEntries[player.ID] != state {
		t.Fatal("note editor closed after overflow")
	}

	if HandleNoteInput(player, "ok") {
		t.Fatal("note editor exited while accepting a fitting line after overflow")
	}
	previousMessages = len(sent)
	if HandleNoteInput(player, strings.Repeat("y", 30)) {
		t.Fatal("note editor exited after a second skipped line")
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long.  Last line skipped.\r\n")
	if HandleNoteInput(player, "@") {
		// Completion is expected; the return value is true on @.
	} else {
		t.Fatal("@ did not complete the still-active note editor")
	}
	want := accepted + "\r\nok\r\n"
	if paper.Runtime.NoteText != want {
		t.Fatalf("saved note = %q, want %q", paper.Runtime.NoteText, want)
	}
}

func TestNoteFirstLineOverflowTruncatesAndKeepsWriting(t *testing.T) {
	var sent []string
	player := NewPlayer(42, "NoteWriter", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	paper := &ObjectInstance{}
	noteWriteMu.Lock()
	noteWriteEntries[player.ID] = &noteWriteEntry{obj: paper}
	noteWriteMu.Unlock()
	t.Cleanup(func() {
		noteWriteMu.Lock()
		delete(noteWriteEntries, player.ID)
		noteWriteMu.Unlock()
	})
	player.SetPlrFlag(PlrWriting, true)

	previousMessages := len(sent)
	if HandleNoteInput(player, strings.Repeat("x", maxNoteLength)) {
		t.Fatal("note editor exited after truncating the first line")
	}
	state := noteWriteEntries[player.ID]
	if len(state.buffer) != maxNoteLength-3 {
		t.Fatalf("first-line truncation buffer = %d bytes, want %d", len(state.buffer), maxNoteLength-3)
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long - Truncated.\r\n")
	if player.GetFlags()&(1<<PlrWriting) == 0 {
		t.Fatal("note editor closed after first-line truncation")
	}
}

func TestMailStringAddOverflowSkipsLineAndKeepsWriting(t *testing.T) {
	var sent []string
	player := NewPlayer(43, "MailWriter", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	mailWriteMu.Lock()
	mailWriteEntries[player.ID] = &mailWriteEntry{recipientID: 99}
	mailWriteMu.Unlock()
	t.Cleanup(func() { CancelMailWriting(player.ID) })
	player.SetPlrFlag(PlrWriting, true)
	player.SetPlrFlag(PlrMailing, true)

	line := strings.Repeat("m", 240)
	for range 15 {
		if HandleMailInput(player, line) {
			t.Fatal("mail editor exited while accepting a line")
		}
	}
	if HandleMailInput(player, strings.Repeat("a", 100)) {
		t.Fatal("mail editor exited while accepting a fitting line")
	}
	state := mailWriteEntries[player.ID]
	accepted := state.buffer
	previousMessages := len(sent)
	if HandleMailInput(player, strings.Repeat("x", 400)) {
		t.Fatal("mail editor exited after skipping an overflowing line")
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long.  Last line skipped.\r\n")
	if state.buffer != accepted {
		t.Fatalf("overflow changed accepted text: buffer=%d want buffer=%d", len(state.buffer), len(accepted))
	}
	if player.GetFlags()&(1<<PlrWriting) == 0 || player.GetFlags()&(1<<PlrMailing) == 0 || mailWriteEntries[player.ID] != state {
		t.Fatal("mail editor closed after overflow")
	}

	if HandleMailInput(player, "ok") {
		t.Fatal("mail editor exited while accepting a fitting line after overflow")
	}
	previousMessages = len(sent)
	if HandleMailInput(player, strings.Repeat("y", 400)) {
		t.Fatal("mail editor exited after a second skipped line")
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long.  Last line skipped.\r\n")
	if player.GetFlags()&(1<<PlrWriting) == 0 || player.GetFlags()&(1<<PlrMailing) == 0 {
		t.Fatal("mail editor closed after the second overflow")
	}
}

func TestMailFirstLineOverflowTruncatesAndKeepsWriting(t *testing.T) {
	var sent []string
	player := NewPlayer(44, "MailWriter", 1001)
	player.entryTextSink = func(msg string) { sent = append(sent, msg) }
	mailWriteMu.Lock()
	mailWriteEntries[player.ID] = &mailWriteEntry{recipientID: 99}
	mailWriteMu.Unlock()
	t.Cleanup(func() { CancelMailWriting(player.ID) })
	player.SetPlrFlag(PlrWriting, true)
	player.SetPlrFlag(PlrMailing, true)

	previousMessages := len(sent)
	if HandleMailInput(player, strings.Repeat("x", MailMaxSize)) {
		t.Fatal("mail editor exited after truncating the first line")
	}
	state := mailWriteEntries[player.ID]
	if len(state.buffer) != MailMaxSize-3 {
		t.Fatalf("first-line truncation buffer = %d bytes, want %d", len(state.buffer), MailMaxSize-3)
	}
	requireNewStringAddNotice(t, sent, previousMessages, "String too long - Truncated.\r\n")
	if player.GetFlags()&(1<<PlrWriting) == 0 || player.GetFlags()&(1<<PlrMailing) == 0 {
		t.Fatal("mail editor closed after first-line truncation")
	}
}
