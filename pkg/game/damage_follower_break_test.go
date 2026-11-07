package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// wireDamageTestCombat installs the game callbacks plus the live
// BroadcastPerReceiver seam (wired in session/manager.go on the server), so the
// fade's room line reaches MessageSink.
func wireDamageTestCombat(t *testing.T, w *World) map[string]*strings.Builder {
	t.Helper()
	cb := w.WireCombatCallbacks()
	cb.BroadcastPerReceiver = func(room int, render func(combat.Combatant) string, exclude []combat.Combatant) {
		for _, p := range w.GetPlayersInRoom(room) {
			skip := false
			for _, e := range exclude {
				if e == combat.Combatant(p) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			p.SendMessage(render(p))
		}
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(cb)
	return captureMovementOutput(w)
}

func captured(output map[string]*strings.Builder, name string) string {
	if b := output[name]; b != nil {
		return b.String()
	}
	return ""
}

// C damage() breaks the victim's following when the attacker is the victim's
// master on EVERY entry, before the hide fade and before any damage message
// (src/fight.c:1457-1464). The spell (mag_damage), skill-tail and mob-special
// seams delayed the break until the next melee round (DP-1489).
func TestDamageEntryBreaksFollower(t *testing.T) {
	newPair := func(t *testing.T) (*World, *Player, *Player) {
		t.Helper()
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Open Field", Zone: 1}}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		leader := addMovementPlayer(t, w, "Leader", 1001)
		victim := addMovementPlayer(t, w, "Victim", 1001)
		leader.SetLevel(30)
		victim.SetLevel(30)
		victim.SetFollowingBody(leader)
		victim.SetInGroup(true)
		victim.SetAffect(affGroup, true)
		leader.SetInGroup(true)
		leader.SetAffect(affGroup, true)
		return w, leader, victim
	}

	for _, tt := range []struct {
		name string
		hit  func(w *World, leader, victim *Player)
	}{
		{
			name: "spell damage (mag_damage -> inflictDamage)",
			hit: func(w *World, leader, victim *Player) {
				spells.MagDamage(30, leader, victim, spells.SpellMindPoke, int(spells.SaveSpell), w)
			},
		},
		{
			name: "skill tail (ApplySkillDamageWithMessage)",
			hit: func(w *World, leader, victim *Player) {
				w.ApplySkillDamageWithMessage(leader, victim, 10, combat.TYPE_HIT, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, leader, victim := newPair(t)
			output := wireDamageTestCombat(t, w)
			tt.hit(w, leader, victim)
			if victim.GetFollowing() != "" || w.FollowingBody(victim) != nil {
				t.Fatalf("damage left victim following %q body %v", victim.GetFollowing(), w.FollowingBody(victim))
			}
			if got := captured(output, "Leader"); !strings.Contains(got, "Victim stops following you.\r\n") {
				t.Fatalf("leader stream = %q, want %q", got, "Victim stops following you.")
			}
			if got := captured(output, "Victim"); !strings.Contains(got, "You stop following Leader.\r\n") {
				t.Fatalf("victim stream = %q, want %q", got, "You stop following Leader.")
			}
		})
	}

	t.Run("mob special (mobSkillDamageAfterGate)", func(t *testing.T) {
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Open Field", Zone: 1}}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		victim := addMovementPlayer(t, w, "Victim", 1001)
		victim.SetLevel(30)
		mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "horse pony", ShortDesc: "a horse"}, 1001)
		w.mu.Lock()
		mob.ID = w.nextMobID
		w.nextMobID++
		w.activeMobs[mob.ID] = mob
		w.mu.Unlock()
		victim.SetFollowingBody(mob)
		output := wireDamageTestCombat(t, w)

		w.mobSkillDamageAfterGate(mob, victim, 5, SkillBashNum)

		if victim.GetFollowing() != "" || w.FollowingBody(victim) != nil {
			t.Fatalf("mob damage left victim following %q body %v", victim.GetFollowing(), w.FollowingBody(victim))
		}
		if got := captured(output, "Victim"); !strings.Contains(got, "You stop following a horse.\r\n") {
			t.Fatalf("victim stream = %q, want %q", got, "You stop following a horse.")
		}
	})
}

// damage()'s inline appear() runs right after the follower break
// (src/fight.c:1460-1464). The spell seam omitted it with the break (DP-1489).
func TestSpellDamageFadesHiddenAttacker(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Open Field", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	leader := addMovementPlayer(t, w, "Leader", 1001)
	victim := addMovementPlayer(t, w, "Victim", 1001)
	leader.SetLevel(30)
	victim.SetLevel(30)
	leader.SetAffect(affHide, true)
	output := wireDamageTestCombat(t, w)

	spells.MagDamage(30, leader, victim, spells.SpellMindPoke, int(spells.SaveSpell), w)

	if leader.IsAffected(affHide) {
		t.Fatal("hidden attacker kept AFF_HIDE through damage()")
	}
	if got := captured(output, "Victim"); !strings.Contains(got, "Leader slowly fades into existence.") {
		t.Fatalf("victim stream = %q, want %q", got, "Leader slowly fades into existence.")
	}
}
