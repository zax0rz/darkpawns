package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestStartRoomWoundedPlayerGetsStars pins DP-1552: C's SPECIAL(start_room)
// calls the full do_look (src/spec_procs.c:2254), whose position gate
// (src/act.informative.c:1107) prints stars below POS_SLEEPING. The port used
// to call lookAtRoom directly and showed the full destination room.
func TestStartRoomWoundedPlayerGetsStars(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: NewbieStartRoom, Name: "Burning Hut"},
		parser.Room{VNum: 8162, Name: "Temple Infirmary", Zone: 1},
	)
	// NewbieHometownRoom(KD=1) is the infirmary; hometown 1 reaches it.
	player := NewPlayer(1, "Wounded", NewbieStartRoom)
	player.Hometown = 1
	player.SetPosition(combat.PosMortally)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if !specStartRoomBody(w, NewbieStartRoom) {
		t.Fatal("start_room declined for a wounded mortal")
	}
	got := messages[player.GetName()]
	if !strings.Contains(got, "You can't see anything but stars!") {
		t.Fatalf("wounded start_room look = %q, want the stars line", got)
	}
	if strings.Contains(got, "Temple Infirmary") {
		t.Fatalf("wounded start_room look leaked the room: %q", got)
	}
}

// TestStartRoomBlindPlayerGetsDamned pins the second do_look gate: a blind
// arrival hears "You can't see a damned thing, you're blind!"
// (src/act.informative.c:1112), not look_at_room's infinite-darkness line.
func TestStartRoomBlindPlayerGetsDamned(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: NewbieStartRoom, Name: "Burning Hut"},
		parser.Room{VNum: 8162, Name: "Temple Infirmary", Zone: 1},
	)
	player := NewPlayer(1, "Blindarr", NewbieStartRoom)
	player.Hometown = 1
	player.SetPosition(combat.PosStanding)
	player.SetAffect(affBlind, true)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if !specStartRoomBody(w, NewbieStartRoom) {
		t.Fatal("start_room declined for a blind mortal")
	}
	got := messages[player.GetName()]
	if !strings.Contains(got, "You're blind!") && !strings.Contains(got, "you're blind!") {
		t.Fatalf("blind start_room look = %q, want the blind line", got)
	}
	if strings.Contains(got, "Temple Infirmary") {
		t.Fatalf("blind start_room look leaked the room: %q", got)
	}
}

// TestDoLookForSpellGates pins the R5c class for spell_mindsight
// (src/spells.c:942): the bridge must run the full do_look gates, not the
// bare look_at_room spell landing.
func TestDoLookForSpellGates(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: 1001, Name: "Look Room", Zone: 1},
	)
	ch := NewPlayer(1, "WoundedMind", 1001)
	ch.SetPosition(combat.PosMortally)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.DoLookForSpell(ch)
	if got := messages[ch.GetName()]; !strings.Contains(got, "You can't see anything but stars!") {
		t.Fatalf("DoLookForSpell wounded = %q, want stars", got)
	}
	if strings.Contains(messages[ch.GetName()], "Look Room") {
		t.Fatalf("DoLookForSpell leaked the room: %q", messages[ch.GetName()])
	}
	// Blind with an awake position gets the blind line, not darkness.
	ch2 := NewPlayer(2, "BlindMind", 1001)
	ch2.SetPosition(combat.PosStanding)
	ch2.SetAffect(affBlind, true)
	if err := w.AddPlayer(ch2); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.DoLookForSpell(ch2)
	if got := messages[ch2.GetName()]; !strings.Contains(got, "you're blind!") {
		t.Fatalf("DoLookForSpell blind = %q, want blind line", got)
	}
}

// TestPortalToTempleAndItohUseDoLook pins the R5c class: both specials call
// do_look (src/spec_procs3.c:537, 606), so a sub-sleeping arrival gets stars
// without revealing the destination room; C's position gate returns before
// room output (src/act.informative.c:1106-1108).
func TestPortalToTempleAndItohUseDoLook(t *testing.T) {
	cases := []struct {
		name string
		call func(w *World, ch *Player) bool
	}{
		{"portal", func(w *World, ch *Player) bool { return specPortalToTemple(w, ch, nil, "say", "setchswayno") }},
		{"itoh", func(w *World, ch *Player) bool { return specItoh(w, ch, nil, "say", "itoh") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, messages := newActorRoomTestWorld(t,
				parser.Room{VNum: 19658, Name: "Portal Room", Zone: 1},
				parser.Room{VNum: 8008, Name: "Temple"},
				parser.Room{VNum: 19875, Name: "Itoh"},
			)
			ch := NewPlayer(1, "WoundedPortal", 19658)
			ch.SetPosition(combat.PosMortally)
			if err := w.AddPlayer(ch); err != nil {
				t.Fatalf("AddPlayer: %v", err)
			}
			if !tc.call(w, ch) {
				t.Fatal("special declined")
			}
			got := messages[ch.GetName()]
			if !strings.Contains(got, "You can't see anything but stars!") {
				t.Fatalf("sub-sleeping %s look = %q, want stars", tc.name, got)
			}
			roomName := "Temple"
			if tc.name == "itoh" {
				roomName = "Itoh"
			}
			if strings.Contains(got, roomName) {
				t.Fatalf("sub-sleeping %s look leaked destination room %q: %q", tc.name, roomName, got)
			}
		})
	}
}
