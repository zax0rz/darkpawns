package game

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// mobSkillDamage is the native damage() path used by fighter's subcommand
// calls. It deliberately emits the skill message after updating HP/position,
// matching fight.c:1308-1718. Unlike the player command path, there is no
// command-layer result to defer the message or combat enrollment.
func (w *World) mobSkillDamage(ch *MobInstance, vict combat.Combatant, dam, skillNum int) bool {
	if ch == nil || vict == nil || w.DamageRefused(ch, vict) {
		return false
	}
	return w.mobSkillDamageAfterGate(ch, vict, dam, skillNum)
}

// mobSkillDamageAfterGate is mobSkillDamage for a caller that has already
// asked damage()'s protection gate.
func (w *World) mobSkillDamageAfterGate(ch *MobInstance, vict combat.Combatant, dam, skillNum int) bool {
	return w.mobSkillDamageWithDamageGate(ch, vict, dam, skillNum, false)
}

// mobileCommandSkillDamage is the command body's direct damage() entry. C
// enrolls the command body and victim here, after damage protection and
// redirects, unlike a fighter special whose source is already in combat
// (src/fight.c:1318-1453).
func (w *World) mobileCommandSkillDamage(ch *MobInstance, vict combat.Combatant, dam, skillNum int) bool {
	if ch == nil || vict == nil || w == nil || w.DamageRefused(ch, vict) {
		return false
	}
	return w.mobSkillDamageWithDamageGate(ch, vict, dam, skillNum, true)
}

func (w *World) mobSkillDamageWithDamageGate(ch *MobInstance, vict combat.Combatant, dam, skillNum int, enrollAttacker bool) bool {
	if enrollAttacker {
		enterAttacker := func() {
			if ch != vict && ch.GetPosition() > combat.PosStunned && ch.GetFightingBody() == nil {
				ch.SetFightingBody(vict)
				ch.SetPosition(combat.PosFighting)
			}
		}
		if redirector, ok := w.combatEngine.(interface {
			ApplyMobDamageRedirectsAfterAttackerEnrollment(combat.Combatant, combat.Combatant, int, func()) bool
		}); ok {
			if redirector.ApplyMobDamageRedirectsAfterAttackerEnrollment(ch, vict, dam, enterAttacker) {
				return false
			}
		} else {
			// Engines without the production seam keep the legacy redirect path;
			// production runs the C jail-before-enrollment order above.
			if w.ApplyMobDamageRedirects(ch, vict, dam) {
				return false
			}
			enterAttacker()
		}
	} else if w.ApplyMobDamageRedirects(ch, vict, dam) {
		return false
	}
	if enrollAttacker {
		if vict != ch && vict.GetPosition() > combat.PosStunned && vict.GetFightingBody() == nil {
			// C's victim-side set_fighting is after those redirects
			// (fight.c:1443-1453), and is independent of the attacker's position.
			vict.SetFightingBody(ch)
			vict.SetPosition(combat.PosFighting)
		}
		if w.combatEngine != nil {
			if err := w.StartCombatAfterDamage(ch, vict); err != nil {
				slog.Warn("StartCombatAfterDamage failed in mobile command damage", "mob", ch.GetName(), "victim", vict.GetName(), "error", err)
			}
		}
		if ch.HasMobFlag(MobFlagHunter) && !vict.IsNPC() && vict.GetLevel() < LVL_IMMORT {
			w.SetHuntingTarget(ch, asActor(vict))
		}
	}

	// damage() enrolls both sides before applying its modifier block. The
	// victim's position is only raised from a downed state at this point when
	// the C position gate allows it; fighter's live target is already fighting.
	if ch.GetPosition() > combat.PosStunned && vict.GetPosition() > combat.PosStunned && vict.GetFightingBody() == nil {
		// C set_fighting(victim, ch) also raises the victim to POS_FIGHTING
		// (fight.c:1443-1445); see mobBackstabDamage.
		vict.SetFightingBody(ch)
		vict.SetPosition(combat.PosFighting)
	}
	// fight.c:1457-1464: break the victim's following when the attacker is
	// the master, then fade a hidden attacker — before the modifier block
	// (:1465 onward). This seam omitted both (DP-1489).
	combat.BreakFollowerOnDamage(ch, vict)
	combat.FadeHiddenAttacker(ch)
	dam = combat.ApplyDamageModifiers(ch, vict, dam)
	if dam > 0 {
		vict.TakeDamage(dam)
	}
	if w.DamageBeforeMessage(ch, vict, dam) {
		return false
	}
	newPos := vict.GetPosition()
	// C's damage() emits skill_message before the wounded/death follow-up
	// (fight.c:1534-1545, 1560-1613). Keep that ordering even for a
	// protection-reduced zero-damage result.
	combat.EmitSkillMessage(dam, ch, vict, skillNum, ch.GetRoom())
	combat.ApplyMountedHitTail(ch, vict, dam)
	w.emitMobSkillSurvival(ch, vict, dam, newPos)
	if newPos == combat.PosDead {
		w.HandleDeath(vict, ch, skillNum)
	}
	return dam > 0
}

// emitMobSkillSurvival contains the player-visible post-message position
// bytes from damage(). The high-damage pain/bleeding paths are kept here too
// so a successful native fighter action does not silently drop those C arms.
// ch is damage()'s ch (the attacker); self-damage callers pass the victim
// itself, matching C's ch == victim calls (comm.c room_activity).
func (w *World) emitMobSkillSurvival(ch combat.Combatant, vict combat.Combatant, dam, pos int) {
	switch pos {
	case combat.PosMortally:
		Act(w, true, vict, nil, nil, nil,
			"$n is mortally wounded, and will die soon, if not aided.", "", ToRoom)
		vict.SendMessage("You are mortally wounded, and will die soon, if not aided.\r\n")
	case combat.PosIncap:
		Act(w, true, vict, nil, nil, nil,
			"$n is incapacitated and will slowly die, if not aided.", "", ToRoom)
		vict.SendMessage("You are incapacitated an will slowly die, if not aided.\r\n")
	case combat.PosStunned:
		Act(w, true, vict, nil, nil, nil,
			"$n is stunned, but will probably regain consciousness again.", "", ToRoom)
		vict.SendMessage("You're stunned, but will probably regain consciousness again.\r\n")
	case combat.PosDead:
		Act(w, false, vict, nil, nil, nil, "$n is dead!  R.I.P.", "", ToRoom)
		vict.SendMessage("You are dead!  Sorry...\r\n")
	default:
		// damage()'s default arm for a victim that survived above POS_STUNNED:
		// the pain line and its number(0,2) scream draw, the wounded-band line,
		// the MOB_WIMPY flee and the player wimpy block
		// (src/fight.c:1587-1614). It lives in one place because C runs it for
		// every attack type; this seam used to print the wounded-band line but
		// neither wimpy block, so a mob-skill victim never wimped out and the
		// flee's draws were missing from the shared stream (R1/R3).
		combat.ApplySurvivingVictimDefaultArm(ch, vict, dam, pos)
	}

	if pos < combat.PosSleeping && vict.GetFightingBody() != nil {
		vict.StopFighting()
	}
}
