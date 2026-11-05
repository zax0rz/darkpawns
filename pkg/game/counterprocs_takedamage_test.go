package game

// Proof for BRIEF-counterprocs-duplicate: C's counter_procs has exactly one
// implementation, and the combat.TakeDamage path reaches it.
//
// C (src/fight.c):
//
//	:1252-1258  static void counter_procs(struct char_data *ch)
//	            { if (IS_NPC(ch) || IS_MOB(ch)) return; switch(GET_KILLS(ch)) {
//	:1271-1277  case 1000: case 2000: case 10000: ... case 50000:
//	:1278         send_to_char("The gods reward your many victories!\n\r", ch);
//	:1280         switch(number(1,3)) {
//	:1281-1282      case 1: GET_MAX_HIT(ch)++;      /* no break */
//	:1283-1284      case 2: GET_MAX_MANA(ch)++;     /* no break */
//	:1285-1286      case 3: GET_MAX_MOVE(ch)++;     /* no break */
//	:1287-1289      default: GET_MAX_HIT(ch)++; break; }
//	:1291         GET_HIT(ch)=GET_MAX_HIT(ch); break;
//	:1689-1690  GET_KILLS(ch)++; counter_procs(ch);   (damage()'s death tail)
//
// The draw (R3a), the fall-through reward (R1) and the LFCR terminator (R1)
// are all player-observable, so the port must produce them from one place.
// pkg/game/World.counter_procs is that place (DP-1040); pkg/combat.CounterProcs
// only forwards, and pkg/combat/fight_core.go:624-626 is the C call site.
//
// This test drives the shipped combat.TakeDamage route rather than calling
// either counter_procs directly: a player victim with no game-layer death
// callback leaves onDeath == nil, so combat.TakeDamage runs C's own death tail
// (pkg/combat/fight_core.go:621-628). Ambush is the shipped caller of that
// route for a player victim (pkg/game/ambush.go:74-89), which is exactly the
// rare-but-real case the brief identifies.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// milestoneRoller scripts number(1,3) and records every draw, so a test can
// both force a roll and prove the shared stream advanced the way C's does.
// Any other range returns its lower bound, which keeps the rest of the death
// path deterministic without consuming scripted values.
type milestoneRoller struct {
	rolls []int
	draws [][2]int
}

func (r *milestoneRoller) Number(from, to int) int {
	r.draws = append(r.draws, [2]int{from, to})
	if from == 1 && to == 3 {
		if len(r.rolls) == 0 {
			return 1
		}
		value := r.rolls[0]
		r.rolls = r.rolls[1:]
		return value
	}
	return from
}

func (r *milestoneRoller) Dice(num, size int) int { return num }

func (r *milestoneRoller) IntN(n int) int { return 0 }

// number13Draws counts only the number(1,3) draws C's counter_procs makes.
func (r *milestoneRoller) number13Draws() int {
	count := 0
	for _, draw := range r.draws {
		if draw[0] == 1 && draw[1] == 3 {
			count++
		}
	}
	return count
}

// TestCounterProcsTakeDamageMilestoneReward proves the TakeDamage route to a
// major milestone for each scripted roll: C's stat deltas, the LFCR bytes and
// exactly one number(1,3) draw.
func TestCounterProcsTakeDamageMilestoneReward(t *testing.T) {
	for _, tc := range []struct {
		roll                        int
		wantHit, wantMana, wantMove int
	}{
		{roll: 1, wantHit: 2, wantMana: 1, wantMove: 1}, // case 1 → 2 → 3 → default
		{roll: 2, wantHit: 1, wantMana: 1, wantMove: 1}, // case 2 → 3 → default
		{roll: 3, wantHit: 1, wantMana: 0, wantMove: 1}, // case 3 → default
	} {
		t.Run(fmt.Sprintf("roll_%d", tc.roll), func(t *testing.T) {
			w := newCounterProcsTestWorld(t)

			oldCallbacks := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
			combat.SetCallbacks(w.WireCombatCallbacks())

			oldRoller := combat.GetRoller()
			roller := &milestoneRoller{rolls: []int{tc.roll}}
			combat.SetRoller(roller)
			t.Cleanup(func() { combat.SetRoller(oldRoller) })

			killer := NewPlayer(1, "Killer", 1001)
			// combat's death tail increments the count before counter_procs
			// (pkg/combat/fight_core.go:624), so 1999 becomes the 2000-kill
			// major milestone.
			killer.Kills = 1999
			killer.MaxHealth, killer.MaxMana, killer.MaxMove = 100, 100, 100
			killer.Health = 100

			victim := NewPlayer(2, "Victim", 1001)
			// damage()'s protection gate refuses a player-kill blow from or
			// against a level-10-or-below body (pkg/game/damage_gate.go:47-56).
			killer.SetLevel(20)
			victim.SetLevel(20)
			for _, p := range []*Player{killer, victim} {
				if err := w.AddPlayer(p); err != nil {
					t.Fatalf("AddPlayer(%s): %v", p.GetName(), err)
				}
			}

			var actorBytes string
			w.MessageSink = func(name string, msg []byte) {
				if name == killer.GetName() {
					actorBytes += string(msg)
				}
			}

			victim.SetPosition(combat.PosStanding)
			w.applyAmbushDamage(killer, victim, victim.GetHP()+100)

			if got := victim.GetPosition(); got != combat.PosDead {
				t.Fatalf("victim position = %d, want PosDead %d", got, combat.PosDead)
			}
			if got := killer.Kills; got != 2000 {
				t.Fatalf("killer kills = %d, want 2000", got)
			}
			if got := killer.MaxHealth - 100; got != tc.wantHit {
				t.Errorf("max hit gain = %d, want %d (C src/fight.c:1282,1288)", got, tc.wantHit)
			}
			if got := killer.MaxMana - 100; got != tc.wantMana {
				t.Errorf("max mana gain = %d, want %d (C src/fight.c:1284)", got, tc.wantMana)
			}
			if got := killer.MaxMove - 100; got != tc.wantMove {
				t.Errorf("max move gain = %d, want %d (C src/fight.c:1286)", got, tc.wantMove)
			}
			if got := killer.Health; got != killer.MaxHealth {
				t.Errorf("killer HP = %d, want GET_HIT = GET_MAX_HIT = %d (C src/fight.c:1291)", got, killer.MaxHealth)
			}

			const line = "The gods reward your many victories!"
			if !strings.Contains(actorBytes, line+"\n\r") {
				t.Errorf("milestone line missing C's LFCR terminator (src/fight.c:1278): %q", actorBytes)
			}
			if strings.Contains(actorBytes, line+"\r\n") {
				t.Errorf("milestone line used CRLF; C sends LFCR (src/fight.c:1278): %q", actorBytes)
			}

			if got := roller.number13Draws(); got != 1 {
				t.Errorf("number(1,3) draws = %d, want exactly 1 (C src/fight.c:1280)", got)
			}
		})
	}
}

