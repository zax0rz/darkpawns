package session

import (
	"strings"
	"testing"
)

func TestCombatBodyKillCorpseSlash(t *testing.T) {
	m := makeInstakillTestManager(t)
	s := makeInstakillSession(t, m, "Killer", LVL_IMPL)
	if _, err := m.world.SpawnMobQuiet(5000, 1001); err != nil {
		t.Fatal(err)
	}
	if err := cmdKill(s, []string{"target"}); err != nil {
		t.Fatal(err)
	}
	for _, obj := range m.world.GetItemsInRoom(1001) {
		if obj.IsCorpse {
			if !strings.Contains(obj.GetLongDesc(), "hacked up, bloody corpse") {
				t.Fatalf("wrong kill corpse: %q", obj.GetLongDesc())
			}
			return
		}
	}
	t.Fatal("missing corpse")
}
