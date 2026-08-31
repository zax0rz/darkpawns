// spec_procs.go — Special procedure implementations for mobiles/objects/rooms.
//
// Ported from Dark Pawns MUD C source (spec_procs.c).
// Each handler is registered via RegisterSpec in an init() function.
//
// Signature: func(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool
//   - cmd is the command string typed by the player (e.g. "practice", "drop", "north")
//   - cmd == "" means the mob triggers on its own pulse/tick
//   - arg is the remainder of the command line after the command
//   - return true if the spec handled the interaction (blocking further processing)
//   - return false if the spec did not handle it

package game

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// ================================================================
// Helpers
// ================================================================

func randRange(min, max int) int {
	return dprng.Number(min, max)
}

// citizenNumber is the C citizen special's RNG seam. It is kept separate from
// the general helper so focused tests can pin both conditional draws without
// changing the process-wide stream used by other specials.
var citizenNumber = dprng.Number

// randN returns a uniform random integer in [0, n). It is exclusive, so it
// is safe for array indexing and switch selection. For C-style inclusive
// probability gates use number(from, to) instead.
func randN(n int) int {
	if n <= 0 {
		return 0
	}
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	return dprng.Number(0, n-1)
}

func (w *World) roomMessage(roomVNum int, msg string) {
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		p.SendMessage(msg + "\r\n")
	}
}

// actMessage sends audience-aware messages to players in a room.
// toChar is sent to the actor (if actor is a player), toVict to the victim,
// and toRoom to everyone else. Empty strings are skipped.
// Mirrors C's act() with TO_CHAR / TO_VICT / TO_ROOM.
func (w *World) actMessage(roomVNum int, actor, victim combat.Combatant, toChar, toVict, toRoom string) {
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if actor != nil && p.GetName() == actor.GetName() {
			if toChar != "" {
				p.SendMessage(toChar + "\r\n")
			}
			continue
		}
		if victim != nil && p.GetName() == victim.GetName() {
			if toVict != "" {
				p.SendMessage(toVict + "\r\n")
			}
			continue
		}
		if toRoom != "" {
			p.SendMessage(toRoom + "\r\n")
		}
	}
}

func sendToChar(ch *Player, msg string) {
	ch.SendMessage(msg + "\r\n")
}

func isMoveCmd(cmd string) bool {
	switch cmd {
	case "north", "south", "east", "west", "up", "down",
		"n", "s", "e", "w", "u", "d":
		return true
	}
	return false
}

func (w *World) roomCleanup(roomVNum int) int {
	items := w.GetItemsInRoom(roomVNum)
	totalVal := 0
	for _, obj := range items {
		w.roomMessage(roomVNum, obj.GetShortDesc()+" vanishes in a puff of smoke!")
		if err := w.MoveObjectToNowhere(obj); err != nil {
			slog.Warn("MoveObjectToNowhere failed in room cleanup", "obj_vnum", obj.GetVNum(), "error", err)
		}
		cost := obj.GetCost()
		if cost < 1 {
			cost = 1
		}
		v := cost / 10
		if v < 1 {
			v = 1
		}
		if v > 10 {
			v = 10
		}
		totalVal += v
	}
	return totalVal
}

// mobMeleeTarget returns the mob's current melee opponent as a MobInstance.
func mobMeleeTarget(me *MobInstance) *MobInstance {
	if me.GetTarget() != nil {
		return me.GetTarget()
	}
	return nil
}

// mobFightingTarget resolves the name-based FIGHTING reference used by the
// combat engine. C's fighter special receives the actual FIGHTING pointer;
// Go's mob instance retains that pointer only for mob-to-mob scripting, so
// the combat pair is the authoritative source for player opponents.
func mobFightingTarget(w *World, me *MobInstance) combat.Combatant {
	if w != nil && w.combatEngine != nil {
		if target, ok := w.combatEngine.GetCombatTarget(me.GetName()); ok && target != nil {
			return target
		}
	}

	if target := me.GetTarget(); target != nil {
		return target
	}
	if targetName := me.GetFightingTarget(); targetName != "" {
		if player, ok := w.GetPlayer(targetName); ok {
			return player
		}
		if mob := w.GetMobByName(targetName); mob != nil {
			return mob
		}
	}
	return nil
}

// ================================================================
// MOB SPECIALS
// ================================================================

