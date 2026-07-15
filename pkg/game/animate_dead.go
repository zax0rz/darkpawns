package game

// animate_dead.go implements the game-layer helpers for SPELL_ANIMATE_DEAD.
// These methods live on *World so pkg/spells can call them through a local
// interface without importing pkg/game (which would create an import cycle).
//
// C source: src/magic.c mag_summons, case SPELL_ANIMATE_DEAD.

const (
	animateDeadGiddyMsg  = "You are too giddy to have any followers!\r\n"
	animateDeadCapMsg    = "You can't have any more followers!\r\n"
	animateDeadPfail     = 8
	animateDeadPfailDice = 102 // number(0, 101)
)

// charWithCharmAndCha describes the concrete types that can animate a corpse.
type charWithCharmAndCha interface {
	GetName() string
	GetCha() int
	IsAffected(bit int) bool
}

// CanRaiseUndeadI reports whether ch may animate a corpse right now, mirroring
// mag_summons' pre-checks. It returns (false, playerMessage) when blocked.
//
// Order matches C:
//  1. charmed (AFF_CHARM) -> "You are too giddy to have any followers!"
//  2. follower count >= GET_CHA(ch)/2 -> "You can't have any more followers!"
func (w *World) CanRaiseUndeadI(ch interface{}) (bool, string) {
	c, ok := ch.(charWithCharmAndCha)
	if !ok {
		// Unknown caster type: don't silently block the spell.
		return true, ""
	}

	if c.IsAffected(affCharm) {
		return false, animateDeadGiddyMsg
	}

	name := c.GetName()
	if name == "" {
		return true, ""
	}

	count := len(w.GetFollowers(name))
	for _, m := range w.activeMobs {
		if m.GetFollowing() == name {
			count++
		}
	}

	if count >= c.GetCha()/2 {
		return false, animateDeadCapMsg
	}

	return true, ""
}

// CharmAndFollowI sets AFF_CHARM on the raised mob and makes it a quiet
// follower of leader — the SET_BIT_AR + add_follower_quiet pair from C.
func (w *World) CharmAndFollowI(mob, leader interface{}) {
	m, ok := mob.(*MobInstance)
	if !ok {
		return
	}
	m.SetAffected(affCharm)

	if l, ok := leader.(*Player); ok {
		AddFollowerQuietMob(m, l)
		return
	}

	// DP-1008: mob leaders are not yet supported by the follower system.
	// The mob is charmed above so the spell effect itself is correct.
}
