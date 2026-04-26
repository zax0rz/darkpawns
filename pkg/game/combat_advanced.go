package game

// combat_advanced.go — advanced combat: retreat, subdue, sleeper, neckbreak, ambush, startCombatBetween
//
// All rights reserved. See license.doc for complete information.
//
// Copyright (C) 1993, 94 by the Trustees of the Johns Hopkins University
// CircleMUD is based on DikuMUD, Copyright (C) 1990, 1991.
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// This includes all original code done for Dark Pawns MUD by other authors.
// All code is the intellectual property of the author, and is used here
// by permission.
//
// No original code may be duplicated, reused, or executed without the
// written permission of the author. All rights reserved.

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// do_retreat — retreat from combat (line 1001)
// ---------------------------------------------------------------------------

func (w *World) doRetreat(ch *Player, me *MobInstance, cmd string, arg string) bool {
	var capmsg, lowmsg string

	if ch.GetClass() == ClassNinja {
		capmsg = "Escape"
		lowmsg = "escape"
	} else {
		capmsg = "Retreat"
		lowmsg = "retreat"
	}

	if ch.GetPosition() < combat.PosFighting {
		ch.SendMessage("Get on your feet first!\r\n")
		return true
	}

	if ch.GetSkill(SkillEscape) == 0 && ch.GetSkill(SkillRetreat) == 0 {
		ch.SendMessage("Huh?\r\n")
		return true
	}

	if !ch.IsFighting() {
		ch.SendMessage(fmt.Sprintf("%s from what? You aren't fighting!\r\n", capmsg))
		return true
	}

	percent := randRange(1, 101)
	var prob int
	if ch.GetClass() == ClassNinja {
		prob = ch.GetSkill(SkillEscape)
	} else {
		prob = ch.GetSkill(SkillRetreat)
	}

	if percent > prob {
		ch.SendMessage(fmt.Sprintf("You try to %s but get cornered in the process!\r\n", lowmsg))
		improveSkill(ch, SkillEscape)
		return true
	}

	_ = arg
	dirs := []string{"north", "east", "south", "west", "up", "down"}
	for i := 0; i < 6; i++ {
		attempt := randRange(0, len(dirs)-1)
		room := w.GetRoomInWorld(ch.RoomVNum)
		if room == nil {
			continue
		}
		exit, hasExit := room.Exits[dirs[attempt]]
		if !hasExit || exit.ToRoom <= 0 {
			continue
		}
		if w.roomHasFlag(exit.ToRoom, "death") {
			continue
		}
		if exit.DoorState == 1 {
			continue
		}

		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s realizes it's a losing cause and gracefully attempts to %s.", ch.Name, lowmsg))
		if doSimpleMove(w, ch, attempt, true) {
			ch.SendMessage(fmt.Sprintf("You make a hasty %s.\r\n", lowmsg))
		} else {
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s is cornered and fails to %s!", ch.Name, lowmsg))
		}
		return true
	}

	ch.SendMessage(fmt.Sprintf("You are cornered and fail to %s!\r\n", lowmsg))
	w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s is cornered and fails to %s!", ch.Name, lowmsg))
	return true
}

// ---------------------------------------------------------------------------
// do_subdue — subdue opponent (line 1084)
// ---------------------------------------------------------------------------