// guild — practice skills with a guildmaster mob
// specGuild ports the guildmaster mob spec SPECIAL(guild) (spec_procs.c:201).
// It intercepts `practice` for a player standing with the guildmaster: no-arg
// lists the catalog; a named skill/spell is learned (guild-owned mutation),
// gaining MIN(MAXGAIN, MAX(MINGAIN, int_app[INT].learn)) up to the learned cap.
func specGuild(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || cmd != "practice" {
		return false
	}

	class := ch.GetClass()
	arg = strings.TrimSpace(arg)

	if arg == "" {
		// RenderSkillList already contains the terminal CRLF because the
		// standalone practice command sends it directly.
		ch.SendMessage(RenderSkillList(ch))
		return true
	}
	if ch.GetPractices() <= 0 {
		sendToChar(ch, "You do not seem to be able to practice now.")
		return true
	}

	skillNum := FindSkillNum(arg)
	if skillNum < 1 || ch.GetLevel() < ClassSkillMinLevel(class, skillNum) {
		sendToChar(ch, fmt.Sprintf("You do not know of that %s.", SplSkl(class)))
		return true
	}

	name := SkillStorageName(skillNum)
	learned := pracLearned(class)
	if ch.GetSkill(name) >= learned {
		sendToChar(ch, "You are already learned in that area.")
		return true
	}

	sendToChar(ch, "You practice for a while...")
	ch.SetPractices(ch.GetPractices() - 1)

	// percent += MIN(MAXGAIN, MAX(MINGAIN, int_app[GET_INT].learn)) (spec_procs.c:242)
	gain := intAppLearn(ch.GetInt())
	if gain < pracMin(class) {
		gain = pracMin(class)
	}
	if gain > pracMax(class) {
		gain = pracMax(class)
	}
	percent := ch.GetSkill(name) + gain
	if percent > learned {
		percent = learned
	}
	ch.SetSkill(name, percent)

	if ch.GetSkill(name) >= learned {
		sendToChar(ch, "You are now learned in that area.")
	}
	return true
}

// intAppLearn returns int_app[intScore].learn with bounds clamping.
func intAppLearn(intScore int) int {
	if intScore < 0 {
		intScore = 0
	}
	if intScore >= len(intApp) {
		intScore = len(intApp) - 1
	}
	return intApp[intScore].Learn
}

// dump — room spec: trash items vanish, player gets XP/gold
func specDump(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	roomVNum := ch.GetRoomVNum()
	_ = w.roomCleanup(roomVNum)

	if cmd != "drop" {
		return false
	}

	// Perform the actual drop (mirrors C's do_drop(ch, argument, cmd, 0)).
	// This moves items from player inventory to the room so they can be valued.
	w.doDrop(ch, me, cmd, arg)

	// Value and remove the dropped items.
	value := w.roomCleanup(roomVNum)
	if value > 0 {
		sendToChar(ch, "You are awarded for outstanding performance.")
		w.roomMessage(roomVNum, ch.GetName()+" has been awarded by the gods!")
		ch.mu.Lock()
		if ch.Level < 3 {
			ch.Exp += value
		} else {
			ch.Gold += value
		}
		ch.mu.Unlock()
	}
	return true
}

// snake — mob spec: poison bite in combat
func specSnake(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	if number(0, 32-me.GetLevel()) != 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil {
		return false
	}
	w.actMessage(
		me.RoomVNum, me, melee,
		"",                         // TO_CHAR (mob doesn't need a message)
		me.GetName()+" bites you!", // TO_VICT
		me.GetName()+" bites "+melee.GetName()+"!", // TO_ROOM
	)
	spells.Cast(me, melee, spells.SpellPoison, me.GetLevel(), w)
	return true
}

// summoner — mob spec: summons player to it
func specSummoner(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding {
		return false
	}
	var vict *Player
	for _, memName := range me.GetMemory() {
		if p, ok := w.players[memName]; ok {
			vict = p
			break
		}
	}
	if vict == nil {
		for _, p := range w.GetPlayersInRoom(me.GetRoom()) {
			if !p.IsNPC() {
				for _, memName := range me.GetMemory() {
					if memName == p.Name {
						vict = p
						break
					}
				}
				if vict != nil {
					break
				}
			}
		}
	}
	if vict != nil && number(0, 4) == 0 {
		spells.Cast(me, vict, spells.SpellTeleport, me.GetLevel(), w)
		if me.RoomVNum == vict.GetRoomVNum() {
			me.SetFighting(vict.Name)
		}
		return true
	}
	return false
}

// thief — mob spec: steals gold
func specThief(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding {
		return false
	}
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if !p.IsNPC() && p.GetLevel() < LVL_IMMORT && number(0, 4) == 0 {
			npcSteal(w, me, p)
			return true
		}
	}
	return false
}

func npcSteal(w *World, me *MobInstance, victim *Player) {
	if victim.IsNPC() || victim.GetLevel() >= 50 {
		return
	}
	if victim.GetPosition() > combat.PosSleeping && number(0, me.GetLevel()) == 0 {
		Act(w, false, me, victim, nil, nil,
			"You discover that $n has $s hands in your wallet.", "", ToVict)
		Act(w, true, me, victim, nil, nil,
			"$n tries to steal gold from $N.", "", ToNotVict)
	} else {
		gold := (victim.GetGold() * randRange(1, 10)) / 100
		if gold > 0 {
			victim.SetGold(victim.GetGold() - gold)
			me.SetGold(me.GetGold() + gold)
		}
	}
}

// castMobSpell is the native NPC cast_spell() path.  Unlike a player command,
// a mob special enters cast_spell() directly, so the procedure must emit the
// verbal component before dispatching call_magic (spell_parser.c:827-909).
func castMobSpell(w *World, me *MobInstance, victim combat.Combatant, spellNum int) bool {
	// C's GET_WIS/GET_INT and GET_MOB_WAIT gates send only to the mob, whose
	// descriptor is absent; retaining the gate avoids inventing player bytes.
	if me.GetWis() == 0 || me.GetInt() == 0 || me.GetWaitState() > 0 {
		return false
	}
	spells.SaySpell(me, spellNum, victim, nil, w)
	return spells.Cast(me, victim, spellNum, me.GetLevel(), w)
}

