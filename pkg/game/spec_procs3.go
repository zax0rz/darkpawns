package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func init() {
	RegisterSpec("clerk", specClerk)
	RegisterSpec("butler", specButler)
	RegisterSpec("brain_eater", specBrainEater)
	RegisterSpec("teleport_victim", specTeleportVictim)
	RegisterSpec("con_seller", specConSeller)
	RegisterSpec("no_move_down", specNoMoveDown)
	RegisterSpec("troll", specTroll)
	RegisterSpec("quan_lo", specQuanLo)
	RegisterSpec("alien_elevator", specAlienElevator)
	RegisterSpec("werewolf", specWerewolf)
	RegisterSpec("field_object", specFieldObject)
	RegisterSpec("portal_to_temple", specPortalToTemple)
	RegisterSpec("turn_undead", specTurnUndead)
	RegisterSpec("itoh", specItoh)
	RegisterSpec("mirror", specMirror)
	RegisterSpec("prostitute", specProstitute)
	RegisterSpec("roach", specRoach)
	RegisterSpec("mortician", specMortician)
	RegisterSpec("conjured", specConjured)
	RegisterSpec("hisc", specHisc)
	RegisterSpec("recruiter", specRecruiter)
	RegisterSpec("elements_master_column", specElementsMasterColumn)
	RegisterSpec("elements_platforms", specElementsPlatforms)
	RegisterSpec("elements_load_cylinders", specElementsLoadCylinders)
	RegisterSpec("elements_galeru_column", specElementsGaleruColumn)
	RegisterSpec("elements_galeru_alive", specElementsGaleruAlive)
	RegisterSpec("elements_minion", specElementsMinion)
	RegisterSpec("elements_guardian", specElementsGuardian)
	RegisterSpec("fly_exit_up", specFlyExitUp)
	RegisterSpec("shop_keeper", specShopKeeper)
	RegisterSpec("cleric", specCleric)
}

// specShopKeeper is a stub — the full implementation is in shop.c/shop.h.
func specShopKeeper(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	return false
}

// specCleric is a stub — the full implementation is in spec_procs.c.
func specCleric(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	return false
}

// specNoMoveDown blocks "down" movement unless the player is an immort.
func specNoMoveDown(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "down" || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	// TODO: IS_NPC(ch) / HUNTING check
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s blocks your way down.", mobName(me)))
	return true
}

// specClerk sells citizenship.
func specClerk(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	// TODO: w.CanSee(me, ch) check
	// TODO: zone-based hometown assignment
	if cmd != "list" && cmd != "buy" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if cmd == "buy" {
		if !strings.EqualFold(arg, "citizenship") {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s BUY CITIZENSHIP, if you're interested.'", mobName(me), ch.GetName()))
			return true
		}
		// TODO: ch.Gold >= 2000 check, ch.Hometown assignment, save
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s Citizenship costs 2,000 coins.'", mobName(me), ch.GetName()))
		return true
	}
	if cmd == "list" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s Citizenship costs 2,000 coins.'", mobName(me), ch.GetName()))
		return true
	}
	return false
}

// specButler tidies up the room, picking up loose items and storing them.
func specButler(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetFighting() != "" {
		return false
	}
	items := w.GetItemsInRoom(me.GetRoomVNum())
	hasCase := false
	hasCabinet := false
	hasChest := false
	for _, obj := range items {
		kw := strings.ToLower(obj.GetKeywords())
		if strings.Contains(kw, "case") {
			hasCase = true
		}
		if strings.Contains(kw, "cabinet") {
			hasCabinet = true
		}
		if strings.Contains(kw, "chest") {
			hasChest = true
		}
	}
	if !hasCase || !hasCabinet || !hasChest {
		return false
	}
	got := 0
	for _, obj := range items {
		if got >= 4 {
			break
		}
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gets %s.", mobName(me), obj.GetShortDesc()))
		// TODO: obj_from_room(obj), obj_to_char(me, obj)
		// TODO: sort into case/cabinet/chest by type
		got++
	}
	if got > 0 {
		// TODO: close containers
		return true
	}
	return false
}

