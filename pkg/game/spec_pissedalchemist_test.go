package game

import "testing"

func TestPissedAlchemistFirstPulseUsesDirectSayCommand(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	me := newSpecProcTestMob(t, w, 1001, 20)
	lastMsg()

	if got := specPissedalchemist(w, nil, me, "", ""); got {
		t.Fatal("first reveal should return FALSE after speaking")
	}
	if got, want := me.GetAlignment(), -69; got != want {
		t.Fatalf("alignment = %d, want %d", got, want)
	}
	if got, want := lastMsg(), "A test mob exclaims, 'So, finally come to steal my secret!?!'\r\n"+
		"A test mob exclaims, 'The secret of the Philosopher's Stone!?!'\r\n"; got != want {
		t.Fatalf("first reveal bytes = %q, want %q", got, want)
	}
}

func TestPissedAlchemistAttacksMageAfterDirectSay(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	me := newSpecProcTestMob(t, w, 1001, 20)
	me.SetAlignment(-69)
	player.Class = ClassMageUser
	engine := &testCombatEngine{}
	w.SetCombatEngine(engine)
	lastMsg()

	if got := specPissedalchemist(w, nil, me, "", ""); !got {
		t.Fatal("attack branch should return TRUE")
	}
	if len(engine.starts) != 1 || engine.starts[0] != (testCombatPair{attacker: me.GetName(), defender: player.GetName()}) {
		t.Fatalf("combat starts = %#v, want the mob to attack the mage", engine.starts)
	}
	if got, want := lastMsg(), "A test mob exclaims, 'Well -- Ya can't have it!! Die!!!'\r\n"; got != want {
		t.Fatalf("attack speech = %q, want %q", got, want)
	}
}

func TestPissedAlchemistRejectsPlayerCommandAndFightingMob(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	me := newSpecProcTestMob(t, w, 1001, 20)
	if got := specPissedalchemist(w, player, me, "look", ""); got {
		t.Fatal("player command should not run autonomous special")
	}
	me.SetFightingBody(player)
	if got := specPissedalchemist(w, nil, me, "", ""); got {
		t.Fatal("already-fighting mobile should return FALSE")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("gated special emitted bytes %q", got)
	}
}
