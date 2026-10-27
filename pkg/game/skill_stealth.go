package game

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	skillNumSneak   = 138
	skillNumSteal   = 139
	skillNumStealth = 153
)

// DoSneak implements do_sneak() from src/act.other.c:214-244.
func DoSneak(ch *Player) SkillResult {
	return doSneak(ch, SkillSneak, skillNumSneak)
}

// doSneak is the shared do_sneak/do_stealth flow. The C handlers differ only
// in the skill used for the probability check and the spell used for the
// installed affect; gate order, draw count, and messages are identical.
func doSneak(ch *Player, skill string, spell int) SkillResult {
	// IS_MOUNTED gate — act.other.c:219-223, before any roll.
	if isMounted(ch) {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	message := "Okay, you'll try to move silently for a while."

	// C removes both the ordinary sneak and ninja stealth affects, clears the
	// bit, and then rerolls. Reissuing sneak is not a toggle-off operation.
	if ch.IsAffected(affSneak) {
		ch.RemoveAffectBySpell(skillNumSneak)
		ch.RemoveAffectBySpell(skillNumStealth)
		ch.SetAffect(affSneak, false)
	}

	// percent = number(1,101); 101 is a complete failure.
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(skill) + dexAppSkill(ch.GetDex()).Sneak
	if percent > prob {
		return SkillResult{Success: false, MessageToCh: message}
	}

	ch.AddAffect(engine.NewAffectDirect(
		spell,
		engine.ApplyNone,
		ch.GetLevel(),
		0,
		engine.AFFSneak,
		skill,
	))
	return SkillResult{Success: true, MessageToCh: message}
}

// DoStealth implements do_stealth() from src/act.other.c. It is do_sneak with
// SKILL_STEALTH driving the probability and the applied affect (same message,
// same AFF_SNEAK bit, same single number(1,101) draw). The former Go stealth
// handler was invented ("become one with the shadows" + a skill gate C has not).
func DoStealth(ch *Player) SkillResult {
	return doSneak(ch, SkillStealth, skillNumStealth)
}

// DoHideInWorld applies the live command's room/weather gates before running
// the ordinary hide roll. C's do_hide reads the global sunlight and the
// actor's current room sector, so this must be called by the session command
// path with its authoritative world.
// Class (d) test seam, kept on purpose: the stealth skill-mechanics tests
// (dex bonus, toggle, daytime gates) drive it; the live command path is
// DoHideInWorld via command.CmdHide. deadcode -test with cmd/tools roots
// does not count this package's own tests, so it is falsely listed.
// DoHide implements the newbie path through do_hide() from
// src/act.other.c:247-306 (subcmd == 0). The world-aware command entry point
// is DoHideInWorld; this compatibility wrapper keeps direct game-layer tests
// independent of a world fixture.
func DoHide(ch *Player) SkillResult {
	return doHide(ch, nil, false)
}

func DoHideInWorld(ch *Player, world *World) SkillResult {
	return doHide(ch, world, false)
}

func doHide(ch *Player, world *World, kabuki bool) SkillResult {
	// IS_MOUNTED gate — act.other.c:251-255, before the sector/weather gate and
	// any roll.
	if isMounted(ch) {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}
	if message := hideDaytimeSectorMessage(world, ch.GetRoom()); message != "" {
		return SkillResult{Success: false, MessageToCh: message}
	}

	message := "You attempt to hide yourself."
	skill := SkillHide
	if kabuki {
		message = "You attempt to practice the art of kabuki."
		skill = SkillKabuki
	}

	// C clears AFF_HIDE and immediately rerolls; it does not toggle out with a
	// separate message or return early.
	if ch.IsAffected(affHide) {
		ch.SetAffect(affHide, false)
	}

	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(skill) + dexAppSkill(ch.GetDex()).Hide
	if percent > prob {
		return SkillResult{Success: false, MessageToCh: message}
	}

	ch.SetAffect(affHide, true)
	return SkillResult{Success: true, MessageToCh: appendImprovementMessage(message, improveSkillMessage(ch, skill))}
}

// hideDaytimeSectorMessage ports the sector switch in do_hide() from
// src/act.other.c:257-282. C checks sunlight before the switch and does not
// consult ROOM_INDOORS; the room's sector value is the complete gate input.
func hideDaytimeSectorMessage(world *World, roomVNum int) string {
	if world == nil || WeatherSnapshot().Sunlight == SunDark {
		return ""
	}
	room := world.GetRoomInWorld(roomVNum)
	if room == nil {
		return ""
	}
	switch room.Sector {
	case SECT_FIELD:
		return "Hide out here during the day? Yeah right."
	case SECT_DESERT:
		return "You can't hide very well with all the sun and sand out here!"
	case SECT_WATER_SWIM, SECT_WATER_NOSWIM, SECT_UNDERWATER, SECT_WATER:
		return "Hide in the water? Don't think so."
	case SECT_FLYING, SECT_FIRE, SECT_EARTH, SECT_WIND:
		return "You are completely exposed here, nowhere to hide!"
	default:
		return ""
	}
}

// DoKabukiInWorld applies the shared do_hide room/weather gates for the
// SCMD_KABUKI command variant.
func DoKabukiInWorld(ch *Player, world *World) SkillResult {
	return doHide(ch, world, true)
}

// DoSteal implements the ordinary (subcmd == 0) path through do_steal() from
// src/act.other.c:309-531. Draw order follows C: the initial percent roll is
// made before object lookup and the coin amount is rolled only after success.
func DoSteal(ch *Player, target combat.Combatant, itemName string, world *World) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Steal what from who?"}
	}
	if target == ch {
		return SkillResult{Success: false, MessageToCh: "Come on now, that's rather stupid!"}
	}
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "You can't contemplate stealing in such a place!"}
	}
	if isMounted(ch) {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// Player stealing is reserved for outlaws in the ordinary command path.
	if player, ok := target.(*Player); ok && ch.GetFlags()&(1<<uint(PlrOutlaw)) == 0 {
		return SkillResult{
			Success:       false,
			MessageToCh:   fmt.Sprintf("You can not steal from %s because you are not an Outlaw!", player.Name),
			MessageToVict: fmt.Sprintf("%s failed to steal from you because %s is not an Outlaw!", ch.Name, ch.Name),
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101) - dexAppSkill(ch.GetDex()).PPocket

	if player, ok := target.(*Player); ok && player.GetPosition() <= combat.PosSleeping && player.GetLevel() < LVL_IMMORT {
		percent = -1
	}
	if player, ok := target.(*Player); ok && ch.GetLevel() < LVL_IMMORT &&
		(player.GetLevel() > ch.GetLevel()+3 || player.GetLevel() < ch.GetLevel()-3) {
		percent = 101
	}
	if target.GetLevel() >= LVL_IMMORT || isShopKeeper(target) || combatantIsAffected(target, affRobbed) {
		percent = 101
	}
	if ch.GetLevel() > LVL_IMMORT && target.GetLevel() < ch.GetLevel() {
		percent = -1
	}

	if !strings.EqualFold(itemName, "coins") && !strings.EqualFold(itemName, "gold") {
		item, found := findCarriedItem(ch, target, itemName)
		if found {
			return stealInventoryItem(world, ch, target, item, percent, chPronouns, victPronouns)
		}

		item, slot, found := findEquippedItem(ch, target, itemName)
		if !found {
			return SkillResult{
				Success:     false,
				MessageToCh: stealthActMessage("$E hasn't got that item.", chPronouns, &victPronouns, ""),
			}
		}
		return stealEquippedItem(world, ch, target, item, slot, percent, chPronouns, victPronouns)
	}

	return stealCoins(ch, target, percent, chPronouns, victPronouns)
}

func stealInventoryItem(world *World, ch *Player, target combat.Combatant, item *ObjectInstance, percent int, chPronouns, victPronouns Pronouns) SkillResult {
	percent += item.GetTotalWeight()
	if target.GetLevel() > ch.GetLevel() {
		percent += target.GetLevel() - ch.GetLevel()
	}
	if percent > ch.GetSkill(SkillSteal) {
		result := SkillResult{
			Success:       false,
			MessageToCh:   stealthActMessage("$N catches you trying to steal something...", chPronouns, &victPronouns, ""),
			MessageToVict: stealthActMessage("$n tried to steal something from you!", chPronouns, &victPronouns, ""),
			MessageToRoom: stealthActMessage("$n tries to steal something from $N.", chPronouns, &victPronouns, ""),
			WaitCh:        1,
		}
		applyStealFailure(ch, target, &result)
		return result
	}

	if ok, message := canCarryStolenItem(ch, item); !ok {
		return SkillResult{Success: false, MessageToCh: message, WaitCh: 1}
	}
	// transfer performs C's carried steal: obj_from_char(obj, victim)
	// (src/act.other.c:471), which flags the victim (handler.c:596-598), then
	// obj_to_char(obj, ch) (src/act.other.c:472), which flags the thief
	// (handler.c:569-571). C checks the thief's capacity BEFORE obj_from_char
	// (act.other.c:468-470); the port checks it after the removal, so both
	// flags go after the AddItem succeeds — the failure path, like C's
	// capacity refusal, flags nobody.
	//
	// It mutates two inventories and the object's Location. Every other
	// transfer (give/put/drop/corpse) does that under w.mu through MoveObject,
	// so running this sequence unlocked let a concurrent MoveObject detach
	// through the by-then-stale item.Location (a silent no-op — the victim's
	// list no longer held it) and attach the same *ObjectInstance to a second
	// inventory, after which this path added it again: one instance reachable
	// from two Inventory.Items lists, which the file and DB save seams then
	// persist as two independent items (CWE-362/CWE-367).
	transfer := func() bool {
		if !removeCarriedItem(target, item) {
			return false
		}
		if err := ch.Inventory.AddItem(item); err != nil {
			if !restoreCarriedItem(target, item) {
				slog.Error("DoSteal rollback failed", "target", target.GetName(), "item", item.GetShortDesc(), "error", err)
			}
			return false
		}
		if targetPlayer, ok := target.(*Player); ok {
			targetPlayer.MarkCrashNeeded()
		}
		ch.MarkCrashNeeded()
		item.Location = LocInventoryPlayer(ch.Name)
		return true
	}

	// Holding w.mu across the whole sequence makes it atomic against those
	// transfers (lock order w.mu -> Equipment.mu -> Inventory.mu). It is
	// released before the mudlog below, which fans out to session callbacks.
	var transferred bool
	if world != nil {
		world.mu.Lock()
		transferred = transfer()
		world.mu.Unlock()
	} else {
		// World-less seam: no world means no concurrent transfers to
		// serialize against.
		transferred = transfer()
	}
	if !transferred {
		return SkillResult{Success: false, MessageToCh: "You cannot carry that much.", WaitCh: 1}
	}
	if !target.IsNPC() {
		MudLog(fmt.Sprintf("(PS) %s stole %s from %s.", ch.GetName(), item.GetShortDesc(), target.GetName()), MudlogComplete, LVL_IMMORT, true)
	}
	applyRobbedAffect(target)
	message := appendImprovementMessage("Got it!", improveSkillMessage(ch, SkillSteal))
	return SkillResult{Success: true, MessageToCh: message, WaitCh: 1}
}

func stealEquippedItem(world *World, ch *Player, target combat.Combatant, item *ObjectInstance, slot int, percent int, chPronouns, victPronouns Pronouns) SkillResult {
	if target.GetPosition() > combat.PosSleeping {
		return SkillResult{Success: false, MessageToCh: "Steal the equipment now?  Impossible!"}
	}
	if percent > 100 {
		return SkillResult{
			Success:       false,
			MessageToCh:   stealthActMessage("You try to unequip $p, but fail!", chPronouns, &victPronouns, item.GetShortDesc()),
			MessageToRoom: stealthActMessage("$n tries to steal $p from $N, but fails!", chPronouns, &victPronouns, item.GetShortDesc()),
			WaitCh:        1,
		}
	}
	if ok, message := canCarryStolenItem(ch, item); !ok {
		return SkillResult{Success: false, MessageToCh: message, WaitCh: 1}
	}
	// The transfer mutates the victim's equipment, both inventories and the
	// object's Location, so it runs under w.mu as one step — the discipline
	// every other object transfer already uses through MoveObject. finish must
	// run after the lock is released: checkEquipmentStats and
	// finishMobileEquipment both reach back into the World (GetAllMobs,
	// selfDamage, Act) and would deadlock under it.
	var transferred bool
	var finish func()
	if world != nil {
		world.mu.Lock()
		transferred, finish = stealEquippedTransferLocked(world, ch, target, item, slot)
		world.mu.Unlock()
	} else {
		transferred, finish = stealEquippedTransferLocked(world, ch, target, item, slot)
	}
	if finish != nil {
		finish()
	}
	if !transferred {
		return SkillResult{Success: false, MessageToCh: "You cannot carry that much.", WaitCh: 1}
	}
	if !target.IsNPC() {
		MudLog(fmt.Sprintf("(PS) %s stole %s from %s.", ch.GetName(), item.GetShortDesc(), target.GetName()), MudlogComplete, LVL_IMMORT, true)
	}
	applyRobbedAffect(target)
	message := appendImprovementMessage(
		stealthActMessage("You unequip $p and steal it.", chPronouns, &victPronouns, item.GetShortDesc()),
		improveSkillMessage(ch, SkillSteal),
	)
	return SkillResult{
		Success:       true,
		MessageToCh:   message,
		MessageToRoom: stealthActMessage("$n steals $p from $N.", chPronouns, &victPronouns, item.GetShortDesc()),
		WaitCh:        1,
	}
}

func stealCoins(ch *Player, target combat.Combatant, percent int, chPronouns, victPronouns Pronouns) SkillResult {
	if percent > ch.GetSkill(SkillSteal) {
		result := SkillResult{
			Success:       false,
			MessageToCh:   "Oops..",
			MessageToVict: stealthActMessage("You discover that $n has $s hands in your wallet.", chPronouns, &victPronouns, ""),
			MessageToRoom: stealthActMessage("$n tries to steal gold from $N.", chPronouns, &victPronouns, ""),
			WaitCh:        1,
		}
		applyStealFailure(ch, target, &result)
		return result
	}

	// #nosec G404 — game RNG, not cryptographic
	gold := targetGold(target) * dprng.Number(1, 10) / 100
	gold = min(1782, gold)
	if gold > 0 {
		// Take the haul atomically. C sizes it from the victim's balance and
		// then subtracts unconditionally; doing that as a read-then-write pair
		// let two concurrent steals both size from the same balance and both
		// credit a thief, minting the difference (CWE-362/CWE-367).
		if !spendTargetGold(target, gold) {
			return SkillResult{Success: true, MessageToCh: "You couldn't get any gold...", WaitCh: 1}
		}
		ch.AddGold(gold)
		if gold > 1 {
			message := appendImprovementMessage(
				fmt.Sprintf("Bingo!  You got %d gold coins.", gold),
				improveSkillMessage(ch, SkillSteal),
			)
			return SkillResult{Success: true, MessageToCh: message, WaitCh: 1}
		}
		return SkillResult{Success: true, MessageToCh: "You manage to swipe a solitary gold coin.", WaitCh: 1}
	}
	return SkillResult{Success: true, MessageToCh: "You couldn't get any gold...", WaitCh: 1}
}

func findCarriedItem(ch *Player, target combat.Combatant, itemName string) (*ObjectInstance, bool) {
	switch target := target.(type) {
	case *Player:
		for _, item := range target.Inventory.FindItems("") {
			if item != nil && isName(itemName, item.GetKeywords()) && canSeeObject(ch, item) {
				return item, true
			}
		}
	case *MobInstance:
		target.mu.RLock()
		defer target.mu.RUnlock()
		for _, item := range target.Inventory {
			if item != nil && isName(itemName, item.GetKeywords()) && canSeeObject(ch, item) {
				return item, true
			}
		}
	}
	return nil, false
}

func findEquippedItem(ch *Player, target combat.Combatant, itemName string) (*ObjectInstance, int, bool) {
	switch target := target.(type) {
	case *Player:
		for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
			item, ok := target.Equipment.GetItemInSlot(slot)
			if ok && item != nil && isName(itemName, item.GetKeywords()) && canSeeObject(ch, item) {
				return item, int(slot), true
			}
		}
	case *MobInstance:
		target.mu.RLock()
		defer target.mu.RUnlock()
		for slot := 0; slot < NumWears; slot++ {
			item := target.Equipment[slot]
			if item != nil && isName(itemName, item.GetKeywords()) && canSeeObject(ch, item) {
				return item, slot, true
			}
		}
	}
	return nil, 0, false
}

