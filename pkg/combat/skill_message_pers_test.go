package combat

import (
	"strings"
	"testing"
)

// TestSkillMessageMasksUnseenActorPerReceiver pins DP-1481 shape 3. C's
// skill_message acts render $n/$N per recipient through PERS() — the name when
// CAN_SEE holds, "someone" otherwise (src/fight.c:1039-1090; src/utils.h:515) —
// so a blind recipient reads "someone". Broadcasting one pre-rendered string
// leaked the attacker's real name to every observer in the room.
func TestSkillMessageMasksUnseenActorPerReceiver(t *testing.T) {
	original := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(original) })

	attacker := &mockCombatant{name: "Attacker", position: PosStanding}
	victim := &mockCombatant{name: "Victim", position: PosStanding}
	sighted := &mockCombatant{name: "Sighted", position: PosStanding}
	blind := &mockCombatant{name: "Blind", position: PosStanding}

	rendered := map[string]string{}
	cb := &GameCallbacks{
		BroadcastPerReceiver: func(_ int, render func(Combatant) string, _ []Combatant) {
			for _, to := range []Combatant{sighted, blind} {
				rendered[to.GetName()] = render(to)
			}
		},
		// The legacy single-string path: recording it here keeps the revert
		// proof honest — without the per-receiver seam the blind recipient
		// receives the same real-name string as everyone else.
		Broadcast: func(_ int, msg string, _ []Combatant) {
			for _, to := range []Combatant{sighted, blind} {
				rendered[to.GetName()] = msg
			}
		},
		SendToChar:    func(Combatant, string) {},
		SendRaw:       func(Combatant, string) {},
		GetSex:        func(Combatant) int { return 0 },
		GetHP:         func(Combatant) int { return 50 },
		GetLevel:      func(Combatant) int { return 1 },
		IsNPC:         func(Combatant) bool { return false },
		GetColorLevel: func(Combatant) int { return 0 },
		// The blind recipient cannot see anyone; the sighted one can.
		CanSee: func(observer, subject Combatant) bool { return observer.GetName() != "Blind" },
	}
	SetCallbacks(cb)
	InitFightMessages(cb, FightMessages{
		901: {{Hit: FightMessageAction{
			Attacker: "$n hits $N.",
			Victim:   "$n hits you.",
			Room:     "$n hits $N.",
		}}},
	})

	WithRoller(NewScriptedRoller([]int{1}), func() {
		if handled := cb.SkillMessage(5, attacker, victim, 901, 100); !handled {
			t.Fatal("set 901 was not handled")
		}
	})

	if got := rendered["Sighted"]; !strings.Contains(got, "Attacker hits Victim") {
		t.Errorf("sighted recipient room line = %q, want the real names", got)
	}
	if got := rendered["Blind"]; !strings.Contains(got, "Someone hits someone") {
		t.Errorf("blind recipient room line = %q, want PERS() masking to \"someone\"", got)
	}
}
