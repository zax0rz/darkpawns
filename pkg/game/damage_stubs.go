package game

// CRIT-009: Dual hit-resolution paths in Dark Pawns combat
//
// Path 1 — processCombatPair (pkg/combat/engine.go)
//   The main combat tick. Runs every round for each fighting pair.
//   Flow: hit roll → parry check → dodge check → damage → death
//   Used for: standard melee attacks (auto-attack each combat round)
//   parry/dodge: CHECKED (via CheckParry/CheckDodge in formulas.go)
//
// Path 2 — doDamage / DoSpellDamage (this file, pkg/game/damage_stubs.go)
//   Called directly by skills, spells, and special attacks.
//   Flow: hit roll (skill-specific) → damage → death
//   Used for: bash, kick, backstab, spell damage, etc.
//   parry/dodge: NOT CHECKED (intentional)
//
// Why two paths? This matches CircleMUD design:
//   - Skill attacks (bash, kick, trip) have their own hit/miss logic and
//     intentionally bypass parry/dodge — that's the tradeoff for using
//     a limited-use skill vs auto-attack.
//   - Backstab from behind explicitly cannot be parried or dodged.
//   - Spell damage (fireball, etc.) uses saving throws instead of parry/dodge.
//
// Future consideration: Some melee skills (e.g., second attack, third attack)
//   MAY want parry/dodge checks. Currently they go through processCombatPair
//   and are handled correctly. Only manually-invoked skills use path 2.

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// ---------------------------------------------------------------------------
// Damage helper stubs for act_offensive.go
// These match the CircleMUD fight.c patterns: damage() and hit_skill()
// ---------------------------------------------------------------------------

// skillToAttackType maps a skill/spell name to the C numeric attack type that C
// passes to damage(). That single value selects the fight-message set, the
// corpse wording (fight.c make_corpse) and the SPELL_DISINTEGRATE dust check, so
// every damaging skill must carry its own C SKILL_* constant: make_corpse
// groups SKILL_* and TYPE_* values into different descriptions, and a
// substituted TYPE_* value silently changes the corpse bytes (bash/kick/headbutt
// are the "bruised, battered" group, while trip/bearhug/strike have no group at
// all and take the plain default).
//
// Unknown names take TYPE_UNDEFINED, C's own "no specific attack type" value:
// make_corpse's TYPE_UNDEFINED case and its default case print the same line.
func skillToAttackType(skill string) int {
	switch skill {
	case SkillBackstab:
		return SkillBackstabNum // 131 SKILL_BACKSTAB
	case SkillBash:
		return SkillBashNum // 132 SKILL_BASH
	case SkillKick:
		return SkillKickNum // 134 SKILL_KICK
	case SkillPunch:
		return SkillPunchNum // 136 SKILL_PUNCH
	case SkillHeadbutt:
		return SkillHeadbuttNum // 141 SKILL_HEADBUTT
	case SkillBearhug:
		return SkillBearhugNum // 142 SKILL_BEARHUG
	case SkillCutthroat:
		return SkillCutthroatNum // 143 SKILL_CUTTHROAT
	case SkillTrip:
		return SkillTripNum // 144 SKILL_TRIP
	case SkillSmackheads:
		return SkillSmackheadsNum // 145 SKILL_SMACKHEADS
	case SkillSlug:
		return SkillSlugNum // 146 SKILL_SLUG
	case SkillCharge:
		return SkillChargeNum // 147 SKILL_CHARGE
	case SkillBite:
		return SkillBiteNum // 150 SKILL_BITE
	case SkillStrike:
		return SkillStrikeNum // 155 SKILL_STRIKE
	case SkillSerpentKick:
		return SkillSerpentKickNum // 156 SKILL_SERPENT_KICK
	case SkillCircle:
		return SkillCircleNum // 173 SKILL_CIRCLE
	case SkillGroinrip:
		return SkillGroinripNum // 174 SKILL_GROINRIP
	case SkillDisembowel:
		return SkillDisembowelNum // 184 SKILL_DISEMBOWEL
	case SkillDragonKick:
		return SkillDragonKickNum // 188 SKILL_DRAGON_KICK
	case SkillTigerPunch:
		return SkillTigerPunchNum // 189 SKILL_TIGER_PUNCH
	case SkillNeckbreak:
		return SkillNeckbreakNum // 190 SKILL_NECKBREAK
	case SkillAmbush:
		return SkillAmbushNum // 191 SKILL_AMBUSH
	case "meteor swarm":
		return spells.SpellMeteorSwarm
	case "hellfire":
		return spells.SpellHellfire
	default:
		return TypeUndefined
	}
}

