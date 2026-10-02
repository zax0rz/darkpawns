package session

import (
	"fmt"
	"sort"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// Independent C roll specification: src/class.c:380-497. Sorting six totals
// directly differs from the production insertion loop. The minotaur warrior
// can draw STR_ADD both before and after its race bonus.
func cCreationWarriorRoll(r *dprng.Generator) game.CharStats {
	totals := make([]int, 6)
	for i := range totals {
		dice := []int{r.Number(1, 6), r.Number(1, 6), r.Number(1, 6), r.Number(1, 6)}
		sort.Ints(dice)
		totals[i] = dice[1] + dice[2] + dice[3]
	}
	sort.Sort(sort.Reverse(sort.IntSlice(totals)))
	stats := game.CharStats{Str: totals[0], Dex: totals[1], Con: totals[2], Wis: totals[3], Int: totals[4], Cha: totals[5]}
	if stats.Str == 18 {
		stats.StrAdd = r.Number(0, 100)
	}
	stats.Str = min(stats.Str+1, 18)
	if stats.Str == 18 {
		stats.StrAdd = r.Number(0, 100)
	}
	return stats
}

// R3/R5h: invalid leaves scores and RNG alone; rerolls replace all values;
// first-byte accept preserves the last values into init_char for God/mortal.
func TestEntryStatsRerollMatrix(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		for _, god := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/god=%v", seed, god), func(t *testing.T) {
				t.Setenv("DP_CLOCK", "1")
				t.Setenv("DP_FRESH_MUD", "")
				if god {
					t.Setenv("DP_FRESH_MUD", "1")
				}
				s := makeCharSession(t, makeTestManager(t))
				s.charCreating = true
				s.charName = "Rollhero"
				s.charStage = "hometown"
				s.charClass = game.ClassWarrior
				s.charRace = game.RaceMinotaur
				dprng.ResetStream(seed)
				ref := dprng.New(seed)
				want := cCreationWarriorRoll(ref)
				sendCharInput(t, s, "K")
				_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
				if s.charStats != want || prompt.Stats == nil {
					t.Fatalf("initial roll=%+v want=%+v", s.charStats, want)
				}
				sendCharInput(t, s, "invalid")
				_, prompt = unmarshalCharCreate(t, drainMsg(t, s))
				if s.charStats != want || prompt.Prompt != "Invalid choice! Select 'Y' or 'N':" {
					t.Fatalf("invalid changed state: %+v %+v", s.charStats, prompt)
				}
				for _, input := range []string{"no again", "  N reroll"} {
					want = cCreationWarriorRoll(ref)
					sendCharInput(t, s, input)
					_, prompt = unmarshalCharCreate(t, drainMsg(t, s))
					if s.charStats != want || prompt.Stats == nil || prompt.Stats.Str != want.Str || prompt.Stats.Dex != want.Dex || prompt.Stats.Int != want.Int || prompt.Stats.Wis != want.Wis || prompt.Stats.Con != want.Con || prompt.Stats.Cha != want.Cha {
						t.Fatalf("reroll %q: stats=%+v want=%+v prompt=%+v", input, s.charStats, want, prompt)
					}
				}
				// src/db.c:3038-3048: accepted male body draws weight then height.
				weight, height := ref.Number(120, 180), ref.Number(160, 200)
				sendCharInput(t, s, "  Yes keep")
				_ = drainMsg(t, s)
				if s.charStage != "motd" || !s.creationSaved || s.player == nil {
					t.Fatalf("accept did not persist: stage=%q", s.charStage)
				}
				if s.player.Stats != want || s.player.Weight != weight || s.player.Height != height {
					t.Fatalf("accepted values=%+v weight=%d height=%d want=%+v %d %d", s.player.Stats, s.player.Weight, s.player.Height, want, weight, height)
				}
				if s.player.GetLevel() != map[bool]int{false: 0, true: game.LVL_IMPL}[god] {
					t.Fatal("wrong God/mortal level")
				}
				if got := dprng.Next(); got != ref.Next() {
					t.Fatalf("draw stream desynced: %d", got)
				}
			})
		}
	}
}
