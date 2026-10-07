package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestZeditOpenParentUsesWriterRoot(t *testing.T) {
	w := makeZeditTestWorld(t)
	// Zone writes use WorldPath. Keep the parsed SourceDir's zon parent
	// valid, so accidentally classifying that different root loses the log.
	w.WorldPath = t.TempDir()
	if err := os.WriteFile(filepath.Join(w.WorldPath, "zon"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Rootactor", 40, 3000)
	watch := makeCommandTestSession(t, m, "Rootwatch", 31, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	registerTestSession(t, m, actor, actor.player.Name)
	registerTestSession(t, m, watch, watch.player.Name)
	captureMudlogFile(t)
	if err := ExecuteCommand(actor, "zedit", []string{"save", "30"}); err != nil {
		t.Fatal(err)
	}
	want := "[ SYSERR: OLC: zedit_save_to_disk:  Can't write zone 30. ]\r\n"
	if got := strings.Join(drainSessionText(t, watch), ""); got != want {
		t.Fatalf("wrong root used for zone open classification: got=%q want=%q", got, want)
	}
}
