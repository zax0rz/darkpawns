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

// doRideMob reaches C's CAN_MOUNT(mob) refusal. NPCs have no descriptor, so
// the final "You can't ride!" send_to_char has no player-visible recipient;
// the source path creates no mount or follower state for an NPC rider.
func (w *World) doRideMob(rider *MobInstance, mountName string) {
	if rider == nil || !rider.IsNPC() {
		return
	}
	// src/utils.h:364-366 defines CAN_MOUNT(ch) as !IS_MOB(ch) &&
	// !IS_MOUNTED(ch). In do_ride's branch order this point is reached only
	// after resolving a mountable target and passing both mounted checks.
	// The Lua caller already supplies that target handle; the NPC actor cannot
	// pass CAN_MOUNT, and C's attempted private message is descriptorless.
	_ = mountName
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

	// Extract the victim's attributes into a uniform shape (player or mob).
	var (
		victName      string
		victSex       int
		victFollowing string
		victPos       int
		victMounted   bool
		tellVictim    func(string)
		setVictimPos  func(int)
	)
	switch {
	case victimPl != nil:
		victName = victimPl.Name
		victSex = victimPl.GetSex()
		victFollowing = victimPl.GetFollowing()
		victPos = victimPl.GetPosition()
		victMounted = victimPl.IsMounted()
		tellVictim = func(msg string) { victimPl.SendMessage(msg) }
		setVictimPos = func(pos int) { victimPl.SetPosition(pos) }
	case victimMob != nil:
		victName = victimMob.GetName()
		victSex = victimMob.GetSex()
		victFollowing = victimMob.GetFollowing()
		victPos = victimMob.GetPosition()
		// Mobs are never "mounted" in C's IS_MOUNTED sense (a mount is a mob a
		// player rides, not the reverse); the mount sub-branch never fires here.
		victMounted = false
		tellVictim = func(msg string) { victimMob.SendMessage(msg) }
		setVictimPos = func(pos int) { victimMob.SetPosition(pos) }
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(victName, victSex)

	// Self-check — act.other.c:1632 (the victim is the actor). Comes BEFORE the
	// follower check in C.
	if victName == ch.Name {
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
			ch.SendMessage(ActMessage("$N is already on $S feet.", chPronouns, &victPronouns, "") + "\r\n")
		} else {
			ch.SendMessage(ActMessage("You can't yank $M off $S mount!", chPronouns, &victPronouns, "") + "\r\n")
		}
		return true
	}

	// Sleeping/below — act.other.c:1650. C: GET_POS(victim) <= POS_SLEEPING.
	if victPos <= combat.PosSleeping {
		// sic: "is is" (R1 — keep the typo)
		ch.SendMessage(ActMessage("$N is is no position to be yanked around!", chPronouns, &victPronouns, "") + "\r\n")
		return true
	}

	// Success — act.other.c:1656-1659. The three act() lines, then POS_STANDING.
	// $M/$S expand to the victim's objective/possessive pronouns (him/her/it,
	// his/her/its), NOT the name.
	ch.SendMessage(ActMessage("You yank $M to $S feet.", chPronouns, &victPronouns, "") + "\r\n")
	tellVictim(ActMessage("$n yanks you to your feet.", chPronouns, &victPronouns, "") + "\r\n")
	actToRoom(w, ch.GetRoomVNum(), ActMessage("$n yanks $N to $S feet.", chPronouns, &victPronouns, "")+"\r\n", ch.Name)
	setVictimPos(combat.PosStanding)
	return true
}
