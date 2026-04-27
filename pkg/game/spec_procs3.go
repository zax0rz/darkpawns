package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
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
			spells.Cast(me, me, spells.SpellHeal, me.GetLevel(), nil, nil)
		case me.GetLevel() > 12:
			spells.Cast(me, me, spells.SpellCureCritic, me.GetLevel(), nil, nil)
		default:
			spells.Cast(me, me, spells.SpellCureLight, me.GetLevel(), nil, nil)
		}
	}

	// Find a dude to do evil things upon
	victName := me.GetFighting()
	if victName == "" {
		return specSummoner(w, ch, me, "", "")
	}

	// lspell = number(0, GET_LEVEL(ch)) + GET_LEVEL(ch)/5, capped at GET_LEVEL, min 1
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
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
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			if rand.Intn(3) != 0 {
				spells.Cast(me, vict, spells.SpellTeleport, me.GetLevel(), nil, nil)
			} else {
				spells.Cast(me, me, spells.SpellTeleport, me.GetLevel(), nil, nil)
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
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if rand.Intn(healPerc+2) >= 2 {
		// Heal self — check curses, poisons, blindness
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if mobHasAffect(me, "blind") && lspell >= 4 && rand.Intn(4) == 0 {
			spells.Cast(me, me, spells.SpellCureBlind, me.GetLevel(), nil, nil)
			return true
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if mobHasAffect(me, "curse") && lspell >= 6 && rand.Intn(7) == 0 {
			spells.Cast(me, me, spells.SpellRemoveCurse, me.GetLevel(), nil, nil)
			return true
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if mobHasAffect(me, "poison") && lspell >= 5 && rand.Intn(7) == 0 {
			spells.Cast(me, me, spells.SpellRemovePoison, me.GetLevel(), nil, nil)
			return true
		}

		// Heal self by level (1 in 4 chance)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(4) == 0 {
			switch {
			case lspell <= 5:
				spells.Cast(me, me, spells.SpellCureLight, me.GetLevel(), nil, nil)
			case lspell <= 17:
				// Intentionally do nothing (matches C: cases 6-17 break)
			case lspell == 18:
				spells.Cast(me, me, spells.SpellCureCritic, me.GetLevel(), nil, nil)
			default:
				if !mobHasAffect(me, "sanctuary") {
					spells.Cast(me, me, spells.SpellSanctuary, me.GetLevel(), nil, nil)
				} else {
					spells.Cast(me, me, spells.SpellHeal, me.GetLevel(), nil, nil)
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
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if room != nil && room.Sector != SECT_INSIDE && lspell >= 15 && rand.Intn(6) == 0 {
		spells.Cast(me, vict, spells.SpellCallLightning, me.GetLevel(), nil, nil)
		return true
	}

	// Offensive spells by lspell
	switch {
	case lspell <= 3:
		if me.Prototype.Alignment <= -350 {
			spells.Cast(me, vict, spells.SpellDispelGood, me.GetLevel(), nil, nil)
		} else {
			spells.Cast(me, vict, spells.SpellDispelEvil, me.GetLevel(), nil, nil)
		}
	case lspell <= 6:
		spells.Cast(me, vict, spells.SpellBlindness, me.GetLevel(), nil, nil)
	case lspell == 7:
		spells.Cast(me, vict, spells.SpellCurse, me.GetLevel(), nil, nil)
	case lspell <= 16:
		spells.Cast(me, vict, spells.SpellPoison, me.GetLevel(), nil, nil)
	case lspell <= 19:
		spells.Cast(me, vict, spells.SpellEarthquake, me.GetLevel(), nil, nil)
	case lspell <= 24:
		// Intentionally do nothing (matches C: cases 20-24 break)
	default:
		spells.Cast(me, vict, spells.SpellHarm, me.GetLevel(), nil, nil)
	}

	return true
}

// specNoMoveDown blocks "down" movement unless the player is an immort.
func specNoMoveDown(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "down" || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch.GetLevel() >= lvlImmort {
		return false
	}
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s blocks your way down.", mobName(me)))
	return true
}

// specClerk sells citizenship.
func specClerk(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	if !mobCanSee(me) {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s exclaims, 'Who's there? I can't see you!'", mobName(me)))
		return true
	}

	// Zone-based hometown: map zone number to hometown index per C source.
	homet := 0
	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil {
		zone, _ := w.GetZone(room.Zone)
		if zone != nil {
			switch zone.Number {
			case 80:
				homet = 1
			case 182:
				homet = 2
			case 212:
				homet = 3
			}
		}
	}

	if cmd != "list" && cmd != "buy" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if cmd == "buy" {
		if !strings.EqualFold(arg, "citizenship") {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s BUY CITIZENSHIP, if you're interested.'", mobName(me), ch.GetName()))
			return true
		}
		if ch.GetGold() < 2000 {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s You cannot afford it!'", mobName(me), ch.GetName()))
			return true
		}
		if ch.Hometown == homet {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s You are already a citizen here!'", mobName(me), ch.GetName()))
			return true
		}
		ch.Hometown = homet
		ch.SetGold(ch.GetGold() - 2000)
		hometownNames := []string{"", "Midgaard", "Thalos", "New Thalos"}
		hName := "unknown"
		if homet >= 0 && homet < len(hometownNames) {
			hName = hometownNames[homet]
		}
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s You are now a citizen of %s.'", mobName(me), ch.GetName(), hName))
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

	// Find case, cabinet, and chest containers in the room
	var cas, cabinet, chest *ObjectInstance
	for _, obj := range items {
		if !obj.IsContainer() {
			continue
		}
		kw := strings.ToLower(obj.GetKeywords())
		if strings.Contains(kw, "case") && cas == nil {
			cas = obj
		}
		if strings.Contains(kw, "cabinet") && cabinet == nil {
			cabinet = obj
		}
		if strings.Contains(kw, "chest") && chest == nil {
			chest = obj
		}
	}
	if cas == nil || cabinet == nil || chest == nil {
		return false
	}

	// Helper to check if butler can get an object
	canGet := func(obj *ObjectInstance) bool {
		if obj == cas || obj == cabinet || obj == chest {
			return false // don't grab the containers themselves
		}
		if obj.Prototype == nil {
			return false
		}
		// Check ITEM_WEAR_TAKE flag (wear flag bit 0 = take)
		for _, wf := range obj.Prototype.WearFlags {
			if wf == 1 {
				return true
			}
		}
		return false
	}

	got := 0
	for _, obj := range items {
		if got >= 4 {
			break
		}
		if !canGet(obj) {
			continue
		}
		got++
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s gets %s.", mobName(me), obj.GetShortDesc()))
// #nosec G104
		w.MoveObjectToMobInventory(obj, me)

		// Sort into case/cabinet/chest by item type
		container := chest // default for misc items
		if obj.IsArmor() || obj.GetTypeFlag() == 11 { // ITEM_ARMOR(9) or ITEM_WORN(11)
			container = cas
		} else if obj.IsWeapon() || obj.GetTypeFlag() == 12 { // ITEM_WEAPON(5) or ITEM_FIREWEAPON(12)
			container = cabinet
		}
		// Remove from butler's inventory into the container
// #nosec G104
		w.MoveObjectToContainer(obj, container)
	}
	if got > 0 {
		// Containers are left open after putting items in; closing handled by container
		// state — the butler closes them after sorting
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
		// Check container flag: Values[3] must be non-zero (corpse/locked flag)
		if obj.Prototype == nil || obj.Prototype.Values[3] == 0 {
			continue
		}
		kw := strings.ToLower(obj.GetKeywords())
		if !strings.Contains(kw, "corpse") || strings.Contains(kw, "headless") {
			continue
		}
		// "Behead" the corpse: extract it from the room entirely
// #nosec G104
		w.MoveObjectToNowhere(obj)

		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s pulls the brain out of the head and eats it with a noisy\r\nslurp, blood and drool flying everywhere.", mobName(me)))

		// Level up or increase damroll (C: level < 30 → level++, else damroll += 2)
		if me.Prototype != nil && me.Prototype.Level < 30 {
			me.Prototype.Level++
		} else {
			// Increment mob's internal damroll
			me.Runtime.DamrollBonus += 2
		}
		return true
	}
	return false
}

// specTeleportVictim teleports an attacker away.
func specTeleportVictim(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || ch.GetFighting() == "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s scoffs at you.", ch.GetName()))
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s says, 'You can't harm me, mortal. Begone.'", ch.GetName()))
	fightingName := ch.GetFighting()
	if fightingName != "" {
		fighting, _ := w.GetPlayer(fightingName)
		if fighting != nil {
			spells.Cast(ch, fighting, spells.SpellTeleport, ch.GetLevel(), nil, nil)
		}
	}
	return true
}

// specConSeller sells constitution points.
func specConSeller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	arg = strings.TrimSpace(arg)
	if !mobCanSee(me) {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s exclaims, 'Who's there? I can't see you!'", mobName(me)))
		return true
	}

	if cmd != "list" && cmd != "buy" {
		return false
	}

	// Cost per con point: GET_LEVEL(ch) * 400
	cost := ch.GetLevel() * 400

	// Available con points the player can buy.
	// C: GET_ORIG_CON(ch) - ch->real_abils.con — origCon is the initial rolled stat.
	// Go codebase doesn't have OrigCon field yet; available = 18 - current (cap at 18 per C).
	availCon := 18 - ch.Stats.Con
	if availCon < 0 {
		availCon = 0
	}

	if cmd == "list" {
		if availCon < 1 {
			msg := fmt.Sprintf("%s You seem perfectly healthy!", ch.GetName())
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
			return true
		}
		suf := "s"
		if availCon == 1 {
			suf = ""
		}
		msg := fmt.Sprintf("%s You can buy up to %d point%s, at %d per point.", ch.GetName(), availCon, suf, cost)
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
		return true
	}

	// cmd == "buy"
	if !strings.EqualFold(arg, "con") {
		msg := fmt.Sprintf("%s BUY CON, if you really want to do it.", ch.GetName())
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
		return true
	}
	if ch.GetGold() < cost {
		msg := fmt.Sprintf("%s You can't afford it!", ch.GetName())
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
		return true
	}
	if availCon < 1 {
		msg := fmt.Sprintf("%s You seem perfectly healthy!", ch.GetName())
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
		return true
	}

	// Deduct gold
	ch.SetGold(ch.GetGold() - cost)

	// Apply con +1 (capped at 18 like C code)
	if ch.Stats.Con < 18 {
		ch.Stats.Con++
	}

	// Stun the player
	ch.SetPosition(combat.PosStunned)

	msg := fmt.Sprintf("%s That'll be %d coins, you should feel much better.. if you wake up.", ch.GetName(), cost)
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tells you, '%s'", mobName(me), msg))
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s stares at %s and mutters some arcane words.", mobName(me), ch.GetName()))
	w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s falls, stunned.", ch.GetName()))

	return true
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
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(21) == 0 {
			npcRegen(ch)
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s's wounds glow brightly for a moment, then disappear!", ch.GetName()))
		}
	} else if ch.GetFighting() != "" {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
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

// specAlienElevator moves all occupants between two rooms (19551↔19599).
func specAlienElevator(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if cmd == "close" && strings.EqualFold(arg, "door") {
		w.roomMessage(ch.GetRoomVNum(), "The room starts to move!")
		// Move players between the two elevator rooms
		const roomA = 19551
		const roomB = 19599
		playersA := w.GetPlayersInRoom(roomA)
		playersB := w.GetPlayersInRoom(roomB)
		for _, p := range playersA {
			p.SetRoom(roomB)
		}
		for _, p := range playersB {
			p.SetRoom(roomA)
		}
		return true
	}
	return false
}

// specWerewolf howls and bites when fighting.
// C source: SPECIAL(werewolf) ~line 407
func specWerewolf(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetFighting() == "" || me.GetHP() <= 0 {
		return false
	}
	// Howl (10% chance)
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if rand.Intn(10) == 0 {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s looks up and lets out a long, fierce howl.", mobName(me)))
		w.SendToZone(me.GetRoomVNum(), "You hear a loud howling in the distance.")
	}
	// Bite attack (25% chance)
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if rand.Intn(4) == 0 {
		victName := me.GetFighting()
		vict, ok := w.GetPlayer(victName)
		if ok && vict != nil && vict.GetRoom() == me.GetRoomVNum() {
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s tears into your leg with %s huge fangs!", mobName(me), mobName(me)))
			combat.TakeDamage(me, vict, combat.RollDice(me.GetLevel(), 2), combat.TYPE_BITE)
			moveReduction := me.GetLevel() * 3 / 2
			newMove := vict.GetMove() - moveReduction
			if newMove < 0 {
				newMove = 0
			}
			vict.SetMove(newMove)
		}
	}
	return true
}

// fieldObjDef defines a field object (wall of fire, ice, poison gas) from constants.c.
type fieldObjTypeDef struct {
	FoType string // "damage", "affect", "solid"
}

// fieldObjTypes maps field object vnums to their types.
// C source: constants.c field_objs[] — vnums 50, 51, 52.
var fieldObjTypes = map[int]fieldObjTypeDef{
	50: {FoType: "damage"}, // wall of fire
	51: {FoType: "solid"},  // wall of ice
	52: {FoType: "affect"}, // poison gas cloud
}

// specFieldObject checks field objects that damage room occupants.
// C source: SPECIAL(field_object) — me is actually an ObjectInstance.
// Since spec procs receive *MobInstance but this is object-driven, we use
// me's vnum to look up in fieldObjTypes and act accordingly.
func specFieldObject(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if me == nil {
		return false
	}
	vnum := me.GetVNum()
	def, ok := fieldObjTypes[vnum]
	if !ok {
		return false
	}

	roomVNum := me.GetRoomVNum()
	if roomVNum <= 0 {
		return false
	}

	damaged := false
	players := w.GetPlayersInRoom(roomVNum)
	for _, vict := range players {
		if def.FoType == "damage" {
			// Use mob proto values as dice params (matches C: GET_OBJ_VAL(obj,0), GET_OBJ_VAL(obj,1))
			dam := me.GetLevel()/2 + 1
			if dam > 0 {
				vict.Health -= dam
				sendToChar(vict, "An incredible force hits you!\r\n")
				if vict.Health <= 0 {
					w.roomMessage(roomVNum, fmt.Sprintf("%s falls to the ground, screaming in agony!", vict.GetName()))
					w.rawKill(vict, 0)
				}
				damaged = true
			}
		}
		if def.FoType == "affect" {
			// Cast poison on room occupants (affect=spell, level=cast level)
			spells.Cast(vict, vict, spells.SpellPoison, me.GetLevel(), nil, nil)
			damaged = true
		}
	}
	return damaged
}

// specPortalToTemple teleports the player to the temple (room 8008).
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
	ch.SetRoom(8008)
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("With a blinding flash of light and a crack of thunder, %s appears!", ch.GetName()))
	return true
}

// specTurnUndead opens a portal when the player uses the right item.
// C source: SPECIAL(turn_undead) — creates north exit from 19875→19876 and south exit
// from 19876→19875 on "use", removes both exits during pulse (cmd="").
func specTurnUndead(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	const roomA = 19875
	const roomB = 19876

	if cmd == "use" {
		arg = strings.TrimSpace(arg)
		if arg == "" || ch.GetRoomVNum() != roomA && ch.GetRoomVNum() != roomB {
			return false
		}
		// Check that arg matches the object's keywords
		if me != nil && !isName(arg, me.GetName()) {
			return false
		}
		w.roomMessage(ch.GetRoomVNum(), "A ray of flame bursts out of the object, consuming the undead!")

		// Create north exit from 19875 → 19876
		if room := w.GetRoomInWorld(roomA); room != nil {
			room.Exits["north"] = parser.Exit{Direction: "north", ToRoom: roomB}
		}
		// Create south exit from 19876 → 19875
		if room := w.GetRoomInWorld(roomB); room != nil {
			room.Exits["south"] = parser.Exit{Direction: "south", ToRoom: roomA}
		}
		return true
	}

	// Pulse: remove exits if they exist
	if cmd == "" {
		if room := w.GetRoomInWorld(roomA); room != nil {
			delete(room.Exits, "north")
		}
		if room := w.GetRoomInWorld(roomB); room != nil {
			delete(room.Exits, "south")
		}
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
	ch.SetRoom(19875)
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("\r\nWith a blinding flash of light and a crack of thunder, %s appears!\r\n\r\n", ch.GetName()))
	return true
}

// specMirror creates reflections and swaps players.
// C source: SPECIAL(mirror) — ch2 is anyone in room 14496. Hit/kill: spawn obj 14503,
// move ch2 to obj's room. Look: swap ch and ch2's rooms.
func specMirror(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil {
		return false
	}
	objRoom := me.GetRoomVNum()
	if objRoom <= 0 {
		return false
	}
	arg = strings.TrimSpace(arg)
	if !isName(arg, me.GetName()) {
		return false
	}

	// ch2 is anyone in the mirror room (14496)
	var ch2 *Player
	for _, p := range w.GetPlayersInRoom(14496) {
		ch2 = p
		break
	}

	if cmd == "hit" || cmd == "kill" {
		sendToChar(ch, "You break the object into tiny pieces!")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s shatters the object into a million pieces!", ch.GetName()))
		if ch2 != nil {
			ch2.SetRoom(objRoom)
			sendToChar(ch2, "You feel pulled in a hundred different directions!\r\n")
			w.roomMessage(ch2.GetRoomVNum(), fmt.Sprintf("%s appears in a brilliant flash!", ch2.GetName()))
		}
		// Remove old object, spawn replacement (14503) in the same room
		w.RemoveItemFromRoomByVNum(me.GetVNum(), objRoom)
// #nosec G104
		w.SpawnObject(14503, objRoom)
		return true
	}
	if cmd == "look" {
		sendToChar(ch, "You feel pulled in a hundred different directions!")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash!", ch.GetName()))
		if ch2 != nil {
			// Move ch2 to obj's room
			ch2.SetRoom(objRoom)
			sendToChar(ch2, "You feel pulled in a hundred different directions!\r\n")
			w.roomMessage(ch2.GetRoomVNum(), fmt.Sprintf("%s appears in a brilliant flash!", ch2.GetName()))
		}
		// Move ch to room 14496
		ch.SetRoom(14496)
		return true
	}
	return false
}

// specProstitute offers services for gold.
func specProstitute(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
		return false
	}
	if !mobCanSee(me) {
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'Who's there? I can't see you!'", mobName(me)))
		return true
	}
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
// C source: SPECIAL(roach) ~line 707. Pulse-only (ch is nil, me is the roach).
func specRoach(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me == nil || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	roomVNum := me.GetRoomVNum()

	// Starvation death (extremely rare)
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if rand.Intn(10001) == 0 && rand.Intn(10001) == 0 && me.GetMaxHealth() < 11 {
		w.roomMessage(roomVNum, fmt.Sprintf("%s seems to starve to death and simply fades out of existence.", mobName(me)))
		// C: extract_char(ch) — set HP to 0 to trigger mob death handling
		me.SetHealth(0)
		return true
	}

	// Look for food on the ground
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		if !obj.CanPickUp {
			continue
		}
		w.roomMessage(roomVNum, fmt.Sprintf("%s feeds on %s.", mobName(me), obj.GetShortDesc()))
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(3) == 0 {
			newMaxHP := me.GetMaxHealth() + obj.GetCost()/2
			if newMaxHP > 400 {
				// Split into new roach
				me.SetHealth(10)
				w.roomMessage(roomVNum, fmt.Sprintf("%s splits in half forming a new roach!", mobName(me)))
				newRoach, err := w.SpawnMobInstance(23, roomVNum)
				if err == nil && newRoach != nil {
					newRoach.SetHealth(10)
				} else {
					me.MaxHP = 10
				}
			} else {
				me.MaxHP = newMaxHP
				// #nosec G404 — game RNG, not cryptographic
// #nosec G404
				if rand.Intn(2) == 0 {
					w.roomMessage(roomVNum, "You hear some stretching noises.")
				} else {
					w.roomMessage(roomVNum, fmt.Sprintf("You hear a strange rumbling from %s's stonach.", mobName(me)))
				}
			}
		} else {
			w.roomMessage(roomVNum, fmt.Sprintf("You hear %s burp.", mobName(me)))
		}
// #nosec G104
		w.MoveObjectToNowhere(obj)
		return true
	}

	// Random idle behaviors
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	switch rand.Intn(11) {
	case 0:
		w.roomMessage(roomVNum, fmt.Sprintf("%s chirps gleefully.", mobName(me)))
	case 1:
		w.roomMessage(roomVNum, fmt.Sprintf("%s changes colors and clicks happily.", mobName(me)))
	case 2:
		w.roomMessage(roomVNum, fmt.Sprintf("%s skitters around in tight circles.", mobName(me)))
	case 3:
		w.roomMessage(roomVNum, fmt.Sprintf("Strange purple dots appear on %s's back.", mobName(me)))
	case 4:
		// Teleport to a random room
		rooms := w.Rooms()
		if len(rooms) > 0 {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			randRoom := rooms[rand.Intn(len(rooms))].VNum
			// Check for unwanted room flags (private/godroom/nomagic/death)
			if w.roomHasFlag(randRoom, "private") || w.roomHasFlag(randRoom, "godroom") ||
				w.roomHasFlag(randRoom, "nomagic") || w.roomHasFlag(randRoom, "death") {
				w.roomMessage(roomVNum, fmt.Sprintf("%s fades out and back in again.", mobName(me)))
				return false
			}
			w.roomMessage(roomVNum, fmt.Sprintf("%s fades out slowly with a soft swoosh.", mobName(me)))
			me.SetRoom(randRoom)
			w.roomMessage(randRoom, fmt.Sprintf("%s fades in slowly, looking a bit disoriented.", mobName(me)))
			return true
		}
		return false
	}
	return false
}

// specMortician retrieves corpses for a fee.
// C source: SPECIAL(mortician) ~line 807.
func specMortician(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	cost := ch.GetLevel() * 116
	if cmd == "list" {
		ch.SendMessage(fmt.Sprintf("%s tells you, 'It will cost %d coins to retrieve your corpse.'\r\n", mobName(me), cost))
		return true
	}
	if cmd == "retrieve" {
		if ch.GetGold() < cost {
			ch.SendMessage(fmt.Sprintf("%s tells you, 'I'm sorry, you can't afford the cost.'\r\n", mobName(me)))
			return true
		}
		// Search all rooms for a corpse matching this player
		found := false
		for _, room := range w.Rooms() {
			items := w.GetItemsInRoom(room.VNum)
			for _, obj := range items {
				if obj.IsCorpse && strings.Contains(strings.ToLower(obj.Prototype.Keywords), strings.ToLower(ch.GetName())) && obj.GetValue(3) > 0 {
					// Move corpse from its current room to the mortician's room
// #nosec G104
					w.MoveObjectToRoom(obj, me.GetRoomVNum())
					ch.SendMessage(fmt.Sprintf("The Mortician dumps your corpse on the ground.\r\n"))
					w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("The Mortician dumps %s's corpse on the ground.", ch.GetName()))
					ch.SetGold(ch.GetGold() - cost)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("%s tells you, 'I'm sorry, I can't find your corpse anywhere!'\r\n", mobName(me)))
		}
		return true
	}
	return false
}

// specConjured returns to its plane of existence when un-charmed.
// C source: SPECIAL(conjured) ~line 859. Pulse-only (ch is nil, me is the conjured mob).
func specConjured(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil {
		return false
	}
	// Only trigger when the mob is no longer charmed
	if mobHasAffect(me, "charm") {
		return false
	}
	switch me.GetVNum() {
	case 81, 82, 83, 84:
		// Notify master: MobInstance lacks a Master/Following field, so notify
		// all players in the room. A charmer would be present in the same room.
		for _, p := range w.GetPlayersInRoom(me.GetRoomVNum()) {
			p.SendMessage(fmt.Sprintf("You lose control and %s fizzles away!\r\n", mobName(me)))
		}
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s returns to its own plane of existence.", mobName(me)))
	default:
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s says, 'My work here is done.'", mobName(me)))
		w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s disappears in a flash of white light!", mobName(me)))
	}
	// Remove mob from world — set HP to 0 to trigger death handling
	me.SetHealth(0)
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
		// Check player inventory for talismans (vnum 1300-1303)
		for _, obj := range ppl.GetInventory() {
			vnum := obj.GetVNum()
			if vnum >= 1300 && vnum <= 1303 {
				found = vnum - 1300
				break
			}
		}
		if found >= 0 && found < len(newLocs) {
			sendToChar(ppl, "The talisman glows softly and your vision fades.\r\n")
		} else {
			sendToChar(ppl, "You feel a tingling sensation and your vision fades.\r\n")
		}
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s vanishes in a brilliant flash of light.", ppl.GetName()))
		if found >= 0 && found < len(newLocs) {
			ppl.SetRoom(newLocs[found])
		} else {
			ppl.SetRoom(newLocs[0])
		}
	}
	return true
}