// DamageAttackType is C's numeric attack type for this result's damage() call.
//
// C passes one value — the skill's own constant (SKILL_BASH for do_bash,
// spellnum for a damage spell) — and derives everything from it, so the number
// the port hands to the damage pipeline is the same one skill_message keys on.
// SkillMsgType already carries it for every result that emits a skill_message
// set; DamageSkill names the skill for results whose dispatch the command tail
// switches on. Preferring SkillMsgType keeps the two from drifting, which is how
// bash, kick, headbutt, trip, bearhug and strike reached the wrong corpse
// wording while their messages stayed right.
func (r SkillResult) DamageAttackType() int {
	if r.SkillMsgType != 0 {
		return r.SkillMsgType
	}
	return skillToAttackType(r.DamageSkill)
}

// combatantFromInterface converts an interface{} attacker to a combat.Combatant.
func combatantFromInterface(attacker interface{}) combat.Combatant {
	if attacker == nil {
		return nil
	}
	if c, ok := attacker.(combat.Combatant); ok {
		return c
	}
	return nil
}

// DoSpellDamage applies damage to a player or mob, handling C's fight-message
// dispatch and death order. Used by area damage spells such as Hellfire and
// Meteor Swarm. It asks damage()'s protection gate first (DP-1327).
func (w *World) DoSpellDamage(attacker, victim interface{}, dam int, skill string) bool {
	killer := combatantFromInterface(attacker)
	victimC := combatantFromInterface(victim)
	if killer != nil && victimC != nil && w.DamageRefused(killer, victimC) {
		return false
	}
	if killer == nil || victimC == nil {
		return w.ApplySkillDamage(attacker, victim, dam, skillToAttackType(skill))
	}
	attackType := skillToAttackType(skill)
	return combat.TakeDamageAfterGate(killer, victimC, dam, attackType, func() {
		w.HandleDeath(victimC, killer, attackType)
	})
}

// ApplySkillDamage is DoSpellDamage for a caller that has already asked
// DamageRefused where C calls damage(): the skill command tail, which emits
// skill_message itself between the gate and the damage. attackType is C's own
// damage() argument (SkillResult.DamageAttackType), so it carries the corpse
// wording as well as the message set.
func (w *World) ApplySkillDamage(attacker, victim interface{}, dam, attackType int) bool {
	return w.ApplySkillDamageWithMessage(attacker, victim, dam, attackType, nil)
}

// ApplySkillDamageWithMessage places a command-owned skill message after
// damage()'s HP/position/rescue block (src/fight.c:1484-1535).
func (w *World) ApplySkillDamageWithMessage(attacker, victim interface{}, dam, attackType int, message func(int)) bool {
	killer := combatantFromInterface(attacker)
	combat.EnterDamageFighting(killer, combatantFromInterface(victim))

	// Route the computed damage through the shared damage() modifier block
	// (sanctuary, protect evil/good, race-hate, immortal immunity, 3000 cap)
	// before applying — the same funnel melee and skills use (DP-1025). A hit
	// reduced to 0 (immortal victim / sanctuary on a tiny hit) deals nothing.
	if victimC := combatantFromInterface(victim); victimC != nil {
		dam = combat.ApplyDamageModifiers(killer, victimC, dam)
	}
	if v := combatantFromInterface(victim); v != nil {
		v.TakeDamage(dam)
		if w.DamageBeforeMessage(killer, v, dam) {
			return false
		}
	}
	if message != nil {
		message(dam)
	}
	if dam <= 0 {
		return false
	}

	switch v := victim.(type) {
	case *Player:
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		newPos := combat.UpdatePositionAfterDamage(v, w.woundBroadcast, w.StopFightingBody)
		// C's default arm: the pain line, the number(0,2) scream draw, the
		// wounded-band line and the wimpy block (src/fight.c:1587-1614). The
		// draw is unconditional whenever the blow exceeds a quarter of the
		// victim's maximum hit points, whatever the attack type; burning it
		// only for charge left every later draw in the round one early.
		combat.ApplySurvivingVictimDefaultArm(killer, v, dam, newPos)
		if newPos == combat.PosDead {
			combat.EmitDeathPositionMessage(v, w.woundBroadcast)

			w.HandleDeath(v, killer, attackType)
		}
		return true
	case *MobInstance:
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		newPos := combat.UpdatePositionAfterDamage(v, w.woundBroadcast, w.StopFightingBody)
		combat.ApplySurvivingVictimDefaultArm(killer, v, dam, newPos)
		if newPos == combat.PosDead {
			combat.EmitDeathPositionMessage(v, w.woundBroadcast)

			w.HandleDeath(v, killer, attackType)
		}
		return true
	default:
		return false
	}
}

