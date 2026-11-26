package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdSethuntSetsHunterState(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	wizard := makeTestSession(t, m, "Wizard", 1001, true)
	wizard.player.Level = LVL_GRGOD
	registerInWorld(t, wizard)
	victim := makeTestSession(t, m, "Victim", 1001, true)
	victim.player.Level = 1
	registerInWorld(t, victim)
	hunter, err := m.world.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	keywords := strings.Fields(hunter.Proto().Keywords)
	if len(keywords) == 0 {
		t.Fatal("test mob has no keyword")
	}

	if err := cmdSethunt(wizard, []string{victim.player.Name, keywords[0]}); err != nil {
		t.Fatalf("cmdSethunt: %v", err)
	}
	if !hunter.HasMobFlag(game.MobFlagHunter) {
		t.Fatal("hunter flag is not set")
	}
	if got, want := hunter.GetHunting(), victim.player.Name; got != want {
		t.Fatalf("hunting target = %q, want %q", got, want)
	}
}
