package game

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// msgNoStealHere is src/shop.h:193's MSG_NO_STEAL_HERE, expanded by the
// native shopkeeper special in src/shop.c:975.
const msgNoStealHere = "$n is a bloody thief!"

// mobHasAffect checks if a MobInstance has a given affect flag string in its prototype.
func mobHasAffect(me *MobInstance, affect string) bool {
	for _, f := range me.Proto().AffectFlags {
		if strings.EqualFold(f, affect) {
			return true
		}
	}
	return false
}

// castClericSpell preserves the NPC cast_spell() entry point used by C's
// cleric procedure. In particular, it emits the verbal component before
// dispatching call_magic; a direct spells.Cast would silently omit those
// player-facing bytes.
func castClericSpell(w *World, me *MobInstance, target interface{}, spellNum int) bool {
	victim, ok := target.(combat.Combatant)
	if !ok {
		return false
	}
	return castMobSpell(w, me, victim, spellNum)
}

// clericNumber is the cleric procedure's RNG seam. It preserves the shared
// game roller in production while letting the branch tests pin C's draw order.
var clericNumber = dprng.Number

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

// specShopKeeper handles the command-time shopkeeper behavior that cannot be
// reached through the ordinary session shop handlers. Transaction commands
// are handled by pkg/session/shop_cmds.go; C's steal interception remains a
// mobile special because it must run before the steal command handler.
func specShopKeeper(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil || cmd != "steal" {
		return false
	}
	shop, ok := w.GetShopByKeeper(me.GetVNum())
	if !ok {
		return false
	}
	inShopRoom := false
	for _, room := range shop.Rooms {
		if room == ch.GetRoomVNum() {
			inShopRoom = true
			break
		}
	}
	if !inShopRoom || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if ch.GetRace() == RaceKender {
		w.mobShoutGenComm(me, "Stinkin' Kender scum!")
	}
	w.npcSocial(me, "slap", ch.GetName())
	Act(w, false, ch, me, nil, nil, "$N shouts '"+msgNoStealHere+"'", "", ToChar)
	return true
}