// magic_user — mob spec: casts combat spells while fighting
func specMagicUser(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}

	var vict *Player
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if p.GetFighting() == me.GetName() && number(0, 4) == 0 {
			vict = p
			break
		}
	}
	if vict == nil {
		if tName := me.GetFightingTarget(); tName != "" {
			if p, ok := w.players[tName]; ok {
				vict = p
			}
		}
	}
	if vict == nil {
		return false
	}

	spellRoll := number(0, me.GetLevel()/2) + me.GetLevel()/2
	switch {
	case spellRoll <= 5:
		castMobSpell(w, me, vict, spells.SpellMagicMissile)
	case spellRoll <= 7:
		castMobSpell(w, me, vict, spells.SpellChillTouch)
	case spellRoll <= 9:
		castMobSpell(w, me, vict, spells.SpellBurningHands)
	case spellRoll <= 11:
		castMobSpell(w, me, vict, spells.SpellShockingGrasp)
	case spellRoll == 12:
		if !mobIsEvil(me) && vict.GetAlignment() <= -350 {
			castMobSpell(w, me, vict, spells.SpellDispelEvil)
		} else if mobIsEvil(me) && vict.GetAlignment() >= 350 {
			castMobSpell(w, me, vict, spells.SpellDispelGood)
		}
	case spellRoll == 13:
		castMobSpell(w, me, vict, spells.SpellLightningBolt)
	case spellRoll == 14:
		if number(0, 10) == 0 {
			castMobSpell(w, me, vict, spells.SpellTeleport)
		}
	case spellRoll >= 15 && spellRoll <= 17:
		castMobSpell(w, me, vict, spells.SpellColorSpray)
	case spellRoll == 20:
		castMobSpell(w, me, vict, spells.SpellHellfire)
	case spellRoll == 25:
		castMobSpell(w, me, vict, spells.SpellFlamestrike)
	case spellRoll == 30:
		castMobSpell(w, me, vict, spells.SpellDisintegrate)
	case spellRoll >= 31 && spellRoll <= 33:
		castMobSpell(w, me, vict, spells.SpellDisrupt)
	case spellRoll == 34:
		castMobSpell(w, me, me, spells.SpellInvulnerability)
	case spellRoll >= 35 && spellRoll <= 36:
		if w.IsOutside(me.GetRoom()) {
			castMobSpell(w, me, vict, spells.SpellFlamestrike)
		}
	case spellRoll == 37:
		if w.IsOutside(me.GetRoom()) {
			castMobSpell(w, me, vict, spells.SpellMeteorSwarm)
		}
	case spellRoll == 38:
		castMobSpell(w, me, vict, spells.SpellDisrupt)
	default:
		castMobSpell(w, me, vict, spells.SpellFireball)
	}
	return true
}

// fighter — mob spec: uses martial skills in combat
func specFighter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 || me.GetFighting() == "" {
		return false
	}
	if me.GetWaitState() > 0 {
		return false
	}
	melee := mobFightingTarget(w, me)
	if melee == nil {
		return false
	}
	switch number(0, 10) {
	case 1:
		mobHeadbutt(w, me, melee)
	case 2:
		mobParry(w, me, melee)
	case 3:
		mobBash(w, me, melee)
	case 4:
		mobBerserk(me)
	default:
		return false
	}
	return true
}

// paladin — mob spec: paladin combat
func specPaladin(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 || me.GetFighting() == "" {
		return false
	}
	if me.GetWaitState() > 0 {
		return false
	}
	melee := mobFightingTarget(w, me)
	if melee == nil {
		return false
	}
	switch number(0, 8) {
	case 0:
		mobParry(w, me, melee)
	case 1:
		mobBash(w, me, melee)
	case 2:
		mobCharge(w, me, melee)
	case 3:
		castMobSpell(w, me, melee, paladinDispelSpell(me))
	case 5:
		mobDisarm(w, me, melee)
	}
	return true
}

func paladinDispelSpell(me *MobInstance) int {
	if mobIsEvil(me) {
		return spells.SpellDispelGood
	}
	return spells.SpellDispelEvil
}

// guild_guard — mob spec: blocks unauthorized entry
func specGuildGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "flee" || cmd == "escape" || cmd == "retreat" {
		sendToChar(ch, "You try to flee inside the guild but the guard stops you!")
		Act(w, false, ch, nil, nil, nil,
			"$n tries to flee inside the guild but the guard block $s way!", "", ToRoom)
		return true
	}
	direction, moving := guildGuardDirection(cmd)
	if !moving || me.IsAffected(affBlind) {
		if me.IsFighting() {
			return specFighter(w, nil, me, "", "")
		}
		return false
	}
	if ch.GetLevel() >= LVL_IMMORT || isRemortOnlyClass(ch.GetClass()) || w.IsHunting(ch.GetName(), false) {
		return false
	}
	roomVNum := ch.GetRoomVNum()
	for _, e := range GuildInfo {
		if ch.GetClass() != e.Class && roomVNum == e.Room && direction == e.Direction {
			sendToChar(ch, "The guard humiliates you, and blocks your way.")
			Act(w, false, ch, nil, nil, nil,
				"The guard humiliates $n, and blocks $s way.", "", ToRoom)
			return true
		}
	}
	return false
}

