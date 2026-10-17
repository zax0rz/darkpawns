package combat

import (
	"strings"
	"testing"
)

// skillMessageRig records every audience write the wired SkillMessage makes, in
// call order, so a test can assert both the branch text and the exact
// interleaving of the bare color writes C performs around it.
type skillMessageRig struct {
	cb     *GameCallbacks
	events []string
}

// newSkillMessageRig wires InitFightMessages over a synthetic single-variant
// set 900 so each branch's expected text is explicit and the dice(1,1) draw is
// exercised exactly as in production.
func newSkillMessageRig(t *testing.T, hp int, victimLevel int, victimIsNPC bool, colorLevel int) *skillMessageRig {
	t.Helper()
	original := GetCallbacks()
	rig := &skillMessageRig{}
	rig.cb = &GameCallbacks{
		Broadcast: func(_ int, msg string, _ []Combatant) { rig.events = append(rig.events, "room:"+msg) },
		SendToChar: func(bodyname Combatant, msg string) {
			name := bodyname.GetName()

			rig.events = append(rig.events, "to:"+name+":"+msg)
		},
		SendRaw: func(bodyname Combatant, msg string) {
			name := bodyname.GetName()
			rig.events = append(rig.events, "raw:"+name+":"+msg)
		},
		GetSex: func(Combatant) int { return 0 },
		GetHP:  func(Combatant) int { return hp },
		GetLevel: func(bodyname Combatant) int {
			name := bodyname.GetName()

			if name == "Victim" {
				return victimLevel
			}
			return 1
		},
		IsNPC: func(bodyname Combatant) bool {
			name := bodyname.GetName()

			return name == "Victim" && victimIsNPC
		},
		GetColorLevel: func(Combatant) int { return colorLevel },
	}
	SetCallbacks(rig.cb)
	InitFightMessages(rig.cb, FightMessages{
		900: {{
			Die:  FightMessageAction{Attacker: "Die attacker line", Victim: "Die victim line", Room: "Die room line"},
			Miss: FightMessageAction{Attacker: "Miss attacker line", Victim: "Miss victim line", Room: "Miss room line"},
			Hit:  FightMessageAction{Attacker: "Hit attacker line", Victim: "Hit victim line", Room: "Hit room line"},
			God:  FightMessageAction{Attacker: "God attacker line", Victim: "God victim line", Room: "God room line"},
		}},
	})
	t.Cleanup(func() { SetCallbacks(original) })
	return rig
}

func (r *skillMessageRig) run(t *testing.T, dam int) {
	t.Helper()
	WithRoller(NewScriptedRoller([]int{1}), func() {
		attacker := &mockCombatant{name: "Attacker", position: PosStanding}
		victim := &mockCombatant{name: "Victim", position: PosStanding}
		if handled := r.cb.SkillMessage(dam, attacker, victim, 900, 100); !handled {
			t.Fatal("set 900 was not handled")
		}
	})
}

func (r *skillMessageRig) joined() string {
	return strings.Join(r.events, "\n")
}

