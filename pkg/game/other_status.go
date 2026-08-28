package game

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// do_afk — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAFK(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.GetFlags()&(1<<PrfAFK) != 0 {
		ch.SetPlrFlag(PrfAFK, false)
		ch.SetAFK(false)
		ch.SetAFKMessage("")
		Act(w, false, ch, nil, nil, nil, "$n returns from some repulsive act...", "", ToRoom)
		ch.SendMessage("You return from the world of the living.\r\n")
	} else {
		ch.SetPlrFlag(PrfAFK, true)
		ch.SetAFK(true)
		ch.SetAFKMessage("")
		Act(w, false, ch, nil, nil, nil, "$n goes AFK...", "", ToRoom)
		// C's command/prompt cycle leaves a blank line before the AFK prompt.
		ch.SendMessage("Go leave..no one will notice anyways.\r\n\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_auto — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAuto(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// C only skips leading spaces here. The command table has already removed
	// the command word, but do_auto compares the remaining argument literally:
	// case and any trailing text are significant.
	arg = strings.TrimLeft(arg, " \t")

	if arg == "" {
		var result strings.Builder
		result.WriteString("You have the following autos set:\r\n")
		if ch.GetAutoExit() {
			result.WriteString("Exits ")
		}
		if ch.GetFlags()&(1<<PrfAutoLoot) != 0 {
			result.WriteString("Loot ")
		}
		if ch.GetFlags()&(1<<PrfAutoGold) != 0 {
			result.WriteString("Gold ")
		}
		if ch.GetFlags()&(1<<PrfAutoSplit) != 0 {
			result.WriteString("Split")
		}
		if result.Len() == len("You have the following autos set:\r\n") {
			result.WriteString("None.")
		}
		result.WriteString("\r\n")
		ch.SendMessage(result.String())
		return true
	}

	switch arg {
	case "exit", "exits":
		if ch.GetAutoExit() {
			ch.SetAutoExit(false)
			ch.SendMessage("You will no longer see room exits.\r\n")
		} else {
			ch.SetAutoExit(true)
			ch.SendMessage("You will now see room exits.\r\n")
		}
	case "loot":
		if ch.GetFlags()&(1<<PrfAutoLoot) != 0 {
			ch.SetPlrFlag(PrfAutoLoot, false)
			ch.SendMessage("You will no longer loot corpses.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoLoot, true)
			ch.SendMessage("You will now automatically loot corpses.\r\n")
		}
	case "gold":
		if ch.GetFlags()&(1<<PrfAutoGold) != 0 {
			ch.SetPlrFlag(PrfAutoGold, false)
			ch.SendMessage("You will no longer get the gold from corpses.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoGold, true)
			ch.SendMessage("You will now get the gold from corpses.\r\n")
		}
	case "split":
		if ch.GetFlags()&(1<<PrfAutoSplit) != 0 {
			ch.SetPlrFlag(PrfAutoSplit, false)
			ch.SendMessage("You will no longer split gold with your group.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoSplit, true)
			ch.SendMessage("You will now split gold with your group.\r\n")
		}
	default:
		ch.SendMessage("What do you want to make automatic?\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_transform — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doTransform(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// C do_transform (act.other.c): only werewolves/vampires can transform;
	// everyone else is rejected up front. This guard was missing from the port.
	if ch.GetFlags()&(1<<PlrWerewolf) == 0 && ch.GetFlags()&(1<<PlrVampire) == 0 {
		ch.SendMessage("You aren't transformable!\n\r")
		return true
	}

	if ch.GetFlags()&(1<<PlrWerewolf) != 0 {
		// Werewolf: toggle affWerewolf
		if ch.IsAffected(affWerewolf) {
			ch.SetAffect(affWerewolf, false)
			// Restore pre-transform MaxHP to prevent the HP exploit
			if ch.WolfBaseMaxHP > 0 {
				ch.SetMaxHP(ch.WolfBaseMaxHP)
				ch.WolfBaseMaxHP = 0
			}
			if ch.GetHP() > ch.GetMaxHP() {
				ch.SetHP(ch.GetMaxHP())
			}
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			// Must be night and near full moon to transform
			sun := GetSunlight()
			if sun == SunLight || sun == SunRise {
				ch.SendMessage("You cannot transform in the light of day!\r\n")
				return true
			}
			moon := GetMoon()
			if moon != MoonFull && moon != MoonThreeFull {
				ch.SendMessage("The moon is not full enough for you to transform!\r\n")
				return true
			}
			ch.WolfBaseMaxHP = ch.GetMaxHP()
			ch.SetAffect(affWerewolf, true)
			bonus := randRange(2, 6) * 10
			ch.SetHP(ch.GetHP() + bonus)
			if ch.GetHP() > 666 {
				ch.SetHP(666)
			}
			if ch.GetHP() > ch.GetMaxHP() {
				ch.SetMaxHP(ch.GetHP())
			}
			ch.SendMessage("You transform into a werewolf!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a werewolf!\r\n", ch.Name), ch.Name)
		}
	} else if ch.GetFlags()&(1<<PlrVampire) != 0 {
		// Vampire: toggle affVampire
		if ch.IsAffected(affVampire) {
			ch.SetAffect(affVampire, false)
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			ch.SetAffect(affVampire, true)
			bonus := randRange(2, 6) * 10
			ch.SetMana(ch.GetMana() + bonus)
			ch.SendMessage("You transform into a vampire!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a vampire!\r\n", ch.Name), ch.Name)
		}
	} else {
		ch.SendMessage("You have no idea how to transform!\r\n")
	}
	return true
}
