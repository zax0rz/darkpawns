package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func movementSendToChar(ch *Player, message string) {
	Act(nil, false, ch, nil, nil, nil, strings.TrimSuffix(message, "\r\n"), "", ToChar|ToSleep)
}

// DoStand implements the C position transition, including stand-as-dismount.
func (w *World) DoStand(ch *Player) {
	switch ch.GetPosition() {
	case combat.PosStanding:
		if ch.IsMounted() {
			w.dismountForStand(ch)
		} else {
			movementSendToChar(ch, "You are already standing.\r\n")
		}
	case combat.PosSitting:
		movementSendToChar(ch, "You stand up.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n clambers to $s feet.", "", ToRoom)
		ch.SetPosition(combat.PosStanding)
	case combat.PosResting:
		movementSendToChar(ch, "You stop resting, and stand up.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n stops resting, and clambers on $s feet.", "", ToRoom)
		ch.SetPosition(combat.PosStanding)
	case combat.PosSleeping:
		movementSendToChar(ch, "You have to wake up first!\r\n")
	case combat.PosFighting:
		movementSendToChar(ch, "Do you not consider fighting as standing?\r\n")
	default:
		movementSendToChar(ch, "You stop floating around, and put your feet on the ground.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n stops floating around, and puts $s feet on the ground.", "", ToRoom)
		ch.SetPosition(combat.PosStanding)
	}
}

func (w *World) dismountForStand(ch *Player) {
	mount := w.GetMount(ch)
	movementSendToChar(ch, "You hop off your mount.\r\n")
	if mount != nil {
		Act(w, true, ch, mount, nil, nil, "$n dismounts from the back of $N.", "", ToRoom)
		mount.SendMessage("Your rider dismounts, whew!\r\n")
	}
	Unmount(ch, mount)
	ch.SetAffect(affMounted, false)
	if mount != nil && (strings.EqualFold(ch.GetFollowing(), mount.GetName()) ||
		strings.EqualFold(ch.GetFollowing(), mount.GetShortDesc())) {
		ch.SetFollowing("")
	}
}

// DoSit implements do_sit from act.movement.c.
func (w *World) DoSit(ch *Player) {
	switch ch.GetPosition() {
	case combat.PosStanding:
		if ch.IsMounted() {
			movementSendToChar(ch, "You can't rest while mounted.\r\n")
			return
		}
		movementSendToChar(ch, "You sit down.\r\n")
		Act(w, false, ch, nil, nil, nil, "$n sits down.", "", ToRoom)
		ch.SetPosition(combat.PosSitting)
	case combat.PosSitting:
		movementSendToChar(ch, "You're sitting already.\r\n")
	case combat.PosResting:
		movementSendToChar(ch, "You stop resting, and sit up.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n stops resting.", "", ToRoom)
		ch.SetPosition(combat.PosSitting)
	case combat.PosSleeping:
		movementSendToChar(ch, "You have to wake up first.\r\n")
	case combat.PosFighting:
		movementSendToChar(ch, "Sit down while fighting? are you MAD?\r\n")
	default:
		movementSendToChar(ch, "You stop floating around, and sit down.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n stops floating around, and sits down.", "", ToRoom)
		ch.SetPosition(combat.PosSitting)
	}
}

// DoRest implements do_rest from act.movement.c.
func (w *World) DoRest(ch *Player) {
	switch ch.GetPosition() {
	case combat.PosStanding:
		if ch.IsMounted() {
			movementSendToChar(ch, "You can't rest while mounted.\r\n")
			return
		}
		movementSendToChar(ch, "You sit down and rest your tired bones.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n sits down and rests.", "", ToRoom)
		ch.SetPosition(combat.PosResting)
	case combat.PosSitting:
		movementSendToChar(ch, "You rest your tired bones.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n rests.", "", ToRoom)
		ch.SetPosition(combat.PosResting)
	case combat.PosResting:
		movementSendToChar(ch, "You are already resting.\r\n")
	case combat.PosSleeping:
		movementSendToChar(ch, "You have to wake up first.\r\n")
	case combat.PosFighting:
		movementSendToChar(ch, "Rest while fighting?  Are you MAD?\r\n")
	default:
		movementSendToChar(ch, "You stop floating around, and stop to rest your tired bones.\r\n")
		Act(w, false, ch, nil, nil, nil, "$n stops floating around, and rests.", "", ToRoom)
		ch.SetPosition(combat.PosSitting)
	}
}

