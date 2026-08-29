package game

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
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
// ch is nil during autonomous AI ticks (no player is present to check).
func guardCanAct(ch *Player, me *MobInstance) bool {
	if me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch == nil {
		return true
	}
	if !ch.IsNPC() && ch.GetLevel() >= LVL_IMMORT {
		return false
	}
	return true
}

// isOwnerGrouped checks if a player is grouped with the house owner.
func isOwnerGrouped(w *World, ch *Player, roomVNum int) bool {
	if ch.GetFollowing() == "" {
		return false
	}
	leader, ok := w.GetPlayer(ch.GetFollowing())
	if !ok {
		return false
	}
	return isOwner(w, leader, roomVNum)
}

// tellFromMob sends a tell-style message from a mob to a player.
func tellFromMob(me *MobInstance, target *Player, msg string) {
	target.SendMessage(fmt.Sprintf("%s tells you, '%s'\r\n", cap(me.GetShortDesc()), msg))
}

// mobName returns the display name for a mob — use ShortDesc for display.
func mobName(me *MobInstance) string {
	return me.GetShortDesc()
}

// rescuerNumber is the C number(1, 101) seam for focused rescuer tests. The
// production value remains the process-wide deterministic game stream.
var rescuerNumber = dprng.Number

// ================================================================
// normal_checker — Sees non-immortals, jumps and attacks them
// ================================================================
func specNormalChecker(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// ch is nil during autonomous mob activity (mobileActivityForMob); in the
	// original C (spec_procs2.c:162), ch IS the mob on that path, so these
	// guards are about the mob's own state (me), not a triggering player's.
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	if me.GetFighting() != "" {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetLevel() < LVL_IMMORT {
			Act(w, true, me, pl, nil, nil, "$n sees $N and jumps quite high!", "", ToNotVict)
			Act(nil, false, me, pl, nil, nil, "$n sees you and jumps high, right at you!", "", ToVict)
			if err := w.mobHit(me, pl); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
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
				if err := me.Attack(ch, w); err != nil {
					slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
				}
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
		ch.SetMaxMove(lives)
		ch.SetHP(ch.GetMaxHP())
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
			// Pick random room 4600-4699 that isn't private/godroom/death/nomob
			var toRoom int
			for i := 0; i < 100; i++ {
				// #nosec G404 — game RNG, not cryptographic
				// #nosec G404
				candidate := dprng.Number(4600, 4699)
				r := w.GetRoomInWorld(candidate)
				if r == nil {
					continue
				}
				if w.roomHasFlag(candidate, "private") || w.roomHasFlag(candidate, "godroom") ||
					w.roomHasFlag(candidate, "death") || w.roomHasFlag(candidate, "nomob") {
					continue
				}
				toRoom = candidate
				break
			}
			if toRoom == 0 {
				continue
			}
			pl.SetRoom(toRoom)
			sendToChar(pl, "A ravaging whirlpool sucks you under!\r\n")
			sendToChar(pl, "You finally surface, sputtering...\r\n\r\n")
			w.LookAtRoom(pl, false)
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
	if (cmd != "look" && cmd != "examine") || !strings.Contains(a, "couch") {
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
			w.roomMessage(me.GetRoomVNum(), "Starved and needing food to make more pillows, the couch attacks!")
			sendToChar(ch, "Starved and needing food to make more pillows, the couch attacks you!\r\n\r\n")
			for _, m := range w.GetMobsInRoom(playerRoom) {
				if m.GetRoomVNum() == playerRoom && m != me {
					if err := m.Attack(ch, w); err != nil {
						slog.Warn("Attack failed in spec proc", "mob", m.GetName(), "error", err)
					}
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
	if ch == nil || ch.IsNPC() {
		return false
	}
	a := strings.TrimSpace(arg)

	switch cmd {
	case "list":
		tellFromMob(me, ch, "You can buy a horse for 300 gold coins.")
		return true

	case "buy":
		if a != "horse" {
			tellFromMob(me, ch, "Buy what, fine adventurer?")
			return true
		}
		if w.NumFollowers(ch.Name) >= ch.GetCha()/2 {
			sendToChar(ch, "You can't have any more followers!")
			return true
		}
		if ch.GetGold() < 300 {
			tellFromMob(me, ch, "You can't afford a mount!")
			return true
		}
		horse, err := w.spawnMobQuiet(horseVnum, ch.GetRoom())
		if err != nil {
			tellFromMob(me, ch, "Sorry we are all out of mounts at the moment, try again later.")
			return true
		}
		horse.SetAffected(affCharm)
		Act(w, false, horse, me, nil, nil, "$N brings $n up from the stables out back.", "", ToRoom)
		AddFollowerMob(w, horse, ch)
		horse.Runtime.Horse = &HorseState{CarryWeight: 1000, CarryNumber: 100, Move: 230, MaxMove: 230}
		ch.SetGold(ch.GetGold() - 300)
		tellFromMob(me, ch, "That'll be 300 coins, treat'er well")
		return true

	case "stable":
		var horse *MobInstance
		if ch.IsMounted() {
			// Find the mount mob and unmount
			for _, m := range w.GetMobsInRoom(ch.GetRoom()) {
				if m.GetMountRider() == ch.Name {
					horse = m
					Unmount(ch, horse)
					horse.RemoveAffected(affMounted)
					ch.SetAffect(affMounted, false)
					break
				}
			}
		} else {
			// Find an unmounted mountable follower in the room.
			for _, m := range w.GetMobsInRoom(ch.GetRoom()) {
				if m.GetFollowing() == ch.Name && m.HasFlag("mountable") && !m.IsMountedMob() {
					horse = m
					break
				}
			}
		}
		if horse == nil {
			tellFromMob(me, ch, "How do you expect to stable a mount, you don't have a mount!")
			return true
		}
		horse.RemoveAffected(affCharm)
		StopFollowerMob(w, horse)
		ch.MountRentTime = time.Now().Unix()
		ch.MountVNum = horse.VNum
		ch.MountCostDay = 5
		Act(w, false, horse, me, nil, nil, "$N takes $n out back to the stables.", "", ToRoom)
		w.ExtractMob(horse)
		tellFromMob(me, ch, fmt.Sprintf("I will take good care of 'em, for %d coins a day.", ch.MountCostDay))
		return true

	case "collect":
		if ch.MountVNum == 0 {
			tellFromMob(me, ch, "Hey now, you need to have stabled a mount to pick one up.")
			return true
		}
		rentDuration := time.Now().Unix() - ch.MountRentTime
		days := int(rentDuration / 86400)
		if days < 1 {
			days = 1
		}
		cost := ch.MountCostDay * days
		if cost > ch.GetGold() {
			tellFromMob(me, ch, fmt.Sprintf("Hey man, you can't afford the %d gold you need to get your mount outa' hock.", cost))
			return true
		}
		horse, err := w.spawnMobQuiet(ch.MountVNum, ch.GetRoom())
		if err != nil {
			tellFromMob(me, ch, "Sorry, we are unable to gather your mount, try back later.")
			return true
		}
		ch.MountVNum = 0
		ch.MountCostDay = 0
		ch.MountRentTime = 0
		horse.SetAffected(affCharm)
		Act(w, false, horse, me, nil, nil, "$N brings $n up from the stables out back.", "", ToRoom)
		AddFollowerMob(w, horse, ch)
		horse.Runtime.Horse = &HorseState{CarryWeight: 1000, CarryNumber: 100, Move: 230, MaxMove: 230}
		ch.SetGold(ch.GetGold() - cost)
		tellFromMob(me, ch, fmt.Sprintf("Here ya go pal, all patted down and ready to go... cost ya %d to keep 'em here.", cost))
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
// rescuer — a mob defends a fellow mob that's being attacked by a
// non-NPC, by attacking that attacker.
// Source: src/spec_procs2.c:523 SPECIAL(rescuer) — `ch` there is the mob
// itself during autonomous activity, and the proc scans for another NPC
// `i` fighting a non-NPC, then does `do_rescue(ch, GET_NAME(i), 0, 1)`
// (ch defends i). The Go port previously misread this as "rescue a
// player" and routed through the player-only doRescue(), which both
// diverged from the real behavior and crashed on the nil ch passed in
// during autonomous ticks.
// ================================================================
func specRescuer(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	if mobRescuerIsReciprocallyFighting(w, me) {
		return false
	}

	allies := w.GetMobsInRoom(me.GetRoomVNum())
	sort.Slice(allies, func(i, j int) bool { return allies[i].GetID() < allies[j].GetID() })
	for _, ally := range allies {
		if ally == nil || ally.GetID() == me.GetID() || ally.GetFighting() == "" {
			continue
		}
		// C's GET_MOB_SPEC(i) != rescuer gate is on the ally, not on the
		// rescuer. A second rescuer may therefore be selected only when it is
		// assigned some other procedure (or no procedure at all).
		if MobSpecAssign[ally.GetVNum()] == "rescuer" {
			continue
		}
		if mobRescuerPlayer(w, ally.GetFighting(), me.GetRoomVNum()) == nil {
			continue
		}
		// SPECIAL(rescuer) unconditionally returns TRUE after calling
		// do_rescue(), including when do_rescue rejects its short-description
		// argument or its 1-in-101 success roll.
		mobDoRescue(w, me, ally)
		return true
	}
	return false
}

// mobRescuerIsReciprocallyFighting implements the exact early return in
// src/spec_procs2.c:528-529. C rejects the special only when the rescuer's
// current opponent points back at it; a one-way/stale FIGHTING pointer still
// reaches do_rescue and is then cleared by that procedure.
func mobRescuerIsReciprocallyFighting(w *World, me *MobInstance) bool {
	targetName := me.GetFighting()
	if targetName == "" {
		return false
	}
	for _, player := range w.GetAllPlayers() {
		if player.GetName() == targetName && player.GetFighting() == me.GetName() {
			return true
		}
	}
	for _, mob := range w.GetAllMobs() {
		if mob.GetName() == targetName && mob.GetFighting() == me.GetName() {
			return true
		}
	}
	return false
}

func mobRescuerPlayer(w *World, name string, roomVNum int) *Player {
	for _, player := range w.GetPlayersInRoom(roomVNum) {
		if player.GetName() == name {
			return player
		}
	}
	return nil
}

// mobRescueVictim resolves the first word of an NPC's short description with
// get_char_room_vis semantics. SPECIAL(rescuer) passes GET_NAME(ally), and
// one_argument() inside do_rescue therefore makes an article-leading short
// description such as "a guard trainee" fail to resolve.
func mobRescueVictim(w *World, me *MobInstance, shortDesc string) combat.Combatant {
	fields := strings.Fields(shortDesc)
	if len(fields) == 0 {
		return nil
	}
	arg := fields[0]
	for _, player := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if canSee(me, player) && isnameWithAbbrevs(arg, charKeywords(player)) {
			return player
		}
	}
	mobs := w.GetMobsInRoom(me.GetRoomVNum())
	sort.Slice(mobs, func(i, j int) bool { return mobs[i].GetID() < mobs[j].GetID() })
	for _, mob := range mobs {
		if canSee(me, mob) && isnameWithAbbrevs(arg, charKeywords(mob)) {
			return mob
		}
	}
	return nil
}

// mobDoRescue is the subcmd=1 do_rescue() path used by SPECIAL(rescuer).
// It deliberately does not reuse DoRescue: that API models a player command,
// while C invokes the command handler with an NPC ch and then immediately
// calls hit(vict, tmp_ch) after interposing the combat state.
func mobDoRescue(w *World, me, ally *MobInstance) {
	victim := mobRescueVictim(w, me, ally.GetName())
	if victim == nil || victim.GetName() == me.GetName() || me.GetFighting() == victim.GetName() {
		return
	}

	var tmp combat.Combatant
	for _, player := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if player.GetFighting() == victim.GetName() {
			tmp = player
			break
		}
	}
	if tmp == nil {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob.GetFighting() == victim.GetName() {
				tmp = mob
				break
			}
		}
	}
	if tmp == nil {
		return
	}

	// C always uses prob=100 for this special, but still consumes the
	// number(1, 101) draw, where 101 is the complete-failure edge.
	// #nosec G404 — game RNG, not cryptographic
	if rescuerNumber(1, 101) > 100 {
		return
	}

	// The only player-visible act() in the successful native branch is
	// TO_NOTVICT. The TO_CHAR/TO_VICT recipients are both NPCs here.
	Act(w, false, me, victim, nil, nil, "$n heroically rescues $N!", "", ToNotVict)

	// stop_fighting() mutates the three characters even if a combat pair was
	// not registered (the C list is pointer-based, while Go's engine is pair-
	// based). Stop the engine's pairs first, then clear each pointer directly.
	if stopper, ok := w.combatEngine.(interface{ StopCombat(string) }); ok {
		stopper.StopCombat(victim.GetName())
		stopper.StopCombat(tmp.GetName())
		stopper.StopCombat(me.GetName())
	}
	victim.StopFighting()
	tmp.StopFighting()
	me.StopFighting()

	me.SetFighting(tmp.GetName())
	tmp.SetFighting(me.GetName())
	// hit(vict, tmp_ch) calls damage(), which starts the NPC victim's side
	// because stop_fighting(vict) just cleared it; the player already faces me.
	victim.SetFighting(tmp.GetName())

	if w.combatEngine != nil {
		if err := w.combatEngine.StartCombat(me, tmp); err != nil {
			slog.Warn("rescuer combat entry failed", "mob", me.GetName(), "target", tmp.GetName(), "error", err)
		}
	}
	if allyVictim, ok := victim.(*MobInstance); ok {
		if err := w.mobHit(allyVictim, tmp); err != nil {
			slog.Warn("rescuer ally hit failed", "mob", allyVictim.GetName(), "target", tmp.GetName(), "error", err)
		}
	}
	if waiter, ok := victim.(interface{ SetWaitState(int) }); ok {
		waiter.SetWaitState(2 * engine.PULSE_VIOLENCE)
	}
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
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s throws a potion at the ground, and a large globe of thick black mushroom cloud creeps up toward the heavens!", mobName(me)))
	ch.SetHP(ch.GetMaxHP())
	ch.SetMove(ch.GetMaxMove())
	return true
}

// ================================================================
// remorter — Remort info NPC, random tips on pulse, can buy remort
// ================================================================
func specRemorter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// C's SPECIAL(remorter) is a player-command procedure. Its autonomous
	// invocation exits at IS_NPC(ch), so a nil actor must not draw RNG either.
	if ch == nil || ch.IsNPC() {
		return false
	}

	switch cmd {
	case "buy", "list":
		remorterTell(me, ch, "Type REMORT to remort!")
		return true
	case "remort":
		if ch.GetLevel() < LVL_IMMORT-1 {
			remorterTell(me, ch, fmt.Sprintf("You can't remort until level %d!\r\n", LVL_IMMORT-1))
			return true
		}
		if ch.GetLevel() >= LVL_IMMORT {
			remorterTell(me, ch, "Immortals cannot remort!\r\n")
			return true
		}
		if ch.GetGold() < 60000 {
			remorterTell(me, ch, fmt.Sprintf("It costs %d gold to work my magicks.", 60000))
			return true
		}
		if ch.Equipment != nil && len(ch.Equipment.GetEquippedItems()) > 0 {
			SendToChar(ch, "You must come unto me naked, wearing only thy old body.")
			return true
		}

		remortPlayer(ch)
		remorterTell(me, ch, "Enjoy your new life...")
		SendToChar(ch, "The remorter moves his hands over your eyes, closing them with a touch.\r\nColors spiral in your sight... You open your eyes, feeling refreshed.")
		return true
	default:
		return false
	}
}

// remorterTell is the C do_tell(mobile, ...) path used by SPECIAL(remorter).
// Keeping the mob as the act() actor preserves the player-facing name and
// punctuation of a native NPC tell while sending only to the command actor.
func remorterTell(me *MobInstance, ch *Player, msg string) {
	Act(nil, false, me, ch, nil, nil, "$n tells you, '$T'", msg, ToVict)
}

// remortPlayer is the successful body of SPECIAL(remorter), following
// src/spec_procs2.c:728-840.  Keep the state transition here, after all entry
// gates, so failed remort commands cannot consume RNG or mutate the player.
func remortPlayer(ch *Player) {
	oldClass := ch.GetClass()
	newClass := remorterClass(oldClass, ch.GetRace())

	ch.SetPlrFlag(PlrIt, false)
	ch.SetPlrFlag(PlrVampire, false)
	ch.SetPlrFlag(PlrWerewolf, false)
	ch.SetAffect(affVampire, false)
	ch.SetAffect(affWerewolf, false)

	// The live Go path stores spell effects in ActiveAffects. MasterAffects is
	// the compatibility path for older affect callers; unwind its direct
	// modifiers before dropping the records, matching C affect_remove().
	clearRemortAffects(ch)

	ch.SetGold(ch.GetGold() - 60000)
	ch.SetClass(newClass)
	ch.SetLevel(1)
	ch.SetExp(1)

	ch.SetMaxHP(number(30, 40))
	ch.SetMaxMana(100 + number(20, 30))
	ch.SetHP(ch.GetMaxHP())
	ch.SetMana(ch.GetMaxMana())
	ch.SetMove(ch.GetMaxMove())

	// C removes tattoo modifiers before its two remort stat adjustments.
	TattooAf(ch, false)
	stat := firstRemortAdjust(ch)
	secondRemortAdjust(ch, stat)

	ch.SetPlrFlag(PlrRemort, true)
	if ch.SkillManager != nil {
		for _, skill := range ch.SkillManager.GetAllSkills() {
			ch.SkillManager.ForgetSkill(skill.Name)
		}
	}
	setRemortSkills(ch, newClass)
	if ch.GetRace() == RaceKender {
		ch.SetSkill(SkillSteal, 45)
	}
	if ch.GetRace() == RaceMinotaur {
		ch.SetSkill(SkillHeadbutt, 45)
	}
	ch.SetPractices(10)

	TattooAf(ch, true)
	// Active affects are represented by getters in the current path, while
	// legacy MasterAffects were removed above; this is the resulting
	// affect_total state for a naked remort.
	health, mana, move := ch.GetHP(), ch.GetMana(), ch.GetMove()
	ch.AdvanceLevel()
	// C advance_level() raises maxima but does not heal the current pools. The
	// remorter set them immediately before advance_level(), so restore those
	// pre-level-up values after the shared Go helper returns.
	ch.SetHP(health)
	ch.SetMana(mana)
	ch.SetMove(move)
}

func clearRemortAffects(ch *Player) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for _, affect := range ch.MasterAffects {
		if affect == nil {
			continue
		}
		switch affect.Location {
		case engine.ApplyStr:
			ch.Stats.Str -= affect.Modifier
		case engine.ApplyDex:
			ch.Stats.Dex -= affect.Modifier
		case engine.ApplyInt:
			ch.Stats.Int -= affect.Modifier
		case engine.ApplyWis:
			ch.Stats.Wis -= affect.Modifier
		case engine.ApplyCon:
			ch.Stats.Con -= affect.Modifier
		case engine.ApplyCha:
			ch.Stats.Cha -= affect.Modifier
		case engine.ApplyMana:
			ch.MaxMana -= affect.Modifier
		case engine.ApplyHit:
			ch.MaxHealth -= affect.Modifier
		case engine.ApplyMove:
			ch.MaxMove -= affect.Modifier
		case engine.ApplyAC:
			ch.AC += affect.Modifier
		case engine.ApplyHitroll:
			ch.Hitroll -= affect.Modifier
		case engine.ApplyDamroll:
			ch.Damroll -= affect.Modifier
		case engine.ApplySavingPara, engine.ApplySavingRod, engine.ApplySavingPetri, engine.ApplySavingBreath, engine.ApplySavingSpell:
			index := affect.Location - engine.ApplySavingPara
			if index >= 0 && index < len(ch.SavingThrows) {
				ch.SavingThrows[index] -= affect.Modifier
			}
		}
		ch.Affects &^= affect.Bitvector
	}
	ch.ActiveAffects = nil
	ch.MasterAffects = nil
	ch.Affects = 0
}

func remorterClass(class, race int) int {
	switch class {
	case ClassWarrior, ClassPaladin, ClassRanger:
		if race == RaceHuman || race == RaceElf || race == RaceDwarf {
			return ClassPaladin
		}
		return ClassRanger
	case ClassCleric:
		return ClassAvatar
	case ClassThief:
		return ClassAssassin
	case ClassMageUser:
		return ClassMagus
	case ClassPsionic:
		return ClassMystic
	default:
		return class
	}
}

const (
	remortStatDefault = iota
	remortStatCon
	remortStatStr
	remortStatWis
	remortStatInt
	remortStatDex
	remortStatCha
)

func firstRemortAdjust(ch *Player) int {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.Stats.Con < 17 {
		ch.Stats.Con += 2
		return remortStatCon
	}
	if ch.Stats.Str < 17 {
		ch.Stats.Str += 2
		return remortStatStr
	}
	if ch.Stats.Wis < 17 {
		ch.Stats.Wis += 2
		return remortStatWis
	}
	if ch.Stats.Int < 17 {
		ch.Stats.Int += 2
		return remortStatInt
	}
	if ch.Stats.Dex < 17 {
		ch.Stats.Dex += 2
		return remortStatDex
	}
	if ch.Stats.Cha < 17 {
		ch.Stats.Cha += 2
		return remortStatCha
	}
	return remortStatDefault
}

func secondRemortAdjust(ch *Player, adjusted int) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.Stats.Str == 18 && adjusted != remortStatStr && ch.Stats.StrAdd < 100 {
		if ch.Stats.StrAdd != 0 {
			ch.Stats.StrAdd += 20
		} else {
			ch.Stats.StrAdd = 20
		}
	}

	// GET_ORIG_CON is not separately persisted by the Go Player model. On a
	// fresh character it equals real CON; the visible real-stat adjustment is
	// therefore only made by firstRemortAdjust, just as in C.
	if ch.Stats.Con < 18 && adjusted != remortStatCon {
		return
	}
	if ch.Stats.Str < 18 && adjusted != remortStatStr {
		ch.Stats.Str++
		return
	}
	if ch.Stats.Wis < 18 && adjusted != remortStatWis {
		ch.Stats.Wis++
		return
	}
	if ch.Stats.Int < 18 && adjusted != remortStatInt {
		ch.Stats.Int++
		return
	}
	if ch.Stats.Dex < 18 && adjusted != remortStatDex {
		ch.Stats.Dex++
		return
	}
	if ch.Stats.Cha < 18 && adjusted != remortStatCha {
		ch.Stats.Cha++
		return
	}

	// If every stat path is exhausted, C uses the bonus adjustment to turn
	// off thirst first, then hunger. Keep both the named fields and the
	// persisted condition array synchronized.
	if ch.Conditions[CondThirst] != -1 {
		ch.Conditions[CondThirst] = -1
		ch.Thirst = -1
		return
	}
	if ch.Conditions[CondFull] != -1 {
		ch.Conditions[CondFull] = -1
		ch.Hunger = -1
		return
	}
}

func setRemortSkills(ch *Player, class int) {
	setSkill := func(num, level int) {
		name := strings.ToLower(SkillCatalogName(num))
		if name != "" {
			ch.SetSkill(name, level)
		}
	}

	switch class {
	case ClassMageUser:
		setSkill(spells.SpellMagicMissile, 20)
		setSkill(spells.SpellAcidBlast, 20)
	case ClassMagus:
		setSkill(spells.SpellMagicMissile, 40)
		setSkill(spells.SpellAcidBlast, 40)
	case ClassCleric:
		setSkill(spells.SpellCureLight, 20)
		setSkill(spells.SpellArmor, 20)
	case ClassAvatar:
		setSkill(spells.SpellCureLight, 40)
		setSkill(spells.SpellArmor, 40)
	case ClassWarrior:
		ch.SetSkill(SkillKick, 20)
	case ClassPaladin, ClassRanger:
		ch.SetSkill(SkillKick, 40)
	case ClassPsionic, ClassMystic:
		setSkill(spells.SpellMindPoke, 20)
	case ClassThief, ClassAssassin:
		ch.SetSkill(SkillSneak, 20)
		ch.SetSkill(SkillHide, 10)
		ch.SetSkill(SkillSteal, 30)
		ch.SetSkill(SkillBackstab, 20)
		ch.SetSkill(SkillPickLock, 20)
		ch.SetSkill("track", 20)
	}
}

// ================================================================
// assassin — Room-based assassin shop
// ================================================================
// C: src/spec_procs2.c:845-928, assigned by ASSIGNROOM(8114, assassin).
// The C procedure uses the next internal room index as its hidden roster;
// room vnums are contiguous in the authoritative world here, so room 8115
// is the corresponding Go room. It is a room special, not a mob special.
func specAssassin(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me != nil {
		return false
	}

	rosterRoom := ch.GetRoomVNum() + 1
	switch cmd {
	case "list":
		sendToChar(ch, "To hire an assassin: hire <assassin> <victim>")
		sendToChar(ch, "Available assassins are:")
		for _, member := range assassinRoster(w, rosterRoom) {
			sendToChar(ch, fmt.Sprintf("%8d - %s", memberLevel(member)*1000, memberName(member)))
		}
		return true
	case "hire":
		first, second := twoAssassinArguments(arg)
		if first == "" {
			sendToChar(ch, "Hire who?")
			return true
		}

		assassin := findAssassinRosterMember(w, rosterRoom, first)
		if assassin == nil {
			sendToChar(ch, "There is nobody called that!")
			return true
		}
		if player, ok := assassin.(*Player); ok {
			sendToChar(player, "GET THE HELL OUT OF THAT ROOM, NOW !!!")
			slog.Info("player found in assassin store room", "player", player.GetName())
			sendToChar(ch, "You can't hire players.")
			return true
		}
		assassinMob, ok := assassin.(*MobInstance)
		if !ok {
			return true
		}
		if second == "" {
			sendToChar(ch, "Whom do you want to assassinate?")
			return true
		}

		price := memberLevel(assassin) * 1000
		if ch.GetGold() < price {
			sendToChar(ch, "You don't have enough gold!")
			return true
		}

		victim := findVisibleAssassinVictim(w, assassinMob, second)
		if victim == nil {
			sendToChar(ch, "Our underground doesn't know the whereabouts of the victim!")
			return true
		}
		if victim.GetLevel() < 5 {
			sendToChar(ch, "We cannot lower ourselves to such easy prey.")
			return true
		}

		ch.SetGold(ch.GetGold() - price)
		hired, err := w.spawnMobQuiet(assassinMob.GetVNum(), ch.GetRoom())
		if err != nil {
			slog.Error("assassin hire failed to spawn mob", "vnum", assassinMob.GetVNum(), "error", err)
			return true
		}
		hired.SetMobFlag(MobFlagHunter)
		hired.SetHunting(victim.GetName())
		sendToChar(ch, "We cannot contact you if the job succeeds or not...security, you know.")
		Act(w, false, ch, hired, nil, nil, "$n hires $N for a job.", "", ToRoom)
		return true
	default:
		return false
	}
}

func twoAssassinArguments(arg string) (string, string) {
	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return "", ""
	}
	if len(fields) == 1 {
		return fields[0], ""
	}
	return fields[0], fields[1]
}

