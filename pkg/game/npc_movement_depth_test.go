package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// npcMovementWorld builds two rooms (1001 Hall north of 1002 Yard) with a
// spawned trainee in 1001 and per-player output capture. The exit map is
// editable through the returned room pointers before dispatch.
func npcMovementWorld(t *testing.T) (*World, *MobInstance, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	hall := parser.Room{VNum: 1001, Name: "Hall", Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}}
	yard := parser.Room{VNum: 1002, Name: "Yard", Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}}
	parsed := &parser.World{
		Rooms: []parser.Room{hall, yard},
		Mobs: []parser.Mob{{
			VNum: 99051, Keywords: "guard trainee", ShortDesc: "a guard trainee",
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	watcher := NewPlayer(1, "Watcher", 1002)
	watcher.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatal(err)
	}
	local := NewPlayer(2, "Local", 1001)
	local.SetPosition(combat.PosStanding)
	local.SetMove(100)
	local.SetMaxMove(100)
	if err := w.AddPlayer(local); err != nil {
		t.Fatal(err)
	}
	actor, err := w.SpawnMob(99051, 1001)
	if err != nil {
		t.Fatal(err)
	}
	actor.SetPosition(combat.PosStanding)
	out := map[string]*strings.Builder{"Watcher": {}, "Local": {}}
	w.MessageSink = func(name string, message []byte) {
		if b, ok := out[name]; ok {
			b.Write(message)
		}
	}
	return w, actor, local, watcher, out
}

// TestNPCPerformMoveLeaveAndArrivalBytes pins perform_move's leave and arrival
// audiences for the mobile lane (src/act.movement.c:181-183, 210-235): the
// origin room hears "$n leaves north." and the destination room hears
// "$n arrives from the south."
func TestNPCPerformMoveLeaveAndArrivalBytes(t *testing.T) {
	w, actor, _, _, out := npcMovementWorld(t)

	if !w.performMoveMob(actor, 0, false) {
		t.Fatal("perform_move north failed")
	}
	if got := actor.GetRoom(); got != 1002 {
		t.Fatalf("NPC room = %d, want 1002", got)
	}
	if got, want := out["Local"].String(), "A guard trainee leaves north.\r\n"; got != want {
		t.Fatalf("origin bytes = %q, want %q", got, want)
	}
	if got, want := out["Watcher"].String(), "A guard trainee arrives from the south.\r\n"; got != want {
		t.Fatalf("destination bytes = %q, want %q", got, want)
	}
}

// TestNPCPerformMoveClosedDoorRefusals pins perform_move's three closed-door
// refusals for a mobile mover (src/act.movement.c:322-336): the TO_CHAR text
// is observable through the mobile message sink (a switched body's session
// renders it), and the move fails without the do_simple_move leg (no leave
// act, body stays).
func TestNPCPerformMoveClosedDoorRefusals(t *testing.T) {
	for _, tc := range []struct {
		name string
		exit parser.Exit
		want string
	}{
		{name: "named door", exit: parser.Exit{ToRoom: 1002, ExitInfo: parser.ExitClosed, Keywords: "gate"}, want: "The gate seems to be closed.\r\n"},
		{name: "unnamed door", exit: parser.Exit{ToRoom: 1002, ExitInfo: parser.ExitClosed}, want: "It seems to be closed.\r\n"},
		{name: "secret door", exit: parser.Exit{ToRoom: 1002, ExitInfo: parser.ExitClosed, Keywords: "secret hatch"}, want: "Alas, you cannot go that way...\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, _, out := npcMovementWorld(t)
			var mobOut strings.Builder
			w.MobileMessageSink = func(_ *MobInstance, message []byte) {
				mobOut.Write(message)
			}
			w.mu.Lock()
			room := w.rooms[1001]
			exits := map[string]parser.Exit{"north": tc.exit}
			room.Exits = exits
			w.mu.Unlock()

			if w.performMoveMob(actor, 0, false) {
				t.Fatal("closed-door move succeeded")
			}
			if got := actor.GetRoom(); got != 1001 {
				t.Fatalf("NPC room = %d, want refusal to keep 1001", got)
			}
			if got := mobOut.String(); got != tc.want {
				t.Fatalf("mover bytes = %q, want %q", got, tc.want)
			}
			if got := out["Local"].String(); got != "" {
				t.Fatalf("origin saw leave bytes for a refused move: %q", got)
			}
			if got := out["Watcher"].String(); got != "" {
				t.Fatalf("destination saw arrival bytes for a refused move: %q", got)
			}
		})
	}
}

// TestNPCPerformMoveNoExitFailsQuietly pins the missing-exit refusal
// (src/act.movement.c:318-319): "Alas, you cannot go that way..." goes to a
// descriptor-less mover's send_to_char — dropped — and the body stays.
func TestNPCPerformMoveNoExitFailsQuietly(t *testing.T) {
	w, actor, _, _, out := npcMovementWorld(t)
	w.mu.Lock()
	delete(w.rooms[1001].Exits, "north")
	w.mu.Unlock()

	if w.performMoveMob(actor, 0, false) {
		t.Fatal("missing-exit move succeeded")
	}
	if got := actor.GetRoom(); got != 1001 {
		t.Fatalf("NPC room = %d, want 1001", got)
	}
	for who, b := range out {
		if b.String() != "" {
			t.Fatalf("%s saw bytes for a refused move: %q", who, b.String())
		}
	}
}