// specCleric — cleric mob: heals self, casts offensive/defensive spells while fighting.
// Ported from SPECIAL(cleric) in spec_procs.c (line 1425).
// Uses `me` (MobInstance) for all mob state — `ch` is nil during pulse calls.
func specCleric(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// In C, IS_NPC(ch) and AWAKE(ch) guard on the mob. In Go, `me` is always the mob.
	// If ch != nil and ch is not an NPC, a player triggered this via command — not our spec.
	if me == nil || ch != nil || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	if cmd != "" || me.GetHP() < 0 {
		return false
	}

	// Stand up if between stunned and standing (C: AWAKE check + do_stand)
	if me.GetPosition() != combat.PosFighting {
		if me.GetPosition() > combat.PosStunned && me.GetPosition() < combat.PosStanding {
			switch me.GetPosition() {
			case combat.PosSitting:
				Act(w, true, me, nil, nil, nil, "$n clambers to $s feet.", "", ToRoom)
			case combat.PosResting:
				Act(w, true, me, nil, nil, nil, "$n stops resting, and clambers on $s feet.", "", ToRoom)
			}
			me.SetPosition(combat.PosStanding)
		}
	}

	// Do nothing in peaceful rooms
	if w.roomHasFlag(me.GetRoomVNum(), "peaceful") {
		return false
	}

	// If not fighting and below max HP-10, heal self
	if me.GetFightingBody() == nil && me.GetHP() < me.GetMaxHP()-10 {
		switch {
		case me.GetLevel() >= 20:
			castMobSpell(w, me, me, spells.SpellHeal)
		case me.GetLevel() > 12:
			castMobSpell(w, me, me, spells.SpellCureCritic)
		default:
			castMobSpell(w, me, me, spells.SpellCureLight)
		}
	}

	// Find a dude to do evil things upon
	vict := me.GetFightingBody()
	if vict == nil {
		return specSummoner(w, ch, me, "", "")
	}

	// lspell = number(0, GET_LEVEL(ch)) + GET_LEVEL(ch)/5, capped at GET_LEVEL, min 1
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	lspell := clericNumber(0, me.GetLevel())
	lspell += me.GetLevel() / 5
	if lspell > me.GetLevel() {
		lspell = me.GetLevel()
	}
	if lspell < 1 {
		lspell = 1
	}

	// Prevent dispel-self if same alignment as victim (lspell < 3)
	if lspell < 3 {
		casterAlign := me.GetAlignment()
		// Check mobs in room for target
		for _, m := range w.GetMobsInRoom(me.GetRoomVNum()) {
			if m == vict {
				if (casterAlign <= -350 && m.GetAlignment() <= -350) ||
					(casterAlign >= 350 && m.GetAlignment() >= 350) {
					lspell = 4
				}
				break
			}
		}
		// Also check players in room for target
		for _, p := range w.GetPlayersInRoom(me.GetRoomVNum()) {
			if p == vict {
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
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if clericNumber(0, 2) != 0 {
			castMobSpell(w, me, me, spells.SpellTeleport)
		} else {
			castClericSpell(w, me, vict, spells.SpellTeleport)
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
	if clericNumber(1, healPerc+2) >= 3 {
		// Heal self — check curses, poisons, blindness
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if mobHasAffect(me, "blind") {
			// C parses the bitwise '&' before the surrounding '&&', so an
			// affected cleric consumes this draw even when lspell < 4.
			blindRoll := clericNumber(0, 3)
			if lspell >= 4 && blindRoll == 0 {
				castClericSpell(w, me, vict, spells.SpellCureBlind)
				return true
			}
		}
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if mobHasAffect(me, "curse") && lspell >= 6 && clericNumber(0, 6) == 0 {
			castClericSpell(w, me, vict, spells.SpellRemoveCurse)
			return true
		}
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if mobHasAffect(me, "poison") && lspell >= 5 && clericNumber(0, 6) == 0 {
			castClericSpell(w, me, vict, spells.SpellRemovePoison)
			return true
		}

		// Heal self by level (1 in 4 chance)
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if clericNumber(0, 3) == 0 {
			switch {
			case lspell <= 5:
				castMobSpell(w, me, me, spells.SpellCureLight)
			case lspell <= 17:
				// Intentionally do nothing (matches C: cases 6-17 break)
			case lspell == 18:
				castMobSpell(w, me, me, spells.SpellCureCritic)
			default:
				if !mobHasAffect(me, "sanctuary") {
					castMobSpell(w, me, me, spells.SpellSanctuary)
				} else {
					castMobSpell(w, me, me, spells.SpellHeal)
				}
			}
		}
		return true
	}

	// Call lightning if outside, lspell >= 15 (1-in-6)
	room := w.GetRoomInWorld(me.GetRoomVNum())
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	if room != nil && w.IsOutside(me.GetRoom()) && WeatherSnapshot().Sky >= SkyRaining && lspell >= 15 && clericNumber(0, 5) == 0 {
		Act(w, true, me, nil, nil, nil, "$n stares into the sky.", "", ToRoom)
		castClericSpell(w, me, vict, spells.SpellCallLightning)
		return true
	}

	// Offensive spells by lspell
	switch {
	case lspell <= 3:
		if me.GetAlignment() <= -350 {
			castClericSpell(w, me, vict, spells.SpellDispelGood)
		} else {
			castClericSpell(w, me, vict, spells.SpellDispelEvil)
		}
	case lspell <= 6:
		castClericSpell(w, me, vict, spells.SpellBlindness)
	case lspell == 7:
		castClericSpell(w, me, vict, spells.SpellCurse)
	case lspell <= 11, lspell >= 13 && lspell <= 16:
		castClericSpell(w, me, vict, spells.SpellPoison)
	case lspell >= 17 && lspell <= 19:
		castClericSpell(w, me, vict, spells.SpellEarthquake)
	case lspell <= 24:
		// Intentionally do nothing (matches C: cases 20-24 break)
	default:
		castClericSpell(w, me, vict, spells.SpellHarm)
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
	if me == nil || ch == nil || cmd == "" || ch.GetFightingBody() != nil || ch.GetPosition() <= combat.PosSleeping {
		return false
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
			default:
				sendToChar(ch, "default case reached in clerk special - tell a god")
			}
		}
	}

	if cmd != "list" && cmd != "buy" {
		return false
	}
	if !canSeeForPers(me, ch) {
		Act(w, true, me, nil, nil, nil, "$n exclaims, 'Who's there? I can't see you!'", "", ToRoom)
		return true
	}
	arg = strings.TrimSpace(arg)
	if cmd == "buy" {
		if !strings.EqualFold(arg, "citizenship") {
			tellFromMob(me, ch, "BUY CITIZENSHIP, if you're interested.")
			return true
		}
		if ch.GetGold() < 2000 {
			tellFromMob(me, ch, "You cannot afford it!")
			return true
		}
		if ch.Hometown == homet {
			tellFromMob(me, ch, "You are already a citizen here!")
			return true
		}
		ch.Hometown = homet
		ch.SetGold(ch.GetGold() - 2000)
		hName := HometownName(homet)
		tellFromMob(me, ch, fmt.Sprintf("You are now a citizen of %s.", hName))
		// C's clerk saves after the citizenship purchase
		// (spec_procs3.c:139).
		if res := w.SavePlayerRecord(ch, "clerk citizenship", LoadRoomNowhere, SaveCharOnly); res == SaveFailed {
			slog.Error("store-of-record save failed on citizenship", "player", ch.GetName())
		}
		return true
	}
	if cmd == "list" {
		tellFromMob(me, ch, "Citizenship costs 2,000 coins.")
		return true
	}
	return false
}

// butlerVisibleRoomObject mirrors get_obj_in_list_vis(mobile, name, room
// contents), including numbered names, prefix matching, and object visibility.
func butlerVisibleRoomObject(w *World, me *MobInstance, name string, items []*ObjectInstance) *ObjectInstance {
	query := strings.TrimSpace(name)
	number := GetNumber(&query)
	if number <= 0 || query == "" {
		return nil
	}
	found := 0
	for _, obj := range items {
		if obj == nil || !isnameWithAbbrevs(query, obj.GetKeywords()) {
			continue
		}
		if !canSeeObject(me, obj) && obj.GetTypeFlag() != ITEM_LIGHT {
			continue
		}
		found++
		if found == number {
			return obj
		}
	}
	return nil
}

// butlerDoorToggle mirrors do_gen_door/do_doorcmd for the object targets used
// by SPECIAL(butler). C's mob has no descriptor, but the room act is still
// visible to players in the room.
func butlerDoorToggle(w *World, me *MobInstance, container *ObjectInstance, open bool) {
	if container == nil || container.GetTypeFlag() != ITEM_CONTAINER {
		return
	}
	flags := container.GetValue(contFlags)
	if flags&contCloseable == 0 {
		return
	}
	if open {
		if flags&contClosed != 0 && flags&contLocked == 0 {
			// C's do_gen_door reaches ok_pick on each successful state change,
			// even though the mobile has no descriptor. Preserve its draw.
			// #nosec G404 — game RNG, not cryptographic
			_ = doorNumber(1, 101) // src/act.movement.c:541
			container.SetValue(contFlags, flags&^contClosed)
			Act(w, false, me, nil, container, nil, "$n opens $p.", "", ToRoom)
		}
		return
	}
	if flags&contClosed == 0 {
		// #nosec G404 — game RNG, not cryptographic
		_ = doorNumber(1, 101) // src/act.movement.c:541
		container.SetValue(contFlags, flags|contClosed)
		Act(w, false, me, nil, container, nil, "$n closes $p.", "", ToRoom)
	}
}

// butlerPerformPut mirrors perform_put's base-weight capacity check. C uses
// GET_OBJ_WEIGHT(cont), not the container's recursive contents weight.
func (w *World) butlerPerformPut(me *MobInstance, obj, container *ObjectInstance) bool {
	if container.GetWeight()+obj.GetWeight() > container.GetValue(contCapacity) {
		return false
	}
	if err := w.MoveObjectToContainer(obj, container); err != nil {
		slog.Warn("MoveObjectToContainer failed in butler spec", "obj", obj.GetVNum(), "container", container.GetVNum(), "error", err)
		return false
	}
	Act(w, true, me, nil, obj, container, "$n puts $p in $P.", "", ToRoom)
	return true
}

// specButler tidies up the room, picking up loose items and storing them.
func specButler(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetFightingBody() != nil {
		return false
	}
	items := append([]*ObjectInstance(nil), w.GetItemsInRoom(me.GetRoomVNum())...)

	// C resolves the three targets with get_obj_in_list_vis before its entry
	// gate. Keep the same room-list order and visibility semantics.
	cas := butlerVisibleRoomObject(w, me, "case", items)
	cabinet := butlerVisibleRoomObject(w, me, "cabinet", items)
	chest := butlerVisibleRoomObject(w, me, "chest", items)
	if cas == nil || cabinet == nil || chest == nil {
		return false
	}

	canGet := func(obj *ObjectInstance) bool { return w.canGetObject(me, obj) }

	got := 0
	// C walks world[room].contents from the head (newest first; obj_to_room
	// prepends). Room lists are stored in that order, so walk forward.
	for _, obj := range items {
		if got >= 4 {
			break
		}
		if !canGet(obj) {
			continue
		}
		got++
		Act(w, true, me, nil, nil, obj, "$n gets $P.", "", ToRoom)
		if err := w.MoveObjectToMobInventoryFront(obj, me); err != nil {
			slog.Warn("MoveObjectToMobInventory failed in butler spec", "obj", obj.GetVNum(), "mob", me.GetName(), "error", err)
			continue
		}

		// Sort into case/cabinet/chest by item type, matching the C branches.
		container := chest
		if obj.GetTypeFlag() == ITEM_ARMOR || obj.GetTypeFlag() == ITEM_WORN {
			container = cas
		} else if obj.GetTypeFlag() == ITEM_WEAPON || obj.GetTypeFlag() == ITEM_FIRE_WEAPON {
			container = cabinet
		}
		butlerDoorToggle(w, me, container, true)
		w.butlerPerformPut(me, obj, container)
	}
	if got > 0 {
		butlerDoorToggle(w, me, cas, false)
		butlerDoorToggle(w, me, cabinet, false)
		butlerDoorToggle(w, me, chest, false)
		return true
	}
	return false
}

// specBrainEater eats brains from headless corpses.
// C: SPECIAL(brain_eater) is always called with ch==the mob itself
// (mobile_activity.c calls func(ch, ch, 0, "")); ch is nil in the Go
// autonomous path, so the mob's own state must be read via me.
func specBrainEater(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || me == nil || me.GetFightingBody() != nil || cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() < 0 {
		return false
	}
	items := w.GetItemsInRoom(me.GetRoomVNum())
	eligible := false
	for _, obj := range items {
		if obj == nil || !obj.IsContainer() || obj.GetValue(3) == 0 {
			continue
		}
		// C's outer loop uses literal strstr() checks before it invokes
		// do_behead(mobile, "corpse", 0, 0).
		kw := obj.GetKeywords()
		if strings.Contains(kw, "corpse") && !strings.Contains(kw, "headless") {
			eligible = true
			break
		}
	}
	if !eligible {
		return false
	}

	// do_behead() resolves the first visible room object matching the literal
	// argument "corpse", which can differ from the qualifying outer-loop
	// object. Its direct messages target the NPC and are not player-visible;
	// brain_eater nevertheless continues with its own room Act on failure.
	var target *ObjectInstance
	for _, obj := range items {
		if obj != nil && isnameWithAbbrevs("corpse", obj.GetKeywords()) && canSeeObject(me, obj) {
			target = obj
			break
		}
	}
	if target != nil && target.GetTypeFlag() == ITEM_CONTAINER && target.GetValue(3) != 0 && !strings.Contains(target.GetKeywords(), "headless") {
		wielded := me.Equipped(mobWearWield)
		slashWeapon := wielded != nil && wielded.GetValue(3) == 3
		result := performBehead(w, me, target, wielded != nil, slashWeapon,
			func(headObj *ObjectInstance) bool {
				return headObj.IsTakeable() && mobCarriedWeight(me)+headObj.GetWeight() <= mobMaxCarryWeight(me) && mobCarriedItemCount(me)+1 <= mobMaxCarryCount(me)
			},
			func(headObj *ObjectInstance) error { return w.MoveObjectToMobInventoryFront(headObj, me) },
		)
		if result.Success {
			if slashWeapon {
				Act(w, true, me, nil, target, nil, "$n beheads $p!", "", ToRoom)
			} else {
				Act(w, true, me, nil, target, nil, "$n rips the head off $p!", "", ToRoom)
			}
		}
	}

	// C ignores do_behead's void result and always continues here.
	Act(w, true, me, nil, nil, nil,
		"$n pulls the brain out of the head and eats it with a noisy\r\nslurp, blood and drool flying everywhere.", "", ToRoom)

	// GET_LEVEL(mobile) and GET_DAMROLL(mobile) are per-instance fields in
	// read_mobile's copied character, never shared prototype state.
	if me.GetLevel() < 30 {
		me.SetLevel(me.GetLevel() + 1)
	} else {
		me.AddDamrollBonus(2)
	}
	return true
}

// specTeleportVictim teleports an attacker away.
// C: SPECIAL(teleport_victim) is always called with ch==the mob itself
// (mobile_activity.c / fight.c call func(ch, ch, 0, "")); ch is nil in the
// Go autonomous path, so the mob's own state and identity come from me.
func specTeleportVictim(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || me == nil || cmd != "" || me.GetFightingBody() == nil || me.GetPosition() <= combat.PosSleeping {
		return false
	}

	// mobile_activity() and perform_violence() pass the actual FIGHTING(ch)
	// pointer to SPECIAL(), not merely a player-name lookup. Resolve that
	// combatant before emitting any branch output so mob victims take the same
	// call path as players (src/mobact.c:68-93; src/fight.c:2030-2031).
	fighting := mobFightingTarget(w, me)
	if fighting == nil {
		return false
	}

	// C do_action("scoff", GET_NAME(FIGHTING(ch))) reaches the no-arg social
	// arm because scoff has no char_found message. Its room message is exactly
	// the social's others_no_arg string; the typed target is ignored.
	Act(w, false, me, nil, nil, nil, "$n scoffs at the idea.", "", ToRoom)
	if me.CanSpeak() {
		Act(w, true, me, nil, nil, nil,
			"$n says, 'You can't harm me, mortal. Begone.'", "", ToRoom)
	}

	// SPECIAL calls call_magic() directly. Preserve that native entry point's
	// position gate rather than routing through command casting.
	spells.CastFromSpecial(me, fighting, spells.SpellTeleport, me.GetLevel(), w)
	return true
}

// specConSeller sells constitution points.
func specConSeller(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me == nil || cmd == "" || ch.GetFightingBody() != nil || ch.GetPosition() <= combat.PosSleeping {
		return false
	}

	if cmd != "list" && cmd != "buy" {
		return false
	}
	// C's skip_spaces() removes only leading whitespace. The subsequent
	// strcasecmp() is exact, so a trailing space is not accepted as "con".
	arg = strings.TrimLeftFunc(arg, unicode.IsSpace)
	if !conSellerCanSee(w, me, ch) {
		conSellerRoomAct(w, me, nil, true, func(to Actor) string {
			return fmt.Sprintf("%s exclaims, 'Who's there? I can't see you!'", cap(conSellerPersName(w, me, to)))
		})
		return true
	}

	// Cost per con point: GET_LEVEL(ch) * 400
	cost := ch.GetLevel() * 400

	// Available con points the player can buy.
	// C: GET_ORIG_CON(ch) - ch->real_abils.con.
	availCon := ch.GetOrigCon() - ch.Stats.Con
	if availCon < 0 {
		availCon = 0
	}

	if cmd == "list" {
		if availCon < 1 {
			msg := "You seem perfectly healthy!"
			conSellerTell(w, me, ch, msg)
			return true
		}
		suf := "s"
		if availCon == 1 {
			suf = ""
		}
		msg := fmt.Sprintf("You can buy up to %d point%s, at %d per point.", availCon, suf, cost)
		conSellerTell(w, me, ch, msg)
		return true
	}

	// cmd == "buy"
	if !strings.EqualFold(arg, "con") {
		msg := "BUY CON, if you really want to do it."
		conSellerTell(w, me, ch, msg)
		return true
	}
	if ch.GetGold() < cost {
		msg := "You can't afford it!"
		conSellerTell(w, me, ch, msg)
		return true
	}
	if availCon < 1 {
		msg := "You seem perfectly healthy!"
		conSellerTell(w, me, ch, msg)
		return true
	}

	// C deducts gold, tells the buyer, then sends two TO_NOTVICT room acts.
	ch.SetGold(ch.GetGold() - cost)

	msg := fmt.Sprintf("That'll be %d coins, you should feel much better.. if you wake up.", cost)
	conSellerTell(w, me, ch, msg)
	conSellerRoomAct(w, me, ch, false, func(to Actor) string {
		return cap(fmt.Sprintf("%s stares at %s and mutters some arcane words.",
			conSellerPersName(w, me, to), conSellerPersName(w, ch, to)))
	})
	conSellerRoomAct(w, me, ch, false, func(to Actor) string {
		return cap(fmt.Sprintf("%s falls, stunned.", conSellerPersName(w, ch, to)))
	})

	if ch.Stats.Con < 18 {
		ch.Stats.Con++
	}
	ch.AffectTotal() // src/spec_procs3.c:308
	ch.SetPosition(combat.PosStunned)

	return true
}

// conSellerCanSee mirrors CAN_SEE for this procedure's world-aware path. The
// shared act() engine has no room argument in its PERS substitution, while C's
// CAN_SEE also includes LIGHT_OK; the seller is deliberately in a dark shop.
func conSellerCanSee(w *World, observer, subject Actor) bool {
	if observer == nil || subject == nil {
		return false
	}
	return canSeeForPers(observer, subject)
}

// conSellerPersName is PERS with the seller's room-aware LIGHT_OK gate.
func conSellerPersName(w *World, subject, observer Actor) string {
	if !conSellerCanSee(w, observer, subject) {
		return "someone"
	}
	return subject.GetName()
}

// conSellerTell is C's do_tell(mobile, msg, find_command("tell"), 0)
// (src/spec_procs3.c:116, :121, :126, :131, :138), where msg is
// "<player name> <text>". The line is rendered with conSellerPersName rather
// than act()'s PERS: the shared act engine has no room-light argument, and C's
// CAN_SEE includes LIGHT_OK for this deliberately dark shop. perform_tell's
// reply-target assignment is kept, so reply behaves as C's does
// (src/act.comm.c:894).
func conSellerTell(w *World, me *MobInstance, target *Player, msg string) {
	target.SendMessage(fmt.Sprintf("%s tells you, '%s'\r\n", cap(conSellerPersName(w, me, target)), msg))
	target.SetLastTeller(me.GetShortDesc())
}

// conSellerRoomAct emits the two fixed TO_NOTVICT/TO_ROOM acts without using
// roomMessage, preserving C's actor/victim exclusion and PERS substitutions.
func conSellerRoomAct(w *World, me *MobInstance, victim *Player, hideInvisible bool, render func(Actor) string) {
	for _, to := range w.GetPlayersInRoom(me.GetRoomVNum()) {
		if victim != nil && to == victim {
			continue
		}
		if hideInvisible && !conSellerCanSee(w, to, me) {
			continue
		}
		to.SendMessage(render(to) + "\r\n")
	}
}

// specTroll regenerates health over time.
var trollNumber = dprng.Number

func specTroll(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() <= combat.PosSleeping || me.GetHP() <= 0 {
		return false
	}
	if me.GetFightingBody() == nil && me.GetHP() != me.GetMaxHP() {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if trollNumber(0, 20) == 0 {
			regenRate := 2
			newHP := me.GetHP() + me.GetLevel()*regenRate
			if newHP > me.GetMaxHP() {
				newHP = me.GetMaxHP()
			}
			me.SetHealth(newHP)
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s's wounds glow brightly for a moment, then disappear!", mobName(me)))
		}
	} else if me.GetFightingBody() != nil {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if trollNumber(0, 10) == 0 {
			regenRate := 2
			newHP := me.GetHP() + me.GetLevel()*regenRate
			if newHP > me.GetMaxHP() {
				newHP = me.GetMaxHP()
			}
			me.SetHealth(newHP)
			w.roomMessage(me.GetRoomVNum(), fmt.Sprintf("%s's wounds glow brightly for a moment, then disappear!", mobName(me)))
		}
	} else {
		return false
	}
	return true
}

// specQuanLo comments on flee/retreat commands and responds to look.
// specQuanLo is the player lane of SPECIAL(quan_lo) (src/spec_procs3.c:372-397).
func specQuanLo(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	return specQuanLoBody(w, ch, me, cmd, arg)
}

// isCName matches the exact case-insensitive token semantics of C isname(),
// used by quan_lo rather than the prefix semantics of isname_with_abbrevs().
func isCName(name, namelist string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, keyword := range strings.Fields(namelist) {
		if strings.EqualFold(name, keyword) {
			return true
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

// werewolfNumber is the werewolf special's RNG seam. Production keeps the
// shared deterministic game stream; focused tests pin C's two conditional
// draws without perturbing unrelated combat cases.
var werewolfNumber = dprng.Number

// mobSendToZone sends a native NPC message through C send_to_zone's exact
// audience: awake players in the mob's zone, excluding the mob's own room.
// Keep this separate from the broader World.SendToZone helper because the C
// special's exclusion and awake gate are player-visible behavior.
func (w *World) mobSendToZone(me *MobInstance, message string) {
	if me == nil {
		return
	}
	w.SendToZone(me.GetRoomVNum(), message)
}

// specWerewolf howls and bites when fighting.
// C source: SPECIAL(werewolf) src/spec_procs3.c:427-448
func specWerewolf(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil || cmd != "" || me.GetFightingBody() == nil || me.GetHP() <= 0 {
		return false
	}
	vict := mobFightingTarget(w, me)
	if vict == nil {
		return true
	}
	// Howl (10% chance)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	if werewolfNumber(0, 9) == 0 {
		Act(w, true, me, nil, nil, nil, "$n looks up and lets out a long, fierce howl.", "", ToRoom)
		w.mobSendToZone(me, "You hear a loud howling in the distance.")
	}
	// Bite attack (25% chance)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	if werewolfNumber(0, 3) == 0 && vict.GetRoom() == me.GetRoomVNum() && me.GetHP() > 0 {
		Act(w, true, me, vict, nil, nil, "$n tears into your leg with $s huge fangs!", "", ToVict)
		Act(w, true, me, vict, nil, nil, "$n rips apart $N's leg with $s fangs!", "", ToNotVict)
		combat.TakeDamage(me, vict, combat.RollDice(me.GetLevel(), 2), combat.TYPE_BITE)
		if player, ok := vict.(*Player); ok {
			moveReduction := me.GetLevel() * 3 / 2
			newMove := player.GetMove() - moveReduction
			if newMove < 0 {
				newMove = 0
			}
			player.SetMove(newMove)
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
				vict.TakeDamage(dam)
				vict.SendMessage("An incredible force hits you!\r\n")
				// Wounded band / POS_DEAD from the new HP; only kill at POS_DEAD
				// (HP <= -11) — fight.c update_pos (DP-1021).
				if combat.UpdatePositionAfterDamage(vict, w.woundBroadcast) == combat.PosDead {
					w.roomMessage(roomVNum, fmt.Sprintf("%s falls to the ground, screaming in agony!", vict.GetName()))
					w.rawKill(vict, 0)
				}
				damaged = true
			}
		}
		if def.FoType == "affect" {
			// Cast poison on room occupants (affect=spell, level=cast level)
			spells.Cast(vict, vict, spells.SpellPoison, me.GetLevel(), w)
			damaged = true
		}
	}
	return damaged
}

// specPortalToTemple teleports the player to the temple (room 8008).
func specPortalToTemple(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || (cmd != "say" && cmd != "'") {
		return false
	}
	// C calls skip_spaces(), which removes leading whitespace but leaves
	// trailing bytes for the exact strcasecmp gate.
	arg = skipSpaces(arg)
	if !strings.EqualFold(arg, "setchswayno") {
		return false
	}
	w.DoSay(ch, arg)
	sendToChar(ch, "With a blinding flash of light and a crack of thunder, you are teleported...")
	Act(w, true, ch, nil, nil, nil, "With a blinding flash of light and a crack of thunder, $n disappears!", "", ToRoom)
	ch.SetRoom(8008)
	Act(w, true, ch, nil, nil, nil, "With a blinding flash of light and a crack of thunder, $n appears!", "", ToRoom)
	w.lookAtRoom(ch, false)
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
	ch.SendMessage("\r\nWith a blinding flash of light and a crack of thunder, you are teleported...\r\n")
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
			ch2.SendMessage("You feel pulled in a hundred different directions!\r\n")
			w.roomMessage(ch2.GetRoomVNum(), fmt.Sprintf("%s appears in a brilliant flash!", ch2.GetName()))
		}
		// Remove old object, spawn replacement (14503) in the same room
		w.RemoveItemFromRoomByVNum(me.GetVNum(), objRoom)
		if _, err := w.SpawnObject(14503, objRoom); err != nil {
			slog.Warn("SpawnObject failed in teleport spec", "vnum", 14503, "room", objRoom, "error", err)
		}
		return true
	}
	if cmd == "look" {
		sendToChar(ch, "You feel pulled in a hundred different directions!")
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s disappears in a brilliant flash!", ch.GetName()))
		if ch2 != nil {
			// Move ch2 to obj's room
			ch2.SetRoom(objRoom)
			ch2.SendMessage("You feel pulled in a hundred different directions!\r\n")
			w.roomMessage(ch2.GetRoomVNum(), fmt.Sprintf("%s appears in a brilliant flash!", ch2.GetName()))
		}
		// Move ch to room 14496
		ch.SetRoom(14496)
		return true
	}
	return false
}

// prostituteCanSee mirrors CAN_SEE(mobile, ch) for SPECIAL(prostitute).
// C's LIGHT_OK gate is part of this procedure's visibility test; the generic
// Go canSee helper has no World argument and therefore cannot inspect room
// darkness here. Source: src/utils.h:515-530.
func prostituteCanSee(w *World, mobile *MobInstance, ch *Player) bool {
	if w == nil || mobile == nil || ch == nil {
		return false
	}
	if mobile.IsAffected(affBlind) {
		return false
	}
	if w.isRoomDark(mobile.GetRoomVNum()) && !mobile.IsAffected(affInfravision) {
		return false
	}
	if ch.IsAffected(affInvisible) && !mobile.IsAffected(affDetectInvisible) {
		return false
	}
	return true
}

// specProstitute offers services for gold.
// C source: src/spec_procs3.c:670-705. The native procedure only handles
// exact BUY/LIST commands after its entry gates; unrelated commands fall
// through to the ordinary interpreter.
func specProstitute(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil || cmd == "" || ch.GetFightingBody() != nil || ch.GetPosition() <= combat.PosSleeping {
		return false
	}

	if (cmd == "buy" || cmd == "list") && !prostituteCanSee(w, me, ch) {
		Act(w, true, me, nil, nil, nil, "$n says, 'If I could see you, we could do business..'", "", ToRoom)
		Act(w, true, me, nil, nil, nil, "$n winks coyly.", "", ToRoom)
		return true
	}

	switch cmd {
	case "buy":
		tellFromMob(me, ch, "I ain't for sale, just rent. Give me 5 gold for a good time.")
		return true
	case "list":
		tellFromMob(me, ch, "For five coins, I'll show you a good time.")
		return true
	default:
		return false
	}
}

// roachNumber is the C number() seam for the roach's pulse-local draws.
var roachNumber = dprng.Number

// specRoach — a living cockroach that eats, grows, and reproduces.
// C source: SPECIAL(roach) ~line 707. Autonomous C dispatch passes the mob as
// both ch and me; the Go adapter has no player for that call, so this proc uses
// me for the mob state and room acts.
func specRoach(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me == nil || me.GetPosition() <= combat.PosSleeping {
		return false
	}
	roomVNum := me.GetRoomVNum()

	// Starvation death (extremely rare: 1/10001 * 1/10001 probability)
	// #nosec G404 — game RNG, not cryptographic
	//nolint:gocritic,staticcheck // badCond/SA4000: two independent RNG rolls are intentional, not a copy-paste error
	if roachNumber(0, 10000) == 0 && roachNumber(0, 10000) == 0 && me.GetMaxHealth() < 11 {
		Act(w, false, me, nil, nil, nil, "$n seems to starve to death and simply fades out of existence.", "", ToRoom)
		me.SetAlive(false)
		if w.spawner != nil {
			w.spawner.RemoveMobInstance(me.VNum, me)
		}
		w.ExtractMob(me)
		return true
	}

	// Look for food on the ground
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		if obj == nil || !obj.IsTakeable() {
			continue
		}
		Act(w, false, me, nil, obj, nil, "$n feeds on $p.", "", ToRoom)
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if roachNumber(0, 2) == 0 {
			newMaxHP := me.GetMaxHealth() + obj.GetCost()/2
			roll := me.GetDamageRoll()
			// #nosec G404 — game RNG, not cryptographic
			if roachNumber(0, 2) == 0 {
				roll.Num++
			}
			// #nosec G404 — game RNG, not cryptographic
			if roachNumber(0, 2) == 0 {
				roll.Sides++
			}
			me.SetDamageDice(roll.Num, roll.Sides)
			if newMaxHP > 400 {
				// C resets the original roach's max/current HP and damage dice,
				// then creates a fresh prototype roach. The final C assignments
				// repeat ch rather than mob, so the new roach keeps its defaults.
				me.SetMaxHP(10)
				me.SetHealth(10)
				me.SetDamageDice(2, 4)
				newRoach, err := w.spawnMobQuiet(23, roomVNum)
				if err == nil && newRoach != nil {
					// read_mobile + char_to_room is silent in this C path.
					newRoach.SetHealth(10)
					newRoach.SetMaxHP(10)
				}
				Act(w, false, me, nil, nil, nil, "$n splits in half forming a new roach!", "", ToRoom)
			} else {
				me.SetMaxHP(newMaxHP)
				// #nosec G404 — game RNG, not cryptographic
				// #nosec G404
				if roachNumber(0, 1) == 0 {
					Act(w, false, me, nil, nil, nil, "You hear some stretching noises.", "", ToRoom)
				} else {
					Act(w, false, me, nil, nil, nil, "You hear a strange rumbling from $n's stonach.", "", ToRoom)
				}
			}
		} else {
			Act(w, false, me, nil, nil, nil, "You hear $n burp.", "", ToRoom)
		}
		if err := w.MoveObjectToNowhere(obj); err != nil {
			slog.Warn("MoveObjectToNowhere failed in burp spec", "obj", obj.GetVNum(), "error", err)
		}
		return true
	}

	// Random idle behaviors
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	switch roachNumber(0, 10) {
	case 0:
		Act(w, false, me, nil, nil, nil, "$n chirps gleefully.", "", ToRoom)
	case 1:
		Act(w, false, me, nil, nil, nil, "$n changes colors and clicks happily.", "", ToRoom)
	case 2:
		Act(w, false, me, nil, nil, nil, "$n skitters around in tight circles.", "", ToRoom)
	case 3:
		Act(w, false, me, nil, nil, nil, "Strange purple dots appear on $n's back.", "", ToRoom)
	case 4:
		// C only attempts the teleport one time in six after choosing case 4.
		if roachNumber(0, 5) != 0 {
			return false
		}
		// Teleport to a random room
		rooms := w.Rooms()
		if len(rooms) > 0 {
			// #nosec G404 — game RNG, not cryptographic
			// #nosec G404
			randRoom := rooms[roachNumber(0, len(rooms)-1)].VNum
			// Check for unwanted room flags (private/godroom/nomagic/death)
			if w.roomHasFlag(randRoom, "private") || w.roomHasFlag(randRoom, "godroom") ||
				w.roomHasFlag(randRoom, "nomagic") || w.roomHasFlag(randRoom, "death") {
				Act(w, false, me, nil, nil, nil, "$n fades out and back in again.", "", ToRoom)
				return false
			}
			Act(w, true, me, nil, nil, nil, "$n fades out slowly with a soft swoosh.", "", ToRoom)
			me.SetRoom(randRoom)
			Act(w, true, me, nil, nil, nil, "$n fades in slowly, looking a bit disoriented.", "", ToRoom)
			return true
		}
		return false
	}
	return false
}

