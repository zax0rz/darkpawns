package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZeditUnknownCommandMudlogBoundary(t *testing.T) {
	for _, route := range []string{"command", "save-all", "admin", "open-failure"} {
		t.Run(route, func(t *testing.T) {
			w := makeZeditTestWorld(t)
			for _, ext := range []string{"wld", "obj", "mob", "shp"} {
				if err := os.Mkdir(filepath.Join(w.WorldPath, ext), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			// Use the real parser's accepted imported-data topology. C's
			// failed G followed by conditional unknown opcodes survives boot.
			input := filepath.Join(t.TempDir(), "30.zon")
			if err := os.WriteFile(input, []byte("#30\nImported zone~\n3099 30 2\nG 0 3003 0\nX 1 0 0\nZ 1 0 0\nS\n$\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			zone, err := parser.ParseZonFile(input)
			if err != nil {
				t.Fatal(err)
			}
			if zone.Commands[1].IfFlag != 1 || zone.Commands[2].IfFlag != 1 {
				t.Fatal("import lost conditional flags")
			}
			// '*' is a silent disabled command; S ends scanning even if a
			// malformed in-memory tail exists. Neither may emit a diagnostic.
			zone.Commands = append(zone.Commands, parser.ZoneCommand{Command: "*"}, parser.ZoneCommand{Command: "S"}, parser.ZoneCommand{Command: "Q"})
			loaded := *w.GetParsedWorld()
			loaded.Zones = []parser.Zone{*zone}
			w, err = game.NewWorld(&loaded)
			if err != nil {
				t.Fatal(err)
			}
			w.WorldPath = loaded.SourceDir
			t.Cleanup(func() { w.StopAITicker() })
			m := newTestManager(t, w, nil)
			actor := makeCommandTestSession(t, m, "Zoneactor", 40, 3000)
			actor.player.SetInvisLevel(40)
			watch := makeCommandTestSession(t, m, "Zonewatch", 31, 3001)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			below := makeCommandTestSession(t, m, "Zonebelow", 30, 3001)
			below.player.SetPlrFlag(game.PrfLog1, true)
			for _, s := range []*Session{actor, watch, below} {
				registerTestSession(t, m, s, s.player.Name)
			}
			markOLCDirty(olc.KindZone, 30)
			t.Cleanup(func() { clearOLCDirty(olc.KindZone, 30) })
			parent := filepath.Join(w.WorldPath, "zon")
			target := filepath.Join(parent, "30.zon")
			if err := os.WriteFile(target, []byte("old target"), 0o600); err != nil {
				t.Fatal(err)
			}
			if route == "open-failure" {
				if err := os.RemoveAll(parent); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(parent, []byte("obstruction"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			file := captureMudlogFile(t)
			var actorText string
			calls := 0
			game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: "Unknown cmd", onError: func() {
				calls++
				actorText += strings.Join(drainSessionText(t, actor), "")
				ack := "Saving all zone information.\r\n"
				if route == "save-all" {
					ack = "Zone info saved for zone 30.\r\n"
				}
				if actorText != ack {
					t.Errorf("acknowledgement before unknown diagnostic=%q", actorText)
				}
				entries, err := filepath.Glob(filepath.Join(parent, ".tmp-*"))
				if err != nil || len(entries) != 1 {
					t.Errorf("diagnostic must follow successful open: files=%v err=%v", entries, err)
				}
				data, err := os.ReadFile(target)
				if err != nil || string(data) != "old target" {
					t.Errorf("diagnostic must precede atomic publication: data=%q err=%v", data, err)
				}
				if !olcSaveList.Dirty(olc.KindZone, 30) {
					t.Error("diagnostic follows premature dirty cleanup")
				}
				lock := zoneSaveLock(30)
				if lock.TryLock() {
					lock.Unlock()
					t.Error("save no longer owns zone lock during rendering")
				}
			}})
			switch route {
			case "save-all":
				err = ExecuteCommand(actor, "olc", []string{"save"})
			case "admin":
				err = m.SaveOLCZone(30)
			default:
				err = ExecuteCommand(actor, "zedit", []string{"save", "30"})
			}
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(drainSessionText(t, watch), "")
			wantCalls := 2
			if route == "admin" || route == "open-failure" {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatalf("unknown producer calls=%d want=%d", calls, wantCalls)
			}
			want := "[ SYSERR: OLC: z_save_to_disk(): Unknown cmd 'X' - NOT saving ]\r\n[ SYSERR: OLC: z_save_to_disk(): Unknown cmd 'Z' - NOT saving ]\r\n"
			switch route {
			case "admin":
				want = ""
			case "open-failure":
				want = "[ SYSERR: OLC: zedit_save_to_disk:  Can't write zone 30. ]\r\n"
			}
			if got != want || len(below.send) != 0 {
				t.Fatalf("observer=%q want=%q below=%d", got, want, len(below.send))
			}
			if route == "open-failure" {
				if !olcSaveList.Dirty(olc.KindZone, 30) {
					t.Fatal("failed open cleared marker")
				}
				return
			}
			data, err := os.ReadFile(target)
			if err != nil || strings.Contains(string(data), "X ") || strings.Contains(string(data), "Z ") || strings.Contains(string(data), "Q ") || !strings.HasSuffix(string(data), "S\n$\n") {
				t.Fatalf("saved zone=%q err=%v", data, err)
			}
			if olcSaveList.Dirty(olc.KindZone, 30) {
				t.Fatal("successful save retained marker")
			}
		})
	}
}
