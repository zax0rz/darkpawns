package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestOLCDiskMudlogProducers(t *testing.T) {
	openErrors := map[string]string{
		"redit": "SYSERR: OLC: Cannot open room file!",
		"oedit": "SYSERR: OLC: Cannot open objects file!",
		"medit": "SYSERR: OLC: Cannot open mob file!",
		"sedit": "SYSERR: OLC: Cannot open shop file!",
		"zedit": "SYSERR: OLC: zedit_save_to_disk:  Can't write zone 30.",
	}
	for _, tc := range []struct{ command, kind, ack, dir, ext string }{
		{"redit", "rooms", "Saving all rooms in zone.", "wld", "wld"},
		{"zedit", "zone info", "Saving all zone information.", "zon", "zon"},
		{"oedit", "objects", "Saving all objects in zone.", "obj", "obj"},
		{"medit", "mobs", "Saving all mobiles in zone.", "mob", "mob"},
		{"sedit", "shops", "Saving all shops in zone.", "shp", "shp"},
	} {
		for _, failedWrite := range []bool{false, true} {
			t.Run(tc.command+fmt.Sprint(failedWrite), func(t *testing.T) {
				w := makeOeditTestWorld(t)
				t.Cleanup(w.StopAITicker)
				w.WorldPath = w.GetParsedWorld().SourceDir
				for _, dir := range []string{"wld", "zon"} {
					if err := os.MkdirAll(filepath.Join(w.WorldPath, dir), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				m := newTestManager(t, w, nil)
				a := makeTestSession(t, m, "Olcactor", 3000, true)
				a.player.SetLevel(40)
				a.player.SetInvisLevel(40)
				a.player.SetPosition(combat.PosStanding)
				watch := makeTestSession(t, m, "Olcwatch", 3000, true)
				watch.player.SetLevel(LVL_IMMORT)
				watch.player.SetPlrFlag(game.PrfLog1, true)
				watch.player.SetPlrFlag(game.PrfLog2, true)
				below := makeTestSession(t, m, "Olcbelow", 3000, true)
				below.player.SetLevel(LVL_IMMORT - 1)
				below.player.SetPlrFlag(game.PrfLog1, true)
				below.player.SetPlrFlag(game.PrfLog2, true)
				normal := makeTestSession(t, m, "Olcnormal", 3000, true)
				normal.player.SetLevel(40)
				normal.player.SetPlrFlag(game.PrfLog2, true)
				for _, s := range []*Session{a, watch, below, normal} {
					registerTestSession(t, m, s, s.player.Name)
				}
				if failedWrite {
					blocked := filepath.Join(w.GetParsedWorld().SourceDir, "blocked")
					if err := os.WriteFile(blocked, []byte("fixture"), 0o600); err != nil {
						t.Fatal(err)
					}
					w.GetParsedWorld().SourceDir = blocked
					w.WorldPath = blocked
				}
				path := filepath.Join(w.GetParsedWorld().SourceDir, tc.dir, "30."+tc.ext)
				file := captureMudlogFile(t)
				ready := false
				payload := fmt.Sprintf("OLC: Olcactor saves %s for zone 30", tc.kind)
				game.SetLogWriter(&olcOpenLogProbe{buffer: file, payload: payload, onError: func() { _, err := os.Stat(path); ready = len(a.send) > 0 && err != nil }})
				if err := executeCommand(a, tc.command, []string{"save", "30"}, false); err != nil {
					t.Fatal(err)
				}
				if !ready {
					t.Fatal("OLC must acknowledge and log before disk write")
				}
				if !strings.Contains(file.String(), payload) {
					t.Fatalf("missing file producer: %q", file.String())
				}
				errorLine := ""
				if failedWrite && openErrors[tc.command] != "" {
					errorLine = "[ " + openErrors[tc.command] + " ]\r\n"
				}
				if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n"+errorLine {
					t.Fatalf("observer=%q", got)
				}
				if got := strings.Join(drainSessionText(t, a), ""); got != tc.ack+"\r\n" {
					t.Fatalf("ack=%q", got)
				}
				if len(below.send) != 0 || strings.Join(drainSessionText(t, normal), "") != errorLine {
					t.Fatal("OLC threshold/type leak")
				}
				if _, err := os.Stat(path); (err == nil) == failedWrite {
					t.Fatalf("unexpected write result (failure %v): %v", failedWrite, err)
				}
			})
		}
	}
}