func assassinRoster(w *World, roomVNum int) []Actor {
	members := make([]Actor, 0)
	for _, player := range w.GetPlayersInRoom(roomVNum) {
		members = append(members, player)
	}
	for _, mob := range w.GetMobsInRoom(roomVNum) {
		members = append(members, mob)
	}
	return members
}

func findAssassinRosterMember(w *World, roomVNum int, name string) Actor {
	ordinal := GetNumber(&name)
	if ordinal == 0 {
		return nil
	}
	matches := 0
	for _, member := range assassinRoster(w, roomVNum) {
		if !isnameWithAbbrevs(name, assassinMemberKeywords(member)) {
			continue
		}
		matches++
		if matches == ordinal {
			return member
		}
	}
	return nil
}

func assassinMemberKeywords(member Actor) string {
	switch value := member.(type) {
	case *Player:
		return value.GetName()
	case *MobInstance:
		return charKeywords(value)
	default:
		return ""
	}
}

func memberLevel(member Actor) int {
	switch value := member.(type) {
	case *Player:
		return value.GetLevel()
	case *MobInstance:
		return value.GetLevel()
	default:
		return 0
	}
}

func memberName(member Actor) string {
	if member == nil {
		return ""
	}
	return member.GetName()
}

func findVisibleAssassinVictim(w *World, assassin *MobInstance, name string) *Player {
	for _, player := range w.GetAllPlayers() {
		if strings.EqualFold(player.GetName(), name) && canSee(assassin, player) {
			return player
		}
	}
	return nil
}

