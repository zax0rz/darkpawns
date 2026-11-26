package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLuaEditListMatchesCFilterAndThreeColumnLayout(t *testing.T) {
	m := makeTestManager(t)
	root := t.TempDir()
	m.world.ScriptsDir = root
	for _, name := range []string{"mob", "obj", "room", "archive"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{
		"12",
		"a.lua",
		"ignored.txt",
		"my.lua.txt",
		"README",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzz.lua",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := makeCommandTestSession(t, m, "Luaeditlist", game.LVL_IMMORT, 1001)
	if err := cmdLuaEdit(s, nil); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("%-26.26s%-26.26s%-26.26s\r\n%-26.26s%-26.26s%-26.26s\r\n%-26.26s\r\n",
		"12", "a.lua", "archive", "mob", "obj", "room", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzz.lua")
	if got := readMsgText(t, s); got != want {
		t.Fatalf("luaedit list = %q, want %q", got, want)
	}
	if err := cmdLuaEdit(s, []string{"obj"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "None.\r\n" {
		t.Fatalf("empty luaedit list = %q", got)
	}
}

func TestLuaEditArgumentDanceViewAndSubstringExtension(t *testing.T) {
	m := makeTestManager(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "mob"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.world.ScriptsDir = root
	if err := os.WriteFile(filepath.Join(root, "root.lua"), []byte("root line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mob", "mobfile.lua"), []byte("mob line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := makeCommandTestSession(t, m, "Luaeditview", game.LVL_IMMORT, 1001)
	if err := cmdLuaEdit(s, []string{"ROOT", "ignored"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "root line\r\n" {
		t.Fatalf("root filename argument = %q", got)
	}

	if err := cmdLuaEdit(s, []string{"MOB", "MOBFILE"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "mob line\r\n" {
		t.Fatalf("subdirectory argument = %q", got)
	}

	if err := cmdLuaEdit(s, []string{"my.lua.txt"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != luaEditError {
		t.Fatalf("substring extension error = %q", got)
	}
}

func TestLuaEditViewBelowHigodAndEditAtHigod(t *testing.T) {
	m := makeTestManager(t)
	root := t.TempDir()
	m.world.ScriptsDir = root
	path := filepath.Join(root, "editable.lua")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	viewer := makeCommandTestSession(t, m, "Luaeditviewer", LVL_HIGOD-1, 1001)
	if err := cmdLuaEdit(viewer, []string{"editable"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, viewer); got != "old line\r\n" {
		t.Fatalf("below-HIGOD view = %q", got)
	}
	if viewer.isTextEditing() {
		t.Fatal("below-HIGOD view entered the editor")
	}

	editor := makeCommandTestSession(t, m, "Luaediteditor", LVL_HIGOD, 1001)
	if err := cmdLuaEdit(editor, []string{"editable"}); err != nil {
		t.Fatal(err)
	}
	wantStart := "Instructions: /s or @ to save, /h for more options.\r\n" +
		"Edit file below:\r\n\r\nold line\r\n"
	if got := readMsgText(t, editor); got != wantStart {
		t.Fatalf("at-HIGOD editor entry = %q, want %q", got, wantStart)
	}
	editor.handleTextEditInput("new line")
	editor.handleTextEditInput("/s")
	if got := readMsgText(t, editor); got != "Saved.\r\n" {
		t.Fatalf("luaedit save = %q", got)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "old line\nnew line\n" {
		t.Fatalf("saved script = %q, err=%v", got, err)
	}
}

func TestLuaEditEmptySaveDeletesAndNewFileSaveCreates(t *testing.T) {
	m := makeTestManager(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "mob"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.world.ScriptsDir = root
	deletePath := filepath.Join(root, "delete.lua")
	if err := os.WriteFile(deletePath, []byte("delete me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := makeCommandTestSession(t, m, "Luaeditdelete", LVL_HIGOD, 1001)
	if err := cmdLuaEdit(s, []string{"delete"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleTextEditInput("/c")
	if got := readMsgText(t, s); got != "Current buffer cleared.\r\n" {
		t.Fatalf("clear before delete = %q", got)
	}
	s.handleTextEditInput("/s")
	if got := readMsgText(t, s); got != "Deleted.\r\n" {
		t.Fatalf("empty save = %q", got)
	}
	if _, err := os.Stat(deletePath); !os.IsNotExist(err) {
		t.Fatalf("deleted script stat error = %v", err)
	}

	newPath := filepath.Join(root, "mob", "new.lua")
	if err := cmdLuaEdit(s, []string{"mob", "new"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Instructions: /s or @ to save, /h for more options.\r\nEdit file below:\r\n\r\n" {
		t.Fatalf("new file editor entry = %q", got)
	}
	s.handleTextEditInput("created")
	s.handleTextEditInput("/s")
	if got := readMsgText(t, s); got != "Saved.\r\n" {
		t.Fatalf("new file save = %q", got)
	}
	if got, err := os.ReadFile(newPath); err != nil || string(got) != "created\n" {
		t.Fatalf("created script = %q, err=%v", got, err)
	}
}

func TestLuaEditRejectsTraversalAndCFileReadCap(t *testing.T) {
	m := makeTestManager(t)
	root := t.TempDir()
	m.world.ScriptsDir = root
	if err := os.WriteFile(filepath.Join(root, "huge.lua"), []byte(strings.Repeat("x", cMaxStringLength)), 0o644); err != nil {
		t.Fatal(err)
	}
	s := makeCommandTestSession(t, m, "Luaeditguard", LVL_HIGOD-1, 1001)
	for _, args := range [][]string{{"../outside"}, {"mob/.."}, {"huge"}} {
		if err := cmdLuaEdit(s, args); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, s); got != luaEditError {
			t.Fatalf("luaedit %v = %q, want %q", args, got, luaEditError)
		}
	}
}

// TestLuaEditListAcceptsDottedAbsoluteScriptsRoot pins DP-1376: C's
// valid_directory (src/file-edit.c:78-81) saw the relative SCRIPT_DIR
// ("scripts", src/db.h:79; src/luaedit.c:37-42), never the absolute deploy
// path. A Go root containing a dot (a home or deploy directory) must list
// exactly as C would, while traversal spellings still hit the C regex.
func TestLuaEditListAcceptsDottedAbsoluteScriptsRoot(t *testing.T) {
	m := makeTestManager(t)
	root := filepath.Join(t.TempDir(), "deploy.dir", "scripts")
	if err := os.MkdirAll(filepath.Join(root, "mob"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.lua"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.ScriptsDir = root

	s := makeCommandTestSession(t, m, "Luaeditdots", game.LVL_IMMORT, 1001)
	if err := cmdLuaEdit(s, nil); err != nil {
		t.Fatal(err)
	}
	got := readMsgText(t, s)
	if got == luaEditError || !strings.Contains(got, "a.lua") {
		t.Fatalf("dotted-root listing = %q, want the a.lua entry", got)
	}

	if err := cmdLuaEdit(s, []string{"mob"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "None.\r\n" {
		t.Fatalf("dotted-root subdir listing = %q, want None", got)
	}

	// The same relative check still rejects a traversal spelling.
	if err := cmdLuaEdit(s, []string{"mob/../../etc"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != luaEditError {
		t.Fatalf("traversal listing = %q, want the C error", got)
	}
}
