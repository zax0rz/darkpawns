package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func init() {
	// Chunk 1: normal_checker, ninelives, whirlpool, couch, stableboy, tipster
	RegisterSpec("normal_checker", specNormalChecker)
	RegisterSpec("ninelives", specNinelives)
	RegisterSpec("whirlpool", specWhirlpool)
	RegisterSpec("couch", specCouch)
	RegisterSpec("stableboy", specStableboy)
	RegisterSpec("tipster", specTipster)

	// Chunk 2: rescuer, pissedalchemist, remorter, assassin, tattoo1, tattoo2
	RegisterSpec("rescuer", specRescuer)
	RegisterSpec("pissedalchemist", specPissedalchemist)
	RegisterSpec("remorter", specRemorter)
	RegisterSpec("assassin", specAssassin)
	RegisterSpec("tattoo1", specTattoo1)
	RegisterSpec("tattoo2", specTattoo2)

	// Chunk 3: tattoo3, eviltrade, identifier, tattoo4, evillead, little_boy
	RegisterSpec("tattoo3", specTattoo3)
	RegisterSpec("eviltrade", specEviltrade)
	RegisterSpec("identifier", specIdentifier)
	RegisterSpec("tattoo4", specTattoo4)
	RegisterSpec("evillead", specEvilLead)
	RegisterSpec("little_boy", specLittleBoy)

	// Chunk 4: ira, take_to_jail, jail, medusa, eq_thief, portal_room
	RegisterSpec("ira", specIra)
	RegisterSpec("take_to_jail", specTakeToJail)
	RegisterSpec("jail", specJail)
	RegisterSpec("medusa", specMedusa)
	RegisterSpec("eq_thief", specEqThief)
	RegisterSpec("portal_room", specPortalRoom)

	// Chunk 5: breed_killer, carrion, bat_room, bat, no_move_east, key_seller
	RegisterSpec("breed_killer", specBreedKiller)
	RegisterSpec("carrion", specCarrion)
	RegisterSpec("bat_room", specBatRoom)
	RegisterSpec("bat", specBat)
	RegisterSpec("no_move_east", specNoMoveEast)
	RegisterSpec("key_seller", specKeySeller)

	// Chunk 6: castle_guard_east, mindflayer, backstabber, teleporter, no_move_west, no_move_north
	RegisterSpec("castle_guard_east", specCastleGuardEast)
	RegisterSpec("mindflayer", specMindflayer)
	RegisterSpec("backstabber", specBackstabber)
	RegisterSpec("teleporter", specTeleporter)
	RegisterSpec("no_move_west", specNoMoveWest)
	RegisterSpec("no_move_north", specNoMoveNorth)

	// Chunk 7: never_die, no_move_south, chosen_guard, castle_guard_down, castle_guard_up, castle_guard_north, wall_guard_ns
	RegisterSpec("never_die", specNeverDie)
	RegisterSpec("no_move_south", specNoMoveSouth)
	RegisterSpec("chosen_guard", specChosenGuard)
	RegisterSpec("castle_guard_down", specCastleGuardDown)
	RegisterSpec("castle_guard_up", specCastleGuardUp)
	RegisterSpec("castle_guard_north", specCastleGuardNorth)
	RegisterSpec("wall_guard_ns", specWallGuardNS)
}

// ================================================================
// Helpers
// ================================================================

// isOwner checks if a player owns (or is a guest of) the house at roomVNum.
// C equivalent: is_owner() in spec_procs2.c:1844-1876
func isOwner(w *World, ch *Player, roomVNum int) bool {
	if ch.IsNPC() {
		return false
	}
	i := findHouse(w.HouseControl, roomVNum)
	if i < 0 {
		return false
	}
	h := w.HouseControl[i]
	if int64(ch.GetID()) == h.Owner {
		return true
	}
	for j := 0; j < h.NumOfGuests; j++ {
		if int64(ch.GetID()) == h.Guests[j] {
			return true
		}
	}
	return false
}

// guardCanAct returns false if the guard is asleep/dead or player is immortal.
func guardCanAct(ch *Player, me *MobInstance) bool {
	if me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if !ch.IsNPC() && ch.GetLevel() >= LVL_IMMORT {
		return false
	}
	return true
}

