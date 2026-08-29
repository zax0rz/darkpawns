package game

import (
	"testing"
)

func TestSpecCuchi_CommandGate(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	lastMsg()

	for _, tt := range []struct {
		name string
		ch   *Player
		cmd  string
	}{
		{name: "autonomous", ch: nil, cmd: ""},
		{name: "other-command", ch: player, cmd: "look"},
		{name: "missing-player", ch: nil, cmd: "pat"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if specCuchi(w, tt.ch, mob, tt.cmd, "ignored") {
				t.Fatalf("%s returned true, want false", tt.name)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("%s output = %q, want empty", tt.name, got)
			}
		})
	}
}

func TestSpecCuchi_OrdinaryPat(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 10)
	lastMsg()

	startGold := player.Gold
	if !specCuchi(w, player, mob, "pat", "anything") {
		t.Fatal("ordinary pat should return true")
	}
	if got := player.Gold; got != startGold+10 {
		t.Errorf("ordinary pat gold = %d, want %d", got, startGold+10)
	}
	want := "You pat Cuchi on the head and rub around her ears.\r\n" +
		"Tester pats Cuchi on the head and rubs around her ears.\r\n" +
		"Cuchi purrs at you and bestows a gift from the gods.\r\n" +
		"Cuchi purrs at Tester and bestows a gift from the gods.\r\n"
	if got := lastMsg(); got != want {
		t.Errorf("ordinary pat output = %q, want %q", got, want)
	}
}

func TestSpecCuchi_OrodrethPromotion(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 10)
	lastMsg()

	player.Name = "Orodreth"
	startGold := player.Gold
	if !specCuchi(w, player, mob, "pat", "ignored") {
		t.Fatal("Orodreth pat should return true")
	}
	if got := player.GetLevel(); got != LVL_IMPL {
		t.Errorf("Orodreth level = %d, want LVL_IMPL (%d)", got, LVL_IMPL)
	}
	if got := player.Gold; got != startGold {
		t.Errorf("Orodreth gold = %d, want unchanged %d", got, startGold)
	}
	want := "You pat Cuchi on the head and rub around her ears.\r\n" +
		"Orodreth pats Cuchi on the head and rubs around her ears.\r\n" +
		"Cuchi purrs at you contently.\r\n" +
		"Cuchi purrs contently at Orodreth.\r\n"
	if got := lastMsg(); got != want {
		t.Errorf("Orodreth pat output = %q, want %q", got, want)
	}
}
