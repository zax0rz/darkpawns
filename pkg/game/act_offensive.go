package game

// act_offensive.go — port of src/act.offensive.c
//
// All rights reserved. See license.doc for complete information.
//
// Copyright (C) 1993, 94 by the Trustees of the Johns Hopkins University
// CircleMUD is based on DikuMUD, Copyright (C) 1990, 1991.
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// This includes all original code done for Dark Pawns MUD by other authors.
// All code is the intellectual property of the author, and is used here
// by permission.
//
// No original code may be duplicated, reused, or executed without the
// written permission of the author. All rights reserved.

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Skill constants not yet defined in skills.go
// ---------------------------------------------------------------------------

const (
	SkillDisembowel = "disembowel"
	SkillEscape     = "escape"
	SkillRetreat    = "retreat"
	SkillShoot      = "shoot"
	SkillDragonKick = "dragon_kick"
	SkillTigerPunch = "tiger_punch"
	SkillSubdue     = "subdue"
	SkillSleeper    = "sleeper"
	SkillNeckbreak  = "neckbreak"
	SkillAmbush     = "ambush"
)

// lvlImpl — implementor level for kill command (LVL_IMPL in structs.h)
// Source: act.offensive.c do_kill() check
const lvlImpl = 40

// Internal helpers (ported from C macros)
// ---------------------------------------------------------------------------

// affMounted — AFF_MOUNT bit position from structs.h
const affMounted = 29

// IS_MOUNTED — from act.offensive.c: checks if a player is mounted.
func isMounted(ch *Player) bool {
	return ch.IsAffected(affMounted)
}

// IS_OUTLAW — from act.offensive.c (used in subdue and sleeper)
func isOutlaw(ch *Player) bool {
	return ch.Flags&plrOutlaw != 0
}

// isShopkeeper checks if a victim is a shopkeeper mob.
func isShopkeeper(w *World, victim *Player) bool {
	// In the C code, this checks sh_int spec of the mob prototype.
	// For simplicity, check if the victim is NPC and has shop-related specs.
	// This is a placeholder implementation.
	_ = w
	return false
}

// isPiercingWeapon checks if a weapon is a piercing type (dagger, etc.)
func isPiercingWeapon(obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return false
	}
	// CircleMUD: TYPE_PIERCE weapon type
	return obj.Prototype.Values[3] == 11
}

// improveSkill is a stub for skill improvement.
func improveSkill(ch *Player, skill string) {
	// TODO: Implement proper skill improvement logic
	_ = ch
	_ = skill
}

// ---------------------------------------------------------------------------
// rawKill — handles immediate death (raw_kill() from fight.c)
// ---------------------------------------------------------------------------

// rawKill immediately kills the target with the given attack type.
func (w *World) rawKill(victim *Player, attackType int) {
	// Handle death via existing infrastructure
	// Make a corpse first so items are preserved
	corpse := w.makeCorpse(victim.GetName(), nil, nil, victim.RoomVNum, attackType)
	_ = corpse // corpse is placed in the room by makeCorpse

	// Trigger death processing
	w.HandleDeath(victim, nil, attackType)
}

// ---------------------------------------------------------------------------
// do_assist — assist someone in combat (line 54)
// ---------------------------------------------------------------------------

func (w *World) doAssist(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsFighting() {
		ch.SendMessage("You are already fighting!\r\n")
		return true
	}

	args := strings.Fields(arg)
	var helpee *Player

	if len(args) >= 1 {
		helpee = w.getCharRoomVis(ch, args[0])
	}

	if helpee == nil {
		ch.SendMessage("Assist who?\r\n")
		return true
	}

	if !helpee.IsFighting() {
		ch.SendMessage("They aren't fighting anyone!\r\n")
		return true
	}

	// Target is whoever the helpee is fighting
	targetName := helpee.GetFighting()

	// Find the target in the room
	var target *Player
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == targetName {
			target = p
			break
		}
	}

	if target == nil {
		// For mobs, check mob list
		ch.SendMessage("They don't seem to be fighting anyone here.\r\n")
		return true
	}

	if target == ch {
		ch.SendMessage("But that would be suicide!\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You assist %s!\r\n", helpee.GetName()))
	helpee.SendMessage(fmt.Sprintf("%s assists you!\r\n", ch.GetName()))
	w.roomMessageExcludeTwo(ch.RoomVNum,
		fmt.Sprintf("%s assists %s!", ch.GetName(), helpee.GetName()),
		ch.GetName(), helpee.GetName())

	w.startCombatBetween(ch, target)
	return true
}

// ---------------------------------------------------------------------------
// do_hit — punch/attack unarmed (line 101)
// ---------------------------------------------------------------------------

