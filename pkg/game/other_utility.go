package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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

	first, _ := oneArgument(arg)
	if first == "" {
		ch.SendMessage("Whom do you wish to peek at?\r\n")
		return true
	}

	victim, ok := w.ResolveCharInRoom(ch, first)
	if !ok {
		ch.SendMessage("No-one by that name here.\r\n")
		return true
	}
	if victim.Player == ch {
		ch.SendMessage("Try the 'inventory' command!\r\n")
		return true
	}

	// C skips the skill roll entirely for immortals. Mortals burn exactly one
	// number(1,101) draw before comparing their peek skill.
	if ch.GetLevel() < LVL_IMMORT && dprng.Number(1, 101) > ch.GetSkill(SkillPeek) {
		// The failure path calls do_look with the original, unparsed argument;
		// retain its ordinary look parser and player-facing bytes.
		w.doLook(ch, nil, "look", arg)
		lookedAt, ok := w.lookCharacterTarget(ch, arg)
		if ok && sameCharTarget(lookedAt, victim) && victim.Mob != nil {
			w.KenderSteal(ch, victim.Mob)
		}
		// C look_at_target emits the observer notifications after look_at_char.
		// Peek's successful path calls look_at_char directly and does not emit
		// these notifications; only the failed do_look vehicle reaches them.
		var target Actor
		if victim.Player != nil {
			target = victim.Player
		} else {
			target = victim.Mob
		}
		if ok && sameCharTarget(lookedAt, victim) && target != nil {
			if canSee(target, ch) {
				Act(w, true, ch, target, nil, nil, "$n looks at you.", "", ToVict)
			}
			Act(w, true, ch, target, nil, nil, "$n looks at $N.", "", ToNotVict)
		}
		return true
	}

	// C look_at_char emits the ordinary character description, condition,
	// equipment, and class-authorized inventory probe in that order. Render it
	// before improve_skill, which is the final call in the C handler.
	var result ObservationResult
	w.appendCharacterLook(&result, ch, victim)
	w.RenderObservationMessages(result)
	if victim.Mob != nil {
		w.KenderSteal(ch, victim.Mob)
	}
	ImproveSkill(ch, SkillPeek)

	return true
}

// lookCharacterTarget mirrors do_look's half_chop dispatch for the character
// notification that follows look_at_target. In particular, a failed peek
// passes the original argument to do_look: "peek the name" therefore looks
// at the literal target "the", not at the already-resolved peek victim.
func (w *World) lookCharacterTarget(ch *Player, arg string) (CharTarget, bool) {
	first, rest := splitArg(strings.ToLower(arg))
	switch {
	case first == "", isAbbrev(first, "in"), directionIndex(first) >= 0:
		return CharTarget{}, false
	case isAbbrev(first, "at"):
		return w.ResolveCharInRoom(ch, rest)
	default:
		return w.ResolveCharInRoom(ch, first)
	}
}

