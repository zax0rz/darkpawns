package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecAlienElevatorMovesOnlyOpenEastRoom(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 19551, Name: "Lower Room", Zone: 1, Exits: map[string]parser.Exit{"east": {Direction: "east", ToRoom: 19599, ExitInfo: parser.ExitIsDoor | parser.ExitClosed}}},
		{VNum: 19599, Name: "Upper Room", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	caller := NewPlayer(1, "Caller", 19551)
	peer := NewPlayer(2, "Peer", 19551)
	untouched := NewPlayer(3, "Untouched", 19599)
	for _, p := range []*Player{caller, peer, untouched} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.GetName(), err)
		}
	}
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

	if specAlienElevator(w, caller, nil, "close", "door") {
		t.Fatal("closed east exit should leave the special unhandled")
	}
	if caller.GetRoomVNum() != 19551 || peer.GetRoomVNum() != 19551 {
		t.Fatal("closed east exit moved occupants")
	}
	if !w.SetExitInfo(19551, "east", parser.ExitIsDoor) {
		t.Fatal("SetExitInfo failed")
	}
	if !specAlienElevator(w, caller, nil, "close", "door") {
		t.Fatal("open east exit should handle the room movement")
	}
	if caller.GetRoomVNum() != 19599 || peer.GetRoomVNum() != 19599 {
		t.Fatalf("caller/peer rooms = %d/%d, want both 19599", caller.GetRoomVNum(), peer.GetRoomVNum())
	}
	if untouched.GetRoomVNum() != 19599 {
		t.Fatalf("occupant already in destination moved to %d", untouched.GetRoomVNum())
	}
	for _, name := range []string{"Caller", "Peer"} {
		if !strings.Contains(messages[name], "The room starts to move!\r\n") {
			t.Errorf("%s got %q, want the direct movement line", name, messages[name])
		}
	}
	if messages["Untouched"] != "" {
		t.Fatalf("destination occupant got %q", messages["Untouched"])
	}
}

func TestSpecBrassDragonUsesMoverRoomAndImmortalGate(t *testing.T) {
	w, ch, _ := newSpecProcTestWorld(t)
	ch.SetRoom(5065)
	me := newSpecProcTestMob(t, w, 1001, 10)

	if !specBrassDragon(w, ch, me, "west", "") {
		t.Fatal("mortal in room 5065 should be blocked even when the proc owner is elsewhere")
	}
	ch.SetLevel(LVL_IMMORT)
	if specBrassDragon(w, ch, me, "west", "") {
		t.Fatal("immortal mover should pass the brass dragon")
	}
}

func TestSpecConductorAsksTicketOnAutonomousPulse(t *testing.T) {
	w := newActTemplateWorld(18505)
	observer := addCapturedPlayer(t, w, "Ticketless", 18505)
	conductor := seatMob(t, w, &parser.Mob{VNum: 21208, Keywords: "conductor", ShortDesc: "the conductor", Int: 11, Wis: 11}, 18505)
	conductor.SetPosition(combat.PosStanding)
	conductor.SetAlignment(0)
	player := wPlayer(t, w, "Ticketless")
	player.SetAlignment(0)

	old := conductorNumber
	conductorNumber = func(from, to int) int { return 3 }
	t.Cleanup(func() { conductorNumber = old })
	if !specConductor(w, nil, conductor, "", "") {
		t.Fatal("first ticket check should be handled")
	}
	if conductor.GetTalk(1) != 1 {
		t.Fatalf("talk[1] = %d, want first ask recorded", conductor.GetTalk(1))
	}
	if !strings.Contains(observer.msgs, "Pardon me Ticketless, do you have your ticket?") {
		t.Fatalf("observer got %q, want the first ticket question", observer.msgs)
	}
}

