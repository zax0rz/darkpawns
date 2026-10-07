package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/fileedit"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestFileEditorStorageIdentity(t *testing.T) {
	m, a, _, _, _, _ := fileEditorLogFixture(t)
	for _, f := range textEditFields {
		if err := a.startTextEdit(f); err != nil {
			t.Fatal(err)
		}
		if a.textEdit.storage != "text/"+f.filename || a.textEdit.path != filepath.Join(m.world.LibTextDir, f.filename) || a.textEdit.cacheKey != f.filename {
			t.Fatalf("bad identity for %s: %+v", f.name, a.textEdit)
		}
		a.cancelTextEdit()
		drainSessionText(t, a)
	}
	for _, args := range [][]string{{"rootprobe"}, {"mob", "probe"}, {"obj/123", "probe"}, {"mob//123", "probe"}} {
		if err := cmdLuaEdit(a, args); err != nil {
			t.Fatal(err)
		}
		if a.textEdit == nil {
			t.Fatalf("no editor for %v", args)
		}
		want := "scripts/"
		if len(args) > 1 {
			want += args[0] + "/" + args[1]
		} else {
			want += args[0]
		}
		want += ".lua"
		if a.textEdit.storage != want || a.textEdit.cacheKey != "" {
			t.Fatalf("identity for %v=%+v", args, a.textEdit)
		}
		a.cancelTextEdit()
		drainSessionText(t, a)
	}
}

func TestFileEditorActingBody(t *testing.T) {
	for _, kind := range []string{"player", "mob"} {
		t.Run(kind, func(t *testing.T) {
			w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}, {VNum: 1003}}, Mobs: []parser.Mob{{VNum: 90, Keywords: "guard", ShortDesc: "a retained guard", Level: 40}}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			m := newTestManager(t, w, nil)
			w.ScriptsDir = t.TempDir()
			a := makeCommandTestSession(t, m, "Wizard", 40, 1001)
			a.transportDone = make(chan struct{})
			registerTestSession(t, m, a, "Wizard")
			watch := makeCommandTestSession(t, m, "Watch", 40, 1003)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			watch.player.SetPlrFlag(game.PrfLog2, true)
			registerTestSession(t, m, watch, "Watch")
			name := "a retained guard"
			target := "guard"
			if kind == "mob" {
				if _, err := w.SpawnMobQuiet(90, 1001); err != nil {
					t.Fatal(err)
				}
			} else {
				h := makeCommandTestSession(t, m, "Borrowed", 40, 1002)
				h.player = game.NewPlayer(2, "Borrowed", 1002)
				h.player.SetLevel(40)
				h.transportDone = make(chan struct{})
				h.DetachTransport()
				h.player.SetLinkless(true)
				registerTestSession(t, m, h, "Borrowed")
				name = "Borrowed"
				target = name
			}
			if err := cmdSwitch(a, []string{target}); err != nil {
				t.Fatal(err)
			}
			drainSessionText(t, a)
			drainSessionText(t, watch)
			file := captureMudlogFile(t)
			if err := cmdLuaEdit(a, []string{"probe"}); err != nil {
				t.Fatal(err)
			}
			drainSessionText(t, a)
			a.handleTextEditInput("bytes")
			a.handleTextEditInput("/s")
			payload := "OLC: " + name + " saves 'scripts/probe.lua'."
			if !strings.Contains(file.String(), payload) {
				t.Fatalf("body name log=%q", file.String())
			}
			if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
				t.Fatalf("observer=%q", got)
			}
		})
	}
}

func TestFileEditorNonFileBoundaries(t *testing.T) {
	m, a, watch, _, _, file := fileEditorLogFixture(t)
	a.startLiveStringEditor(func(string) {}, 200)
	a.handleTextEditInput("text")
	a.handleTextEditInput("/s")
	if file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("live string acquired file log")
	}
	if err := cmdLuaEdit(a, []string{"probe"}); err != nil {
		t.Fatal(err)
	}
	drainSessionText(t, a)
	a.handleTextEditInput("/a")
	drainSessionText(t, a)
	if file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("abort acquired file log")
	}
	outside := filepath.Join(t.TempDir(), "outside.lua")
	if err := os.WriteFile(outside, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(m.world.ScriptsDir, "escape.lua")); err != nil {
		t.Fatal(err)
	}
	if err := cmdLuaEdit(a, []string{"escape"}); err != nil {
		t.Fatal(err)
	}
	if a.IsTextEditing() || file.Len() != 0 {
		t.Fatal("escape acquired editor/log")
	}
	drainSessionText(t, a)
	if err := fileedit.AtomicWrite(filepath.Join(m.world.LibTextDir, "news"), []byte("web bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !fileedit.NotifyTextSaved(m.world, "news", "web bytes") {
		t.Fatal("cache hook absent")
	}
	if file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("web save acquired descriptor log")
	}
	if textEditTestCache("news") != "web bytes" {
		t.Fatal("web cache ownership changed")
	}
}

func TestFileEditorOpenStageClassifier(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "probe.lua")
	_, err := os.OpenRoot(parent)
	if err == nil || !fileEditorOpenParentFailure(err, path) {
		t.Fatalf("real root-open stage not recognized: %v", err)
	}
	for _, op := range []string{"write", "sync", "close", "chmod", "rename", "stat"} {
		err := &os.PathError{Op: op, Path: parent, Err: errors.New("failure")}
		if fileEditorOpenParentFailure(err, path) {
			t.Fatalf("late %s stage classified", op)
		}
	}
	if fileEditorOpenParentFailure(&os.PathError{Op: "open", Path: path, Err: errors.New("failure")}, path) {
		t.Fatal("unproven leaf open classified")
	}
}

func TestFileEditorSaveCleanupRace(t *testing.T) {
	_, a, _, _, _, _ := fileEditorLogFixture(t)
	if err := cmdLuaEdit(a, []string{"probe"}); err != nil {
		t.Fatal(err)
	}
	drainSessionText(t, a)
	a.handleTextEditInput("bytes")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.handleTextEditInput("/s") }()
	go func() { defer wg.Done(); a.cancelTextEdit() }()
	wg.Wait()
	if a.IsTextEditing() {
		t.Fatal("editor retained after concurrent cleanup")
	}
}
