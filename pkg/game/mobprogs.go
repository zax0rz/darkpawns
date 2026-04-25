// mobprogs.go — Ported from src/mobprog.c
//
// Mob programming (mobprog) triggers: greet, ride_greet, give, bribe, sound.
//
// This provides Go-side infrastructure that Lua scripting hooks into.
// Each trigger function is a World method that can be called from Lua scripts
// or from other Go code when the relevant event occurs.
//
// Source mobprog.c revision: $Id: mobprog.c 1487 2008-05-22 01:36:10Z jravn $

package game

import (
	"fmt"
	"math/rand"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// VNum-based macros matching src/mobprog.h
// ---------------------------------------------------------------------------

const (
	mobVNumDog       = 8063
	mobVNumDog2      = 8065
	mobVNumJanitor   = 8061
	mobVNumDemon     = 14401
	mobVNumMercenary = 3063
)

// MP sound type constants (matching mobprog.c #defines)
const (
	mpNone  = 0
	mpSpeak = 1
	mpEmote = 2
)

// isDogMob checks if the mob is one of the dog NPCs.
// Source: mobprog.h IS_DOG()
func isDogMob(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	vnum := mob.GetVNum()
	return vnum == mobVNumDog || vnum == mobVNumDog2
}

// isJanitorMob checks if the mob is the janitor NPC.
// Source: mobprog.h IS_JANITOR()
func isJanitorMob(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	return mob.GetVNum() == mobVNumJanitor
}

// isDemonMob checks if the mob is the soul-eater demon NPC.
// Source: mobprog.h IS_DEMON()
func isDemonMob(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	return mob.GetVNum() == mobVNumDemon
}

// isMercenaryMob checks if the mob is a mercenary NPC.
// Source: mobprog.h IS_MERCENARY()
func isMercenaryMob(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	return mob.GetVNum() == mobVNumMercenary
}

// isMobShopkeeper checks if a mob has a shopkeeper-related spec.
// This is distinct from isShopkeeper() in act_offensive.go which takes
// (*World, *Player).
// Source: mobprog.c is_shopkeeper()
func isMobShopkeeper(mob *MobInstance) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}

	// Check spec name
	specName, ok := MobSpecAssign[mob.GetVNum()]
	if ok {
		switch specName {
		case "shop_keeper", "guild", "guild_guard", "butler", "clerk":
			return true
		}
	}

	// Fallback: VNum check
	switch mob.GetVNum() {
	case 8003, 8004, 8005, 8006, 8007, 8008, 8009, 8010, 8011, 8078:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// mp_greet — Greet trigger
// Source: mobprog.c mp_greet()
//
// Called when a character (who) enters a room. The function iterates over
// mobs in the room and triggers behaviors: dog lick/growl, shopkeeper
// reactions, and specific VNum greetings.
func (w *World) MpGreet(who *Player, room int) {
	mobs := w.GetMobsInRoom(room)
	for _, tch := range mobs {
		if tch == nil {
			continue
		}

		// Dog behavior: 1-in-6 chance to react to alignment
		// C: IS_DOG(tch) && !(number(0, 5)) → rand.Intn(6) == 0
		if isDogMob(tch) && rand.Intn(6) == 0 {
			if who.GetAlignment() < 0 {
				// C: do_action(tch, GET_NAME(who), cmd_lick, 0)
				// In C, cmd_lick = find_command("growl") — the dogs are inverted
				w.roomMessage(room, fmt.Sprintf("%s licks %s happily.",
					tch.GetShortDesc(), who.GetName()))
			} else {
				w.roomMessage(room, fmt.Sprintf("%s growls at %s.",
					tch.GetShortDesc(), who.GetName()))
			}
		} else if tch.GetVNum() == 19406 {
			// Quest mob: fire-sitter — greet soloers and group leaders
			// C: !who->master || who->master == who
			if who.Following == "" || who.Following == who.GetName() {
				w.roomMessage(room, fmt.Sprintf("%s says, 'Have a seat there! Stay a while, rest your bones and warm your feet by the fire!'",
					tch.GetShortDesc()))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// mp_ride_greet — Mount greeting trigger
// Source: mobprog.c mp_ride_greet()
//
// In the original C this iterated the room's people but did nothing for each.
// It exists as a hook point for Lua scripts or future behavior.
func (w *World) MpRideGreet(who *Player, room int) {
	// This is a no-op in the original C — the loop body was just:
	//   for (tch = world[room].people; tch; tch = tch_next) {
	//     tch_next = tch->next_in_room;
	//     if (tch == who) continue;
	//   }
	// It exists as a hook point. Preserved here for Lua script integration.
	_ = who
	_ = room
}

// ---------------------------------------------------------------------------
// mp_give — Give trigger
// Source: mobprog.c mp_give()
//
// Called when a player (ch) gives an object (obj) to a mob.
func (w *World) MpGive(ch *Player, mob *MobInstance, obj *ObjectInstance) {
	// Dog behavior
	if isDogMob(mob) {
		if obj.GetTypeFlag() == 2 { // ITEM_FOOD
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s devours %s and wags %s tail happily.",
				mob.GetShortDesc(), obj.GetShortDesc(), hisHer(mob.GetSex())))
			mob.RemoveFromInventory(obj)
			w.RemoveItemFromRoom(obj, mob.RoomVNum)
			// extract_obj — drop reference
			obj.RoomVNum = -1
			obj.Carrier = nil
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s sniffs around and plays with %s for a while.",
				mob.GetShortDesc(), obj.GetShortDesc()))
			mob.RemoveFromInventory(obj)
			w.AddItemToRoom(obj, mob.RoomVNum)
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s quickly loses interest.",
				mob.GetShortDesc()))
		}
		return
	}

	// Demon (soul eater) behavior
	if isDemonMob(mob) {
		if obj.GetVNum() != 9900 { // not a soul stone
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s peers at %s closely, then hands it back.",
				mob.GetShortDesc(), obj.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s growls, 'Are you mocking me?'",
				mob.GetShortDesc()))
			mob.RemoveFromInventory(obj)
			ch.Inventory.AddItem(obj)
		} else {
			// Soul stone accepted
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s peers at the soul, then licks %s lips.",
				mob.GetShortDesc(), hisHer(mob.GetSex())))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'This will do nicely.. you may enter!'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s pops the soul into his mouth and swallows it, as a hideous screaming rings in your ears...",
				mob.GetShortDesc()))

			// Create portal (vnum 19611)
			mob.RemoveFromInventory(obj)
			obj.RoomVNum = -1
			obj.Carrier = nil

			if proto, ok := w.GetObjPrototype(19611); ok {
				portal := NewObjectInstance(proto, mob.RoomVNum)
				portal.VNum = 19611
				portal.RoomVNum = mob.RoomVNum
				w.AddItemToRoom(portal, mob.RoomVNum)
			}

			w.DoSayMob(mob, "Enter the portal quickly! It will not last long!")
		}
		return
	}

	// Janitor behavior
	if isJanitorMob(mob) {
		if isJunk(obj) {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Thanks for helping clean this place up...'",
				mob.GetShortDesc()))
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Wow, this is pretty neat, thanks.'",
				mob.GetShortDesc()))
		}
	}
}