// specBrainEater eats brains from headless corpses.
func specBrainEater(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetFighting() != "" || cmd != "" || ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	items := w.GetItemsInRoom(ch.GetRoomVNum())
	for _, obj := range items {
		if !obj.IsContainer() {
			continue
		}
		// TODO: check Values[3] (container flags / locked flag)
		kw := strings.ToLower(obj.GetKeywords())
		if !strings.Contains(kw, "corpse") || strings.Contains(kw, "headless") {
			continue
		}
		// TODO: do_behead(me, "corpse")
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s pulls the brain out of the head and eats it with a noisy\r\nslurp, blood and drool flying everywhere.", mobName(me)))
		// TODO: level up or increase damroll
		return true
	}
	return false
}

// specTeleportVictim teleports an attacker away.
func specTeleportVictim(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetFighting() == "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	// TODO: do_action(ch, target_name, find_command("scoff"), 0)
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s says, 'You can't harm me, mortal. Begone.'", ch.GetName()))
	// TODO: call_magic(ch, FIGHTING(ch), 0, SPELL_TELEPORT, GET_LEVEL(ch), CAST_SPELL)
	return true
}

// specConSeller sells constitution points.
func specConSeller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	arg = strings.TrimSpace(arg)
	// TODO: w.CanSee(me, ch) check
	if cmd != "list" && cmd != "buy" {
		return false
	}
	if cmd == "buy" {
		if !strings.EqualFold(arg, "con") {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s BUY CON, if you really want to do it.'", mobName(me), ch.GetName()))
			return true
		}
		// TODO: gold check, OrigCon check, apply con +1, affect_total
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s That'll be some coins. You should feel much better.. if you wake up.'", mobName(me), ch.GetName()))
		return true
	}
	if cmd == "list" {
		// TODO: calculate and show available con points
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s I can sell you constitution points.'", mobName(me), ch.GetName()))
		return true
	}
	return false
}

// npcRegen regenerates health for NPC mobs.
func npcRegen(ch *Player) {
	regenRate := 2
	ch.Health += ch.GetLevel() * regenRate
	if ch.Health > ch.GetMaxHP() {
		ch.Health = ch.GetMaxHP()
	}
}

// specTroll regenerates health over time.
func specTroll(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping || ch.GetHP() <= 0 {
		return false
	}
	if ch.GetFighting() == "" && ch.GetHP() != ch.GetMaxHP() {
		if rand.Intn(21) == 0 {
			npcRegen(ch)
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s's wounds glow brightly for a moment, then disappear!", ch.GetName()))
		}
	} else if ch.GetFighting() != "" {
		if rand.Intn(11) == 0 {
			npcRegen(ch)
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s's wounds glow brightly for a moment, then disappear!", ch.GetName()))
		}
	} else {
		return false
	}
	return true
}

// specQuanLo comments on flee/retreat commands and responds to look.
func specQuanLo(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" && me.GetPosition() > combat.PosSleeping {
		if cmd == "flee" || cmd == "retreat" || cmd == "escape" {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gossips, 'What was that, %s? This is not a shawade. Try it again. This time with fewing.'", mobName(me), ch.GetName()))
		}
		arg = strings.TrimSpace(arg)
		if (cmd == "look" || cmd == "examine") && arg != "" && strings.Contains(me.GetName(), arg) {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'What is it you seek, %s? Tell me and be gone.'", mobName(me), ch.GetName()))
		}
	}
	return false
}

// specAlienElevator moves all occupants between two rooms.
func specAlienElevator(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if cmd == "close" && strings.EqualFold(arg, "door") {
		w.roomMessage(ch.GetRoomVNum(), "The room starts to move!")
		// TODO: char_from_room / char_to_room for all occupants between 19551 and 19599
		return true
	}
	return false
}

// specWerewolf howls when fighting.
func specWerewolf(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetFighting() == "" {
		return false
	}
	if rand.Intn(10) == 0 && me.GetHP() > 0 {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s looks up and lets out a long, fierce howl.", mobName(me)))
		// TODO: send_to_zone("You hear a loud howling in the distance.", me)
	}
	return true
}

// specFieldObject checks field objects that damage room occupants.
func specFieldObject(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	// TODO: me is actually an ObjectInstance variant; check against field_objs table
	return false
}

// specPortalToTemple teleports the player to the temple.
func specPortalToTemple(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "say" && cmd != "'" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if !strings.EqualFold(arg, "setchswayno") {
		return false
	}
	sendToChar(ch, "With a blinding flash of light and a crack of thunder, you are teleported...\r\n")
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("With a blinding flash of light and a crack of thunder, %s disappears!", ch.GetName()))
	// TODO: char_from_room(ch); char_to_room(ch, w.RealRoom(8008))
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("With a blinding flash of light and a crack of thunder, %s appears!", ch.GetName()))
	return true
}