// guardAssist checks if any mob in the room with the same spec vnum is fighting,
// and if so, joins combat. Returns true if assisted.
func guardAssist(w *World, me *MobInstance, specVNum int) bool {
	if me.GetFighting() != "" || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
		if mob == me {
			continue
		}
		if mob.GetVNum() == specVNum && mob.Fighting {
			// Find who the mob is fighting
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.FightingTarget && !pl.IsNPC() {
					me.Attack(pl, w)
					return true
				}
			}
		}
	}
	return false
}

// isOwnerGrouped checks if a player is grouped with the house owner.
func isOwnerGrouped(w *World, ch *Player, roomVNum int) bool {
	if ch.Following == "" {
		return false
	}
	leader, ok := w.GetPlayer(ch.Following)
	if !ok {
		return false
	}
	return isOwner(w, leader, roomVNum)
}

// tellFromMob sends a tell-style message from a mob to a player.
func tellFromMob(me *MobInstance, target *Player, msg string) {
	target.SendMessage(fmt.Sprintf("%s tells you, '%s'\r\n", me.GetShortDesc(), msg))
}

// mobName returns the display name for a mob — use ShortDesc for display.
func mobName(me *MobInstance) string {
	return me.GetShortDesc()
}

// ================================================================
// normal_checker — Sees non-immortals, jumps and attacks them
// ================================================================
func specNormalChecker(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetLevel() < 50 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s sees %s and jumps quite high!", mobName(me), pl.GetName()))
			sendToChar(pl, fmt.Sprintf("%s sees you and jumps high, right at you!\r\n", mobName(me)))
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// ninelives — Cat has 9 lives (using MaxMove as life counter), auto-revives
// ================================================================
func specNinelives(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetPosition() <= combat.PosSleeping && ch.GetHP() > 0 {
		return false
	}
	if cmd != "" {
		a := strings.TrimSpace(arg)
		if a == "" {
			return false
		}
		if !strings.Contains(a, "chest") {
			return false
		}
		if cmd == "open" || cmd == "look" || cmd == "examine" {
			if !ch.IsNPC() {
				me.Attack(ch, w)
			}
			return true
		}
		return false
	}
	if ch.GetFighting() == "" || ch.GetHP() > 0 {
		return false
	}
	lives := ch.GetMaxMove()
	if lives > 0 {
		if lives > 8 {
			lives = 8
		} else {
			lives--
		}
		ch.MaxMove = lives
		ch.Health = ch.GetMaxHP()
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s rises from the dead and keeps fighting!", mobName(me)))
		return true
	}
	return false
}

// ================================================================
// whirlpool — Sucks players in and teleports them to random rooms 4600-4699
// ================================================================
func specWhirlpool(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	specOccurred := false
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() {
			// TODO: pick random room 4600-4699 that isn't private/godroom/death/nomob
			// char_from_room(pl); char_to_room(pl, to_room)
			sendToChar(pl, "A ravaging whirlpool sucks you under!\r\n")
			sendToChar(pl, "You finally surface, sputtering...\r\n\r\n")
			// TODO: look_at_room(vict, 0)
			specOccurred = true
		}
	}
	return specOccurred
}

// ================================================================
// couch — Mimic attacks when player looks at couch
// ================================================================
const mimicRoomVnum = 5798

func specCouch(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		return false
	}
	a := strings.TrimSpace(arg)
	if !((cmd == "look" || cmd == "examine") && strings.Contains(a, "couch")) {
		return false
	}
	for _, obj := range w.GetItemsInRoom(me.GetRoomVNum()) {
		if strings.Contains(obj.GetKeywords(), "couch") {
			w.RemoveItemFromRoom(obj, me.GetRoomVNum())
			// Find mimic mob in its home room and move it here
			playerRoom := me.GetRoomVNum()
			for _, m := range w.GetMobsInRoom(mimicRoomVnum) {
				m.SetRoom(playerRoom)
				break
			}
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("Starved and needing food to make more pillows, the couch attacks!"))
			sendToChar(ch, "Starved and needing food to make more pillows, the couch attacks you!\r\n\r\n")
			for _, m := range w.GetMobsInRoom(playerRoom) {
				if m.GetRoomVNum() == playerRoom && m != me {
					m.Attack(ch, w)
					break
				}
			}
			return true
		}
	}
	return false
}

// ================================================================
// stableboy — Buy/list/stable/collect horses
// ================================================================
const horseVnum = 8021