// ================================================================
// tattoo1 — Buy one of the tattooist's five tattoos.
// C: src/spec_procs2.c:927-1008, src/constants.c:1416-1433
// ================================================================

type tattooOffer struct {
	number      int
	price       int
	name        string
	description string
}

var tattoo1Offers = [...]tattooOffer{
	{number: TattooDragon, price: 30666, name: "of a green dragon", description: "grow stronger and hit harder"},
	{number: TattooTribal, price: 3000, name: "in a tribal design", description: "increase your dexterity"},
	{number: TattooEagle, price: 10000, name: "of a screaming eagle", description: "move like the wind"},
	{number: TattooFox, price: 3000, name: "of a fox", description: "gain the intelligence of the fox"},
	{number: TattooOwl, price: 3000, name: "of an owl", description: "gain the wisdom of the owl"},
}

func specTattoo1(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || ch.IsNPC() {
		return false
	}

	switch strings.ToLower(cmd) {
	case "list":
		ch.SendMessage("To buy a tattoo: BUY <number of tattoo>.\r\n")
		ch.SendMessage("Available tattoos are:\r\n")
		for i, offer := range tattoo1Offers {
			ch.SendMessage(fmt.Sprintf("[%d] - (%d) tattoo %s : %s\r\n", i, offer.price, offer.name, offer.description))
		}
		return true
	case "buy":
		if me == nil {
			return false
		}
		if ch.Tattoo != TattooNone {
			tellFromMob(me, ch, "Your magickal center is already tattooed. Get a new arm or get rid of that tattoo then come back.")
			return true
		}

		arg = skipSpaces(arg)
		if arg == "" {
			sendToChar(ch, "Buy what number?")
			return true
		}
		if arg[0] < '0' || arg[0] > '9' {
			sendToChar(ch, "Buy by number!")
			return true
		}

		choice := atoi(arg)
		if choice >= len(tattoo1Offers) {
			sendToChar(ch, "Buy by number!")
			return true
		}

		offer := tattoo1Offers[choice]
		if ch.GetGold() < offer.price {
			tellFromMob(me, ch, "You look a little short on the price there, kid.")
			return true
		}

		giveTattoo(w, ch, me, offer)
		return true
	}
	return false
}

