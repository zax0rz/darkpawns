package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestForceNotificationVisibilityAndSleep(t *testing.T) {
	for _, branch := range []string{"Logvictim", "room", "all"} {
		for _, asleep := range []bool{false, true} {
			t.Run(branch+map[bool]string{false: "-invisible", true: "-sleeping"}[asleep], func(t *testing.T) {
				_, actor, victim, _ := wizardLogFixture(t)
				actor.player.SetInvisLevel(38)
				if asleep {
					victim.player.SetPosition(combat.PosSleeping)
				}
				if err := cmdForce(actor, []string{branch, "wake"}); err != nil {
					t.Fatal(err)
				}
				got := strings.Join(drainSessionText(t, victim), "")
				if asleep && strings.Contains(got, "has forced you") {
					t.Fatalf("sleeping victim received notice: %q", got)
				}
				if !asleep && !strings.Contains(got, "Someone has forced you to 'wake'.\r\n") {
					t.Fatalf("invisible actor notice = %q", got)
				}
				if victim.player.GetPosition() == combat.PosSleeping {
					t.Fatal("forced wake did not execute")
				}
			})
		}
	}
}

func TestForceShutdownNotificationAct(t *testing.T) {
	for _, asleep := range []bool{false, true} {
		t.Run(map[bool]string{false: "invisible", true: "sleeping"}[asleep], func(t *testing.T) {
			m, a, v, _ := wizardLogFixture(t)
			a.player.SetInvisLevel(38)
			if asleep {
				v.player.SetPosition(combat.PosSleeping)
			}
			m.forceAllSave(a)
			got := strings.Join(drainSessionText(t, v), "")
			if asleep && strings.Contains(got, "has forced you") {
				t.Fatalf("sleeping shutdown notice: %q", got)
			}
			if !asleep && !strings.Contains(got, "Someone has forced you to 'save'.\r\n") {
				t.Fatalf("shutdown notice: %q", got)
			}
		})
	}
}
