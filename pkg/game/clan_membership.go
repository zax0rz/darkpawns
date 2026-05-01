package game

import (
	"fmt"
)

func (w *World) doClanEnroll(ch *Player, arg string) {
	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	// If immortal, the arg already has the clan parsed out; for mortals argg is the arg
	var targetArg string
	if immcom {
		// arg was "player clan" — we already consumed clan via resolveClanForImmortal
		// Re-parse: first word is player, second is clan (already consumed)
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	// If no target, show applicant list
	if targetArg == "" {
		ch.SendMessage("The following players have applied to your clan:\r\n" +
			"-----------------------------------------------\r\n")
		for _, p := range w.players {
			if p.ClanID == c.ID && p.ClanRank == 0 {
				ch.SendMessage(fmt.Sprintf("%s\r\n", p.Name))
			}
		}
		return
	}

	if !immcom && ch.ClanRank < c.Privilege[CPEnroll] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}

	if victim.ClanID != c.ID {
		if victim.ClanRank > 0 {
			ch.SendMessage("They're already in a clan.\r\n")
			return
		}
		ch.SendMessage("They didn't request to join your clan.\r\n")
		return
	}

	if victim.ClanRank > 0 {
		ch.SendMessage("They're already in your clan.\r\n")
		return
	}
	if victim.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot enroll immortals in clans.\r\n")
		return
	}

	victim.ClanRank++
	c.Power += victim.Level
	c.Members++

	victim.SendMessage("You've been enrolled in the clan you chose!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_expel
// ---------------------------------------------------------------------------

func (w *World) doClanExpel(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPExpel] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if !immcom && victim.ClanRank >= ch.ClanRank {
		ch.SendMessage("You cannot kick out that person.\r\n")
		return
	}

	victim.ClanID = 0
	victim.ClanRank = 0
	c.Members--
	c.Power -= victim.Level

	victim.SendMessage("You've been kicked out of your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_promote
// ---------------------------------------------------------------------------

func (w *World) doClanPromote(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPPromote] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if victim.ClanRank == 0 {
		ch.SendMessage("They're not enrolled yet.\r\n")
		return
	}
	if !immcom && victim.ClanRank+1 > ch.ClanRank {
		ch.SendMessage("You cannot promote that person over your rank!\r\n")
		return
	}
	if victim.ClanRank == c.Ranks {
		ch.SendMessage("You cannot promote someone over the top rank!\r\n")
		return
	}

	victim.ClanRank++
	victim.SendMessage("You've been promoted within your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_demote
// ---------------------------------------------------------------------------

func (w *World) doClanDemote(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPDemote] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if victim.ClanRank == 1 {
		ch.SendMessage("They can't be demoted any further, use expel now.\r\n")
		return
	}
	if !immcom && victim.ClanRank >= ch.ClanRank {
		ch.SendMessage("You cannot demote a person of this rank!\r\n")
		return
	}

	victim.ClanRank--
	victim.SendMessage("You've been demoted within your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_who
// ---------------------------------------------------------------------------

func (w *World) doClanWho(ch *Player) {
	if ch.ClanRank == 0 {
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	ch.SendMessage("\r\nClan members online\r\n" +
		"-------------------------\r\n")

	for _, p := range w.players {
		if p.ClanID == ch.ClanID && p.ClanRank > 0 && chCanSee(ch, p) {
			rankName := ""
			if p.ClanRank-1 >= 0 && p.ClanRank-1 < len(c.RankName) {
				rankName = c.RankName[p.ClanRank-1]
			}
			ch.SendMessage(fmt.Sprintf("%s %s\r\n", rankName, p.Name))
		}
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_members
// ---------------------------------------------------------------------------

func (w *World) doClanMembers(ch *Player) {
	if ch.ClanID == 0 || ch.ClanRank == 0 {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	// For now, only list online members (can't read all saved players without the pfile system)
	ch.SendMessage("\r\nList of your clan members (online)\r\n" +
		"-------------------------\r\n")

	for _, p := range w.players {
		if p.ClanID == ch.ClanID && p.ClanRank != 0 {
			rankName := ""
			if p.ClanRank-1 >= 0 && p.ClanRank-1 < len(c.RankName) {
				rankName = c.RankName[p.ClanRank-1]
			}
			ch.SendMessage(fmt.Sprintf("%s %s\r\n", rankName, p.Name))
		}
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_quit
// ---------------------------------------------------------------------------

func (w *World) doClanQuit(ch *Player) {
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot quit any clan!\r\n")
		return
	}

	clanNum, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	ch.ClanID = 0
	ch.ClanRank = 0
	c.Members--
	c.Power -= ch.Level

	if c.Members == 0 {
		w.Clans.RemoveClan(clanNum)
		ch.SendMessage("You've quit your clan and it has been disbanded.\r\n")
	} else {
		ch.SendMessage("You've quit your clan.\r\n")
	}

	w.SaveClans()
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_status
// ---------------------------------------------------------------------------