// giveTattoo is the give_tat() helper shared by the C tattoo procedures.
// The position assignment followed by update_pos is intentional: a healthy
// player ends standing even though give_tat briefly assigns POS_STUNNED.
func giveTattoo(w *World, ch *Player, me *MobInstance, offer tattooOffer) {
	ch.SetGold(ch.GetGold() - offer.price)
	ch.Tattoo = offer.number
	Act(w, true, me, ch, nil, nil, "$n starts to work on $N's tattoo...", "", ToNotVict)
	Act(w, true, me, ch, nil, nil, "A ghastly scream is ripped from $N's lips just before $E blacks out.", "", ToNotVict)
	Act(nil, true, me, ch, nil, nil, "$n starts to work on your tattoo...", "", ToVict)
	ch.SendMessage("The pain is incredible; it seems to eat into your soul.\r\nA scream is ripped from your lips...\r\n")
	w.doGenComm(ch, me, "shout", "Arrrrrrrrrgggggggghhhh!")
	ch.SendMessage("You black out.\r\n")
	ch.SetPosition(combat.PosStunned)
	updatePosFromHP(ch, ch.GetHP())
	TattooAf(ch, true)
}

// ================================================================
// tattoo2 — Buy one of the monk tattooist's four tattoos.
// C: src/spec_procs2.c:1010-1072, src/constants.c:1416-1433
// ================================================================