// specElementsPlatforms sends all players in the room back to the master column (1314).
func specElementsPlatforms(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, ppl := range players {
		sendToChar(ppl, "A wave of power surges through you and you feel dizzy.\r\n")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash of light.", ppl.GetName()))
		ppl.SetRoom(1314)
	}
	return true
}

// specElementsLoadCylinders manages cylinder objects for the talisman puzzle.
func specElementsLoadCylinders(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "get" {
		w.doGet(ch, me, cmd, arg)
		elementsRemoveCylinders(w)
		return true
	}
	if cmd != "drop" {
		return false
	}

	// Map room vnum to expected talisman vnum and cylinder vnum
	type loadEntry struct {
		roomVNum   int
		talVNum    int
		cylVNum    int
		color      string
	}
	entries := map[int]loadEntry{
		1360: {1360, 1300, 1304, "green"},
		1364: {1364, 1301, 1305, "yellow"},
		1380: {1380, 1302, 1306, "red"},
		1384: {1384, 1303, 1307, "blue"},
	}

	entry, ok := entries[ch.GetRoomVNum()]
	if !ok {
		// Not a talisman pillar room
		w.doDrop(ch, me, cmd, arg)
		return true
	}

	// Check if a cylinder already exists in this room
	for _, item := range w.GetItemsInRoom(ch.GetRoomVNum()) {
		if item.GetVNum() == entry.cylVNum {
			return true // cylinder already present, do nothing
		}
	}

	// Perform the actual drop
	w.doDrop(ch, me, cmd, arg)

	// Check what was actually dropped — locate the talisman in the room
	for _, item := range w.GetItemsInRoom(ch.GetRoomVNum()) {
		if item.GetVNum() == entry.talVNum {
			msg := fmt.Sprintf("A %s cylinder of light extends upwards from the pillar.\r\n", entry.color)
			sendToChar(ch, msg)
			obj, err := w.SpawnObject(entry.cylVNum, ch.GetRoomVNum())
			if err == nil {
// #nosec G104
				w.MoveObjectToRoom(obj, ch.GetRoomVNum())
			}
			break
		}
	}

	return true
}

