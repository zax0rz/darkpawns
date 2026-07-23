package combat

import (
	"strings"
	"testing"
)

// TestPerformOneHit_WeaponOffsetReachesMessageSender — the core DP-1204 fix: the
// weapon-derived attack-type OFFSET (val3) reaches the message sender, not the
// hardcoded AttackNormal (0). fight.c:1792-1806 derives w_type = val3 + TYPE_HIT
// and feeds it ONLY to the message path; SendWeaponMessage adds TYPE_HIT itself,
// so the offset (not a TYPE_* constant) must be passed.
func TestPerformOneHit_WeaponOffsetReachesMessageSender(t *testing.T) {
	cases := []struct {
		name         string
		weaponOffset int // the val3 offset GetWeaponInfo returns
		wantOffset   int // the attackType MessageFunc must receive
		wantVerb     string
	}{
		{"piercing weapon (dagger)", 11, 11, "pierce"},
		{"slashing weapon", 3, 3, "slash"},
		{"barehand", 0, 0, "hit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := GetCallbacks()
			defer SetCallbacks(orig)

			cb := defaultCombatCallbacks()
			cb.GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
				if chName == "Hero" {
					return tc.weaponOffset, 0, 0, false
				}
				return 0, 0, 0, false
			}
			SetCallbacks(cb)

			engine := NewCombatEngine()
			defer engine.Stop()

			var gotOffset int
			var gotMessages []string
			// Capture the attackType handed to the message sender AND the text
			// rendered for the attacker, by routing through the production
			// SendWeaponMessage (which calls DamMessage → cbSendToChar).
			cb.SendToChar = func(name, msg string) {
				if name == "Hero" {
					gotMessages = append(gotMessages, msg)
				}
			}
			engine.MessageFunc = func(attacker, defender Combatant, dam, attackType int) bool {
				gotOffset = attackType
				SendWeaponMessage(dam, attacker, defender, attackType)
				return true
			}
			engine.BroadcastFunc = func(int, string, string) {}

			attacker := &mockCombatant{
				name: "Hero", npc: false, room: 100, level: 10,
				hp: 100, maxHP: 100, ac: 0, thac0: 1, hitroll: 0, damroll: 0,
				str: 16, damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
				class: ClassWarrior,
			}
			defender := &mockCombatant{
				name: "Orc", npc: true, room: 100, level: 1,
				hp: 1000, maxHP: 1000, ac: 100, thac0: 20,
				damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
			}

			// Force a hit: natural 20 always hits (CalculateHitChance line 393).
			WithRoller(NewScriptedRoller([]int{20}), func() {
				engine.performOneHit(&CombatPair{Attacker: attacker, Defender: defender})
			})

			if gotOffset != tc.wantOffset {
				t.Errorf("message attackType = %d, want %d (weapon offset %d); "+
					"performOneHit must derive the message verb from the wielded weapon, not hardcode AttackNormal(0)",
					gotOffset, tc.wantOffset, tc.weaponOffset)
			}
			// The rendered message must carry the offset's verb (pierce/slash/hit).
			joined := strings.Join(gotMessages, " ")
			if !strings.Contains(joined, tc.wantVerb) {
				t.Errorf("rendered message %q should contain verb %q (offset %d)", joined, tc.wantVerb, tc.wantOffset)
			}
		})
	}
}

// TestPerformOneHit_DamageUnchangedByWeaponOffset — R3/damage-correctness: even
// though the message attack-type now reflects the weapon, the DAMAGE must be
// identical regardless of offset. CalculateDamage must keep receiving
// AttackNormal so AC reduction (getMinusDam) applies identically. A piercing
// weapon and a barehand attack against the same defender must deal equal damage
// for the same dice/str/damroll.
func TestPerformOneHit_DamageUnchangedByWeaponOffset(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	runOnce := func(t *testing.T, weaponOffset int) int {
		cb := defaultCombatCallbacks()
		cb.GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
			if chName == "Hero" {
				return weaponOffset, 0, 0, false
			}
			return 0, 0, 0, false
		}
		SetCallbacks(cb)

		engine := NewCombatEngine()
		defer engine.Stop()
		engine.MessageFunc = func(Combatant, Combatant, int, int) bool { return true }
		engine.BroadcastFunc = func(int, string, string) {}

		attacker := &mockCombatant{
			name: "Hero", npc: false, room: 100, level: 10,
			hp: 100, maxHP: 100, ac: 0, thac0: 1, hitroll: 0, damroll: 0,
			str: 16, damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
		}
		defender := &mockCombatant{
			name: "Dummy", npc: true, room: 100, level: 1,
			hp: 1000, maxHP: 1000, ac: 30, thac0: 20,
			damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
		}
		// Force a hit (natural 20) with zero weapon dice → deterministic damage.
		WithRoller(NewScriptedRoller([]int{20}), func() {
			engine.performOneHit(&CombatPair{Attacker: attacker, Defender: defender})
		})
		return 1000 - defender.hp
	}

	damPierce := runOnce(t, 11)
	damBarehand := runOnce(t, 0)
	if damPierce != damBarehand {
		t.Errorf("damage changed with weapon offset: pierce=%d barehand=%d; "+
			"CalculateDamage must keep AttackNormal so AC reduction applies identically",
			damPierce, damBarehand)
	}
	if damPierce <= 0 {
		t.Errorf("expected nonzero damage (str 16 vs AC 30), got %d", damPierce)
	}
}

// TestPerformOneHit_MissBranchUsesFreshWeaponOffset — trap #3: the miss branch
// must read a FRESH weapon offset, not the stale pair.LastAttackType. A miss on
// the very first round (LastAttackType still zero) with a piercing weapon must
// still pass offset 11 to the miss message sender.
func TestPerformOneHit_MissBranchUsesFreshWeaponOffset(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
		if chName == "Hero" {
			return 11, 1, 6, false // piercing
		}
		return 0, 0, 0, false
	}
	SetCallbacks(cb)

	engine := NewCombatEngine()
	defer engine.Stop()

	var missOffset int
	var missMessages []string
	cb.SendToChar = func(name, msg string) {
		if name == "Hero" {
			missMessages = append(missMessages, msg)
		}
	}
	engine.MessageFunc = func(attacker, defender Combatant, dam, attackType int) bool {
		missOffset = attackType
		SendWeaponMessage(dam, attacker, defender, attackType)
		return true
	}
	engine.BroadcastFunc = func(int, string, string) {}

	attacker := &mockCombatant{
		name: "Hero", npc: false, room: 100, level: 1,
		hp: 100, maxHP: 100, ac: 0, thac0: 20, hitroll: 0, damroll: 0,
		str: 10, damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding,
	}
	defender := &mockCombatant{
		name: "Ghost", npc: true, room: 100, level: 20,
		hp: 100, maxHP: 100, ac: 0, thac0: 1,
		damageRoll: DiceRoll{Num: 0, Sides: 0}, position: PosStanding, // awake
	}

	// Force a miss: natural 1 with an awake defender always misses (line 397).
	WithRoller(NewScriptedRoller([]int{1}), func() {
		engine.performOneHit(&CombatPair{Attacker: attacker, Defender: defender})
	})

	if missOffset != 11 {
		t.Errorf("miss message attackType = %d, want 11 (pierce); the miss branch "+
			"must compute the weapon offset FRESH, not read the stale pair.LastAttackType", missOffset)
	}
	joined := strings.Join(missMessages, " ")
	if !strings.Contains(joined, "pierce") {
		t.Errorf("miss message %q should contain 'pierce' (offset 11 from weapon), not 'hit'", joined)
	}
}