func sameCharTarget(left, right CharTarget) bool {
	return left.Player != nil && left.Player == right.Player || left.Mob != nil && left.Mob == right.Mob
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
	directionArg := extractArg(arg)
	if directionArg == "" {
		ch.SendMessage("Scout where?\r\n")
		return true
	}

	dir := directionIndex(directionArg)
	if dir < 0 {
		ch.SendMessage("Scout in which direction?\r\n")
		return true
	}

	if ch.GetSkill("scout") <= 0 {
		ch.SendMessage("You have no idea how!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || (hasRoomFlag(room, "indoors") && room.Sector == 0) {
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

	// C's do_scout switch uses the raw sector constants from structs.h.
	sectorDesc := scoutTerrain(toRoom.Sector)
	ch.SendMessage(fmt.Sprintf("You see %s to the %s.\r\n", sectorDesc, dirList[dir]))

	// Room flags
	if w.IsRoomDark(toRoom.VNum) {
		ch.SendMessage("It looks pretty dark there.\r\n")
	}
	if hasRoomFlag(toRoom, "death") {
		ch.SendMessage("You sense that it is not safe to travel there.\r\n")
	}
	if hasRoomFlag(toRoom, "tunnel") {
		ch.SendMessage("It looks like a very narrow passage.\r\n")
	}

	if len(w.GetItemsInRoom(toRoom.VNum)) > 0 {
		ch.SendMessage("It looks like there is something on the ground there.\r\n")
	}
	peopleCount := len(w.GetPlayersInRoom(toRoom.VNum)) + len(w.GetMobsInRoom(toRoom.VNum))
	if peopleCount > 0 {
		ch.SendMessage(fmt.Sprintf("It looks like there is %s over there.\r\n", scoutCrowdSize(peopleCount)))
	}

	return true
}

func scoutTerrain(sector int) string {
	switch sector {
	case 0:
		return "the inside of a structure"
	case 1:
		return "the cobblestones of a city"
	case 2:
		return "a wide swath of field"
	case 3:
		return "the dense forest"
	case 4:
		return "high hills"
	case 5:
		return "jagged mountains"
	case 6, 7:
		return "a large stretch of water"
	case 8:
		return "the watery depths"
	case 9:
		return "thin air"
	case 10:
		return "a vast wasteland"
	case 15:
		return "a murky swamp"
	default:
		return "the endless elemental plane"
	}
}

func scoutCrowdSize(count int) string {
	switch {
	case count == 1:
		return "someone"
	case count <= 3:
		return "a few people"
	case count <= 5:
		return "a group of people"
	case count <= 9:
		return "a large group of people"
	case count <= 12:
		return "a crowd of people"
	case count <= 14:
		return "a large crowd of people"
	default:
		return "a large mob"
	}
}

// ---------------------------------------------------------------------------
// do_roll — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRoll(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	maxRoll := rollMaximum(arg)

	result := rollNumber(maxRoll, dprng.Number)
	// C do_roll (act.other.c:1942): "You roll %u (1-%u)." + act TO_ROOM.
	ch.SendMessage(fmt.Sprintf("You roll %d (1-%d).\r\n", result, maxRoll))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("With a toss of the dice, %s rolls %d (1-%d).\r\n", ch.Name, result, maxRoll), ch.Name)
	return true
}

// rollMaximum mirrors do_roll's one_argument()/atoi()/zero-default path. The
// C storage type is unsigned int, so a signed atoi result is converted to the
// same 32-bit representation before it reaches number(int, int).
func rollMaximum(arg string) uint32 {
	first, _ := oneArgument(arg)
	maxRoll := cIntToUint32(atoiC(first))
	if maxRoll == 0 {
		return 100
	}
	return maxRoll
}

// rollNumber preserves C's implicit unsigned-int-to-int conversion at the
// number(1, max_roll) call and returns the unsigned result stored by do_roll.
func rollNumber(maxRoll uint32, number func(int, int) int) uint32 {
	return cIntToUint32(number(1, cUnsignedToInt(maxRoll)))
}

// cIntToUint32 performs the modulo conversion C applies when an int is stored
// in an unsigned int, without relying on an unchecked Go narrowing cast.
func cIntToUint32(value int) uint32 {
	const modulus = int64(1 << 32)
	normalized := int64(value) % modulus
	if normalized < 0 {
		normalized += modulus
	}
	if normalized < 0 || normalized > int64(^uint32(0)) {
		return 0
	}
	return uint32(normalized)
}

// cUnsignedToInt performs the implementation-defined C conversion used when
// do_roll passes unsigned max_roll to number(int, int). C int is 32 bits here.
func cUnsignedToInt(value uint32) int {
	const (
		cIntMax = int64(1<<31 - 1)
		cIntMin = -1 << 31
	)
	signed := int64(value)
	if signed > cIntMax {
		signed -= 1 << 32
	}
	if signed < cIntMin || signed > cIntMax {
		return 0
	}
	return int(signed)
}
