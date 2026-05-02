//nolint:unused // Game logic port — not yet wired to command registry.
package game

// combat_control.go — combat control: order, flee
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
