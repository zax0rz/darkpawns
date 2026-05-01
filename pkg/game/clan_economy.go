package game

import (
	"strconv"
)

func (w *World) doClanMoney(ch *Player, arg string, action int) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		arg1, arg2 := halfChop(arg)
		arg = arg1
		clanNum, c = w.Clans.FindClan(arg2)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank < c.Privilege[CPSetFees] && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set it to how much?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set it to what?\r\n")
		return
	}

	amount, _ := strconv.Atoi(arg)

	if amount < 0 || amount > 10000 {
		ch.SendMessage("Please pick a number between 0 and 10,000 coins.\r\n")
		return
	}

	_ = clanNum

	switch action {
	case CMAppFee:
		c.AppFee = amount
		ch.SendMessage("You change the application fee.\r\n")
	case CMDues:
		c.Dues = amount
		ch.SendMessage("You change the monthly dues.\r\n")
	default:
		ch.SendMessage("Problem in command, please report.\r\n")
	}

	w.SaveClans()
}

// doClanAppLevel manages clan application level requirements.
// In C: do_clan_application()
func (w *World) doClanAppLevel(ch *Player, arg string) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		arg1, arg2 := halfChop(arg)
		arg = arg1
		clanNum, c = w.Clans.FindClan(arg2)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank < c.Privilege[CPSetAppLev] && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set to which level?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the application level to what?\r\n")
		return
	}

	appLevel, _ := strconv.Atoi(arg)

	if appLevel < 1 || appLevel > 30 {
		ch.SendMessage("The application level can go from 1 to 30.\r\n")
		return
	}

	_ = clanNum

	c.ApplLevel = appLevel
	w.SaveClans()
}

// doClanSet dispatches the "clan set" subcommand.
// In C: do_clan_set()
func (w *World) doClanSet(ch *Player, arg string) {
	a1, a2 := halfChop(arg)

	if isAbbrev(a1, "plan") {
		w.doClanPlan(ch, a2)
		return
	}
	if isAbbrev(a1, "ranks") {
		w.doClanRanks(ch, a2)
		return
	}
	if isAbbrev(a1, "title") {
		w.doClanTitles(ch, a2)
		return
	}
	if isAbbrev(a1, "privilege") {
		w.doClanPrivilege(ch, a2)
		return
	}
	if isAbbrev(a1, "dues") {
		w.doClanMoney(ch, a2, 1)
		return
	}
	if isAbbrev(a1, "appfee") {
		w.doClanMoney(ch, a2, 2)
		return
	}
	if isAbbrev(a1, "applev") {
		w.doClanAppLevel(ch, a2)
		return
	}

	w.sendClanFormat(ch)
}
