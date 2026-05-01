package game

import (
	"fmt"
	"strconv"
)

func (w *World) doClanPrivate(ch *Player, arg string) {
	var clanNum int
	var c *Clan

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
		if ch.ClanRank != c.Ranks {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		clanNum, c = w.Clans.FindClan(arg)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	_ = clanNum

	if c.Private == ClanPublic {
		c.Private = ClanPrivate
		ch.SendMessage("Your clan is now private.\r\n")
		w.SaveClans()
		return
	}

	if c.Private == ClanPrivate {
		c.Private = ClanPublic
		ch.SendMessage("Your clan is now public.\r\n")
		w.SaveClans()
		return
	}
}

// doClanPlan shows/edits the clan's plan (description).
// In C: do_clan_plan() — uses string_write (descriptor-based editor).
//
// string_write is a descriptor-based multi-line editor activated by setting
// ch.WriteMagic and storing a callback. The session layer intercepts all
// subsequent input until a line containing only "@" is received, then calls
// the callback with the accumulated text.
//
// For now the plan is set via simple single-argument input from the god
// command, or cleared in anticipation of session-layer editor support.
// When the session editor layer is wired (string_write in session/), set
// ch.WriteMagic = magicToken and register a callback that writes the
// accumulated text into c.Plan.
func (w *World) doClanPlan(ch *Player, arg string) {
	var clanNum int
	var c *Clan

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
		if ch.ClanRank < c.Privilege[CPSetPlan] {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		if arg == "" {
			w.sendClanFormat(ch)
			return
		}
		clanNum, c = w.Clans.FindClan(arg)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	_ = clanNum

	if c.Plan != "" {
		ch.SendMessage(fmt.Sprintf("Old plan for clan <<%s>>:\r\n%s\r\n", c.Name, c.Plan))
	}
	ch.SendMessage(fmt.Sprintf("Enter the description, or plan for clan <<%s>>.\r\n", c.Name))
	ch.SendMessage("End with @ on a line by itself.\r\n")
	c.Plan = ""
	// C uses string_write(ch->desc, &clan[clan_num].plan, CLAN_PLAN_LENGTH, 0, NULL)
	// The session layer should check ch.WriteMagic after each input line
	// and accumulate text until "@" alone appears, then call the registered
	// callback. Once wired, this function would set:
	//   ch.WriteMagic = someToken
	// and register a callback that assigns c.Plan = accumulatedText and
	// calls w.SaveClans().
	w.SaveClans()
}

// doClanRanks manages clan rank names and adjusts existing members' ranks.
// In C: do_clan_ranks()
func (w *World) doClanRanks(ch *Player, arg string) {
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
		a1, _ := halfChop(arg)
		arg = a1
		clanNum, c = w.Clans.FindClan(a1)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank != c.Ranks && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set how many ranks?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the ranks to what?\r\n")
		return
	}

	newRanks, _ := strconv.Atoi(arg)

	if newRanks == c.Ranks {
		ch.SendMessage("The clan already has this number of ranks.\r\n")
		return
	}

	if newRanks < 2 || newRanks > 20 {
		ch.SendMessage("Clans must have from 2 to 20 ranks.\r\n")
		return
	}

	if ch.Gold < 5000 && !immcom {
		ch.SendMessage("Changing the clan hierarchy requires 5,000 coins!\r\n")
		return
	}

	if !immcom {
		ch.Gold -= 5000
	}

	// Adjust existing clan members' ranks
	for _, p := range w.allPlayers() {
		if p.ClanID == c.ID {
			if p.ClanRank < c.Ranks && p.ClanRank > 0 {
				p.ClanRank = 1
			}
			if p.ClanRank == c.Ranks {
				p.ClanRank = newRanks
			}
		}
	}

	_ = clanNum

	c.Ranks = newRanks
	for i := 0; i < c.Ranks-1; i++ {
		c.RankName[i] = "Member"
	}
	c.RankName[c.Ranks-1] = "Leader"
	for i := 0; i < NumCP; i++ {
		c.Privilege[i] = newRanks
	}

	w.SaveClans()
}

// doClanTitles manages clan rank titles.
// In C: do_clan_titles()
func (w *World) doClanTitles(ch *Player, arg string) {
	var clanNum int
	var c *Clan

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
		if ch.ClanRank != c.Ranks {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		a1, a2 := halfChop(arg)
		arg = a2
		if !isNumber(a1) {
			ch.SendMessage("You need to specify a clan number.\r\n")
			return
		}
		idx, _ := strconv.Atoi(a1)
		c = w.Clans.GetClanByIndex(idx)
		if c == nil {
			ch.SendMessage("There is no clan with that number.\r\n")
			return
		}
		clanNum = idx
	}

	a1, a2 := halfChop(arg)

	if !isNumber(a1) {
		ch.SendMessage("You need to specify a rank number.\r\n")
		return
	}

	rank, _ := strconv.Atoi(a1)

	if rank < 1 || rank > c.Ranks {
		ch.SendMessage("This clan has no such rank number.\r\n")
		return
	}

	if len(a2) < 1 || len(a2) > 19 {
		ch.SendMessage("You need a clan title of under 20 characters.\r\n")
		return
	}

	_ = clanNum

	c.RankName[rank-1] = a2
	w.SaveClans()
	ch.SendMessage("Done.\r\n")
}

// doClanPrivilege manages clan privilege levels for ranks.
// In C: do_clan_privilege()
func (w *World) doClanPrivilege(ch *Player, arg string) {
	a1, a2 := halfChop(arg)

	if isAbbrev(a1, "setplan") {
		w.doClanSP(ch, a2, CPSetPlan)
		return
	}
	if isAbbrev(a1, "enroll") {
		w.doClanSP(ch, a2, CPEnroll)
		return
	}
	if isAbbrev(a1, "expel") {
		w.doClanSP(ch, a2, CPExpel)
		return
	}
	if isAbbrev(a1, "promote") {
		w.doClanSP(ch, a2, CPPromote)
		return
	}
	if isAbbrev(a1, "demote") {
		w.doClanSP(ch, a2, CPDemote)
		return
	}
	if isAbbrev(a1, "withdraw") {
		w.doClanSP(ch, a2, CPWithdraw)
		return
	}
	if isAbbrev(a1, "setfees") {
		w.doClanSP(ch, a2, CPSetFees)
		return
	}
	if isAbbrev(a1, "setapplev") {
		w.doClanSP(ch, a2, CPSetAppLev)
		return
	}

	ch.SendMessage("\r\nClan privileges:\r\n")
	for i := 0; i < NumCP; i++ {
		ch.SendMessage(fmt.Sprintf("\t%s\r\n", clanPrivileges[i]))
	}
}

// doClanSP manages a single clan privilege for a rank.
// In C: do_clan_sp()
func (w *World) doClanSP(ch *Player, arg string, priv int) {
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
		arg1, _ := halfChop(arg)
		arg = arg1
		// In C: uses arg1 (the clan name) for find_clan, same arg updated
		clanNum, c = w.Clans.FindClan(arg1)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank != c.Ranks && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set the privilege to which rank?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the privilege to what?\r\n")
		return
	}

	rank, _ := strconv.Atoi(arg)

	if rank < 1 || rank > c.Ranks {
		ch.SendMessage("There is no such rank in the clan.\r\n")
		return
	}

	_ = clanNum

	c.Privilege[priv] = rank
	w.SaveClans()
}

// doClanMoney manages clan dues and app fees.
// In C: do_clan_money()