// DoSayMob makes a mob say something in the room.
func (w *World) DoSayMob(mob *MobInstance, message string) {
	w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, '%s'", mob.GetShortDesc(), message))
}

// ---------------------------------------------------------------------------
// mp_bribe — Bribe trigger
// Source: mobprog.c mp_bribe()
//
// Called when a player (ch) gives gold to a mob.
func (w *World) MpBribe(ch *Player, mob *MobInstance, amount int) {
	if isDogMob(mob) {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s sniffs the coins and proceeds to eat them.",
			mob.GetShortDesc()))
	} else if isJanitorMob(mob) {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s tips his hat and smiles a thank you.",
			mob.GetShortDesc()))
	} else if isMercenaryMob(mob) {
		if amount > 99 && mob.Hunting == "" {
			// Hire the mercenary — use Hunting to track master
			mob.Hunting = ch.GetName()
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s counts the coins then secrets them away.",
				mob.GetShortDesc()))
			sendToChar(ch, "The mercenary swears his allegiance to you.")
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s hires %s.", ch.GetName(), mob.GetShortDesc()))
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s laughs somewhat rudely.",
				mob.GetShortDesc()))
		}
	} else if mob.GetVNum() == 8088 {
		// Jail guard
		if amount < ch.GetLevel()*ch.GetLevel() {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Are you trying to bribe me? That's against the law you know...'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'And not quite enough cash, either.'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s grins evilly.", mob.GetShortDesc()))
			ch.Gold += amount
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Thank you very much, monsieur.'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Now get outta here!'",
				mob.GetShortDesc()))
			sendToChar(ch, fmt.Sprintf("%s throws you out of the cell!", mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s throws %s out of the cell!",
				mob.GetShortDesc(), ch.GetName()))
			w.MovePlayerToRoom(ch, 8117)
		}
	} else if isCityguard(mob) {
		// C: !number(0,2) → 2/3 chance to attack
		// Otherwise accept bribe if amount >= 200
		if rand.Intn(3) != 0 || amount < 200 {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Are you trying to bribe me? That's against the law you know...'",
				mob.GetShortDesc()))
			if aiCombatEngine != nil {
				aiCombatEngine.StartCombat(mob, ch)
			}
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s glances around warily and says, 'I am off duty now...'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s lays down and falls asleep on the job!",
				mob.GetShortDesc()))
			mob.SetStatus("sleeping")
		}
	} else if isCitizen(mob) {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Thanks for the investment, I appreciate it.'",
			mob.GetShortDesc()))
		w.DoAction(mob, ch.GetName(), "bow")
	} else if isWhoreMob(mob) {
		if amount >= 5 {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s pulls %s into the shadows for a few minutes... you decide not to watch.",
				mob.GetShortDesc(), ch.GetName()))
			sendToChar(ch, fmt.Sprintf("%s pulls you into the shadows, and gives you a lot more than you expected for your money.",
				mob.GetShortDesc()))
			sendToChar(ch, "A few coins lighter and quite a bit happier, you continue on your way.")
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s says, 'Thanks hon, but I ain't THAT cheap.'",
				mob.GetShortDesc()))
		}
	} else if mob.GetVNum() == 13108 {
		// Gremlin guide
		if amount >= 1000 {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s leads %s through a hidden door.",
				mob.GetShortDesc(), ch.GetName()))
			w.MovePlayerToRoom(ch, 13154)
			sendToChar(ch, "  You follow the gremlin through a series of tunnels, full of twists, turns,\r\nand circles. Slowly you become aware of a tiny crack of light coming through\r\nthe top of the cavern. Suddenly the gremlin turns around and hurries away\r\nbefore you can even turn around.")
		} else {
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s exclaims, 'Cheap bastard, come back with some money!'",
				mob.GetShortDesc()))
			w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s gets kicked out on %s ass!'",
				ch.GetName(), hisHer(ch.GetSex())))
			sendToChar(ch, fmt.Sprintf("%s kicks you out on your ass!", mob.GetShortDesc()))
			w.MovePlayerToRoom(ch, 13193)
		}
	}
}

