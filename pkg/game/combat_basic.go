package game

// combat_basic.go — basic combat actions: assist, hit, kill, backstab, disembowel
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