// DoSleep implements do_sleep from act.movement.c.
func (w *World) DoSleep(ch *Player) {
	switch ch.GetPosition() {
	case combat.PosStanding:
		if ch.IsMounted() {
			movementSendToChar(ch, "You can't rest while mounted.\r\n")
			return
		}
		fallthrough
	case combat.PosSitting, combat.PosResting:
		movementSendToChar(ch, "You go to sleep.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n lies down and falls asleep.", "", ToRoom)
		ch.SetPosition(combat.PosSleeping)
	case combat.PosSleeping:
		movementSendToChar(ch, "You are already sound asleep.\r\n")
	case combat.PosFighting:
		movementSendToChar(ch, "Sleep while fighting?  Are you MAD?\r\n")
	default:
		movementSendToChar(ch, "You stop floating around, and lie down to sleep.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n stops floating around, and lie down to sleep.", "", ToRoom)
		ch.SetPosition(combat.PosSleeping)
	}
}

// DoWake implements self and targeted wake behavior, including magical sleep.
func (w *World) DoWake(ch *Player, argument string) {
	argument = firstWord(strings.TrimSpace(argument))
	if argument != "" {
		if ch.GetPosition() == combat.PosSleeping {
			movementSendToChar(ch, "Maybe you should wake yourself up first.\r\n")
			return
		}

		var target Actor
		if strings.EqualFold(argument, ch.Name) || strings.EqualFold(argument, "self") || strings.EqualFold(argument, "me") {
			target = ch
		} else if resolved, ok := w.ResolveCharInRoom(ch, argument); ok {
			target = asActor(resolved.Combatant)
		}
		if target == nil {
			movementSendToChar(ch, "No-one by that name here.\r\n")
			return
		}
		if target != ch {
			switch {
			case target.GetPosition() > combat.PosSleeping:
				Act(nil, false, ch, target, nil, nil, "$E is already awake.", "", ToChar)
			case actorIsAffected(target, affSleep):
				Act(nil, false, ch, target, nil, nil, "You can't wake $M up!", "", ToChar)
			case target.GetPosition() < combat.PosSleeping:
				Act(nil, false, ch, target, nil, nil, "$E's in pretty bad shape!", "", ToChar)
			default:
				Act(nil, false, ch, target, nil, nil, "You wake $M up.", "", ToChar)
				Act(w, false, ch, target, nil, nil, "$n wakes up $N.", "", ToNotVict)
				// Make the victim awake before rendering their message so PERS($n)
				// resolves the waker just as C's TO_SLEEP wake delivery does.
				setActorPosition(target, combat.PosSitting)
				Act(nil, false, ch, target, nil, nil, "You are awakened by $n.", "", ToVict|ToSleep)
			}
			return
		}
	}

	if ch.IsAffected(affSleep) {
		movementSendToChar(ch, "You can't wake up!\r\n")
		Act(w, true, ch, nil, nil, nil, "$n tosses and turns uncomfortably.", "", ToRoom)
	} else if ch.GetPosition() > combat.PosSleeping {
		movementSendToChar(ch, "You are already awake...\r\n")
	} else {
		movementSendToChar(ch, "You awaken, and sit up.\r\n")
		Act(w, true, ch, nil, nil, nil, "$n awakens.", "", ToRoom)
		ch.SetPosition(combat.PosSitting)
	}
}

func actorIsAffected(actor Actor, affect int) bool {
	switch target := actor.(type) {
	case *Player:
		return target.IsAffected(affect)
	case *MobInstance:
		return target.IsAffected(affect)
	default:
		return false
	}
}

func setActorPosition(actor Actor, position int) {
	switch target := actor.(type) {
	case *Player:
		target.SetPosition(position)
	case *MobInstance:
		target.SetPosition(position)
	}
}

