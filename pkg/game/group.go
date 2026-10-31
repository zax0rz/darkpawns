package game

import "strings"

// GroupedActor reports C's IS_AFFECTED(victim, AFF_GROUP) for either body. A
// player also carries the port's InGroup mirror, which the group commands set
// alongside the affect.
func GroupedActor(victim Actor) bool {
	switch ch := victim.(type) {
	case *Player:
		return ch.IsAffected(AffGroup) || ch.IsInGroup()
	case *MobInstance:
		return ch.IsAffected(AffGroup)
	default:
		return false
	}
}

// SetGroupedActor sets or clears AFF_GROUP on either body.
func SetGroupedActor(victim Actor, grouped bool) {
	switch ch := victim.(type) {
	case *Player:
		ch.SetInGroup(grouped)
		ch.SetAffect(AffGroup, grouped)
	case *MobInstance:
		if grouped {
			ch.SetAffected(AffGroup)
		} else {
			ch.RemoveAffected(AffGroup)
		}
	}
}

// PerformGroup ports C's perform_group (src/act.other.c:624-635). It returns one
// only when the victim newly receives AFF_GROUP, which is do_group's "found"
// count.
func (w *World) PerformGroup(leader, victim Actor) int {
	if leader == nil || victim == nil || GroupedActor(victim) || !CanSee(leader, victim) {
		return 0
	}
	SetGroupedActor(victim, true)
	if victim != leader {
		Act(nil, false, leader, victim, nil, nil,
			"$N is now a member of your group.", "", ToChar)
	}
	Act(nil, false, leader, victim, nil, nil,
		"You are now a member of $n's group.", "", ToVict)
	Act(w, false, leader, victim, nil, nil,
		"$N is now a member of $n's group.", "", ToNotVict)
	return 1
}

// GroupCommand ports C's do_group (src/act.other.c:685-740) for any body.
//
// C hands the line to command_interpreter whichever body runs it, so a mobile
// forced or ordered to `group` reaches the same code as a player. The port had a
// session-only implementation, which is why `force <mob> group <player>` was
// accepted and then dropped (F9).
//
// It returns false for the empty argument, which is C's print_group(ch) branch;
// that listing is session-owned, so the caller renders it.
func (w *World) GroupCommand(leader Actor, argument string) bool {
	if leader == nil {
		return true
	}
	arg, _ := OneArgument(argument)
	if arg == "" {
		return false
	}

	// Must have no master to enrol others — src/act.other.c:699.
	if actorFollowingName(leader) != "" {
		Act(nil, false, leader, nil, nil, nil,
			"You can not enroll group members without being head of a group.", "", ToChar)
		return true
	}

	// "group all" — src/act.other.c:706-717.
	if strings.EqualFold(arg, "all") {
		w.PerformGroup(leader, leader)
		found := 0
		for _, follower := range w.followersOfInRoom(leader) {
			if isShadowingBody(follower) {
				continue
			}
			found += w.PerformGroup(leader, follower)
		}
		if found == 0 {
			Act(nil, false, leader, nil, nil, nil,
				"Everyone following you here is already in your group.", "", ToChar)
		}
		return true
	}

	// Single target — src/act.other.c:719-738.
	victim := w.resolveRoomActor(leader, arg)
	if victim == nil {
		Act(nil, false, leader, nil, nil, nil, NoPersonHere, "", ToChar)
		return true
	}
	// vict->master != ch: identity, not a name, because two NPCs can share one.
	if !w.FollowsBody(victim, leader) && victim != leader {
		Act(nil, false, leader, victim, nil, nil,
			"$N must follow you to enter your group.", "", ToChar)
		return true
	}
	if !GroupedActor(victim) {
		w.PerformGroup(leader, victim)
		return true
	}
	if victim != leader {
		Act(nil, false, leader, victim, nil, nil,
			"$N is no longer a member of your group.", "", ToChar)
	}
	Act(nil, false, leader, victim, nil, nil,
		"You have been kicked out of $n's group!", "", ToVict)
	Act(w, false, leader, victim, nil, nil,
		"$N has been kicked out of $n's group!", "", ToNotVict)
	SetGroupedActor(victim, false)
	return true
}

// actorFollowingName is the string form of C's ch->master, used only for the
// "do you already have a master" gate.
func actorFollowingName(a Actor) string {
	switch v := a.(type) {
	case *Player:
		return v.GetFollowing()
	case *MobInstance:
		return v.GetFollowing()
	}
	return ""
}

// FollowsBody reports whether victim follows leader, by body identity: C's
// ch->master is a pointer, so two same-named NPCs are two different masters
// (R3). SetFollowingBody records the retained body for exactly this reason.
func (w *World) FollowsBody(victim, leader Actor) bool {
	switch v := victim.(type) {
	case *Player:
		return w.FollowingActor(v) == leader
	case *MobInstance:
		return w.FollowingActor(v) == leader
	}
	return false
}

// followersOfInRoom is do_group "all"'s walk: the leader's followers filtered to
// those standing in its room, in list order (src/act.other.c:709-712).
func (w *World) followersOfInRoom(leader Actor) []Actor {
	var inRoom []Actor
	for _, follower := range w.FollowersOf(leader) {
		if follower.GetRoom() == leader.GetRoom() {
			inRoom = append(inRoom, follower)
		}
	}
	return inRoom
}

// resolveRoomActor is C's get_char_room_vis for any body, which do_group uses.
func (w *World) resolveRoomActor(leader Actor, name string) Actor {
	switch v := leader.(type) {
	case *Player:
		if target, ok := w.resolveCharInRoomAt(v, v.GetRoom(), name); ok {
			return asActor(target.Combatant)
		}
	case *MobInstance:
		if target, ok := w.resolveCharInRoomAt(v, v.GetRoom(), name); ok {
			return asActor(target.Combatant)
		}
	}
	return nil
}

// isShadowingBody is C's IS_SHADOWING(victim): AFF_DODGE, which do_follow's
// quiet path sets.
func isShadowingBody(a Actor) bool {
	switch v := a.(type) {
	case *Player:
		return v.IsAffected(affDodge)
	case *MobInstance:
		return v.IsAffected(affDodge)
	}
	return false
}