func removeCarriedItem(target combat.Combatant, item *ObjectInstance) bool {
	switch target := target.(type) {
	case *Player:
		return target.Inventory.RemoveItem(item)
	case *MobInstance:
		return target.RemoveFromInventory(item)
	default:
		return false
	}
}

func restoreCarriedItem(target combat.Combatant, item *ObjectInstance) bool {
	switch target := target.(type) {
	case *Player:
		if err := target.Inventory.AddItem(item); err != nil {
			return false
		}
		item.Location = LocInventoryPlayer(target.Name)
	case *MobInstance:
		target.AddToInventory(item)
	default:
		return false
	}
	return true
}

// stealEquippedTransferLocked performs C's equipped steal,
// obj_to_char(unequip_char(vict, eq_pos), ch) (src/act.other.c:429), assuming
// the caller holds w.mu.
//
// It changes the victim's equipment slot, both inventories and the object's
// Location, so it must be atomic against every other object transfer — those
// all run under w.mu through MoveObject. Running it unlocked let a concurrent
// MoveObject detach through the by-then-stale Location (a silent no-op — the
// victim's equipment no longer held it) and attach the same *ObjectInstance to
// a second inventory, after which this path added it again: one instance
// reachable from two lists, which the save seams then persist as two
// independent items (CWE-362/CWE-367).
//
// It returns (transferred, finish); finish must run after w.mu is released.
func stealEquippedTransferLocked(world *World, ch *Player, target combat.Combatant, item *ObjectInstance, slot int) (bool, func()) {
	if !unequipStolenItemLocked(world, target, item, slot) {
		return false, nil
	}
	if err := ch.Inventory.AddItem(item); err != nil {
		restored, finish := restoreEquippedItemLocked(world, target, item, slot)
		if !restored {
			slog.Error("DoSteal equipment rollback failed", "target", target.GetName(), "item", item.GetShortDesc(), "error", err)
		}
		return false, finish
	}
	// C's equipment steal is obj_to_char(unequip_char(vict, eq_pos), ch)
	// (src/act.other.c:429): only the thief is flagged — unequip_char never
	// sets PLR_CRASH.
	ch.MarkCrashNeeded()
	item.Location = LocInventoryPlayer(ch.Name)
	return true, nil
}

