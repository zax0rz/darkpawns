package command

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestShootAudienceOutput(t *testing.T) {
	for _, npc := range []bool{false, true} {
		for _, hit := range []bool{false, true} {
			t.Run(fmt.Sprintf("npc=%t/hit=%t", npc, hit), func(t *testing.T) {
				skill := 1
				if hit {
					skill = 200
				}
				s, _, v, _, wire := shootOutcomeFixture(t, npc, skill)
				dprng.ResetStream(71)
				fireOutcome(t, s, v)
				wantOrigin := "Shooter fires an arrow with a bow.\r\n"
				name := "Victim"
				if npc {
					name = "a guard"
				}
				action := "narrowly misses"
				if hit {
					action = "strikes"
				}
				wantDest := fmt.Sprintf("Some kind of arrow streaks in from the south and %s %s!\r\n", action, name)
				if npc && hit {
					wantOrigin += "A guard bursts into the room and scowls at Shooter.\r\n\r\n"
				}
				if wire["Observer0"] != wantOrigin || wire["Observer1"] != wantDest {
					t.Fatalf("audiences=%q/%q want %q/%q", wire["Observer0"], wire["Observer1"], wantOrigin, wantDest)
				}
				if !npc {
					action = "just misses"
					if hit {
						action = "hits"
					}
					wantVict := fmt.Sprintf("Some kind of arrow streaks in from the south and %s you!\r\n", action)
					if wire["Victim"] != wantVict {
						t.Fatalf("victim=%q want %q", wire["Victim"], wantVict)
					}
				}
			})
		}
	}
}
