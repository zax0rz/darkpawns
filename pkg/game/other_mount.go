package game

import (
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// do_ride — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRide(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You're too busy fighting!\n\r")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("Go outside if you want to ride!\r\n")
		return true
	}

	arg, _ = OneArgument(arg)
	if arg == "" {
		ch.SendMessage("What do you wish to ride?\r\n")
		return true
	}

	// Find mount
	mountPlayer, mountMob := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if mountPlayer == nil && mountMob == nil {
		ch.SendMessage("No-one by that name here.\r\n")
		return true
	}
	if mountPlayer != nil {
		if mountPlayer == ch {
			ch.SendMessage("That's disgusting!\r\n")
		} else {
			Act(w, true, ch, mountPlayer, nil, nil, "You can't ride $N!", "", ToChar)
		}
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't ride two beasts at once!\r\n")
		return true
	}

	if mountMob.IsAffected(affMounted) {
		ch.SendMessage("The beast is already being ridden!\r\n")
		return true
	}
	if !mountMob.HasFlag("mountable") {
		Act(w, true, ch, mountMob, nil, nil, "You can't ride $N!", "", ToChar)
		return true
	}
	if ch.IsAffected(affCharm) {
		ch.SendMessage("Get your master's permission first!\r\n")
		return true
	}
	if mountMob.IsAffected(affCharm) && mountMob.GetFollowing() != ch.Name {
		Act(w, true, ch, mountMob, nil, nil, "$S master would not like that!", "", ToChar)
		return true
	}

	mountMob.SetMountRider(ch.Name)
	mountMob.SetAffected(affMounted)
	if !mountMob.IsAffected(affCharm) {
		mountMob.SetAffected(affCharm)
	}
	mountMob.SetFollowingBody(ch)
	ch.MountName = mountMob.GetName()
	ch.SetAffect(affMounted, true)
	ch.SendMessage("You hop on your mount.\r\n")
	Act(w, true, ch, mountMob, nil, nil, "$n hops on your back!", "", ToVict)
	Act(w, true, ch, mountMob, nil, nil, "$n hops onto the back of $N.", "", ToRoom)
	return true
}

// doRideMob ports do_ride (src/act.other.c:1545-1595) for a mobile rider
// reached through command_interpreter (order/force, or a switched body). Every
// refusal is TO_CHAR to the rider: a descriptor-less mobile drops it, but a
// switched body has the descriptor and must see it. CAN_MOUNT(ch) is
// !IS_MOB(ch) && !IS_MOUNTED(ch) (src/utils.h:364-366), so a mobile rider can
// never complete the mount and always reaches "You can't ride!".
func (w *World) doRideMob(rider *MobInstance, mountName string) {
	if rider == nil || !rider.IsNPC() {
		return
	}
	if rider.IsFighting() {
		Act(w, false, rider, nil, nil, nil, "You're too busy fighting!", "", ToChar)
		return
	}
	room := w.GetRoomInWorld(rider.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		Act(w, false, rider, nil, nil, nil, "Go outside if you want to ride!", "", ToChar)
		return
	}
	name, _ := OneArgument(mountName)
	if name == "" {
		Act(w, false, rider, nil, nil, nil, "What do you wish to ride?", "", ToChar)
		return
	}
	target, found := w.resolveCharInRoomAt(rider, rider.GetRoomVNum(), name)
	if !found {
		Act(w, false, rider, nil, nil, nil, "No-one by that name here.", "", ToChar)
		return
	}
	mountPlayer, mountMob := target.Player, target.Mob
	if mountMob == rider {
		Act(w, false, rider, nil, nil, nil, "That's disgusting!", "", ToChar)
		return
	}
	if mountPlayer != nil {
		Act(w, true, rider, mountPlayer, nil, nil, "You can't ride $N!", "", ToChar)
		return
	}
	if rider.IsAffected(affMounted) {
		Act(w, false, rider, nil, nil, nil, "You can't ride two beasts at once!", "", ToChar)
		return
	}
	if mountMob.IsAffected(affMounted) {
		Act(w, false, rider, nil, nil, nil, "The beast is already being ridden!", "", ToChar)
		return
	}
	if !mountMob.HasFlag("mountable") {
		Act(w, true, rider, mountMob, nil, nil, "You can't ride $N!", "", ToChar)
		return
	}
	// CAN_MOUNT(mobile) is false, so the final refusal is "You can't ride!".
	Act(w, false, rider, nil, nil, nil, "You can't ride!", "", ToChar)
}

// ---------------------------------------------------------------------------
// do_dismount — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doDismount(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}
	w.doDismountActor(ch)
	return true
}

// doDismountActor implements the Lua binding's do_dismount call for the real
// actor body, including descriptorless mobile riders. C sends the private
// lines to the actor/mount and the room act to peers, then unmounts affect bits
// only; master/follower identity is not changed by utils.c:378-385.
func (w *World) doDismountActor(rider Actor) {
	if !actorHasMountAffect(rider) {
		rider.SendMessage("You need to be riding before you can dismount!\r\n")
		return
	}
	rider.SendMessage("You hop off your mount.\r\n")
	mount := w.mountedFollowerFor(rider)
	if mount != nil {
		Act(w, true, rider, mount, nil, nil, "$n dismounts from the back of $N.", "", ToRoom)
		mount.SendMessage("Your rider dismounts, whew!\r\n")
	}
	w.unmountActor(rider)
}