func guildGuardDirection(cmd string) (int, bool) {
	switch cmd {
	case "north", "n":
		return 0, true
	case "east", "e":
		return 1, true
	case "south", "s":
		return 2, true
	case "west", "w":
		return 3, true
	case "up", "u":
		return 4, true
	case "down", "d":
		return 5, true
	default:
		return -1, false
	}
}

func isRemortOnlyClass(class int) bool {
	switch class {
	case ClassMagus, ClassAvatar, ClassAssassin, ClassPaladin, ClassRanger, ClassMystic:
		return true
	default:
		return false
	}
}

// puff — mob spec: ambient random speech/emotes on pulse
func specPuff(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if me.GetHP() < 0 {
		puffSay(w, me, "Shit, I'm dead.")
		return true
	}
	switch number(0, 90) {
	case 0:
		puffSay(w, me, "My god!  It's full of stars!")
	case 1:
		puffSay(w, me, "How'd all those fish get up here?")
	case 2:
		puffSay(w, me, "I'm a very female dragon.")
	case 3:
		puffSay(w, me, "I've got this peaceful, easy feeling.")
	case 4, 5, 6:
		return true
	case 7:
		puffSay(w, me, "Goddamn, what a trip! Listen to those colors!")
	case 8:
		puffSay(w, me, "Bring out your dead!")
	case 9:
		puffSay(w, me, "Rule number 6...there is NO rule number 6.")
	case 10:
		puffSay(w, me, "To be rich is no longer a sin...its a MIRACLE!")
	case 11, 12:
		return true
	case 13:
		puffEmote(w, me, "$n looks at you and then breaks out in a fit of laughter!")
	case 14:
		return true
	case 15:
		puffSay(w, me, "What is the sound of down?")
	case 16:
		return true
	case 17:
		puffEmote(w, me, "$n wonders where she left that darn wand.")
	case 18, 19:
		return true
	case 20, 21:
		puffSay(w, me, "Do you want to stroke my tail?")
	case 22:
		return true
	case 23, 24:
		puffEmote(w, me, "$n does female stuff.")
	case 25:
		return true
	case 26:
		puffEmote(w, me, "$n contemplates the meaning of life.")
	case 27:
		puffSay(w, me, "NIH!")
	case 28, 29, 30:
		return true
	case 31, 32:
		puffEmote(w, me, "$n rocks out to some funky beats.")
	case 33, 34, 35, 36:
		return true
	case 37, 38, 39:
		puffSay(w, me, "I'm gonna kick your ASS!")
	case 40, 41, 42:
		return true
	default:
		return false
	}
	return true
}

func puffSay(w *World, me *MobInstance, saying string) {
	verb := "says"
	if saying != "" {
		switch saying[len(saying)-1] {
		case '!':
			verb = "exclaims"
		case '?':
			verb = "asks"
		case '.':
			verb = "states"
		}
	}
	Act(w, false, me, nil, nil, nil, "$n "+verb+", '$T'", saying, ToRoom)
}

func puffEmote(w *World, me *MobInstance, emote string) {
	Act(w, true, me, nil, nil, nil, emote, "", ToRoom)
}

// fido — mob spec: dog scavenges corpses
func specFido(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me.IsFighting() || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	items := w.GetItemsInRoom(me.RoomVNum)
	for _, obj := range items {
		if obj.GetTypeFlag() == ITEM_CONTAINER && obj.GetValue(3) != 0 {
			Act(w, false, me, nil, nil, nil, "$n savagely devours a corpse.", "", ToRoom)
			// Spill corpse contents to the room before extracting the corpse.
			// Matches C src/spec_procs.c:735-741: obj_from_obj + obj_to_room.
			for len(obj.GetContents()) > 0 {
				content := obj.GetContents()[0]
				if err := w.MoveObjectToRoomFront(content, me.RoomVNum); err != nil {
					slog.Error("MoveObjectToRoom failed in fido spec", "obj_vnum", content.GetVNum(), "room", me.RoomVNum, "error", err)
					return false
				}
			}
			if err := w.MoveObjectToNowhere(obj); err != nil {
				slog.Error("MoveObjectToNowhere failed in fido spec", "obj_vnum", obj.GetVNum(), "error", err)
				return false
			}
			return true
		}
	}
	return false
}

// janitor — mob spec: cleans up items
func specJanitor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	items := w.GetItemsInRoom(me.RoomVNum)
	for _, obj := range items {
		// C uses CAN_WEAR(i, ITEM_WEAR_TAKE) and the reversed isname(i->name,
		// "corpse") call at src/spec_procs.c:759.
		if !obj.IsTakeable() || isName(obj.GetKeywords(), "corpse") {
			continue
		}
		Act(w, false, me, nil, nil, nil, "$n picks up some trash.", "", ToRoom)
		// Move to janitor's inventory through the canonical path. C obj_to_char
		// prepends, so preserve that order for later player-visible inspection.
		if err := w.MoveObjectToMobInventoryFront(obj, me); err != nil {
			slog.Error("MoveObjectToMobInventoryFront failed in janitor spec", "obj_vnum", obj.GetVNum(), "mob", me.GetVNum(), "error", err)
			return false
		}
		return true
	}
	return false
}

