// Package game — Wave 2 skill commands from new_cmds2.c
package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// ---------------------------------------------------------------------------
// DoScrounge — do_scrounge() from new_cmds2.c
// Search room for edible items based on sector type.
// Port of new_cmds2.c:64-135. The C path draws number(1,101), waits two
// violence rounds after a skill attempt, and always emits the room search act
// once it reaches the sector switch.
// ---------------------------------------------------------------------------
func DoScrounge(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillScrounge) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You can't seem to find anything edible.\r\n",
			WaitCh:      2,
		}
	}

	// IS_MOUNTED is checked before the random draw in new_cmds2.c:77-80.
	if isMounted(ch) {
		return SkillResult{
			Success:     false,
			MessageToCh: "Dismount first!\r\n",
		}
	}

	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return SkillResult{}
	}

	sector := room.Sector

	// C consumes the roll before selecting the sector branch. number() is
	// inclusive, so the upper bound is 101 rather than 100.
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillScrounge)

	// find is TRUE only for the mountain branch. All other wilderness branches
	// use the capture-and-kill wording from new_cmds2.c:88-114.
	var (
		foodVNum int
		find     bool
	)

	switch sector {
	case SECT_FOREST:
		foodVNum = 28
	case SECT_FIELD, SECT_HILLS:
		foodVNum = 29
	case SECT_DESERT:
		foodVNum = 30
	case SECT_MOUNTAIN:
		foodVNum = 31
		find = true
	case SECT_WATER_SWIM, SECT_WATER_NOSWIM, SECT_UNDERWATER:
		foodVNum = 27
	default:
		return SkillResult{
			Success:       false,
			MessageToCh:   "You need to be in the wilderness to scrounge!\r\n",
			MessageToRoom: fmt.Sprintf("%s searches for something to eat.", ch.Name),
		}
	}

	roomMessage := fmt.Sprintf("%s searches for something to eat.", ch.Name)

	if percent < prob {
		// C only emits the find/capture act after read_object and obj_to_char
		// succeed. A missing prototype therefore still reaches the unconditional
		// room act but must not receive an invented player-facing fallback.
		proto, ok := world.objs[foodVNum]
		if !ok {
			return SkillResult{MessageToRoom: roomMessage}
		}
		obj := world.NewObjectFromProto(proto, ch.GetRoom())
		if obj == nil {
			return SkillResult{MessageToRoom: roomMessage}
		}
		if err := ch.Inventory.AddItem(obj); err != nil {
			world.ExtractObject(obj, ch.GetRoom())
			return SkillResult{MessageToRoom: roomMessage}
		}
		// C's do_scrounge hands the found object over with obj_to_char
		// (src/new_cmds2.c:119): PLR_CRASH is set (handler.c:569-571).
		ch.MarkCrashNeeded()

		message := "You capture and kill %s.\r\n"
		if find {
			message = "You find %s.\r\n"
		}
		return SkillResult{
			Success:         true,
			MessageToCh:     fmt.Sprintf(message, proto.ShortDesc),
			MessageToRoom:   roomMessage,
			WaitCh:          2,
			DeferredImprove: []string{SkillScrounge},
		}
	}

	return SkillResult{
		Success:       false,
		MessageToCh:   "You can't seem to find anything edible.\r\n",
		MessageToRoom: roomMessage,
		WaitCh:        2,
	}
}