// DoFollow implements C do_follow. quiet preserves the structural shadow path;
// applying SKILL_SHADOW/AFF_DODGE remains a skill-system TODO.
func (w *World) DoFollow(ch *Player, argument string, quiet bool) {
	argument = firstWord(strings.TrimSpace(argument))
	if argument == "" {
		movementSendToChar(ch, "Whom do you wish to follow?\r\n")
		return
	}

	var leader Actor
	var leaderPlayer *Player
	if strings.EqualFold(argument, ch.Name) || strings.EqualFold(argument, "self") || strings.EqualFold(argument, "me") {
		leader = ch
		leaderPlayer = ch
	} else if resolved, ok := w.ResolveCharInRoom(ch, argument); ok {
		leader = asActor(resolved.Combatant)
		leaderPlayer = resolved.Player
	}
	if leader == nil {
		movementSendToChar(ch, "No-one by that name here.\r\n")
		return
	}

	if strings.EqualFold(ch.GetFollowing(), leader.GetName()) {
		Act(nil, false, ch, leader, nil, nil, "You are already following $M.", "", ToChar)
		return
	}
	if ch.IsAffected(affCharm) && ch.GetFollowing() != "" {
		master := w.followingActor(ch.GetFollowing())
		Act(nil, false, ch, master, nil, nil, "But you only feel like following $N!", "", ToChar)
		return
	}

	if leader == ch {
		if ch.GetFollowing() == "" {
			movementSendToChar(ch, "You are already following yourself.\r\n")
			return
		}
		StopFollower(w, ch)
		return
	}

	if leaderPlayer != nil && CircleFollow(w, ch, leaderPlayer) {
		movementSendToChar(ch, "Sorry, but following in loops is not allowed.\r\n")
		return
	}
	if ch.GetFollowing() != "" {
		StopFollower(w, ch)
	}
	ch.SetInGroup(false)
	ch.SetAffect(affGroup, false)

	if quiet {
		// TODO(DP-shadow): apply SKILL_SHADOW/AFF_DODGE when the skill domain
		// exposes the C success roll here.
		Act(nil, false, ch, leader, nil, nil, "You now follow $N.", "", ToChar)
		ch.SetFollowing(leader.GetName())
		return
	}
	if leaderPlayer != nil {
		AddFollower(w, ch, leaderPlayer)
		return
	}

	ch.SetFollowing(leader.GetName())
	Act(nil, false, ch, leader, nil, nil, "You now follow $N.", "", ToChar)
	if canSee(leader, ch) && leader.GetPosition() > combat.PosSleeping {
		Act(nil, true, ch, leader, nil, nil, "$n starts following you.", "", ToVict)
	}
	Act(w, true, ch, leader, nil, nil, "$n starts to follow $N.", "", ToNotVict)
}

func (w *World) followingActor(name string) Actor {
	if player, ok := w.GetPlayer(name); ok {
		return player
	}
	w.mu.RLock()
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, mob := range w.activeMobs {
		mobs = append(mobs, mob)
	}
	w.mu.RUnlock()
	for _, mob := range mobs {
		if strings.EqualFold(mob.GetName(), name) || strings.EqualFold(mob.GetShortDesc(), name) {
			return mob
		}
	}
	return nil
}

// DoEnter implements named-door and automatic indoor entry.
func (w *World) DoEnter(ch *Player, argument string) MoveResult {
	argument = firstWord(strings.TrimSpace(argument))
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return MoveResult{}
	}
	if argument != "" {
		for dir, direction := range dirs {
			exit, ok := room.Exits[direction]
			if ok && exit.Keywords != "" && strings.EqualFold(strings.TrimSpace(exit.Keywords), argument) {
				return w.moveByIndex(ch, dir, true)
			}
		}
		movementSendToChar(ch, "There is no "+argument+" here.\r\n")
		return MoveResult{}
	}
	if movementRoomHasFlag(room, roomFlagIndoors, "indoors") {
		movementSendToChar(ch, "You are already indoors.\r\n")
		return MoveResult{}
	}
	for dir, direction := range dirs {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom < 0 || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		if destination := w.GetRoomInWorld(exit.ToRoom); movementRoomHasFlag(destination, roomFlagIndoors, "indoors") {
			return w.moveByIndex(ch, dir, true)
		}
	}
	movementSendToChar(ch, "You can't seem to find anything to enter.\r\n")
	return MoveResult{}
}

// DoLeave finds the first open exit from an indoor room to outdoors.
func (w *World) DoLeave(ch *Player) MoveResult {
	room := w.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return MoveResult{}
	}
	if !movementRoomHasFlag(room, roomFlagIndoors, "indoors") {
		movementSendToChar(ch, "You are outside.. where do you want to go?\r\n")
		return MoveResult{}
	}
	for dir, direction := range dirs {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom < 0 || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		if destination := w.GetRoomInWorld(exit.ToRoom); destination != nil && !movementRoomHasFlag(destination, roomFlagIndoors, "indoors") {
			return w.moveByIndex(ch, dir, true)
		}
	}
	movementSendToChar(ch, "I see no obvious exits to the outside.\r\n")
	return MoveResult{}
}

func (w *World) moveByIndex(ch *Player, dir int, needSpecialsCheck bool) MoveResult {
	result := MoveResult{}
	result.Success = performMoveResult(w, ch, dir, needSpecialsCheck, &result)
	if result.Success {
		result.NewRoomVNum = ch.GetRoom()
	}
	return result
}