// specMortician retrieves corpses for a fee.
// C source: SPECIAL(mortician) ~line 807.
func specMortician(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil || cmd == "" {
		return false
	}
	cost := ch.GetLevel() * 116
	if cmd == "list" {
		tellFromMob(me, ch, fmt.Sprintf("It will cost %d coins to retrieve your corpse.", cost))
		return true
	}
	if cmd == "retrieve" {
		if ch.GetGold() < cost {
			tellFromMob(me, ch, "I'm sorry, you can't afford the cost.")
			return true
		}
		// Search all rooms for a corpse matching this player
		objects := w.GetAllObjects()
		// C walks object_list, which is newest-first because create_obj and
		// read_object prepend. World object IDs are monotonic, so descending
		// IDs preserve that order in the Go object registry.
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].ID > objects[j].ID
		})
		for _, obj := range objects {
			if !isnameWithAbbrevs(ch.GetName(), obj.GetKeywords()) ||
				obj.GetValue(3) == 0 || obj.GetTypeFlag() != ITEM_CONTAINER {
				continue
			}
			// C obj_from_room + obj_to_room moves the matching object to the
			// player's room and prepends it to that room's contents.
			if err := w.MoveObjectToRoomFront(obj, ch.GetRoomVNum()); err != nil {
				slog.Warn("MoveObjectToRoomFront failed in mortician", "corpse", obj.GetVNum(), "room", ch.GetRoomVNum(), "error", err)
			}
			Act(w, false, ch, nil, nil, nil, "The Mortician dumps your corpse on the ground.", "", ToChar)
			Act(w, true, ch, nil, nil, nil, "The Mortician dumps $n's corpse on the ground.", "", ToRoom)
			ch.SetGold(ch.GetGold() - cost)
			return true
		}
		tellFromMob(me, ch, "I'm sorry, I can't find your corpse anywhere!")
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
	// C checks the live AFF_CHARM bit, not the prototype's innate flags.
	if me.IsAffected(affCharm) {
		return false
	}
	switch me.GetVNum() {
	case 81, 82, 83, 84:
		// C's send_to_char() reaches only a live player master. Mob masters
		// have no descriptor in the C path, so followingActor() is intentionally
		// narrowed to *Player here.
		if leaderName := me.GetFollowing(); leaderName != "" {
			if leader, ok := w.followingActor(leaderName).(*Player); ok {
				leader.SendMessage(fmt.Sprintf("You lose control and %s fizzles away!\r\n", me.GetName()))
			}
		}
		Act(w, true, me, nil, nil, nil, "$n returns to its own plane of existence.", "", ToRoom)
	default:
		// do_say() selects "states" for a period-terminated utterance.
		Act(w, false, me, nil, nil, nil, "$n states, 'My work here is done.'", "", ToRoom)
		// C sprintf's the vanish line with its own \r\n and act() appends
		// the second (spec_procs3.c:878-882) — the audience's blank line.
		Act(w, false, me, nil, nil, nil, fmt.Sprintf("%s disappears in a flash of white light!\r\n", me.GetName()), "", ToRoom)
	}
	// extract_char() queues the mob for heartbeat removal without running the
	// player death pipeline: conjured creatures leave no corpse or death line.
	w.ExtractMob(me)
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
		// C calls do_tell(mobile, "<player> Why ...") and the target parser
		// strips the player's name before perform_tell sends the direct
		// $n-tells-you line (spec_procs3.c:913-918, act.comm.c:905-930).
		Act(nil, false, me, ch, nil, nil, "$n tells you, 'Why don't you sign up for training?  Just head south through those doors!'", "", ToVict|ToSleep)
		return true
	}
	if cmd == "cast" || cmd == "will" {
		// Same direct do_tell path; only the C message body changes
		// (spec_procs3.c:921-926).
		Act(nil, false, me, ch, nil, nil, "$n tells you, 'Hey now! None of that voodoo mumbo jumbo in my office!'", "", ToVict|ToSleep)
		return true
	}
	return false
}

