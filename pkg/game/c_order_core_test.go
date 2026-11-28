package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCOrderListOutput_DarkOccupants verifies R5h for appendDarkOccupants
// (src/act.informative.c:450): in a dark room with mixed players and mobs
// entered in a known order, glowing-red-eyes vs moving-nearby lines must appear
// in C's world[room].people newest-first order, 50/50 times.
func TestCOrderListOutput_DarkOccupants(t *testing.T) {
	const roomVNum = 9400
	protos := []parser.Mob{
		makeCOrderMobProto(9401, "mobA", "Mob Alpha", 10, 0),
		makeCOrderMobProto(9402, "mobB", "Mob Beta", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		viewer := NewPlayer(1, "Viewer", roomVNum)
		if err := w.AddPlayer(viewer); err != nil {
			t.Fatalf("iter %d: AddPlayer viewer: %v", iter, err)
		}
		// Enter order: mA (normal, oldest), peer (infravision, middle), mB (normal, newest).
		mA, _ := w.SpawnMob(9401, roomVNum)
		_ = mA
		peer := NewPlayer(2, "Peer", roomVNum)
		peer.SetAffect(affInfravision, true)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("iter %d: AddPlayer peer: %v", iter, err)
		}
		mB, _ := w.SpawnMob(9402, roomVNum)
		_ = mB

		var res ObservationResult
		w.appendDarkOccupants(&res, viewer, w.rooms[roomVNum])
		var got []string
		for _, d := range res.Messages {
			got = append(got, d.Format)
		}
		want := []string{
			"You hear someone or something moving around nearby.",
			"You see a pair of glowing red eyes.",
			"You hear someone or something moving around nearby.",
		}
		if len(got) != len(want) {
			t.Fatalf("iter %d: got %d lines %v, want %d", iter, len(got), got, len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("iter %d: line %d = %q, want %q (all=%v)", iter, i, got[i], want[i], got)
			}
		}
	}
}

// TestCOrderFirstMatch_DoRescue verifies R5h for DoRescue
// (src/act.offensive.c:544-546): when three mobs are fighting the rescue
// target, DoRescue must engage the newest arrival in world[room].people,
// 50/50 times.
func TestCOrderFirstMatch_DoRescue(t *testing.T) {
	const roomVNum = 9500
	protos := []parser.Mob{
		makeCOrderMobProto(9501, "foeA", "Foe Alpha", 10, 0),
		makeCOrderMobProto(9502, "foeB", "Foe Beta", 10, 0),
		makeCOrderMobProto(9503, "foeC", "Foe Gamma", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		rescuer := NewPlayer(1, "Rescuer", roomVNum)
		rescuer.SetSkill(SkillRescue, 100)
		ally := NewPlayer(2, "Ally", roomVNum)
		if err := w.AddPlayer(rescuer); err != nil {
			t.Fatalf("iter %d: AddPlayer rescuer: %v", iter, err)
		}
		if err := w.AddPlayer(ally); err != nil {
			t.Fatalf("iter %d: AddPlayer ally: %v", iter, err)
		}

		foeA, _ := w.SpawnMob(9501, roomVNum)
		foeB, _ := w.SpawnMob(9502, roomVNum)
		foeC, _ := w.SpawnMob(9503, roomVNum)
		foeA.SetFightingBody(ally)
		foeB.SetFightingBody(ally)
		foeC.SetFightingBody(ally)

		oldCB := combat.GetCallbacks()
		ce := combat.NewCombatEngine()
		ce.SetCallbacks(w.WireCombatCallbacks())
		w.SetCombatEngine(ce)

		dprng.ResetStream(1)
		res := DoRescue(rescuer, ally, w, ce)
		ce.Stop()
		combat.SetCallbacks(oldCB)
		if !res.Success {
			t.Fatalf("iter %d: DoRescue failed: %+v", iter, res)
		}
		if rescuer.GetFightingBody() != foeC {
			t.Fatalf("iter %d: rescuer engaged %v, want newest arrival foeC (%q)", iter, rescuer.GetFightingBody(), foeC.GetName())
		}
	}
}

// TestCOrderListOutput_GetFollowersInRoom verifies R5h for GetFollowersInRoom:
// three player followers in the same room must be returned in C ch->followers
// newest-first order, 50/50 times.
func TestCOrderListOutput_GetFollowersInRoom(t *testing.T) {
	const roomVNum = 9600
	for iter := range 50 {
		w := newCOrderTestWorld(t, nil, roomVNum)
		leader := NewPlayer(1, "Leader", roomVNum)
		f1 := NewPlayer(2, "Alpha", roomVNum)
		f2 := NewPlayer(3, "Beta", roomVNum)
		f3 := NewPlayer(4, "Gamma", roomVNum)
		for _, p := range []*Player{leader, f1, f2, f3} {
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("iter %d: AddPlayer %s: %v", iter, p.Name, err)
			}
		}
		f1.SetFollowing("Leader")
		f2.SetFollowing("Leader")
		f3.SetFollowing("Leader")

		got := w.GetFollowersInRoom("Leader", roomVNum)
		if len(got) != 3 {
			t.Fatalf("iter %d: got %d followers, want 3", iter, len(got))
		}
		want := []string{"Gamma", "Beta", "Alpha"}
		for i := range want {
			if got[i].Name != want[i] {
				t.Fatalf("iter %d: follower[%d] = %q, want %q", iter, i, got[i].Name, want[i])
			}
		}
	}
}
