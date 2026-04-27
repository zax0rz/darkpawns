package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// do_peek — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doPeek(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Peek at whom?\r\n")
		return true
	}

	if ch.Class != ClassThief && ch.Class != ClassAssassin && ch.Level < LVL_IMMORT {
		ch.SendMessage("You have no idea how to peek!\r\n")
		return true
	}

	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	percent := randRange(1, 101)
	skill := ch.GetSkill("peek")
	if percent > skill {
		ch.SendMessage(fmt.Sprintf("You try to peek at %s but fail.\r\n", victimPl.Name))
		return true
	}

	ch.SendMessage(fmt.Sprintf("You peek at %s's belongings:\r\n", victimPl.Name))
	ch.SendMessage("[Equipment and inventory]\r\n")
	// Improve skill
	skillVal := ch.GetSkill("peek")
	if skillVal > 0 && skillVal < 97 && randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
		skillVal += randRange(1, 3)
		if skillVal > 97 {
			skillVal = 97
		}
		ch.SetSkill("peek", skillVal)
		if randRange(1, 3) == 3 {
			ch.SendMessage("Your skill in peek improves.\r\n")
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// do_recall — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRecall(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Level > 5 {
		ch.SendMessage("You are too powerful to be teleported to the temple!\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("No way!  You are fighting!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil && (hasRoomFlag(room, "no_recall") || hasRoomFlag(room, "bfr")) {
		ch.SendMessage("You can't recall from this room!\r\n")
		return true
	}

	ch.SetPosition(combat.PosStanding)

	// Recall to temple (room 8004)
	recallRoom := 8004

	ch.SendMessage("You close your eyes and pray...\r\n")
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s closes %s eyes and prays...\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)

	ch.SendMessage("You are recalled!\r\n")
	ch.RoomVNum = recallRoom
	actToRoom(w, recallRoom, fmt.Sprintf("%s appears in the room.\r\n", ch.Name), "")

	return true
}

// ---------------------------------------------------------------------------
// do_stealth — from act.other.c (superior sneak)
// ---------------------------------------------------------------------------

func (w *World) doStealth(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't sneak around on a mount!\r\n")
		return true
	}

	skill := ch.GetSkill("stealth")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to become one with the shadows.\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill {
		ch.SendMessage("You try to become one with the shadows, but fail.\r\n")
		return true
	}

	ch.SetAffect(affSneak, true)
	ch.SendMessage("You become one with the shadows.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_appraise — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAppraise(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Appraise what?\r\n")
		return true
	}

	// Find object in inventory
	obj := w.findObjNear(ch, arg)
	if obj == nil {
		ch.SendMessage("You don't have that item.\r\n")
		return true
	}

	cost := obj.Prototype.Cost
	skill := ch.GetSkill("appraise")
	percent := randRange(1, 101)

	if percent > skill {
		// Failed appraise — random value
		badCost := cost + randRange(-cost, cost*2)
		if badCost < 0 {
			badCost = 0
		}
		ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", badCost))
		return true
	}

	// Successful appraise
	actual := cost + randRange(-20, 20)
	if actual < 0 {
		actual = 0
	}
	ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", actual))

	// Improve skill
	skillVal := ch.GetSkill("appraise")
	if skillVal > 0 && skillVal < 97 && randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
		skillVal += randRange(1, 3)
		if skillVal > 97 {
			skillVal = 97
		}
		ch.SetSkill("appraise", skillVal)
		if randRange(1, 3) == 3 {
			ch.SendMessage("Your skill in appraise improves.\r\n")
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// do_inactive — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doInactive(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Flags&(1<<PrfInactive) != 0 {
		ch.Flags &^= 1 << PrfInactive
		ch.SendMessage("You are now active.\r\n")
	} else {
		ch.Flags |= 1 << PrfInactive
		ch.SendMessage("You are now inactive.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_scout — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doScout(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	skill := ch.GetSkill("scout")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to scout.\r\n")
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Scout which direction?\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("You can't scout from in here!\r\n")
		return true
	}

	currentRoom := w.GetRoomInWorld(ch.GetRoomVNum())
	if currentRoom == nil {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}
	exitObj, exitOk := currentRoom.Exits[strings.ToLower(arg)]
	if !exitOk {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}

	toRoom := w.GetRoomInWorld(exitObj.ToRoom)
	if toRoom == nil {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}

	// Sector description
	sectorNames := map[int]string{
		0:  "the cobblestones of a city",
		1:  "a wide swath of field",
		2:  "the dense forest",
		3:  "high hills",
		4:  "jagged mountains",
		5:  "a large stretch of water",
		6:  "thin air",
		7:  "a murky swamp",
		8:  "the inside of a structure",
		9:  "a vast wasteland",
		10: "the watery depths",
		11: "the endless elemental plane",
	}

	sectorDesc, ok := sectorNames[toRoom.Sector]
	if !ok {
		sectorDesc = "the endless elemental plane"
	}

	ch.SendMessage(fmt.Sprintf("There is %s to the %s.\r\n", sectorDesc, arg))

	// Room flags
	if isDark(toRoom) {
		ch.SendMessage("It is dark in that direction.\r\n")
	}
	if hasRoomFlag(toRoom, "death") {
		ch.SendMessage("You sense certain death in that direction.\r\n")
	}
	if hasRoomFlag(toRoom, "tunnel") {
		ch.SendMessage("It looks narrow in that direction.\r\n")
	}

	// Count people
	players := w.GetPlayersInRoom(toRoom.VNum)
	mobs := w.GetMobsInRoom(toRoom.VNum)

	playerCount := 0
	for _, p := range players {
		if !p.IsNPC() {
			playerCount++
		}
	}

	totalCount := playerCount + len(mobs)
	if totalCount == 0 {
		ch.SendMessage("You see no one there.\r\n")
	} else if totalCount == 1 {
		ch.SendMessage("You see one being there.\r\n")
	} else if totalCount < 10 {
		ch.SendMessage(fmt.Sprintf("You see a group of %d beings there.\r\n", totalCount))
	} else {
		ch.SendMessage("You see a huge crowd there!\r\n")
	}

	return true
}

// ---------------------------------------------------------------------------
// do_roll — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRoll(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	maxRoll := 100
	if arg != "" {
// #nosec G104
		fmt.Sscanf(arg, "%d", &maxRoll)
		if maxRoll < 1 {
			maxRoll = 1
		}
	}

	result := randRange(1, maxRoll)
	ch.SendMessage(fmt.Sprintf("You roll %d out of %d.\r\n", result, maxRoll))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s rolls %d out of %d.\r\n", ch.Name, result, maxRoll), ch.Name)
	return true
}