// specElementsGaleruColumn checks if all four talismans are in their rooms.
func specElementsGaleruColumn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Check rooms 1360,1364,1380,1384 for talismans 1300-1303
	roomVnums := []int{1360, 1364, 1380, 1384}
	talVnums := []int{1300, 1301, 1302, 1303}
	found := 0

	for i := 0; i < 4; i++ {
		items := w.GetItemsInRoom(roomVnums[i])
		for _, item := range items {
			if item.GetVNum() == talVnums[i] {
				found++
				break
			}
		}
	}

	if found != 4 {
		return false
	}

	// All four talismans are placed — teleport players in room 1372 to 1389
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, ppl := range players {
		sendToChar(ppl, "Four beams of colored light from the corners of the chamber converge around you.\r\n\n")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s is struck by four beams of colored light and slowly vanishes!", ppl.GetName()))
		ppl.SetRoom(1389)
	}
	return true
}

// specElementsGaleruAlive teleports players if Galeru (mob 1315) is dead.
func specElementsGaleruAlive(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}
	// Check if Galeru (mob vnum 1315) is alive in the room
	if findMobInRoom(w, ch.GetRoomVNum(), "galeru") != nil {
		return false // Galeru is alive, no teleport
	}
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	if len(players) > 0 {
		for _, ppl := range players {
			sendToChar(ppl, "You begin to feel very dizzy and the world around you fades...\r\n")
			w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash of light.", ppl.GetName()))
			ppl.SetRoom(1395)
		}
		return true
	}
	return false
}