func specStableboy(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	a := strings.TrimSpace(arg)

	switch cmd {
	case "list":
		tellFromMob(me, ch, "You can buy a horse for 300 gold coins.")
		return true

	case "buy":
		if a == "" || !strings.Contains(a, "horse") {
			tellFromMob(me, ch, "Buy what, fine adventurer?")
			return true
		}
		if ch.GetGold() < 300 {
			tellFromMob(me, ch, "You can't afford a mount!")
			return true
		}
		ch.SetGold(ch.GetGold() - 300)
		// TODO: horse = read_mobile(HORSE_VNUM, VIRTUAL); char_to_room(horse, ch->in_room)
		// TODO: SET_BIT(AFF_FLAGS(horse), AFF_CHARM); add_follower(horse, ch)
		// TODO: GET_MOVE(horse) = 230; GET_MAX_MOVE(horse) = 230
		tellFromMob(me, ch, "That'll be 300 coins, treat'er well")
		return true

	case "stable":
		// TODO: unmount logic, stop_follower, set rent time/cost
		// GET_MOUNT_RENT_TIME(ch) = time(0); GET_MOUNT_NUM(ch) = mob vnum; GET_MOUNT_COST_DAY(ch) = 5
		tellFromMob(me, ch, "How do you expect to stable a mount, you don't have a mount!")
		return true

	case "collect":
		// TODO: retrieve stabled horse, charge days * cost
		tellFromMob(me, ch, "Hey now, you need to have stabled a mount to pick one up.")
		return true
	}
	return false
}

// ================================================================
// tipster — Random tip messages on pulse
// ================================================================
func specTipster(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}

	tips := []string{
		"For ansi color, type COLOR COMPLETE!",
		"Wargs make a tasty meal if you CARVE their corpse.",
		"AUTO EXIT will show you the exits for every room.",
		"It's always safest to quit in the temple.",
		"You're allowed to play up to 3 characters at once.",
		"You can hire a mercenary by giving him 100 coins.",
		"A bribe of a couple hundred coins will make a guard look the other way while you fight.  Guards will attack you if you fight in front of them.",
		"Use the CONSIDER command!",
		"If you don't like something, use the IDEA command.",
		"If you see something out of place, use the BUG command.",
		"If you see something spelled incorrectly, use the TYPO command.",
		"Use an identify scroll on yourself to see your numerical statistics.  (Available at your local magick shop.)",
		"Check out HELP ALIAS to see how to abbreviate commands or do multiple commands at once.",
	}

	n := randN(len(tips))
	sendToChar(ch, fmt.Sprintf("%s says '%s'\r\n", mobName(me), tips[n]))
	return false
}

// ================================================================
// rescuer — Rescues players being attacked in the same room
// ================================================================
func specRescuer(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	if ch.GetFighting() != "" || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetLevel() < 50 && pl.GetFighting() != "" {
			sendToChar(ch, fmt.Sprintf("%s says 'Fear not! I shall rescue you!'\r\n", mobName(me)))
			w.doRescue(ch, me, "rescue", pl.GetName())
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// pissedalchemist — When low on HP, throws a potion healing cloud
// ================================================================
func specPissedalchemist(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if !ch.IsNPC() {
		return false
	}
	if ch.GetFighting() == "" || randRange(1, 4) != 1 {
		return false
	}
	if ch.GetHP() > ch.GetMaxHP()/4 {
		return false
	}
	// TODO: find a potion in zone 194, give it
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s throws a potion at the ground, and a large globe of thick black mushroom cloud creeps up toward the heavens!", mobName(me)))
	ch.Health = ch.GetMaxHP()
	ch.Move = ch.GetMaxMove()
	return true
}

// ================================================================
// remorter — Remort info NPC, random tips on pulse, can buy remort
// ================================================================
func specRemorter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		a := strings.TrimSpace(arg)
		if cmd == "buy" && strings.Contains(a, "remort") {
			sendToChar(ch, "HA!  You must be insane!\r\n")
			return true
		}
		if cmd == "offer" || cmd == "donate" || cmd == "give" {
			sendToChar(ch, "The remorter tells you 'There is a collection box inside the temple'\r\n")
			return true
		}
		if cmd == "list" {
			sendToChar(ch, "The remorter tells you 'If you want to remort, ask me to BUY REMORT.'\r\n")
			return true
		}
		return false
	}
	if randN(6) != 0 {
		return false
	}
	msgs := []string{
		"So you wish to remort?",
		"Remorting allows you to keep your skills.",
		"Remorting grants an additional 2 hitpoints per level!",
		"Remorting costs 10 levels and much experience.",
		"You can only remort once per 10 levels.",
		"To remort, just BUY REMORT off me.",
		"Remorting is not for the faint of heart!",
	}
	n := randN(len(msgs))
	sendToChar(ch, fmt.Sprintf("The remorter says '%s'\r\n", msgs[n]))
	return false
}

