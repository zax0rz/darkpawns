package game

import (
	"strconv"
)

func (w *World) doClanBank(ch *Player, arg string, action int) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	var c *Clan
	var immcom bool
	if ch.GetLevel() < LVL_IMMORT {
		_, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil || ch.ClanRank == 0 {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.GetLevel() < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		amountArg, clanArg := halfChop(arg)
		arg = amountArg
		_, c = w.Clans.FindClan(clanArg)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if action == CBWithdraw && !immcom && !c.CanWithdraw(ch) {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		switch action {
		case CBDeposit:
			ch.SendMessage("Deposit how much?\r\n")
		case CBWithdraw:
			ch.SendMessage("Withdraw how much?\r\n")
		default:
			ch.SendMessage("Bad clan banking call, please report to a God.\r\n")
		}
		return
	}

	if !isClanNumber(arg) {
		switch action {
		case CBDeposit:
			ch.SendMessage("Deposit what?\r\n")
		case CBWithdraw:
			ch.SendMessage("Withdraw what?\r\n")
		default:
			ch.SendMessage("Bad clan banking call, please report to a God.\r\n")
		}
		return
	}

	amount, _ := strconv.Atoi(arg)

	switch action {
	case CBWithdraw:
		if c.Treasure < int64(amount) {
			ch.SendMessage("The clan is not wealthy enough for your needs!\r\n")
			return
		}
		ch.SetGold(ch.GetGold() + amount)
		c.Treasure -= int64(amount)
		ch.SendMessage("You withdraw from the clan's treasure.\r\n")
	case CBDeposit:
		if !immcom && ch.GetGold() < amount {
			ch.SendMessage("You do not have that kind of money!\r\n")
			return
		}
		if !immcom {
			ch.SetGold(ch.GetGold() - amount)
		}
		c.Treasure += int64(amount)
		ch.SendMessage("You add to the clan's treasure.\r\n")
	}

	w.SaveClans()
}

// doClanPrivate toggles a clan between public and private room access.
// In C: do_clan_private()
