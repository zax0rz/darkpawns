package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestStopFollowerMobClearsCharmAndGroup(t *testing.T) {
	for _, spell := range []bool{false, true} {
		t.Run(map[bool]string{false: "ride-bit", true: "spell-record"}[spell], func(t *testing.T) {
			w, master, mob := newRideDepthWorld(t)
			w.StopAITicker()
			mob.SetFollowingBody(master)
			mob.SetAffected(affCharm)
			mob.SetAffected(affGroup)
			if spell {
				mob.AddAffect(engine.NewAffectDirect(7, engine.ApplyNone, 10, 0, engine.AFFCharm, "charm"))
			}
			StopFollowerMob(w, mob)
			if mob.GetFollowing() != "" || w.FollowingBody(mob) != nil || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
				t.Fatalf("stop relation=%q charm=%t group=%t", mob.GetFollowing(), mob.IsAffected(affCharm), mob.IsAffected(affGroup))
			}
			if spell {
				for _, value := range mob.CustomData {
					if aff, ok := value.(*engine.Affect); ok && aff.SpellID == 7 {
						t.Fatal("spell charm record survived stop")
					}
				}
			}
		})
	}
}

func TestDamageBreaksMountCharmAndGroup(t *testing.T) {
	w, master, mob := newRideDepthWorld(t)
	w.StopAITicker()
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(w.WireCombatCallbacks())
	mob.SetFollowingBody(master)
	mob.SetAffected(affCharm)
	mob.SetAffected(affGroup)
	combat.TakeDamageWithDeath(master, mob, 0, combat.TYPE_HIT, func() { t.Fatal("nonlethal attack died") })
	if w.FollowingBody(mob) != nil || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
		t.Fatalf("damage left master=%v charm=%t group=%t", w.FollowingBody(mob), mob.IsAffected(affCharm), mob.IsAffected(affGroup))
	}
}

func TestDeathReleasesFollowers(t *testing.T) {
	for _, path := range []string{"combat", "noncombat", "instakill", "extract-flag"} {
		t.Run(path, func(t *testing.T) {
			w, master, mob := newRideDepthWorld(t)
			w.StopAITicker()
			master.SetLevel(1)
			older := addMovementPlayer(t, w, "Older", 1001)
			newer := addMovementPlayer(t, w, "Newer", 1001)
			observer := addMovementPlayer(t, w, "Observer", 1001)
			older.SetFollowing(master.Name) // legacy player-name edge must resolve before deletion
			older.SetInGroup(true)
			older.SetAffect(affGroup, true)
			mob.SetFollowingBody(master)
			mob.SetAffected(affCharm)
			mob.SetAffected(affGroup)
			newer.SetFollowingBody(master)
			newer.SetInGroup(true)
			newer.SetAffect(affGroup, true)
			capture := captureMovementOutput(w)
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			combat.SetCallbacks(w.WireCombatCallbacks())
			master.SetHP(-11)
			master.SetPosition(combat.PosDead)
			switch path {
			case "combat":
				w.HandleDeath(master, observer, combat.TYPE_HIT)
			case "noncombat":
				w.HandleNonCombatDeath(master)
			case "instakill":
				w.Instakill(master, observer, combat.TYPE_HIT)
			case "extract-flag":
				ExtractChar(master)
			}
			if w.FollowingBody(older) != master || w.FollowingBody(newer) != master || w.FollowingBody(mob) != master {
				t.Fatal("followers released before deferred extraction")
			}
			// Isolate the final extraction bytes from the death cry/corpse pipeline.
			for _, out := range capture {
				out.Reset()
			}
			extracted := w.ExtractPendingPlayers()
			if len(extracted) != 1 || extracted[0] != master {
				t.Fatalf("extracted=%v", extracted)
			}
			for _, p := range []*Player{older, newer} {
				if p.GetFollowing() != "" || w.FollowingBody(p) != nil || p.IsInGroup() || p.IsAffected(affGroup) {
					t.Fatalf("player %s remains a follower/group member", p.Name)
				}
				if got := capture[p.Name].String(); !strings.Contains(got, "You stop following Rider.\r\n") {
					t.Fatalf("follower bytes=%q", got)
				}
			}
			if w.FollowingBody(mob) != nil || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
				t.Fatal("NPC follower relation/affects survived master's death")
			}
			want := "Newer stops following Rider.\r\nA horse realizes that Rider is a jerk!\r\nOlder stops following Rider.\r\n"
			if got := capture[observer.Name].String(); got != want {
				t.Fatalf("extraction order=%q want %q", got, want)
			}
			if master.GetRoom() != roomNowhere {
				t.Fatalf("master room=%d", master.GetRoom())
			}
		})
	}
}

