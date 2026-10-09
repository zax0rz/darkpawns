// Package game — follower management.
//
// Ported from src/utils.c:
//   add_follower(), add_follower_quiet(), stop_follower(), die_follower(),
//   circle_follow(), get_mount(), get_rider()
//
// In the original C code, followers are stored as a linked list on the
// leader (ch->followers). In this Go port, followers are identified by
// the string field Following (player.Following / mob.GetFollowing()), which
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
	"github.com/zax0rz/darkpawns/pkg/engine"
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
		if cur.GetFollowing() == "" {
			return false
		}
		next, ok := w.GetPlayer(cur.GetFollowing())
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
	ch.SetFollowingBody(leader)
}

// AddFollowerQuietMob adds a mob as a follower of a player (charmed pet, etc.)
// without sending messages.
// C: src/utils.c:463-475
func AddFollowerQuietMob(mob *MobInstance, leader *Player) {
	mob.SetFollowingBody(leader)
}

// AddFollowerMob adds a mob as a follower with C's visible follower notices.
// C add_follower sends TO_VICT and TO_NOTVICT act() messages; its TO_CHAR
// message targets the NPC itself and is not player-visible.
func AddFollowerMob(w *World, mob *MobInstance, leader *Player) {
	mob.SetFollowingBody(leader)
	Act(w, true, mob, leader, nil, nil, "$n starts following you.", "", ToVict)
	Act(w, true, mob, leader, nil, nil, "$n starts to follow $N.", "", ToNotVict)
}

// AddFollower adds ch as a follower of leader with notifications.
// Caller must verify no follow loop exists first (use CircleFollow).
// C: src/utils.c:480-498
func AddFollower(w *World, ch *Player, leader *Player) {
	ch.SetFollowingBody(leader)

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
	if ch.GetFollowing() == "" {
		return
	}

	// Look up the leader for act messages that need $N.
	leader := asActor(w.combatFollowingBody(ch))

	// C computes IS_SHADOWING from AFF_DODGE, removes the SKILL_SHADOW
	// affect/bit before choosing the stop message, and suppresses leader/room
	// notices for this branch (utils.c:400-420; R1/R5e).
	shadowing := ch.IsAffected(affDodge)
	if shadowing {
		ch.RemoveAffectBySpell(skillNumShadow)
		ch.RemoveAffectBit(affDodge)
	}
	charmAffected := ch.IsAffected(affCharm)

	if charmAffected {
		Act(w, false, ch, leader, nil, nil,
			"You realize that $N is a jerk!", "", ToChar)
		Act(w, true, ch, leader, nil, nil,
			"$n realizes that $N is a jerk!", "", ToNotVict)
		if leader != nil {
			Act(w, true, ch, leader, nil, nil,
				"$n hates your guts!", "", ToVict)
		}
		// Remove SPELL_CHARM from active affects if present.
		removeCharmAffect(ch)
	} else if shadowing {
		Act(w, false, ch, leader, nil, nil,
			"You stop shadowing $N.", "", ToChar)
	} else {
		Act(w, false, ch, leader, nil, nil,
			"You stop following $N.", "", ToChar)
		Act(w, true, ch, leader, nil, nil,
			"$n stops following $N.", "", ToNotVict)
		if leader != nil && canSee(leader, ch) && leader.GetPosition() > combat.PosSleeping {
			Act(w, true, ch, leader, nil, nil,
				"$n stops following you.", "", ToVict)
		}
	}

	// Unmount if this is a mount.
	// In the C code: if (IS_NPC(ch) && IS_MOUNTED(ch)) unmount(get_rider(ch), ch);
	// Players can't be mounts in Go (only mobs can), so we skip this for players.

	// Clear the following field — this is the Go equivalent of removing ch from
	// the leader's follower list.
	ch.SetFollowing("")

	ch.SetAffect(affCharm, false)
	if ch.IsInGroup() {
		ch.SetInGroup(false)
	}
	ch.SetAffect(affGroup, false)
}