// The Do*Damage seams below are called only by the skill command tail
// (pkg/command sendSkillResult), which asks DamageRefused at the point C
// calls damage(). They skip the gate so a refusal prints once (DP-1327).

// DoDisembowelDamage preserves do_disembowel's damage() call path: damage
// updates the victim position, emits skill_message after that update, and
// runs death_cry/raw_kill only after the numbered message and death bytes.
func (w *World) DoDisembowelDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillDisembowelNum, func() {
		w.HandleDeath(victim, attacker, SkillDisembowelNum)
	})
}

// DoGroinripDamage preserves do_groinrip's damage() boundary: damage applies
// HP and position changes, emits numbered skill_message set 174, then returns
// to the command for its victim-room act() and skill improvement.
func (w *World) DoGroinripDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillGroinripNum, func() {
		w.HandleDeath(victim, attacker, SkillGroinripNum)
	})
}

// DoNeckbreakDamage preserves do_neckbreak's damage() boundary: apply damage,
// update position, emit numbered skill_message set 190, then return for the
// command's deferred improve_skill and WAIT_STATE (fight.c:1023-1092).
func (w *World) DoNeckbreakDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillNeckbreakNum, func() {
		w.HandleDeath(victim, attacker, SkillNeckbreakNum)
	})
}

// DoTigerPunchDamage preserves do_tiger_punch's damage() boundary
// (act.offensive.c:736-739): damage() applies HP and the position update,
// emits numbered skill_message set 189 (die/hit/miss chosen from the
// post-damage position), and only then runs death_cry/raw_kill. Selecting the
// message before the HP change would always read the hit variant.
func (w *World) DoTigerPunchDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillTigerPunchNum, func() {
		// C's damage() awards the mob XP before raw_kill emits the death
		// cry, and make_corpse announces nothing (fight.c:1638-1660, 257-283).
		// HandleDeath owns the Go extraction/XP seam (ambush.go:76-85).
		w.HandleDeath(victim, attacker, SkillTigerPunchNum)
	})
}

// DoStrikeDamage preserves do_strike's damage() boundary (src/new_cmds.c:1482-1486):
// damage() applies HP and the position update, emits numbered skill_message set
// 155, then returns so the command can run improve_skill and WAIT_STATE.
func (w *World) DoStrikeDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillStrikeNum, func() {
		// damage() awards XP before raw_kill's death cry; make_corpse is
		// silent (fight.c:1638-1660, 257-283).
		w.HandleDeath(victim, attacker, SkillStrikeNum)
	})
}

// DoDragonKickDamage preserves do_dragon_kick's damage() boundary
// (act.offensive.c:683-686): the numbered set-188 message is selected after the
// HP/position update, then death_cry/raw_kill run for a killing blow.
func (w *World) DoDragonKickDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillDragonKickNum, func() {
		// damage() awards XP before raw_kill's death cry; make_corpse is
		// silent (fight.c:1638-1660, 257-283).
		w.HandleDeath(victim, attacker, SkillDragonKickNum)
	})
}

// DoSmackheadsDamage preserves do_smackheads' damage() boundary: the two
// ordered damage calls must use the C skill attack type and complete death
// path, including the authored death bytes and XP/autogold before the death cry and corpse (src/new_cmds.c:1080-1081;
// src/fight.c:1644-1691, 573-578; R1/R5e).
func (w *World) DoSmackheadsDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillSmackheadsNum, func() {
		w.HandleDeath(victim, attacker, SkillSmackheadsNum)
	})
}

// MaybeSpawnPuke preserves do_groinrip's post-room 1-in-11 vnum-21 object
// branch, including its shared RNG draw and two-tick timer.
func (w *World) MaybeSpawnPuke(roomVNum int) {
	// #nosec G404 — game RNG, not cryptographic
	if dprng.Number(0, 10) != 0 {
		return
	}
	obj, err := w.SpawnObject(21, -1)
	if err != nil {
		slog.Error("groinrip puke prototype missing", "error", err)
		return
	}
	if err := w.MoveObjectToRoomFront(obj, roomVNum); err != nil {
		slog.Error("groinrip puke room placement failed", "room", roomVNum, "error", err)
		return
	}
	obj.SetTimer(2)
}

