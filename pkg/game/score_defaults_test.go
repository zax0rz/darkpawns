package game

import "testing"

func TestCanonicalHometownNames(t *testing.T) {
	want := []string{
		"!Bad Hometown - Tell a God!",
		"Kir Drax'in",
		"Kir-Oshi",
		"Alaozar",
		"Add a new hometown here!",
	}
	for hometown, name := range want {
		if got := HometownName(hometown); got != name {
			t.Errorf("HometownName(%d) = %q, want %q", hometown, got, name)
		}
	}
	if got := HometownName(len(want)); got != "\n" {
		t.Errorf("HometownName(%d) = %q, want sentinel %q", len(want), got, "\n")
	}
	for _, hometown := range []int{-1, len(want) + 1} {
		if got := HometownName(hometown); got != want[0] {
			t.Errorf("HometownName(%d) = %q, want fallback %q", hometown, got, want[0])
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
