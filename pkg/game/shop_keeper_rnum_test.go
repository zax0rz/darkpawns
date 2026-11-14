package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestShowShopsKeeperRNumZeroCollision pins DP-1527's keeper resolution:
// C's get_char_num (src/shop.c:1345-1350) scans character_list newest-first
// for nr == keeper rnum; players' nr is 0 (clear_char, src/db.c), so with a
// keeper at proto rnum 0 -- Puff, the first mobile in the database -- the
// logged-in player wins and show shops prints the player's gold. Fails if
// the resolver reverts to a mobile-only scan (R5h).
func TestShowShopsKeeperRNumZeroCollision(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Limbo"}},
		Mobs: []parser.Mob{
			{VNum: 1, Keywords: "puff dragon", ShortDesc: "Puff"},
			{VNum: 5001, Keywords: "keeper merchant", ShortDesc: "a merchant"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.MessageSink = func(string, []byte) {}

	puff, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob puff: %v", err)
	}
	puff.SetGold(10)

	// No player logged in: the rnum-0 keeper herself is found.
	player, mob := w.ShowShopsKeeper(1)
	if player != nil || mob != puff {
		t.Fatalf("no-player lookup = (%v, %v), want Puff", player, mob)
	}

	// A player logs in and prepends to the C list: the collision wins.
	god := NewPlayer(1, "Shopdiv", 1001)
	god.SetGold(500)
	if err := w.AddPlayer(god); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	player, mob = w.ShowShopsKeeper(1)
	if player != god || mob != nil {
		t.Fatalf("player-collision lookup = (%v, %v), want the player", player, mob)
	}

	// A keeper at a higher rnum is unaffected by the player's nr 0.
	merchant, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob merchant: %v", err)
	}
	player, mob = w.ShowShopsKeeper(5001)
	if player != nil || mob != merchant {
		t.Fatalf("merchant lookup = (%v, %v), want the merchant", player, mob)
	}
}
