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
		msg := fmt.Sprintf("%s is no longer AFK.\r\n", ch.Name)
		actToRoom(w, ch.GetRoomVNum(), msg, ch.Name)
		ch.SendMessage("You are no longer AFK.\r\n")
	} else {
		ch.SetPlrFlag(PrfAFK, true)
		ch.SetAFK(true)
		ch.SetAFKMessage(arg)
		msg := fmt.Sprintf("%s is now AFK.\r\n", ch.Name)
		actToRoom(w, ch.GetRoomVNum(), msg, ch.Name)
		if arg != "" {
			ch.SendMessage("You are now AFK: " + arg + "\r\n")
		} else {
			ch.SendMessage("You are now AFK.\r\n")
		}
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

	arg = strings.TrimSpace(arg)

	if arg == "" {
		var autos []string
		if ch.GetFlags()&(1<<PrfAutoexit) != 0 {
			autos = append(autos, "exits")
		}
		if ch.GetFlags()&(1<<PrfAutoLoot) != 0 {
			autos = append(autos, "loot")
		}
		if ch.GetFlags()&(1<<PrfAutoGold) != 0 {
			autos = append(autos, "gold")
		}
		if ch.GetFlags()&(1<<PrfAutoSplit) != 0 {
			autos = append(autos, "split")
		}

		if len(autos) == 0 {
			ch.SendMessage("None.\r\n")
		} else {
			ch.SendMessage("Autos: " + strings.Join(autos, ", ") + "\r\n")
		}
		return true
	}

	switch strings.ToLower(arg) {
	case "exit", "exits":
		if ch.GetFlags()&(1<<PrfAutoexit) != 0 {
			ch.SetPlrFlag(PrfAutoexit, false)
			ch.SendMessage("Auto exits off.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoexit, true)
			ch.SendMessage("Auto exits on.\r\n")
		}
	case "loot":
		if ch.GetFlags()&(1<<PrfAutoLoot) != 0 {
			ch.SetPlrFlag(PrfAutoLoot, false)
			ch.SendMessage("Auto loot off.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoLoot, true)
			ch.SendMessage("Auto loot on.\r\n")
		}
	case "gold":
		if ch.GetFlags()&(1<<PrfAutoGold) != 0 {
			ch.SetPlrFlag(PrfAutoGold, false)
			ch.SendMessage("Auto gold off.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoGold, true)
			ch.SendMessage("Auto gold on.\r\n")
		}
	case "split":
		if ch.GetFlags()&(1<<PrfAutoSplit) != 0 {
			ch.SetPlrFlag(PrfAutoSplit, false)
			ch.SendMessage("Auto split off.\r\n")
		} else {
			ch.SetPlrFlag(PrfAutoSplit, true)
			ch.SendMessage("Auto split on.\r\n")
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