// ================================================================
// assassin — Bodyguard-ish: attacks whoever's fighting master
// ================================================================
func specAssassin(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if !ch.IsNPC() {
		return false
	}
	// TODO: check if master is fighting; if so, attack master's opponent
	// For now: stub
	return false
}

// ================================================================
// tattoo1 — Remove scarab tattoo to fully heal
// ================================================================
func specTattoo1(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "remove" && strings.Contains(arg, "tattoo") {
		if ch.GetFighting() != "" {
			sendToChar(ch, "You can't do that while fighting!\r\n")
		} else {
			// Message for other players in room
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s focuses on removing a scarab tattoo...", ch.GetName()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s focuses on removing %s scarab tattoo...", ch.GetName(), hisHer(ch.GetSex())))
			ch.Health = ch.GetMaxHP()
			ch.Move = ch.GetMaxMove()
			if obj, err := w.SpawnObject(7103, ch.GetRoom()); err == nil {
				if ch.Inventory != nil {
					ch.Inventory.AddItem(obj)
				}
			}
		}
		return true
	}
	return false
}

// ================================================================
// tattoo2 — Remove snake tattoo, automatically passes to another player
// ================================================================
func specTattoo2(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "remove" && strings.Contains(arg, "tattoo") {
		if ch.GetFighting() != "" {
			sendToChar(ch, "You can't do that while fighting!\r\n")
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s focuses on removing a tattoo...", ch.GetName()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s focuses on removing %s tattoo...", ch.GetName(), hisHer(ch.GetSex())))
			ch.Health = ch.GetMaxHP()
			ch.Move = ch.GetMaxMove()
			if obj, err := w.SpawnObject(7104, ch.GetRoom()); err == nil {
				// Give to another player in the room
				for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
					if !pl.IsNPC() && pl != ch {
						if pl.Inventory != nil {
							pl.Inventory.AddItem(obj)
						}
						break
					}
				}
			}
		}
		return true
	}
	return false
}

// ================================================================
// tattoo3 — Buy a cheap 'tramp stamp' tattoo
// ================================================================
func specTattoo3(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "buy" && strings.Contains(arg, "tattoo") {
		if ch.GetGold() < 500 {
			sendToChar(ch, "You don't have enough gold!\r\n")
			return true
		}
		ch.SetGold(ch.GetGold() - 500)
		if obj, err := w.SpawnObject(7103, ch.GetRoom()); err == nil {
			if ch.Inventory != nil {
				ch.Inventory.AddItem(obj)
			}
		}
		sendToChar(ch, "You buy a cheap 'tramp stamp' tattoo.\r\n")
		return true
	}
	return false
}

// ================================================================
// eviltrade — Trade keys for experience points
// ================================================================
func specEviltrade(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd != "trade" && cmd != "give" {
		return false
	}
	if cmd == "trade" {
		// Check for gold watch (VNum 13111) in inventory
		if ch.Inventory != nil {
			var toRemove []*ObjectInstance
			for _, item := range ch.Inventory.Items {
				if item.GetVNum() == 13111 {
					ch.Exp += (ch.GetLevel() * 200)
					toRemove = append(toRemove, item)
				}
			}
			for _, item := range toRemove {
				ch.Inventory.RemoveItem(item)
			}
			if len(toRemove) > 0 {
				sendToChar(ch, fmt.Sprintf("You trade your key for %d experience.\r\n", ch.GetLevel()*200*len(toRemove)))
				w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s trades in some keys for experience!", ch.GetName()))
			} else {
				sendToChar(ch, "You don't have anything to trade.\r\n")
			}
		}
		return true
	}
	return false
}

// ================================================================
// identifier — Identify items/characters for a level-based fee
// ================================================================
func specIdentifier(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "identify" {
		return false
	}
	if ch.IsNPC() {
		return false
	}
	a := strings.TrimSpace(arg)
	if a == "" {
		sendToChar(ch, "Identify what?\r\n")
		return true
	}
	cost := ch.GetLevel() * 50
	if ch.GetGold() < cost {
		sendToChar(ch, "You can't afford it!\r\n")
		return true
	}
	ch.SetGold(ch.GetGold() - cost)
	// Look up item in inventory by name
	if ch.Inventory != nil {
		items := ch.Inventory.FindItems(a)
		if len(items) > 0 {
			obj := items[0]
			sendToChar(ch, fmt.Sprintf("%s studies %s carefully...\r\n", mobName(me), obj.GetShortDesc()))
			// Cast identify on the object
			spells.Cast(ch, obj, spells.SpellIdentify, ch.GetLevel(), nil)
			return true
		}
	}
	sendToChar(ch, "No such thing around.\r\n")
	return true
}

