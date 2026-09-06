package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func movementSendToChar(ch *Player, message string) {
	Act(nil, false, ch, nil, nil, nil, strings.TrimSuffix(message, "\r\n"), "", ToChar|ToSleep)
}

type positionCommandArm struct {
	self     string
	room     string
	hideRoom bool
	next     int
}

type positionCommandSpec struct {
	arms            [combat.PosStanding + 1]positionCommandArm
	defaultArm      positionCommandArm
	mountedDismount bool
	mountedMessage  string
}

var positionCommandSpecs = [...]positionCommandSpec{
	{
		mountedDismount: true,
		arms: [combat.PosStanding + 1]positionCommandArm{
			combat.PosStanding: {self: "You are already standing."},
			combat.PosSitting:  {self: "You stand up.", room: "$n clambers to $s feet.", hideRoom: true, next: combat.PosStanding},
			combat.PosResting:  {self: "You stop resting, and stand up.", room: "$n stops resting, and clambers on $s feet.", hideRoom: true, next: combat.PosStanding},
			combat.PosSleeping: {self: "You have to wake up first!"},
			combat.PosFighting: {self: "Do you not consider fighting as standing?"},
		},
		defaultArm: positionCommandArm{
			self:     "You stop floating around, and put your feet on the ground.",
			room:     "$n stops floating around, and puts $s feet on the ground.",
			hideRoom: true,
			next:     combat.PosStanding,
		},
	},
	{
		mountedMessage: "You can't rest while mounted.",
		arms: [combat.PosStanding + 1]positionCommandArm{
			combat.PosStanding: {self: "You sit down.", room: "$n sits down.", next: combat.PosSitting},
			combat.PosSitting:  {self: "You're sitting already."},
			combat.PosResting:  {self: "You stop resting, and sit up.", room: "$n stops resting.", hideRoom: true, next: combat.PosSitting},
			combat.PosSleeping: {self: "You have to wake up first."},
			combat.PosFighting: {self: "Sit down while fighting? are you MAD?"},
		},
		defaultArm: positionCommandArm{
			self:     "You stop floating around, and sit down.",
			room:     "$n stops floating around, and sits down.",
			hideRoom: true,
			next:     combat.PosSitting,
		},
	},
	{
		mountedMessage: "You can't rest while mounted.",
		arms: [combat.PosStanding + 1]positionCommandArm{
			combat.PosStanding: {self: "You sit down and rest your tired bones.", room: "$n sits down and rests.", hideRoom: true, next: combat.PosResting},
			combat.PosSitting:  {self: "You rest your tired bones.", room: "$n rests.", hideRoom: true, next: combat.PosResting},
			combat.PosResting:  {self: "You are already resting."},
			combat.PosSleeping: {self: "You have to wake up first."},
			combat.PosFighting: {self: "Rest while fighting?  Are you MAD?"},
		},
		defaultArm: positionCommandArm{
			self: "You stop floating around, and stop to rest your tired bones.",
			room: "$n stops floating around, and rests.",
			next: combat.PosSitting,
		},
	},
	{
		mountedMessage: "You can't rest while mounted.",
		arms: [combat.PosStanding + 1]positionCommandArm{
			combat.PosStanding: {self: "You go to sleep.", room: "$n lies down and falls asleep.", hideRoom: true, next: combat.PosSleeping},
			combat.PosSitting:  {self: "You go to sleep.", room: "$n lies down and falls asleep.", hideRoom: true, next: combat.PosSleeping},
			combat.PosResting:  {self: "You go to sleep.", room: "$n lies down and falls asleep.", hideRoom: true, next: combat.PosSleeping},
			combat.PosSleeping: {self: "You are already sound asleep."},
			combat.PosFighting: {self: "Sleep while fighting?  Are you MAD?"},
		},
		defaultArm: positionCommandArm{
			self:     "You stop floating around, and lie down to sleep.",
			room:     "$n stops floating around, and lie down to sleep.",
			hideRoom: true,
			next:     combat.PosSleeping,
		},
	},
}

func (w *World) doPositionCommand(ch *Player, spec positionCommandSpec) {
	position := ch.GetPosition()
	if position == combat.PosStanding && ch.IsMounted() {
		if spec.mountedDismount {
			w.dismountForStand(ch)
		} else {
			movementSendToChar(ch, spec.mountedMessage)
		}
		return
	}

	arm := spec.defaultArm
	if position >= 0 && position < len(spec.arms) && spec.arms[position].self != "" {
		arm = spec.arms[position]
	}
	movementSendToChar(ch, arm.self)
	if arm.room != "" {
		Act(w, arm.hideRoom, ch, nil, nil, nil, arm.room, "", ToRoom)
		ch.SetPosition(arm.next)
	}
}

// DoStand implements the C position transition, including stand-as-dismount.
func (w *World) DoStand(ch *Player) { w.doPositionCommand(ch, positionCommandSpecs[0]) }

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
func (w *World) DoSit(ch *Player) { w.doPositionCommand(ch, positionCommandSpecs[1]) }

// DoRest implements do_rest from act.movement.c.
func (w *World) DoRest(ch *Player) { w.doPositionCommand(ch, positionCommandSpecs[2]) }

// DoSleep implements do_sleep from act.movement.c.
func (w *World) DoSleep(ch *Player) { w.doPositionCommand(ch, positionCommandSpecs[3]) }

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

const skillNumShadow = 181

// DoFollow implements C do_follow. The quiet path is the registered shadow
// subcommand (act.movement.c:923-951); its skill draw and affect are kept here
// so the shared follow state machine remains in C order (R1/R3/R5e).
func (w *World) DoFollow(ch *Player, argument string, quiet bool) {
	argument, _ = oneArgument(argument)
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
		// C evaluates the skill draw before the immortal shortcut, so immortals
		// consume the same number(0,101) draw even though they always succeed.
		// #nosec G404 — game RNG, not cryptographic
		if ch.GetSkill(SkillShadow) > dprng.Number(0, 101) || ch.GetLevel() >= LVL_IMMORT {
			// C IS_SHADOWING is AFF_DODGE, and a successful re-shadow first
			// removes the prior SKILL_SHADOW affect and its bit.
			if ch.IsAffected(affDodge) {
				ch.RemoveAffectBySpell(skillNumShadow)
				ch.RemoveAffectBit(affDodge)
			}
			ch.AddAffect(engine.NewAffectDirect(
				skillNumShadow,
				engine.ApplyNone,
				ch.GetLevel(),
				0,
				engine.AFFDodge,
				SkillShadow,
			))
			ch.SetAffect(affDodge, true)
			Act(nil, false, ch, leader, nil, nil, "You now follow $N.", "", ToChar)
			ch.SetFollowing(leader.GetName())
			return
		}
		// C's failed quiet roll falls through to add_follower, including its
		// leader and room audience messages.
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
	argument, _ = oneArgument(argument)
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