// cityguard — mob spec: guards patrol, arrest outlaws, and protect citizens.
// Ported from src/spec_procs.c:771-821. Handles the reachable autonomous
// branches; fighter() is a shared callee owned by mob.fighter.
func specCityguard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= 0 {
		return false
	}

	// C's mobile_activity() skips fighting mobs before calling a special, so
	// this branch is only relevant to the command path, where cityguard's
	// cmd gate has already returned FALSE. Keep the direct C call path explicit
	// for callers that invoke the proc synchronously in focused tests.
	if me.IsFighting() {
		return specFighter(w, ch, me, cmd, arg)
	}

	players := w.GetPlayersInRoom(me.RoomVNum)

	// Scan for outlaws — attack on sight (src/spec_procs.c:785-796). C gates
	// this branch with CAN_SEE and emits an act() template, not a pre-rendered
	// room string.
	for _, tch := range players {
		if !canSee(me, tch) || tch.GetFlags()&(1<<uint(PlrOutlaw)) == 0 {
			continue
		}
		Act(w, false, me, nil, nil, nil,
			"$n says, 'We don't like OUTLAWS like you in this city!'", "", ToRoom)
		if err := w.mobHit(me, tch); err != nil {
			slog.Warn("cityguard outlaw attack failed", "guard", me.GetName(), "target", tch.GetName(), "error", err)
		}
		return specFighter(w, ch, me, cmd, arg)
	}

	// Find the lowest-aligned visible combatant attacking a good-aligned target
	// (src/spec_procs.c:799-821). The C condition intentionally allows either
	// side of the fight to be an NPC; preserve that topology here.
	var evil cityguardAlignedCombatant
	var evilTarget cityguardAlignedCombatant
	maxEvil := 1000
	for _, candidate := range cityguardRoomCombatants(w, me.RoomVNum) {
		tch, ok := candidate.(cityguardAlignedCombatant)
		if !ok {
			continue
		}
		if !canSee(me, tch) || tch.GetFighting() == "" {
			continue
		}
		align := tch.GetAlignment()
		target := cityguardCombatantByName(w, me.RoomVNum, tch.GetFighting())
		targetAligned, ok := target.(cityguardAlignedCombatant)
		if targetAligned == nil || !ok || align >= maxEvil || (!tch.IsNPC() && !target.IsNPC()) {
			continue
		}
		maxEvil = align
		evil = tch
		evilTarget = targetAligned
	}
	if evil != nil && evilTarget.GetAlignment() >= 0 {
		Act(w, false, me, evil, nil, nil,
			"$n says, 'You just pissed me off, $N!'", "", ToRoom)
		if err := w.mobHit(me, evil); err != nil {
			slog.Warn("cityguard protect attack failed", "guard", me.GetName(), "target", evil.GetName(), "error", err)
		}
		return specFighter(w, ch, me, cmd, arg)
	}

	return false
}

type cityguardAlignedCombatant interface {
	combat.Combatant
	GetAlignment() int
}

// cityguardRoomCombatants mirrors the C world[].people walk closely enough
// for the cityguard's visible protection scan. The Go world stores players and
// mobs separately, so retain both sets at this boundary.
func cityguardRoomCombatants(w *World, roomVNum int) []combat.Combatant {
	players := w.GetPlayersInRoom(roomVNum)
	mobs := w.GetMobsInRoom(roomVNum)
	actors := make([]combat.Combatant, 0, len(players)+len(mobs))
	for _, player := range players {
		actors = append(actors, player)
	}
	for _, mob := range mobs {
		actors = append(actors, mob)
	}
	return actors
}

func cityguardCombatantByName(w *World, roomVNum int, name string) combat.Combatant {
	for _, player := range w.GetPlayersInRoom(roomVNum) {
		if player.GetName() == name {
			return player
		}
	}
	for _, mob := range w.GetMobsInRoom(roomVNum) {
		if mob.GetName() == name {
			return mob
		}
	}
	return nil
}

// mobHit mirrors C hit(): a mob special calls the synchronous combat entry,
// not the placeholder damage helper used by older mob paths. The fallback
// keeps focused spec tests useful when they intentionally omit a combat
// engine; production worlds provide the canonical initial-attack seam.
func (w *World) mobHit(attacker *MobInstance, defender combat.Combatant) error {
	if w.combatEngine == nil {
		player, ok := defender.(*Player)
		if !ok {
			return fmt.Errorf("mob special fallback cannot attack non-player %q", defender.GetName())
		}
		return attacker.Attack(player, w)
	}
	starter, hasDeferredStarter := w.combatEngine.(interface {
		StartCombatFromMob(combat.Combatant, combat.Combatant) error
	})
	var err error
	if hasDeferredStarter {
		err = starter.StartCombatFromMob(attacker, defender)
	} else {
		err = w.combatEngine.StartCombat(attacker, defender)
	}
	if err != nil {
		return err
	}
	if initial, ok := w.combatEngine.(interface {
		PerformInitialAttack(combat.Combatant, combat.Combatant) error
	}); ok {
		return initial.PerformInitialAttack(attacker, defender)
	}
	return nil
}

// mayorState holds per-mob state for the mayor path-walking system.
// Keyed by mob instance ID. Goroutine-safe via mayorMu.
type mayorState struct {
	path string
	pos  int // current index into path
}

var (
	mayorMu     sync.Mutex
	mayorStates = map[int]*mayorState{}
)

