package game

import (
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (w *World) doClanRename(ch *Player, arg string) {
	arg1, arg2 := halfChop(arg)

	if !isNumber(arg1) {
		ch.SendMessage("You need to specify a clan number.\r\n")
		return
	}
	clanIdx, _ := strconv.Atoi(arg1)
	if clanIdx < 0 || clanIdx >= w.Clans.ClanCount() {
		ch.SendMessage("There is no clan with that number.\r\n")
		return
	}

	if arg2 == "" {
		ch.SendMessage("What do you want to rename it?\r\n")
		return
	}

	c := w.Clans.GetClanByIndex(clanIdx)
	if c == nil {
		ch.SendMessage("There is no clan with that number.\r\n")
		return
	}
	if len(arg2) > 32 {
		arg2 = arg2[:32]
	}
	c.Name = cases.Title(language.English).String(strings.ToLower(arg2))
	w.SaveClans()
	ch.SendMessage("Clan renamed.\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_create
// ---------------------------------------------------------------------------

func (w *World) doClanCreate(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level < LVL_IMMORT {
		ch.SendMessage("You are not mighty enough to create new clans!\r\n")
		return
	}
	if w.Clans.ClanCount() >= MaxClans {
		ch.SendMessage("Max clans reached. WOW!\r\n")
		return
	}

	arg1, arg2 := halfChop(arg)

	leader, hasLeader := w.GetPlayer(arg1)
	if !hasLeader {
		ch.SendMessage("The leader of the new clan must be present.\r\n")
		return
	}

	if len(arg2) >= 32 {
		ch.SendMessage("Clan name too long! (32 characters max)\r\n")
		return
	}
	if leader.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot set an immortal as the leader of a clan.\r\n")
		return
	}
	if leader.ClanID != 0 && leader.ClanRank != 0 {
		ch.SendMessage("The leader already belongs to a clan!\r\n")
		return
	}

	if _, c := w.Clans.FindClan(arg2); c != nil {
		ch.SendMessage("That clan name already exists!\r\n")
		return
	}

	newClan := &Clan{
		Name:      cases.Title(language.English).String(strings.ToLower(arg2)),
		Ranks:     2,
		Members:   1,
		Power:     leader.Level,
		ApplLevel: DefaultAppLvl,
		Private:   ClanPublic,
	}
	newClan.RankName[0] = "Member"
	newClan.RankName[1] = "Leader"

	// All privileges default to leader rank
	for i := 0; i < 20; i++ {
		newClan.Privilege[i] = newClan.Ranks
	}

	w.Clans.AddClan(newClan)
	w.SaveClans()
	ch.SendMessage("Clan created.\r\n")

	// Assign leader
	leader.ClanID = newClan.ID
	leader.ClanRank = newClan.Ranks
	// Save player state (simplified)
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_destroy
// ---------------------------------------------------------------------------

func (w *World) doClanDestroy(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level < LVL_IMMORT {
		ch.SendMessage("Your not mighty enough to destroy clans!\r\n")
		return
	}

	i, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	// Clear clan from all online members
	for _, p := range w.players {
		if p.ClanID == c.ID {
			p.ClanID = 0
			p.ClanRank = 0
		}
	}

	// Remove clan
	w.Clans.RemoveClan(i)
	w.SaveClans()
	ch.SendMessage("Clan deleted.\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_enroll
// ---------------------------------------------------------------------------

