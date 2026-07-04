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
)

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
	for _, p := range w.GetPlayersInRoom(ch.GetRoom()) {
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

	w.roomMessageExcludeTwo(ch.GetRoom(),
		fmt.Sprintf("%s rescues %s!", ch.Name, vict.GetName()),
		ch.Name, vict.GetName())

	improveSkill(ch, SkillRescue)
	return true
}

// ---------------------------------------------------------------------------
