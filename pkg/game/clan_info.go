package game

import (
	"fmt"
)

func (w *World) doClanStatus(ch *Player) {
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("You are immortal and cannot join any clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)

	if ch.ClanRank == 0 {
		if c != nil {
			ch.SendMessage(fmt.Sprintf("You applied to %s\r\n", c.Name))
			return
		}
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	rankName := ""
	if c != nil && ch.ClanRank-1 >= 0 && ch.ClanRank-1 < len(c.RankName) {
		rankName = c.RankName[ch.ClanRank-1]
	}
	if c != nil {
		ch.SendMessage(fmt.Sprintf("You are %s (Rank %d) of %s\r\n",
			rankName, ch.ClanRank, c.Name))
	} else {
		ch.SendMessage(fmt.Sprintf("You are Rank %d of a clan", ch.ClanRank))
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_apply
// ---------------------------------------------------------------------------

func (w *World) doClanApply(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("Gods cannot apply for any clan.\r\n")
		return
	}
	if ch.ClanRank > 0 {
		ch.SendMessage("You already belong to a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	if ch.Level < c.ApplLevel {
		ch.SendMessage("You are not mighty enough to apply to this clan.\r\n")
		return
	}
	ch.mu.Lock()
	if ch.Gold < c.AppFee {
		ch.mu.Unlock()
		ch.SendMessage("You cannot afford the application fee!\r\n")
		return
	}

	ch.Gold -= c.AppFee
	ch.mu.Unlock()
	c.Treasure += int64(c.AppFee)
	w.SaveClans()

	ch.ClanID = c.ID
	ch.SendMessage("You've applied to the clan!\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_info
// ---------------------------------------------------------------------------

func (w *World) doClanInfo(ch *Player, arg string) {
	if w.Clans.ClanCount() == 0 {
		ch.SendMessage("No clans have formed yet.\r\n")
		return
	}

	if arg == "" {
		// Show all clans
		msg := "\r\n\t\tooO Clans of Dark Pawns Ooo\r\n"
		for i := 0; i < w.Clans.ClanCount(); i++ {
			c := w.Clans.GetClanByIndex(i)
			if c == nil {
				continue
			}
			if ch.Level >= LVL_IMMORT {
				msg += fmt.Sprintf("[%-2d]  %-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.ID, c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			} else if c.Private == 0 {
				msg += fmt.Sprintf("%-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			}
		}
		ch.SendMessage(msg)
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	msg := fmt.Sprintf("Info for the clan %s :\r\n", c.Name)
	msg += fmt.Sprintf("Ranks      : %d\r\nTitles     : ", c.Ranks)
	for j := 0; j < c.Ranks && j < len(c.RankName); j++ {
		msg += c.RankName[j] + " "
	}
	msg += fmt.Sprintf("\r\nMembers    : %d\r\nPower      : %d\r\nTreasure   : %d\r\nSpells     : ", c.Members, c.Power, c.Treasure)
	for j := 0; j < 5; j++ {
		if c.Spells[j] != 0 {
			msg += fmt.Sprintf("%d ", c.Spells[j])
		}
	}
	msg += "\r\n"
	msg += "Clan privileges:\r\n"
	for j := 0; j < NumCP; j++ {
		msg += fmt.Sprintf("   %-10s: %d\r\n", clanPrivileges[j], c.Privilege[j])
	}
	msg += "\r\n"
	msg += fmt.Sprintf("Description:\r\n%s\r\n\r\n", c.Plan)

	atWar := false
	for j := 0; j < 4; j++ {
		if c.AtWar[j] != 0 {
			atWar = true
			break
		}
	}
	if !atWar {
		msg += "This clan is at peace with all others.\r\n"
	} else {
		msg += "This clan is at war.\r\n"
	}
	ch.SendMessage(msg)
}

// ExecClanCommand dispatches the "clan" player command.
// In C: ACMD(do_clan)