var tattoo2Offers = [...]tattooOffer{
	{number: TattooTiger, price: 14000, name: "of a leaping tiger", description: "the nimbleness and stamina of the tiger"},
	{number: TattooHeart, price: 17000, name: "of a heart", description: "live longer through trust in your heart"},
	{number: TattooStar, price: 17000, name: "of a star", description: "gain the magic of the stars"},
	{number: TattooJyhadi, price: 19000, name: "of the symbol of the Jyhad", description: "the power of fighting a holy war"},
}

func specTattoo2(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil {
		return false
	}

	switch strings.ToLower(cmd) {
	case "list":
		ch.SendMessage("To buy a tattoo: BUY <number of tattoo>.\r\n")
		ch.SendMessage("Available tattoos are:\r\n")
		for i, offer := range tattoo2Offers {
			ch.SendMessage(fmt.Sprintf("[%d] - (%d) tattoo %s : %s\r\n", i, offer.price, offer.name, offer.description))
		}
		return true
	case "buy":
		arg = skipSpaces(arg)
		if ch.IsNPC() {
			return false
		}
		if arg == "" {
			sendToChar(ch, "Buy what number?")
			return true
		}
		if arg[0] < '0' || arg[0] > '9' {
			sendToChar(ch, "Buy by number!")
			return true
		}

		choice := atoi(arg)
		if choice >= len(tattoo2Offers) {
			sendToChar(ch, "Buy by number!")
			return true
		}
		if me == nil {
			return false
		}
		if ch.Tattoo != TattooNone {
			tellFromMob(me, ch, "Your magickal center is already tattooed. Your tattoo... is enough magick for such as yourself.")
			return true
		}

		offer := tattoo2Offers[choice]
		if ch.GetGold() < offer.price {
			tellFromMob(me, ch, "Without more coins, I can give no wisdom.")
			return true
		}

		giveTattoo(w, ch, me, offer)
		return true
	}
	return false
}

// ================================================================
// tattoo3 — Buy one of the tattoo artist's four tattoos.
// C: src/spec_procs2.c:1075-1137, src/constants.c:1416-1433
// ================================================================