func (w *World) doHit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() && me != nil {
		// NPC: attack their fighting target if no argument
		if !ch.IsFighting() {
			ch.SendMessage("But you aren't fighting!\r\n")
			return true
		}
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Hit who?\r\n")
		return true
	}

	victName := args[0]
	vict := w.getCharRoomVis(ch, victName)

	// Check mobs too
	if vict == nil {
		for _, m := range w.GetMobsInRoom(ch.RoomVNum) {
			if strings.Contains(strings.ToLower(m.GetName()), strings.ToLower(victName)) {
				// For mob targets, create combat with the mob directly
				ch.SendMessage(fmt.Sprintf("You hit %s!\r\n", m.GetName()))
				w.roomMessageExcludeTwo(ch.RoomVNum,
					fmt.Sprintf("%s hits %s!", ch.GetName(), m.GetName()),
					ch.GetName(), m.GetName())
				w.startCombatBetween(ch, nil)
				ch.SetFighting(m.GetName())
				return true
			}
		}
	}

	if vict == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("You hit yourself... Ouch!\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You hit %s!\r\n", vict.GetName()))
	vict.SendMessage(fmt.Sprintf("%s hits you!\r\n", ch.GetName()))
	w.roomMessageExcludeTwo(ch.RoomVNum,
		fmt.Sprintf("%s hits %s!", ch.GetName(), vict.GetName()),
		ch.GetName(), vict.GetName())

	w.startCombatBetween(ch, vict)
	return true
}

// ---------------------------------------------------------------------------
// do_kill — declare kill intent / attack (line 134)
// ---------------------------------------------------------------------------

func (w *World) doKill(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetLevel() < lvlImpl-1 || !ch.IsNPC() {
		// Non-immortals just use doHit
		return w.doHit(ch, me, cmd, arg)
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Kill who?\r\n")
		return true
	}

	victName := args[0]
	vict := w.getCharRoomVis(ch, victName)
	if vict == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}
	if ch == vict {
		ch.SendMessage("Your mother would be so sad.. :(\r\n")
		return true
	}
	if vict.GetLevel() == ch.GetLevel() {
		ch.SendMessage("No can do, buddy.. \r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You chop %s to pieces!  Ah!  The blood!\r\n", himHer(vict.GetSex())))
	vict.SendMessage(fmt.Sprintf("%s chops you to pieces!\r\n", ch.Name))
	w.roomMessageExcludeTwo(ch.RoomVNum,
		fmt.Sprintf("%s brutally slays %s!", ch.Name, vict.Name),
		ch.Name, vict.Name)

	w.rawKill(vict, 303)
	return true
}

// ---------------------------------------------------------------------------
// do_backstab — backstab from behind (line 165)
// ---------------------------------------------------------------------------

