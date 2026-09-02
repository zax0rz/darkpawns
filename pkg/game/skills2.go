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
		obj := NewObjectInstance(proto, ch.GetRoom())
		if obj == nil {
			return SkillResult{MessageToRoom: roomMessage}
		}
		if err := ch.Inventory.AddItem(obj); err != nil {
			return SkillResult{MessageToRoom: roomMessage}
		}

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

	if target.GetName() == ch.Name {
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

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Just try removing your weapon instead.\r\n"}
	}

	var weapon *ObjectInstance
	switch victim := target.(type) {
	case *MobInstance:
		weapon = victim.Equipment[int(SlotWield)]
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
	if ch.GetFighting() == "" || ch.GetFighting() != target.GetName() {
		return SkillResult{Success: false, MessageToCh: "You can't disarm them if you aren't fighting them!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 101+target.GetLevel())
	prob := ch.GetSkill(SkillDisarm)
	retaliate := target.GetFighting() == ""

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent < prob {
		// Unequip the target's wielded weapon (C: obj_to_char(unequip_char(vict, WEAR_WIELD), vict))
		if targetMob, ok := target.(*MobInstance); ok {
			targetMob.UnequipItem(int(SlotWield))
		} else if targetPlayer, ok := target.(*Player); ok {
			if err := targetPlayer.Equipment.Unequip(SlotWield, targetPlayer.Inventory); err != nil {
				slog.Error("disarm failed to move weapon to player inventory", "actor", ch.GetName(), "target", target.GetName(), "error", err)
				return SkillResult{Success: false}
			}
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
	if target.GetName() == ch.Name {
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

	if ch.IsFighting() || target.GetFighting() != "" {
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
	if ch.GetSkill(SkillDetect) == 0 && ch.GetClass() != RaceElf {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}

	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	prob := ch.GetSkill(SkillDetect)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	if prob <= dprng.Number(1, 100) {
		return SkillResult{Success: false, MessageToCh: "You can't seem to find anything.\r\n"}
	}

	// Check exits for "secret" keyword
	var found bool
	results := "You carefully check the room...\r\n"
	for dir, exit := range room.Exits {
		if strings.Contains(strings.ToLower(exit.Keywords), "secret") {
			dirNames := map[string]string{
				"north": "the north wall",
				"south": "the south wall",
				"east":  "the east wall",
				"west":  "the west wall",
				"up":    "the ceiling",
				"down":  "the floor",
				"n":     "the north wall",
				"s":     "the south wall",
				"e":     "the east wall",
				"w":     "the west wall",
				"u":     "the ceiling",
				"d":     "the floor",
			}
			where := dirNames[dir]
			if where == "" {
				where = fmt.Sprintf("the %s wall", dir)
			}
			results += fmt.Sprintf("You notice something funny about %s.\r\n", where)
			found = true
		}
	}

	if !found {
		results += "You can't seem to find anything.\r\n"
	}

	return SkillResult{Success: found, MessageToCh: results}
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

	if target.GetName() == ch.Name {
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
func DoDig(ch *Player, world *World) SkillResult {
	if ch.GetHP() < 5 {
		return SkillResult{Success: false, MessageToCh: "You're too exhausted to dig.\r\n"}
	}

	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	sector := room.Sector
	switch sector {
	case 2, 3, 4, 5: // SECT_DIRT, SECT_FOREST, SECT_FIELD, SECT_HILLS
		// Valid digging terrain
	default:
		return SkillResult{
			Success:     false,
			MessageToCh: "The ground here isn't suitable for digging.\r\n",
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := dprng.Number(1, 100)
	prob := ch.GetSkill(SkillDig)
	if prob == 0 {
		prob = 10
	}

	if percent <= prob {
		// Found something — random loot
		lootRoll := dprng.Number(0, 4)
		if lootRoll == 0 {
			goldAmt := dprng.Number(10, 50) // 10-50 gold
			ch.mu.Lock()
			ch.Gold += goldAmt
			ch.mu.Unlock()
			return SkillResult{
				Success:       true,
				MessageToCh:   fmt.Sprintf("You dig in the earth and find %d gold coins!\r\n", goldAmt),
				MessageToRoom: fmt.Sprintf("%s digs in the earth and finds some gold coins!\r\n", ch.Name),
			}
		} else {
			obj, err := world.SpawnObject(3001, ch.GetRoomVNum())
			if err == nil && obj != nil {
				world.GiveObjectToChar(obj, ch)
				return SkillResult{
					Success:       true,
					MessageToCh:   fmt.Sprintf("You dig in the earth and find %s!\r\n", obj.GetShortDesc()),
					MessageToRoom: fmt.Sprintf("%s digs in the earth and finds something!\r\n", ch.Name),
				}
			}
		}
	}

	return SkillResult{
		Success:       false,
		MessageToCh:   "You dig but find nothing.\r\n",
		MessageToRoom: fmt.Sprintf("%s digs around but finds nothing.\r\n", ch.Name),
	}
}

// ---------------------------------------------------------------------------
// DoTurn — do_turn() from new_cmds2.c
// Turn undead. SKILL_TURN check. Target must be undead.
// Damage = level * 2. If diff > 3: flee. If diff >= 15: destroy.
// WAIT_STATE.
// ---------------------------------------------------------------------------
func DoTurn(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillTurn) == 0 {
		return SkillResult{Success: false, MessageToCh: "Huh?!?\r\n"}
	}

	if !ch.IsGood() {
		return SkillResult{
			Success:     false,
			MessageToCh: "You are not holy enough to turn away the Undead!\r\n",
		}
	}

	msgToCh := "You attempt to turn away the unholy presence in this room.\r\n"

	// Check if target is undead race (3 = RACE_UNDEAD in C, using our own convention)
	// In the C source, it checks GET_RACE(tch) == RACE_UNDEAD || RACE_VAMPIRE
	// We don't have full race tracking on combatants yet, so we check the name for clues
	// or assume any target that can be turned is valid
	targetName := strings.ToLower(target.GetName())
	isUndead := strings.Contains(targetName, "skeleton") ||
		strings.Contains(targetName, "zombie") ||
		strings.Contains(targetName, "ghost") ||
		strings.Contains(targetName, "lich") ||
		strings.Contains(targetName, "wraith") ||
		strings.Contains(targetName, "spectre") ||
		strings.Contains(targetName, "vampire") ||
		strings.Contains(targetName, "undead")

	if !isUndead {
		return SkillResult{
			Success:     false,
			MessageToCh: msgToCh + "There is nothing unholy to turn here.\r\n",
		}
	}

	diff := ch.GetLevel() - target.GetLevel()

	if diff <= -5 {
		return SkillResult{
			Success:       false,
			MessageToCh:   msgToCh + "A disturbing feeling washes over your body.\r\n",
			MessageToRoom: fmt.Sprintf("%s shivers uncomfortably.\r\n", target.GetName()),
		}
	}

	if diff >= 15 {
		return SkillResult{
			Success:     true,
			Damage:      9999, // instant kill
			MessageToCh: msgToCh + "The undead creature explodes into a cloud of dust!\r\n",
			MessageToRoom: fmt.Sprintf("%s grimaces and then explodes into a cloud of dust!\r\n",
				target.GetName()),
			MessageToVict: "You feel your body twist horribly and disintegrate into nothing!\r\n",
		}
	}

	if diff > 3 {
		return SkillResult{
			Success:       true,
			Damage:        ch.GetLevel() * 2,
			MessageToCh:   msgToCh + "The undead creature shrieks and flees from your holiness!\r\n",
			MessageToRoom: fmt.Sprintf("%s shrieks in terror!\r\n", target.GetName()),
		}
	}

	// Basic damage
	dam := ch.GetLevel() * 2
	if dam < 1 {
		dam = 1
	}
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   msgToCh + fmt.Sprintf("Your holy power sears the undead for %d damage!\r\n", dam),
		MessageToVict: "The holy light sears your undead flesh!\r\n",
		MessageToRoom: fmt.Sprintf("%s is bathed in holy light!\r\n", ch.Name),
	}
}