// unequipStolenItemLocked takes the item out of the victim's equipment and
// parks it nowhere, assuming the caller holds w.mu.
func unequipStolenItemLocked(world *World, target combat.Combatant, item *ObjectInstance, slot int) bool {
	switch target := target.(type) {
	case *Player:
		// UnequipItem normally places the object in its owner's inventory. Stage
		// it in a temporary inventory so a full victim inventory cannot block the
		// direct equipment-to-thief transfer performed by C's unequip_char().
		staging := NewInventory()
		litLight := isLitLightSource(item)
		room := target.GetRoom()
		if !target.Equipment.UnequipItem(item, staging) || !staging.RemoveItem(item) {
			return false
		}
		// unequip_char drops a lit light from the victim's room
		// (src/act.other.c:429; src/handler.c:779-782). adjustRoomLight is the
		// locked-internal form, so the caller's w.mu is the lock it needs.
		if world != nil && litLight && room >= 0 {
			world.adjustRoomLight(room, -1)
		}
		item.Location = LocNowhere()
		return true
	case *MobInstance:
		// The public UnequipItem routes through MoveObjectToMobInventoryFront,
		// which takes w.mu; re-entering it under the caller's lock would
		// deadlock. unequipMobileLocked is the form that assumes w.mu.
		target.mu.Lock()
		removed := target.unequipMobileLocked(world, slot)
		target.mu.Unlock()
		return removed == item
	default:
		return false
	}
}

