package combat

import "testing"

func hiddenAttackerFadeRig(t *testing.T, observerName string) (hidden *bool, chameleonRecord *bool, rendered *string) {
	t.Helper()
	original := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(original) })

	attacker := &mockCombatant{
		name: "ItemBuilder", room: 100, level: 20, hp: 100, maxHP: 100,
		position: PosStanding, class: ClassWarrior, str: 10, dex: 10, intVal: 10, wis: 10,
		damageRoll: DiceRoll{Num: 1, Sides: 1},
	}
	defender := &mockCombatant{
		name: "Target", room: 100, level: 20, hp: 100, maxHP: 100, npc: true,
		position: PosStanding, class: ClassWarrior, str: 10, dex: 10, intVal: 10, wis: 10,
	}
	observer := &mockCombatant{name: observerName, room: 100, position: PosStanding}
	hidden = new(bool)
	*hidden = true
	chameleonRecord = new(bool)
	*chameleonRecord = true
	rendered = new(string)

	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		HasAffect: func(body Combatant, aff int) bool {
			return body == attacker && aff == AFF_HIDE && *hidden
		},
		RemoveAffectBit: func(body Combatant, aff int) {
			if body != attacker || aff != AFF_HIDE {
				t.Errorf("RemoveAffectBit(%s, %d), want attacker AFF_HIDE", body.GetName(), aff)
			}
			*hidden = false
		},
		RemoveAffect: func(Combatant, int) {
			t.Error("damage() removed the chameleon affect record instead of only AFF_HIDE")
			*chameleonRecord = false
		},
		CanSee: func(observer, subject Combatant) bool {
			return observer.GetName() != "Blind"
		},
		BroadcastPerReceiver: func(_ int, render func(Combatant) string, _ []Combatant) {
			*rendered = render(observer)
		},
		Broadcast: func(_ int, message string, _ []Combatant) {
			*rendered = message
		},
	})
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatal(err)
	}
	WithRoller(NewScriptedRoller([]int{20, 1}), func() {
		if err := ce.PerformInitialAttack(attacker, defender); err != nil {
			t.Fatal(err)
		}
	})
	return hidden, chameleonRecord, rendered
}

func TestHiddenAttackerFirstHitFadesAndKeepsChameleonRecord(t *testing.T) {
	hidden, chameleonRecord, rendered := hiddenAttackerFadeRig(t, "Sighted")
	if *hidden {
		t.Error("first hit left AFF_HIDE set")
	}
	if !*chameleonRecord {
		t.Error("first hit removed the chameleon affect record")
	}
	if got, want := *rendered, "ItemBuilder slowly fades into existence."; got != want {
		t.Errorf("fade message = %q, want %q", got, want)
	}
}

func TestHiddenAttackerFadeMasksUnseenAttacker(t *testing.T) {
	_, _, rendered := hiddenAttackerFadeRig(t, "Blind")
	if got, want := *rendered, "Someone slowly fades into existence."; got != want {
		t.Errorf("blind observer fade message = %q, want %q", got, want)
	}
}
