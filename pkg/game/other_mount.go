package game

import (
	"fmt"
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

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Ride what?\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You are fighting!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("You can't ride in here!\r\n")
		return true
	}

	// Find mount
	_, mountMob := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if mountMob == nil {
		ch.SendMessage("There's nothing here to ride!\r\n")
		return true
	}

	if !mountMob.HasFlag("mountable") {
		ch.SendMessage("You can't ride that!\r\n")
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You're already riding!\r\n")
		return true
	}

	// For now, check if mount already has a rider by checking affMounted
	// We can check via a simple loop
	mobs := w.GetMobsInRoom(ch.GetRoomVNum())
	mountAlreadyRidden := false
	for _, m := range mobs {
		if m.HasFlag("mount") && m.HasFlag("mounted") {
			mountAlreadyRidden = true
			break
		}
	}
	_ = mountAlreadyRidden

	ch.SetAffect(affMounted, true)
	ch.Following = mountMob.GetShortDesc()
	ch.SendMessage(fmt.Sprintf("You climb onto %s.\r\n", mountMob.GetShortDesc()))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s climbs onto %s.\r\n", ch.Name, mountMob.GetShortDesc()), ch.Name)
	return true
}

// ---------------------------------------------------------------------------
// do_dismount — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doDismount(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if !ch.IsAffected(affMounted) {
		ch.SendMessage("You need to be riding before you can dismount!\r\n")
		return true
	}

	ch.SetAffect(affMounted, false)
	ch.SendMessage("You dismount.\r\n")
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s dismounts.\r\n", ch.Name), ch.Name)
	return true
}

// ---------------------------------------------------------------------------
// do_yank — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doYank(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Yank whom from what?\r\n")
		return true
	}

	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	// Must be a follower
	if victimPl.Following != ch.Name {
		ch.SendMessage("They aren't following you!\r\n")
		return true
	}

	if victimPl.GetPosition() >= combat.PosStanding {
		ch.SendMessage("They're already on their feet.\r\n")
		return true
	}

	victimPl.SetPosition(combat.PosStanding)
	ch.SendMessage(fmt.Sprintf("You yank %s to %s feet.\r\n", victimPl.Name, hisHer(victimPl.GetSex())))
	victimPl.SendMessage(fmt.Sprintf("%s yanks you to your feet.\r\n", ch.Name))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s yanks %s to %s feet.\r\n", ch.Name, victimPl.Name, hisHer(victimPl.GetSex())), ch.Name)
	return true
}