// ---------------------------------------------------------------------------
// entry_prog — Entry program trigger
// Source: mobprog.c entry_prog()
//
// Called when a mob enters a room (spawns or walks in).
func (w *World) EntryProg(mob *MobInstance, room int) {
	if room < 0 {
		return
	}

	if mob.GetVNum() != 8059 {
		return
	}

	// Captain of the Guard (VNum 8059)
	mobs := w.GetMobsInRoom(room)
	for _, tch := range mobs {
		if tch == mob {
			continue
		}
		if isCityguard(tch) {
			if tch.GetPosition() <= combat.PosSleeping {
				w.roomMessage(room, fmt.Sprintf("%s barks 'On your feet, slacker!'", mob.GetShortDesc()))
				w.roomMessage(room, fmt.Sprintf("%s wakes up and quickly snaps to attention!", tch.GetShortDesc()))
				w.roomMessage(room, fmt.Sprintf("%s growls, 'Report to my office at 0500 tomorrow morning.'", mob.GetShortDesc()))
				tch.SetStatus("standing")
			} else {
				w.roomMessage(room, fmt.Sprintf("%s snaps to attention and salutes!", tch.GetShortDesc()))
				w.roomMessage(room, fmt.Sprintf("%s growls, 'At ease, soldier.'", mob.GetShortDesc()))
			}
			return
		} else if isCitizen(tch) {
			w.roomMessage(room, fmt.Sprintf("%s frowns at %s.", mob.GetShortDesc(), tch.GetShortDesc()))
			w.roomMessage(room, fmt.Sprintf("%s says, 'Hail unto the True One, Captain.'", tch.GetShortDesc()))
			return
		}
	}
}