// restoreEquippedItemLocked puts the item back in the victim's slot after a
// failed transfer, assuming the caller holds w.mu. It returns
// (restored, finish); finish must run after w.mu is released.
func restoreEquippedItemLocked(world *World, target combat.Combatant, item *ObjectInstance, slot int) (bool, func()) {
	switch target := target.(type) {
	case *Player:
		if err := target.Equipment.SetSlot(EquipmentSlot(slot), item); err != nil {
			return false, nil
		}
		item.Location = LocEquippedPlayer(target.Name, EquipmentSlot(slot))
		// Symmetric rollback of unequipStolenItemLocked's light accounting. C
		// checks the thief's capacity before unequipping (act.other.c:425-428),
		// so it never reaches this intermediate state.
		if world != nil && isLitLightSource(item) {
			if room := target.GetRoom(); room >= 0 {
				world.adjustRoomLight(room, 1)
			}
		}
		// Deferred: checkEquipmentStats reaches applyBadStat, which calls
		// GetAllMobs/selfDamage and would deadlock under w.mu.
		return true, target.checkEquipmentStats
	case *MobInstance:
		// equipMobileLocked assumes w.mu; the Act/self-damage half of equipping
		// runs in finishMobileEquipment, which must follow the release.
		target.mu.Lock()
		effects, err := target.equipMobileLocked(world, item, slot)
		equipped := target.Equipment[slot] == item
		target.mu.Unlock()
		if err != nil || !equipped {
			return false, nil
		}
		return true, func() {
			if world != nil {
				world.finishMobileEquipment(target, item, effects)
			}
		}
	default:
		return false, nil
	}
}

