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
// Shipped producers include pkg/game/death.go:585-609 in World.handlePlayerDeath
// and pkg/combat/fight_core.go:582 in default damage(). The latter reaches the
// same room NAME through World.WireCombatCallbacks and is exercised below.
//
// Proof scope: TestCombatBodyDeathCreditNameCollision (combat_body_loot_review_test.go)
// proves the death-credit and outlaw side of this C block and is left untouched;
// it does NOT prove the producer. This file proves the producer.
//
// Proof scope: the live callback tests below prove both C payload arms in
// fight_core.go's default damage path. World.handlePlayerDeath's separate
// non-PK helper branch remains untested.

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

// TestPKDeathMudlogThroughCombatCallback proves the live damage() producer,
// not only the later World death helper. Ambush is one shipped caller of
// combat.TakeDamage for a player victim (pkg/game/ambush.go:76-89).
func TestPKDeathMudlogThroughCombatCallback(t *testing.T) {
	w, provider, file, rig, killer, victim := pkDeathFixture(t)
	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	combat.SetCallbacks(w.WireCombatCallbacks())

	victim.SetPosition(combat.PosStanding)
	w.applyAmbushDamage(killer, victim, victim.GetHP()+100)

	payload := pkPayload("Victim", "Killer")
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("combat file PK payload count = %d, want 1; log = %q", got, file.String())
	}
	if strings.Count(file.String(), pkRoomName) != 1 {
		t.Fatalf("room name occurrences in combat file log = %d, want 1; log = %q", strings.Count(file.String(), pkRoomName), file.String())
	}
	if got := strings.Count(provider.lines[rig.log1.Name], bracket(payload)); got != 1 {
		t.Fatalf("combat LOG1 PK payload count = %d, want 1; bytes = %q", got, provider.lines[rig.log1.Name])
	}
	if !rigSaw(provider, rig.log2, payload) {
		t.Fatal("combat BRF PK payload did not reach a LOG2-only observer")
	}
	if rigSaw(provider, rig.noFlag, payload) {
		t.Fatal("combat PK payload reached an observer without LOG1/LOG2")
	}
	if rigSaw(provider, rig.underFloor, payload) {
		t.Fatalf("combat PK payload reached level %d, want fixed LVL_IMMORT %d", LVL_IMMORT-1, LVL_IMMORT)
	}
	if !rigSaw(provider, rig.floor, payload) {
		t.Fatalf("combat PK payload missed level-%d observer", LVL_IMMORT)
	}
}

// fight.c:1678 takes the unprefixed arm for a self-kill. The ambush damage
// seam accepts a real player as both attacker and victim, so this exercises
// that defaultTakeDamage branch without manufacturing callbacks or actors.
func TestPlayerSelfDeathMudlogThroughCombatCallback(t *testing.T) {
	w, provider, file, rig, _, victim := pkDeathFixture(t)
	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	combat.SetCallbacks(w.WireCombatCallbacks())

	victim.SetPosition(combat.PosStanding)
	w.applyAmbushDamage(victim, victim, victim.GetHP()+100)

	payload := "Victim killed by Victim at " + pkRoomName
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("self-death file payload count = %d, want 1; log = %q", got, file.String())
	}
	if strings.Contains(file.String(), "(PK) "+payload) {
		t.Fatalf("self-death unexpectedly used PK payload: %q", file.String())
	}
	if got := strings.Count(provider.lines[rig.log1.Name], bracket(payload)); got != 1 {
		t.Fatalf("self-death LOG1 payload count = %d, want 1; bytes = %q", got, provider.lines[rig.log1.Name])
	}
	if rigSaw(provider, rig.noFlag, payload) || rigSaw(provider, rig.underFloor, payload) {
		t.Fatal("self-death payload bypassed the BRF/31 observers")
	}
}

// src/fight.c:1300-1309: counter_procs calls mudlog with NRM, LVL_IMMORT,
// FALSE. Exercise combat.CounterProcs through the World's installed callback
// set, alongside the game-layer producer test in game_diagnostic_mudlog_test.go.
func TestCombatKillMilestoneMudlogCallbackWiring(t *testing.T) {
	w, provider, file, rig, killer, _ := pkDeathFixture(t)
	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	combat.SetCallbacks(w.WireCombatCallbacks())

	killer.Kills = 5000
	killer.SetHP(1)
	combat.CounterProcs(killer)

	payload := "Killer hit 5000 kills."
	if got := strings.Count(provider.lines[rig.log2.Name], bracket(payload)); got != 1 {
		t.Fatalf("combat NRM LOG2 payload count = %d, want 1; bytes = %q", got, provider.lines[rig.log2.Name])
	}
	if rigSaw(provider, rig.log1, payload) || rigSaw(provider, rig.noFlag, payload) {
		t.Fatal("combat NRM milestone reached a LOG1-only or no-flag observer")
	}
	if rigSaw(provider, rig.underFloor, payload) {
		t.Fatalf("combat milestone reached level %d, want LVL_IMMORT %d", LVL_IMMORT-1, LVL_IMMORT)
	}
	if !rigSaw(provider, rig.floor, payload) {
		t.Fatalf("combat milestone missed level-%d observer", LVL_IMMORT)
	}
	if strings.Contains(file.String(), payload) {
		t.Fatalf("combat NRM milestone unexpectedly wrote to file: %q", file.String())
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
