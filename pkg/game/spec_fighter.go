package game

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// mobSkillDamage is the native damage() path used by fighter's subcommand
// calls. It deliberately emits the skill message after updating HP/position,
// matching fight.c:1308-1718. Unlike the player command path, there is no
// command-layer result to defer the message or combat enrollment.
func (w *World) mobSkillDamage(ch *MobInstance, vict combat.Combatant, dam, skillNum int) bool {
	if ch == nil || vict == nil {
		return false
	}

	// Native damage() performs jail/charm/switcheroo redirects before applying
	// the supplied amount. CombatEngine already owns that shared C seam for
	// one_hit(); use it here too for direct mob-special damage (R3/R5b/R5c).
	if redirector, ok := w.combatEngine.(interface {
		ApplyMobDamageRedirects(combat.Combatant, combat.Combatant) bool
	}); ok && redirector.ApplyMobDamageRedirects(ch, vict) {
		return false
	}

	// damage() enrolls both sides before applying its modifier block. The
	// victim's position is only raised from a downed state at this point when
	// the C position gate allows it; fighter's live target is already fighting.
	if ch.GetPosition() > combat.PosStunned && vict.GetPosition() > combat.PosStunned && vict.GetFighting() == "" {
		vict.SetFighting(ch.GetName())
	}
	dam = combat.ApplyDamageModifiers(ch, vict, dam)
	if dam > 0 {
		vict.TakeDamage(dam)
	}
	newPos := combat.GetPositionFromHP(vict.GetHP(), vict.GetPosition())
	vict.SetPosition(newPos)
	// C's damage() emits skill_message before the wounded/death follow-up
	// (fight.c:1534-1545, 1560-1613). Keep that ordering even for a
	// protection-reduced zero-damage result.
	combat.EmitSkillMessage(dam, ch.GetName(), vict.GetName(), skillNum, ch.GetRoom())
	w.emitMobSkillSurvival(ch, vict, dam, newPos)
	if newPos == combat.PosDead {
		w.HandleDeath(vict, ch, skillNum)
	}
	return dam > 0
}

// mobBackstabDamage completes the C damage() entry that follows hit() from a
// mobile special. Unlike the ordinary fighter helpers, backstabber starts from
// a non-fighting mob, so damage() must put the attacker on combat_list before
// applying the skill damage (fight.c:1400-1408). The victim's position is left
// for mobSkillDamage's update_pos-equivalent after the damage amount has been
// computed; C's set_fighting(victim) is likewise immediately followed by
// update_pos(victim) (fight.c:1443-1445, 1484-1490).
func (w *World) mobBackstabDamage(me *MobInstance, vict combat.Combatant, dam int) bool {
	if me == nil || vict == nil {
		return false
	}

	// damage() refuses non-outlaws in peaceful rooms unless the victim is
	// already fighting the attacker. Backstabber's target gate selects a
	// non-fighting player, so this is the only reachable peaceful-room branch.
	if player, ok := vict.(*Player); ok &&
		w.roomHasFlag(me.GetRoom(), "peaceful") &&
		player.GetFlags()&(1<<uint(PlrOutlaw)) == 0 &&
		vict.GetFighting() != me.GetName() {
		return false
	}

	if me.GetFighting() == "" && me.GetPosition() > combat.PosStunned {
		if w.combatEngine != nil {
			if err := w.combatEngine.StartCombat(me, vict); err != nil {
				// C has no fallible combat-engine boundary. Preserve the native
				// damage path after recording an unexpected Go-side failure.
				slog.Warn("StartCombat failed in backstabber", "mob", me.GetName(), "victim", vict.GetName(), "error", err)
			}
		}
		// Focused spec vehicles may intentionally omit a full combat engine;
		// retain C's observable fighting state in that test seam as well.
		if me.GetFighting() == "" {
			me.SetFighting(vict.GetName())
			me.SetPosition(combat.PosFighting)
		}
	}
	if vict.GetFighting() == "" && vict.GetPosition() > combat.PosStunned {
		vict.SetFighting(me.GetName())
	}

	return w.mobSkillDamage(me, vict, dam, SkillBackstabNum)
}