// specElementsMasterColumn is the player lane of SPECIAL(elements_master_column);
// the shared body walks every occupant (src/spec_procs3.c:936-1002).
func specElementsMasterColumn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil {
		return false
	}
	specElementsMasterColumnMoves(w, ch.GetRoomVNum())
	return true
}

// specElementsPlatforms is the player lane of SPECIAL(elements_platforms); the
// shared body walks every occupant (src/spec_procs3.c:1004-1024).
func specElementsPlatforms(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil {
		return false
	}
	specElementsPlatformsMove(w, ch.GetRoomVNum())
	return true
}

// specElementsLoadCylinders manages cylinder objects for the talisman puzzle.
func specElementsLoadCylinders(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil {
		return false
	}
	if cmd == "get" {
		w.doGet(ch, me, cmd, arg)
		elementsRemoveCylinders(w, ch.GetRoomVNum())
		return true
	}
	if cmd != "drop" {
		return false
	}

	// Map room vnum to expected talisman vnum and cylinder vnum
	type loadEntry struct {
		roomVNum int
		talVNum  int
		cylVNum  int
		color    string
	}
	entries := map[int]loadEntry{
		1360: {1360, 1300, 1304, "green"},
		1364: {1364, 1301, 1305, "yellow"},
		1380: {1380, 1302, 1306, "red"},
		1384: {1384, 1303, 1307, "blue"},
	}

	// C checks for any cylinder before running do_drop. Returning false lets
	// the ordinary command path handle the drop when one is already present.
	for _, item := range w.GetItemsInRoom(ch.GetRoomVNum()) {
		if item == nil {
			continue
		}
		if item.GetVNum() == 1304 || item.GetVNum() == 1305 || item.GetVNum() == 1306 || item.GetVNum() == 1307 {
			return false
		}
	}

	// Perform the actual drop
	w.doDrop(ch, me, cmd, arg)

	arg1, _ := oneArgument(arg)
	if arg1 == "" {
		return true
	}
	var dropped *ObjectInstance
	for _, item := range w.GetItemsInRoom(ch.GetRoomVNum()) {
		if item != nil && canSeeObject(ch, item) && isnameWithAbbrevs(arg1, item.GetKeywords()) {
			dropped = item
			break
		}
	}
	if dropped == nil {
		return true
	}

	entry, ok := entries[ch.GetRoomVNum()]
	if !ok || dropped.GetVNum() != entry.talVNum {
		return true
	}

	msg := fmt.Sprintf("A %s cylinder of light extends upwards from the pillar.", entry.color)
	w.roomMessage(ch.GetRoomVNum(), msg)
	obj, err := w.SpawnObject(entry.cylVNum, ch.GetRoomVNum())
	if err == nil {
		if err2 := w.MoveObjectToRoomFront(obj, ch.GetRoomVNum()); err2 != nil {
			slog.Warn("MoveObjectToRoomFront failed in talisman spec", "obj", obj.GetVNum(), "room", ch.GetRoomVNum(), "error", err2)
		}
	}

	return true
}

