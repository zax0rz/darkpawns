package game

// spec_voice_depth_test.go — DP-1440's class audit (R5c): the special
// procedures that speak for a mobile must use C's act() and C's exact text, not
// a hand-built room line (src/spec_procs.c:1155-1310).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// specVoiceWorld is a room with a speaking mobile and one observer.
func specVoiceWorld(t *testing.T, shortDesc string) (*World, *MobInstance, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Voice Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	mob := NewMob(&parser.Mob{
		VNum: 9801, Keywords: "knight", ShortDesc: shortDesc,
		Race: 1, Level: 20, Position: 8, DefaultPos: 8, Sex: 1, // C SEX_MALE
	}, 1001)
	watcher := NewPlayer(1, "Watcher", 1001)
	watcher.SetPosition(8)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, mob, watcher, out
}

// seedForFirstDraw finds a stream seed whose first number(1,20) is want, so the
// taunt's own draw can be pinned without a seam.
func seedForFirstDraw(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 500; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 20) == want {
			return seed
		}
	}
	t.Fatalf("no seed in 1..499 draws %d first", want)
	return 0
}

// TestUndeadKnightTauntTexts pins C's taunt texts verbatim, including the
// typos the port had corrected: "your going" (:1163), the double space in
// "scoundrel.  I'm" (:1166), "what your thinking" (:1171) and "more that one
// way" (:1240). The sink is act() to the room, so an observer hears the line.
func TestUndeadKnightTauntTexts(t *testing.T) {
	for _, tc := range []struct {
		name string
		desc string
		spec func(*World, *Player, *MobInstance, string, string) bool
		text map[int]string
	}{
		{
			name: "black undead knight",
			desc: "the black knight",
			spec: specBlackUndeadKnight,
			text: map[int]string{
				2: "The black knight shouts, 'If I'm going to hell, your going with me!'\r\n",
				3: "The black knight says, 'You dirty rotten scoundrel.  I'm gonna make you very sorry.'\r\n",
				4: "The black knight says, 'I know what your thinking...'\r\n",
			},
		},
		{
			name: "red undead knight",
			desc: "the red knight",
			spec: specRedUndeadKnight,
			text: map[int]string{
				4: "The red knight says, 'There's more that one way to skin a cat:'\r\n",
			},
		},
	} {
		for pick, want := range tc.text {
			t.Run(tc.name, func(t *testing.T) {
				seed := seedForFirstDraw(t, pick)
				w, mob, _, out := specVoiceWorld(t, tc.desc)
				// The taunt arm needs the mobile to be fighting.
				mob.SetFightingBody(mob)
				dprng.ResetStream(seed)

				tc.spec(w, nil, mob, "", "")

				if got := out["Watcher"]; !strings.Contains(got, want) {
					t.Fatalf("room bytes = %q, want %q among them", got, want)
				}
			})
		}
	}
}

// TestMickeyStandUsesDoStand pins C's do_stand call (src/spec_procs.c:1280):
// the room hears the ordinary stand act, not a hand-built "stands up!".
func TestMickeyStandUsesDoStand(t *testing.T) {
	w, mob, _, out := specVoiceWorld(t, "Mickey")
	mob.SetFightingBody(mob)
	mob.SetPosition(combat.PosResting)

	specMickey(w, nil, mob, "", "")

	want := "Mickey stops resting, and clambers on his feet.\r\n"
	if got := out["Watcher"]; !strings.Contains(got, want) {
		t.Fatalf("room bytes = %q, want %q among them", got, want)
	}
	if strings.Contains(out["Watcher"], "stands up!") {
		t.Fatalf("the hand-built stand line survived: %q", out["Watcher"])
	}
}

// TestConSellerTellSetsTheReplyTarget pins perform_tell's GET_LAST_TELL
// (src/act.comm.c:894) on the con-seller's do_tell path
// (src/spec_procs3.c:119-122), which the port's room-light-aware rendering
// bypasses.
func TestConSellerTellSetsTheReplyTarget(t *testing.T) {
	w, mob, buyer, _ := specVoiceWorld(t, "the con seller")

	specConSeller(w, buyer, mob, "list", "")

	if got := buyer.GetLastTeller(); got != mob.GetShortDesc() {
		t.Fatalf("GET_LAST_TELL = %q, want %q so reply reaches the seller", got, mob.GetShortDesc())
	}
}
