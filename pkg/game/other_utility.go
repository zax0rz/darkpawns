package game

import (
	"fmt"
	"log/slog"
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

	// C do_peek (act.other.c:1665): the class gate runs BEFORE the no-arg path
	// — a non-thief/assassin mortal is rejected with "You're not a thief!"
	// regardless of arguments.
	if ch.Class != ClassThief && ch.Class != ClassAssassin && ch.GetLevel() < LVL_IMMORT {
		ch.SendMessage("You're not a thief!\r\n")
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Whom do you wish to peek at?\r\n")
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

	// List equipment
	ch.SendMessage("[Equipment]\r\n")
	for slotID := 0; slotID < int(SlotMax); slotID++ {
		slot := EquipmentSlot(slotID)
		item, ok := victimPl.Equipment.GetItemInSlot(slot)
		if ok && item != nil && item.Prototype != nil {
			ch.SendMessage(fmt.Sprintf("  %s\r\n", item.Prototype.ShortDesc))
		}
	}

	// List inventory
	ch.SendMessage("[Inventory]\r\n")
	for _, item := range victimPl.Inventory.Items {
		if item != nil {
			name := "a generic object"
			if item.Prototype != nil {
				name = item.Prototype.ShortDesc
			}
			ch.SendMessage(fmt.Sprintf("  %s\r\n", name))
		}
	}
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
// do_recall — from act.other.c:1727-1748, with spell_recall (spells.c:124-163)
// inlined; the command is always self-targeted (spell_recall(30, ch, ch, ...)),
// so spell_recall's BFR/FIGHTING re-checks are subsumed by the gates below.
// ---------------------------------------------------------------------------

func (w *World) doRecall(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.GetLevel() > 5 {
		ch.SendMessage("This command is not available for someone of your experience!\r\n")
		return true
	}

	// Only ROOM_BFR (bit 17) blocks by room; C has no no_recall room flag.
	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil && hasRoomFlag(room, "bfr") {
		ch.SendMessage("You can't recall from this magickal place.\r\n")
		return true
	}

	if ch.IsFighting() {
		// C sends this one without a trailing CRLF (act.other.c:1742).
		ch.SendMessage("Your concentration is broken by your fighting!")
		return true
	}

	// "$n disappears." to the old room; the recaller sees nothing.
	Act(w, true, ch, nil, nil, nil, "$n disappears.", "", ToRoom)

	// Target keys on GET_HOME (config.c): 2 = kiroshi_start_room,
	// 3 = alaozar_start_room, else mortal_start_room — same 2/3/else
	// hometown split as quit's isokquit.
	recallRoom := MortalStartRoom
	switch ch.GetHometown() {
	case 2:
		recallRoom = 18201
	case 3:
		recallRoom = 21258
	}

	// C unmounts in place and the mount stays behind; unmount before the
	// transfer so CharTransfer's mount-follow does not bring it along.
	if ch.IsMounted() {
		Unmount(ch, w.GetMount(ch))
		ch.SetAffect(affMounted, false)
	}

	if err := w.PlayerTransfer(ch, recallRoom); err != nil {
		slog.Error("recall transfer failed", "player", ch.Name, "target", recallRoom, "error", err)
		return true
	}

	Act(w, true, ch, nil, nil, nil, "$n appears in the middle of the room.", "", ToRoom)

	// AWAKE(victim): the recaller sees only the new room's look output.
	if ch.GetPosition() > combat.PosSleeping {
		w.lookAtRoom(ch, false)
	} else {
		ch.SendMessage("You have a strange dream about falling..\r\n")
	}

	return true
}

// ExecuteWordOfRecall ports spell_recall (spells.c:124-165) — the SPELL
// surface, reached by reciting a word-of-recall scroll or casting the spell.
// Its gate bytes differ from the recall COMMAND above ("Your magic ebbs..."
// to the victim vs "You can't recall from this magickal place."), and the
// BFR check covers the caster's room too. The spells package calls this
// through a narrow interface so it cannot import game.
func (w *World) ExecuteWordOfRecall(ch, victim interface{}) {
	v, ok := victim.(*Player)
	if !ok {
		return
	}
	c, cok := ch.(*Player)
	if !cok {
		return
	}

	// ROOM_BFR on either the caster's or the victim's room dissolves the
	// spell (message to the VICTIM).
	for _, who := range []*Player{c, v} {
		if room := w.GetRoomInWorld(who.GetRoomVNum()); room != nil && hasRoomFlag(room, "bfr") {
			v.SendMessage("Your magic ebbs and dissolves as you lose your concentration.\r\n")
			return
		}
	}
	if c.IsFighting() {
		c.SendMessage("Your concentration is broken by your fighting!\r\n")
		return
	}

	Act(w, true, v, nil, nil, nil, "$n disappears.", "", ToRoom)

	recallRoom := MortalStartRoom
	switch v.GetHometown() {
	case 2:
		recallRoom = 18201
	case 3:
		recallRoom = 21258
	}

	// C unmounts in place and the mount stays behind; unmount before the
	// transfer so CharTransfer's mount-follow does not bring it along.
	if v.IsMounted() {
		Unmount(v, w.GetMount(v))
		v.SetAffect(affMounted, false)
	}

	if err := w.PlayerTransfer(v, recallRoom); err != nil {
		slog.Error("word-of-recall transfer failed", "player", v.Name, "target", recallRoom, "error", err)
		return
	}

	Act(w, true, v, nil, nil, nil, "$n appears in the middle of the room.", "", ToRoom)
	if v.GetPosition() > combat.PosSleeping {
		w.lookAtRoom(v, false)
	} else {
		v.SendMessage("You have a strange dream about falling..\r\n")
	}
}

// ---------------------------------------------------------------------------
// do_appraise — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAppraise(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// C do_appraise (act.other.c:1786) draws number(1,101) at the TOP, before
	// the no-arg/obj checks — so even a rejection consumes one stream draw. Move
	// it here for draw parity (R3a); randRange is the shared dprng stream.
	percent := randRange(1, 101)

	arg, _ = halfChop(arg)
	if arg == "" {
		ch.SendMessage("What do you want to appraise?\r\n")
		return true
	}

	// C passes ch->carrying to get_obj_in_list_vis, so equipped and room
	// objects are intentionally outside this command's lookup scope.
	obj := getObjInInvVis(ch, arg)
	if obj == nil {
		ch.SendMessage("You don't seem to have one of those...\r\n")
		return true
	}

	cost := obj.GetCost()
	skill := ch.GetSkill(SkillAppraise)

	if percent > skill {
		// Failed appraise — random value. C's number() helper is inclusive;
		// for the success arm below, costs under 20 use a zero lower bound.
		cost += randRange(-cost, cost*2)
		ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", cost))
		return true
	}

	// Successful appraise.
	low := -20
	if cost <= 20 {
		low = 0
	}
	cost += randRange(low, 20)
	ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", cost))
	ImproveSkill(ch, SkillAppraise)

	return true
}

// ---------------------------------------------------------------------------
// do_inactive — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doInactive(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.GetFlags()&(1<<PrfInactive) != 0 {
		ch.SetPlrFlag(PrfInactive, false)
	} else {
		ch.SetPlrFlag(PrfInactive, true)
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

	// C do_scout (new_cmds.c): no-arg → valid-direction-word → skill gate →
	// outside → exit-exists → execute. Ordering and reject messages are
	// byte-faithful to C. The skill gate is LATE (after the direction check)
	// but it IS a hard reject — a no-skill scouter gets "You have no idea how!"
	// before the outside/exit checks, regardless of location — so it must be
	// repositioned to match C, never removed.
	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Scout where?\r\n")
		return true
	}

	dir := directionIndex(arg)
	if dir < 0 {
		ch.SendMessage("Scout in which direction?\r\n")
		return true
	}

	if ch.GetSkill("scout") <= 0 {
		ch.SendMessage("You have no idea how!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("You can only do this outdoors.\r\n")
		return true
	}

	exitObj, exitOk := room.Exits[dirs[dir]]
	if !exitOk {
		ch.SendMessage("There is nothing of interest there.\r\n")
		return true
	}

	toRoom := w.GetRoomInWorld(exitObj.ToRoom)
	if toRoom == nil {
		ch.SendMessage("There is nothing of interest there.\r\n")
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
		if _, err := fmt.Sscanf(arg, "%d", &maxRoll); err != nil {
			ch.SendMessage("That doesn't look like a number.\r\n")
			slog.Warn("roll parse failed", "player", ch.Name, "arg", arg, "error", err)
			return true
		}
		if maxRoll < 1 {
			maxRoll = 1
		}
	}

	result := randRange(1, maxRoll)
	// C do_roll (act.other.c:1942): "You roll %u (1-%u)." + act TO_ROOM.
	ch.SendMessage(fmt.Sprintf("You roll %d (1-%d).\r\n", result, maxRoll))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("With a toss of the dice, %s rolls %d (1-%d).\r\n", ch.Name, result, maxRoll), ch.Name)
	return true
}
