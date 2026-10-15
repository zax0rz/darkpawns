package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func spawnSocialTestMob(t *testing.T, w *World) *MobInstance {
	t.Helper()
	proto := &parser.Mob{VNum: 9701, Race: 7, Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 10}}
	w.mu.Lock()
	w.mobs[9701] = proto
	w.mu.Unlock()
	mob, err := w.SpawnMob(9701, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	return mob
}

// A mob executing a short social must stay inside the message table: blush
// carries three messages (char/others no-arg, then "#"), so the unguarded
// mob path indexed past the table on both target shapes and killed the whole
// process on the recover-less telnet input goroutine (order <pet> blush
// <word>). The unified npcSocial path renders it like C's do_action instead.
func TestNPCCommandShortSocialDoesNotPanic(t *testing.T) {
	w, actor, local, _, output := newChannelWorld(t)
	mob := spawnSocialTestMob(t, w)

	// Target not found and target found — both routes on a three-message
	// social. (The retired adapter's "social <name>" prefix form was an R4
	// invention: C registers no such command, so it is gone with the adapter.)
	w.NPCCommand(mob, "blush nobodyhere")
	w.NPCCommand(mob, "blush "+local.Name)

	for _, name := range []string{actor.Name, local.Name} {
		if got := channelOutput(output, name); strings.Contains(got, "#") {
			t.Fatalf("%s output = %q, want no raw # slot marker", name, got)
		}
	}

	// grin carries all slots: the targeted form still reaches the target.
	w.NPCCommand(mob, "grin "+actor.Name)
	if got := channelOutput(output, actor.Name); !strings.Contains(got, "grins at you") {
		t.Fatalf("actor output after grin = %q, want the ToVict grin message", got)
	}
}

// C's do_action ignores the argument when a social has no targeted form and
// plays the no-arg version (src/act.social.c:120-130): ordering a mob to
// blush at anyone shows the room the plain no-arg blush line, not nothing.
func TestMobSocialNoTargetedFormPlaysNoArgRoomLine(t *testing.T) {
	w, actor, local, _, output := newChannelWorld(t)
	mob := spawnSocialTestMob(t, w)

	w.NPCCommand(mob, "blush "+local.Name)

	if got := channelOutput(output, actor.Name); !strings.Contains(got, "blushes.") {
		t.Fatalf("actor output for targeted blush = %q, want the no-arg room line", got)
	}
	if got := channelOutput(output, local.Name); !strings.Contains(got, "blushes.") {
		t.Fatalf("local output for targeted blush = %q, want the no-arg room line", got)
	}
}