// ================================================================
// tattoo4 — Remove shadowy tattoo to fully heal
// ================================================================
func specTattoo4(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "remove" && strings.Contains(arg, "tattoo") {
		if ch.GetFighting() != "" {
			sendToChar(ch, "You can't do that while fighting!\r\n")
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s concentrates and the tattoo disappears...", ch.GetName()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s's tattoo glows brightly and then fades away...", ch.GetName()))
			ch.Health = ch.GetMaxHP()
			ch.Move = ch.GetMaxMove()
		}
		return true
	}
	return false
}

// ================================================================
// evillead — Evil-leaning mob attacks evil (alignment < 0) players
// ================================================================
func specEvilLead(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if !ch.IsNPC() {
		return false
	}
	if randRange(1, 100) > 5 {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetAlignment() < 0 {
			sendToChar(ch, fmt.Sprintf("%s says 'You're an evil one! That won't be allowed here!'\r\n", mobName(me)))
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// little_boy — Give a flower, get a note
// ================================================================
const littleBoyVnum = 2767

func specLittleBoy(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "give" && strings.Contains(arg, "flower") {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gives a flower to the little boy.", ch.GetName()))
		sendToChar(ch, "The boy smiles and hands you a small note.\r\n")
		if obj, err := w.SpawnObject(7107, ch.GetRoom()); err == nil {
			if ch.Inventory != nil {
				ch.Inventory.AddItem(obj)
			}
		}
		sendToChar(ch, "The little boy runs off!\r\n")
		// Remove the little boy mob from the room
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob != me && mob.GetVNum() == littleBoyVnum {
				mob.SetStatus("dead")
				break
			}
		}
		return true
	}
	return false
}

// ================================================================
// ira — Angry mob, 3% chance per pulse to attack random player
// ================================================================
func specIra(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if pl.IsNPC() || pl == ch || pl.GetFighting() != "" {
			continue
		}
		if randN(5) != 0 {
			continue
		}
		if randN(31) == 0 {
			sendToChar(ch, fmt.Sprintf("%s says 'I don't like you, and you'd better leave before I make you!'\r\n", mobName(me)))
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// take_to_jail — Grabs non-immortal players and drags them to jail
// ================================================================
const jailRoomVnum = 2014

func specTakeToJail(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if ch.GetPosition() <= combat.PosSleeping || me.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if pl.IsNPC() || pl == ch || pl.GetFighting() != "" {
			continue
		}
		if pl.GetLevel() >= 50 {
			continue
		}
		if randN(6) != 0 {
			continue
		}
		sendToChar(ch, fmt.Sprintf("%s says 'You're under arrest!'\r\n", mobName(me)))
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s grabs %s and drags them off to jail!", mobName(me), pl.GetName()))
		pl.SetRoom(jailRoomVnum)
		w.roomMessage(jailRoomVnum, fmt.Sprintf("%s drags %s into the room and throws them in a cell!", mobName(me), pl.GetName()))
		sendToChar(ch, fmt.Sprintf("%s says 'You'll rot in there!'\r\n", mobName(me)))
		return true
	}
	return false
}

// ================================================================
// jail — Say "release" to be set free (costs 25% gold + 25% move)
// ================================================================
func specJail(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "say" && strings.Contains(arg, "release") {
		gold := int(ch.GetGold() * 25 / 100)
		move := int(ch.GetMove() * 25 / 100)
		if gold < 1 {
			gold = 1
		}
		if move < 1 {
			move = 1
		}
		ch.SetGold(ch.GetGold() - gold)
		ch.Move = ch.GetMove() - move
		sendToChar(ch, "A guard opens the cell door and lets you out.\r\n")
		ch.SetRoom(8117) // release room per C source
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s is released from jail.", ch.GetName()))
		ch.SetPosition(combat.PosStanding)
		return true
	}
	return false
}

// ================================================================
// medusa — Snake-hair gorgon: look at her and risk petrification
// ================================================================
func specMedusa(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" && cmd != "look" && cmd != "examine" {
		return false
	}
	if cmd != "" {
		a := strings.TrimSpace(arg)
		if a == "" {
			return false
		}
		if !strings.Contains(a, "medusa") && !strings.Contains(a, "gorgon") {
			return false
		}
		if !ch.IsNPC() {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s sees you looking and snaps out with snake-like speed!", mobName(me)))
			// Petrify all players in room
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if !pl.IsNPC() {
					w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s's gaze catches %s, who starts to turn to stone!", mobName(me), pl.GetName()))
				}
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if !pl.IsNPC() {
					// TODO: save check SAVE_PETRIFY — if fail, hit
					me.Attack(pl, w)
				}
			}
		}
		return true
	}
	if randN(6) != 0 {
		return false
	}
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s's snake-hair writhes around and it snaps at the air!", mobName(me)))
	return false
}

// ================================================================
// eq_thief — Steals non-rent items when you give/offer something (20% chance)
// ================================================================
func specEqThief(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	if cmd != "give" && cmd != "offer" {
		return false
	}
	if ch.IsNPC() {
		return false
	}
	if randN(101) > 20 {
		return false
	}
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	// TODO: iterate ch->carrying, for each item visible that has GET_OBJ_RENT(obj) == 0:
	//   obj_from_char(obj); extract_obj(obj); count++
	sendToChar(ch, "The eq thief steals 0 items!\r\n")
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gets stripped of equipment by the eq thief!", ch.GetName()))
	ch.Move = 0
	ch.Health = 1
	return true
}

// ================================================================
// portal_room — Random teleport on move command
// ================================================================
func specPortalRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "north" || cmd == "south" || cmd == "east" || cmd == "west" || cmd == "up" || cmd == "down" {
		if !ch.IsNPC() && randN(2) != 0 {
			sendToChar(ch, "A shimmering portal appears and sucks you in!\r\n")
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s stumbles through a shimmering portal!", ch.GetName()))
			// TODO: char_from_room(ch); char_to_room(ch, get_random_room())
			sendToChar(ch, "You tumble out into a strange place...\r\n")
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s appears from a shimmering portal!", ch.GetName()))
			return true
		}
	}
	return false
}

