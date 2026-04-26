// Package game — follower management.
//
// Ported from src/utils.c:
//   add_follower(), add_follower_quiet(), stop_follower(), die_follower(),
//   circle_follow(), get_mount(), get_rider()
//
// In the original C code, followers are stored as a linked list on the
// leader (ch->followers). In this Go port, followers are identified by
// the string field Following (player.Following / mob.Following), which
// stores the name of the leader. World-level queries scan the player and
// mob tables to find followers of a given leader.
//
// Mount/rider relationships use separate fields:
//   Player.MountName  → name of the mob being ridden
//   MobInstance.MountRider → name of the player riding this mob

package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// --------------------------------------------------------------------------
// Circle detection
// --------------------------------------------------------------------------

// CircleFollow returns true if following victim would create a follow loop.
// C: src/utils.c:365-374 — for (k = victim; k; k = k->master) { if (k == ch) return TRUE; }
//
// In the Go string-based follow system, this walks the chain by name.
// The function takes concrete types since it needs to look up actors by name.
func CircleFollow(w *World, ch *Player, victim *Player) bool {
	cur := victim
	for {
		if cur == ch {
			return true
		}
		if cur.Following == "" {
			return false
		}
		next, ok := w.GetPlayer(cur.Following)
		if !ok || next == nil {
			return false
		}
		cur = next
	}
}

// --------------------------------------------------------------------------
// Follower addition
// --------------------------------------------------------------------------

// AddFollowerQuiet adds ch as a follower of leader without sending messages.
// Caller must verify no follow loop exists first (use CircleFollow).
// C: src/utils.c:463-475
func AddFollowerQuiet(ch *Player, leader *Player) {
	ch.Following = leader.Name
}

// AddFollowerQuietMob adds a mob as a follower of a player (charmed pet, etc.)
// without sending messages.
// C: src/utils.c:463-475
func AddFollowerQuietMob(mob *MobInstance, leader *Player) {
	mob.Following = leader.Name
}

// AddFollower adds ch as a follower of leader with notifications.
// Caller must verify no follow loop exists first (use CircleFollow).
// C: src/utils.c:480-498
func AddFollower(w *World, ch *Player, leader *Player) {
	ch.Following = leader.Name

	Act(w, false, ch, leader, nil, nil,
		"You now follow $N.", "", ToChar)
	if canSee(leader, ch) && leader.GetPosition() > combat.PosSleeping {
		Act(w, true, ch, leader, nil, nil,
			"$n starts following you.", "", ToVict)
	}
	Act(w, true, ch, leader, nil, nil,
		"$n starts to follow $N.", "", ToNotVict)
}

// --------------------------------------------------------------------------
// Follower removal
// --------------------------------------------------------------------------

// StopFollower removes ch from his master's follower list.
// C: src/utils.c:397-440
func StopFollower(w *World, ch *Player) {
	if ch.Following == "" {
		return
	}

	// Look up the leader for act messages that need $N.
	leaderName := ch.Following
	var leader Actor
	if l, ok := w.GetPlayer(leaderName); ok {
		leader = l
	}

	charmAffected := ch.IsAffected(affCharm)

	if charmAffected {
		Act(w, false, ch, leader, nil, nil,
			"You realize that $N is a jerk!", "", ToChar)
		Act(w, true, ch, leader, nil, nil,
			"$n realizes that $N is a jerk!", "", ToNotVict)
		if leader != nil {
			Act(w, true, ch, nil, nil, nil,
				"$n hates your guts!", "", ToVict)
		}
		// Remove SPELL_CHARM from active affects if present.
		removeCharmAffect(ch)
	} else {
		Act(w, false, ch, leader, nil, nil,
			"You stop following $N.", "", ToChar)
		Act(w, true, ch, leader, nil, nil,
			"$n stops following $N.", "", ToNotVict)
	}

	// Unmount if this is a mount.
	// In the C code: if (IS_NPC(ch) && IS_MOUNTED(ch)) unmount(get_rider(ch), ch);
	// Players can't be mounts in Go (only mobs can), so we skip this for players.

	// Clear the following field — this is the Go equivalent of removing ch from
	// the leader's follower list.
	ch.Following = ""

	ch.SetAffect(affCharm, false)
	if ch.InGroup {
		ch.InGroup = false
	}
	ch.SetAffect(affGroup, false)
}