// mayorOpenPath and mayorClosePath are the mayor's daily walk routes.
// Ported from src/spec_procs.c:823-924.
// Characters: 0-3=direction(N/E/S/W), W=wake, S=sleep, a-e/E=emote/say, O=open gate, C=close gate, .=done.
const (
	mayorOpenPath  = "W3a3003b33000c111d0d111Oe333333Oe22c222112212111a1S."
	mayorClosePath = "W3a3003b33000c111d0d111CE333333CE22c222112212111a1S."
)

// specMayor — mob spec: daily walk schedule (New Thalos mayor opens/closes bazaar).
// Ported from src/spec_procs.c:823-924.
func specMayor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}

	mayorMu.Lock()
	st, exists := mayorStates[me.GetID()]
	if !exists {
		st = &mayorState{}
		mayorStates[me.GetID()] = st
	}
	mayorMu.Unlock()

	// Check if we should start a new path based on game hour
	if st.path == "" || st.pos >= len(st.path) {
		hour := timeInfo.Hours
		switch hour {
		case 6:
			mayorMu.Lock()
			st.path = mayorOpenPath
			st.pos = 0
			mayorMu.Unlock()
		case 20:
			mayorMu.Lock()
			st.path = mayorClosePath
			st.pos = 0
			mayorMu.Unlock()
		default:
			return false
		}
	}

	mayorMu.Lock()
	if st.pos >= len(st.path) {
		mayorMu.Unlock()
		return false
	}
	action := st.path[st.pos]
	st.pos++
	mayorMu.Unlock()

	switch action {
	case '0':
		w.mobPerformMove(me, 0) // NORTH
	case '1':
		w.mobPerformMove(me, 1) // EAST
	case '2':
		w.mobPerformMove(me, 2) // SOUTH
	case '3':
		w.mobPerformMove(me, 3) // WEST
	case 'W':
		me.SetPosition(8) // PosStanding
		w.roomMessage(me.RoomVNum, me.GetName()+" awakens and groans loudly.")
	case 'S':
		me.SetPosition(4) // PosSleeping
		w.roomMessage(me.RoomVNum, me.GetName()+" lies down and instantly falls asleep.")
	case 'a':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Hello Honey!'")
		w.roomMessage(me.RoomVNum, me.GetName()+" smirks.")
	case 'b':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'What a view!  I must get something done about that dump!'")
	case 'c':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Vandals!  Youngsters nowadays have no respect for anything!'")
	case 'd':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Good day, citizens!'")
	case 'e':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'I hereby declare the bazaar open!'")
	case 'E':
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'I hereby declare Bourbon closed!'")
	case 'O':
		w.mobOpenDoor(me, 0, "gate") // unlock + open gate to the NORTH
	case 'C':
		mayorMobCloseDoor(w, me, 0, "gate") // close + lock gate to the NORTH
	case '.':
		mayorMu.Lock()
		st.path = ""
		st.pos = 0
		mayorMu.Unlock()
	}

	return true
}

// mayorMobCloseDoor closes and locks a door for the mayor, matching C do_gen_door(CLOSE+LOCK).
func mayorMobCloseDoor(w *World, me *MobInstance, dir int, keyword string) {
	dirName := dirKeys[dir]
	w.mu.Lock()

	room := w.rooms[me.GetRoom()]
	if room == nil {
		w.mu.Unlock()
		return
	}
	ext, hasExit := room.Exits[dirName]
	if !hasExit {
		w.mu.Unlock()
		return
	}

	ext.ExitInfo |= parser.ExitClosed | parser.ExitLocked
	room.Exits[dirName] = ext

	otherRoom := w.rooms[ext.ToRoom]
	if otherRoom != nil {
		backDir := revDir[dir]
		backExt, hasBack := otherRoom.Exits[dirs[backDir]]
		if hasBack && backExt.ToRoom == me.GetRoom() {
			backExt.ExitInfo |= parser.ExitClosed | parser.ExitLocked
			otherRoom.Exits[dirs[backDir]] = backExt
		}
	}
	w.mu.Unlock()
	w.roomMessage(me.GetRoom(), me.GetName()+" closes and locks the gate.")
}

// dragonBreathSpell returns the spell selected by SPECIAL(dragon_breath)'s
// exact VNUM switch (src/spec_procs.c:937-954). The default is intentional:
// an assigned future dragon still uses C's fire-breath fallback.
func dragonBreathSpell(vnum int) int {
	switch vnum {
	case 4209, 4705:
		return spells.SpellFrostBreath
	case 11000:
		return spells.SpellAcidBreath
	case 11001, 20027:
		return spells.SpellLightningBreath
	case 11002:
		return spells.SpellFireBreath
	default:
		return spells.SpellFireBreath
	}
}