// StopFollowerMob removes a mob from its master's follower list.
// C: src/utils.c:397-440 — a charmed mob (mount, pet) denounces its master
// with the "jerk"/"hates your guts!" act trio before the relation drops; the
// plain branch still tells the room and master that following stopped.
func StopFollowerMob(w *World, mob *MobInstance) {
	if mob.GetFollowing() == "" {
		return
	}

	leader := asActor(w.combatFollowingBody(mob))

	if mob.IsAffected(affCharm) {
		Act(w, false, mob, leader, nil, nil,
			"You realize that $N is a jerk!", "", ToChar)
		Act(w, true, mob, leader, nil, nil,
			"$n realizes that $N is a jerk!", "", ToNotVict)
		if leader != nil {
			Act(w, true, mob, leader, nil, nil,
				"$n hates your guts!", "", ToVict)
		}
	} else {
		Act(w, false, mob, leader, nil, nil,
			"You stop following $N.", "", ToChar)
		Act(w, true, mob, leader, nil, nil,
			"$n stops following $N.", "", ToNotVict)
		if leader != nil && canSee(leader, mob) && leader.GetPosition() > combat.PosSleeping {
			Act(w, true, mob, leader, nil, nil,
				"$n stops following you.", "", ToVict)
		}
	}

	// Unmount if this mob is a mount (C: IS_NPC && IS_MOUNTED).
	if mob.IsMountedMob() {
		if rider := w.GetRider(mob); rider != nil {
			Unmount(rider, mob)
		} else {
			mob.SetMountRider("")
		}
	}

	// C clears only the master relation. The AFF_CHARM bit survives
	// stop_follower for ride-charm (C's affect_from_char(SPELL_CHARM) is a
	// no-op there — the bit was set directly by do_ride), which is what keeps
	// an attacked mount unrideable afterwards ("$S master would not like
	// that!"). Go's mob bitmask cannot separate spell-charm from ride-charm,
	// so the bit stays, matching the mounted surface.
	mob.SetFollowing("")
}

// --------------------------------------------------------------------------
// Die follower — cleanup when a character dies
// --------------------------------------------------------------------------

// DieFollower cleans up follower relations when ch dies.
// C: src/utils.c:447-457 — if ch->master, stop_follower(ch);
//
//	then for each k in ch->followers, stop_follower(k->follower)
func (w *World) DieFollower(ch *Player) {
	// If ch is following someone, stop following.
	if ch.GetFollowing() != "" {
		StopFollower(w, ch)
	}

	// Find all followers of ch and make them stop following.
	for _, p := range w.GetAllPlayers() {
		if w.combatFollowingBody(p) == ch {
			StopFollower(w, p)
		}
	}

	// Also check mob followers (charmed pets following this player).
	for _, mob := range w.GetAllMobs() {
		if w.combatFollowingBody(mob) == ch {
			StopFollowerMob(w, mob)
		}
	}
}

// DieFollowerMob cleans up follower relations when a mob dies.
// C: src/utils.c:447-457
func (w *World) DieFollowerMob(mob *MobInstance) {
	// If mob is following someone, stop following.
	if mob.GetFollowing() != "" {
		StopFollowerMob(w, mob)
	}

	// If mob is being ridden, dismount the rider.
	riderName := mob.GetMountRider()
	if riderName != "" {
		if rider, ok := w.GetPlayer(riderName); ok {
			rider.MountName = ""
		}
		mob.SetMountRider("")
	}

	// Retained leader references distinguish same-description NPC leaders.
	for _, p := range w.GetAllPlayers() {
		if w.combatFollowingBody(p) == mob {
			StopFollower(w, p)
		}
	}
	for _, follower := range w.GetAllMobs() {
		if w.combatFollowingBody(follower) == mob {
			StopFollowerMob(w, follower)
		}
	}
}

// --------------------------------------------------------------------------
// Mount/rider helpers
// --------------------------------------------------------------------------

// GetRider returns the character riding mount, or nil.
// C: src/utils.c:387-394 — if (mount && IS_NPC(mount) && IS_MOUNTED(mount))
//
//	return mount->master;
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

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// removeCharmAffect removes SPELL_CHARM (type 7) from ch's active affects if present.
func removeCharmAffect(ch *Player) {
	ch.mu.Lock()
	removed := false
	defer func() {
		if removed {
			ch.affectTotalLocked()
		}
		ch.mu.Unlock()
		if removed {
			ch.refreshAttributeCapacity()
		}
	}()

	for i, aff := range ch.ActiveAffects {
		if aff.Source == "charm person" || aff.Source == "charm" || aff.ID == fmt.Sprintf("spell_%d", 7) {
			ch.ActiveAffects = append(ch.ActiveAffects[:i], ch.ActiveAffects[i+1:]...)
			removed = true
			return
		}
	}

	// Also try by flag if it's a charm affect.
	for i, aff := range ch.ActiveAffects {
		if aff.Flags&engine.AFFCharm != 0 {
			ch.ActiveAffects = append(ch.ActiveAffects[:i], ch.ActiveAffects[i+1:]...)
			removed = true
			return
		}
	}
}

// SetFollowingBody retains a command-selected leader, including duplicate NPCs.
// The string remains the existing social display; combat never resolves an NPC by it.
func (p *Player) SetFollowingBody(body combat.Combatant) {
	name := ""
	if body != nil {
		name = body.GetName()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Following = name
	p.followingBody = body
	if body == nil {
		p.followingSequence = 0
	} else {
		p.followingSequence = nextFollowerSequence()
	}
}