// StopFollowerMob removes a mob from its master's follower list.
// C: src/utils.c:397-440
func StopFollowerMob(w *World, mob *MobInstance) {
	if mob.Following == "" {
		return
	}

	mob.Following = ""
	mob.RemoveAffected(affCharm)
	mob.RemoveAffected(affGroup)

	// Unmount if this mob is a mount.
	if mob.IsMountedMob() {
		mob.MountRider = ""
	}
}

// --------------------------------------------------------------------------
// Die follower — cleanup when a character dies
// --------------------------------------------------------------------------

// DieFollower cleans up follower relations when ch dies.
// C: src/utils.c:447-457 — if ch->master, stop_follower(ch);
//     then for each k in ch->followers, stop_follower(k->follower)
func (w *World) DieFollower(ch *Player) {
	// If ch is following someone, stop following.
	if ch.Following != "" {
		StopFollower(w, ch)
	}

	// Find all followers of ch and make them stop following.
	for _, p := range w.players {
		if p.Following == ch.Name {
			StopFollower(w, p)
		}
	}

	// Also check mob followers (charmed pets following this player).
	for _, mob := range w.activeMobs {
		if mob.Following == ch.Name {
			StopFollowerMob(w, mob)
		}
	}
}

// DieFollowerMob cleans up follower relations when a mob dies.
// C: src/utils.c:447-457
func (w *World) DieFollowerMob(mob *MobInstance) {
	// If mob is following someone, stop following.
	if mob.Following != "" {
		StopFollowerMob(w, mob)
	}

	// If mob is being ridden, dismount the rider.
	riderName := mob.MountRider
	if riderName != "" {
		if rider, ok := w.GetPlayer(riderName); ok {
			rider.MountName = ""
		}
		mob.MountRider = ""
	}

	// Players following this mob (via MountName) need cleanup too.
	// Mobs don't have players following them in the Following sense,
	// but players can be riding this mob.
}

// --------------------------------------------------------------------------
// Mount/rider helpers
// --------------------------------------------------------------------------

// GetRider returns the character riding mount, or nil.
// C: src/utils.c:387-394 — if (mount && IS_NPC(mount) && IS_MOUNTED(mount))
//     return mount->master;
func (w *World) GetRider(mount *MobInstance) *Player {
	if mount == nil || !mount.IsNPC() || !mount.IsMountedMob() {
		return nil
	}

	if mount.MountRider == "" {
		return nil
	}

	rider, ok := w.GetPlayer(mount.MountRider)
	if !ok {
		return nil
	}
	return rider
}

// GetRiderName returns the name of the character riding mount, or "".
// Safe string-only variant that doesn't require a World reference.
func GetRiderName(mount *MobInstance) string {
	if mount == nil || !mount.IsNPC() || !mount.IsMountedMob() {
		return ""
	}
	return mount.MountRider
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// removeCharmAffect removes SPELL_CHARM (type 7) from ch's active affects if present.
func removeCharmAffect(ch *Player) {
	for i, aff := range ch.ActiveAffects {
		if aff.Source == "charm person" || aff.Source == "charm" || aff.ID == fmt.Sprintf("spell_%d", 7) {
			ch.ActiveAffects = append(ch.ActiveAffects[:i], ch.ActiveAffects[i+1:]...)
			return
		}
	}

	// Also try by Type if it maps to charm.
	for i, aff := range ch.ActiveAffects {
		if int(aff.Type) == 7 {
			ch.ActiveAffects = append(ch.ActiveAffects[:i], ch.ActiveAffects[i+1:]...)
			return
		}
	}
}
