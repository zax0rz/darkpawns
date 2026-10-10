package game

import "testing"

func newInactiveRegenTest(t *testing.T) (*World, *Player) {
	t.Helper()
	w, p, _, _ := tickDamageWorld(t)
	p.SetLevel(10)
	p.Class = ClassMageUser
	p.SetPosition(PosResting)
	p.SetMaxHP(1000)
	p.SetMaxMana(1000)
	p.SetMaxMove(1000)
	p.SetHP(100)
	p.SetMana(100)
	p.SetMove(100)
	p.SetCondition(CondFull, 20)
	p.SetCondition(CondThirst, 20)
	p.SetCondition(CondDrunk, 0)
	return w, p
}

func TestPointUpdateInactiveRegen(t *testing.T) {
	t.Run("inactive resources do not regenerate", func(t *testing.T) {
		w, p := newInactiveRegenTest(t)
		p.SetPlrFlag(PrfInactive, true)
		before := p.VitalsSnapshot()

		w.PointUpdate()

		after := p.VitalsSnapshot()
		if after.Health != before.Health || after.Mana != before.Mana || after.Move != before.Move {
			t.Fatalf("inactive vitals changed: before %d/%d/%d, after %d/%d/%d; want no regeneration",
				before.Health, before.Mana, before.Move, after.Health, after.Mana, after.Move)
		}
	})

	t.Run("active resources regenerate", func(t *testing.T) {
		w, p := newInactiveRegenTest(t)
		before := p.VitalsSnapshot()
		if w.HitGain(p) <= 0 || w.ManaGain(p) <= 0 || w.MoveGain(p) <= 0 {
			t.Fatal("fixture must give each resource a positive gain")
		}

		w.PointUpdate()

		after := p.VitalsSnapshot()
		if after.Health <= before.Health || after.Mana <= before.Mana || after.Move <= before.Move {
			t.Fatalf("active vitals did not all regenerate: before %d/%d/%d, after %d/%d/%d",
				before.Health, before.Mana, before.Move, after.Health, after.Mana, after.Move)
		}
	})

	t.Run("poison still damages inactive player", func(t *testing.T) {
		w, p := newInactiveRegenTest(t)
		p.SetPlrFlag(PrfInactive, true)
		p.SetAffect(AffPoison, true)

		w.PointUpdate()

		if got := p.GetHP(); got != 90 {
			t.Fatalf("inactive poisoned HP=%d, want 90 (poison damage remains outside regen gate)", got)
		}
	})

	t.Run("cutthroat still damages inactive player", func(t *testing.T) {
		w, p := newInactiveRegenTest(t)
		p.SetPlrFlag(PrfInactive, true)
		p.SetAffect(AffCutthroat, true)

		w.PointUpdate()

		if got := p.GetHP(); got != 87 {
			t.Fatalf("inactive cutthroat HP=%d, want 87 (cutthroat damage remains outside regen gate)", got)
		}
	})
}