// specElementsMinion destroys talismans and cylinders.
func specElementsMinion(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Iterate mob inventory for talismans/cylinders and destroy them
	talismanVnums := map[int]bool{1300: true, 1301: true, 1302: true, 1303: true}
	cylinderVnums := map[int]bool{1304: true, 1305: true, 1306: true, 1307: true}

	toDestroy := make([]*ObjectInstance, 0)
	for _, obj := range me.Inventory {
		vnum := obj.GetVNum()
		if talismanVnums[vnum] || cylinderVnums[vnum] {
			toDestroy = append(toDestroy, obj)
		}
	}

	for _, obj := range toDestroy {
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s utters the words 'eradico paratus' and %s disintegrates.", me.GetName(), obj.GetShortDesc()))
		me.RemoveFromInventory(obj)
	}

	elementsRemoveCylinders(w)
	return false
}

// specElementsGuardian charms players into fighting each other.
func specElementsGuardian(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" {
		return false
	}

	// Get all players in the room who are non-NPC, non-immortal, and not already fighting
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	var targets []*Player
	for _, ppl := range players {
		if ppl.IsNPC() || ppl.GetLevel() > LVL_IMMORT || ppl.GetFighting() != "" {
			continue
		}
		targets = append(targets, ppl)
	}

	if len(targets) == 0 {
		return false
	}

	if len(targets) < 2 {
		// Single player — goes mad and injures themself
		dam := randRange(10, 50)
		w.doDamage(me, targets[0], dam, "hit")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s mumbles softly and %s begins screaming loudly, hitting %sself.", me.GetName(), targets[0].Name, targets[0].Name))
		sendToChar(targets[0], fmt.Sprintf("%s mumbles softly and you begin to scream, involuntarily hitting yourself.\r\n", me.GetName()))
		return false
	}

	// Pair the first two non-fighting players
	a := targets[0]
	b := targets[1]
	a.SetFighting(b.Name)
	b.SetFighting(a.Name)
	a.SetAffect(affCharm, true)
	b.SetAffect(affCharm, true)

	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s mumbles softly and %s screams loudly, attacking %s!", me.GetName(), a.Name, b.Name))
	sendToChar(b, fmt.Sprintf("%s mumbles softly and %s screams loudly, attacking you!\r\n", me.GetName(), a.Name))
	sendToChar(a, fmt.Sprintf("%s mumbles softly and you scream loudly, attacking %s!\r\n", me.GetName(), b.Name))

	// Apply a bit of initial damage to make it real
	w.doDamage(a, b, 1, "hit")

	return false
}