// TestCounterProcsTakeDamageMinorMilestoneHeals pins the other arm of C's
// switch on the same TakeDamage route: a 5000-kill milestone heals to full,
// changes no max stat, and draws nothing (src/fight.c:1262-1270 has no
// number(1,3)).
func TestCounterProcsTakeDamageMinorMilestoneHeals(t *testing.T) {
	w := newCounterProcsTestWorld(t)

	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	combat.SetCallbacks(w.WireCombatCallbacks())

	oldRoller := combat.GetRoller()
	roller := &milestoneRoller{}
	combat.SetRoller(roller)
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	killer := NewPlayer(1, "Killer", 1001)
	killer.Kills = 4999 // the death tail makes this 5000
	killer.MaxHealth, killer.MaxMana, killer.MaxMove = 100, 100, 100
	killer.Health = 1

	victim := NewPlayer(2, "Victim", 1001)
	// Level 20 clears damage()'s level-10 player-kill protection gate.
	killer.SetLevel(20)
	victim.SetLevel(20)
	for _, p := range []*Player{killer, victim} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.GetName(), err)
		}
	}

	var actorBytes string
	w.MessageSink = func(name string, msg []byte) {
		if name == killer.GetName() {
			actorBytes += string(msg)
		}
	}

	victim.SetPosition(combat.PosStanding)
	w.applyAmbushDamage(killer, victim, victim.GetHP()+100)

	if got := killer.Health; got != 100 {
		t.Errorf("killer HP = %d, want 100 (C src/fight.c:1269 GET_HIT = GET_MAX_HIT)", got)
	}
	if killer.MaxHealth != 100 || killer.MaxMana != 100 || killer.MaxMove != 100 {
		t.Errorf("minor milestone changed max stats: hit=%d mana=%d move=%d",
			killer.MaxHealth, killer.MaxMana, killer.MaxMove)
	}
	if !strings.Contains(actorBytes, "The gods reward your glory in battle!\n\r") {
		t.Errorf("minor milestone line missing C's LFCR bytes (src/fight.c:1267): %q", actorBytes)
	}
	if got := roller.number13Draws(); got != 0 {
		t.Errorf("number(1,3) draws = %d, want 0 for a 5000-kill milestone (src/fight.c:1262-1270)", got)
	}
}

// TestCounterProcsTakeDamageNonMilestoneSilent pins the default arm: an
// ordinary kill reward is silent and draws nothing (src/fight.c:1293-1294).
func TestCounterProcsTakeDamageNonMilestoneSilent(t *testing.T) {
	w := newCounterProcsTestWorld(t)

	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	combat.SetCallbacks(w.WireCombatCallbacks())

	oldRoller := combat.GetRoller()
	roller := &milestoneRoller{}
	combat.SetRoller(roller)
	t.Cleanup(func() { combat.SetRoller(oldRoller) })

	killer := NewPlayer(1, "Killer", 1001)
	killer.Kills = 1998 // the death tail makes this 1999, not a milestone
	killer.MaxHealth, killer.MaxMana, killer.MaxMove = 100, 100, 100
	killer.Health = 1

	victim := NewPlayer(2, "Victim", 1001)
	// Level 20 clears damage()'s level-10 player-kill protection gate.
	killer.SetLevel(20)
	victim.SetLevel(20)
	for _, p := range []*Player{killer, victim} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.GetName(), err)
		}
	}

	var actorBytes string
	w.MessageSink = func(name string, msg []byte) {
		if name == killer.GetName() {
			actorBytes += string(msg)
		}
	}

	victim.SetPosition(combat.PosStanding)
	w.applyAmbushDamage(killer, victim, victim.GetHP()+100)

	if got := killer.Kills; got != 1999 {
		t.Fatalf("killer kills = %d, want 1999", got)
	}
	if strings.Contains(actorBytes, "The gods reward") {
		t.Errorf("non-milestone kill produced a reward line: %q", actorBytes)
	}
	if got := killer.Health; got != 1 {
		t.Errorf("killer HP = %d, want unchanged 1 (src/fight.c:1293-1294)", got)
	}
	if got := roller.number13Draws(); got != 0 {
		t.Errorf("number(1,3) draws = %d, want 0 for a non-milestone kill", got)
	}
}