func assertSkillMessageEvents(t *testing.T, rig *skillMessageRig, want []string) {
	t.Helper()
	if got := rig.joined(); got != strings.Join(want, "\n") {
		t.Errorf("skill_message write sequence =\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

// TestSkillMessageGodBranchBeatsZeroDamage — fight.c:1039 tests the immortal
// victim BEFORE the dam != 0 split, so a zero-damage strike against a God emits
// god_msg rather than miss_msg, and the god arm is uncolored (R1/R5e).
func TestSkillMessageGodBranchBeatsZeroDamage(t *testing.T) {
	rig := newSkillMessageRig(t, 10, LVL_IMMORT, false, 3)
	rig.run(t, 0)

	assertSkillMessageEvents(t, rig, []string{
		"room:God room line",
		"to:Attacker:God attacker line",
		"to:Victim:God victim line",
	})
}

// TestSkillMessageBranchOrderByDamageAndPosition — C's dam != 0 split selects
// die_msg at POS_DEAD (GET_HIT <= -11) and hit_msg otherwise; a zero-damage
// strike only reaches miss_msg when ch != vict (fight.c:1045-1090).
func TestSkillMessageBranchOrderByDamageAndPosition(t *testing.T) {
	cases := []struct {
		name       string
		dam        int
		hp         int
		want       string
		wantVictim string
		wantRoom   string
	}{
		{"nonzero damage above POS_DEAD selects hit", 5, 10, "Hit attacker line", "Hit victim line", "Hit room line"},
		{"nonzero damage at POS_DEAD selects die", 5, -12, "Die attacker line", "Die victim line", "Die room line"},
		{"zero damage selects miss", 0, 10, "Miss attacker line", "Miss victim line", "Miss room line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rig := newSkillMessageRig(t, tc.hp, 1, true, 0)
			rig.run(t, tc.dam)
			assertSkillMessageEvents(t, rig, []string{
				"room:" + tc.wantRoom,
				"to:Attacker:" + tc.want,
				"to:Victim:" + tc.wantVictim,
			})
		})
	}
}

// TestSkillMessageDeadSelfTickMatchesActSleepGates — poison and other
// point-update damage call damage(ch, ch, ...). C's TO_CHAR attacker act is
// suppressed after the position update reaches POS_DEAD, while TO_VICT|TO_SLEEP
// still delivers the victim line. The direct color writes bracket the act in
// C even when SENDOK suppresses its message (src/comm.c:2481-2510).
func TestSkillMessageDeadSelfTickMatchesActSleepGates(t *testing.T) {
	rig := newSkillMessageRig(t, -12, 1, true, 3)
	self := &mockCombatant{name: "Victim", position: PosDead}
	WithRoller(NewScriptedRoller([]int{1}), func() {
		if handled := rig.cb.SkillMessage(5, self, self, 900, 100); !handled {
			t.Fatal("set 900 was not handled")
		}
	})

	assertSkillMessageEvents(t, rig, []string{
		"room:Die room line",
		"raw:Victim:\x1b[33m",
		"raw:Victim:\x1b[0m",
		"raw:Victim:\x1b[31m",
		"to:Victim:Die victim line",
		"raw:Victim:\x1b[0m",
	})
}

// TestSkillMessageGodVictimUsesAwakeGate — the god branch's TO_VICT act omits
// TO_SLEEP, so a sleeping immortal receives neither its victim line nor room
// line, while the awake attacker still receives the TO_CHAR line.
func TestSkillMessageGodVictimUsesAwakeGate(t *testing.T) {
	rig := newSkillMessageRig(t, 10, LVL_IMMORT, false, 0)
	attacker := &mockCombatant{name: "Attacker", position: PosStanding}
	victim := &mockCombatant{name: "Victim", position: PosSleeping}
	WithRoller(NewScriptedRoller([]int{1}), func() {
		if handled := rig.cb.SkillMessage(0, attacker, victim, 900, 100); !handled {
			t.Fatal("set 900 was not handled")
		}
	})

	assertSkillMessageEvents(t, rig, []string{
		"room:God room line",
		"to:Attacker:God attacker line",
	})
}

// TestSkillMessageDirectActHonorsWritingGate — comm.c SENDOK also suppresses
// direct act() output while a player is in PLR_WRITING.
func TestSkillMessageDirectActHonorsWritingGate(t *testing.T) {
	rig := newSkillMessageRig(t, 10, 1, true, 0)
	rig.cb.HasPlrFlag = func(body Combatant, flag string) bool {
		return body.GetName() == "Attacker" && flag == "writing"
	}
	attacker := &mockCombatant{name: "Attacker", position: PosStanding}
	victim := &mockCombatant{name: "Victim", position: PosStanding}
	WithRoller(NewScriptedRoller([]int{1}), func() {
		if handled := rig.cb.SkillMessage(5, attacker, victim, 900, 100); !handled {
			t.Fatal("set 900 was not handled")
		}
	})

	assertSkillMessageEvents(t, rig, []string{
		"room:Hit room line",
		"to:Victim:Hit victim line",
	})
}

// TestSkillMessageSelfZeroDamageEmitsNothing — when dam is 0 and ch == vict, C
// falls through skill_message with no act() at all, after the dice draw. The
// variant draw must still be consumed (fight.c:1078).
func TestSkillMessageSelfZeroDamageEmitsNothing(t *testing.T) {
	rig := newSkillMessageRig(t, 10, 1, true, 0)
	roller := NewScriptedRoller([]int{1, 7})
	WithRoller(roller, func() {
		attackerBody := testCombatBody("Attacker")
		if handled := rig.cb.SkillMessage(0, attackerBody, attackerBody, 900, 100); !handled {
			t.Fatal("set 900 was not handled")
		}
	})
	if got := rig.joined(); got != "" {
		t.Errorf("self zero-damage strike emitted %q, want nothing", got)
	}
	if roller.Index != 1 {
		t.Errorf("variant draws = %d, want the single dice(1,1) draw", roller.Index)
	}
}

// TestSkillMessageColorFramingAtCMP — C brackets the attacker line with
// CCYEL/CCNRM and the victim line with CCRED/CCNRM, but only at the complete
// color level (fight.c:1049-1057, 1080-1088); room acts carry no color.
func TestSkillMessageColorFramingAtCMP(t *testing.T) {
	rig := newSkillMessageRig(t, 10, 1, true, 3)
	rig.run(t, 5)

	assertSkillMessageEvents(t, rig, []string{
		"room:Hit room line",
		"raw:Attacker:\x1b[33m",
		"to:Attacker:Hit attacker line",
		"raw:Attacker:\x1b[0m",
		"raw:Victim:\x1b[31m",
		"to:Victim:Hit victim line",
		"raw:Victim:\x1b[0m",
	})
}

// TestSkillMessageColorGatedByLevel — below C_CMP the color macros expand to
// the empty string, so only the message bytes are written.
func TestSkillMessageColorGatedByLevel(t *testing.T) {
	rig := newSkillMessageRig(t, 10, 1, true, 2)
	rig.run(t, 0)

	assertSkillMessageEvents(t, rig, []string{
		"room:Miss room line",
		"to:Attacker:Miss attacker line",
		"to:Victim:Miss victim line",
	})
}
