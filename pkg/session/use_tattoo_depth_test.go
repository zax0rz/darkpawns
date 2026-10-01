package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// C interpreter.c:796 -> act.other.c:920 -> tattoo.c:31. Exercise the
// live session handler; session/tattoo.go's older helper is not this path.
func TestCmdUseTattooDelegatesToWorld(t *testing.T) {
	m := makeFleeTestManager(t)
	s := makeFleeSession(t, m, "Tattooactor", 100)
	s.player.Tattoo = game.TattooEye
	if err := cmdUse(s, []string{"TaTtOo", "ignored"}); err != nil {
		t.Fatal(err)
	}
	if s.player.TatTimer != 24 || len(s.player.ActiveAffects) != 2 {
		t.Fatal("session did not activate world tattoo")
	}
	for _, a := range s.player.ActiveAffects {
		if a.Duration != 10 {
			t.Fatalf("session cast uses wrong level: %+v", a)
		}
	}
	if err := cmdUse(s, []string{"tattoo"}); err != nil {
		t.Fatal(err)
	}
	if len(s.player.ActiveAffects) != 2 {
		t.Fatal("repeat ignored cooldown")
	}
}
