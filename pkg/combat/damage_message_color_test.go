package combat

import (
	"strings"
	"testing"
)

// damage_message()'s color sandwich is player-facing law (src/fight.c:995-1015):
//
//	send_to_char(CCYEL(ch, C_CMP), ch);
//	act(dam_weapons[msgnum].to_char, FALSE, ch, NULL, victim, TO_CHAR);
//	send_to_char(CCNRM(ch, C_CMP), ch);
//	send_to_char(CCRED(victim, C_CMP), victim);
//	act(dam_weapons[msgnum].to_victim, FALSE, ch, NULL, victim, TO_VICT | TO_SLEEP);
//	send_to_char(CCNRM(victim, C_CMP), victim);
//
// The room act() (TO_NOTVICT) carries no color at all. These tests fail if the
// escapes are dropped (the pre-audit port) or if the C_CMP gate is loosened to
// C_NRM.
func TestDamMessageColorFramingAtCMP(t *testing.T) {
	cases := []struct {
		name       string
		colorLevel int
		wantYellow string
		wantRed    string
	}{
		{"complete color brackets both lines", colorLevelCMP, "\x1b[33m", "\x1b[31m"},
		{"normal color emits no escapes", 2, "", ""},
		{"no color emits no escapes", 0, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := GetCallbacks()
			t.Cleanup(func() { SetCallbacks(original) })

			var events []string
			cb := defaultCombatCallbacks()
			cb.Broadcast = func(_ int, msg string, _ []Combatant) {
				events = append(events, "room:"+msg)
			}
			cb.SendToChar = func(body Combatant, msg string) {
				events = append(events, "to:"+body.GetName()+":"+msg)
			}
			cb.SendRaw = func(body Combatant, msg string) {
				events = append(events, "raw:"+body.GetName()+":"+msg)
			}
			cb.GetColorLevel = func(Combatant) int { return tc.colorLevel }
			SetCallbacks(cb)

			attacker := &mockCombatant{name: "Attacker", room: 100, sex: 0, position: PosStanding}
			victim := &mockCombatant{name: "Victim", room: 100, sex: 1, position: PosStanding}
			// Damage 5 selects tier 3 ("$n #W $N."), attackType 0 = "hit".
			DamMessage(5, attacker, victim, 0)

			want := []string{"room:Attacker hits Victim."}
			if tc.wantYellow != "" {
				want = append(want, "raw:Attacker:"+tc.wantYellow)
			}
			want = append(want, "to:Attacker:You hit Victim.")
			if tc.wantYellow != "" {
				want = append(want, "raw:Attacker:"+skillColorReset)
			}
			if tc.wantRed != "" {
				want = append(want, "raw:Victim:"+tc.wantRed)
			}
			want = append(want, "to:Victim:Attacker hits you.")
			if tc.wantRed != "" {
				want = append(want, "raw:Victim:"+skillColorReset)
			}

			if got := strings.Join(events, "\n"); got != strings.Join(want, "\n") {
				t.Errorf("damage_message write sequence =\n%s\nwant\n%s", got, strings.Join(want, "\n"))
			}
		})
	}
}

// The escapes are direct writes outside act()'s SENDOK gate, so a recipient
// whose message is suppressed (sleeping TO_CHAR, PLR_WRITING) still receives
// them — the same shape skill_message has (src/comm.c:2481-2484).
func TestDamMessageColorBracketsSurviveSendOKSuppression(t *testing.T) {
	original := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(original) })

	var events []string
	cb := defaultCombatCallbacks()
	cb.Broadcast = func(_ int, msg string, _ []Combatant) { events = append(events, "room:"+msg) }
	cb.SendToChar = func(body Combatant, msg string) {
		events = append(events, "to:"+body.GetName()+":"+msg)
	}
	cb.SendRaw = func(body Combatant, msg string) {
		events = append(events, "raw:"+body.GetName()+":"+msg)
	}
	cb.GetColorLevel = func(Combatant) int { return colorLevelCMP }
	cb.HasPlrFlag = func(body Combatant, flag string) bool {
		return body.GetName() == "Attacker" && flag == "writing"
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Attacker", room: 100, sex: 0, position: PosStanding}
	victim := &mockCombatant{name: "Victim", room: 100, sex: 1, position: PosStanding}
	DamMessage(5, attacker, victim, 0)

	want := []string{
		"room:Attacker hits Victim.",
		"raw:Attacker:" + skillColorYellow,
		"raw:Attacker:" + skillColorReset,
		"raw:Victim:" + skillColorRed,
		"to:Victim:Attacker hits you.",
		"raw:Victim:" + skillColorReset,
	}
	if got := strings.Join(events, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("writing attacker sequence =\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

// damage()'s wounded-band line gates on C_SPR (1), not C_CMP (3), so a player
// with a single PRF_COLOR bit still sees the red (src/fight.c:1593-1597). The
// reset lands before the CRLF because C builds the whole line with one sprintf.
func TestBleedLineColorGatedAtSPR(t *testing.T) {
	redLine := "\x1b[31m" + bleedLineWoundedText + "\x1b[0m\r\n"
	cases := []struct {
		name       string
		colorLevel int
		want       string
	}{
		{"sparse color sees red", colorLevelSPR, redLine},
		{"normal color sees red", 2, redLine},
		{"complete color sees red", colorLevelCMP, redLine},
		{"no color stays plain", 0, bleedLineWoundedText + "\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := GetCallbacks()
			t.Cleanup(func() { SetCallbacks(original) })
			cb := defaultCombatCallbacks()
			cb.GetColorLevel = func(Combatant) int { return tc.colorLevel }
			SetCallbacks(cb)

			victim := &mockCombatant{name: "Victim", room: 100, position: PosStanding}
			if got := BleedLine(victim); got != tc.want {
				t.Errorf("BleedLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The position switch must route its wounded-band line through BleedLine, not a
// bare literal: a colored defender under max_hit/4 receives the C_SPR red
// sandwich. This is the call-site half of the audit fix (engine.go:1011-1012).
func TestHandleSurvivingVictimStateBleedingLineCarriesColor(t *testing.T) {
	original := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(original) })

	attacker := &msgMockCombatant{mockCombatant: mockCombatant{
		name: "a guard trainee", npc: true, room: 8105, hp: 20, maxHP: 20,
		position: PosFighting, fighting: &mockCombatant{name: "Cfighter"},
	}}
	defender := &msgMockCombatant{mockCombatant: mockCombatant{
		name: "Cfighter", room: 8105, hp: 4, maxHP: 20,
		position: PosFighting, fighting: &mockCombatant{name: "a guard trainee"},
	}}
	attacker.SetFightingBody(defender)
	defender.SetFightingBody(attacker)

	// Damage 3 is under max_hit/4, so no "That really did HURT!" draw; the
	// defender's 4 HP is under max_hit/4, so the wounded-band line fires.
	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		GetColorLevel: func(Combatant) int { return colorLevelCMP },
		GetWimpyLev:   func(Combatant) int { return 0 },
	})

	ce.handleSurvivingVictimState(attacker, defender, 3, PosFighting)

	want := "\x1b[31m" + bleedLineWoundedText + "\x1b[0m\r\n"
	if got := strings.Join(defender.messages, ""); got != want {
		t.Errorf("wounded-band line = %q, want %q", got, want)
	}
}
