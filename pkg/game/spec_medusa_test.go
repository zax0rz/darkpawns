package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func medusaTestMob(t *testing.T, w *World, player *Player) *MobInstance {
	t.Helper()
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.Prototype.Keywords = "Medusa Echidna"
	mob.Prototype.ShortDesc = "Echidna, the greater medusa"
	return mob
}

func medusaSeed(t *testing.T, saves bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 1000; seed++ {
		roll := dprng.New(seed).Number(0, 99)
		if (roll > 75) == saves { // level-1 warrior SAVING_PETRI is 75.
			return seed
		}
	}
	t.Fatal("could not find a deterministic medusa save seed")
	return 0
}

func TestSpecMedusa_EntryGatesAndFightingDelegation(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := medusaTestMob(t, w, player)
	lastMsg()

	for _, tc := range []struct {
		name string
		ch   *Player
		cmd  string
		arg  string
	}{
		{name: "missing player", ch: nil, cmd: "look", arg: "medusa"},
		{name: "wrong command", ch: player, cmd: "say", arg: "medusa"},
		{name: "wrong target", ch: player, cmd: "look", arg: "troll"},
		{name: "blank target", ch: player, cmd: "examine", arg: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := specMedusa(w, tc.ch, mob, tc.cmd, tc.arg); got {
				t.Fatal("medusa gate unexpectedly handled command")
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("medusa gate emitted %q", got)
			}
		})
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	dprng.ResetStream(1)
	if !specMedusa(w, nil, mob, "", "") {
		t.Fatal("fighting commandless medusa did not delegate to magic_user")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("delegated magic_user emitted %q", got)
	}
}

func TestSpecMedusa_SaveFallsThrough(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := medusaTestMob(t, w, player)
	player.SetClass(ClassWarrior)
	player.SetLevel(1)
	lastMsg()

	dprng.ResetStream(medusaSeed(t, true))
	if got := specMedusa(w, player, mob, "look", "medusa"); got {
		t.Fatal("saved medusa look should fall through to ordinary look")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("saved medusa look emitted %q", got)
	}
	if player.Deaths != 0 {
		t.Fatalf("saved medusa look changed deaths to %d", player.Deaths)
	}
}

func TestSpecMedusa_PetrifyAudienceAndState(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", player.GetRoomVNum())
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := medusaTestMob(t, w, player)
	player.SetClass(ClassWarrior)
	player.SetLevel(1)
	player.SetExp(100)
	lastMsg()

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }
	dprng.ResetStream(medusaSeed(t, false))
	if !specMedusa(w, player, mob, "look", "medusa") {
		t.Fatal("failed medusa save did not handle the look")
	}
	if got, want := transcript[player.GetName()],
		"With growing horror and increasing agony, your body slowly turns to stone!\r\n"; got != want {
		t.Errorf("actor transcript = %q, want %q", got, want)
	}
	if got, want := transcript[observer.GetName()],
		"With a sound like that of a crashing wave, Tester slowly turns to stone!\r\n"+
			"Your blood freezes as you hear Tester's death cry.\r\n"; got != want {
		t.Errorf("observer transcript = %q, want %q", got, want)
	}
	if got := player.Deaths; got != 1 {
		t.Fatalf("deaths = %d, want 1", got)
	}
	if got := player.GetExp(); got != 99 {
		t.Fatalf("exp = %d, want 99 after level-cubed loss", got)
	}
	if got := player.GetPosition(); got != combat.PosDead {
		t.Fatalf("position = %d, want dead", got)
	}
}
