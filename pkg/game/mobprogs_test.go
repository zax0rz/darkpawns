package game

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMpSoundRunsDefaultMobileTail(t *testing.T) {
	for _, tc := range []struct {
		name      string
		vnum      int
		wantDraws []mobactDrawRange
	}{
		{name: "dog", vnum: 8063, wantDraws: []mobactDrawRange{{0, 25}}},
		{name: "demon", vnum: 14401, wantDraws: []mobactDrawRange{{0, 2}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newMobactDrawTestWorld()
			mob := NewMob(&parser.Mob{VNum: tc.vnum, Keywords: tc.name, ShortDesc: "a " + tc.name, Int: 11, Wis: 11}, 1001)
			var draws []mobactDrawRange
			oldMobProgNumber := mobProgNumber
			mobProgNumber = func(from, to int) int {
				draws = append(draws, mobactDrawRange{from: from, to: to})
				return to
			}
			t.Cleanup(func() { mobProgNumber = oldMobProgNumber })

			w.MpSound(mob)

			if !reflect.DeepEqual(draws, tc.wantDraws) {
				t.Fatalf("MpSound draws = %v, want %v", draws, tc.wantDraws)
			}
		})
	}
}

func TestMpSoundDemonAdvancesStateAfterTailDraw(t *testing.T) {
	w := newMobactDrawTestWorld()
	demon := NewMob(&parser.Mob{VNum: 14401, Keywords: "demon", ShortDesc: "a demon", Int: 11, Wis: 11}, 1001)
	demon.SetAlignment(-1000)
	observer := NewPlayer(1, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var message string
	w.MessageSink = func(_ string, bytes []byte) { message += string(bytes) }
	var draws []mobactDrawRange
	oldMobProgNumber := mobProgNumber
	mobProgNumber = func(from, to int) int {
		draws = append(draws, mobactDrawRange{from: from, to: to})
		return 0
	}
	t.Cleanup(func() { mobProgNumber = oldMobProgNumber })

	w.MpSound(demon)

	if !reflect.DeepEqual(draws, []mobactDrawRange{{0, 2}}) {
		t.Fatalf("MpSound draws = %v, want the demon tail draw", draws)
	}
	if got := demon.GetAlignment(); got != -999 {
		t.Fatalf("demon alignment = %d, want -999", got)
	}
	if !strings.Contains(message, "I seek the dull blackened stones") || strings.Contains(message, "$n") {
		t.Fatalf("demon message = %q, want its C-style spoken line", message)
	}
}

// TestMpGiveObjectDemonSoulSayRoutesThroughGate pins that the soul-eater's
// parting line goes through C's do_say body. C calls
// do_say(mob, "Enter the portal quickly! It will not last long!", 0, 0)
// (src/mobprog.c:159), so the INT/WIS gate (src/act.comm.c:766-771) silences a
// numbed demon. Before this fix the port printed the line with a literal Act()
// and skipped the gate. Revert proof (R5h): restoring that Act() makes the
// zero-INT case below speak, and the test fails.
func TestMpGiveObjectDemonSoulSayRoutesThroughGate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		intel  int
		spoken bool
	}{
		{"capable demon speaks", 11, true},
		{"numbed demon is silent", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newActTemplateWorld(1001)
			observer := addCapturedPlayer(t, w, "Pracobserver", 1001)
			observerPlayer := wPlayer(t, w, "Pracobserver")
			observerPlayer.SetPosition(8) // POS_STANDING: SENDOK
			addCapturedPlayer(t, w, "Pracgiver", 1001)
			giver := wPlayer(t, w, "Pracgiver")
			giver.SetPosition(8)
			demon := NewMob(&parser.Mob{
				VNum: 14401, Keywords: "demon", ShortDesc: "a soul eater",
				Int: tc.intel, Wis: 11,
			}, 1001)
			soul := NewObjectInstance(&parser.Obj{VNum: 9900}, 1001)

			w.MpGiveObject(giver, demon, soul)

			want := "A soul eater exclaims, 'Enter the portal quickly! It will not last long!'\r\n"
			if tc.spoken && !strings.Contains(observer.msgs, want) {
				t.Fatalf("demon INT %d: observer got %q, want %q", tc.intel, observer.msgs, want)
			}
			if !tc.spoken && strings.Contains(observer.msgs, "Enter the portal quickly") {
				t.Fatalf("demon INT %d spoke while too stupid: %q", tc.intel, observer.msgs)
			}
		})
	}
}
