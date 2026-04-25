package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)


// AFF bit constants — from structs.h (AFF_BLIND=0, AFF_SANCTUARY=7, AFF_CURSE=9, AFF_POISON=11)
const (
	affCurse     = 9  // AFF_CURSE
	affPoison    = 11 // AFF_POISON
	affSanctuary = 7  // AFF_SANCTUARY
)

// findMobInRoom finds a MobInstance by name in a room's mob list.
func findMobInRoom(w *World, roomVNum int, name string) *MobInstance {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if m.GetName() == name {
			return m
		}
	}
	return nil
}

// mobHasAffect checks if a MobInstance has a given affect flag string in its prototype.
func mobHasAffect(me *MobInstance, affect string) bool {
	for _, f := range me.Prototype.AffectFlags {
		if strings.EqualFold(f, affect) {
			return true
		}
	}
	return false
}

// findTargetInRoom finds a mob or player by name in a room. Returns the target
// as an interface{} suitable for passing to spells.Cast (which accepts interface{}).
func findTargetInRoom(w *World, roomVNum int, name string) interface{} {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if m.GetName() == name {
			return m
		}
	}
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}

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

// specCleric — cleric mob: heals self, casts offensive/defensive spells while fighting.
// Ported from SPECIAL(cleric) in spec_procs.c (line 1425).
// Uses `me` (MobInstance) for all mob state — `ch` is nil during pulse calls.
func specCleric(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// In C, IS_NPC(ch) and AWAKE(ch) guard on the mob. In Go, `me` is always the mob.
	// If ch != nil and ch is not an NPC, a player triggered this via command — not our spec.
	if ch != nil {
		return false
	}
	if cmd != "" || me.GetHP() < 0 {
		return false
	}

	// Stand up if between stunned and standing (C: AWAKE check + do_stand)
	if me.GetPosition() != combat.PosFighting {
		if me.GetPosition() > combat.PosStunned && me.GetPosition() < combat.PosStanding {
			me.SetStatus("standing")
		}
	}

	// Do nothing in peaceful rooms
	if w.roomHasFlag(me.GetRoomVNum(), "peaceful") {
		return false
	}

	// If not fighting and below max HP-10, heal self
	if me.GetFighting() == "" && me.GetHP() < me.GetMaxHP()-10 {
		switch {
		case me.GetLevel() >= 20:
			spells.Cast(me, me, spells.SpellHeal, me.GetLevel(), nil)
		case me.GetLevel() > 12:
			spells.Cast(me, me, spells.SpellCureCritic, me.GetLevel(), nil)
		default:
			spells.Cast(me, me, spells.SpellCureLight, me.GetLevel(), nil)
		}
	}

	// Find a dude to do evil things upon
	victName := me.GetFighting()
	if victName == "" {
		return specSummoner(w, ch, me, "", "")
	}

	// lspell = number(0, GET_LEVEL(ch)) + GET_LEVEL(ch)/5, capped at GET_LEVEL, min 1
	lspell := rand.Intn(me.GetLevel() + 1)
	lspell += me.GetLevel() / 5
	if lspell > me.GetLevel() {
		lspell = me.GetLevel()
	}
	if lspell < 1 {
		lspell = 1
	}

	// Prevent dispel-self if same alignment as victim (lspell < 3)
	if lspell < 3 {
		casterAlign := me.Prototype.Alignment
		// Check mobs in room for target
		for _, m := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if m.GetName() == victName {
				if (casterAlign <= -350 && m.Prototype.Alignment <= -350) ||
					(casterAlign >= 350 && m.Prototype.Alignment >= 350) {
					lspell = 4
				}
				break
			}
		}
		// Also check players in room for target
		for _, p := range w.GetPlayersInRoom(me.GetRoomVNum()) {
			if p.GetName() == victName {
				if (casterAlign <= -350 && p.IsEvil()) ||
					(casterAlign >= 350 && p.IsGood()) {
					lspell = 4
				}
				break
			}
		}
	}

	// Emergency teleport: HP < 25%, lspell > 25, not aggressive
	if me.GetHP() < me.GetMaxHP()/4 && lspell > 25 && !me.HasFlag("aggressive") {
		vict := findTargetInRoom(w, me.GetRoomVNum(), victName)
		if vict != nil {
			if rand.Intn(3) != 0 {
				spells.Cast(me, vict, spells.SpellTeleport, me.GetLevel(), nil)
			} else {
				spells.Cast(me, me, spells.SpellTeleport, me.GetLevel(), nil)
			}
		}
		return false
	}

	// Determine heal priority threshold (matches C faithfully, including unreachable branches)
	healPerc := 0
	switch {
	case me.GetHP() < me.GetMaxHP()/2:
		healPerc = 7
	case me.GetHP() < me.GetMaxHP()/4:
		healPerc = 5
	case me.GetHP() < me.GetMaxHP()/8:
		healPerc = 3
	}

	// Roll: hit foe (<3) vs heal self (>=3), out of (healPerc+2)
	if rand.Intn(healPerc+2) >= 2 {
		// Heal self — check curses, poisons, blindness
		if mobHasAffect(me, "blind") && lspell >= 4 && rand.Intn(4) == 0 {
			spells.Cast(me, me, spells.SpellCureBlind, me.GetLevel(), nil)
			return true
		}
		if mobHasAffect(me, "curse") && lspell >= 6 && rand.Intn(7) == 0 {
			spells.Cast(me, me, spells.SpellRemoveCurse, me.GetLevel(), nil)
			return true
		}
		if mobHasAffect(me, "poison") && lspell >= 5 && rand.Intn(7) == 0 {
			spells.Cast(me, me, spells.SpellRemovePoison, me.GetLevel(), nil)
			return true
		}

		// Heal self by level (1 in 4 chance)
		if rand.Intn(4) == 0 {
			switch {
			case lspell <= 5:
				spells.Cast(me, me, spells.SpellCureLight, me.GetLevel(), nil)
			case lspell <= 17:
				// Intentionally do nothing (matches C: cases 6-17 break)
			case lspell == 18:
				spells.Cast(me, me, spells.SpellCureCritic, me.GetLevel(), nil)
			default:
				if !mobHasAffect(me, "sanctuary") {
					spells.Cast(me, me, spells.SpellSanctuary, me.GetLevel(), nil)
				} else {
					spells.Cast(me, me, spells.SpellHeal, me.GetLevel(), nil)
				}
			}
		}
		return true
	}

	// Hit a foe — find the victim
	vict := findTargetInRoom(w, me.GetRoomVNum(), victName)
	if vict == nil {
		return false
	}

	// Call lightning if outside, lspell >= 15 (1-in-6)
	room := w.GetRoomInWorld(me.GetRoomVNum())
	if room != nil && room.Sector != SECT_INSIDE && lspell >= 15 && rand.Intn(6) == 0 {
		spells.Cast(me, vict, spells.SpellCallLightning, me.GetLevel(), nil)
		return true
	}

	// Offensive spells by lspell
	switch {
	case lspell <= 3:
		if me.Prototype.Alignment <= -350 {
			spells.Cast(me, vict, spells.SpellDispelGood, me.GetLevel(), nil)
		} else {
			spells.Cast(me, vict, spells.SpellDispelEvil, me.GetLevel(), nil)
		}
	case lspell <= 6:
		spells.Cast(me, vict, spells.SpellBlindness, me.GetLevel(), nil)
	case lspell == 7:
		spells.Cast(me, vict, spells.SpellCurse, me.GetLevel(), nil)
	case lspell <= 16:
		spells.Cast(me, vict, spells.SpellPoison, me.GetLevel(), nil)
	case lspell <= 19:
		spells.Cast(me, vict, spells.SpellEarthquake, me.GetLevel(), nil)
	case lspell <= 24:
		// Intentionally do nothing (matches C: cases 20-24 break)
	default:
		spells.Cast(me, vict, spells.SpellHarm, me.GetLevel(), nil)
	}

	return true
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