// specElementsGaleruColumn is the player lane of SPECIAL(elements_galeru_column)
// (src/spec_procs3.c:1137-1182).
func specElementsGaleruColumn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil {
		return false
	}
	return specElementsGaleruColumnMoves(w, ch.GetRoomVNum())
}

// specElementsGaleruAlive is the player lane of SPECIAL(elements_galeru_alive)
// (src/spec_procs3.c:1184-1215).
func specElementsGaleruAlive(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || w == nil || ch == nil {
		return false
	}
	return specElementsGaleruAliveMoves(w, ch.GetRoomVNum())
}

// specElementsMinion is the player lane of SPECIAL(elements_minion)
// (src/spec_procs3.c:1217-1240).
func specElementsMinion(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	specElementsMinionBody(w, me)
	return false
}

// elementsGuardianSelfDamage preserves damage(ppl, ppl, dam, TYPE_UNDEFINED):
// in particular, the player is the killer/source and self-damage does not
// enroll a fighting target. The shared damage path owns state/death bytes.
func elementsGuardianSelfDamage(w *World, target *Player, dam int) {
	combat.TakeDamageWithDeath(target, target, dam, combat.TYPE_UNDEFINED, func() {
		w.HandleDeath(target, target, combat.TYPE_UNDEFINED)
	})
}

