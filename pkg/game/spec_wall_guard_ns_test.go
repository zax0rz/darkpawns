package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newWallGuardTestWorld(t *testing.T, southExit bool) (*World, *MobInstance, *Player, *Player, map[string]string) {
	t.Helper()

	room2Exits := map[string]parser.Exit{}
	if southExit {
		room2Exits["south"] = parser.Exit{Direction: "south", ToRoom: 1001}
	}
	parsed := &parser.World{Rooms: []parser.Room{
		{
			VNum: 1001,
			Name: "South Wall",
			Zone: 1,
			Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 1002},
			},
		},
		{VNum: 1002, Name: "North Wall", Zone: 1, Exits: room2Exits},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	oldRoom := NewPlayer(1, "OldRoom", 1001)
	observer := NewPlayer(2, "Observer", 1002)
	if err := w.AddPlayer(oldRoom); err != nil {
		t.Fatalf("AddPlayer old-room witness: %v", err)
	}
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer destination witness: %v", err)
	}

	guardProto := &parser.Mob{
		VNum:        8060,
		Keywords:    "city guard guard_ns",
		ShortDesc:   "a city guard",
		LongDesc:    "A city guard patrols the wall road.",
		Level:       10,
		Position:    combat.PosStanding,
		DefaultPos:  combat.PosStanding,
		HP:          parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		ActionFlags: []string{"SPEC", "SENTINEL"},
	}
	churchProto := &parser.Mob{
		VNum:        8020,
		Keywords:    "guard church armored",
		ShortDesc:   "a church guard",
		LongDesc:    "An armored church guard keeps an eye on the gate.",
		Level:       10,
		Position:    combat.PosStanding,
		DefaultPos:  combat.PosStanding,
		HP:          parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		ActionFlags: []string{"SENTINEL"},
	}
	w.mu.Lock()
	w.mobs[guardProto.VNum] = guardProto
	w.mobs[churchProto.VNum] = churchProto
	w.mu.Unlock()

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	guard, err := w.SpawnMob(8060, 1001)
	if err != nil {
		t.Fatalf("SpawnMob wall guard: %v", err)
	}
	if _, err := w.SpawnMob(8020, 1002); err != nil {
		t.Fatalf("SpawnMob church guard: %v", err)
	}
	clearMessages := func() {
		for name := range messages {
			delete(messages, name)
		}
	}
	clearMessages()

	return w, guard, oldRoom, observer, messages
}

func wallGuardGreeting() string {
	return "A city guard snaps to attention and salutes a church guard!\r\n" +
		"A city guard says, 'Hello gents!'\r\n" +
		"A church guard nods at a city guard.\r\n" +
		"A church guard says, 'On your way, soldier!'\r\n"
}

func TestSpecWallGuardNS_EntryGates(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*MobInstance, *Player)
		cmd   string
	}{
		{name: "command", cmd: "look"},
		{
			name: "sleeping",
			setup: func(guard *MobInstance, _ *Player) {
				guard.SetPosition(combat.PosSleeping)
			},
		},
		{
			name: "fighting",
			setup: func(guard *MobInstance, player *Player) {
				guard.SetFighting(player.GetName())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, guard, player, _, messages := newWallGuardTestWorld(t, false)
			if tc.setup != nil {
				tc.setup(guard, player)
			}
			if got := specWallGuardNS(w, player, guard, tc.cmd, ""); got {
				t.Fatal("gated wall guard invocation should return false")
			}
			if got := guard.GetRoom(); got != 1001 {
				t.Fatalf("gated wall guard moved to room %d, want 1001", got)
			}
			if got := messages[player.GetName()]; got != "" {
				t.Fatalf("gated invocation emitted %q", got)
			}
		})
	}
}

func TestSpecWallGuardNS_PatrolsBothOneWayEnds(t *testing.T) {
	w, guard, _, _, messages := newWallGuardTestWorld(t, true)

	if got := specWallGuardNS(w, nil, guard, "", ""); got {
		t.Fatal("commandless patrol should return false")
	}
	if got := guard.GetRoom(); got != 1002 {
		t.Fatalf("north-end patrol room = %d, want 1002", got)
	}
	if got := len(w.GetMobsInRoom(1001)); got != 0 {
		t.Fatalf("old room still contains %d wall guards", got)
	}
	if got := len(w.GetMobsInRoom(1002)); got != 2 {
		t.Fatalf("destination room mob count = %d, want wall and church guards", got)
	}
	if got := messages["OldRoom"]; got != "" {
		t.Fatalf("old-room witness saw destination greeting %q", got)
	}

	if got := specWallGuardNS(w, nil, guard, "", ""); got {
		t.Fatal("southbound patrol should return false")
	}
	if got := guard.GetRoom(); got != 1001 {
		t.Fatalf("south-end patrol room = %d, want 1001", got)
	}
}

func TestSpecWallGuardNS_GreetingAudienceAndPerCallReset(t *testing.T) {
	w, guard, oldRoom, observer, messages := newWallGuardTestWorld(t, false)

	if got := specWallGuardNS(w, nil, guard, "", ""); got {
		t.Fatal("eligible patrol should return false")
	}
	if got, want := messages[observer.GetName()], wallGuardGreeting(); got != want {
		t.Fatalf("destination greeting = %q, want %q", got, want)
	}
	if got := messages[oldRoom.GetName()]; got != "" {
		t.Fatalf("old-room audience received greeting %q", got)
	}

	for name := range messages {
		messages[name] = ""
	}
	if got := specWallGuardNS(w, nil, guard, "", ""); got {
		t.Fatal("second eligible patrol should return false")
	}
	if got, want := messages[observer.GetName()], wallGuardGreeting(); got != want {
		t.Fatalf("second destination greeting = %q, want %q", got, want)
	}
	if strings.Contains(messages[observer.GetName()], "someone") {
		t.Fatalf("greeting lost visible PERS names: %q", messages[observer.GetName()])
	}
}
