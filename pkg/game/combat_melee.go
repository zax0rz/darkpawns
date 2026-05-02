//nolint:unused // Game logic port — not yet wired to command registry.
package game

// combat_melee.go — melee combat actions: bash, rescue, kick, dragon_kick, tiger_punch
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
