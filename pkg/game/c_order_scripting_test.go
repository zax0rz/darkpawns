package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCOrderListOutput_GetMobsInRoomScriptable verifies R5h for
// GetMobsInRoomScriptable (src/scripts.c:1955): three mobs in the same room
// must be exposed to Lua in C's world[room].people newest-first order, 50/50 times.
func TestCOrderListOutput_GetMobsInRoomScriptable(t *testing.T) {
	const roomVNum = 9700
	protos := []parser.Mob{
		makeCOrderMobProto(9701, "alpha", "Alpha Mob", 10, 0),
		makeCOrderMobProto(9702, "beta", "Beta Mob", 10, 0),
		makeCOrderMobProto(9703, "gamma", "Gamma Mob", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		if _, err := w.SpawnMob(9701, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9701: %v", iter, err)
		}
		if _, err := w.SpawnMob(9702, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9702: %v", iter, err)
		}
		if _, err := w.SpawnMob(9703, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9703: %v", iter, err)
		}

		got := w.GetMobsInRoomScriptable(roomVNum)
		if len(got) != 3 {
			t.Fatalf("iter %d: got %d mobs, want 3", iter, len(got))
		}
		want := []string{"Gamma Mob", "Beta Mob", "Alpha Mob"}
		for i := range want {
			if got[i].GetName() != want[i] {
				t.Fatalf("iter %d: scriptable mob[%d] = %q, want %q", iter, i, got[i].GetName(), want[i])
			}
		}
	}
}

// TestCOrderFirstMatch_SetObjectExtraFlag verifies R5h for SetObjectExtraFlag:
// with three instances of the same object vnum, the newest instance (highest ID,
// C object_list head) must receive the flag, 50/50 times.
func TestCOrderFirstMatch_SetObjectExtraFlag(t *testing.T) {
	const roomVNum = 9800
	const objVNum = 9810
	for iter := range 50 {
		w := newCOrderTestWorld(t, nil, roomVNum)
		proto := &parser.Obj{VNum: objVNum, Keywords: "gem", ShortDesc: "a gem"}
		w.objs[objVNum] = proto
		o1 := w.CreateObject(objVNum, roomVNum)
		o2 := w.CreateObject(objVNum, roomVNum)
		o3 := w.CreateObject(objVNum, roomVNum)
		if o1 == nil || o2 == nil || o3 == nil {
			t.Fatalf("iter %d: CreateObject returned nil", iter)
		}
		if !w.SetObjectExtraFlag(objVNum, 0, true) {
			t.Fatalf("iter %d: SetObjectExtraFlag returned false", iter)
		}
		if !o3.HasExtraFlag(0, 0) || o1.HasExtraFlag(0, 0) || o2.HasExtraFlag(0, 0) {
			t.Fatalf("iter %d: expected only newest object o3 (ID=%d) flagged; got o1=%v o2=%v o3=%v",
				iter, o3.ID, o1.HasExtraFlag(0, 0), o2.HasExtraFlag(0, 0), o3.HasExtraFlag(0, 0))
		}
	}
}

// TestCOrderFirstMatch_GetMobByName verifies R5h for GetMobByName:
// with three mobs matching the substring, the newest spawned (C character_list
// head) must be returned, 50/50 times.
func TestCOrderFirstMatch_GetMobByName(t *testing.T) {
	const roomVNum = 9900
	protos := []parser.Mob{
		makeCOrderMobProto(9901, "guard", "guard Alpha", 10, 0),
		makeCOrderMobProto(9902, "guard", "guard Beta", 10, 0),
		makeCOrderMobProto(9903, "guard", "guard Gamma", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		_, _ = w.SpawnMob(9901, roomVNum)
		_, _ = w.SpawnMob(9902, roomVNum)
		m3, _ := w.SpawnMob(9903, roomVNum)

		got := w.GetMobByName("guard")
		if got != m3 {
			t.Fatalf("iter %d: GetMobByName = %v, want newest m3 (%q)", iter, got, m3.GetName())
		}
	}
}