func TestSpecElementalRoomAffectsEveryMortalOccupant(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Fire Room", Zone: 1, Sector: 0}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	mortal := NewPlayer(1, "Mortal", 1001)
	mortal.SetLevel(5)
	mortal.SetHealth(200)
	secondMortal := NewPlayer(3, "SecondMortal", 1001)
	secondMortal.SetLevel(10)
	secondMortal.SetHealth(250)
	immortal := NewPlayer(2, "Immortal", 1001)
	immortal.SetLevel(LVL_IMMORT)
	for _, p := range []*Player{mortal, secondMortal, immortal} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.GetName(), err)
		}
	}
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	roomProc := NewMob(&parser.Mob{VNum: 21209, Keywords: "elemental room", ShortDesc: "the elemental room"}, 1001)
	roomProc.SetLevel(LVL_IMMORT)

	if specElementalRoom(w, nil, roomProc, "", "") {
		t.Fatal("nonlethal pulse should continue normal mobile activity")
	}
	if got := mortal.GetHealth(); got != 100 {
		t.Fatalf("mortal health = %d, want 100", got)
	}
	if got := secondMortal.GetHealth(); got != 150 {
		t.Fatalf("second mortal health = %d, want 150", got)
	}
	if !strings.Contains(messages["Mortal"], "Your skin blackens as fire burns you alive...\n\r") || !strings.Contains(messages["Mortal"], "You are DYING!") {
		t.Fatalf("mortal got %q, want C's fire and dying lines", messages["Mortal"])
	}
	if !strings.Contains(messages["Immortal"], "You ignore the elements that barrage you.\n\r") {
		t.Fatalf("immortal got %q, want the ignore line", messages["Immortal"])
	}
	if !strings.Contains(messages["SecondMortal"], "Your skin blackens as fire burns you alive...\n\r") || !strings.Contains(messages["SecondMortal"], "You are DYING!") {
		t.Fatalf("second mortal got %q, want C's fire and dying lines", messages["SecondMortal"])
	}
}

func TestSpecEnterCircleCountsPlayersAndConsumesOtherLook(t *testing.T) {
	w := newActTemplateWorld(1001, portalRoom)
	mover := addCapturedPlayer(t, w, "Pracmover", 1001)
	addCapturedPlayer(t, w, "Portalpeer1", portalRoom)
	addCapturedPlayer(t, w, "Portalpeer2", portalRoom)
	actor := wPlayer(t, w, "Pracmover")

	if !specEnterCircle(w, actor, nil, "enter", "circle") {
		t.Fatal("crowded portal entry should be handled")
	}
	if actor.GetRoomVNum() != 1001 || !strings.Contains(mover.msgs, "You can't fit on the portal, it's too crowded.") {
		t.Fatalf("mover room/output = %d/%q, want crowd refusal", actor.GetRoomVNum(), mover.msgs)
	}
	mover.msgs = ""
	// C delegates non-circle looks to do_look, whose direction preface is an
	// observable line even when that room has no north exit
	// (src/spec_procs.c:1955-1962; src/act.informative.c:851-854).
	if !specEnterCircle(w, actor, nil, "look", "north") {
		t.Fatal("a non-circle look should be delegated to do_look and consumed")
	}
	if !strings.Contains(mover.msgs, "You look northwards.") {
		t.Fatalf("non-circle look output = %q, want C's direction-look line", mover.msgs)
	}
}

func TestSpecElevatorMovesOnlyFirstTwoCharacters(t *testing.T) {
	w := newActTemplateWorld(1001, portalRoom, elevatorDest)
	actorCapture := addCapturedPlayer(t, w, "RuneSpeaker", 1001)
	actor := wPlayer(t, w, "RuneSpeaker")
	actor.SetStat("INT", 11)
	actor.SetStat("WIS", 11)
	actor.CopyBaseAttributes()
	addCapturedPlayer(t, w, "OlderPortalGuest", portalRoom)
	addCapturedPlayer(t, w, "MiddlePortalGuest", portalRoom)
	addCapturedPlayer(t, w, "NewestPortalGuest", portalRoom)
	proc := NewMob(&parser.Mob{VNum: 21210, Keywords: "pillar", ShortDesc: "the pillar"}, 1001)

	if !specElevator(w, actor, proc, "say", "sumuni elementi avia elevata") {
		t.Fatal("the exact rune phrase should activate the elevator")
	}
	if actorCapture.msgs == "" {
		t.Fatal("rune speaker should receive the consumed say and portal messages")
	}
	if got := wPlayer(t, w, "NewestPortalGuest").GetRoomVNum(); got != elevatorDest {
		t.Fatalf("newest guest room = %d, want %d", got, elevatorDest)
	}
	if got := wPlayer(t, w, "MiddlePortalGuest").GetRoomVNum(); got != elevatorDest {
		t.Fatalf("middle guest room = %d, want %d", got, elevatorDest)
	}
	if got := wPlayer(t, w, "OlderPortalGuest").GetRoomVNum(); got != portalRoom {
		t.Fatalf("third guest room = %d, want %d", got, portalRoom)
	}
}

func TestSpecHiscDelegatesAutonomousPulseToCleric(t *testing.T) {
	w, _, _ := newSpecProcTestWorld(t)
	hisc := newSpecProcTestMob(t, w, 1001, 10)
	hisc.SetPosition(combat.PosResting)

	specHisc(w, nil, hisc, "", "ignored")
	if got := hisc.GetPosition(); got != combat.PosStanding {
		t.Fatalf("hisc position after cleric delegation = %d, want standing (%d)", got, combat.PosStanding)
	}
}