// unmountActor follows utils.c:378-385: clear AFF_MOUNT on the supplied
// rider and the first mounted mobile follower, retaining master/follower
// links and the Go-side display names that identify those links.
func (w *World) unmountActor(rider Actor) {
	mount := w.mountedFollowerFor(rider)
	clearMountedAffects(rider, mount)
}

func actorHasMountAffect(actor Actor) bool {
	switch actor := actor.(type) {
	case *Player:
		return actor != nil && actor.IsAffected(affMounted)
	case *MobInstance:
		return actor != nil && actor.IsAffected(affMounted)
	default:
		return false
	}
}

func (w *World) mountedFollowerFor(rider Actor) *MobInstance {
	body, ok := rider.(combat.Combatant)
	if !ok || body == nil || !actorHasMountAffect(rider) {
		return nil
	}
	w.mu.RLock()
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, mob := range w.activeMobs {
		mobs = append(mobs, mob)
	}
	w.mu.RUnlock()
	var followers []*MobInstance
	for _, mob := range mobs {
		if mob.IsAffected(affMounted) && w.combatFollowingBody(mob) == body {
			followers = append(followers, mob)
		}
	}
	sort.SliceStable(followers, func(i, j int) bool {
		return actorFollowingSequence(followers[i]) > actorFollowingSequence(followers[j])
	})
	if len(followers) == 0 {
		return nil
	}
	return followers[0]
}

// ---------------------------------------------------------------------------
// do_yank — from act.other.c:1620-1662
// ---------------------------------------------------------------------------

// doYank ports C do_yank byte-for-byte. Yank a sitting follower to their feet
// (e.g. after a trip). Branch order, strings, and the "wierd"/"is is" typos
// match C exactly (R1/R4). Victim can be a player OR a charmed mob follower
// (C's get_char_room_vis resolves both). The act() success/already-up lines
// use $M/$S pronoun expansion (not the victim's name).
func (w *World) doYank(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	// No argument — act.other.c:1628
	if arg == "" {
		ch.SendMessage("Who do you wish to yank?\r\n")
		return true
	}

	// Target lookup (get_char_room_vis — resolves players AND mobs).
	victimPl, victimMob := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil && victimMob == nil {
		// NOPERSON global — act.other.c:1630 (config.c:93). The exact C bytes,
		// not an invented variant (R4/DP-1200).
		ch.SendMessage("No-one by that name here.\r\n")
		return true
	}

	// Extract the victim's attributes into a uniform shape (player or mob) and
	// keep the victim as an Act() target: C's do_yank hands act() ch and victim
	// and lets it render $N/$M/$S and the TO_CHAR/TO_VICT/TO_NOTVICT audiences.
	var (
		victFollowing string
		victPos       int
		victMounted   bool
		victimActor   Actor
		setVictimPos  func(int)
	)
	switch {
	case victimPl != nil:
		victFollowing = victimPl.GetFollowing()
		victPos = victimPl.GetPosition()
		victMounted = victimPl.IsMounted()
		victimActor = victimPl
		setVictimPos = func(pos int) { victimPl.SetPosition(pos) }
	case victimMob != nil:
		victFollowing = victimMob.GetFollowing()
		victPos = victimMob.GetPosition()
		// Mobs are never "mounted" in C's IS_MOUNTED sense (a mount is a mob a
		// player rides, not the reverse); the mount sub-branch never fires here.
		victMounted = false
		victimActor = victimMob
		setVictimPos = func(pos int) { victimMob.SetPosition(pos) }
	}

	// Self-check — act.other.c:1632 (the victim is the actor, by identity
	// like C's victim == ch, not by name). Comes BEFORE the follower check
	// in C.
	if victimPl == ch {
		ch.SendMessage("That's wierd.\r\n") // sic: "wierd" (R1 — keep the typo)
		return true
	}

	// Follower check — act.other.c:1636. C: victim->master != ch.
	if victFollowing != ch.Name {
		ch.SendMessage("That probably wouldn't be appreciated.\r\n")
		return true
	}

	// Already up — act.other.c:1641. C: GET_POS(victim) > POS_SITTING (so
	// FIGHTING and STANDING both count as up). NOT >= PosStanding.
	if victPos > combat.PosSitting {
		if !victMounted {
			Act(w, true, ch, victimActor, nil, nil, "$N is already on $S feet.", "", ToChar)
		} else {
			Act(w, true, ch, victimActor, nil, nil, "You can't yank $M off $S mount!", "", ToChar)
		}
		return true
	}

	// Sleeping/below — act.other.c:1650. C: GET_POS(victim) <= POS_SLEEPING.
	if victPos <= combat.PosSleeping {
		// sic: "is is" (R1 — keep the typo)
		Act(w, true, ch, victimActor, nil, nil, "$N is is no position to be yanked around!", "", ToChar)
		return true
	}

	// Success — act.other.c:1656-1658. C's three act() calls; TO_NOTVICT
	// excludes the actor AND the victim (actDeliver's ToNotVict arm).
	Act(w, true, ch, victimActor, nil, nil, "You yank $M to $S feet.", "", ToChar)
	Act(w, true, ch, victimActor, nil, nil, "$n yanks you to your feet.", "", ToVict)
	Act(w, true, ch, victimActor, nil, nil, "$n yanks $N to $S feet.", "", ToNotVict)
	setVictimPos(combat.PosStanding)
	return true
}
