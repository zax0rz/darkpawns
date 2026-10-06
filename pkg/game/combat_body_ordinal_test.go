package game

import "testing"

func TestCombatBodyOrdinalSelection(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	viewer := NewPlayer(1, "Viewer", 1001)
	if err := w.AddPlayer(viewer); err != nil {
		t.Fatal(err)
	}
	check := func(name string, want interface{}) {
		t.Helper()
		got, ok := w.ResolveCharInRoom(viewer, name)
		if !ok || got.Combatant != want {
			t.Fatalf("%s selected %p want %p", name, got.Combatant, want)
		}
	}
	check("1.guard", second)
	check("2.guard", first)
	if err := w.MobTransfer(first, 1001); err != nil {
		t.Fatal(err)
	}
	check("1.guard", first)
	check("2.guard", second)
	pc := NewPlayer(2, "Guard", 1001)
	if err := w.AddPlayer(pc); err != nil {
		t.Fatal(err)
	}
	check("1.guard", pc)
	check("2.guard", first)
}
