package game

// mob_voice_depth_test.go — DP-1440. C's mob voice: the kill brag
// (src/fight.c:1107-1248), its attitude_loot call site (:1693-1697) and the
// hunter's taunts (src/graph.c:262-404). Every case drives the real death or
// hunt path and reads the players' bytes plus review's output.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// bragRoller scripts the brag's two draws by their bounds, so unrelated draws
// on the death path cannot shift the script, and records every draw so a test
// can assert that a gated-off brag burned nothing (R3).
type bragRoller struct {
	mobChance int
	casePick  int
	calls     [][2]int
}

func (r *bragRoller) Number(from, to int) int {
	r.calls = append(r.calls, [2]int{from, to})
	switch {
	case from == 0 && to == 20:
		return r.mobChance
	case from == 1 && to == 12:
		return r.casePick
	}
	return 0
}

func (r *bragRoller) Dice(num, size int) int { return num }
func (r *bragRoller) IntN(n int) int         { return 0 }

// bragDraws counts the brag's own two draws (number(0,20), number(1,12)).
func (r *bragRoller) bragDraws() int {
	n := 0
	for _, c := range r.calls {
		if (c[0] == 0 && c[1] == 20) || (c[0] == 1 && c[1] == 12) {
			n++
		}
	}
	return n
}

// installBragRoller pins the combat roller for one test.
func installBragRoller(t *testing.T, mobChance, casePick int) *bragRoller {
	t.Helper()
	r := &bragRoller{mobChance: mobChance, casePick: casePick}
	old := combat.GetRoller()
	combat.SetRoller(r)
	t.Cleanup(func() { combat.SetRoller(old) })
	return r
}

// voiceMob registers and spawns one mobile with an explicit short description,
// race, level, alignment and action flags.
func voiceMob(t *testing.T, w *World, vnum, room int, shortDesc string, race, level, alignment int, flags ...string) *MobInstance {
	t.Helper()
	proto := &parser.Mob{
		VNum: vnum, Keywords: "probe", ShortDesc: shortDesc,
		Race: race, Level: level, Alignment: alignment,
		ActionFlags: flags, Position: 8, DefaultPos: 8,
		HP:  parser.DiceRoll{Num: 1, Sides: 1, Plus: 40},
		Str: 11, Dex: 11, Int: 11, Wis: 11, Con: 11, Cha: 11, AC: 100,
	}
	w.mu.Lock()
	w.mobs[vnum] = proto
	w.mu.Unlock()
	mob, err := w.SpawnMob(vnum, room)
	if err != nil {
		t.Fatalf("SpawnMob(%d): %v", vnum, err)
	}
	return mob
}

// TestMobKillBragReachesGossipAndReview is DP-1440's defect 1: the brag was
// computed and thrown away because nothing assigned callbacks.BroadChat. C
// sends it with do_gen_comm(ch, buf2, 0, SCMD_GOSSIP) (src/fight.c:1246), which
// is global and records the line in review (src/act.comm.c:1275-1276).
func TestMobKillBragReachesGossipAndReview(t *testing.T) {
	w, out := croakTestWorld(t)
	killer := voiceMob(t, w, 9601, 1001, "the dune stalker", 1, 20, -500, "AGGR24")
	victim := croakPlayer(t, w, "Victim", 1001)
	watcher := croakPlayer(t, w, "Watcher", 1001)
	croakPlayer(t, w, "Elsewhere", 1002)
	victim.SetHP(-1)
	installBragRoller(t, 0, 3) // case 3: "<name> was easy xp."

	w.HandleDeath(victim, killer, 0)

	want := "The dune stalker gossips, 'Victim was easy xp.'\r\n"
	if got := out["Watcher"]; !strings.Contains(got, want) {
		t.Fatalf("killer's room bytes = %q, want %q among them", got, want)
	}
	if got := out["Elsewhere"]; !strings.Contains(got, want) {
		t.Fatalf("elsewhere bytes = %q, want the global gossip %q", got, want)
	}
	if got := w.ReviewGossip(watcher); !strings.Contains(got, "Victim was easy xp.") {
		t.Fatalf("review = %q, want the brag recorded", got)
	}
}

