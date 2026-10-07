package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func fileEditorLogFixture(t *testing.T) (*Manager, *Session, *Session, *Session, *Session, *bytes.Buffer) {
	t.Helper()
	m := makeTestManager(t)
	m.world.ScriptsDir = t.TempDir()
	m.world.LibTextDir = t.TempDir()
	a := makeCommandTestSession(t, m, "Fileactor", 40, 1001)
	a.player.SetInvisLevel(40)
	a.player.SetPlrFlag(game.PrfLog1, true)
	a.player.SetPlrFlag(game.PrfLog2, true)
	watch := makeCommandTestSession(t, m, "Filewatch", 40, 1002)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	below := makeCommandTestSession(t, m, "Filebelow", 33, 1002)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	normal := makeCommandTestSession(t, m, "Filenormal", 40, 1002)
	normal.player.SetPlrFlag(game.PrfLog2, true)
	for _, s := range []*Session{a, watch, below, normal} {
		registerTestSession(t, m, s, s.player.Name)
	}
	file := captureMudlogFile(t)
	game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: "", onError: func() {
		if a.textEdit == nil || a.player.GetFlags()&(1<<game.PlrWriting) == 0 || len(a.send) != 0 {
			t.Error("file producer must precede ack and common cleanup")
		}
	}})
	return m, a, watch, below, normal, file
}

func fileEditorExpectLog(t *testing.T, a, watch, below, normal *Session, file *bytes.Buffer, payload, ack string) {
	t.Helper()
	if !strings.Contains(file.String(), payload+"\n") {
		t.Fatalf("missing file producer: %q actor=%q", file.String(), strings.Join(drainSessionText(t, a), ""))
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
		t.Fatalf("CMP observer=%q", got)
	}
	if got := strings.Join(drainSessionText(t, a), ""); got != ack {
		t.Fatalf("ack=%q want %q", got, ack)
	}
	if len(below.send) != 0 || len(normal.send) != 0 {
		t.Fatal("producer level/type leak")
	}
	if a.IsTextEditing() || (a.player.GetFlags()&(1<<game.PlrWriting) != 0) {
		t.Fatal("completion did not clean editor/writing")
	}
}

func TestFileEditorSaveMudlog(t *testing.T) {
	for _, kind := range []string{"root", "subdir", "help", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			m, a, watch, below, normal, file := fileEditorLogFixture(t)
			watch.player.SetLevel(34)
			var path, logical string
			switch kind {
			case "help":
				if err := os.Mkdir(filepath.Join(m.world.LibTextDir, "help"), 0o700); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(m.world.LibTextDir, "help/screen")
				logical = "text/help/screen"
				if err := cmdTedit(a, []string{"help"}); err != nil {
					t.Fatal(err)
				}
			case "subdir":
				if err := os.Mkdir(filepath.Join(m.world.ScriptsDir, "mob"), 0o700); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(m.world.ScriptsDir, "mob/probe.lua")
				logical = "scripts/mob/probe.lua"
				if err := cmdLuaEdit(a, []string{"mob", "probe"}); err != nil {
					t.Fatal(err)
				}
			default:
				path = filepath.Join(m.world.ScriptsDir, "probe.lua")
				logical = "scripts/probe.lua"
				if kind == "symlink" {
					target := filepath.Join(m.world.ScriptsDir, "physical.lua")
					if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				}
				if err := cmdLuaEdit(a, []string{"probe"}); err != nil {
					t.Fatal(err)
				}
			}
			drainSessionText(t, a)
			a.handleTextEditInput("/c")
			drainSessionText(t, a)
			a.handleTextEditInput("new line")
			payload := fmt.Sprintf("OLC: Fileactor saves '%s'.", logical)
			calls := 0
			game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: payload, onError: func() {
				calls++
				if a.textEdit == nil || a.player.GetFlags()&(1<<game.PlrWriting) == 0 || len(a.send) != 0 {
					t.Error("save log must precede ack/cleanup")
				}
				if b, err := os.ReadFile(path); err != nil || string(b) != "new line\n" {
					t.Errorf("log precedes disk save: %q %v", b, err)
				}
			}})
			a.handleTextEditInput("/s")
			fileEditorExpectLog(t, a, watch, below, normal, file, payload, "Saved.\r\n")
			if calls != 1 {
				t.Fatalf("log-time probe calls=%d", calls)
			}
		})
	}
}

func TestFileEditorDeleteMudlog(t *testing.T) {
	for _, kind := range []string{"readable", "absent", "unreadable", "dangling"} {
		t.Run(kind, func(t *testing.T) {
			m, a, watch, below, normal, file := fileEditorLogFixture(t)
			watch.player.SetLevel(34)
			path := filepath.Join(m.world.ScriptsDir, "probe.lua")
			if kind == "dangling" {
				if err := os.Symlink(filepath.Join(m.world.ScriptsDir, "missing.lua"), path); err != nil {
					t.Fatal(err)
				}
			} else if kind != "absent" {
				if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := cmdLuaEdit(a, []string{"probe"}); err != nil {
				t.Fatal(err)
			}
			drainSessionText(t, a)
			a.handleTextEditInput("/c")
			drainSessionText(t, a)
			if kind == "unreadable" {
				if os.Getuid() == 0 {
					t.Skip("readability boundary requires non-root uid")
				}
				if err := os.Chmod(path, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
			}
			payload := "OLC: Fileactor deletes 'scripts/probe.lua'."
			calls := 0
			game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: payload, onError: func() {
				calls++
				if a.textEdit == nil || a.player.GetFlags()&(1<<game.PlrWriting) == 0 || len(a.send) != 0 {
					t.Error("delete log must precede ack/cleanup")
				}
			}})
			a.handleTextEditInput("/s")
			fileEditorExpectLog(t, a, watch, below, normal, file, payload, "Deleted.\r\n")
			if calls != 1 {
				t.Fatalf("log-time probe calls=%d", calls)
			}
			_, err := os.Lstat(path)
			if kind == "readable" || kind == "absent" {
				if !os.IsNotExist(err) {
					t.Fatalf("delete failed: %v", err)
				}
			} else if err != nil {
				t.Fatalf("access failure must leave target: %v", err)
			}
		})
	}
}
