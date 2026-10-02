package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// CON_EXDESC: src/interpreter.c:2247-2266; src/modify.c:155-210,240-249;
// EXDSCR_LENGTH src/structs.h:652. Exercise the actual menu/terminal routing.
func TestEntryDescriptionEditor(t *testing.T) {
	for _, stored := range []bool{false, true} {
		t.Run(map[bool]string{false: "creation", true: "returning"}[stored], func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charName = "Scribe"
			if stored {
				s.player = game.NewPlayer(7, "Scribe", 1001)
				s.player.Description = "Old text.\r\n"
			} else {
				s.menuDescription = "Old text.\r\n"
			}
			s.terminalNamed = true
			s.showMainMenu()
			renderedOutput(s)
			s.TerminalLine("2")
			want := "Current description:\r\nOld text.\r\nEnter the new text you'd like others to see when they look at you.\r\nInstructions: /s or @ to save, /h for more options.\r\n] "
			if got := renderedOutput(s); got != want {
				t.Fatalf("entry bytes=%q want=%q", got, want)
			}
			if !s.IsTextEditing() || s.menuStage != "description" {
				t.Fatal("CON_EXDESC editor not installed")
			}
			if editing, playing := s.editorPromptState(); !editing || playing {
				t.Fatal("entry editor incorrectly marked CON_PLAYING")
			}
			s.TerminalLine("Added $$line~")
			renderedOutput(s)
			value := func() string {
				if s.player != nil {
					return s.player.Description
				}
				return s.menuDescription
			}
			if value() != "Old text.\r\nAdded $$line \r\n" {
				t.Fatal("editor did not write through", value())
			}
			s.TerminalLine("/n")
			if got := renderedOutput(s); !strings.Contains(got, "Old text.") || !strings.Contains(got, "Added $$line ") {
				t.Fatal("numbered list did not read current buffer", got)
			}
			s.TerminalLine("/a")
			if got := renderedOutput(s); got != "Description aborted.\r\n"+menuText {
				t.Fatal("abort bytes", got)
			}
			if value() != "Old text.\r\n" || s.IsTextEditing() || s.menuStage != "menu" {
				t.Fatal("abort did not restore/return")
			}
			s.TerminalLine("2")
			renderedOutput(s)
			s.TerminalLine("/c")
			renderedOutput(s)
			s.TerminalLine(strings.Repeat("a", 238))
			if got := renderedOutput(s); got != "String too long - Truncated.\r\n] " {
				t.Fatal("240-byte limit", got)
			}
			if value() != strings.Repeat("a", 237)+"\r\n" {
				t.Fatal("truncation size", len(value()))
			}
			s.TerminalLine("next")
			if got := renderedOutput(s); got != "String too long.  Last line skipped.\r\n] " {
				t.Fatal("overflow append", got)
			}
			s.TerminalLine("@suffix")
			if got := renderedOutput(s); got != menuText {
				t.Fatal("save must emit menu only", got)
			}
			if s.IsTextEditing() || s.menuStage != "menu" {
				t.Fatal("save did not return")
			}
			s.TerminalLine("2")
			renderedOutput(s)
			s.TerminalLine("/c")
			renderedOutput(s)
			s.TerminalLine("")
			renderedOutput(s)
			s.TerminalLine("/s")
			if value() != "\r\n" || renderedOutput(s) != menuText {
				t.Fatal("blank line/save semantics")
			}
		})
	}
}

func TestEntryDescriptionPersistence(t *testing.T) {
	database := entryDatabase(t)
	record := entrySeed(t, database, "Scribe")
	record.Description = "Old text.\r\n"
	if err := database.SavePlayer(record); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Scribe", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	renderedOutput(s)
	sendMenuInput(t, s, "")
	renderedOutput(s)
	sendMenuInput(t, s, "2")
	renderedOutput(s)
	sendMenuInput(t, s, "Added text.")
	sendMenuInput(t, s, "/s")
	renderedOutput(s)
	rec, err := database.GetPlayer("Scribe")
	if err != nil || rec.Description != "Old text.\r\n" {
		t.Fatal("editor prematurely saved durable state")
	}
	sendMenuInput(t, s, "1")
	renderedOutput(s)
	rec, err = database.GetPlayer("Scribe")
	if err != nil || rec.Description != "Old text.\r\nAdded text.\r\n" {
		t.Fatal("entry did not persist description", rec, err)
	}
}

// Loaded empty strings are allocated in C; /c instead leaves a NULL pointer.
func TestEntryDescriptionEmptyPointer(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.player = game.NewPlayer(7, "Scribe", 1001)
	s.menuActive = true
	s.menuStage = "menu"
	sendMenuInput(t, s, "2")
	if got := renderedOutput(s); !strings.HasPrefix(got, "Current description:\r\nEnter ") {
		t.Fatal("loaded empty pointer not shown", got)
	}
	sendMenuInput(t, s, "/c")
	renderedOutput(s)
	sendMenuInput(t, s, "/s")
	renderedOutput(s)
	sendMenuInput(t, s, "2")
	if got := renderedOutput(s); strings.Contains(got, "Current description:") {
		t.Fatal("cleared NULL buffer shown", got)
	}
	sendMenuInput(t, s, "/a")
	renderedOutput(s)
	sendMenuInput(t, s, "2")
	if got := renderedOutput(s); strings.Contains(got, "Current description:") {
		t.Fatal("abort changed original NULL presence", got)
	}
}