// TestMobKillBragGates pins C's three pre-draw gates
// (src/fight.c:1161-1172). C returns before burning number(0,20)/number(1,12)
// when the killer is not AGGR24, cannot speak, or killed itself, so a port that
// draws anyway drifts the shared stream (R3).
func TestMobKillBragGates(t *testing.T) {
	t.Run("silent mobile burns no draw and says nothing", func(t *testing.T) {
		w, out := croakTestWorld(t)
		// Race 12 is outside intelligent_races[] (src/constants.c:353-373).
		killer := voiceMob(t, w, 9602, 1001, "the mute stalker", 12, 20, -500, "AGGR24")
		victim := croakPlayer(t, w, "Victim", 1001)
		victim.SetHP(-1)
		r := installBragRoller(t, 0, 3)

		w.HandleDeath(victim, killer, 0)

		if got := r.bragDraws(); got != 0 {
			t.Fatalf("a silent mobile burned %d brag draws, want 0 (can_speak gate precedes them)", got)
		}
		if strings.Contains(out["Victim"], "gossips") {
			t.Fatalf("a silent mobile gossiped: %q", out["Victim"])
		}
	})

	t.Run("self kill burns no draw", func(t *testing.T) {
		w, _ := croakTestWorld(t)
		mob := voiceMob(t, w, 9603, 1001, "the self-slayer", 1, 20, -500, "AGGR24")
		r := installBragRoller(t, 0, 3)

		combat.BragMessage(mob, mob)

		if got := r.bragDraws(); got != 0 {
			t.Fatalf("a self kill burned %d brag draws, want 0 (ch == victim gate)", got)
		}
	})

	t.Run("non-AGGR24 mobile loots but never brags", func(t *testing.T) {
		w, out := croakTestWorld(t)
		killer := voiceMob(t, w, 9604, 1001, "the looter", 1, 20, -500, "LOOTS")
		victim := croakPlayer(t, w, "Victim", 1001)
		victim.SetHP(-1)
		r := installBragRoller(t, 0, 3)

		w.HandleDeath(victim, killer, 0)

		if got := r.bragDraws(); got != 0 {
			t.Fatalf("a non-AGGR24 mobile burned %d brag draws, want 0 (src/fight.c:1161-1162)", got)
		}
		if strings.Contains(out["Victim"], "gossips") {
			t.Fatalf("a non-AGGR24 mobile gossiped: %q", out["Victim"])
		}
	})

	t.Run("non-AGGR24 mobile burns no draw", func(t *testing.T) {
		w, _ := croakTestWorld(t)
		mob := voiceMob(t, w, 9607, 1001, "the looter", 1, 20, -500, "LOOTS")
		victim := croakPlayer(t, w, "Prey", 1001)
		r := installBragRoller(t, 0, 3)

		combat.BragMessage(mob, victim)

		if got := r.bragDraws(); got != 0 {
			t.Fatalf("BragMessage burned %d draws for a non-AGGR24 mobile, want 0 (src/fight.c:1161-1162)", got)
		}
	})
}

