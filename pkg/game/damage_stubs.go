//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Damage helper stubs for act_offensive.go
// These match the CircleMUD fight.c patterns: damage() and hit_skill()
// ---------------------------------------------------------------------------

// doDamage applies damage to a target.
// DoSpellDamage applies damage to a player or mob, handling death.
// Used by damage spells (hellfire, meteor_swarm, etc.) that need to hit any character type.
func (w *World) DoSpellDamage(attacker, victim interface{}, dam int, skill string) bool {
	if dam <= 0 {
		return false
	}

	attackerName := getAttackerName(attacker)

	switch v := victim.(type) {
	case *Player:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		if v.GetHP() <= 0 {
			w.rawKill(v, 303)
		}
		return true
	case *MobInstance:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		if v.GetHP() <= 0 {
			w.handleMobDeath(v, nil, 303)
		}
		return true
	default:
		return false
	}
}

func (w *World) doDamage(ch, vict interface{}, dam int, skill string) bool {
	victim, ok := vict.(*Player)
	if !ok {
		return false
	}

	if dam <= 0 {
		victim.SendMessage(fmt.Sprintf("%s hits you, but it doesn't hurt!\r\n", getAttackerName(ch)))
		return false
	}

	victim.TakeDamage(dam)
	victim.SetFighting(getAttackerName(ch))

	if victim.GetHP() <= 0 {
		w.rawKill(victim, 303)
	}
	return true
}

// hitSkill performs a skill-based hit (fight.c: hit_skill())
func (w *World) hitSkill(ch, vict interface{}, skill string) bool {
	victim, ok := vict.(*Player)
	if !ok {
		return false
	}
	dam := randRange(1, 8) + 2
	w.doDamage(ch, vict, dam, skill)
	_ = victim
	return true
}

// getAttackerName returns the name of the attacker for messages.
func getAttackerName(ch interface{}) string {
	if p, ok := ch.(*Player); ok {
		return p.GetName()
	}
	if m, ok := ch.(*MobInstance); ok {
		return m.GetName()
	}
	return "someone"
}

// randRange returns a random integer in [min, max].

// executeCommand executes a command string  on behalf of a player
func (w *World) executeCommand(ch *Player, command string) bool {
	_ = ch
	_ = command
	return true
}

// doForced is a stub for perform_act / do_forced — received a forced command string
func (w *World) doForced(ch *Player, command string) bool {
	return w.executeCommand(ch, command)
}

// doMurder handles the murder command
func (w *World) doMurder(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// doBackstab handles the backstab command

// diceRoll rolls N dice of D sides each
func diceRoll(n, d int) int {
	total := 0
	for i := 0; i < n; i++ {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		rand.Intn(d)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		total += rand.Intn(d) + 1
	}
	return total
}

// updatePosFromHP delegates to the canonical free function in limits_exp.go.
// Kept as a World method for combat callers that have World receiver.
func (w *World) updatePosFromHP(victim *Player) {
	updatePosFromHP(victim, victim.GetHP())
}