func (w *World) doBackstab(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillBackstab) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Backstab who?\r\n")
		return true
	}

	victName := args[0]
	vict := w.getCharRoomVis(ch, victName)
	if vict == nil {
		// Could be a mob
		for _, m := range w.GetMobsInRoom(ch.RoomVNum) {
			if strings.Contains(strings.ToLower(m.GetName()), strings.ToLower(victName)) {
				ch.SendMessage("They aren't here.\r\n")
				return true
			}
		}
		ch.SendMessage("Backstab who?\r\n")
		return true
	}
	if vict == ch {
		ch.SendMessage("How can you sneak up on yourself?\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded == nil {
		ch.SendMessage("You need to wield a weapon to make it a success.\r\n")
		return true
	}
	if !isPiercingWeapon(wielded) {
		ch.SendMessage("Only piercing weapons can be used for backstabbing.\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("You can't backstab a fighting person -- they're too alert!\r\n")
		return true
	}

	if vict.IsNPC() && vict.GetPosition() >= posSleeping {
		vict.SendMessage(fmt.Sprintf("You notice %s lunging at you!\r\n", ch.Name))
		ch.SendMessage(fmt.Sprintf("%s notices you lunging at %s!\r\n", vict.Name, himHer(vict.GetSex())))
		w.roomMessageExcludeTwo(ch.RoomVNum,
			fmt.Sprintf("%s notices %s lunging at %s!", vict.Name, ch.Name, himHer(vict.GetSex())),
			ch.Name, vict.Name)
		w.startCombatBetween(vict, ch)
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillBackstab)

	if vict.GetPosition() >= combat.PosSleeping && percent > prob {
		w.doDamage(ch, vict, 0, SkillBackstab)
	} else {
		w.hitSkill(ch, vict, SkillBackstab)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_disembowel — disembowel attack (line 234)
// ---------------------------------------------------------------------------

func (w *World) doDisembowel(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillDisembowel) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	args := strings.Fields(arg)
	var vict *Player

	if len(args) >= 1 {
		vict = w.getCharRoomVis(ch, args[0])
	}

	if vict == nil {
		if ch.IsFighting() {
			fightingName := ch.GetFighting()
			for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
				if p.Name == fightingName {
					vict = p
					break
				}
			}
		}
		if vict == nil {
			ch.SendMessage("Disembowel who?\r\n")
			return true
		}
	}
	if vict == ch {
		ch.SendMessage("Nah. Hari Kari is for wimps.\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded == nil {
		ch.SendMessage("You need to wield a weapon to make it a success.\r\n")
		return true
	}
	if !isPiercingWeapon(wielded) {
		ch.SendMessage("Only piercing weapons can be used for disemboweling.\r\n")
		return true
	}
	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillDisembowel)

	if vict.GetPosition() >= combat.PosSleeping && percent > prob {
		w.doDamage(ch, vict, 0, SkillDisembowel)
	} else {
		w.hitSkill(ch, vict, SkillDisembowel)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_order — order follower/mount (line 294)
// ---------------------------------------------------------------------------

func (w *World) doOrder(ch *Player, me *MobInstance, cmd string, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) < 2 {
		ch.SendMessage("Order who to do what?\r\n")
		return true
	}

	name := parts[0]
	message := strings.Join(parts[1:], " ")

	if ch.IsAffected(affCharm) {
		ch.SendMessage("Your superior would not approve of you giving orders.\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, name)
	if vict != nil && vict == ch {
		ch.SendMessage("You obviously suffer from schizophrenia.\r\n")
		return true
	}

	if vict != nil {
		vict.SendMessage(fmt.Sprintf("%s orders you to '%s'\r\n", ch.Name, message))
		w.roomMessageExcludeTwo(ch.RoomVNum,
			fmt.Sprintf("%s gives %s an order.", ch.Name, vict.Name),
			ch.Name, vict.Name)

		if vict.Following != ch.Name && !vict.IsAffected(affCharm) {
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s has an indifferent look.", vict.Name))
		} else {
			ch.SendMessage("Ok.\r\n")
			w.executeCommand(vict, message)
		}
	} else if strings.HasPrefix(strings.ToLower(name), "follower") || name == "followers" {
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s issues the order '%s'.", ch.Name, message))

		followers := w.GetFollowersInRoom(ch.Name, ch.RoomVNum)
		found := false
		for _, follower := range followers {
			if follower.IsAffected(affCharm) {
				found = true
				w.executeCommand(follower, message)
			}
		}
		if found {
			ch.SendMessage("Ok.\r\n")
		} else {
			ch.SendMessage("Nobody here is a loyal subject of yours!\r\n")
		}
	} else {
		ch.SendMessage("That person isn't here.\r\n")
	}

	return true
}

// ---------------------------------------------------------------------------
// do_flee — flee from combat (line 360)
// ---------------------------------------------------------------------------

func (w *World) doFlee(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() && me != nil {
		return true
	}

	if ch.GetPosition() < combat.PosFighting {
		ch.SendMessage("Get on your feet first!\r\n")
		return true
	}

	if !ch.IsFighting() {
		ch.SendMessage("Flee from what?  You aren't fighting!\r\n")
		return true
	}

	percent := randRange(1, 101) // 101% is a complete failure
	prob := ch.GetSkill(SkillFlee)

	if percent > prob {
		ch.SendMessage("You try to flee but get cornered in the process!\r\n")
		improveSkill(ch, SkillFlee)
		return true
	}

	// Try up to 6 random directions
	dirs := []string{"north", "east", "south", "west", "up", "down"}
	for i := 0; i < 6; i++ {
		attempt := randRange(0, len(dirs)-1)
		dirStr := dirs[attempt]

		room := w.GetRoomInWorld(ch.RoomVNum)
		if room == nil {
			continue
		}
		exit, hasExit := room.Exits[dirStr]
		if !hasExit || exit.ToRoom <= 0 {
			continue
		}
		if w.roomHasFlag(exit.ToRoom, "death") {
			ch.SendMessage("That path would be certain death!\r\n")
			continue
		}

		if exit.DoorState == 1 {
			continue
		}

		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s flees from combat!", ch.Name))
		doSimpleMove(w, ch, attempt, true)
		ch.SendMessage("You flee head over heels!\r\n")
		ch.StopFighting()
		return true
	}

	ch.SendMessage("You are cornered and fail to flee!\r\n")
	w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s tries to flee but is cornered!", ch.Name))
	return true
}

// ---------------------------------------------------------------------------
// do_bash — bash / doorbash opponent (line 419)
// ---------------------------------------------------------------------------

func (w *World) doBash(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillBash) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	args := strings.Fields(arg)
	var vict *Player

	if len(args) >= 1 {
		vict = w.getCharRoomVis(ch, args[0])
	}

	if vict == nil && ch.IsFighting() {
		fightingName := ch.GetFighting()
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if p.Name == fightingName {
				vict = p
				break
			}
		}
	}

	if vict == nil && len(args) >= 1 {
		// Check for door bash
		dirNum := -1
		dirs := []string{"north", "east", "south", "west", "up", "down"}
		for i, d := range dirs {
			if strings.EqualFold(args[0], d) {
				dirNum = i
				break
			}
		}

		if dirNum >= 0 {
			room := w.GetRoomInWorld(ch.RoomVNum)
			if room == nil {
				return true
			}
			exit, hasExit := room.Exits[dirs[dirNum]]
			if !hasExit || exit.ToRoom <= 0 {
				ch.SendMessage("There is no exit in that direction.\r\n")
				return true
			}

			// Door bash
			if exit.DoorState != 1 {
				ch.SendMessage("It's not closed!\r\n")
				return true
			}

			prob := ch.GetSkill(SkillBash)
			percent := randRange(1, prob)
			ch.SendMessage(fmt.Sprintf("You hit the %s with all your might!\r\n", exit.Description))

			if percent > 80 {
				// Removed door // exit.DoorState == 1 = false
				ch.SendMessage("Batter up!  You break the door down!\r\n")
				w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s smashes the %s to pieces!", ch.Name, exit.Description))
				improveSkill(ch, SkillBash)
			} else {
				ch.SendMessage("WHAM!!!\r\n")
				ch.SendMessage("It doesn't seem to help.\r\n")
				w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s smashes against the %s!", ch.Name, exit.Description))
			}
			return true
		}

		ch.SendMessage("Bash who?\r\n")
		return true
	}

	if vict == nil {
		ch.SendMessage("Bash who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("You bash yourself... Ouch!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You feel too peaceful to contemplate violence.\r\n")
		return true
	}

	percent := randRange(1, 101) // 101% is a complete failure
	prob := ch.GetSkill(SkillBash)

	if vict.IsNPC() && false {
		prob = 0
	}

	if percent > prob {
		w.doDamage(ch, vict, 0, SkillBash)
	} else if w.doDamage(ch, vict, (ch.GetLevel()/2)+1, SkillBash) {
		improveSkill(ch, SkillBash)
		// Victim is bashed down
		vict.SetPosition(combat.PosFighting)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_rescue — rescue someone from combat (line 501)
// ---------------------------------------------------------------------------

func (w *World) doRescue(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillRescue) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Rescue who?\r\n")
		return true
	}

	victName := args[0]
	vict := w.getCharRoomVis(ch, victName)
	if vict == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("What about the other person?\r\n")
		return true
	}

	if !vict.IsFighting() {
		ch.SendMessage("They are not fighting!\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You are already fighting!\r\n")
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillRescue)

	if percent > prob {
		ch.SendMessage("You fail the rescue!\r\n")
		improveSkill(ch, SkillRescue)
		return true
	}

	// Find who the victim is fighting
	opponentName := vict.GetFighting()
	var opponent *Player
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == opponentName {
			opponent = p
			break
		}
	}

	if opponent == nil {
		ch.SendMessage("They aren't fighting anyone!\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You rescue %s!\r\n", vict.GetName()))
	vict.SendMessage(fmt.Sprintf("You are rescued by %s!\r\n", ch.GetName()))
	vict.StopFighting()
	ch.SetFighting(opponent.GetName())
	opponent.SetFighting(ch.GetName())

	w.roomMessageExcludeTwo(ch.RoomVNum,
		fmt.Sprintf("%s rescues %s!", ch.Name, vict.GetName()),
		ch.Name, vict.GetName())

	improveSkill(ch, SkillRescue)
	return true
}

// ---------------------------------------------------------------------------
// do_kick — kick attack (line 587)
// ---------------------------------------------------------------------------

func (w *World) doKick(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillKick) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	var vict *Player
	args := strings.Fields(arg)

	if len(args) >= 1 {
		vict = w.getCharRoomVis(ch, args[0])
	}

	if vict == nil && ch.IsFighting() {
		fightingName := ch.GetFighting()
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if p.Name == fightingName {
				vict = p
				break
			}
		}
	}

	if vict == nil {
		if len(args) >= 1 {
			ch.SendMessage("They aren't here.\r\n")
		} else {
			ch.SendMessage("Kick who?\r\n")
		}
		return true
	}

	if vict == ch {
		ch.SendMessage("You kick yourself... ouch!\r\n")
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillKick)

	if percent > prob {
		w.doDamage(ch, vict, 0, SkillKick)
	} else {
		w.doDamage(ch, vict, ch.GetLevel()>>1, SkillKick)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_dragon_kick — dragon kick attack (line 636)
// ---------------------------------------------------------------------------

func (w *World) doDragonKick(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillDragonKick) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	var vict *Player
	args := strings.Fields(arg)

	if len(args) >= 1 {
		vict = w.getCharRoomVis(ch, args[0])
	}

	if vict == nil && ch.IsFighting() {
		fightingName := ch.GetFighting()
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if p.Name == fightingName {
				vict = p
				break
			}
		}
	}

	if vict == nil {
		if len(args) >= 1 {
			ch.SendMessage("They aren't here.\r\n")
		} else {
			ch.SendMessage("Kick who?\r\n")
		}
		return true
	}

	if vict == ch {
		ch.SendMessage("You kick yourself... ouch!\r\n")
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillDragonKick)

	if percent > prob {
		w.doDamage(ch, vict, 0, SkillDragonKick)
	} else {
		damage := int(float64(ch.GetLevel()) * 1.5)
		w.doDamage(ch, vict, damage, SkillDragonKick)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_tiger_punch — tiger punch attack (line 693)
// ---------------------------------------------------------------------------

func (w *World) doTigerPunch(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillTigerPunch) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	var vict *Player
	args := strings.Fields(arg)

	if len(args) >= 1 {
		vict = w.getCharRoomVis(ch, args[0])
	}

	if vict == nil && ch.IsFighting() {
		fightingName := ch.GetFighting()
		for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
			if p.Name == fightingName {
				vict = p
				break
			}
		}
	}

	if vict == nil {
		ch.SendMessage("Punch who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("You punch yourself... ouch!\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded != nil {
		ch.SendMessage("You are not skilled at making fists with a weapon in hand.\r\n")
		return true
	}

	percent := randRange(1, 101)
	prob := ch.GetSkill(SkillTigerPunch)

	if percent > prob {
		w.doDamage(ch, vict, 0, SkillTigerPunch)
	} else {
		damage := int(float64(ch.GetLevel()) * 2.5)
		w.doDamage(ch, vict, damage, SkillTigerPunch)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_shoot — ranged/shoot attack (line 746)
// ---------------------------------------------------------------------------

func (w *World) doShoot(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillShoot) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("But you are already engaged in close-range combat!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 3 {
		ch.SendMessage("Shoot what where?\r\n")
		return true
	}

	projectileName := args[0]
	dirStr := args[1]
	targetName := strings.Join(args[2:], " ")

	// Find projectile in inventory
	var projectile *ObjectInstance
	for _, item := range ch.Inventory.Items {
		if item.Prototype != nil && strings.Contains(strings.ToLower(item.Prototype.ShortDesc), strings.ToLower(projectileName)) {
			projectile = item
			break
		}
	}
	if projectile == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", projectileName))
		return true
	}
	if projectile.Prototype == nil || projectile.Prototype.TypeFlag != 7 {
		ch.SendMessage(fmt.Sprintf("%s is not a projectile!\r\n", projectile.GetShortDesc()))
		return true
	}

	// Find direction
	dir := -1
	for i, d := range dirs {
		if strings.EqualFold(dirStr, d) {
			dir = i
			break
		}
	}
	if dir < 0 {
		ch.SendMessage("Interesting direction.\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return true
	}

	exit, hasExit := room.Exits[dirs[dir]]
	if !hasExit || exit.ToRoom <= 0 {
		ch.SendMessage("Alas, you cannot shoot that way...\r\n")
		return true
	}

	if exit.DoorState == 1 {
		if exit.Keywords != "" {
			ch.SendMessage(fmt.Sprintf("The %s seems to be closed.\r\n", exit.Keywords))
		} else {
			ch.SendMessage("It seems to be closed.\r\n")
		}
		return true
	}

	if w.roomHasFlag(exit.ToRoom, "peaceful") || w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You feel too peaceful to contemplate violence.\r\n")
		return true
	}

	// Check if wielding a bow
	bow := w.GetEquipped(ch, eqWearWield)
	if bow == nil || bow.Prototype == nil || bow.Prototype.TypeFlag != 6 {
		ch.SendMessage("You must wield a bow or sling to fire a projectile.\r\n")
		return true
	}

	// Find target in target room
	var target *Player
	playersInRoom := w.GetPlayersInRoom(exit.ToRoom)
	for _, p := range playersInRoom {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName)) {
			target = p
			break
		}
	}

	// Direction name for messages
	from := "somewhere"
	switch dir {
	case 0:
		from = "the south"
	case 1:
		from = "the west"
	case 2:
		from = "the north"
	case 3:
		from = "the east"
	case 4:
		from = "below"
	case 5:
		from = "above"
	}

	w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s fires %s %s with %s.", ch.Name, an(projectileName), projectileName, bow.GetShortDesc()))
	ch.SendMessage("Twang... your projectile flies into the distance.\r\n")

	_, _ = targetName, from // used if we find a target below

	// Check target if found
	if target != nil {
		if !target.IsNPC() && (target.GetLevel() < 10 || target.GetLevel() > 30) {
			ch.SendMessage("Maybe that isn't such a great idea...\r\n")
			return true
		}

		if target.IsFighting() {
			ch.SendMessage("It looks like they are fighting, you can't aim properly.\r\n")
			return true
		}

		if false {
			ch.SendMessage("You cannot see well enough to aim...\r\n")
			return true
		}

		// Fire at target
		percent := randRange(1, 101)
		prob := ch.GetSkill(SkillShoot) + (ch.GetDex()*10 - target.GetDex()*10)

		if percent < prob {
			// Hit!
			dam := ch.GetDamroll()
			if projectile.Prototype != nil {
				diceNum := projectile.Prototype.Values[0]
				diceSize := projectile.Prototype.Values[1]
				dam += diceRoll(diceNum, diceSize)
			}
			if bow.Prototype != nil {
				diceNum := bow.Prototype.Values[0]
				diceSize := bow.Prototype.Values[1]
				dam += diceRoll(diceNum, diceSize)
			}

			ch.SendMessage("You hear a roar of pain!\r\n")

			if target.IsNPC() {
				w.roomMessage(exit.ToRoom, fmt.Sprintf("Some kind of %s streaks in from %s and strikes %s!", projectileName, from, target.GetName()))
				target.SendMessage(fmt.Sprintf("Suddenly some kind of %s pierces your arm!\r\n", projectileName))
				target.TakeDamage(dam)
				w.updatePosFromHP(target)
				if target.GetPosition() <= combat.PosDead {
					// die
				}
				target.SendMessage("You decide to go investigate...\r\n")
			}
			} else {
			// Miss
			target.SendMessage(fmt.Sprintf("Some kind of %s streaks in from %s and just misses you!\r\n", projectileName, from))
			w.roomMessage(exit.ToRoom, fmt.Sprintf("Some kind of %s streaks in from %s and narrowly misses %s!", projectileName, from, target.GetName()))
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// do_retreat — retreat from combat (line 1001)
// ---------------------------------------------------------------------------

func (w *World) doRetreat(ch *Player, me *MobInstance, cmd string, arg string) bool {
	var capmsg, lowmsg string

	if ch.GetClass() == ClassNinja {
		capmsg = "Escape"
		lowmsg = "escape"
	} else {
		capmsg = "Retreat"
		lowmsg = "retreat"
	}

	if ch.GetPosition() < combat.PosFighting {
		ch.SendMessage("Get on your feet first!\r\n")
		return true
	}

	if ch.GetSkill(SkillEscape) == 0 && ch.GetSkill(SkillRetreat) == 0 {
		ch.SendMessage("Huh?\r\n")
		return true
	}

	if !ch.IsFighting() {
		ch.SendMessage(fmt.Sprintf("%s from what? You aren't fighting!\r\n", capmsg))
		return true
	}

	percent := randRange(1, 101)
	var prob int
	if ch.GetClass() == ClassNinja {
		prob = ch.GetSkill(SkillEscape)
	} else {
		prob = ch.GetSkill(SkillRetreat)
	}

	if percent > prob {
		ch.SendMessage(fmt.Sprintf("You try to %s but get cornered in the process!\r\n", lowmsg))
		improveSkill(ch, SkillEscape)
		return true
	}

	_ = arg
	dirs := []string{"north", "east", "south", "west", "up", "down"}
	for i := 0; i < 6; i++ {
		attempt := randRange(0, len(dirs)-1)
		room := w.GetRoomInWorld(ch.RoomVNum)
		if room == nil {
			continue
		}
		exit, hasExit := room.Exits[dirs[attempt]]
		if !hasExit || exit.ToRoom <= 0 {
			continue
		}
		if w.roomHasFlag(exit.ToRoom, "death") {
			continue
		}
		if exit.DoorState == 1 {
			continue
		}

		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s realizes it's a losing cause and gracefully attempts to %s.", ch.Name, lowmsg))
		if doSimpleMove(w, ch, attempt, true) {
			ch.SendMessage(fmt.Sprintf("You make a hasty %s.\r\n", lowmsg))
		} else {
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s is cornered and fails to %s!", ch.Name, lowmsg))
		}
		return true
	}

	ch.SendMessage(fmt.Sprintf("You are cornered and fail to %s!\r\n", lowmsg))
	w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s is cornered and fails to %s!", ch.Name, lowmsg))
	return true
}

// ---------------------------------------------------------------------------
// do_subdue — subdue opponent (line 1084)
// ---------------------------------------------------------------------------

func (w *World) doSubdue(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillSubdue) == 0 {
		ch.SendMessage("You have no idea how!\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You're too busy right now!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Subdue who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		// Check mobs
		for _, m := range w.GetMobsInRoom(ch.RoomVNum) {
			if strings.Contains(strings.ToLower(m.GetName()), strings.ToLower(args[0])) {
				ch.SendMessage("They aren't here.\r\n")
				return true
			}
		}
		ch.SendMessage("Subdue who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Aren't we funny today...\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You can't contemplate violence in such a place!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("You can't get close enough!\r\n")
		return true
	}

	if !vict.IsNPC() && !isOutlaw(ch) {
		ch.SendMessage("You can't subdue them because you are not an Outlaw!\r\n")
		vict.SendMessage(fmt.Sprintf("%s failed to subdue you because %s is not an Outlaw.\r\n", ch.Name, ch.Name))
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Haha.. Don't think so.\r\n")
		return true
	}

	if vict.GetPosition() <= combat.PosStunned {
		ch.SendMessage("What's the point of doing that now?\r\n")
		return true
	}

	percent := randRange(1, 101+vict.GetLevel())
	prob := ch.GetSkill(SkillSubdue)

	if ch.GetLevel() >= lvlImmort {
		percent = 0
	}
	if vict.IsNPC() && false {
		prob = 0
	}
	if !vict.IsNPC() && ch.GetLevel() < lvlImmort {
		if vict.GetLevel() > ch.GetLevel()+3 || vict.GetLevel() < ch.GetLevel()-3 {
			prob = 0
		}
	}

	levelDiff := vict.GetLevel() - ch.GetLevel()
	if levelDiff > 0 {
		percent += levelDiff
	}

	if percent > prob {
		// Failed
		if vict.IsNPC() && false {
			vict.SendMessage(fmt.Sprintf("%s misses a blow to the back of your head.\r\n", ch.Name))
			ch.SendMessage(fmt.Sprintf("%s avoids your misplaced blow.\r\n", vict.GetName()))
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s avoids %s's misplaced blow.", vict.GetName(), ch.Name))
		}
		w.startCombatBetween(vict, ch)
	} else {
		// Success
		vict.SendMessage("Someone sneaks up behind you and knocks you out!\r\n")
		ch.SendMessage(fmt.Sprintf("You knock %s out cold.\r\n", himHer(vict.GetSex())))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s knocks out %s with a well-placed blow to the back of the head.", ch.Name, vict.Name))
		vict.SetPosition(combat.PosStunned)
		improveSkill(ch, SkillSubdue)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_sleeper — sleeper hold/choke (line 1184)
// ---------------------------------------------------------------------------

func (w *World) doSleeper(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillSleeper) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You can't do this while fighting!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("This room just has such a peaceful, easy feeling...\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded != nil {
		ch.SendMessage("You can't get a good grip on them while you are holding that weapon!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Sleeper who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("Sleeper who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Can't get to sleep fast enough, huh?\r\n")
		return true
	}

	if !vict.IsNPC() && !isOutlaw(ch) {
		ch.SendMessage(fmt.Sprintf("You cannot sleeper them because you are not an Outlaw!\r\n"))
		vict.SendMessage(fmt.Sprintf("%s failed to sleeper you because %s is not an Outlaw.\r\n", ch.Name, ch.Name))
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("You can't get a good grip on them while they're fighting!\r\n")
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Ha Ha. Don't think so.\r\n")
		return true
	}

	if vict.GetPosition() <= combat.PosSleeping {
		ch.SendMessage("What's the point of doing that now?\r\n")
		return true
	}

	percent := randRange(1, 101+vict.GetLevel())
	prob := ch.GetSkill(SkillSleeper)

	if vict.IsNPC() && (false) {
		prob = 0
	}
	if !vict.IsNPC() && ch.GetLevel() < lvlImmort {
		if vict.GetLevel() > ch.GetLevel()+3 || vict.GetLevel() < ch.GetLevel()-3 {
			prob = 0
		}
	}

	levelDiff := vict.GetLevel() - ch.GetLevel()
	if levelDiff > 0 {
		percent += levelDiff
	}

	if percent > prob {
		// Failed
		ch.SendMessage(fmt.Sprintf("You try to grab %s in a sleeper hold but fail!", vict.GetName()))
		vict.SendMessage(fmt.Sprintf("%s tries to put a sleeper hold on you, but you break free!", ch.Name))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s tries to put %s in a sleeper hold...", ch.Name, vict.Name))
		w.startCombatBetween(vict, ch)
	} else {
		// Success
		ch.SendMessage(fmt.Sprintf("You put %s in a sleeper hold.", vict.GetName()))
		vict.SendMessage("You feel very sleepy... Zzzzz..")
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s puts %s in a sleeper hold.", ch.Name, vict.Name))
		vict.SetPosition(combat.PosSleeping)

		// Add AFF_SLEEP affect
		duration := ch.GetLevel() / 9
		if duration < 1 {
			duration = 1
		}
		vict.SetAffect(affSleep, true)

		improveSkill(ch, SkillSleeper)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_neckbreak — neck break (line 1295)
// ---------------------------------------------------------------------------

func (w *World) doNeckbreak(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillNeckbreak) == 0 {
		ch.SendMessage("What's that, idiot-san?\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded != nil {
		ch.SendMessage("You can't do this and wield a weapon at the same time!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("I don't see them here.\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("I don't see them here.\r\n")
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Haha.. Don't think so.\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Aren't we funny today...\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You can't contemplate violence in such a place!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	neededMoves := 51
	if ch.GetMove() < neededMoves {
		ch.SendMessage("You haven't the energy to do this!\r\n")
		return true
	}
	ch.SetMove(ch.GetMove() - neededMoves)

	// Calculate percentage based on AC
	// In C: ((7 - (GET_AC(vict) / 10)) << 1) + number(1, 101)
	ac := vict.GetAC()
	percent := ((7 - (ac / 10)) << 1) + randRange(1, 101)
	prob := ch.GetSkill(SkillNeckbreak)

	if percent > prob {
		// Failed
		ch.SendMessage(fmt.Sprintf("You try to break %s's neck, but %s is too strong!", himHer(vict.GetSex()), vict.GetName()))
		vict.SendMessage(fmt.Sprintf("%s tries to break your neck, but can't!", ch.Name))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s tries to break %s's neck, but %s slips free!", ch.Name, vict.Name, vict.Name))
		w.startCombatBetween(vict, ch)
	} else {
		// Success - damage based on 18d(level)
		dam := diceRoll(18, ch.GetLevel())
		w.doDamage(ch, vict, dam, SkillNeckbreak)
		improveSkill(ch, SkillNeckbreak)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_ambush — ambush (line 1454)
// ---------------------------------------------------------------------------

func (w *World) doAmbush(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillAmbush) == 0 {
		ch.SendMessage("You'd better not.\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Ambush who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("Ambush who?\r\n")
		return true
	}

	if ch.IsAffected(affCharm) {
		ch.SendMessage("You are a little busy for that right now!\r\n")
		return true
	}

	if ch == vict {
		ch.SendMessage("Ambush yourself? You idiot!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.RoomVNum); sector := 0; if room != nil { sector = room.Sector }
	if sector != 3 && sector != 4 &&
		sector != 5 && sector != 1 {
		ch.SendMessage("Ambush someone here? Impossible!\r\n")
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("They're too alert for that, currently.\r\n")
		return true
	}

	ch.SendMessage("You crouch in the shadows and plan your ambush...\r\n")

	// In the original C code, this creates an event callback that fires
	// after PULSE_VIOLENCE*2. For now, we execute the ambush immediately
	// with the success/failure check inline.

	percent := randRange(1, 131)
	prob := ch.GetSkill(SkillAmbush)

	if vict.IsNPC() && false {
		percent = 200
	}

	if percent > prob {
		// Failure — do 0 damage to trigger awareness
		w.doDamage(ch, vict, 0, SkillAmbush)
	} else {
		// Success
		dam := ch.GetDamroll()

		wielded := w.GetEquipped(ch, eqWearWield)
		if wielded != nil {
			diceNum := wielded.Prototype.Values[0]
			diceSize := wielded.Prototype.Values[1]
			dam += diceRoll(diceNum, diceSize)
		}

		// Bonus damage: level * 2.6
		dam += int(float64(ch.GetLevel()) * 2.6)

		// 10% more damage if hidden
		if ch.IsAffected(affHide) {
			dam += int(float64(dam) * 0.10)
		}

		w.doDamage(ch, vict, dam, SkillAmbush)
		improveSkill(ch, SkillAmbush)
	}
	return true
}

// ---------------------------------------------------------------------------
// startCombatBetween — initiate combat between two characters (C: set_fighting)
// ---------------------------------------------------------------------------

// startCombatBetween initiates combat between ch and vict.
// If vict is nil, it just sets ch as fighting (for aggressive mobs attacking players).
func (w *World) startCombatBetween(ch, vict interface{}) {
	// Try to handle *Player vs *Player or *Player vs *MobInstance
	if chPlayer, ok := ch.(*Player); ok {
		if victPlayer, ok2 := vict.(*Player); ok2 {
			chPlayer.SetFighting(victPlayer.Name)
			victPlayer.SetFighting(chPlayer.Name)
			// Broadcast the fight start
			w.roomMessage(chPlayer.RoomVNum, fmt.Sprintf("%s starts fighting %s!\r\n", chPlayer.Name, victPlayer.Name))
		} else if victMob, ok2 := vict.(*MobInstance); ok2 {
			chPlayer.SetFighting(victMob.GetName())
			w.roomMessage(chPlayer.RoomVNum, fmt.Sprintf("%s starts fighting %s!\r\n", chPlayer.Name, victMob.GetName()))
		} else {
			// Only one party — probably mob attacking player
			chPlayer.SetFighting("someone")
		}
	} else if chMob, ok := ch.(*MobInstance); ok {
		if victPlayer, ok2 := vict.(*Player); ok2 {
			chMob.SetFighting(victPlayer.Name)
			victPlayer.SetFighting(chMob.GetName())
		}
	}
}
