package session

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

type olcOpenLogProbe struct {
	buffer  *bytes.Buffer
	payload string
	onError func()
}

func (p *olcOpenLogProbe) Write(b []byte) (int, error) {
	if strings.Contains(string(b), p.payload) {
		p.onError()
	}
	return p.buffer.Write(b)
}

func testOLCOpenParentMudlog(t *testing.T, command, extension, payload, announcement string, kind olc.Kind) {
	t.Helper()
	for _, missing := range []bool{false, true} {
		if missing && command != "redit" {
			continue
		}
		name := "non-directory"
		if missing {
			name = "missing"
		}
		t.Run(name, func(t *testing.T) {
			w := makeZeditTestWorld(t)
			for _, ext := range []string{"wld", "obj", "mob", "shp"} {
				if err := os.Mkdir(filepath.Join(w.WorldPath, ext), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			parent := filepath.Join(w.WorldPath, extension)
			if err := os.Remove(parent); err != nil {
				t.Fatal(err)
			}
			if !missing {
				if err := os.WriteFile(parent, []byte("parent obstruction"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			m := newTestManager(t, w, nil)
			actor := makeCommandTestSession(t, m, "Openactor", 40, 3000)
			actor.player.SetInvisLevel(40)
			watch := makeCommandTestSession(t, m, "Openwatch", 31, 3001)
			watch.player.SetPlrFlag(game.PrfLog1, true) // BRF, so the pre-save CMP line is filtered.
			below := makeCommandTestSession(t, m, "Openbelow", 30, 3001)
			below.player.SetPlrFlag(game.PrfLog1, true)
			below.player.SetPlrFlag(game.PrfLog2, true)
			off := makeCommandTestSession(t, m, "Openoff", 40, 3001)
			for _, s := range []*Session{actor, watch, below, off} {
				registerTestSession(t, m, s, s.player.Name)
			}
			markOLCDirty(kind, 30)
			t.Cleanup(func() { clearOLCDirty(kind, 30) })
			file := captureMudlogFile(t)
			errorCalls := 0
			var actorOutput string
			game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: payload, onError: func() {
				errorCalls++
				actorOutput += strings.Join(drainSessionText(t, actor), "")
				if actorOutput != announcement {
					t.Errorf("announcement must precede error: %q", actorOutput)
				}
				if !olcSaveList.Dirty(kind, 30) {
					t.Error("error cleared dirty marker")
				}
				lock := zoneSaveLock(30)
				if !lock.TryLock() {
					t.Error("producer holds zone save lock")
				} else {
					lock.Unlock()
				}
			}})
			if err := ExecuteCommand(actor, command, []string{"save", "30"}); err != nil {
				t.Fatal(err)
			}
			actorOutput += strings.Join(drainSessionText(t, actor), "")
			if errorCalls != 1 || !strings.Contains(file.String(), payload+"\n") {
				t.Fatalf("missing open producer: calls=%d file=%q", errorCalls, file.String())
			}
			if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
				t.Fatalf("BRF observer=%q", got)
			}
			if actorOutput != announcement {
				t.Fatalf("error acknowledgement: %q", actorOutput)
			}
			if len(below.send) != 0 || len(off.send) != 0 {
				t.Fatal("producer filter leak")
			}
			if !olcSaveList.Dirty(kind, 30) {
				t.Fatal("failed save cleared marker")
			}
			// The shared admin route encounters the same real obstruction but has no C command producer.
			file.Reset()
			if err := m.SaveOLCZone(30); err == nil {
				t.Fatal("admin save unexpectedly succeeded")
			}
			if errorCalls != 1 || file.Len() != 0 || len(watch.send) != 0 {
				t.Fatal("admin save broadcast a command diagnostic")
			}
			if !olcSaveList.Dirty(kind, 30) {
				t.Fatal("admin failure cleared marker")
			}
			// Successful saves keep the existing CMP announcement, with no error producer.
			if !missing {
				if err := os.Remove(parent); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(parent, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := ExecuteCommand(actor, command, []string{"save", "30"}); err != nil {
				t.Fatal(err)
			}
			if errorCalls != 1 || strings.Contains(file.String(), payload) || len(watch.send) != 0 {
				t.Fatal("successful save emitted open error")
			}
			if olcSaveList.Dirty(kind, 30) {
				t.Fatal("successful save retained marker")
			}
		})
	}
}
