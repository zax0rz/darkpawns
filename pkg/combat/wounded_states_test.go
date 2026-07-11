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