// elementsGuardianPlayerHit preserves the C hit(ppl, next, TYPE_UNDEFINED)
// call after its three Act writes (spec_procs3.c:1281): unlike a
// mobile-special hit, both combatants here are players, but the swing still
// consumes hit()'s draws and runs damage()'s gates against the pre-enrolment
// state before either side enrols (fight.c:1318-1445).
func elementsGuardianPlayerHit(w *World, attacker, defender *Player) {
	if w.combatEngine == nil {
		return
	}
	if hitter, ok := w.combatEngine.(interface {
		PerformDeferredInitialAttack(combat.Combatant, combat.Combatant) error
	}); ok {
		if err := hitter.PerformDeferredInitialAttack(attacker, defender); err != nil {
			slog.Warn("elements guardian deferred hit failed", "attacker", attacker.GetName(), "defender", defender.GetName(), "error", err)
		}
		return
	}
	if err := w.combatEngine.StartCombat(attacker, defender); err != nil {
		slog.Warn("elements guardian combat start failed", "attacker", attacker.GetName(), "defender", defender.GetName(), "error", err)
		return
	}
	initial, ok := w.combatEngine.(interface {
		PerformInitialAttack(combat.Combatant, combat.Combatant) error
	})
	if !ok {
		return
	}
	if err := initial.PerformInitialAttack(attacker, defender); err != nil {
		slog.Warn("elements guardian initial attack failed", "attacker", attacker.GetName(), "defender", defender.GetName(), "error", err)
	}
}