// specDragonBreath — mob spec: lair threat and breath weapon.
// Ported from src/spec_procs.c:926-983. The autonomous caller reaches the
// non-fighting branch; the combat engine reaches the fighting branch after
// the ordinary NPC attack loop, matching fight.c:1898-2032.
func specDragonBreath(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}

	spell := dragonBreathSpell(me.GetVNum())
	fighting := mobFightingTarget(w, me)
	if fighting != nil {
		if me.GetPosition() > combat.PosSleeping && me.GetPosition() < combat.PosFighting {
			// C calls do_stand() here. An NPC has no descriptor, so only its
			// state transition and the room Act are player-visible.
			switch me.GetPosition() {
			case combat.PosSitting:
				Act(w, true, me, nil, nil, nil, "$n clambers to $s feet.", "", ToRoom)
			case combat.PosResting:
				Act(w, true, me, nil, nil, nil, "$n stops resting, and clambers on $s feet.", "", ToRoom)
			default:
				Act(w, true, me, nil, nil, nil, "$n stops floating around, and puts $s feet on the ground.", "", ToRoom)
			}
			me.SetPosition(combat.PosStanding)
		} else if number(0, 3) == 0 {
			spells.CallMagic(me, fighting, nil, spell, me.GetLevel(), spells.CastBreath, w)
			return specMagicUser(w, nil, me, "", "")
		}
		return true
	}

	for _, victim := range w.GetPlayersInRoom(me.GetRoom()) {
		if !canSee(me, victim) || victim.GetFlags()&(1<<PrfNohassle) != 0 {
			continue
		}
		Act(w, true, me, nil, nil, nil, "$n looks at you.", "", ToRoom)
		Act(w, true, me, nil, nil, nil, "$n growls, 'So, you have found my lair...'", "", ToRoom)
		Act(w, true, me, nil, nil, nil, "$n exclaims, 'For that you must die!'", "", ToRoom)
		spells.CallMagic(me, victim, nil, spell, me.GetLevel(), spells.CastBreath, w)
		return true
	}
	return false
}

// citizen — mob spec: random greetings.
// Ported from src/spec_procs.c:986-1032. C calls this with ch==the mob on
// autonomous/combat paths, while command dispatch supplies the player as ch;
// the nil/non-nil distinction preserves that NPC gate in this signature.
func specCitizen(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil || ch != nil || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}

	if me.GetFighting() != "" {
		switch me.GetPosition() {
		case combat.PosSitting:
			Act(w, true, me, nil, nil, nil, "$n clambers to $s feet.", "", ToRoom)
			me.SetPosition(combat.PosStanding)
		case combat.PosResting:
			Act(w, true, me, nil, nil, nil, "$n stops resting, and clambers on $s feet.", "", ToRoom)
			me.SetPosition(combat.PosStanding)
		}
		return false
	}

	if citizenNumber(0, 19) != 0 {
		return false
	}

	switch citizenNumber(1, 10) {
	case 1:
		Act(w, true, me, nil, nil, nil, "$n jingles some change in $s pocket.", "", ToRoom)
	case 2:
		Act(w, true, me, nil, nil, nil, "$n stares into the sky.", "", ToRoom)
		Act(w, true, me, nil, nil, nil, "$n says, 'Looks like rain. *sigh*'", "", ToRoom)
	case 3:
		Act(w, true, me, nil, nil, nil, "$n glances at you out of the corner of $s eye.", "", ToRoom)
	case 4:
		Act(w, true, me, nil, nil, nil, "$n mumbles something about the price of a crappy loaf of bread.", "", ToRoom)
	case 5:
		Act(w, true, me, nil, nil, nil, "$n kicks a pebble out of the road.", "", ToRoom)
	case 6:
		Act(w, true, me, nil, nil, nil, "$n looks at you and shouts 'Repent! The end is near!'", "", ToRoom)
	case 7:
		Act(w, true, me, nil, nil, nil, "$n eyes your coin purse.", "", ToRoom)
	case 8:
		Act(w, true, me, nil, nil, nil, "$n looks around for the cityguards just before giving you the bird.", "", ToRoom)
	}

	// C returns FALSE even when it emits a room message, allowing the rest of
	// mobile_activity() to continue with its normal AI work.
	return false
}

// cuchi — mob spec: Easter egg + random speech.
// Matches src/spec_procs.c:1034-1071.
func specCuchi(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// C's CMD_IS("pat") gate is command-table based, so autonomous dispatch
	// (cmd == "") and every other command fall through without side effects.
	if cmd != "pat" || ch == nil || me == nil {
		return false
	}

	// The C act() calls use ch as the actor and TO_ROOM, so the command actor
	// receives only the two stc() messages and everyone else receives the two
	// room messages. The literal Cuchi is intentional; it is not me's runtime
	// short description.
	sendToChar(ch, "You pat Cuchi on the head and rub around her ears.")
	Act(w, false, ch, nil, nil, nil, "$n pats Cuchi on the head and rubs around her ears.", "", ToRoom)

	if ch.GetName() == "Orodreth" {
		ch.SetLevel(LVL_IMPL)
		sendToChar(ch, "Cuchi purrs at you contently.")
		Act(w, false, ch, nil, nil, nil, "Cuchi purrs contently at $n.", "", ToRoom)
	} else {
		ch.mu.Lock()
		ch.Gold += 10
		ch.mu.Unlock()
		sendToChar(ch, "Cuchi purrs at you and bestows a gift from the gods.")
		Act(w, false, ch, nil, nil, nil, "Cuchi purrs at $n and bestows a gift from the gods.", "", ToRoom)
	}

	return true
}

