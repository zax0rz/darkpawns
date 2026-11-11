package game

// Pins the act()-template rendering this PR restored: every $-code line these
// specs emit must come out substituted, never literal. The MpSound cases are
// live player-facing bytes (mobile_activity → mp_sound); the spec_procs4
// bodies are ports of C specials that src/spec_assign.c never assigns, so
// their proof is these direct calls, not the census. Reverting any conversion
// back to roomMessage/SendMessage fails its case here (R5h).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newActTemplateWorld builds a world whose snapshot carries the given rooms so
// occupancy and look-ups resolve in them.
func newActTemplateWorld(rooms ...int) *World {
	snapshots := NewSnapshotManager()
	published := map[int]*parser.Room{}
	for _, vnum := range rooms {
		published[vnum] = &parser.Room{VNum: vnum, Name: "Act Template Room", Zone: 1}
	}
	snapshots.Publish(published)
	return &World{
		snapshots:  snapshots,
		rooms:      published,
		mobs:       make(map[int]*parser.Mob),
		players:    make(map[string]*Player),
		activeMobs: make(map[int]*MobInstance),
		roomItems:  make(map[int][]*ObjectInstance),
	}
}

type capturedPlayer struct {
	name string
	msgs string
}

// addCapturedPlayer seats a player in room vnum and captures everything the
// world sends them.
func addCapturedPlayer(t *testing.T, w *World, name string, room int) *capturedPlayer {
	t.Helper()
	p := NewPlayer(1, name, room)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	cp := &capturedPlayer{name: name}
	previous := w.MessageSink
	w.MessageSink = func(playerName string, msg []byte) {
		if previous != nil {
			previous(playerName, msg)
		}
		if playerName == name {
			cp.msgs += string(msg)
		}
	}
	t.Cleanup(func() { w.MessageSink = previous })
	return cp
}

// forceMobProgDraw pins every mobProgNumber call to one value for a test.
func forceMobProgDraw(t *testing.T, value int) {
	t.Helper()
	old := mobProgNumber
	mobProgNumber = func(from, to int) int { return value }
	t.Cleanup(func() { mobProgNumber = old })
}

// seatMob registers the proto and spawns it into the room so world occupancy
// queries (GetMobsInRoom, Act's room iteration) see it.
func seatMob(t *testing.T, w *World, proto *parser.Mob, room int) *MobInstance {
	t.Helper()
	w.mobs[proto.VNum] = proto
	mob, err := w.SpawnMob(proto.VNum, room)
	if err != nil {
		t.Fatalf("SpawnMob(%d): %v", proto.VNum, err)
	}
	return mob
}

func TestMpSoundRendersSayLineThroughAct(t *testing.T) {
	// The reported bug: the petitioner's line arrived as "$n says, '...'".
	// C renders it through do_say's act(): the '!' ending makes it "exclaims"
	// (src/act.comm.c:787-790; mp_sound delivery at src/mobprog.c:604).
	w := newActTemplateWorld(1001)
	observer := addCapturedPlayer(t, w, "Pracobserver", 1001)
	petitioner := NewMob(&parser.Mob{VNum: 8066, Keywords: "petitioner", ShortDesc: "a petitioner", Int: 11, Wis: 11}, 1001)
	forceMobProgDraw(t, 1) // C's truthy number(0,1) selects case 8066 arm 1

	w.MpSound(petitioner)

	want := "A petitioner exclaims, 'Sign this, please! There's too much violence!'\r\n"
	if !strings.Contains(observer.msgs, want) {
		t.Errorf("observer got %q, want line %q", observer.msgs, want)
	}
	if strings.Contains(observer.msgs, "$n") {
		t.Errorf("observer got literal $-code in %q", observer.msgs)
	}
}

func TestMpSoundRendersEmoteLineThroughAct(t *testing.T) {
	w := newActTemplateWorld(1001)
	observer := addCapturedPlayer(t, w, "Pracobserver", 1001)
	carpenter := NewMob(&parser.Mob{VNum: 8067, Keywords: "carpenter", ShortDesc: "a carpenter", Int: 11, Wis: 11}, 1001)
	forceMobProgDraw(t, 1) // C's truthy number(0,1) selects case 8067 arm 1

	w.MpSound(carpenter)

	want := "A carpenter adjusts his tool belt.\r\n"
	if !strings.Contains(observer.msgs, want) {
		t.Errorf("observer got %q, want line %q", observer.msgs, want)
	}
	if strings.Contains(observer.msgs, "$n") {
		t.Errorf("observer got literal $-code in %q", observer.msgs)
	}
}

