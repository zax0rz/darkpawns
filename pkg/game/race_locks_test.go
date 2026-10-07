package game

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func raceLocksWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room 1", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// Two concurrent gives against one balance were a double-spend: both passed
// the GetGold check, both deducted, recipient credited twice — gold minted
// from nothing (VULN-020). Conservation must hold under concurrency.
func TestGiveGoldConservationUnderConcurrency(t *testing.T) {
	w := raceLocksWorld(t)
	ch := NewPlayer(1, "Giver", 1001)
	vict := NewPlayer(2, "Victim", 1001)
	ch.SetGold(100)
	vict.SetGold(0)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.performGiveGold(ch, vict, 60)
		}()
	}
	wg.Wait()

	total := ch.GetGold() + vict.GetGold()
	if total != 100 {
		t.Fatalf("gold minted/destroyed: total = %d, want 100 (giver %d, victim %d)", total, ch.GetGold(), vict.GetGold())
	}
	if ch.GetGold() < 0 {
		t.Fatalf("giver balance negative: %d", ch.GetGold())
	}
}

// Concurrent drop/give interleave was the second double-spend shape: the
// victim's balance read by the giver races the victim's own deduction.
func TestDropGoldConcurrentWithGive(t *testing.T) {
	w := raceLocksWorld(t)
	ch := NewPlayer(1, "Dropper", 1001)
	other := NewPlayer(2, "Giver", 1001)
	vict := NewPlayer(3, "Victim", 1001)
	ch.SetGold(100)
	other.SetGold(100)
	vict.SetGold(0)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); w.performDropGold(ch, 60) }()
	go func() { defer wg.Done(); w.performGiveGold(other, vict, 60) }()
	wg.Wait()

	// ch's own drop is internally consistent; the cross-player invariant is
	// global conservation of the 200 starting gold.
	if got := ch.GetGold() + other.GetGold() + vict.GetGold(); got > 200 {
		t.Fatalf("gold minted: total %d > 200", got)
	}
	if ch.GetGold() < 0 || other.GetGold() < 0 {
		t.Fatalf("negative balance: dropper %d, giver %d", ch.GetGold(), other.GetGold())
	}
}

// GetItemsInRoom must return a snapshot: iterating the live slice while a
// writer mutates the room was a fatal concurrent map read (VULN-019/055).
// Run under -race.
func TestGetItemsInRoomSnapshotVsWriter(t *testing.T) {
	w := raceLocksWorld(t)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	// The writer churns one object in and out of the room forever: constant
	// map-write traffic at constant memory. (An add-only writer was an
	// unbounded slice — the very CWE-400 under repair — and OOMed the host.)
	wg.Add(1)
	go func() {
		defer wg.Done()
		obj := NewObjectInstance(&parser.Obj{VNum: 42, Keywords: "coin", ShortDesc: "a coin", LongDesc: "A coin."}, -1)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = w.MoveObjectToRoom(obj, 1001)
			w.ExtractObject(obj, 1001)
		}
	}()
	// Drive the real bare-read site (getFromRoom) plus the accessor: the
	// pre-fix direct w.roomItems[...] read against a locked writer is the
	// process-fatal concurrent map access. A non-matching keyword exercises
	// the bare read without picking anything up, so the reader allocates
	// nothing per iteration.
	ch := NewPlayer(1, "Taker", 1001)
	for i := 0; i < 2000; i++ {
		w.getFromRoom(ch, "nosuchthing")
		for range w.GetItemsInRoom(1001) {
		}
	}
	close(stop)
	wg.Wait()
}

// The ban list is read on every telnet accept and mutated by admin
// ban/unban with no lock (VULN-026). Run under -race.
func TestBanManagerConcurrentReadWrite(t *testing.T) {
	bm := &BanManager{}
	if err := bm.AddBan("evil.example", BanNew, "test"); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = bm.IsBanned("host.evil.example")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_, _ = bm.RemoveBan("evil.example")
			_ = bm.AddBan("evil.example", BanNew, "test")
		}
		close(stop)
	}()
	wg.Wait()
}

// The grapevine reconnect goroutine clears the gossip callback while session
// goroutines call it (VULN-040). Run under -race.
func TestOnGossipConcurrentClearDuringCall(t *testing.T) {
	w := raceLocksWorld(t)
	w.SetOnGossip(func(name, msg string) {})
	p := NewPlayer(1, "Gossiper", 1001)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			w.SetOnGossip(nil)
			w.SetOnGossip(func(name, msg string) {})
		}
	}()
	for i := 0; i < 2000; i++ {
		w.DoChannel(p, "gossip", "hello")
	}
	close(stop)
	wg.Wait()
}
