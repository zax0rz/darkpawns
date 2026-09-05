package game

import (
	"fmt"
)

func (w *World) doClanStatus(ch *Player) {
	if ch.GetLevel() >= LVL_IMMORT {
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
	if ch.GetLevel() >= LVL_IMMORT {
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

	if ch.GetLevel() < c.ApplLevel {
		ch.SendMessage("You are not mighty enough to apply to this clan.\r\n")
		return
	}
	if ch.GetGold() < c.AppFee {
		ch.SendMessage("You cannot afford the application fee!\r\n")
		return
	}

	ch.SetGold(ch.GetGold() - c.AppFee)
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
		msg := "\r"
		visible := false
		for i := 0; i < w.Clans.ClanCount(); i++ {
			c := w.Clans.GetClanByIndex(i)
			if c == nil {
				continue
			}
			if ch.GetLevel() >= LVL_IMMORT {
				msg += fmt.Sprintf("[%-2d]  %-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.ID, c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			} else if c.Private == 0 {
				visible = true
				msg += fmt.Sprintf("%-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			}
		}
		if ch.GetLevel() < LVL_IMMORT && !visible {
			msg = "\r\t\t\tooO Clans of Dark Pawns Ooo\r\n"
		}
		ch.SendMessage(msg)
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	msg := fmt.Sprintf("Info for the clan %s :\r\n\r\n\r\nDescription:\r\n", c.Name)
	if c.Plan == "" {
		msg += "(null)"
	} else {
		msg += c.Plan
	}
	msg += "\r\n\r\n"

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
	msg += fmt.Sprintf("Application fee  : %d gold\r\nMonthly Dues     : %d gold\r\n", c.AppFee, c.Dues)
	msg += fmt.Sprintf("Application level: %d\r\n", c.ApplLevel)
	ch.SendMessage(msg)
}

// ExecClanCommand dispatches the "clan" player command.
// In C: ACMD(do_clan)
