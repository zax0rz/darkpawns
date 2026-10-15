package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// visibilityLightWorld builds a two-player room and captures each player's
// bytes so the LIGHT_OK matrix can assert act() rendering.
func visibilityLightWorld(t *testing.T, dark bool) (*World, *Player, *Player, *strings.Builder, *strings.Builder) {
	t.Helper()
	w, observer := newCombatTestWorld(t)
	if dark && !w.SetRoomFlagBit(1001, 0) {
		t.Fatal("SetRoomFlagBit failed")
	}
	actor := NewPlayer(2, "Actor", 1001)
	actor.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	var obsOut, actOut strings.Builder
	sink := w.MessageSink
	w.MessageSink = func(name string, msg []byte) {
		if sink != nil {
			sink(name, msg)
		}
		switch name {
		case observer.Name:
			obsOut.Write(msg)
		case actor.Name:
			actOut.Write(msg)
		}
	}
	return w, observer, actor, &obsOut, &actOut
}

// TestCanSeeLightOKMatrix pins DP-1295's matrix on the shared CAN_SEE
// predicate (src/utils.h:515-530): a dark room hides every body from a plain
// observer — mortal or immortal without holylight — while infravision and
// PRF_HOLYLIGHT restore sight, and a lit room never gates.
func TestCanSeeLightOKMatrix(t *testing.T) {
	for _, tc := range []struct {
		name        string
		dark        bool
		infravision bool
		holylight   bool
		immortal    bool
		wantSee     bool
	}{
		{name: "lit room mortal", wantSee: true},
		{name: "dark room mortal", dark: true},
		{name: "dark room immortal without holylight", dark: true, immortal: true},
		{name: "dark room immortal with holylight", dark: true, immortal: true, holylight: true, wantSee: true},
		{name: "dark room infravision", dark: true, infravision: true, wantSee: true},
		{name: "lit room infravision", infravision: true, wantSee: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, observer, actor, obsOut, _ := visibilityLightWorld(t, tc.dark)
			if tc.immortal {
				observer.SetLevel(LVL_IMMORT)
			}
			if tc.holylight {
				observer.SetHolyLight(true)
			}
			if tc.infravision {
				observer.SetAffect(affInfravision, true)
			}

			// A hide_invisible TO_ROOM act reaches the observer only when
			// they can see the actor (comm.c:2537-2538).
			Act(w, true, actor, nil, nil, nil, "$n waves at you.", "", ToRoom)
			if got := obsOut.String(); tc.wantSee && got == "" {
				t.Fatal("observer who can see received no act bytes")
			} else if !tc.wantSee && got != "" {
				t.Fatalf("observer received hidden act bytes: %q", got)
			}

			// PERS resolves "someone" for a non-hide rendering.
			obsOut.Reset()
			if got := persName(actor, observer); tc.wantSee && got != actor.GetName() {
				t.Fatalf("PERS = %q, want the name", got)
			} else if !tc.wantSee && got != "someone" {
				t.Fatalf("PERS = %q, want someone", got)
			}

			// Targeting flows through the same predicate (get_char_room_vis).
			_, ok := w.ResolveCharInRoom(observer, "Actor")
			if ok != tc.wantSee {
				t.Fatalf("room targeting = %v, want %v", ok, tc.wantSee)
			}
		})
	}
}

// TestCanSeeObjectDarkRoom pins CAN_SEE_OBJ's LIGHT_OK arm
// (src/utils.h:538-541): a dark room hides objects from a plain observer;
// infravision and holylight restore them.
func TestCanSeeObjectDarkRoom(t *testing.T) {
	w, observer, _, _, _ := visibilityLightWorld(t, true)
	_ = w
	obj := NewObjectInstance(newTransferItem(9405, "a probe gem", "probe gem", 0).Prototype, 1001)

	if CanSeeObject(observer, obj) {
		t.Fatal("dark room revealed an object")
	}
	observer.SetAffect(affInfravision, true)
	if !CanSeeObject(observer, obj) {
		t.Fatal("infravision did not reveal the object")
	}
	observer.SetAffect(affInfravision, false)
	observer.SetHolyLight(true)
	if !CanSeeObject(observer, obj) {
		t.Fatal("holylight did not reveal the object")
	}
}