// ================================================================
// breed_killer — 5% chance per tick to screech and attack
// ================================================================
func specBreedKiller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if ch.GetPosition() <= combat.PosSleeping || ch.GetHP() < 0 {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}
	if ch.IsNPC() {
		return false
	}
	if randRange(1, 100) > 5 {
		return false
	}
	// Attack players in room
	for _, victim := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if victim.GetLevel() >= 50 {
			continue
		}
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s lets out a blood-chilling screech!", mobName(me)))
		me.Attack(victim, w)
		return true
	}
	return false
}

// ================================================================
// carrion — While fighting, 20% chance to attack a bystander
// ================================================================
func specCarrion(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetHP() < 0 {
		return false
	}
	if ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch.GetFighting() == "" {
		return false
	}
	if randN(5) != 0 {
		return false
	}
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tears into its victim with renewed fury!", mobName(me)))
	for _, vict := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !vict.IsNPC() && vict.GetName() != ch.GetName() {
			if randRange(1, vict.GetLevel()) <= me.GetLevel() {
				me.Attack(vict, w)
				return true
			}
		}
	}
	return false
}

// ================================================================
// bat_room — Bats in room block movement if bat object present
// ================================================================
func specBatRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd != "north" && cmd != "south" && cmd != "east" && cmd != "west" {
		return false
	}
	found := false
	for _, obj := range w.GetItemsInRoom(me.GetRoomVNum()) {
		if strings.Contains(obj.GetKeywords(), "bat") {
			found = true
			break
		}
	}
	if found {
		sendToChar(ch, "The bats swarm around you, blocking your escape!\r\n")
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s is swarmed by bats and forced back!", ch.GetName()))
		return true
	}
	return false
}

// ================================================================
// bat — Bat swoops and attacks when player looks at "dripping"
// ================================================================
func specBat(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd != "look" && cmd != "examine" {
		return false
	}
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	if strings.Contains(a, "dripping") && randN(4) == 0 {
		sendToChar(ch, "A bat swoops down and attacks you!\r\n")
		me.Attack(ch, w)
		return true
	}
	return false
}