// restoreEquippedItem is the non-locked entry point for callers that do not
// already hold w.mu. The steal path uses restoreEquippedItemLocked directly.
func restoreEquippedItem(world *World, target combat.Combatant, item *ObjectInstance, slot int) bool {
	var restored bool
	var finish func()
	if world != nil {
		world.mu.Lock()
		restored, finish = restoreEquippedItemLocked(world, target, item, slot)
		world.mu.Unlock()
	} else {
		restored, finish = restoreEquippedItemLocked(world, target, item, slot)
	}
	if finish != nil {
		finish()
	}
	return restored
}

func canCarryStolenItem(ch *Player, item *ObjectInstance) (bool, string) {
	// The strict '<' comparisons are intentional and match do_steal rather than
	// the usual CAN_GET_OBJ <= checks. C only prints the carry message when the
	// item-count check fails; its weight-failure path is silent.
	if ch.Inventory.GetItemCount()+1 >= ch.MaxCarryItems() {
		return false, "You cannot carry that much."
	}
	if ch.CarriedWeight()+item.GetTotalWeight() >= ch.MaxCarryWeight() {
		return false, ""
	}
	return true, ""
}

func applyStealFailure(ch *Player, target combat.Combatant, result *SkillResult) {
	if target.IsNPC() {
		if target.GetPosition() > combat.PosSleeping {
			result.StartCombat = true
		}
		return
	}
	ch.SetPlrFlag(PlrOutlaw, true)
	result.StealCaughtPlayer = true
}