// specTurnUndead opens a portal when the player uses the right item.
func specTurnUndead(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "use" {
		w.roomMessage(ch.GetRoomVNum(), "A ray of flame bursts out of the object, consuming the undead!")
		// TODO: create/remove exits in rooms 19875/19876
		return true
	}
	return false
}

// specItoh teleports the player to room 19875.
func specItoh(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "say" && cmd != "'" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if !strings.EqualFold(arg, "itoh") {
		return false
	}
	sendToChar(ch, "\r\nWith a blinding flash of light and a crack of thunder, you are teleported...\r\n")
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("\r\nWith a blinding flash of light and a crack of thunder, %s disappears!\r\n\r\n", ch.GetName()))
	// TODO: char_from_room(ch); char_to_room(ch, w.RealRoom(19875))
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("\r\nWith a blinding flash of light and a crack of thunder, %s appears!\r\n\r\n", ch.GetName()))
	return true
}

// specMirror creates reflections and swaps players.
func specMirror(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "hit" || cmd == "kill" {
		sendToChar(ch, "You break the object into tiny pieces!")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s shatters the object into a million pieces!", ch.GetName()))
		// TODO: char_from_room(ch2), char_to_room(ch2, obj->in_room)
		// TODO: extract_obj(obj); obj_to_room(read_object(14503, VIRTUAL), obj->in_room)
		return true
	}
	if cmd == "look" {
		sendToChar(ch, "You feel pulled in a hundred different directions!")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash!", ch.GetName()))
		// TODO: swap ch and ch2
		return true
	}
	return false
}

// specProstitute offers services for gold.
func specProstitute(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	// TODO: w.CanSee(me, ch) check
	if cmd == "buy" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s I ain't for sale, just rent. Give me 5 gold for a good time.'", mobName(me), ch.GetName()))
		return true
	}
	if cmd == "list" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s For five coins, I'll show you a good time.'", mobName(me), ch.GetName()))
		return true
	}
	return false
}

// specRoach — a living cockroach that eats, grows, and reproduces.
func specRoach(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	// Starvation death (extremely rare)
	if rand.Intn(10001) == 0 && rand.Intn(10001) == 0 && ch.GetMaxHP() < 11 {
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s seems to starve to death and simply fades out of existence.", ch.GetName()))
		// TODO: extract_char(ch)
		return true
	}
	// Look for food on the ground
	items := w.GetItemsInRoom(ch.GetRoomVNum())
	for _, obj := range items {
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s feeds on %s.", ch.GetName(), obj.GetShortDesc()))
		if rand.Intn(3) == 0 {
			ch.MaxHealth += obj.GetCost() / 2
			if ch.MaxHealth > 400 {
				ch.MaxHealth = 10
				ch.Health = ch.GetMaxHP()
				w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s splits in half forming a new roach!", ch.GetName()))
				// TODO: read_mobile(23, VIRTUAL); char_to_room(new, ch->in_room)
			} else {
				if rand.Intn(2) == 0 {
					w.roomMessage(ch.GetRoomVNum(), "You hear some stretching noises.")
				} else {
					w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("You hear a strange rumbling from %s's stonach.", ch.GetName()))
				}
			}
		} else {
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("You hear %s burp.", ch.GetName()))
		}
		// TODO: extract_obj(obj)
		return true
	}
	// Random idle behaviors
	switch rand.Intn(11) {
	case 0:
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s chirps gleefully.", ch.GetName()))
	case 1:
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s changes colors and clicks happily.", ch.GetName()))
	case 2:
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s skitters around in tight circles.", ch.GetName()))
	case 3:
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("Strange purple dots appear on %s's back.", ch.GetName()))
	case 4:
		if rand.Intn(6) == 0 {
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s fades out and back in again.", ch.GetName()))
			return false
		}
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s fades out slowly with a soft swoosh.", ch.GetName()))
		// TODO: char_from_room(ch); char_to_room(ch, random_room)
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s fades in slowly, looking a bit disoriented.", ch.GetName()))
		return true
	}
	return false
}