// ================================================================
// no_move_east — Blocks movement east
// ================================================================
func specNoMoveEast(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "east" {
		sendToChar(ch, "You try to go east but are blocked by a heavy object.\r\n")
		return true
	}
	return false
}
// ================================================================
// specKeySeller — Sells an old rusty key for 50 gold
// ================================================================
func specKeySeller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "list" {
		sendToChar(ch, "You can buy an 'old rusty key' for 50 gold.\r\n")
		return true
	}
	if cmd == "buy" {
		a := strings.TrimSpace(arg)
		if a == "" {
			return false
		}
		if strings.Contains(arg, "key") {
			if ch.GetGold() < 50 {
				sendToChar(ch, "You don't have enough gold!\r\n")
				return true
			}
			ch.SetGold(ch.GetGold() - 50)
			if obj, err := w.SpawnObject(5181, ch.GetRoom()); err == nil {
				if ch.Inventory != nil {
					ch.Inventory.AddItem(obj)
				}
			}
			sendToChar(ch, "You buy an old rusty key.\r\n")
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s buys an old rusty key.", ch.GetName()))
			return true
		}
	}
	return false
}

// ================================================================
// specCastleGuardEast — Blocks movement east into the castle.
// C equivalent: castle_guard_east in spec_procs2.c:1934-1994
// ================================================================
func specCastleGuardEast(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || !guardCanAct(ch, me) {
		return false
	}

	if cmd == "east" && !isOwner(w, ch, me.GetRoomVNum()+2) {
		if isOwnerGrouped(w, ch, me.GetRoomVNum()+2) {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s snaps to attention as %s passes.", me.GetShortDesc(), ch.GetName()))
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s yells, 'Stay outta there!'", me.GetShortDesc()))
			me.Attack(ch, w)
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.Fighting || mob.FightingTarget == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.FightingTarget && !pl.IsNPC() {
					me.Attack(pl, w)
					return true
				}
			}
		}
	}

	return false
}

// ================================================================
// specMindflayer — Drains intelligence from players
// ================================================================
func specMindflayer(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetLevel() < 50 && randN(5) == 0 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s stares at %s with hollow, empty eyes!", mobName(me), pl.GetName()))
			// TODO: drain intelligence (GET_INT) by 1
			sendToChar(pl, "You feel your intelligence draining away...\r\n")
			return true
		}
	}
	return false
}

// ================================================================
// specBackstabber — Backstabs unsuspecting players
// ================================================================
func specBackstabber(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetFighting() == "" && randN(3) == 0 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("From the shadows, %s backstabs %s!", mobName(me), pl.GetName()))
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// specTeleporter — Picks random room and teleports players there
// ================================================================
func specTeleporter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}
	if !ch.IsNPC() && randN(4) == 0 {
		// TODO: pick random room, ensure not private/godroom/death/nomob
		// char_from_room(ch); char_to_room(ch, random_room)
		sendToChar(ch, "You are suddenly yanked through the fabric of reality!\r\n")
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s suddenly vanishes!", ch.GetName()))
		return true
	}
	return false
}

// ================================================================
// specNoMoveWest — Blocks movement west
// ================================================================
func specNoMoveWest(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "west" {
		sendToChar(ch, "You try to go west but are blocked by a heavy object.\r\n")
		return true
	}
	return false
}

// ================================================================
// specNoMoveNorth — Blocks movement north
// ================================================================
func specNoMoveNorth(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "north" {
		sendToChar(ch, "You try to go north but are blocked by a heavy object.\r\n")
		return true
	}
	return false
}

// ================================================================
// specNeverDie — Revives at full HP when killed
// ================================================================
func specNeverDie(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if ch.GetPosition() <= combat.PosDead && ch.GetHP() > 0 {
		return false
	}
	if ch.GetHP() <= 0 && ch.GetPosition() <= combat.PosMortally {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s refuses to die!", mobName(me)))
		ch.Health = ch.GetMaxHP()
		return true
	}
	return false
}

// ================================================================
// specNoMoveSouth — Blocks movement south
// ================================================================
func specNoMoveSouth(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd == "south" {
		sendToChar(ch, "You try to go south but are blocked by a heavy object.\r\n")
		return true
	}
	return false
}

// ================================================================
// specChosenGuard — Guards the chosen, attacks players who fight near it
// ================================================================
func specChosenGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.GetPosition() <= combat.PosSleeping || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if cmd != "" {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetFighting() != "" {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says 'None shall harm the chosen!'", mobName(me)))
			me.Attack(pl, w)
			return true
		}
	}
	return false
}

