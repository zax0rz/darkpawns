package game

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1371 batch 7 — fight.c:1672-1681, the player-death syslog.
//
// C (src/fight.c):
//
//	:1674-1676  if (!IS_NPC(ch) && ch != victim) { sprintf(buf2, "(PK) %s killed by %s at %s",
//	                    GET_NAME(victim), GET_NAME(ch), world[victim->in_room].name);
//	                if (!PLR_FLAGGED(victim, PLR_OUTLAW)) SET_BIT_AR(PLR_FLAGS(ch), PLR_OUTLAW); }
//	            else sprintf(buf2, "%s killed by %s at %s", GET_NAME(victim), GET_NAME(ch), ...);
//	:1681       mudlog(buf2, BRF, LVL_IMMORT, TRUE);
//
// Shipped producer: pkg/game/death.go:585-609 in World.handlePlayerDeath — the
// room NAME in the payload, BRF (MudlogBrief) at lvlImmort with file TRUE.
//
// Proof scope: TestCombatBodyDeathCreditNameCollision (combat_body_loot_review_test.go)
// proves the death-credit and outlaw side of this C block and is left untouched;
// it does NOT prove the producer. This file proves the producer.
//
// Findings recorded, not exercised here:
//   - pkg/combat/fight_core.go:582 logs the same C site only when its onDeath hook
//     is nil, with a different payload ("at room %d" instead of the room name).
//     That text is not in cmd/dp-oracle-diff/expected_divergences.tsv.
//   - The C else-arm (fight.c:1678, no "(PK) " prefix) is not proven in this batch.

const pkRoomName = "The Killing Field"

func pkPayload(victim, killer string) string {
	return "(PK) " + victim + " killed by " + killer + " at " + pkRoomName
}

func pkDeathFixture(t *testing.T) (*World, *milestoneProvider, *bytes.Buffer, mudlogRig, *Player, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: pkRoomName, Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	provider, file, rig := newMudlogRig(t)

	killer := NewPlayer(1, "Killer", 1001)
	killer.SetLevel(20)
	victim := NewPlayer(2, "Victim", 1001)
	victim.SetLevel(20)
	for _, p := range []*Player{killer, victim} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.GetName(), err)
		}
	}
	w.MessageSink = func(string, []byte) {}
	return w, provider, file, rig, killer, victim
}

// TestPKDeathMudlogProducer pins payload, file and the negatives for the PK line.
func TestPKDeathMudlogProducer(t *testing.T) {
	w, provider, file, rig, killer, victim := pkDeathFixture(t)
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, combat.TYPE_HIT)

	payload := pkPayload("Victim", "Killer")
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file PK payload count = %d, want 1; log = %q", got, file.String())
	}
	// BRF reaches a LOG1-only observer and never a no-flag one.
	if !rigSaw(provider, rig.log1, payload) {
		t.Fatalf("LOG1-only observer did not receive %q", bracket(payload))
	}
	if rigSaw(provider, rig.noFlag, payload) {
		t.Fatalf("no-flag observer received %q: type would be OFF", bracket(payload))
	}
	if rigSaw(provider, rig.underFloor, payload) {
		t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracket(payload))
	}
}

// TestPKDeathMudlogTypeIsBRF: LOG1-only receipt bounds the type at BRF and the
// no-flag silence bounds it below the OFF case.
func TestPKDeathMudlogTypeIsBRF(t *testing.T) {
	w, provider, _, rig, killer, victim := pkDeathFixture(t)
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, combat.TYPE_HIT)
	payload := pkPayload("Victim", "Killer")
	if !rigSaw(provider, rig.log1, payload) {
		t.Fatalf("LOG1-only observer did not receive %q: type > BRF", bracket(payload))
	}
	if rigSaw(provider, rig.noFlag, payload) {
		t.Fatalf("no-flag observer received %q: type < BRF", bracket(payload))
	}
}

// TestPKDeathMudlogLevelIsLVLIMMORT: fixed LVL_IMMORT, with the killer's
// invisibility not raising it (C has no invis term in this mudlog call).
func TestPKDeathMudlogLevelIsLVLIMMORT(t *testing.T) {
	t.Run("lv_immort_floor", func(t *testing.T) {
		w, provider, _, rig, killer, victim := pkDeathFixture(t)
		victim.SetHP(-1)
		w.HandleDeath(victim, killer, combat.TYPE_HIT)
		payload := pkPayload("Victim", "Killer")
		if rigSaw(provider, rig.underFloor, payload) {
			t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracket(payload))
		}
		if !rigSaw(provider, rig.floor, payload) {
			t.Fatalf("level-%d observer did not receive %q", LVL_IMMORT, bracket(payload))
		}
	})
	t.Run("killer_invisibility_does_not_raise_it", func(t *testing.T) {
		w, provider, _, rig, killer, victim := pkDeathFixture(t)
		// Both actors invisible: the level is a fixed LVL_IMMORT in C, so no
		// invisibility term on either the victim or the killer may raise it.
		killer.SetInvisLevel(36)
		victim.SetInvisLevel(36)
		victim.SetHP(-1)
		w.HandleDeath(victim, killer, combat.TYPE_HIT)
		payload := pkPayload("Victim", "Killer")
		if !rigSaw(provider, rig.floor, payload) {
			t.Fatalf("invis-36 killer suppressed the line for a level-%d observer: %q", LVL_IMMORT, bracket(payload))
		}
		if rigSaw(provider, rig.underFloor, payload) {
			t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracket(payload))
		}
	})
}
