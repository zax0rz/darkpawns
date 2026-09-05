package game

import (
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

	climate := TimeWeatherSnapshot()
	night := climate.Weather.Sunlight == SunSet || climate.Weather.Sunlight == SunDark

	// C checks werewolf first. Every branch returns, so a player carrying both
	// PLR flags still follows only the werewolf state machine.
	if ch.GetFlags()&(1<<PlrWerewolf) != 0 {
		if night {
			if ch.IsAffected(affWerewolf) {
				ch.SendMessage("You can't change back until morning!\n\r")
				return true
			}
			if climate.Time.Day < 6 && climate.Time.Day >= 1 {
				ch.SendMessage("You can't transform when there's no moon in the sky!\r\n")
				return true
			}

			ch.SendMessage("Your nails grow into talons, and hair sprouts from every pore.\n\r")
			Act(w, false, ch, nil, nil, nil, "$n shivers and transforms into a werewolf!", "", ToRoom)
			ch.SetAffect(affWerewolf, true)
			bonus := werewolfTransformBonus(ch.GetMaxHP(), climate.Time.Moon)
			ch.SetHP(ch.GetHP() + bonus)
			if ch.GetHP() > 666 {
				ch.SetHP(666)
			}
			return true
		}

		if ch.IsAffected(affWerewolf) {
			ch.SendMessage("Your hair and nails shorten and you revert to your normal shape.\n\r")
			Act(w, false, ch, nil, nil, nil, "$n shivers and transforms out of werewolf form!", "", ToRoom)
			ch.SetAffect(affWerewolf, false)
			if ch.GetHP() > ch.GetMaxHP() {
				ch.SetHP(ch.GetMaxHP())
			}
			return true
		}
		ch.SendMessage("You can't transform during the day!\n\r")
		return true
	}

	if ch.GetFlags()&(1<<PlrVampire) != 0 {
		if night {
			if ch.IsAffected(affVampire) {
				ch.SendMessage("You can't change back until morning!\n\r")
				return true
			}

			ch.SendMessage("Your nails grow transluscent and fangs sprout from your incisors!\n\r")
			Act(w, false, ch, nil, nil, nil, "$n shivers and transforms into a vampire!", "", ToRoom)
			ch.SetAffect(affVampire, true)
			bonus := vampireTransformBonus(ch.GetMaxMana(), climate.Time.Moon)
			ch.SetMana(ch.GetMana() + bonus)
			return true
		}

		if ch.IsAffected(affVampire) {
			ch.SendMessage("Your fangs recess, and you revert to your normal shape.\n\r")
			Act(w, false, ch, nil, nil, nil, "$n shivers and transforms out of vampire form!", "", ToRoom)
			ch.SetAffect(affVampire, false)
			if ch.GetMana() > ch.GetMaxMana() {
				ch.SetMana(ch.GetMaxMana())
			}
			return true
		}
		ch.SendMessage("You can't transform during the day!\n\r")
	}
	return true
}

func werewolfTransformBonus(maxHP, moon int) int {
	switch moon {
	case MoonNew:
		return maxHP / 6
	case MoonQuarterFull, MoonThreeEmpty:
		return maxHP / 5
	case MoonHalfFull, MoonHalfEmpty:
		return maxHP / 4
	case MoonThreeFull, MoonQuarterEmpty:
		return maxHP / 3
	case MoonFull:
		return maxHP / 2
	default:
		return 0
	}
}

func vampireTransformBonus(maxMana, moon int) int {
	switch moon {
	case MoonNew:
		return maxMana / 5
	case MoonHalfFull, MoonHalfEmpty:
		return maxMana / 4
	case MoonThreeFull, MoonQuarterEmpty:
		return maxMana / 3
	case MoonFull:
		return maxMana / 2
	default:
		return 0
	}
}
