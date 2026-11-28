package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCOrderDefault_GetMobsInRoom verifies R5h for GetMobsInRoom returning
// C's world[room].people newest-first order by default, 50/50 times.
func TestCOrderDefault_GetMobsInRoom(t *testing.T) {
	const roomVNum = 9970
	protos := []parser.Mob{
		makeCOrderMobProto(9971, "alpha", "Alpha Mob", 10, 0),
		makeCOrderMobProto(9972, "beta", "Beta Mob", 10, 0),
		makeCOrderMobProto(9973, "gamma", "Gamma Mob", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		m1, _ := w.SpawnMob(9971, roomVNum)
		m2, _ := w.SpawnMob(9972, roomVNum)
		m3, _ := w.SpawnMob(9973, roomVNum)
		got := w.GetMobsInRoom(roomVNum)
		if len(got) != 3 || got[0] != m3 || got[1] != m2 || got[2] != m1 {
			t.Fatalf("iter %d: GetMobsInRoom = %v, want [m3 m2 m1]", iter, got)
		}
	}
}

// TestCOrderDefault_GetPlayersInRoom verifies R5h for GetPlayersInRoom returning
// C's world[room].people newest-first order by default, 50/50 times.
func TestCOrderDefault_GetPlayersInRoom(t *testing.T) {
	const roomVNum = 9974
	for iter := range 50 {
		w := newCOrderTestWorld(t, nil, roomVNum)
		p1 := NewPlayer(1, "Alpha", roomVNum)
		p2 := NewPlayer(2, "Beta", roomVNum)
		p3 := NewPlayer(3, "Gamma", roomVNum)
		for _, p := range []*Player{p1, p2, p3} {
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("iter %d: AddPlayer %s: %v", iter, p.Name, err)
			}
		}
		got := w.GetPlayersInRoom(roomVNum)
		if len(got) != 3 || got[0] != p3 || got[1] != p2 || got[2] != p1 {
			t.Fatalf("iter %d: GetPlayersInRoom = %v, want [p3 p2 p1]", iter, got)
		}
	}
}

// TestCOrderDefault_GetAllPlayers verifies R5h for GetAllPlayers returning
// C's character_list newest-first order by default, 50/50 times.
func TestCOrderDefault_GetAllPlayers(t *testing.T) {
	const r1, r2 = 9975, 9976
	for iter := range 50 {
		w := newCOrderTestWorld(t, nil, r1, r2)
		p1 := NewPlayer(1, "Alpha", r1)
		p2 := NewPlayer(2, "Beta", r2)
		p3 := NewPlayer(3, "Gamma", r1)
		for _, p := range []*Player{p1, p2, p3} {
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("iter %d: AddPlayer %s: %v", iter, p.Name, err)
			}
		}
		got := w.GetAllPlayers()
		if len(got) != 3 || got[0] != p3 || got[1] != p2 || got[2] != p1 {
			t.Fatalf("iter %d: GetAllPlayers = %v, want [p3 p2 p1]", iter, got)
		}
	}
}

// TestCOrderDefault_GetAllMobs verifies R5h for GetAllMobs returning
// C's character_list newest-first order by default, 50/50 times.
func TestCOrderDefault_GetAllMobs(t *testing.T) {
	const r1, r2 = 9977, 9978
	protos := []parser.Mob{
		makeCOrderMobProto(9977, "alpha", "Alpha Mob", 10, 0),
		makeCOrderMobProto(9978, "beta", "Beta Mob", 10, 0),
		makeCOrderMobProto(9979, "gamma", "Gamma Mob", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, r1, r2)
		m1, _ := w.SpawnMob(9977, r1)
		m2, _ := w.SpawnMob(9978, r2)
		m3, _ := w.SpawnMob(9979, r1)
		got := w.GetAllMobs()
		if len(got) != 3 || got[0] != m3 || got[1] != m2 || got[2] != m1 {
			t.Fatalf("iter %d: GetAllMobs = %v, want [m3 m2 m1]", iter, got)
		}
	}
}

// TestCOrderDefault_GetAllObjects verifies R5h for GetAllObjects returning
// C's object_list newest-first order by default, 50/50 times.
func TestCOrderDefault_GetAllObjects(t *testing.T) {
	const roomVNum = 9979
	const objVNum = 9980
	for iter := range 50 {
		w := newCOrderTestWorld(t, nil, roomVNum)
		w.objs[objVNum] = &parser.Obj{VNum: objVNum, Keywords: "coin", ShortDesc: "a coin"}
		o1 := w.CreateObject(objVNum, roomVNum)
		o2 := w.CreateObject(objVNum, roomVNum)
		o3 := w.CreateObject(objVNum, roomVNum)
		got := w.GetAllObjects()
		if len(got) != 3 || got[0] != o3 || got[1] != o2 || got[2] != o1 {
			t.Fatalf("iter %d: GetAllObjects = %v, want [o3 o2 o1]", iter, got)
		}
	}
}