// mini_thief — mob spec: steals small items
func specMiniThief(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding || number(0, 3) != 0 {
		return false
	}
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if !p.IsNPC() && number(0, 2) == 0 {
			stealAmt := randRange(1, 20)
			p.mu.Lock()
			if p.Gold >= stealAmt {
				p.Gold -= stealAmt
				p.mu.Unlock()
				w.roomMessage(me.RoomVNum, me.GetName()+" snatches some coins and giggles!")
				sendToChar(p, "You notice your coin purse feels lighter...")
			} else {
				p.mu.Unlock()
			}
			return true
		}
	}
	return false
}

// black_undead_knight — mob spec: taunts + hates red undead
func specBlackUndeadKnight(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.IsFighting() {
		switch randRange(1, 20) {
		case 1:
			w.roomMessage(me.RoomVNum, me.GetName()+" screams, 'Protect the kingdom!'")
		case 2:
			w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'If I'm going to hell, you're going with me!'")
		case 3:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'You dirty rotten scoundrel. I'm gonna make you very sorry.'")
		case 4:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'I know what you're thinking...'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Did he fire five shots, or did he fire six.'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Well let me ask you...'")
			w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Do you feel lucky PUNK?  Well... DO YOU?'")
		case 5:
			w.roomMessage(me.RoomVNum, me.GetName()+" claims, 'I am the greatest!'")
		}
		return true
	}
	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if m.VNum == 11471 && m != me && number(0, 3) == 0 {
			w.roomMessage(me.GetRoom(), me.GetName()+" sees "+m.GetName()+" and gives a battle cry!")
			me.SetTarget(m)
			me.SetFighting(m.GetName())
			return true
		}
	}
	return false
}

// red_undead_knight — mob spec: taunts + hates black undead
func specRedUndeadKnight(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.IsFighting() {
		switch randRange(1, 20) {
		case 1:
			w.roomMessage(me.RoomVNum, me.GetName()+" screams, 'Protect the homeland!'")
		case 2:
			w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'If you think you have had a bad day before, watch this!'")
		case 3:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Don't ever argue with the big dog,'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'cause the big dog is always right.'")
		case 4:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'There's more than one way to skin a cat:'")
			w.roomMessage(me.RoomVNum, me.GetName()+" continues: 'Way number 15 -- Krazy Glue and a toothbrush.'")
		case 5:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'A friend with weed is a friend indeed.'")
		}
		return true
	}
	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if m.VNum == 11470 && m != me && number(0, 3) == 0 {
			w.roomMessage(me.GetRoom(), me.GetName()+" sees "+m.GetName()+" and gives a battle cry!")
			me.SetTarget(m)
			me.SetFighting(m.GetName())
			return true
		}
	}
	return false
}

// mickey — mob spec: harasses and attacks (from Natural Born Killers)
func specMickey(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.IsFighting() {
		if me.GetPosition() > combat.PosSleeping && me.GetPosition() < combat.PosFighting {
			w.roomMessage(me.RoomVNum, me.GetName()+" stands up!")
		} else {
			switch randRange(1, 10) {
			case 1:
				w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'I'll always love you Mal, no matter what!'")
			case 2:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Do you believe in fate?'")
			case 3:
				w.roomMessage(me.RoomVNum, me.GetName()+" says, 'You're not centered.'")
			case 4:
				w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'")
			case 5:
				w.roomMessage(me.RoomVNum, me.GetName()+" states, 'It's not nice to point.'")
			}
		}
		return true
	}
	return false
}

// mallory — mob spec: barks + calls mickey for revenge
func specMallory(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.IsFighting() {
		if me.GetPosition() > combat.PosSleeping && me.GetPosition() < combat.PosFighting {
			w.roomMessage(me.RoomVNum, me.GetName()+" stands up!")
		} else {
			switch randRange(1, 10) {
			case 1:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How do you like me now?'")
			case 2:
				w.roomMessage(me.RoomVNum, me.GetName()+" says, 'That was the worst head I've ever got.'")
			case 3:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How sexy am I now, fucker?'")
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How sexy am I NOW?'")
			case 5:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Did you bring enough for everybody?  Here... try one...'")
			}
		}
		return true
	}
	return false
}

// ================================================================
// Initialization — registers all spec procs into SpecRegistry
// ================================================================

func init() {
	RegisterSpec("guild", specGuild)
	RegisterSpec("dump", specDump)
	RegisterSpec("snake", specSnake)
	RegisterSpec("summoner", specSummoner)
	RegisterSpec("thief", specThief)
	RegisterSpec("magic_user", specMagicUser)
	RegisterSpec("fighter", specFighter)
	RegisterSpec("paladin", specPaladin)
	RegisterSpec("guild_guard", specGuildGuard)
	RegisterSpec("puff", specPuff)
	RegisterSpec("fido", specFido)
	RegisterSpec("janitor", specJanitor)
	RegisterSpec("cityguard", specCityguard)
	RegisterSpec("mayor", specMayor)
	RegisterSpec("dragon_breath", specDragonBreath)
	RegisterSpec("citizen", specCitizen)
	RegisterSpec("cuchi", specCuchi)
	RegisterSpec("mini_thief", specMiniThief)
	RegisterSpec("black_undead_knight", specBlackUndeadKnight)
	RegisterSpec("red_undead_knight", specRedUndeadKnight)
	RegisterSpec("mickey", specMickey)
	RegisterSpec("mallory", specMallory)
}