// mobBackstab ports hit(ch, vict, SKILL_BACKSTAB) as called by
// do_backstab(..., subcmd=1). The caller has already passed the command,
// target, weapon, mount, and target-fighting gates, so this helper starts with
// the native percent/probability draws and preserves their order
// (act.offensive.c:220-229; fight.c:1825-1881).
func (w *World) mobBackstab(me *MobInstance, vict combat.Combatant) {
	percent := dprng.Number(1, 101)
	prob := dprng.Number(50, 100)
	if vict.GetPosition() > combat.PosSleeping && percent > prob {
		w.mobBackstabDamage(me, vict, 0)
		return
	}

	if !combat.CalculateHitChance(me, vict, combat.HitModifiers{}) {
		w.mobBackstabDamage(me, vict, 0)
		return
	}

	// C's NPC hit() uses the mob damage dice, not the wielded weapon dice.
	// Unlike a normal weapon hit, SKILL_BACKSTAB bypasses get_minusdam().
	damRoll := me.GetDamageRoll()
	dam := combat.StrAppToDam(me) + me.GetDamroll()
	dam += dprng.Dice(damRoll.Num, damRoll.Sides) + damRoll.Plus
	if vict.GetPosition() < combat.PosFighting {
		dam = int(float64(dam) * (1.0 + float64(combat.PosFighting-vict.GetPosition())/3.0))
	}
	if dam < 1 {
		dam = 1
	}
	dam *= int(combat.BackstabMult(me.GetLevel()))
	w.mobBackstabDamage(me, vict, dam)
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
		if dam > vict.GetMaxHP()/4 {
			Act(w, false, vict, nil, nil, nil, "That really did HURT!", "", ToChar)
			if dprng.Number(0, 2) == 0 {
				Act(w, false, ch, vict, nil, nil, "$N screams in pain!", "", ToNotVict)
			}
		}
		if vict.GetHP() < vict.GetMaxHP()/4 {
			vict.SendMessage("You wish that your wounds would stop BLEEDING so much!\r\n")
		}
	}

	if pos < combat.PosSleeping && vict.GetFighting() != "" {
		vict.StopFighting()
	}
}

// mobHeadbutt ports do_headbutt(ch, ..., subcmd=1). All player-facing
// command text is suppressed because the native caller is an NPC with no
// descriptor, while the damage() skill-message branch remains visible.
func mobHeadbutt(w *World, me *MobInstance, vict combat.Combatant) {
	if w.roomHasFlag(me.GetRoom(), "peaceful") {
		return
	}
	percent := dprng.Number(1, 121)
	if vict.GetPosition() <= combat.PosSleeping || me.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	if mob, ok := vict.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
		percent = 0
	}
	if me.GetLevel()/2 > me.GetHP() {
		return
	}
	if vict.GetPosition() <= combat.PosDead {
		return
	}
	if percent > dprng.Number(50, 100) {
		w.mobSkillDamage(me, vict, 0, SkillHeadbuttNum)
		return
	}

	recoil := me.GetLevel() / 4
	if _, wearingHelm := me.Equipment[int(SlotHead)]; wearingHelm {
		recoil = me.GetLevel() / 3
	}
	me.TakeDamage(recoil)
	if w.mobSkillDamage(me, vict, me.GetLevel(), SkillHeadbuttNum) && vict.GetPosition() > combat.PosStunned {
		vict.SetPosition(combat.PosSitting)
	}
}

// mobBash ports do_bash(ch, ..., subcmd=1). C uses a fixed probability of 131
// for the fighter special and spends movement before consuming the hit roll.
func mobBash(w *World, me *MobInstance, vict combat.Combatant) {
	if w.roomHasFlag(me.GetRoom(), "peaceful") || vict.GetPosition() < combat.PosFighting {
		return
	}
	if !me.SpendMove(10) {
		return
	}
	percent := ((5 - (vict.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	if mob, ok := vict.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) && me.GetLevel() < LVL_IMMORT {
		percent = 101
	}
	if vict.GetPosition() <= combat.PosSleeping || me.GetLevel() >= LVL_IMMORT {
		percent = 0
	}
	if percent > 131 {
		w.mobSkillDamage(me, vict, 0, SkillBashNum)
		me.SetPosition(combat.PosSitting)
		return
	}
	if w.mobSkillDamage(me, vict, (me.GetLevel()/2)+1, SkillBashNum) {
		vict.SetPosition(combat.PosSitting)
		if player, ok := vict.(*Player); ok {
			player.SetWaitState(2)
		}
	}
}

// mobParry ports do_parry(ch, ..., subcmd=1). The procedure's successful
// act() calls use TO_ROOM and TO_VICT separately, so the victim receives both
// C messages when it is the only player in the room.
func mobParry(w *World, me *MobInstance, vict combat.Combatant) {
	if vict.GetFighting() != me.GetName() {
		return
	}
	if _, wielded := me.Equipment[int(SlotWield)]; !wielded {
		return
	}
	percent := dprng.Number(1, 101)
	if percent > dprng.Number(50, 100) {
		return
	}
	Act(w, true, me, vict, nil, nil,
		"$n displays a dazzling show of swordplay, fending off $N's every blow!", "", ToRoom)
	Act(w, true, me, vict, nil, nil,
		"$n displays a dazzling show of swordplay, fending off your every blow!", "", ToVict)
	if marker, ok := w.combatEngine.(interface{ MarkParried(name, action string) }); ok {
		marker.MarkParried(vict.GetName(), "parry")
	}
}

// mobBerserk ports do_berserk(ch, NULL, ..., subcmd=1). The failed attempt
// still installs the three zero-modifier AFF_BERSERK affects, as C does.
func mobBerserk(me *MobInstance) {
	percent := dprng.Number(1, 101)
	if me.HasAffect(affBerserk) {
		return
	}
	if me.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	failed := percent > dprng.Number(50, 100)
	hitDamMod, acMod := 2, 25
	if failed {
		hitDamMod, acMod = 0, 0
	}
	me.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyHitroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	me.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyDamroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	me.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyAC, 1, acMod, engine.AFFBerserk, "berserk"))
}
