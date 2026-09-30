package game

import (
	"log/slog"
	"strconv"
)

func (w *World) doClanRename(ch *Player, arg string) {
	arg1, arg2 := halfChop(arg)

	if !isClanNumber(arg1) {
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
	c.Name = capClanName(arg2)
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
	if ch.GetLevel() < LVL_GOD {
		ch.SendMessage("You are not mighty enough to create new clans!\r\n")
		return
	}
	if w.Clans.ClanCount() >= MaxClans {
		ch.SendMessage("Max clans reached. WOW!\r\n")
		return
	}

	arg1, arg2 := halfChop(arg)

	target, hasLeader := w.ResolveCharWorld(ch, arg1)
	leader := target.Player
	if !hasLeader || leader == nil {
		ch.SendMessage("The leader of the new clan must be present.\r\n")
		return
	}

	if len(arg2) >= 32 {
		ch.SendMessage("Clan name too long! (32 characters max)\r\n")
		return
	}
	if leader.GetLevel() >= LVL_IMMORT {
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
		Name:      capClanName(arg2),
		Ranks:     2,
		Members:   1,
		Power:     leader.GetLevel(),
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
	// C saves the leader after the assignment (clan.c:233).
	w.saveCharSiteInRoom(leader, "clan create")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_destroy
// ---------------------------------------------------------------------------

func (w *World) doClanDestroy(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.GetLevel() < LVL_GOD {
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
			// C saves each cleared member (clan.c:266).
			w.saveCharSiteInRoom(p, "clan destroy")
		}
	}

	// C walks the player store as well as live members (clan.c:266).
	players, err := w.StoredPlayers()
	if err != nil {
		slog.Error("clan destroy: read players", "error", err)
	}
	for _, p := range players {
		if _, online := w.players[p.Name]; online || p.ClanID != c.ID {
			continue
		}
		if err := w.EditStoredPlayer(p.Name, func(current *Player) {
			if current.ClanID == c.ID {
				current.ClanID, current.ClanRank = 0, 0
			}
		}); err != nil {
			slog.Error("clan destroy: save offline member", "name", p.Name, "error", err)
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
