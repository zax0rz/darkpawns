package combat

import (
	"strings"
	"testing"
)

// TestUpdatePositionAfterDamage_WoundedBand is the DP-1021 regression: damage
// that drops HP to 0 or below must transition the victim into the wounded band
// (stunned/incap/mortally) or POS_DEAD per fight.c update_pos — NOT kill
// instantly at 0. Only HP <= -11 is POS_DEAD; the helper emits the matching
// wounded message (but no message for death — the death handler owns that) and
// drops the victim's FIGHTING reference once it can no longer fight.
func TestUpdatePositionAfterDamage_WoundedBand(t *testing.T) {
	tests := []struct {
		name        string
		hp          int
		wantPos     int
		wantMsgWord string // substring of the personal message; "" = no message
		wantStopped bool   // FIGHTING reference cleared?
	}{
		{"stunned at -2", -2, PosStunned, "stunned", true},
		{"incap at -4", -4, PosIncap, "incapacitated", true},
		{"mortally at -8", -8, PosMortally, "mortally wounded", true},
		{"mortally at boundary -10", -10, PosMortally, "mortally wounded", true},
		{"dead at -11", -11, PosDead, "", true},
		{"alive fighter preserved", 20, PosFighting, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &msgMockCombatant{
				mockCombatant: mockCombatant{name: "Victim", hp: tt.hp, position: PosFighting, fighting: "Attacker"},
			}
			var broadcasts []string
			got := UpdatePositionAfterDamage(v, func(_ int, msg, _ string) {
				broadcasts = append(broadcasts, msg)
			})

			if got != tt.wantPos {
				t.Errorf("returned pos = %d, want %d", got, tt.wantPos)
			}
			if v.GetPosition() != tt.wantPos {
				t.Errorf("victim position = %d, want %d", v.GetPosition(), tt.wantPos)
			}

			if tt.wantMsgWord == "" {
				if len(v.messages) != 0 {
					t.Errorf("expected no personal message, got %q", v.messages)
				}
				if len(broadcasts) != 0 {
					t.Errorf("expected no room broadcast, got %q", broadcasts)
				}
			} else {
				if len(v.messages) != 1 || !strings.Contains(v.messages[0], tt.wantMsgWord) {
					t.Errorf("personal message = %q, want one containing %q", v.messages, tt.wantMsgWord)
				}
				if len(broadcasts) != 1 || !strings.Contains(broadcasts[0], tt.wantMsgWord) {
					t.Errorf("room broadcast = %q, want one containing %q", broadcasts, tt.wantMsgWord)
				}
			}

			if tt.wantStopped && v.GetFighting() != "" {
				t.Errorf("expected FIGHTING cleared for a downed victim, still %q", v.GetFighting())
			}
			if !tt.wantStopped && v.GetFighting() == "" {
				t.Error("a still-standing fighter should keep its FIGHTING reference")
			}
		})
	}
}

// TestUpdatePositionAfterDamage_IncapExactStrings is the DP-1318 regression: C's
// victim-facing incap message contains the typo "an will slowly die" while the
// room-facing message reads "and will slowly die". R1 — player-facing bytes are law.
func TestUpdatePositionAfterDamage_IncapExactStrings(t *testing.T) {
	v := &msgMockCombatant{
		mockCombatant: mockCombatant{name: "Victim", hp: -4, position: PosFighting, fighting: "Attacker"},
	}
	var broadcasts []string
	UpdatePositionAfterDamage(v, func(_ int, msg, _ string) {
		broadcasts = append(broadcasts, msg)
	})
	if len(v.messages) != 1 {
		t.Fatalf("expected 1 personal message, got %d", len(v.messages))
	}
	wantVictim := "You are incapacitated an will slowly die, if not aided.\r\n"
	if v.messages[0] != wantVictim {
		t.Errorf("victim message = %q, want %q", v.messages[0], wantVictim)
	}
	if len(broadcasts) != 1 {
		t.Fatalf("expected 1 room broadcast, got %d", len(broadcasts))
	}
	wantRoom := "Victim is incapacitated and will slowly die, if not aided."
	if broadcasts[0] != wantRoom {
		t.Errorf("room broadcast = %q, want %q", broadcasts[0], wantRoom)
	}
}

// TestUpdatePositionAfterDamage_NilBroadcast verifies the room broadcast is
// optional (personal message still fires).
func TestUpdatePositionAfterDamage_NilBroadcast(t *testing.T) {
	v := &msgMockCombatant{mockCombatant: mockCombatant{name: "Victim", hp: -4, position: PosFighting}}
	if got := UpdatePositionAfterDamage(v, nil); got != PosIncap {
		t.Fatalf("pos = %d, want %d", got, PosIncap)
	}
	if len(v.messages) != 1 {
		t.Errorf("expected the personal wounded message even with nil broadcast, got %q", v.messages)
	}
}

// TestWoundedBandBroadcastsCapitalizeAndExcludeVictim is the DP-1330 class fix:
// C's act() CAPitalizes the assembled line (comm.c:2477) and TO_ROOM excludes
// the victim, who already received the "You are ..." form (fight.c:1560-1582).
// A lowercase mob name is the sharp case: before the fix the room saw
// "a guard trainee is mortally wounded...", and the victim saw its own room line.
func TestWoundedBandBroadcastsCapitalizeAndExcludeVictim(t *testing.T) {
	cases := []struct {
		name     string
		hp       int
		wantPos  int
		wantRoom string
	}{
		{"mortally", -6, PosMortally, "A guard trainee is mortally wounded, and will die soon, if not aided."},
		{"incap", -4, PosIncap, "A guard trainee is incapacitated and will slowly die, if not aided."},
		{"stunned", -1, PosStunned, "A guard trainee is stunned, but will probably regain consciousness again."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := &msgMockCombatant{
				mockCombatant: mockCombatant{name: "a guard trainee", hp: tc.hp, position: PosFighting},
			}
			var roomMsg, exclude string
			if got := UpdatePositionAfterDamage(v, func(_ int, msg, ex string) {
				roomMsg, exclude = msg, ex
			}); got != tc.wantPos {
				t.Fatalf("pos = %d, want %d", got, tc.wantPos)
			}
			if roomMsg != tc.wantRoom {
				t.Errorf("room broadcast = %q, want %q (act CAP)", roomMsg, tc.wantRoom)
			}
			if exclude != "a guard trainee" {
				t.Errorf("room broadcast excluded %q, want the victim (TO_ROOM excludes ch)", exclude)
			}
		})
	}
}
