package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestShootActorOutput(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(fmt.Sprint(hit), func(t *testing.T) {
			skill := 1
			if hit {
				skill = 200
			}
			s, _, v, _, _ := shootOutcomeFixture(t, false, skill)
			dprng.ResetStream(71)
			fireOutcome(t, s, v)
			want := "Twang... your projectile flies into the distance.\r\n"
			if hit {
				want += "You hear a roar of pain!\r\n"
			}
			if got := strings.Join(s.getMessages(), ""); got != want {
				t.Fatalf("actor=%q want %q", got, want)
			}
		})
	}
}