func (w *World) doSubdue(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillSubdue) == 0 {
		ch.SendMessage("You have no idea how!\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You're too busy right now!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Subdue who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		// Check mobs
		for _, m := range w.GetMobsInRoom(ch.RoomVNum) {
			if strings.Contains(strings.ToLower(m.GetName()), strings.ToLower(args[0])) {
				ch.SendMessage("They aren't here.\r\n")
				return true
			}
		}
		ch.SendMessage("Subdue who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Aren't we funny today...\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You can't contemplate violence in such a place!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("You can't get close enough!\r\n")
		return true
	}

	if !vict.IsNPC() && !isOutlaw(ch) {
		ch.SendMessage("You can't subdue them because you are not an Outlaw!\r\n")
		vict.SendMessage(fmt.Sprintf("%s failed to subdue you because %s is not an Outlaw.\r\n", ch.Name, ch.Name))
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Haha.. Don't think so.\r\n")
		return true
	}

	if vict.GetPosition() <= combat.PosStunned {
		ch.SendMessage("What's the point of doing that now?\r\n")
		return true
	}

	percent := randRange(1, 101+vict.GetLevel())
	prob := ch.GetSkill(SkillSubdue)

	if ch.GetLevel() >= lvlImmort {
		percent = 0
	}
	if vict.IsNPC() && false {
		prob = 0
	}
	if !vict.IsNPC() && ch.GetLevel() < lvlImmort {
		if vict.GetLevel() > ch.GetLevel()+3 || vict.GetLevel() < ch.GetLevel()-3 {
			prob = 0
		}
	}

	levelDiff := vict.GetLevel() - ch.GetLevel()
	if levelDiff > 0 {
		percent += levelDiff
	}

	if percent > prob {
		// Failed
		if vict.IsNPC() && false {
			vict.SendMessage(fmt.Sprintf("%s misses a blow to the back of your head.\r\n", ch.Name))
			ch.SendMessage(fmt.Sprintf("%s avoids your misplaced blow.\r\n", vict.GetName()))
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s avoids %s's misplaced blow.", vict.GetName(), ch.Name))
		}
		w.startCombatBetween(vict, ch)
	} else {
		// Success
		vict.SendMessage("Someone sneaks up behind you and knocks you out!\r\n")
		ch.SendMessage(fmt.Sprintf("You knock %s out cold.\r\n", himHer(vict.GetSex())))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s knocks out %s with a well-placed blow to the back of the head.", ch.Name, vict.Name))
		vict.SetPosition(combat.PosStunned)
		improveSkill(ch, SkillSubdue)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_sleeper — sleeper hold/choke (line 1184)
// ---------------------------------------------------------------------------

func (w *World) doSleeper(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillSleeper) == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You can't do this while fighting!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("This room just has such a peaceful, easy feeling...\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded != nil {
		ch.SendMessage("You can't get a good grip on them while you are holding that weapon!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Sleeper who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("Sleeper who?\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Can't get to sleep fast enough, huh?\r\n")
		return true
	}

	if !vict.IsNPC() && !isOutlaw(ch) {
		ch.SendMessage(fmt.Sprintf("You cannot sleeper them because you are not an Outlaw!\r\n"))
		vict.SendMessage(fmt.Sprintf("%s failed to sleeper you because %s is not an Outlaw.\r\n", ch.Name, ch.Name))
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("You can't get a good grip on them while they're fighting!\r\n")
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Ha Ha. Don't think so.\r\n")
		return true
	}

	if vict.GetPosition() <= combat.PosSleeping {
		ch.SendMessage("What's the point of doing that now?\r\n")
		return true
	}

	percent := randRange(1, 101+vict.GetLevel())
	prob := ch.GetSkill(SkillSleeper)

	if vict.IsNPC() && (false) {
		prob = 0
	}
	if !vict.IsNPC() && ch.GetLevel() < lvlImmort {
		if vict.GetLevel() > ch.GetLevel()+3 || vict.GetLevel() < ch.GetLevel()-3 {
			prob = 0
		}
	}

	levelDiff := vict.GetLevel() - ch.GetLevel()
	if levelDiff > 0 {
		percent += levelDiff
	}

	if percent > prob {
		// Failed
		ch.SendMessage(fmt.Sprintf("You try to grab %s in a sleeper hold but fail!", vict.GetName()))
		vict.SendMessage(fmt.Sprintf("%s tries to put a sleeper hold on you, but you break free!", ch.Name))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s tries to put %s in a sleeper hold...", ch.Name, vict.Name))
		w.startCombatBetween(vict, ch)
	} else {
		// Success
		ch.SendMessage(fmt.Sprintf("You put %s in a sleeper hold.", vict.GetName()))
		vict.SendMessage("You feel very sleepy... Zzzzz..")
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s puts %s in a sleeper hold.", ch.Name, vict.Name))
		vict.SetPosition(combat.PosSleeping)

		// Add AFF_SLEEP affect
		duration := ch.GetLevel() / 9
		if duration < 1 {
			duration = 1
		}
		vict.SetAffect(affSleep, true)

		improveSkill(ch, SkillSleeper)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_neckbreak — neck break (line 1295)
// ---------------------------------------------------------------------------

func (w *World) doNeckbreak(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillNeckbreak) == 0 {
		ch.SendMessage("What's that, idiot-san?\r\n")
		return true
	}

	wielded := w.GetEquipped(ch, eqWearWield)
	if wielded != nil {
		ch.SendMessage("You can't do this and wield a weapon at the same time!\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("I don't see them here.\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("I don't see them here.\r\n")
		return true
	}

	if isShopkeeper(w, vict) {
		ch.SendMessage("Haha.. Don't think so.\r\n")
		return true
	}

	if vict == ch {
		ch.SendMessage("Aren't we funny today...\r\n")
		return true
	}

	if w.roomHasFlag(ch.RoomVNum, "peaceful") {
		ch.SendMessage("You can't contemplate violence in such a place!\r\n")
		return true
	}

	if isMounted(ch) {
		ch.SendMessage("Dismount first!\r\n")
		return true
	}

	neededMoves := 51
	if ch.GetMove() < neededMoves {
		ch.SendMessage("You haven't the energy to do this!\r\n")
		return true
	}
	ch.SetMove(ch.GetMove() - neededMoves)

	// Calculate percentage based on AC
	// In C: ((7 - (GET_AC(vict) / 10)) << 1) + number(1, 101)
	ac := vict.GetAC()
	percent := ((7 - (ac / 10)) << 1) + randRange(1, 101)
	prob := ch.GetSkill(SkillNeckbreak)

	if percent > prob {
		// Failed
		ch.SendMessage(fmt.Sprintf("You try to break %s's neck, but %s is too strong!", himHer(vict.GetSex()), vict.GetName()))
		vict.SendMessage(fmt.Sprintf("%s tries to break your neck, but can't!", ch.Name))
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s tries to break %s's neck, but %s slips free!", ch.Name, vict.Name, vict.Name))
		w.startCombatBetween(vict, ch)
	} else {
		// Success - damage based on 18d(level)
		dam := diceRoll(18, ch.GetLevel())
		w.doDamage(ch, vict, dam, SkillNeckbreak)
		improveSkill(ch, SkillNeckbreak)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_ambush — ambush (line 1454)
// ---------------------------------------------------------------------------

func (w *World) doAmbush(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetSkill(SkillAmbush) == 0 {
		ch.SendMessage("You'd better not.\r\n")
		return true
	}

	args := strings.Fields(arg)
	if len(args) < 1 {
		ch.SendMessage("Ambush who?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, args[0])
	if vict == nil {
		ch.SendMessage("Ambush who?\r\n")
		return true
	}

	if ch.IsAffected(affCharm) {
		ch.SendMessage("You are a little busy for that right now!\r\n")
		return true
	}

	if ch == vict {
		ch.SendMessage("Ambush yourself? You idiot!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.RoomVNum); sector := 0; if room != nil { sector = room.Sector }
	if sector != 3 && sector != 4 &&
		sector != 5 && sector != 1 {
		ch.SendMessage("Ambush someone here? Impossible!\r\n")
		return true
	}

	if vict.IsFighting() {
		ch.SendMessage("They're too alert for that, currently.\r\n")
		return true
	}

	ch.SendMessage("You crouch in the shadows and plan your ambush...\r\n")

	// In the original C code, this creates an event callback that fires
	// after PULSE_VIOLENCE*2. For now, we execute the ambush immediately
	// with the success/failure check inline.

	percent := randRange(1, 131)
	prob := ch.GetSkill(SkillAmbush)

	if vict.IsNPC() && false {
		percent = 200
	}

	if percent > prob {
		// Failure — do 0 damage to trigger awareness
		w.doDamage(ch, vict, 0, SkillAmbush)
	} else {
		// Success
		dam := ch.GetDamroll()

		wielded := w.GetEquipped(ch, eqWearWield)
		if wielded != nil {
			diceNum := wielded.Prototype.Values[0]
			diceSize := wielded.Prototype.Values[1]
			dam += diceRoll(diceNum, diceSize)
		}

		// Bonus damage: level * 2.6
		dam += int(float64(ch.GetLevel()) * 2.6)

		// 10% more damage if hidden
		if ch.IsAffected(affHide) {
			dam += int(float64(dam) * 0.10)
		}

		w.doDamage(ch, vict, dam, SkillAmbush)
		improveSkill(ch, SkillAmbush)
	}
	return true
}

// ---------------------------------------------------------------------------
// startCombatBetween — initiate combat between two characters (C: set_fighting)
// ---------------------------------------------------------------------------

// startCombatBetween initiates combat between ch and vict.
// If vict is nil, it just sets ch as fighting (for aggressive mobs attacking players).
func (w *World) startCombatBetween(ch, vict interface{}) {
	// Try to handle *Player vs *Player or *Player vs *MobInstance
	if chPlayer, ok := ch.(*Player); ok {
		if victPlayer, ok2 := vict.(*Player); ok2 {
			chPlayer.SetFighting(victPlayer.Name)
			victPlayer.SetFighting(chPlayer.Name)
			// Broadcast the fight start
			w.roomMessage(chPlayer.RoomVNum, fmt.Sprintf("%s starts fighting %s!\r\n", chPlayer.Name, victPlayer.Name))
		} else if victMob, ok2 := vict.(*MobInstance); ok2 {
			chPlayer.SetFighting(victMob.GetName())
			w.roomMessage(chPlayer.RoomVNum, fmt.Sprintf("%s starts fighting %s!\r\n", chPlayer.Name, victMob.GetName()))
		} else {
			// Only one party — probably mob attacking player
			chPlayer.SetFighting("someone")
		}
	} else if chMob, ok := ch.(*MobInstance); ok {
		if victPlayer, ok2 := vict.(*Player); ok2 {
			chMob.SetFighting(victPlayer.Name)
			victPlayer.SetFighting(chMob.GetName())
		}
	}
}
