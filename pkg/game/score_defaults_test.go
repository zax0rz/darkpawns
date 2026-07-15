package game

import "testing"

func TestCanonicalHometownNames(t *testing.T) {
	want := []string{
		"!Bad Hometown - Tell a God!",
		"Kir Drax'in",
		"Kir-Oshi",
		"Alaozar",
	}
	for hometown, name := range want {
		if got := HometownName(hometown); got != name {
			t.Errorf("HometownName(%d) = %q, want %q", hometown, got, name)
		}
	}
	for _, hometown := range []int{-1, len(want)} {
		if got := HometownName(hometown); got != want[0] {
			t.Errorf("HometownName(%d) = %q, want sentinel %q", hometown, got, want[0])
		}
	}
}

func TestNewCharacterGetsDefaultClassTitle(t *testing.T) {
	for class, title := range Titles {
		player := NewCharacter(class+1, "Newbie", class, RaceHuman)
		if player.Title != title {
			t.Errorf("class %d title = %q, want %q", class, player.Title, title)
		}
	}
}
