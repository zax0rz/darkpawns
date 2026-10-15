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
			mob := NewMob(&parser.Mob{VNum: tc.vnum, Keywords: tc.name, ShortDesc: "a " + tc.name}, 1001)
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
	demon := NewMob(&parser.Mob{VNum: 14401, Keywords: "demon", ShortDesc: "a demon"}, 1001)
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
