//nolint:unused // Game logic port — not yet wired to command registry.
package game

// combat_ranged.go — ranged combat: shoot
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
				if target.GetHP() <= 0 {
					// Source: act.offensive.c do_shoot() — die(to) on POS_DEAD
					// Uses die() not die_with_killer() — 1/3 EXP loss, no CON loss
					w.HandleNonCombatDeath(target)
				} else {
					// Source: act.offensive.c do_shoot() — mob bursts into
					// shooter's room and attacks: hit(to, ch, TYPE_UNDEFINED)
					target.SendMessage("You decide to go investigate...\r\n")
					if target.IsNPC() {
						// Source: act.offensive.c — mob bursts into shooter's room and attacks
						w.roomMessage(exit.ToRoom, fmt.Sprintf("%s bursts into the room and scowls at %s!\r\n", target.Name, ch.Name))
						ch.SendMessage(fmt.Sprintf("%s bursts into the room and scowls at you!\r\n", target.Name))
						// Move mob to shooter's room (C: char_from_room + char_to_room)
						target.RoomVNum = ch.RoomVNum
						// Mob attacks the shooter (C: hit(to, ch, TYPE_UNDEFINED))
						w.startCombatBetween(target, ch)
					}
				}
			}
			} else {
			// Miss
			target.SendMessage(fmt.Sprintf("Some kind of %s streaks in from %s and just misses you!\r\n", projectileName, from))
			w.roomMessage(exit.ToRoom, fmt.Sprintf("Some kind of %s streaks in from %s and narrowly misses %s!", projectileName, from, target.GetName()))
		}
	}
	return true
}