var tattoo3Offers = [...]tattooOffer{
	{number: TattooEye, price: 18000, name: "of an open eye", description: "see that which is normally unseen"},
	{number: TattooSwords, price: 20000, name: "of crossed swords", description: "miss less and hit harder"},
	{number: TattooShip, price: 11000, name: "of a ship", description: "gain the ability of movement over water"},
	{number: TattooMom, price: 15000, name: "of the word 'MOM'", description: "the wisdom of your elders"},
}

func specTattoo3(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil {
		return false
	}

	switch strings.ToLower(cmd) {
	case "list":
		ch.SendMessage("To buy a tattoo: BUY <number of tattoo>.\r\n")
		ch.SendMessage("Available tattoos are:\r\n")
		for i, offer := range tattoo3Offers {
			ch.SendMessage(fmt.Sprintf("[%d] - (%d) tattoo %s : %s\r\n", i, offer.price, offer.name, offer.description))
		}
		return true
	case "buy":
		arg = skipSpaces(arg)
		if ch.IsNPC() {
			return false
		}
		if arg == "" {
			sendToChar(ch, "Buy what number?")
			return true
		}
		if arg[0] < '0' || arg[0] > '9' {
			sendToChar(ch, "Buy by number!")
			return true
		}

		choice := atoi(arg)
		if choice >= len(tattoo3Offers) {
			sendToChar(ch, "Buy by number!")
			return true
		}
		if me == nil {
			return false
		}
		if ch.Tattoo != TattooNone {
			tellFromMob(me, ch, "Your mathickal thenter is awready tattooed. Your tattoo... ith enough mathick for such as yoursewf.")
			return true
		}

		offer := tattoo3Offers[choice]
		if ch.GetGold() < offer.price {
			tellFromMob(me, ch, "You don't have enough cash, hot stuff.")
			return true
		}

		giveTattoo(w, ch, me, offer)
		return true
	}
	return false
}

// identifierValCost reproduces val_cost() from src/spec_procs2.c:1179-1191.
// The magic surcharge is part of the C helper even though the first proof
// vehicle uses an ordinary loaf of bread.
func identifierValCost(obj *ObjectInstance) int {
	if obj == nil {
		return 1
	}
	cost := obj.GetCost()
	price := cost / 10
	if cost >= 5000 {
		price = int(float64(cost) * 0.14)
	}
	if obj.HasExtraFlag(0, itemExtraMagic) {
		price += cost / 20
	}
	if price < 1 {
		return 1
	}
	return price
}