func TestMpSoundAversinEmotesBeforeSaying(t *testing.T) {
	// C case 8059: do_echo(mob, "looks at you.", emote, SCMD_EMOTE) before
	// the say (src/mobprog.c:582-584). Case 8059 draws no mana of the RNG.
	w := newActTemplateWorld(1001)
	observer := addCapturedPlayer(t, w, "Pracobserver", 1001)
	aversin := NewMob(&parser.Mob{VNum: 8059, Keywords: "aversin", ShortDesc: "the captain", Int: 11, Wis: 11}, 1001)

	w.MpSound(aversin)

	// "Carry on, citizen." ends in '.', so C's do_say renders "states"
	// (src/act.comm.c:793-795).
	for _, want := range []string{
		"The captain looks at you.\r\n",
		"The captain states, 'Carry on, citizen.'\r\n",
	} {
		if !strings.Contains(observer.msgs, want) {
			t.Errorf("observer got %q, want line %q", observer.msgs, want)
		}
	}
}

func TestNpcRescueRendersActLines(t *testing.T) {
	// C npc_rescue (src/mobprog.c:445-448): the victim hears "You are rescued
	// by $N..." TO_CHAR and the room hears "$n heroically rescues $N."
	// TO_NOTVICT — so the victim must not also get the room line.
	w := newActTemplateWorld(1001)
	victim := addCapturedPlayer(t, w, "Pracvictim", 1001)
	roomMate := addCapturedPlayer(t, w, "Pracmate", 1001)
	hero := seatMob(t, w, &parser.Mob{VNum: 8059, Keywords: "guard", ShortDesc: "a gruff guard"}, 1001)
	badGuy := seatMob(t, w, &parser.Mob{VNum: 8062, Keywords: "thug", ShortDesc: "a thug"}, 1001)
	badGuy.SetFightingBody(victimName(t, w, "Pracvictim"))

	if !w.NpcRescue(hero, wPlayer(t, w, "Pracvictim")) {
		t.Fatal("NpcRescue = false, want true")
	}

	wantVictim := "You are rescued by a gruff guard, your loyal friend!\r\n\r\n"
	if !strings.Contains(victim.msgs, wantVictim) {
		t.Errorf("victim got %q, want line %q", victim.msgs, wantVictim)
	}
	wantRoom := "A gruff guard heroically rescues Pracvictim.\r\n"
	if !strings.Contains(roomMate.msgs, wantRoom) {
		t.Errorf("room mate got %q, want line %q", roomMate.msgs, wantRoom)
	}
	if strings.Contains(victim.msgs, "heroically rescues") {
		t.Errorf("victim also received the TO_NOTVICT line: %q", victim.msgs)
	}
	if strings.Contains(roomMate.msgs+victim.msgs, "$n") || strings.Contains(roomMate.msgs+victim.msgs, "$N") {
		t.Errorf("literal $-code in rescue output: %q / %q", victim.msgs, roomMate.msgs)
	}
}

func TestSpecBrassDragonBlocksThroughAct(t *testing.T) {
	// C: act("The brass dragon humiliates $n, and blocks $s way.", FALSE, ch,
	// 0, 0, TO_ROOM) + the mover's \n\r self line (src/spec_procs.c:1750-1753).
	w := newActTemplateWorld(5065)
	mover := addCapturedPlayer(t, w, "Pracmover", 5065)
	roomMate := addCapturedPlayer(t, w, "Pracmate", 5065)
	dragon := NewMob(&parser.Mob{VNum: 8058, Keywords: "dragon", ShortDesc: "the brass dragon"}, 5065)

	if !specBrassDragon(w, wPlayer(t, w, "Pracmover"), dragon, "west", "") {
		t.Fatal("specBrassDragon = false, want true")
	}

	wantSelf := "The brass dragon humiliates you, and blocks your way.\n\r"
	if !strings.Contains(mover.msgs, wantSelf) {
		t.Errorf("mover got %q, want C's reversed-terminator line %q", mover.msgs, wantSelf)
	}
	wantRoom := "The brass dragon humiliates Pracmover, and blocks his way.\r\n"
	if !strings.Contains(roomMate.msgs, wantRoom) {
		t.Errorf("room mate got %q, want line %q", roomMate.msgs, wantRoom)
	}
}

