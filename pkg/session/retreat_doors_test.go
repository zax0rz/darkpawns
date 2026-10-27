package session

// retreat_doors_test.go — CAN_GO's third clause (src/utils.h:565-567) skips
// closed doors, so C's do_retreat loop (src/act.offensive.c:1056-1076) never
// attempts a closed exit; do_simple_move has no door check of its own. The
// port's retreat gate must skip closed exits too, or a lucky draw walks the
// player through a shut door do_flee's own gate already refuses.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeRetreatDoorsManager rooms 1002..1007 all closed behind doors from 1001.
func makeRetreatDoorsManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Vault", Zone: 1, Exits: make(map[string]parser.Exit, 6)}},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			Level:     5,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
		}},
	}
	for i, direction := range []string{"north", "east", "south", "west", "up", "down"} {
		parsed.Rooms[0].Exits[direction] = parser.Exit{
			Direction: direction, ToRoom: 1002 + i,
			Keywords: "vaultdoor", ExitInfo: parser.ExitClosed,
		}
		parsed.Rooms = append(parsed.Rooms, parser.Room{VNum: 1002 + i, Name: "Beyond", Zone: 1})
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	m := newTestManager(t, w, nil)
	m.combatEngine.Stop()
	m.combatEngine = combat.NewCombatEngine()
	return m
}

// TestCmdRetreatNeverCrossesClosedDoors fights in a room whose every exit is
// a closed door: whichever of the six draws is taken, C skips it and the
// command ends cornered, in the same room, with C's cornered bytes.
func TestCmdRetreatNeverCrossesClosedDoors(t *testing.T) {
	m := makeRetreatDoorsManager(t)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeFleeSession(t, m, "Retreater", 5)
	s.player.SetSkill(game.SkillRetreat, 100)
	if err := m.combatEngine.StartCombat(s.player, mob); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	seed := uint32(1)
	for dprng.New(seed).Number(1, 101) == 101 {
		seed++
	}
	dprng.ResetStream(seed)
	if err := cmdRetreat(s); err != nil {
		t.Fatalf("cmdRetreat: %v", err)
	}

	var sent strings.Builder
drain:
	for {
		select {
		case msg := <-s.send:
			sent.Write(msg)
		default:
			break drain
		}
	}
	if got := sent.String(); !strings.Contains(got, "You are cornered and fail to retreat!") {
		t.Fatalf("retreat output lacks C's cornered line:\n%q", got)
	}
	if room := s.player.GetRoom(); room != 1001 {
		t.Fatalf("retreat crossed a closed door: room %d, want 1001 (CAN_GO skips closed exits, utils.h:565-567)", room)
	}
}
