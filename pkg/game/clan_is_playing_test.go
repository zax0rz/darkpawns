package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestClanIsPlayingGates(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.Clans = NewClanManager()
	clan := &Clan{Name: "Wolves", Ranks: 2}
	clan.RankName[0] = "Member"
	clan.RankName[1] = "Leader"
	clan.Privilege[CPEnroll] = 2
	w.Clans.AddClan(clan)

	var actorOut strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == "Leader" {
			actorOut.Write(msg)
		}
	}

	leader := &Player{Name: "Leader", Level: 34, RoomVNum: 1001, ClanID: clan.ID, ClanRank: 2}
	mortalLeader := &Player{Name: "Mortal", Level: 10, RoomVNum: 1001, ClanID: clan.ID, ClanRank: 2}
	linkless := &Player{Name: "Linkless", Level: 10, RoomVNum: 1001, ClanID: clan.ID, ClanRank: 0}
	storedLinkless := &Player{Name: "Linkless", Level: 10, ClanID: 0, ClanRank: 0}

	w.AddPlayer(leader)
	w.AddPlayer(mortalLeader)
	w.AddPlayer(linkless)
	w.PlayerHasPlayingDescriptor = func(p *Player) bool {
		return p == leader || p == mortalLeader
	}
	w.PlayerStoreList = func() ([]*Player, error) {
		return []*Player{leader, storedLinkless}, nil
	}
	w.PlayerStoreEdit = func(name string, change func(*Player)) error {
		if name == storedLinkless.Name {
			change(storedLinkless)
		}
		return nil
	}

	// 1. do_clan_enroll applicant list (clan.c:328): linkless in-memory applicant
	// is skipped by is_playing(), so stored state (ClanID=0) is used -> omitted.
	w.doClanEnroll(mortalLeader, "")
	if strings.Contains(actorOut.String(), "Linkless") {
		t.Fatalf("doClanEnroll listed linkless unsaved applicant: %q", actorOut.String())
	}

	// 2. do_clan_set_rank (clan.c:1132): stored rank is updated for non-playing
	// member without mutating the in-memory linkless body.
	storedLinkless.ClanID, storedLinkless.ClanRank = clan.ID, 2
	linkless.ClanID, linkless.ClanRank = clan.ID, 2
	clan.Ranks = 3
	w.doClanRanks(leader, "2 Wolves")
	if storedLinkless.ClanRank != 1 {
		t.Fatalf("storedLinkless.ClanRank = %d, want 1", storedLinkless.ClanRank)
	}
	if linkless.ClanRank != 2 {
		t.Fatalf("linkless.ClanRank = %d, want 2 (unmodified)", linkless.ClanRank)
	}

	// 3. do_clan_destroy (clan.c:262): clears stored record when not is_playing,
	// leaving linkless in-memory body untouched and clearing CON_PLAYING leader.
	w.doClanDestroy(leader, "Wolves")
	if storedLinkless.ClanID != 0 || storedLinkless.ClanRank != 0 {
		t.Fatalf("storedLinkless after destroy = (%d,%d), want (0,0)", storedLinkless.ClanID, storedLinkless.ClanRank)
	}
	if linkless.ClanID == 0 {
		t.Fatal("doClanDestroy mutated non-playing linkless body instead of stored record")
	}
	if leader.ClanID != 0 || leader.ClanRank != 0 {
		t.Fatalf("leader after destroy = (%d,%d), want (0,0)", leader.ClanID, leader.ClanRank)
	}
}
