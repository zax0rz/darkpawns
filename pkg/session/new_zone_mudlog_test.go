package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func testNewZoneMudlog(t *testing.T, failed string) {
	w := makeZeditTestWorld(t)
	extensions := []string{"zon", "wld", "mob", "obj", "shp"}
	for _, ext := range extensions {
		dir := filepath.Join(w.WorldPath, ext)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index"), []byte("30."+ext+"\n$\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if failed != "" {
		if err := os.Mkdir(filepath.Join(w.WorldPath, failed, "31."+failed), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := newTestManager(t, w, nil)
	a := makeCommandTestSession(t, m, "Zoneactor", 40, 3000)
	a.player.SetInvisLevel(40)
	minimum := LVL_IMMORT
	if failed != "" {
		minimum = game.LVL_IMPL
	}
	watch := makeCommandTestSession(t, m, "Zonewatch", minimum, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	below := makeCommandTestSession(t, m, "Zonebelow", minimum-1, 3001)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	off := makeCommandTestSession(t, m, "Zoneoff", 40, 3001)
	for _, s := range []*Session{a, watch, below, off} {
		registerTestSession(t, m, s, s.player.Name)
	}
	payload := "OLC: Zoneactor creates new zone #31"
	if failed != "" {
		kind := map[string]string{"zon": "zone", "wld": "world", "mob": "mob", "obj": "obj", "shp": "shop"}[failed]
		payload = "SYSERR: OLC: Can't write new " + kind + " file"
	}
	file := captureMudlogFile(t)
	called := false
	game.SetLogWriter(&flagProbe{buf: file, when: func() {
		called = true
		if len(a.send) != 0 {
			t.Error("new-zone log must precede actor acknowledgement")
		}
		_, created := w.SnapshotZone(31)
		if created != (failed == "") {
			t.Error("wrong world state at new-zone producer")
		}
		reachedFailure := false
		for _, ext := range extensions {
			if ext == failed {
				reachedFailure = true
				continue
			}
			_, err := os.Stat(filepath.Join(w.WorldPath, ext, "31."+ext))
			if (err == nil) == reachedFailure {
				t.Errorf("wrong file ordering at log: %s: %v", ext, err)
			}
			data, err := os.ReadFile(filepath.Join(w.WorldPath, ext, "index"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "31."+ext) != (failed == "") {
				t.Errorf("wrong index state at log: %s", ext)
			}
		}
	}})
	if err := ExecuteCommand(a, "zedit", []string{"new", "31"}); err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(file.String(), payload) {
		t.Fatalf("missing new-zone producer: %q", file.String())
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
		t.Fatalf("observer=%q", got)
	}
	ack := ""
	if failed == "" {
		ack = "Zone created.\r\n"
	}
	if got := strings.Join(drainSessionText(t, a), ""); got != ack {
		t.Fatalf("actor=%q", got)
	}
	if len(below.send) != 0 || len(off.send) != 0 {
		t.Fatal("new-zone producer filter leak")
	}
	// C's coverage and ceiling refusals precede all file and producer activity.
	file.Reset()
	called = false
	for _, number := range []string{"30", "327"} {
		if err := ExecuteCommand(a, "zedit", []string{"new", number}); err != nil {
			t.Fatal(err)
		}
	}
	if called || file.Len() != 0 || len(watch.send) != 0 {
		t.Fatal("refusals must not log")
	}
	if got := strings.Join(drainSessionText(t, a), ""); got != "A zone already covers that area.\r\n326 is the highest zone allowed.\r\n" {
		t.Fatalf("refusals=%q", got)
	}
}