// specIdentifier is the identifier mob special.
// C: src/spec_procs2.c:1193-1280. The unusual give syntax is intentional:
// the procedure only consumes "give <object> <identifier-keyword>" and lets
// normal do_give handle every other form.
func specIdentifier(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil {
		return false
	}

	switch strings.ToLower(cmd) {
	case "list":
		tellFromMob(me, ch, "Just read the sign!")
		return true
	case "value":
		arg = skipSpaces(arg)
		if arg == "" {
			tellFromMob(me, ch, "Value what?")
			return true
		}
		obj, ok := w.ResolveObjectInInventory(ch, arg)
		if !ok {
			tellFromMob(me, ch, "You don't seem to have that.")
			return true
		}
		tellFromMob(me, ch, fmt.Sprintf("I'll identify that fully for about %d coins.", identifierValCost(obj)))
		return true
	case "give":
		if skipSpaces(arg) == "" {
			return false
		}
		objName, rest := oneArgument(arg)
		targetName, _ := oneArgument(rest)
		if objName == "" || targetName == "" || !isnameWithAbbrevs(targetName, charKeywords(me)) {
			return false
		}
		obj, ok := w.ResolveObjectInInventory(ch, objName)
		if !ok {
			return false
		}

		price := identifierValCost(obj)
		if ch.GetGold() < price {
			tellFromMob(me, ch, fmt.Sprintf("That's a fine item, but I'll need %d coins from you to id it.. and you're a little short..", price))
			tellFromMob(me, ch, "Keep it until you get the gold.")
			return true
		}

		ch.SetGold(ch.GetGold() - price)
		Act(nil, false, ch, me, obj, nil, "You give $p to $N.", "", ToChar)
		Act(nil, false, ch, me, obj, nil, "$n gives you $p.", "", ToVict)
		Act(w, true, ch, me, obj, nil, "$n gives $p to $N.", "", ToNotVict)
		Act(w, true, me, nil, nil, nil, "$n studies it carefully, comparing it to ancient texts,\r\nweighing it on scales, and chanting a number of odd spells over its surface.", "", ToRoom)
		Act(nil, false, me, ch, obj, nil, "Finally looking up, you give $p back to $N.", "", ToChar)
		Act(nil, false, me, ch, obj, nil, "Finally looking up, $n gives you back $p.", "", ToVict)
		Act(w, true, me, ch, obj, nil, "Finally looking up, $n gives back $p to $N.", "", ToNotVict)
		Act(nil, false, ch, me, obj, nil, "$N touches your forehead, and knowledge fills your mind.", "", ToChar)
		Act(nil, false, ch, me, obj, nil, "You touch $n gently on the forehead.", "", ToVict)
		Act(w, true, ch, me, obj, nil, "$N touches $n gently on the forehead.", "", ToNotVict)
		ch.SendMessage("\r\n")
		spells.CallMagic(ch, nil, obj, spells.SpellIdentify, ch.GetLevel(), spells.CastSpell, w)
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
					ch.AddExp(ch.GetLevel() * 200)
					toRemove = append(toRemove, item)
				}
			}
			for _, item := range toRemove {
				ch.Inventory.removeItem(item)
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
// tattoo4 — Buy one of the sleazy tattoo artist's three tattoos.
// C: src/spec_procs2.c:1282-1340, src/constants.c:174-193,1416-1433
// ================================================================

var tattoo4Offers = [...]tattooOffer{
	{number: TattooWorm, price: 25000, name: "of an ice worm", description: "hit with the fierceness of the remorhaz"},
	{number: TattooDragon, price: 30666, name: "of a green dragon", description: "grow stronger and hit harder"},
	{number: TattooSkull, price: 18000, name: "of a flaming skull", description: "summon a flaming skull to aid against thy enemies."},
}

func specTattoo4(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || ch.IsNPC() || me == nil {
		return false
	}

	switch strings.ToLower(cmd) {
	case "list":
		ch.SendMessage("To buy a tattoo: BUY <number of tattoo>.\r\n")
		ch.SendMessage("Available tattoos are:\r\n")
		for i, offer := range tattoo4Offers {
			ch.SendMessage(fmt.Sprintf("[%d] - (%d) tattoo %s : %s\r\n", i, offer.price, offer.name, offer.description))
		}
		return true
	case "buy":
		arg = skipSpaces(arg)
		if arg == "" {
			sendToChar(ch, "Buy what number?")
			return true
		}
		if arg[0] < '0' || arg[0] > '9' {
			sendToChar(ch, "Buy by number!")
			return true
		}
		choice := atoi(arg)
		if choice >= len(tattoo4Offers) {
			sendToChar(ch, "Buy by number!")
			return true
		}
		if ch.Tattoo != TattooNone {
			tellFromMob(me, ch, "Get outta here, punk, you already have one. ")
			return true
		}

		offer := tattoo4Offers[choice]
		if ch.GetGold() < offer.price {
			tellFromMob(me, ch, "You don't have enough gold, get outta here!")
			return true
		}
		giveTattoo(w, ch, me, offer)
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
			if err := me.Attack(pl, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
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
				if err := ch.Inventory.addItem(obj); err != nil {
					slog.Warn("spec proc item grant failed", "vnum", 7107, "player", ch.Name, "error", err)
				}
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
		if number(0, 5) != 0 {
			continue
		}
		if number(0, 31) == 0 {
			sendToChar(ch, fmt.Sprintf("%s says 'I don't like you, and you'd better leave before I make you!'\r\n", mobName(me)))
			if err := me.Attack(pl, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}
	return false
}

// ================================================================
// take_to_jail — city guard special; hit() owns the jail redirect
// ================================================================
func specTakeToJail(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || me == nil || cmd != "" {
		return false
	}
	// mobile_activity() passes the mob as both ch and me, while the Go special
	// adapter uses ch=nil for that same autonomous path. C's AWAKE(ch) gate is
	// therefore the mob's position here.
	if me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if me.IsFighting() {
		// C delegates an already-fighting guard to fighter(). The autonomous
		// mobile_activity() filter normally makes this unreachable, but retain
		// the real special call path for direct callers and combat handoffs.
		return specFighter(w, nil, me, "", "")
	}

	// C scans world[].people in order and gives an outlaw the first matching
	// intervention. hit() then recognizes this special and performs the jail
	// redirect before ordinary melee damage (src/fight.c:1370-1400).
	for _, candidate := range cityguardRoomCombatants(w, me.GetRoomVNum()) {
		pl, ok := candidate.(*Player)
		if !ok || !canSee(me, pl) || pl.GetFlags()&(1<<uint(PlrOutlaw)) == 0 {
			continue
		}
		Act(w, false, me, nil, nil, nil,
			"$n says 'We don't like OUTLAWS like you in this city!'", "", ToRoom)
		if err := w.mobHit(me, pl); err != nil {
			slog.Warn("take_to_jail outlaw attack failed", "guard", me.GetName(), "target", pl.GetName(), "error", err)
		}
		return specFighter(w, nil, me, "", "")
	}

	// The C body calls breed_killer() while walking this same room list. That
	// shared procedure is separately owned by the queued breed_killer slice;
	// do not duplicate or invent its currently-unproven nightbreed branches
	// here (R5b/R5c). With no eligible nightbreed, it is a no-op and the scan
	// continues to the shared protection selection below.
	var evil cityguardAlignedCombatant
	var evilTarget cityguardAlignedCombatant
	maxEvil := 1000
	for _, candidate := range cityguardRoomCombatants(w, me.GetRoomVNum()) {
		tch, ok := candidate.(cityguardAlignedCombatant)
		if !ok || !canSee(me, tch) || tch.GetFighting() == "" {
			continue
		}
		target := cityguardCombatantByName(w, me.GetRoomVNum(), tch.GetFighting())
		targetAligned, ok := target.(cityguardAlignedCombatant)
		if !ok || targetAligned == nil || tch.GetAlignment() >= maxEvil ||
			(!tch.IsNPC() && !target.IsNPC()) {
			continue
		}
		maxEvil = tch.GetAlignment()
		evil = tch
		evilTarget = targetAligned
	}
	if evil != nil && evilTarget.GetAlignment() >= 0 {
		Act(w, false, me, evil, nil, nil,
			"$n says, 'You just pissed me off, $N!'", "", ToRoom)
		if err := w.mobHit(me, evil); err != nil {
			slog.Warn("take_to_jail protection attack failed", "guard", me.GetName(), "target", evil.GetName(), "error", err)
		}
		return specFighter(w, nil, me, "", "")
	}

	return false
}

// ================================================================
// jail — registered room special whose commandless body is unreachable
// ================================================================
func specJail(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// C's room special receives a nonzero command index from both interpreter.c
	// and the movement path, so its first `if (cmd || mini_mud)` gate always
	// returns FALSE on the real player-facing call path. In particular, the C
	// body does not implement a `say release` command; retaining that invented
	// Go branch would violate R1/R2/R5e. The commandless timer body is not
	// reachable from either C dispatcher and remains an explicit fallthrough.
	_ = w
	_ = ch
	_ = me
	_ = arg
	_ = cmd
	return false
}

// ================================================================
// medusa — look at the medusa and risk petrification
// ================================================================
func specMedusa(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// SPECIAL(medusa), src/spec_procs2.c:1552-1590. The commandless arm is
	// the same re-entrant magic_user() call used by other spellcaster mobs.
	if cmd == "" {
		if me != nil && me.IsFighting() {
			return specMagicUser(w, nil, me, "", "")
		}
		return false
	}
	if ch == nil || me == nil || (cmd != "look" && cmd != "examine") {
		return false
	}

	// C uses isname(argument, mobile->player.name), so match the complete
	// character keyword list rather than searching the display description.
	if !isnameWithAbbrevs(strings.TrimSpace(arg), charKeywords(me)) {
		return false
	}

	// mag_savingthrow() returns TRUE when the actor saves. A save falls through
	// to ordinary look/examine; only the actor who looked can be petrified.
	if spells.CheckSavingThrow(ch, spells.SavePetrify) {
		return false
	}

	Act(w, false, ch, ch, nil, nil,
		"With a sound like that of a crashing wave, $N slowly turns to stone!", "", ToNotVict)
	Act(nil, false, ch, nil, nil, nil,
		"With growing horror and increasing agony, your body slowly turns to stone!", "", ToChar)

	// SPECIAL(medusa) explicitly accounts the death and applies a level-cubed
	// loss before raw_kill(); this is not die_with_killer()'s combat penalty.
	ch.Deaths++
	level := ch.GetLevel()
	w.GainExp(ch, -(level * level * level))
	w.Instakill(ch, nil, spells.SpellPetrify)
	return true
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
	if number(0, 101) > 20 {
		return false
	}
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	count := 0
	items := ch.GetInventory()
	for i := len(items) - 1; i >= 0; i-- {
		obj := items[i]
		// Steal items with zero cost (junk/free items)
		if obj.GetCost() == 0 {
			ch.Inventory.removeItem(obj)
			count++
		}
	}
	sendToChar(ch, fmt.Sprintf("The eq thief steals %d items!\r\n", count))
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gets stripped of equipment by the eq thief!", ch.GetName()))
	ch.SetMove(0)
	ch.SetHP(1)
	return true
}

// ================================================================
// portal_room — Random teleport on move command
// ================================================================
func specPortalRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "north" || cmd == "south" || cmd == "east" || cmd == "west" || cmd == "up" || cmd == "down" {
		if !ch.IsNPC() && number(0, 2) != 0 {
			sendToChar(ch, "A shimmering portal appears and sucks you in!\r\n")
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s stumbles through a shimmering portal!", ch.GetName()))
			// Teleport to a random room
			rooms := w.Rooms()
			if len(rooms) > 0 {
				// #nosec G404 — game RNG, not cryptographic
				// #nosec G404
				target := rooms[dprng.Number(0, len(rooms)-1)]
				ch.SetRoom(target.VNum)
			}
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
	// ch is nil during autonomous AI ticks; the mob's own state is checked
	// via me (mirrors the C source, which is always called with ch==the mob).
	if cmd != "" {
		return false
	}
	if me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	if me.GetFighting() != "" {
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
		if err := me.Attack(victim, w); err != nil {
			slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
		}
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
	if number(0, 5) != 0 {
		return false
	}
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tears into its victim with renewed fury!", mobName(me)))
	for _, vict := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !vict.IsNPC() && vict.GetName() != ch.GetName() {
			if randRange(1, vict.GetLevel()) <= me.GetLevel() {
				if err := me.Attack(vict, w); err != nil {
					slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
				}
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
	if strings.Contains(a, "dripping") && number(0, 4) == 0 {
		sendToChar(ch, "A bat swoops down and attacks you!\r\n")
		if err := me.Attack(ch, w); err != nil {
			slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
		}
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
		Act(w, false, me, ch, nil, nil, "$n humiliates $N, and blocks $S way.", "", ToNotVict)
		Act(w, false, me, ch, nil, nil, "$n humiliates you and blocks your way.", "", ToVict)
		return true
	}
	return false
}

// ================================================================
// specKeySeller — Sells the configured house key to its owner.
// C: spec_procs2.c:key_seller (1870-1932)
// ================================================================
func specKeySeller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me == nil || ch.IsNPC() {
		return false
	}
	if cmd != "buy" && cmd != "list" {
		return false
	}

	i := findHouse(w.HouseControl, me.GetRoomVNum())
	if i < 0 {
		return false
	}
	house := w.HouseControl[i]
	if int64(ch.GetID()) != house.Owner {
		tellFromMob(me, ch, "Sorry, I only serve the house owner.")
		return true
	}

	a := strings.TrimSpace(arg)
	if cmd == "buy" {
		if a != "key" {
			tellFromMob(me, ch, "I only sell keys, currently.")
			return true
		}
		if ch.GetGold() < 10000 {
			tellFromMob(me, ch, "You can't afford a key.")
			return true
		}
		obj, err := w.SpawnObject(house.Key, -1)
		if err != nil {
			return false
		}
		if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
			slog.Warn("key seller item grant failed", "vnum", house.Key, "player", ch.Name, "error", err)
			w.ExtractObject(obj, -1)
			return false
		}
		Act(w, true, me, ch, obj, nil, "$n produces $p from the folds of $s long robe and hands it to $N.", "", ToNotVict)
		Act(nil, false, me, ch, obj, nil, "$n produces $p from the folds of $s long robe and hands it to you.", "", ToVict)
		ch.SetGold(ch.GetGold() - 10000)
		return true
	}

	tellFromMob(me, ch, "You can buy a key for 10,000 gold coins.")
	return true
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
			if err := me.Attack(ch, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.IsFighting() || mob.GetFightingTarget() == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.GetFightingTarget() && !pl.IsNPC() {
					if err := me.Attack(pl, w); err != nil {
						slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
					}
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
		if !pl.IsNPC() && pl.GetLevel() < 50 && number(0, 5) == 0 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s stares at %s with hollow, empty eyes!", mobName(me), pl.GetName()))
			pl.Stats.Int = max(3, pl.Stats.Int-1)
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
		if !pl.IsNPC() && pl.GetFighting() == "" && number(0, 3) == 0 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("From the shadows, %s backstabs %s!", mobName(me), pl.GetName()))
			if err := me.Attack(pl, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}
	return false
}

// ================================================================
// specTeleporter — Picks random room and teleports players there
// ================================================================
func specTeleporter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch == nil || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch.GetFighting() != "" {
		return false
	}
	if !ch.IsNPC() && number(0, 4) == 0 {
		// Pick random room, ensure not private/godroom/death/nomob
		rooms := w.Rooms()
		var toRoom int
		for i := 0; i < 200; i++ {
			// #nosec G404 — game RNG, not cryptographic
			// #nosec G404
			candidate := rooms[dprng.Number(0, len(rooms)-1)]
			if w.roomHasFlag(candidate.VNum, "private") || w.roomHasFlag(candidate.VNum, "godroom") ||
				w.roomHasFlag(candidate.VNum, "death") || w.roomHasFlag(candidate.VNum, "nomob") {
				continue
			}
			toRoom = candidate.VNum
			break
		}
		if toRoom != 0 {
			ch.SetRoom(toRoom)
		}
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
		Act(w, false, me, ch, nil, nil, "$n humiliates $N, and blocks $S way.", "", ToNotVict)
		Act(w, false, me, ch, nil, nil, "$n humiliates you and blocks your way.", "", ToVict)
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
	// C: SPECIAL(never_die) is always called with ch==the mob itself here
	// (mobile_activity.c calls func(ch, ch, 0, "")); ch is nil in the Go
	// autonomous path, so the mob's own state must be read via me.
	if cmd != "" {
		return false
	}
	if me.GetHP() < me.GetMaxHP() {
		me.SetHealth(me.GetMaxHP())
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
		Act(w, false, me, ch, nil, nil, "$n blocks $N's way.", "", ToNotVict)
		Act(w, false, me, ch, nil, nil, "$n blocks your way.", "", ToVict)
		Act(w, false, me, nil, nil, nil, "$n says 'Thou shalt not pass.'", "", ToRoom)
		return true
	}
	return false
}

// ================================================================
// specChosenGuard — Guards the chosen, attacks players who fight near it
// ================================================================
func specChosenGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || ch.GetPosition() <= combat.PosSleeping || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if cmd != "" {
		return false
	}
	for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if !pl.IsNPC() && pl.GetFighting() != "" {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says 'None shall harm the chosen!'", mobName(me)))
			if err := me.Attack(pl, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
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
			if err := me.Attack(ch, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.IsFighting() || mob.GetFightingTarget() == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.GetFightingTarget() && !pl.IsNPC() {
					if err := me.Attack(pl, w); err != nil {
						slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
					}
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
			if err := me.Attack(ch, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.IsFighting() || mob.GetFightingTarget() == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.GetFightingTarget() && !pl.IsNPC() {
					if err := me.Attack(pl, w); err != nil {
						slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
					}
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
			if err := me.Attack(ch, w); err != nil {
				slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
			}
			return true
		}
	}

	if cmd == "" && me.GetFighting() == "" {
		for _, mob := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if mob == me || !mob.IsFighting() || mob.GetFightingTarget() == "" {
				continue
			}
			for _, pl := range w.GetPlayersInRoom(me.GetRoomVNum()) {
				if pl.GetName() == mob.GetFightingTarget() && !pl.IsNPC() {
					if err := me.Attack(pl, w); err != nil {
						slog.Warn("Attack failed in spec proc", "mob", me.GetName(), "error", err)
					}
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