// TestMobOnMobKillLootsAndBrags is DP-1440's defect 3: C's attitude_loot call
// site sits outside the !IS_NPC(victim) arm (src/fight.c:1693-1697), so a
// mobile victim is looted and bragged about too. The port ran it only in its
// player-victim branch, so mob-on-mob kills were silent.
func TestMobOnMobKillLootsAndBrags(t *testing.T) {
	w, out := croakTestWorld(t)
	killer := voiceMob(t, w, 9605, 1001, "the dune stalker", 1, 20, -500, "AGGR24")
	prey := voiceMob(t, w, 9606, 1001, "a sand crawler", 1, 5, 0)
	croakPlayer(t, w, "Watcher", 1001)
	// The prey carries something so the loot pass has a transcript.
	w.mu.Lock()
	w.objs[9101] = &parser.Obj{VNum: 9101, Keywords: "trinket", ShortDesc: "a bone trinket", TypeFlag: ITEM_OTHER, WearFlags: [4]int{1}, Weight: 1, Cost: 300}
	w.mu.Unlock()
	trinket, err := w.SpawnObject(9101, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToMobInventoryFront(trinket, prey); err != nil {
		t.Fatalf("MoveObjectToMobInventoryFront: %v", err)
	}
	installBragRoller(t, 0, 3) // the 1-in-21 mob gate passes, then case 3
	prey.TakeDamage(prey.GetHP() + 1)

	w.HandleDeath(prey, killer, 0)

	wantLoot := "The dune stalker gets a bone trinket from the corpse of a sand crawler.\r\n"
	if got := out["Watcher"]; !strings.Contains(got, wantLoot) {
		t.Fatalf("room bytes = %q, want the loot line %q among them", got, wantLoot)
	}
	wantBrag := "The dune stalker gossips, 'a sand crawler was easy xp.'\r\n"
	if got := out["Watcher"]; !strings.Contains(got, wantBrag) {
		t.Fatalf("room bytes = %q, want the mob-victim brag %q among them", got, wantBrag)
	}
}

// TestBragCaseFiveRequiresAGoodVictim pins C's case 5 test
// (src/fight.c:1195): the evil-triumphs line needs an evil killer AND a
// non-evil victim. The port tested only the killer (R1/R4).
func TestBragCaseFiveRequiresAGoodVictim(t *testing.T) {
	const evilLine = "Now you will see that evil will always triumph, because good is dumb."
	for _, tc := range []struct {
		name        string
		killerAlign int
		victimAlign int
		want        string
	}{
		{name: "evil killer, good victim", killerAlign: -500, victimAlign: 500, want: evilLine},
		{name: "evil killer, evil victim", killerAlign: -500, victimAlign: -500, want: "Prey is dead! R.I.P."},
		{name: "good killer, good victim", killerAlign: 500, victimAlign: 500, want: "Prey is dead! R.I.P."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, out := croakTestWorld(t)
			killer := voiceMob(t, w, 9610, 1001, "the barker", 1, 20, tc.killerAlign, "AGGR24")
			victim := croakPlayer(t, w, "Prey", 1001)
			victim.Alignment = tc.victimAlign
			installBragRoller(t, 0, 5) // case 5

			combat.BragMessage(killer, victim)

			if got := out["Prey"]; !strings.Contains(got, tc.want) {
				t.Fatalf("gossip = %q, want %q among it", got, tc.want)
			}
		})
	}
}

// TestBragPossessiveUsesPortSexEncoding pins C's HSHR(victim) in case 1
// (src/fight.c:1179-1180) against the port's encoding: 0=male, 1=female,
// 2=neutral (MobInstance.GetSex translates the mob file's C order).
func TestBragPossessiveUsesPortSexEncoding(t *testing.T) {
	for _, tc := range []struct {
		sex  int
		want string
	}{
		{sex: 0, want: "his"},
		{sex: 1, want: "her"},
		{sex: 2, want: "its"},
	} {
		w, out := croakTestWorld(t)
		killer := voiceMob(t, w, 9620, 1001, "the barker", 1, 20, -500, "AGGR24")
		victim := croakPlayer(t, w, "Prey", 1001)
		victim.Sex = tc.sex
		installBragRoller(t, 0, 1) // case 1

		combat.BragMessage(killer, victim)

		want := "I killed Prey and looted " + tc.want + " stinkin' corpse!"
		if got := out["Prey"]; !strings.Contains(got, want) {
			t.Fatalf("sex %d: gossip = %q, want %q among it", tc.sex, got, want)
		}
	}
}
