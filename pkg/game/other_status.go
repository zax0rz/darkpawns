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

	if ch.Flags&(1<<PrfAFK) != 0 {
		ch.Flags &^= 1 << PrfAFK
		ch.AFK = false
		ch.AFKMessage = ""
		msg := fmt.Sprintf("%s is no longer AFK.\r\n", ch.Name)
		actToRoom(w, ch.GetRoomVNum(), msg, ch.Name)
		ch.SendMessage("You are no longer AFK.\r\n")
	} else {
		ch.Flags |= 1 << PrfAFK
		ch.AFK = true
		ch.AFKMessage = arg
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
		if ch.Flags&(1<<PrfAutoexit) != 0 {
			autos = append(autos, "exits")
		}
		if ch.Flags&(1<<PrfAutoLoot) != 0 {
			autos = append(autos, "loot")
		}
		if ch.Flags&(1<<PrfAutoGold) != 0 {
			autos = append(autos, "gold")
		}
		if ch.Flags&(1<<PrfAutoSplit) != 0 {
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
		if ch.Flags&(1<<PrfAutoexit) != 0 {
			ch.Flags &^= 1 << PrfAutoexit
			ch.SendMessage("Auto exits off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoexit
			ch.SendMessage("Auto exits on.\r\n")
		}
	case "loot":
		if ch.Flags&(1<<PrfAutoLoot) != 0 {
			ch.Flags &^= 1 << PrfAutoLoot
			ch.SendMessage("Auto loot off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoLoot
			ch.SendMessage("Auto loot on.\r\n")
		}
	case "gold":
		if ch.Flags&(1<<PrfAutoGold) != 0 {
			ch.Flags &^= 1 << PrfAutoGold
			ch.SendMessage("Auto gold off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoGold
			ch.SendMessage("Auto gold on.\r\n")
		}
	case "split":
		if ch.Flags&(1<<PrfAutoSplit) != 0 {
			ch.Flags &^= 1 << PrfAutoSplit
			ch.SendMessage("Auto split off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoSplit
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

	if ch.Flags&(1<<PlrWerewolf) != 0 {
		// Werewolf: toggle affWerewolf
		if ch.IsAffected(affWerewolf) {
			ch.SetAffect(affWerewolf, false)
			if ch.Health > ch.MaxHealth {
				ch.Health = ch.MaxHealth
			}
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			ch.SetAffect(affWerewolf, true)
			bonus := randRange(2, 6) * 10
			ch.Health += bonus
			if ch.Health > 666 {
				ch.Health = 666
			}
			if ch.Health > ch.MaxHealth {
				ch.MaxHealth = ch.Health
			}
			ch.SendMessage("You transform into a werewolf!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a werewolf!\r\n", ch.Name), ch.Name)
		}
	} else if ch.Flags&(1<<PlrVampire) != 0 {
		// Vampire: toggle affVampire
		if ch.IsAffected(affVampire) {
			ch.SetAffect(affVampire, false)
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			ch.SetAffect(affVampire, true)
			bonus := randRange(2, 6) * 10
			ch.Mana += bonus
			ch.SendMessage("You transform into a vampire!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a vampire!\r\n", ch.Name), ch.Name)
		}
	} else {
		ch.SendMessage("You have no idea how to transform!\r\n")
	}
	return true
}