func TestSpecMiniThiefUsesCChanceAndAmountBands(t *testing.T) {
	for _, tc := range []struct {
		name          string
		skillRoll     int
		amountRoll    int
		gold          int
		wantVictim    int
		wantThief     int
		wantOutput    string
		wantNoOutput  bool
		wantDrawRange [][2]int
	}{
		{
			name: "unnoticed theft", skillRoll: 5, amountRoll: 20, gold: 100,
			wantVictim: 80, wantThief: 20, wantNoOutput: true,
			wantDrawRange: [][2]int{{0, 19}, {1, 100}, {10, 25}},
		},
		{
			name: "small purse cap", skillRoll: 5, amountRoll: 20, gold: 30,
			wantVictim: 24, wantThief: 6, wantNoOutput: true,
			wantDrawRange: [][2]int{{0, 19}, {1, 100}, {10, 25}},
		},
		{
			name: "caught after successful theft", skillRoll: 20, amountRoll: 20, gold: 100,
			wantVictim: 80, wantThief: 20,
			wantOutput: "You catch a test mob's hand coming out of your coin purse.\r\n" +
				"Tester catchs a test mob's hand leaving thier coin purse a little lighter.\r\n",
			wantDrawRange: [][2]int{{0, 19}, {1, 100}, {10, 25}},
		},
		{
			name: "caught before theft", skillRoll: 50, gold: 100,
			wantVictim: 100,
			wantOutput: "You catch a test mob's hand going into your coin purse.\r\n" +
				"Tester catchs a test mob's hand entering thier coin purse.\r\n",
			wantDrawRange: [][2]int{{0, 19}, {1, 100}},
		},
		{
			name: "uncaught empty hand", skillRoll: 80, gold: 100,
			wantVictim: 100, wantNoOutput: true,
			wantDrawRange: [][2]int{{0, 19}, {1, 100}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, victim, lastMsg := newSpecProcTestWorld(t)
			victim.SetLevel(5)
			victim.SetGold(tc.gold)
			thief := newSpecProcTestMob(t, w, victim.GetRoomVNum(), 10)
			thief.SetGold(0)

			old := miniThiefNumber
			values := []int{0, tc.skillRoll}
			if tc.skillRoll < 35 {
				values = append(values, tc.amountRoll)
			}
			var ranges [][2]int
			miniThiefNumber = func(from, to int) int {
				ranges = append(ranges, [2]int{from, to})
				value := values[0]
				values = values[1:]
				return value
			}
			t.Cleanup(func() { miniThiefNumber = old })

			if !specMiniThief(w, nil, thief, "", "") {
				t.Fatal("selected theft pulse should be handled")
			}
			if got := victim.GetGold(); got != tc.wantVictim {
				t.Fatalf("victim gold = %d, want %d", got, tc.wantVictim)
			}
			if got := thief.GetGold(); got != tc.wantThief {
				t.Fatalf("thief gold = %d, want %d", got, tc.wantThief)
			}
			if len(ranges) != len(tc.wantDrawRange) {
				t.Fatalf("draw ranges = %v, want %v", ranges, tc.wantDrawRange)
			}
			for i := range tc.wantDrawRange {
				if ranges[i] != tc.wantDrawRange[i] {
					t.Fatalf("draw ranges = %v, want %v", ranges, tc.wantDrawRange)
				}
			}
			gotOutput := lastMsg()
			if tc.wantNoOutput && gotOutput != "" {
				t.Fatalf("output = %q, want silence", gotOutput)
			}
			if tc.wantOutput != "" && gotOutput != tc.wantOutput {
				t.Fatalf("output = %q, want exact C transcript %q", gotOutput, tc.wantOutput)
			}
		})
	}
}

func TestSpecMayorWakesAndFallsThroughToMobileActivity(t *testing.T) {
	w := newActTemplateWorld(1001)
	observer := addCapturedPlayer(t, w, "Mayorobserver", 1001)
	mayor := seatMob(t, w, &parser.Mob{VNum: 21211, Keywords: "mayor", ShortDesc: "the mayor"}, 1001)
	mayor.SetPosition(combat.PosSleeping)

	oldTime, oldStates := timeInfo, mayorStates
	timeInfo.Hours = 6
	mayorStates = make(map[int]*mayorState)
	t.Cleanup(func() { timeInfo, mayorStates = oldTime, oldStates })
	if specMayor(w, nil, mayor, "", "") {
		t.Fatal("mayor path action returns FALSE so normal mobile activity continues")
	}
	if mayor.GetPosition() != combat.PosStanding {
		t.Fatalf("mayor position = %d, want standing", mayor.GetPosition())
	}
	if observer.msgs != "The mayor awakens and groans loudly.\r\n" {
		t.Fatalf("observer got %q, want C's exact wake act", observer.msgs)
	}
}