// ---------------------------------------------------------------------------
// isCitizen — Check if a mob is a citizen
// Source: mobprog.c is_citizen()
func isCitizen(mob *MobInstance) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}
	switch mob.GetVNum() {
	case 2749, 2750, 8062, 18201, 18202, 21243:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// isCityguard — Check if a mob is a city guard
// Source: mobprog.c is_cityguard()
func isCityguard(mob *MobInstance) bool {
	if mob == nil || mob.Prototype == nil {
		return false
	}

	// Check spec first
	specName, ok := MobSpecAssign[mob.GetVNum()]
	if ok && specName == "cityguard" {
		return true
	}

	// Fallback: VNum check
	switch mob.GetVNum() {
	case 2747, 8001, 8002, 8020, 8027, 8059, 8060, 12111,
		21200, 21201, 21203, 21227, 21228:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// isJunk — Check if an object qualifies as "junk"
// Source: mobprog.c is_junk()
func isJunk(obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return false
	}
	// Check ITEM_WEAR_TAKE flag (bit 0 of WearFlags array)
	// Source: structs.h:446 — ITEM_WEAR_TAKE = 0
	// IS_SET_AR(wear_flags, 0) — checks bit 0 of first element
	hasTake := (obj.Prototype.WearFlags[0] & 1) != 0

	// Check type ITEM_DRINKCON (6) or cost <= 10
	isDrinkCon := obj.GetTypeFlag() == 6
	return hasTake && (isDrinkCon || obj.GetCost() <= 10)
}

// ---------------------------------------------------------------------------
// getBadGuy — Find a player/mob fighting citizens or guards
// Source: mobprog.c get_bad_guy()
func (w *World) getBadGuy(chAtChar *MobInstance) *MobInstance {
	mobs := w.GetMobsInRoom(chAtChar.RoomVNum)

	badGuys := make([]*MobInstance, 0)

	for _, ch := range mobs {
		if ch == nil {
			continue
		}
		fightingTarget := ch.GetFighting()
		if fightingTarget == "" {
			continue
		}
		// Check if they're fighting a citizen or guard
		for _, tch := range mobs {
			if tch.GetName() == fightingTarget && (isCitizen(tch) || isCityguard(tch)) {
				badGuys = append(badGuys, ch)
				break
			}
		}
	}

	if len(badGuys) == 0 {
		return nil
	}

	// C: iVictim = number(0, len) → 0..len inclusive, 0 = "no one" (1-in-N+1 chance)
	// Then if iVictim == 0 → return NULL; else badGuys[iVictim-1]
	if iVictim := rand.Intn(len(badGuys) + 1); iVictim == 0 {
		return nil
	} else {
		return badGuys[iVictim-1]
	}
}

// ---------------------------------------------------------------------------
// killBadGuy — Make a mob attack someone fighting a citizen/guard
// Source: mobprog.c kill_bad_guy()
func (w *World) killBadGuy(ch *MobInstance) bool {
	if ch == nil || ch.GetPosition() < combat.PosStanding || ch.GetFighting() != "" {
		return false
	}

	opponent := w.getBadGuy(ch)
	if opponent != nil {
		w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s roars: 'Protect the innocent!  BANZAIIII!  CHARGE!'",
			ch.GetShortDesc()))
		if aiCombatEngine != nil {
			aiCombatEngine.StartCombat(ch, opponent)
		}
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// NpcRescue — Make an NPC rescue another NPC from combat
// Source: mobprog.c npc_rescue()
func (w *World) NpcRescue(chHero *MobInstance, chVictim *MobInstance) bool {
	mobs := w.GetMobsInRoom(chHero.RoomVNum)

	// Find the bad guy fighting the victim
	var chBadGuy *MobInstance
	for _, ch := range mobs {
		if ch == nil {
			continue
		}
		if ch.GetFighting() == chVictim.GetName() {
			chBadGuy = ch
			break
		}
	}

	if chBadGuy == nil {
		return false
	}
	if chBadGuy == chHero {
		return false
	}

	chHero.SendMessage(fmt.Sprintf("You bravely rescue %s.\r\n", chVictim.GetShortDesc()))
	chVictim.SendMessage(fmt.Sprintf("You are rescued by %s, your loyal friend!\r\n", chHero.GetShortDesc()))
	w.roomMessage(chHero.RoomVNum, fmt.Sprintf("%s heroically rescues %s.", chHero.GetShortDesc(), chVictim.GetShortDesc()))

	if aiCombatEngine != nil {
		aiCombatEngine.StartCombat(chHero, chBadGuy)
	}

	return true
}

// ---------------------------------------------------------------------------
// MpSound — Ambient sound for specific mobs
// Source: mobprog.c mp_sound()
func (w *World) MpSound(mob *MobInstance) {
	sound := ""
	mtype := mpNone

	switch mob.GetVNum() {
	case 8066: // Petitioner
		if rand.Intn(2) == 0 {
			sound = "Sign this, please! There's too much violence!"
		} else {
			sound = "You look like a kind person.. sign this petition?"
		}
		mtype = mpSpeak

	case 8067: // Carpenter
		if rand.Intn(2) == 0 {
			sound = "adjusts his tool belt."
		} else {
			sound = "wipes the sweat of labor from his brow."
		}
		mtype = mpEmote

	case 8068: // Town crier
		if rand.Intn(2) == 0 {
			sound = "Arch Bishop Dinive to arrive on the Day of Winter Dawning!"
		} else {
			sound = "By mandate of the church, no violence in town! The penalty is jail time!"
		}
		mtype = mpSpeak

	case 8069: // Zealot
		sound = "Repent sinners! The end time is near!"
		mtype = mpSpeak

	case 8071: // Beggar
		if rand.Intn(2) == 0 {
			sound = "Spare a coin, buddy?"
			mtype = mpSpeak
		} else {
			sound = "jingles his cup."
			mtype = mpEmote
		}

	case 8072: // Singing drunk
		sound = "sings an old war ditty... badly off-key."
		mtype = mpEmote

	case 8074: // Minstrel
		if rand.Intn(2) == 0 {
			sound = "plays a lilting tune about your mother's beauty."
		} else {
			sound = "sings a melody about your conquests in battle."
		}
		mtype = mpEmote

	case 8079: // Mime
		sound = "tries to escape from an invisible box only he can see."
		mtype = mpEmote

	case 14202: // Bhang
		sound = "tokes up on some kind bud."
		mtype = mpEmote

	case 8059: // Aversin (Captain of the Guard)
		w.DoAction(mob, "", "emote")
		sound = "Carry on, citizen."
		mtype = mpSpeak

	case 8023: // Elven prostitute
		sound = "jiggles in your direction."
		mtype = mpEmote

	case 16300: // KD recruiter
		if rand.Intn(2) == 0 {
			sound = "smiles at you."
		} else {
			sound = "shuffles some papers around on his desk."
		}
		mtype = mpEmote
	}

	// Output sound
	switch mtype {
	case mpSpeak:
		w.DoSayMob(mob, sound)
	case mpEmote:
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s %s", mob.GetShortDesc(), sound))
	}

	// Demon dialogue — runs in addition to normal sound
	if isDemonMob(mob) {
		if rand.Intn(3) != 0 {
			return
		}
		if mob.Prototype != nil {
			switch mob.Prototype.Alignment {
			case -1000:
				w.DoSayMob(mob, "I seek the dull blackened stones in which the souls of mortals have been trapped!")
				mob.Prototype.Alignment = -999
			case -999:
				w.DoSayMob(mob, "I shall open a portal to the Grey Fortress in exchange for a soul stone.")
				mob.Prototype.Alignment = -1000
			}
		}
		return
	}

	// Dog behavior: 1-in-26 chance to make a puddle
	if isDogMob(mob) && rand.Intn(26) == 0 {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s relieves itself, nearly hitting your foot.", mob.GetShortDesc()))
		if proto, ok := w.GetObjPrototype(20); ok {
			puddle := NewObjectInstance(proto, mob.RoomVNum)
			puddle.VNum = 20
			puddle.RoomVNum = mob.RoomVNum
			w.AddItemToRoom(puddle, mob.RoomVNum)
		}
	}
}

// ---------------------------------------------------------------------------
// DoAction — Execute a social/action by name on a mob
// Source: mobprog.c do_action() pattern
//
// Makes a mob perform a named action/social (e.g., "bow", "emote").
// In the original C, uses find_command() and the command interpreter.
// For the Go port, emits a basic action message as a stand-in.
func (w *World) DoAction(mob *MobInstance, target string, actionName string) {
	if mob == nil {
		return
	}
	if target != "" {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s %ss at %s.", mob.GetShortDesc(), actionName, target))
	} else {
		w.roomMessage(mob.RoomVNum, fmt.Sprintf("%s %ss.", mob.GetShortDesc(), actionName))
	}
}

// ---------------------------------------------------------------------------
// MovePlayerToRoom — Moves a player to a room, handling mount side-effects
// Source: mobprog.c mp_bribe() uses char_from_room/char_to_room for jail/transport
func (w *World) MovePlayerToRoom(player *Player, targetVNum int) {
	_ = player.RoomVNum
	player.RoomVNum = targetVNum

	// TODO: Handle mount following as in C:
	//   if (get_mount(ch)) {
	//     char_from_room(get_mount(ch));
	//     char_to_room(get_mount(ch), real_room(target));
	//   }
}

// ---------------------------------------------------------------------------
// isWhoreMob — Check if mob has prostitute spec
// Source: mobprog.h IS_WHORE()
func isWhoreMob(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	specName, ok := MobSpecAssign[mob.GetVNum()]
	return ok && specName == "prostitute"
}

// ---------------------------------------------------------------------------
// Integration hooks — to be called from spec_procs*, mobact.go, or world.go
// ---------------------------------------------------------------------------

// OnMobReceiveItem is called when a player gives an item to a mob.
// Triggers mp_give behavior.
func (w *World) OnMobReceiveItem(ch *Player, mob *MobInstance, obj *ObjectInstance) {
	w.MpGive(ch, mob, obj)
}

// OnMobReceiveGold is called when a player gives gold to a mob.
// Triggers mp_bribe behavior.
func (w *World) OnMobReceiveGold(ch *Player, mob *MobInstance, amount int) {
	w.MpBribe(ch, mob, amount)
}

// OnMobEntry is called when a mob enters a room (spawn or walk-in).
// Triggers entry_prog behavior.
func (w *World) OnMobEntry(mob *MobInstance, room int) {
	w.EntryProg(mob, room)
}
