package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func smackheadsTestTargets(t *testing.T, w *World) (*MobInstance, *MobInstance) {
	t.Helper()
	w.mobs[2002] = &parser.Mob{
		VNum:      2002,
		Keywords:  "second dummy",
		ShortDesc: "a second dummy",
		Level:     1,
		HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 20},
		Damage:    parser.DiceRoll{Num: 1, Sides: 1},
	}
	first := spawnTargetMob(t, w)
	second, err := w.SpawnMob(2002, 1001)
	if err != nil {
		t.Fatalf("SpawnMob second target: %v", err)
	}
	return first, second
}

func TestDoSmackheads_ResultContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	first, second := smackheadsTestTargets(t, w)
	ch.SetSkill(SkillSmackheads, 100)

	var hit SkillResult
	for seed := uint32(1); seed < 100 && !hit.Success; seed++ {
		first.SetPosition(combat.PosStanding)
		second.SetPosition(combat.PosStanding)
		ch.SetFighting("")
		dprng.ResetStream(seed)
		hit = DoSmackheads(ch, "training", "second", w)
	}
	if !hit.Success {
		t.Fatal("no smackheads hit observed in 99 deterministic seeds")
	}
	if hit.Damage != 3*ch.GetLevel() {
		t.Fatalf("hit damage = %d, want %d", hit.Damage, 3*ch.GetLevel())
	}
	if len(hit.Targets) != 2 || hit.Targets[0] != first || hit.Targets[1] != second {
		t.Fatalf("hit targets = %#v, want ordered victims", hit.Targets)
	}
	if hit.DamageSkill != SkillSmackheads || !hit.StartCombat || hit.WaitCh != 3 || hit.WaitTarget != 3 {
		t.Fatalf("hit metadata = %#v, want smackheads/start/three pulse waits", hit)
	}
	if len(hit.DeferredImprove) != 1 || hit.DeferredImprove[0] != SkillSmackheads {
		t.Fatalf("hit deferred improvement = %v, want [%q]", hit.DeferredImprove, SkillSmackheads)
	}
	if first.GetPosition() != combat.PosStunned || second.GetPosition() != combat.PosStunned {
		t.Fatalf("hit positions = %d/%d, want stunned", first.GetPosition(), second.GetPosition())
	}

	first.SetPosition(combat.PosStanding)
	second.SetPosition(combat.PosStanding)
	ch.SetFighting("")
	ch.SetSkill(SkillSmackheads, 1)
	dprng.ResetStream(1)
	miss := DoSmackheads(ch, "training", "second", w)
	if miss.Success || miss.Damage != 0 {
		t.Fatalf("skill-1 smackheads = success %v damage %d, want false/0", miss.Success, miss.Damage)
	}
	if len(miss.Targets) != 2 || miss.Targets[0] != first || miss.Targets[1] != second {
		t.Fatalf("miss targets = %#v, want ordered victims", miss.Targets)
	}
	if miss.DamageSkill != "" || !miss.StartCombat || miss.WaitCh != 3 {
		t.Fatalf("miss metadata = %#v, want no damage skill/start/three pulse wait", miss)
	}
	if len(miss.PreDamageImprove) != 1 || miss.PreDamageImprove[0] != SkillSmackheads {
		t.Fatalf("miss pre-damage improvement = %v, want [%q]", miss.PreDamageImprove, SkillSmackheads)
	}
}

func TestDoSmackheads_GateOrderAndMessages(t *testing.T) {
	t.Run("no skill", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		first, second := smackheadsTestTargets(t, w)
		result := DoSmackheads(ch, first.GetName(), second.GetName(), w)
		if result.MessageToCh != "The only heads you're gonna smack are yours and Rosie's." {
			t.Fatalf("no-skill message = %q", result.MessageToCh)
		}
	})

	tests := []struct {
		name string
		set  func(*World, *Player, *MobInstance, *MobInstance)
		want string
	}{
		{name: "missing target", want: "Looks like the gangs not all here..."},
		{
			name: "same target",
			set:  func(_ *World, _ *Player, _ *MobInstance, _ *MobInstance) {},
			want: "Looks like the gangs not all here...",
		},
		{name: "self target", want: "We call that 'headbutt' around here, son..."},
		{name: "wielded precedes mounted", want: "You need your hands free to smack some heads!"},
		{name: "mounted", want: "Dismount first!"},
		{name: "actor fighting", want: "You're a little busy right now!"},
		{name: "target fighting", want: "They are too busy fighting at the moment!"},
		{name: "peaceful", want: "You can't commit acts of violence here!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newCombatTestWorld(t)
			first, second := smackheadsTestTargets(t, w)
			ch.SetSkill(SkillSmackheads, 100)
			firstName, secondName := "training", "second"
			switch tc.name {
			case "missing target":
				firstName = "nobody"
			case "same target":
				secondName = "training"
			case "self target":
				firstName = ch.GetName()
			case "wielded precedes mounted":
				equipWeapon(t, ch, makeSpikeWeapon("smackheads"))
				ch.SetAffect(affMounted, true)
			case "mounted":
				ch.SetAffect(affMounted, true)
			case "actor fighting":
				ch.SetFighting("another target")
			case "target fighting":
				first.SetFighting("another target")
			case "peaceful":
				room := w.GetRoomInWorld(ch.GetRoomVNum())
				room.Flags = []string{"peaceful"}
			}
			if tc.set != nil {
				tc.set(w, ch, first, second)
			}

			result := DoSmackheads(ch, firstName, secondName, w)
			if result.MessageToCh != tc.want {
				t.Fatalf("message = %q, want %q", result.MessageToCh, tc.want)
			}
		})
	}
}