// specElementsGuardian is the player lane of SPECIAL(elements_guardian)
// (src/spec_procs3.c:1242-1289).
func specElementsGuardian(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me == nil || cmd == "" {
		return false
	}
	specElementsGuardianBody(w, me, ch.GetRoomVNum())
	return false
}

// specFlyExitUp blocks going up unless the player can fly.
func specFlyExitUp(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// C: spec_procs3.c SPECIAL(fly_exit_up). The Go room-special interface
	// receives players only, so C's IS_NPC arm is enforced by that boundary.
	if w == nil || ch == nil || ch.GetLevel() > LVL_IMMORT || cmd != "up" {
		return false
	}
	if ch.IsAffected(affFly) {
		return false // player can fly, allow passage
	}
	sendToChar(ch, "You try and jump up there but it's just too high.")
	Act(w, true, ch, nil, nil, nil,
		"$n jumps up and down in a vain attempt to travel upwards.", "", ToNotVict)
	return true
}

// elementsRemoveCylinders checks the current room and removes cylinders when
// their corresponding talisman leaves. C returns immediately if it sees the
// first talisman in its four-pass scan, so preserve that odd early exit.
func elementsRemoveCylinders(w *World, roomVNum int) {
	talismanVnums := []int{1300, 1301, 1302, 1303}
	cylinderVnums := []int{1304, 1305, 1306, 1307}
	colors := []string{"green", "yellow", "red", "blue"}

	for i := range talismanVnums {
		var cylinder *ObjectInstance
		for _, item := range w.GetItemsInRoom(roomVNum) {
			if item == nil {
				continue
			}
			switch item.GetVNum() {
			case talismanVnums[i]:
				return
			case cylinderVnums[i]:
				cylinder = item
			}
		}
		if cylinder == nil {
			continue
		}
		w.roomMessage(roomVNum, fmt.Sprintf("The %s cylinder of light slowly sinks back into the pillar.", colors[i]))
		w.ExtractObject(cylinder, roomVNum)
	}
}