// DoCutthroatDamage completes do_cutthroat's damage() call with the shared C
// ordering: apply damage, update position, emit message set 143, emit the
// death-position bytes, then run the game-layer death bookkeeping. The
// callback is necessary because a lethal mob is removed by HandleDeath before
// a command wrapper can look it up for skill_message (fight.c:1534-1718).
func (w *World) DoCutthroatDamage(attacker, victim combat.Combatant, dam int) bool {
	if attacker == nil || victim == nil {
		return false
	}
	return combat.TakeDamageAfterGate(attacker, victim, dam, SkillCutthroatNum, func() {
		w.HandleDeath(victim, attacker, SkillCutthroatNum)
	})
}

// doDamage applies skill/offensive damage to a player or mob and handles death.
// Used by bash, kick, backstab, disembowel, ambush, neckbreak, tiger/dragon
// kick, spec_proc hits, etc. — every skill that goes through the "path 2"
// damage pipeline documented at the top of this file.
//
// DP-901: previously type-asserted vict.(*Player) and silently returned false
// for mobs, so every offensive skill against a mob no-op'd. Now mirrors
// DoSpellDamage: damages both types, sets fighting state, and routes death
// through HandleDeath so XP, kill counters, events, and autoloot fire.
func (w *World) doDamage(ch, vict interface{}, dam int, skill string) bool {
	attackerName := getAttackerName(ch)
	killer := combatantFromInterface(ch)
	attackType := skillToAttackType(skill)

	// Route skill/offensive damage through the shared damage() modifier block
	// (sanctuary, protect evil/good, race-hate, immortal immunity, 3000 cap)
	// before applying — the same funnel melee and spells use (DP-1025).
	if victimC := combatantFromInterface(vict); victimC != nil {
		dam = combat.ApplyDamageModifiers(killer, victimC, dam)
	}

	if dam <= 0 {
		// C: damage(ch, vict, 0, ...) prints the no-damage message and still
		// counts as a hit (used by skills that connect for zero damage).
		switch v := vict.(type) {
		case *Player:
			v.SendMessage(fmt.Sprintf("%s hits you, but it doesn't hurt!\r\n", attackerName))
		case *MobInstance:
			// Mobs don't get a player-style message; the call site reports.
		}
		return false
	}

	switch v := vict.(type) {
	case *Player:
		v.TakeDamage(dam)
		v.SetFightingBody(killer)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		if combat.UpdatePositionAfterDamage(v, w.woundBroadcast, w.StopFightingBody) == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	case *MobInstance:
		v.TakeDamage(dam)
		v.SetFightingBody(killer)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		if combat.UpdatePositionAfterDamage(v, w.woundBroadcast, w.StopFightingBody) == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	default:
		return false
	}
}

// woundBroadcast adapts World's room messaging to the signature that
// combat.UpdatePositionAfterDamage expects (roomVNum, message, exclude).
func (w *World) woundBroadcast(roomVNum int, message string, exclude []combat.Combatant) {
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if !sendOk(p, false) {
			continue
		}
		excluded := false
		for _, body := range exclude {
			if body == p {
				excluded = true
				break
			}
		}
		if !excluded {
			p.SendMessage(message + "\r\n")
		}
	}
}

// WoundBroadcast exposes woundBroadcast across the package boundary so the
// spells package can route spell-damage wound/position messages through the
// exact same room broadcaster melee and skills use (DP-1022). It is the
// exported wrapper the spells package asserts via its woundBroadcaster
// interface; keep its signature aligned with woundBroadcast.
func (w *World) WoundBroadcast(roomVNum int, message string, exclude []combat.Combatant) {
	w.woundBroadcast(roomVNum, message, exclude)
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

// executeCommand executes a command string on behalf of a player (e.g. from doOrder).
// Delegates to the session layer via CommandExecFunc if wired.
func (w *World) executeCommand(ch *Player, command string) bool {
	if w.CommandExecFunc != nil {
		return w.CommandExecFunc(ch, command)
	}
	return false
}

// doForced is a stub for perform_act / do_forced — received a forced command string
func (w *World) doForced(ch *Player, command string) bool {
	return w.executeCommand(ch, command)
}

// doBackstab handles the backstab command

// diceRoll rolls N dice of D sides each through the canonical stream.
func diceRoll(n, d int) int {
	return dprng.Dice(n, d)
}