// specMortician retrieves corpses for a fee.
func specMortician(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	cost := ch.GetLevel() * 116
	if cmd == "list" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s It will cost %d coins to retrieve your corpse.'", mobName(me), ch.GetName(), cost))
		return true
	}
	if cmd == "retrieve" {
		if ch.GetGold() < cost {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s I'm sorry, you can't afford the cost.'", mobName(me), ch.GetName()))
			return true
		}
		// TODO: iterate object_list, find corpse matching player name
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s I'm sorry, I can't find your corpse anywhere!'", mobName(me), ch.GetName()))
		return true
	}
	return false
}

// specConjured returns to its plane of existence when un-charmed.
func specConjured(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// TODO: check AFF_CHARM
	switch me.GetVNum() {
	case 81, 82, 83, 84:
		// TODO: notify master
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s returns to its own plane of existence.", mobName(me)))
	default:
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'My work here is done.'", mobName(me)))
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s disappears in a flash of white light!", mobName(me)))
	}
	// TODO: extract_char(me)
	return true
}

// specHisc dispatches to other specs based on the command.
func specHisc(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "south" {
		return specNoMoveSouth(w, ch, me, cmd, arg)
	}
	return false
}

// specRecruiter responds to kill and cast commands.
func specRecruiter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	if cmd == "kill" || cmd == "hit" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s Why don't you sign up for training? Just head south through those doors!'", mobName(me), ch.GetName()))
		return true
	}
	if cmd == "cast" || cmd == "will" {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s Hey now! None of that voodoo mumbo jumbo in my office!'", mobName(me), ch.GetName()))
		return true
	}
	return false
}

// specElementsMasterColumn teleports players based on which talismans they carry.
func specElementsMasterColumn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	newLocs := []int{1320, 1331, 1342, 1353, 1372}

	for _, ppl := range players {
		found := -1
		// TODO: check player inventory for talismans (vnum 1300-1303)
		if found >= 0 && found < len(newLocs) {
			sendToChar(ppl, "The talisman glows softly and your vision fades.\r\n")
		} else {
			sendToChar(ppl, "You feel a tingling sensation and your vision fades.\r\n")
		}
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s vanishes in a brilliant flash of light.", ppl.GetName()))
		// TODO: char_from_room(ppl); char_to_room(ppl, newLocs[found])
	}
	return true
}

// specElementsPlatforms sends all players in the room back to the master column.
func specElementsPlatforms(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, ppl := range players {
		sendToChar(ppl, "A wave of power surges through you and you feel dizzy.\r\n")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash of light.", ppl.GetName()))
		// TODO: char_from_room(ppl); char_to_room(ppl, 1314)
	}
	return true
}

// specElementsLoadCylinders manages cylinder objects for the talisman puzzle.
func specElementsLoadCylinders(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "get" {
		// TODO: do_get(ch, arg, cmd, 0)
		elementsRemoveCylinders(w)
		return true
	}
	if cmd != "drop" {
		return false
	}
	// TODO: check if cylinder already exists; load cylinder matching talisman dropped
	return true
}

// specElementsGaleruColumn checks if all four talismans are in their rooms.
func specElementsGaleruColumn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// TODO: check rooms 1360,1364,1380,1384 for talismans 1300-1303
	// If all four, teleport players to room 1389
	return false
}

// specElementsGaleruAlive teleports players if Galeru (mob 1315) is dead.
func specElementsGaleruAlive(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	// TODO: check if mob vnum 1315 exists in room
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	if len(players) > 0 {
		for _, ppl := range players {
			sendToChar(ppl, "You begin to feel very dizzy and the world around you fades...\r\n")
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash of light.", ppl.GetName()))
			// TODO: char_from_room(ppl); char_to_room(ppl, 1395)
		}
		return true
	}
	return false
}

// specElementsMinion destroys talismans and cylinders.
func specElementsMinion(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// TODO: iterate mob inventory for talismans/cylinders and destroy them
	elementsRemoveCylinders(w)
	return false
}

// specElementsGuardian charms players into fighting each other.
func specElementsGuardian(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	// TODO: charm players through song, pair non-fighting players
	return false
}

// specFlyExitUp blocks going up unless the player can fly.
func specFlyExitUp(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "up" {
		return false
	}
	// TODO: check AFF_FLY
	sendToChar(ch, "You try and jump up there but it's just too high.\r\n")
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s jumps up and down in a vain attempt to travel upwards.", ch.GetName()))
	return true
}

// elementsRemoveCylinders checks room contents and removes cylinders when talismans leave.
func elementsRemoveCylinders(w *World) {
	// TODO: check room for cylinder vnums, remove if corresponding talisman not present
}