// TestNPCPerformMoveDragsPlayerFollower pins perform_move's follower drag for
// a mobile mover (src/act.movement.c:341-352): a same-room standing player
// follower receives "You follow $N." and is moved by the recursive
// perform_move, whose specials check runs for the follower alone.
func TestNPCPerformMoveDragsPlayerFollower(t *testing.T) {
	w, actor, local, _, out := npcMovementWorld(t)
	if err := w.SetFollower(local.Name, actor.GetName(), false); err != nil {
		t.Fatal(err)
	}

	if !w.performMoveMob(actor, 0, false) {
		t.Fatal("perform_move north failed")
	}
	if got := actor.GetRoom(); got != 1002 {
		t.Fatalf("NPC room = %d, want 1002", got)
	}
	if got := local.GetRoom(); got != 1002 {
		t.Fatalf("follower room = %d, want dragged to 1002", got)
	}
	if got := out["Local"].String(); !strings.Contains(got, "You follow a guard trainee.") {
		t.Fatalf("follower bytes = %q, want the follow notice", got)
	}
	if got := out["Watcher"].String(); !strings.Contains(got, "A guard trainee arrives from the south.") ||
		!strings.Contains(got, "Local arrives from the south.") {
		t.Fatalf("destination bytes = %q, want both arrivals in C order", got)
	}
}

// TestNPCSimpleMoveCharmedRefusal pins do_simple_move's charm gate for the
// mobile lane (src/act.movement.c:121-125): a charmed mover whose master
// shares its room stays put; the weep line is TO_CHAR and reaches no one.
func TestNPCSimpleMoveCharmedRefusal(t *testing.T) {
	w, actor, local, _, out := npcMovementWorld(t)
	actor.SetAffected(affCharm)
	actor.SetFollowingBody(local)

	if w.doSimpleMoveMob(actor, 0, false) {
		t.Fatal("charmed move succeeded")
	}
	if got := actor.GetRoom(); got != 1001 {
		t.Fatalf("NPC room = %d, want 1001", got)
	}
	if got := out["Local"].String(); got != "" {
		t.Fatalf("origin saw bytes for a charmed refusal: %q", got)
	}
}

// TestNPCSimpleMoveBoatRequirement pins do_simple_move's boat gate for the
// mobile lane (src/act.movement.c:127-135): water_noswim on either side
// without a boat refuses the move (has_boat, act.movement.c:59-79).
func TestNPCSimpleMoveBoatRequirement(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fromSect  int
		toSect    int
		boat      bool
		wantMoved bool
	}{
		{name: "water without boat", fromSect: SECT_WATER_NOSWIM, toSect: SECT_FIELD, boat: false, wantMoved: false},
		{name: "water with boat", fromSect: SECT_WATER_NOSWIM, toSect: SECT_FIELD, boat: true, wantMoved: true},
		{name: "land to land", fromSect: SECT_FIELD, toSect: SECT_FIELD, boat: false, wantMoved: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, _, out := npcMovementWorld(t)
			w.mu.Lock()
			w.rooms[1001].Sector = tc.fromSect
			w.rooms[1002].Sector = tc.toSect
			w.mu.Unlock()
			if tc.boat {
				boat := NewObjectInstance(&parser.Obj{
					VNum: 99052, Keywords: "raft", ShortDesc: "a raft", TypeFlag: ITEM_BOAT,
				}, -1)
				boat.SetWeight(0)
				registerTransferObject(w, boat)
				actor.AddToInventory(boat)
			}

			moved := w.doSimpleMoveMob(actor, 0, false)
			if moved != tc.wantMoved {
				t.Fatalf("moved = %v, want %v", moved, tc.wantMoved)
			}
			if got, want := actor.GetRoom(), map[bool]int{true: 1002, false: 1001}[tc.wantMoved]; got != want {
				t.Fatalf("NPC room = %d, want %d", got, want)
			}
			if !tc.wantMoved && out["Watcher"].String() != "" {
				t.Fatalf("destination saw bytes for a refused move: %q", out["Watcher"].String())
			}
		})
	}
}

// TestNPCSimpleMoveTunnelLimit pins do_simple_move's tunnel gate for the
// mobile lane (src/act.movement.c:170-173): ROOM_TUNNEL with one player
// already inside refuses the move.
func TestNPCSimpleMoveTunnelLimit(t *testing.T) {
	w, actor, _, watcher, _ := npcMovementWorld(t)
	setRoomFlagBit(w.rooms[1002], roomFlagTunnel)

	if w.doSimpleMoveMob(actor, 0, false) {
		t.Fatal("tunnel move succeeded")
	}
	if got := actor.GetRoom(); got != 1001 {
		t.Fatalf("NPC room = %d, want 1001", got)
	}
	if got := watcher.GetRoom(); got != 1002 {
		t.Fatalf("watcher room = %d, want undisturbed 1002", got)
	}
}