// ================================================================
// specCastleGuardDown — Blocks movement down into the castle.
// C equivalent: castle_guard_down in spec_procs2.c:2123-2184
// ================================================================
func specCastleGuardDown(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || !guardCanAct(ch, me) {
		return false
	}

	if cmd == "down" && !isOwner(w, ch, me.GetRoomVNum()+2) {
		if isOwnerGrouped(w, ch, me.GetRoomVNum()+2) {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s moves aside and allows %s to pass.", me.GetShortDesc(), ch.GetName()))
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s states, 'Thou shalt not pass.'", me.GetShortDesc()))
			me.Attack(ch, w)
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.Fighting || mob.FightingTarget == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.FightingTarget && !pl.IsNPC() {
					me.Attack(pl, w)
					return true
				}
			}
		}
	}

	return false
}

// ================================================================
// specCastleGuardUp — Blocks movement up into the castle.
// C equivalent: castle_guard_up in spec_procs2.c:2186-2259
// Uses +1 for the house check (vs +2 for other guards).
// ================================================================
func specCastleGuardUp(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || !guardCanAct(ch, me) {
		return false
	}

	if cmd == "up" && !isOwner(w, ch, me.GetRoomVNum()+1) {
		// Group check: uses current room, not +1
		if isOwnerGrouped(w, ch, me.GetRoomVNum()) {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s moves aside and allows %s to pass.", me.GetShortDesc(), ch.GetName()))
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s states, 'Thou shalt not pass.'", me.GetShortDesc()))
			me.Attack(ch, w)
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.Fighting || mob.FightingTarget == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.FightingTarget && !pl.IsNPC() {
					me.Attack(pl, w)
					return true
				}
			}
		}
	}

	return false
}

// ================================================================
// specCastleGuardNorth — Blocks movement north into the castle.
// C equivalent: castle_guard_north in spec_procs2.c:2078-2122
// ================================================================
func specCastleGuardNorth(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || !guardCanAct(ch, me) {
		return false
	}

	if cmd == "north" && !isOwner(w, ch, me.GetRoomVNum()+2) {
		if isOwnerGrouped(w, ch, me.GetRoomVNum()+2) {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s moves aside and allows %s to pass.", me.GetShortDesc(), ch.GetName()))
		} else {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s states, 'Thou shalt not pass.'", me.GetShortDesc()))
			me.Attack(ch, w)
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.Fighting || mob.FightingTarget == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.FightingTarget && !pl.IsNPC() {
					me.Attack(pl, w)
					return true
				}
			}
		}
	}

	return false
}

// ================================================================
// specWallGuardNS — Patrols north-south corridor, walks wall.
// C equivalent: wall_guard_ns in spec_procs2.c:2260-2310
// Uses package-level state for patrol direction and talk flag.
// ================================================================

// Direction constants for mob movement specs.
const (
	DIR_NORTH = 1
	DIR_SOUTH = 2
	DIR_EAST  = 3
	DIR_WEST  = 4
	DIR_UP    = 5
	DIR_DOWN  = 6
)

var (
	wallGuardDirToMove int
	wallGuardTalk      bool = true
)

func specWallGuardNS(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetFighting() != "" {
		return false
	}

	room := w.GetRoomInWorld(me.GetRoomVNum())
	if room == nil {
		return false
	}

	_, hasNorth := room.Exits["north"]
	_, hasSouth := room.Exits["south"]

	if hasSouth && !hasNorth {
		wallGuardDirToMove = DIR_NORTH
	}
	if hasNorth && !hasSouth {
		wallGuardDirToMove = DIR_SOUTH
	}

	// Walk the wall: move the mob
	switch wallGuardDirToMove {
	case DIR_NORTH:
		if exit, ok := room.Exits["north"]; ok {
			me.SetRoom(exit.ToRoom)
		}
	case DIR_SOUTH:
		if exit, ok := room.Exits["south"]; ok {
			me.SetRoom(exit.ToRoom)
		}
	}

	// Greet church guard (VNum 8020) when encountered on patrol
	for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
		if mob == me {
			continue
		}
		if mob.IsNPC() && mob.GetVNum() == 8020 && wallGuardTalk {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s snaps to attention and salutes %s!", me.GetShortDesc(), mob.GetShortDesc()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'Hello gents!'", me.GetShortDesc()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s nods at %s.", mob.GetShortDesc(), me.GetShortDesc()))
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'On your way, soldier!'", mob.GetShortDesc()))
			wallGuardTalk = false
		}
	}

	wallGuardTalk = true
	return false
}