// ---------------------------------------------------------------------------
// DoFirstAid — do_first_aid() from new_cmds2.c
// Heal a target who is at 0 HP. SKILL_FIRST_AID check.
// On success: target HP = 1. WAIT_STATE on target + ch.
// ---------------------------------------------------------------------------
func DoFirstAid(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillFirstAid) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how!\r\n"}
	}

	if target == ch {
		return SkillResult{Success: false, MessageToCh: "You wish you could.\r\n"}
	}

	if target.GetHP() >= 1 {
		return SkillResult{Success: false, MessageToCh: "They don't really need first aid.\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101+target.GetLevel())
	prob := ch.GetSkill(SkillFirstAid)

	if percent < prob || ch.GetLevel() > LVL_IMMORT {
		// Success
		if p, ok := target.(*Player); ok {
			p.SetHP(1)
			updatePosFromHP(p, 1)
		} else if mob, ok := target.(*MobInstance); ok {
			mob.SetHealth(1)
			updateMobPosFromHP(mob, 1)
		}

		chPronouns := GetPronouns(ch.Name, ch.GetSex())
		victPronouns := GetPronouns(target.GetName(), target.GetSex())

		return SkillResult{
			Success:         true,
			MessageToCh:     ActMessage("You apply some makeshift bandages to $N's wounds.", chPronouns, &victPronouns, ""),
			MessageToVict:   ActMessage("$n applies some bandaging to your wounds.", chPronouns, &victPronouns, ""),
			MessageToRoom:   ActMessage("$n applies some bandaging to $N's wounds.", chPronouns, &victPronouns, ""),
			WaitTarget:      1,
			WaitChPulses:    engine.PULSE_VIOLENCE + 3, // src/new_cmds2.c:187, both outcomes.
			DeferredImprove: []string{SkillFirstAid},
		}
	}

	// Failure
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	return SkillResult{
		Success:            false,
		MessageToCh:        "You fumble and ruin the bandages.\r\n",
		MessageToRoom:      ActMessage("$n fumbles with some bandaging and drops it all over the place!", chPronouns, nil, ""),
		WaitChPulses:       engine.PULSE_VIOLENCE + 3,
		RoomIncludesTarget: true,
	}
}

// ---------------------------------------------------------------------------
// DoDisarm — do_disarm() from new_cmds2.c
// Disarm opponent's weapon. SKILL_DISARM check. The weapon moves to the
// victim's inventory, matching obj_to_char(unequip_char(...), vict).
// Target must be fighting ch.
// ---------------------------------------------------------------------------
func DoDisarm(ch *Player, target combat.Combatant, world *World) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Disarm who?\r\n"}
	}

	if ch.GetSkill(SkillDisarm) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You'd better leave all the martial arts to fighters.\r\n",
		}
	}

	if target == ch {
		return SkillResult{Success: false, MessageToCh: "Just try removing your weapon instead.\r\n"}
	}

	var weapon *ObjectInstance
	switch victim := target.(type) {
	case *MobInstance:
		weapon = victim.Equipped(mobWearWield)
	case *Player:
		weapon, _ = victim.Equipment.GetItemInSlot(SlotWield)
	}
	if weapon == nil {
		chPronouns := GetPronouns(ch.Name, ch.GetSex())
		victPronouns := GetPronouns(target.GetName(), target.GetSex())
		return SkillResult{
			Success:     false,
			MessageToCh: ActMessage("$E doesn't have anything to disarm.", chPronouns, &victPronouns, ""),
		}
	}

	// C's command is POS_FIGHTING, and do_disarm also insists the target is
	// the actor's current opponent. Keep the handler-level check here for
	// direct callers and for the shared special-procedure seam.
	if ch.GetFightingBody() == nil || ch.GetFightingBody() != target {
		return SkillResult{Success: false, MessageToCh: "You can't disarm them if you aren't fighting them!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101+target.GetLevel())
	prob := ch.GetSkill(SkillDisarm)
	retaliate := target.GetFightingBody() == nil

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent < prob {
		// Unequip the target's wielded weapon (C: obj_to_char(unequip_char(vict, WEAR_WIELD), vict))
		if targetMob, ok := target.(*MobInstance); ok {
			targetMob.UnequipItem(mobWearWield)
		} else if targetPlayer, ok := target.(*Player); ok {
			if err := targetPlayer.Equipment.Unequip(SlotWield, targetPlayer.Inventory); err != nil {
				slog.Error("disarm failed to move weapon to player inventory", "actor", ch.GetName(), "target", target.GetName(), "error", err)
				return SkillResult{Success: false}
			}
			// C's disarm hands the weapon back to the victim —
			// obj_to_char(unequip_char(vict, WEAR_WIELD), vict) flags the
			// victim, not the actor (new_cmds2.c:236, handler.c:569-571).
			targetPlayer.MarkCrashNeeded()
		}

		return SkillResult{
			Success:                   true,
			Damage:                    0, // disarm doesn't directly damage
			MessageToCh:               ActMessage("You disarm $N and $p goes flying!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			MessageToVict:             ActMessage("$n deftly disarms you, knocking $p from your hand!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			MessageToRoom:             ActMessage("$n knocks $p from $N's hand!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			RetaliateHit:              retaliate,
			RetaliateHitAfterMessages: true,
			WaitCh:                    2,
			DeferredImprove:           []string{SkillDisarm},
		}
	}

	return SkillResult{
		Success:                   false,
		MessageToCh:               ActMessage("You try to disarm $N but fail, tumbling to the ground in the process!", chPronouns, &victPronouns, ""),
		MessageToVict:             ActMessage("$n tries to disarm you but fails and falls flat on $s face instead!", chPronouns, &victPronouns, ""),
		MessageToRoom:             ActMessage("$n tries to disarm $N, but fails and falls flat on $s face!", chPronouns, &victPronouns, ""),
		RetaliateHit:              retaliate,
		RetaliateHitAfterMessages: true,
		SelfStumble:               true,
		WaitCh:                    2,
	}
}

// ---------------------------------------------------------------------------
// DoMindlink — do_mindlink() simplified from new_cmds2.c
// Link minds for telepathic communication. Simplified version:
// Check target is in room, check skill, drain HP, share mana.
// ---------------------------------------------------------------------------
func DoMindlink(ch *Player, target combat.Combatant) SkillResult {
	if target == ch {
		return SkillResult{Success: false, MessageToCh: "You wish you could.\r\n"}
	}

	if ch.GetSkill(SkillMindlink) == 0 {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}

	// Target must be an NPC (not a player)
	if p, ok := target.(*Player); ok && !p.IsNPC() {
		chPronouns := GetPronouns(ch.Name, ch.GetSex())
		victPronouns := GetPronouns(target.GetName(), target.GetSex())
		return SkillResult{
			Success:            false,
			MessageToCh:        ActMessage("$N stares at you blankly.", chPronouns, &victPronouns, "") + "\r\nYou fail.",
			MessageToRoom:      ActMessage("$n stares at $N for a while and then falls flat on $s face.", chPronouns, &victPronouns, ""),
			RoomIncludesTarget: true,
		}
	}

	if ch.IsFighting() || target.GetFightingBody() != nil {
		return SkillResult{Success: false, MessageToCh: "There's too much going on to establish a mind link.\r\n"}
	}

	if ch.GetHP() < 100 {
		return SkillResult{Success: false, MessageToCh: "You don't have enough life to spare!\r\n"}
	}
	if mob, ok := target.(*MobInstance); ok && mob.GetMana() < 100 {
		return SkillResult{Success: false, MessageToCh: "They don't have enough energy to spare!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillMindlink)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// C's IS_PSIONIC/IS_MYSTIC macros both include !IS_NPC. Since the
	// non-NPC target arm returned above, this success arm is unreachable from
	// the command surface; retain the source-shaped test for clarity while
	// keeping the valid NPC path on C's failure arm.
	if _, ok := target.(*Player); ok && percent < prob {
		// Success
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		x := dprng.Number(20+ch.GetLevel(), 100)
		ch.SetHP(ch.GetHP() - x)
		if ch.GetHP() < 0 {
			ch.SetHP(0)
		}
		// Give mana to target
		if p, ok := target.(*Player); ok {
			p.SetMana(p.GetMana() + x)
		} else if m, ok := target.(*MobInstance); ok {
			m.SetMana(m.GetMana() + x)
		}

		return SkillResult{
			Success:       true,
			MessageToCh:   "You feel a little drained...\r\n",
			MessageToRoom: ActMessage("$n and $N stare at each other for a while and drop to the ground in unison!", chPronouns, &victPronouns, ""),
			StunTarget:    true,
			SelfStumble:   true,
		}
	}

	ch.SetHP(ch.GetHP() - 100)
	return SkillResult{
		Success:                  false,
		MessageToCh:              "You feel a little drained...\r\n",
		MessageToRoom:            ActMessage("$n stares at $N for a while and then falls flat on $s face.", chPronouns, &victPronouns, ""),
		DeferredImprove:          []string{SkillMindlink},
		DeferredImproveAfterRoom: true,
		MessageToChAfterRoom:     true,
		SelfStunnedAfterMessage:  true,
	}
}

// ---------------------------------------------------------------------------
// DoDetect — do_detect() from new_cmds2.c
// Detect hidden/magical things. SKILL_DETECT check.
// Find secret exits. WAIT_STATE.
// ---------------------------------------------------------------------------
func DoDetect(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillDetect) == 0 && ch.GetRace() != RaceElf {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}
	if ch.IsAffected(affBlind) {
		return SkillResult{Success: false, MessageToCh: "You're fucking blind, you can't find anything!!\r\n"}
	}

	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	var results strings.Builder
	results.WriteString("You carefully check the room...\r\n")
	prob := ch.GetSkill(SkillDetect)
	// C's number(1, 101) is inclusive; the roll follows the blind gate and
	// the initial room-check line is emitted before this branch.
	// #nosec G404 — game RNG, not cryptographic
	if prob <= dprng.Number(1, 101) {
		return SkillResult{
			Success:      false,
			MessageToCh:  results.String() + "You can't seem to find anything.\r\n",
			WaitChPulses: engine.PULSE_VIOLENCE + 1,
		}
	}

	// C scans the six exits in dirs[] order and uses case-sensitive strstr.
	var found bool
	for _, direction := range dirs {
		if exit, ok := room.Exits[direction]; ok && strings.Contains(exit.Keywords, "secret") {
			where := "the " + direction + " wall"
			switch direction {
			case "up":
				where = "the ceiling"
			case "down":
				where = "the floor"
			}
			fmt.Fprintf(&results, "You notice something funny about %s.\r\n", where)
			found = true
			// The flag write must go through the locked World setter:
			// room.Flags is shared mutable state and this runs on the
			// session goroutine, not the game loop.
			world.SetRoomFlagBit(ch.GetRoom(), roomFlagSecretMark)
		}
	}
	return SkillResult{Success: found, MessageToCh: results.String()}
}

// ---------------------------------------------------------------------------
// DoSerpentKick — do_serpent_kick() from new_cmds2.c lines 693-728
// Special spinning kick. SKILL_SERPENT_KICK check.
// Damage = level * 1.5. WAIT_STATE.
// Ported from C with all features: WAIT_STATE, IS_MOUNTED check,
// improve_skill, training mob spawn (1/81 at level 19+).
// C source: new_cmds2.c lines 693-728
// ---------------------------------------------------------------------------
func DoSerpentKick(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillSerpentKick) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You'd better leave all the martial arts to others.\r\n",
		}
	}

	if target == ch {
		return SkillResult{
			Success:     false,
			MessageToCh: "Aren't we funny today...\r\n",
		}
	}

	if isMounted(ch) {
		return SkillResult{
			Success:     false,
			MessageToCh: "Dismount first!\r\n",
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	prob := ch.GetSkill(SkillSerpentKick)

	if target.GetPosition() <= combat.PosSleeping {
		prob = 110 // auto-hit sleeping targets
	}

	if percent > prob {
		return SkillResult{
			Success:      false,
			SkillMsgType: SkillSerpentKickNum,
			StartCombat:  true,
			WaitCh:       2, // PULSE_VIOLENCE * 2 — C source: WAIT_STATE(ch, PULSE_VIOLENCE * 2)
		}
	}

	dam := int(float64(ch.GetLevel()) * 1.5)

	return SkillResult{
		Success:         true,
		Damage:          dam,
		SkillMsgType:    SkillSerpentKickNum,
		DamageSkill:     SkillSerpentKick,
		StartCombat:     true,
		WaitCh:          2, // PULSE_VIOLENCE * 2 — C source: WAIT_STATE(ch, PULSE_VIOLENCE * 2)
		DeferredImprove: []string{SkillSerpentKick},
		// C draws this branch after damage()/skill_message and only for
		// level > 18. The wrapper consumes it before DeferredImprove.
		SpawnMobVNum:    18221,
		SpawnMobLevel:   ch.GetLevel() + 3,
		SpawnMobRoom:    18201,
		SpawnMobHunting: true,
	}
}

// ---------------------------------------------------------------------------
// DoDig — do_dig() from new_cmds2.c
// Simplified version: dig in current room based on sector type.
// WAIT_STATE, move cost.
// Sector types:
//
//	SECT_DIRT (2), SECT_FOREST (3), SECT_FIELD (4), SECT_HILLS (5)
//
// Success chance based on SKILL_DIG. Finds random loot.
// ---------------------------------------------------------------------------

// DoTurn ports do_turn() from src/new_cmds2.c:960-1030. The command ignores
// its arguments, spends one random mana cost before scanning the C-ordered
// room list, and applies one of three race/level outcomes to each target.
func (w *World) DoTurn(ch *Player) {
	if ch.GetSkill(SkillTurn) == 0 {
		ch.SendMessage("Huh?!?\r\n")
		return
	}
	if !ch.IsGood() {
		ch.SendMessage("You are not holy enough to turn away the Undead!\r\n")
		return
	}

	ch.SendMessage("You attempt to turn away the unholy presence in this room.\r\n")
	Act(w, false, ch, nil, nil, nil, "$n's eyes are suddenly filled with blinding white light!", "", ToRoom)
	needMana := ch.GetLevel()*3 + dprng.Number(1, 55)
	if needMana > ch.GetMana() {
		ch.SendMessage("You fail to summon enough energy to dispel the darkness!\r\n")
		mana := ch.GetMana() - needMana/4
		if mana < 0 {
			mana = 0
		}
		ch.SetMana(mana)
		return
	}
	ch.SetMana(ch.GetMana() - needMana)

	for _, body := range w.GetAllCharsInRoom(ch.GetRoom()) {
		target, ok := body.(combat.Combatant)
		if !ok || target.GetRoom() != ch.GetRoom() {
			continue
		}
		race := turnRace(target)
		if race != combat.RACE_UNDEAD && race != combat.RACE_VAMPIRE {
			continue
		}
		diff := ch.GetLevel() - target.GetLevel()
		switch {
		case diff <= -5:
			Act(w, false, target, nil, nil, nil, "$n shivers uncomfortably.", "", ToRoom)
			actForPrivateBody(w, false, target, nil, nil, nil, "A disturbing feeling washes over your body.", "", ToChar, target)
		case diff >= 15:
			Act(w, false, target, nil, nil, nil, "$n grimaces and then explodes into a cloud of dust!", "", ToRoom)
			actForPrivateBody(w, false, target, nil, nil, nil, "You feel your body twist horribly and disintegrate into nothing!", "", ToChar, target)
			w.RawKillCombatant(target, 93)
		case diff > 3:
			Act(w, false, target, nil, nil, nil, "$n shrieks in terror!", "", ToRoom)
			actForPrivateBody(w, false, target, nil, nil, nil, "You are suddenly terrified!", "", ToChar, target)
			w.DoFlee(target)
		}
	}
}

func turnRace(body combat.Combatant) int {
	switch body := body.(type) {
	case *Player:
		return body.GetRace()
	case *MobInstance:
		if proto := body.Proto(); proto != nil {
			return proto.Race
		}
	}
	return -1
}