func TestDeathFollowerOwnMasterAndRemoteFollower(t *testing.T) {
	w, master, _ := newRideDepthWorld(t)
	w.StopAITicker()
	elder := addMovementPlayer(t, w, "Elder", 1001)
	remote := addMovementPlayer(t, w, "Remote", 1001)
	w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Remote room", Zone: 1}
	remote.SetRoom(1002)
	master.SetFollowingBody(elder)
	remote.SetFollowingBody(master)
	remote.SetAffect(affGroup, true)
	remote.SetInGroup(true)
	capture := captureMovementOutput(w)
	ExtractChar(master)
	w.ExtractPendingPlayers()
	if master.GetFollowing() != "" || remote.GetFollowing() != "" || remote.IsInGroup() || remote.IsAffected(affGroup) {
		t.Fatal("outgoing/remote relation survived extraction")
	}
	if got := capture[remote.Name].String(); got != "You stop following Rider.\r\n" {
		t.Fatalf("remote bytes=%q", got)
	}
	if got := capture[elder.Name].String(); !strings.HasPrefix(got, "Rider stops following you.\r\n") {
		t.Fatalf("outgoing master bytes=%q", got)
	}
}

func TestNPCDeathReleasesFollowersAtExtraction(t *testing.T) {
	for _, path := range []string{"death", "raw-kill", "extract", "extract-flag"} {
		t.Run(path, func(t *testing.T) {
			w, master, mob := newRideDepthWorld(t)
			w.StopAITicker()
			follower := addMovementPlayer(t, w, "Follower", 1001)
			follower.SetFollowingBody(mob)
			follower.SetAffect(affGroup, true)
			follower.SetInGroup(true)
			mob.SetFollowingBody(master)
			mob.SetAffected(affCharm)
			mob.SetAffected(affGroup)
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			combat.SetCallbacks(w.WireCombatCallbacks())
			capture := captureMovementOutput(w)
			switch path {
			case "death":
				mob.SetHealth(-11)
				mob.SetPosition(combat.PosDead)
				w.HandleDeath(mob, master, combat.TYPE_HIT)
			case "raw-kill":
				w.RawKillCombatant(mob, combat.TYPE_HIT)
			case "extract":
				w.ExtractMob(mob)
			case "extract-flag":
				mob.SetMobFlag(MobFlagExtract)
			}
			if w.FollowingBody(follower) != mob {
				t.Fatal("NPC followers released before final extraction")
			}
			for _, out := range capture {
				out.Reset()
			}
			w.ExtractPendingPlayers()
			if follower.GetFollowing() != "" || follower.IsInGroup() || follower.IsAffected(affGroup) || mob.GetFollowing() != "" || mob.IsAffected(affCharm) || mob.IsAffected(affGroup) {
				t.Fatal("NPC extraction left follower/charm/group state")
			}
			if got := capture[follower.Name].String(); !strings.HasSuffix(got, "You stop following a horse.\r\n") {
				t.Fatalf("NPC master bytes=%q", got)
			}
		})
	}
}

func TestDeathFollowerCleanupBeforeDropAndRemoval(t *testing.T) {
	w, master, _ := newRideDepthWorld(t)
	w.StopAITicker()
	follower := addMovementPlayer(t, w, "Follower", 1001)
	follower.SetFollowing(master.Name)
	proto := &parser.Obj{VNum: 9002, Keywords: "token", ShortDesc: "a token", TypeFlag: 12}
	obj := w.NewObjectFromProto(proto, roomNowhere)
	if err := w.MoveObjectToPlayerInventory(obj, master); err != nil {
		t.Fatal(err)
	}
	notified := false
	w.MessageSink = func(name string, msg []byte) {
		if name != follower.Name || !strings.Contains(string(msg), "You stop following Rider.") {
			return
		}
		notified = true
		// These getters acquire World.mu: delivery must hold no world lock.
		if current, ok := w.GetPlayer(master.Name); !ok || current != master || master.GetRoom() != 1001 {
			t.Error("master removed before follower act")
		}
		if obj.Location != LocInventoryPlayer(master.Name) {
			t.Error("possessions dropped before follower act")
		}
	}
	ExtractChar(master)
	w.ExtractPendingPlayers()
	if !notified {
		t.Fatal("no follower act at final extraction")
	}
	if obj.Location != LocRoom(1001) {
		t.Fatal("extraction did not drop remaining possessions")
	}
}