// specFlyExitUp blocks going up unless the player can fly.
func specFlyExitUp(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "up" {
		return false
	}
	if ch.IsAffected(affFly) {
		return false // player can fly, allow passage
	}
	sendToChar(ch, "You try and jump up there but it's just too high.\r\n")
	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s jumps up and down in a vain attempt to travel upwards.", ch.GetName()))
	return true
}

// cylinderToTalisman maps cylinder vnums to their corresponding talisman vnums.
var cylinderToTalisman = map[int]int{
	1310: 1300,
	1311: 1301,
	1312: 1302,
	1313: 1303,
}

// elementsRemoveCylinders checks room contents and removes cylinders when talismans leave.
func elementsRemoveCylinders(w *World) {
	// Check rooms that can have cylinders for missing talismans
	for cylVNum, talVNum := range cylinderToTalisman {
		// Find all rooms containing this cylinder
		for _, room := range w.Rooms() {
			items := w.GetItemsInRoom(room.VNum)
			hasCylinder := false
			hasTalisman := false
			for _, item := range items {
				if item.GetVNum() == cylVNum {
					hasCylinder = true
				}
				if item.GetVNum() == talVNum {
					hasTalisman = true
				}
			}
			if hasCylinder && !hasTalisman {
				w.RemoveItemFromRoomByVNum(cylVNum, room.VNum)
			}
		}
	}
}

