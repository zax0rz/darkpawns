package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// A mob executing a short social must render like the guarded player path:
// blush carries three messages (char/others no-arg, then "#"), so a targeted
// blush has no target-found or not-found message at all. The unguarded mob
// path indexed past the table and killed the whole process on the
// recover-less telnet input goroutine (order <pet> blush <word>).
func TestExecMobCommandShortSocialDoesNotPanic(t *testing.T) {
	w, actor, local, _, output := newChannelWorld(t)

	proto := &parser.Mob{VNum: 9701, Race: 7, Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 10}}
	w.mu.Lock()
	w.mobs[9701] = proto
	w.mu.Unlock()
	mob, err := w.SpawnMob(9701, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	// Target not found: socNotFound (index 5) does not exist for blush.
	w.ExecMobCommand(mob.GetVNum(), "blush nobodyhere")
	// Target found: socOthersFound (index 3) does not exist for blush.
	w.ExecMobCommand(mob.GetVNum(), "blush "+local.Name)

	if got := channelOutput(output, actor.Name); got != "" {
		t.Fatalf("actor output after short socials = %q, want none", got)
	}

	// Positive control: a full social still renders to the target.
	w.ExecMobCommand(mob.GetVNum(), "grin "+actor.Name)
	if got := channelOutput(output, actor.Name); !strings.Contains(got, "grins at you") {
		t.Fatalf("actor output after grin = %q, want the ToVict grin message", got)
	}
}