func TestSpecConductorFightLinesThroughAct(t *testing.T) {
	// Every fight line goes through act(..., TO_ROOM)
	// (src/spec_procs.c:1643-1663); case 4 keeps C's double space.
	w := newActTemplateWorld(18505)
	observer := addCapturedPlayer(t, w, "Pracobserver", 18505)
	conductor := seatMob(t, w, &parser.Mob{VNum: 21207, Keywords: "conductor", ShortDesc: "the conductor"}, 18505)
	conductor.SetFightingBody(observerName(t, w, "Pracobserver"))
	conductor.SetPosition(combat.PosStanding)

	old := conductorNumber
	for _, tc := range []struct {
		draw int
		want string
	}{
		{1, "The conductor shouts, 'I said give me your ticket!'\r\n"},
		{4, "The conductor shouts 'Security!  Help me with this piece of garbage!'\r\n"},
	} {
		conductorNumber = func(from, to int) int { return tc.draw }
		// The port's gate runs this arm for command dispatch (C runs it on
		// pulses; the conductor special is unassigned in both engines).
		specConductor(w, nil, conductor, "look", "")
		if !strings.Contains(observer.msgs, tc.want) {
			t.Errorf("draw %d: observer got %q, want line %q", tc.draw, observer.msgs, tc.want)
		}
		if strings.Contains(observer.msgs, "$n") {
			t.Errorf("draw %d: literal $n in %q", tc.draw, observer.msgs)
		}
	}
	conductorNumber = old
}

func TestSpecEnterCircleThroughAct(t *testing.T) {
	// C: send_to_char("You stand in the circle.\n\r") then act("$n enters the
	// circle ... view of $m!", 1, ch, 0, 0, TO_ROOM) (src/spec_procs.c:1945-1947).
	w := newActTemplateWorld(1001, portalRoom)
	mover := addCapturedPlayer(t, w, "Pracmover", 1001)
	roomMate := addCapturedPlayer(t, w, "Pracmate", 1001)
	pillar := NewMob(&parser.Mob{VNum: 8058, Keywords: "pillar", ShortDesc: "the pillar"}, 1001)

	if !specEnterCircle(w, wPlayer(t, w, "Pracmover"), pillar, "enter", "circle") {
		t.Fatal("specEnterCircle = false, want true")
	}

	wantSelf := "You stand in the circle.\n\r"
	if !strings.Contains(mover.msgs, wantSelf) {
		t.Errorf("mover got %q, want C's reversed-terminator line %q", mover.msgs, wantSelf)
	}
	wantRoom := "Pracmover enters the circle which suddenly starts glowing brightly, obscuring your view of him!\r\n"
	if !strings.Contains(roomMate.msgs, wantRoom) {
		t.Errorf("room mate got %q, want line %q", roomMate.msgs, wantRoom)
	}
}

func TestSpecElevatorThroughAct(t *testing.T) {
	// C: the rune say triggers the self line with \n\r\n\r and the room line
	// through act() whose format keeps that tail (src/spec_procs.c:1999-2003).
	w := newActTemplateWorld(1001, portalRoom, elevatorDest)
	mover := addCapturedPlayer(t, w, "Pracmover", 1001)
	roomMate := addCapturedPlayer(t, w, "Pracmate", 1001)
	pillar := NewMob(&parser.Mob{VNum: 8058, Keywords: "pillar", ShortDesc: "the pillar"}, 1001)

	if !specElevator(w, wPlayer(t, w, "Pracmover"), pillar, "say", "sumuni elementi avia elevata") {
		t.Fatal("specElevator = false, want true")
	}

	wantSelf := "The portal begins to rise, lifted by the air elemental summoned by your rune!\n\r\n\r"
	if !strings.Contains(mover.msgs, wantSelf) {
		t.Errorf("mover got %q, want C's reversed-terminator line %q", mover.msgs, wantSelf)
	}
	wantRoom := "The portal begins to rise, lifted by the air elemental summoned by Pracmover!\r\n\r\n\r\n"
	if !strings.Contains(roomMate.msgs, wantRoom) {
		t.Errorf("room mate got %q, want line %q (act() appends its own terminator)", roomMate.msgs, wantRoom)
	}
}

// victimName / observerName / wPlayer resolve the combat.Player adapter the
// fighting-body and spec calls need from a seated player.
func victimName(t *testing.T, w *World, name string) combat.Combatant {
	t.Helper()
	return wPlayer(t, w, name)
}

func observerName(t *testing.T, w *World, name string) combat.Combatant {
	t.Helper()
	return wPlayer(t, w, name)
}

func wPlayer(t *testing.T, w *World, name string) *Player {
	t.Helper()
	p := w.players[name]
	if p == nil {
		t.Fatalf("player %s not seated", name)
	}
	return p
}
