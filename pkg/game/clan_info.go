package game

import (
	"fmt"
	"strings"
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
		var msg strings.Builder
		msg.WriteString("\r")
		visible := false
		for i := 0; i < w.Clans.ClanCount(); i++ {
			c := w.Clans.GetClanByIndex(i)
			if c == nil {
				continue
			}
			if ch.GetLevel() >= LVL_IMMORT {
				fmt.Fprintf(&msg, "[%-2d]  %-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.ID, c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			} else if c.Private == 0 {
				visible = true
				fmt.Fprintf(&msg, "%-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			}
		}
		if ch.GetLevel() < LVL_IMMORT && !visible {
			msg.Reset()
			msg.WriteString("\r\t\t\tooO Clans of Dark Pawns Ooo\r\n")
		}
		ch.SendMessage(msg.String())
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "Info for the clan %s :\r\n\r\n\r\nDescription:\r\n", c.Name)
	if c.Plan == "" {
		msg.WriteString("(null)")
	} else {
		msg.WriteString(c.Plan)
	}
	msg.WriteString("\r\n\r\n")

	atWar := false
	for j := 0; j < 4; j++ {
		if c.AtWar[j] != 0 {
			atWar = true
			break
		}
	}
	if !atWar {
		msg.WriteString("This clan is at peace with all others.\r\n")
	} else {
		msg.WriteString("This clan is at war.\r\n")
	}
	fmt.Fprintf(&msg, "Application fee  : %d gold\r\nMonthly Dues     : %d gold\r\n", c.AppFee, c.Dues)
	fmt.Fprintf(&msg, "Application level: %d\r\n", c.ApplLevel)
	ch.SendMessage(msg.String())
}

// ExecClanCommand dispatches the "clan" player command.
// In C: ACMD(do_clan)