func applyRobbedAffect(target combat.Combatant) {
	player, ok := target.(*Player)
	if !ok {
		return
	}
	player.AddAffect(engine.NewAffectDirect(
		skillNumSteal,
		engine.ApplyNone,
		6,
		0,
		engine.AFFRobbed,
		SkillSteal,
	))
}

func combatantIsAffected(target combat.Combatant, bit int) bool {
	switch target := target.(type) {
	case *Player:
		return target.IsAffected(bit)
	case *MobInstance:
		return target.IsAffected(bit)
	default:
		return false
	}
}

func isShopKeeper(target combat.Combatant) bool {
	mob, ok := target.(*MobInstance)
	return ok && mob.Proto() != nil && MobSpecAssign[mob.Proto().VNum] == "shop_keeper"
}

// isShopKeeperInWorld includes the boot-time shop assignment. C's
// assign_the_shopkeepers() writes the shop_keeper mob spec from .shp data;
// Go keeps that source-of-truth membership on World rather than mutating the
// static special-procedure table (shop.c:1232-1243).
func isShopKeeperInWorld(world *World, target combat.Combatant) bool {
	if isShopKeeper(target) {
		return true
	}
	mob, ok := target.(*MobInstance)
	if !ok || mob == nil || mob.Proto() == nil || world == nil {
		return false
	}
	_, ok = world.ShopBitvectorForKeeper(mob.Proto().VNum)
	return ok
}

// C act() applies CAP() after token substitution. SkillResult messages bypass
// the shared Act broadcaster, so preserve that final transformation here.
func stealthActMessage(message string, chPronouns Pronouns, victPronouns *Pronouns, itemName string) string {
	message = ActMessage(message, chPronouns, victPronouns, itemName)
	runes := []rune(message)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func appendImprovementMessage(message, improvement string) string {
	if improvement == "" {
		return message
	}
	return message + "\r\n" + strings.TrimSuffix(improvement, "\r\n")
}

func targetGold(target combat.Combatant) int {
	switch target := target.(type) {
	case *Player:
		return target.GetGold()
	case *MobInstance:
		return target.GetGold()
	default:
		return 0
	}
}

// spendTargetGold atomically debits gold from a Player or Mob victim,
// returning false when the purse is short. The coin steal read the victim's
// balance to size its haul and then subtracted it through separate accessors,
// so two concurrent steals both passed the sizing read and both credited a
// thief (CWE-362/CWE-367).
func spendTargetGold(target combat.Combatant, gold int) bool {
	switch target := target.(type) {
	case *Player:
		return target.SpendGold(gold)
	case *MobInstance:
		return target.SpendGold(gold)
	default:
		return false
	}
}

// DoCarve implements do_carve() — carve food from a corpse.
