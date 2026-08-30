package game

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
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
		}
	}

	// Clear clan from offline members
	files, err := os.ReadDir(saveDir)
	if err != nil {
		slog.Error("clan destroy: cannot read save dir to clear offline members",
			"clan", c.ID, "error", err)
	} else {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(f.Name(), ".json")
			// Skip if currently online
			if _, ok := w.players[name]; ok {
				continue
			}
			p, lerr := LoadPlayer(name)
			if lerr != nil {
				slog.Warn("clan destroy: skipping unloadable player file",
					"name", name, "error", lerr)
				continue
			}
			if p.ClanID == c.ID {
				p.ClanID = 0
				p.ClanRank = 0
				// Offline member; log the failure with context rather than swallow.
				// DP-911.
				if serr := SavePlayer(p); serr != nil {
					slog.Error("failed to save offline player after clan destroy",
						"name", name, "clan", c.ID, "error", serr)
				}
			}
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
