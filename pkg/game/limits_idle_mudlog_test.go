package game

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1371 batch 7 — limits.c:449-450, the idle force-rent/disconnect line.
//
// C (src/limits.c, check_idling's second threshold):
//
//	:449-450  sprintf(buf, "%s force-rented and extracted (idle).", GET_NAME(ch));
//	          mudlog(buf, CMP, LVL_GOD, TRUE);
//
// Go producer: pkg/game/limits_misc.go:111 in World.CheckIdling.
//
// This row is CMP (3) at a fixed LVL_GOD with no invisibility term, so its
// discriminators differ from the house/quit rows:
//
//	type  CMP: a both-bits observer receives (type <= CMP) while a LOG2-only
//	          observer does not (type > NRM) and a LOG1-only observer does not;
//	level     fixed LVL_GOD: silent at LVL_GOD-1 and receiving at LVL_GOD, and an
//	          invisible idler must NOT raise it (no MAX(LVL_IMMORT, invis)).

const idleForceRentPayload = "Idler force-rented and extracted (idle)."

// idleMudlogFixture reuses the shape of idle_destination_test.go: rooms 0/1/3/4
// so the second threshold's char_to_room(RNUM 3) resolves, and WasInRoom set so
// the force-rent arm (not the void pull) is the one exercised.
func idleMudlogFixture(t *testing.T) (*World, *milestoneProvider, *bytes.Buffer, mudlogRig, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 0}, {VNum: 1}, {VNum: 3}, {VNum: 4}, {VNum: 1001},
	}})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	provider, file, rig := newMudlogRig(t)

	idler := NewPlayer(1, "Idler", 1)
	idler.SetLevel(1)                 // mortals only (limits.c:424-425)
	idler.IdleTimer = IDLE_DISCONNECT // +1 inside CheckIdling crosses the threshold
	idler.WasInRoom = 1001            // already voided: take the force-rent arm
	if err := w.AddPlayer(idler); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.MessageSink = func(string, []byte) {}
	return w, provider, file, rig, idler
}

// TestIdleForceRentMudlogProducer pins payload, file and the type/level negatives.
func TestIdleForceRentMudlogProducer(t *testing.T) {
	w, provider, file, rig, idler := idleMudlogFixture(t)
	w.CheckIdling(idler)
	payload := idleForceRentPayload
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	if !rigSaw(provider, rig.god, payload) {
		t.Fatalf("both-bits LVL_GOD observer did not receive %q", bracket(payload))
	}
	if rigSaw(provider, rig.log2, payload) {
		t.Fatalf("LOG2-only observer received %q: type <= NRM", bracket(payload))
	}
	if rigSaw(provider, rig.log1, payload) {
		t.Fatalf("LOG1-only observer received %q: type <= BRF", bracket(payload))
	}
	if rigSaw(provider, rig.underGod, payload) {
		t.Fatalf("level-%d observer received %q", LVL_GOD-1, bracket(payload))
	}
}

// TestIdleForceRentMudlogTypeIsCMP: both bits are required, which is CMP.
func TestIdleForceRentMudlogTypeIsCMP(t *testing.T) {
	w, provider, _, rig, idler := idleMudlogFixture(t)
	w.CheckIdling(idler)
	payload := idleForceRentPayload
	if !rigSaw(provider, rig.god, payload) {
		t.Fatalf("both-bits observer did not receive %q: type > CMP", bracket(payload))
	}
	if rigSaw(provider, rig.log2, payload) {
		t.Fatalf("LOG2-only observer received %q: type <= NRM", bracket(payload))
	}
	if rigSaw(provider, rig.log1, payload) {
		t.Fatalf("LOG1-only observer received %q: type <= BRF", bracket(payload))
	}
}

// TestIdleForceRentMudlogLevelIsLVLGOD: fixed LVL_GOD, not the MAX(LVL_IMMORT,
// invis) form the house/quit producers use.
func TestIdleForceRentMudlogLevelIsLVLGOD(t *testing.T) {
	t.Run("lv_god_floor", func(t *testing.T) {
		w, provider, _, rig, idler := idleMudlogFixture(t)
		w.CheckIdling(idler)
		payload := idleForceRentPayload
		if rigSaw(provider, rig.underGod, payload) {
			t.Fatalf("level-%d observer received %q", LVL_GOD-1, bracket(payload))
		}
		if !rigSaw(provider, rig.god, payload) {
			t.Fatalf("level-%d observer did not receive %q", LVL_GOD, bracket(payload))
		}
	})
	t.Run("invisibility_does_not_raise_it", func(t *testing.T) {
		w, provider, _, rig, idler := idleMudlogFixture(t)
		idler.SetInvisLevel(36)
		w.CheckIdling(idler)
		payload := idleForceRentPayload
		if !rigSaw(provider, rig.god, payload) {
			t.Fatalf("invis-36 idler suppressed the line for a level-%d observer: %q", LVL_GOD, bracket(payload))
		}
		if rigSaw(provider, rig.underGod, payload) {
			t.Fatalf("level-%d observer received %q", LVL_GOD-1, bracket(payload))
		}
	})
}
