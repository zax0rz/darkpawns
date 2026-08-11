package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoCarve(ch *Player, targetName string, world *World) SkillResult {
	// Find target corpse in room
	objects := world.GetItemsInRoom(ch.GetRoomVNum())
	var corpse *ObjectInstance
	for _, obj := range objects {
		if obj.Prototype.TypeFlag == 9 && strings.Contains(strings.ToLower(obj.GetShortDesc()), strings.ToLower(targetName)) {
			corpse = obj
			break
		}
	}

	if corpse == nil {
		return SkillResult{Success: false, MessageToCh: "There is nothing like that here."}
	}

	if ch.GetSkill(SkillCarve) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Create food item
	food := &ObjectInstance{
		VNum:     corpse.VNum,
		RoomVNum: ch.GetRoomVNum(),
	}
	food.Runtime.ShortDescOverride = "some carved meat from " + corpse.GetShortDesc()

	if err := world.MoveObjectToPlayerInventory(food, ch); err != nil {
		if err2 := world.MoveObjectToRoom(food, ch.GetRoomVNum()); err2 != nil {
			slog.Warn("MoveObjectToRoom failed in carve fallback", "obj_vnum", food.GetVNum(), "error", err2)
		}
	}

	// Remove corpse from room
	if err := world.MoveObjectToNowhere(corpse); err != nil {
		slog.Warn("MoveObjectToNowhere failed in carve", "obj_vnum", corpse.GetVNum(), "error", err)
	}

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("You carve some meat from %s.", corpse.GetShortDesc()),
	}
}

// DoCutthroat implements do_cutthroat() — attempt throat slit from behind.
func DoCutthroat(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillCutthroat) == 0 {
		return SkillResult{Success: false, MessageToCh: "You don't know how!"}
	}

	if target.GetPosition() == combat.PosDead {
		return SkillResult{Success: false, MessageToCh: "They're already dead!"}
	}

	// Skill check: D100 vs skill
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	roll := dprng.Number(1, 100)
	if roll > ch.GetSkill(SkillCutthroat) {
		return SkillResult{
			Success:     false,
			MessageToCh: "Your attempt fails!",
		}
	}

	// C: GET_LEVEL(ch)/2 damage + silence affect
	damage := ch.GetLevel() / 2
	target.TakeDamage(damage)

	return SkillResult{
		Success:       true,
		Damage:        damage,
		MessageToCh:   "You slash their throat!",
		MessageToVict: "Your throat is slashed!",
		MessageToRoom: fmt.Sprintf("%s slashes %s's throat!", ch.Name, target.GetName()),
	}
}

// DoStrike implements do_strike() — quick attack.
func DoStrike(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillStrike) == 0 {
		return SkillResult{Success: false, MessageToCh: "You don't know how!"}
	}

	// Simple damage based on level
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	damage := dprng.Number(1, ch.GetLevel())

	return SkillResult{
		Success:       true,
		Damage:        damage,
		MessageToCh:   fmt.Sprintf("You strike %s!", target.GetName()),
		MessageToVict: fmt.Sprintf("%s strikes you!", ch.Name),
		MessageToRoom: fmt.Sprintf("%s strikes %s!", ch.Name, target.GetName()),
	}
}

// DoCompare is a faithful port of do_compare (src/new_cmds.c:1952). C never
// gates compare on a skill (it only uses APPRAISE for the success probability);
// the prior Go wrapper invented a CanUseSkill gate and this was a stub that
// printed "X vs Y: comparing...". The deterministic rejection paths below are
// oracle-verified; the comparison path (RNG) is transcribed verbatim but marked
// TODO(port) as oracle-unverified — the fresh-mortal fixture carries only one
// weapon/armor, so two comparable items can't be constructed to exercise it.
func DoCompare(ch *Player, objName1, objName2 string) SkillResult {
	// prob = APPRAISE if learned, else 20 + level. C sets prob only.
	prob := ch.GetSkill(SkillAppraise)
	if prob == 0 {
		prob = 20 + ch.GetLevel()
	}

	if ch.IsAffected(affBlind) {
		return SkillResult{MessageToCh: "You can't see a damned thing!\r\n"}
	}

	// C get_obj_in_list_vis("", carrying) returns NULL (empty name matches
	// nothing); Go's helper would match the first item via Contains(x,""), so
	// gate the lookup on a non-empty name.
	var obj1, obj2 *ObjectInstance
	var found1, found2 bool
	if objName1 != "" {
		obj1, found1 = findItemByName(ch, objName1)
	}
	if objName2 != "" {
		obj2, found2 = findItemByName(ch, objName2)
	}
	if !found1 || !found2 || ch.Fighting != "" {
		if ch.Fighting != "" {
			return SkillResult{MessageToCh: "You're pretty busy right now!\n\r"}
		}
		return SkillResult{MessageToCh: "Looks like you don't have those objects..\n\r"}
	}

	if obj1 == obj2 {
		return SkillResult{MessageToCh: "They're the same thing!\n\r"}
	}

	t1 := obj1.GetTypeFlag()
	if t1 != obj2.GetTypeFlag() {
		return SkillResult{MessageToCh: "You can't compare those things!\r\n"}
	}
	if t1 != ITEM_WEAPON && t1 != ITEM_ARMOR {
		return SkillResult{MessageToCh: "Compare is only for weapons and armor.\r\n"}
	}

	// --- Comparison path: RNG-gated, oracle-unverified (see TODO above). ---
	if t1 == ITEM_ARMOR {
		if armorWearSlot(obj1) != armorWearSlot(obj2) {
			return SkillResult{MessageToCh: "You can only compare the same types of armor!\r\n"}
		}
	}

	percent := dprng.Number(1, 101) // C: "101% is a complete failure"
	diff := 0
	if t1 == ITEM_WEAPON {
		diff = int((((float64(obj1.GetValue(2)) + 1) / 2.0) * float64(obj1.GetValue(1))) -
			(((float64(obj2.GetValue(2)) + 1) / 2.0) * float64(obj2.GetValue(1))))
	}
	if t1 == ITEM_ARMOR {
		diff = int(((float64(obj1.GetValue(0)) + 1) / 2.0) - ((float64(obj2.GetValue(0)) + 1) / 2.0))
	}
	if percent > prob {
		diff += dprng.Number(-3, 3)
	} else {
		improveSkill(ch, SkillCompare)
	}

	var msg string
	switch {
	case diff < -5:
		msg = fmt.Sprintf("%s looks much worse than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	case diff < -3:
		msg = fmt.Sprintf("%s looks a little worse than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	case diff < 0:
		msg = fmt.Sprintf("%s looks slightly worse than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	case diff == 0:
		msg = "They look just about the same.\r\n"
	case diff < 3:
		msg = fmt.Sprintf("%s looks slightly better than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	case diff < 5:
		msg = fmt.Sprintf("%s looks a little better than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	default:
		msg = fmt.Sprintf("%s looks much better than %s.\r\n", obj1.GetShortDesc(), obj2.GetShortDesc())
	}
	return SkillResult{MessageToCh: cap(msg)}
}

// armorWearSlot mirrors C do_compare's CAN_WEAR→WEAR mapping for the "same type
// of armor" check (new_cmds.c:2014-2048). Each CAN_WEAR check OVERWRITES where,
// so the LAST set wear flag wins — the check order below is C's source order
// (note WIELD is last even though its bit is lower than ABLEGS/FACE/HOVER).
// ITEM_WEAR_* bits and WEAR_* slots from structs.h:446-464 / 391-411.
func armorWearSlot(obj *ObjectInstance) int {
	wf := obj.Prototype.WearFlags[0]
	where := 0
	if wf&(1<<1) != 0 { // ITEM_WEAR_FINGER  → WEAR_FINGER_R
		where = 1
	}
	if wf&(1<<2) != 0 { // ITEM_WEAR_NECK    → WEAR_NECK_1
		where = 3
	}
	if wf&(1<<3) != 0 { // ITEM_WEAR_BODY    → WEAR_BODY
		where = 5
	}
	if wf&(1<<4) != 0 { // ITEM_WEAR_HEAD    → WEAR_HEAD
		where = 6
	}
	if wf&(1<<5) != 0 { // ITEM_WEAR_LEGS    → WEAR_LEGS
		where = 7
	}
	if wf&(1<<6) != 0 { // ITEM_WEAR_FEET    → WEAR_FEET
		where = 8
	}
	if wf&(1<<7) != 0 { // ITEM_WEAR_HANDS   → WEAR_HANDS
		where = 9
	}
	if wf&(1<<8) != 0 { // ITEM_WEAR_ARMS    → WEAR_ARMS
		where = 10
	}
	if wf&(1<<9) != 0 { // ITEM_WEAR_SHIELD  → WEAR_SHIELD
		where = 11
	}
	if wf&(1<<10) != 0 { // ITEM_WEAR_ABOUT  → WEAR_ABOUT
		where = 12
	}
	if wf&(1<<11) != 0 { // ITEM_WEAR_WAIST  → WEAR_WAIST
		where = 13
	}
	if wf&(1<<12) != 0 { // ITEM_WEAR_WRIST  → WEAR_WRIST_R
		where = 14
	}
	if wf&(1<<16) != 0 { // ITEM_WEAR_ABLEGS → WEAR_ABLEGS
		where = 19
	}
	if wf&(1<<17) != 0 { // ITEM_WEAR_FACE   → WEAR_FACE
		where = 20
	}
	if wf&(1<<18) != 0 { // ITEM_WEAR_HOVER  → WEAR_HOVER
		where = 21
	}
	if wf&(1<<13) != 0 { // ITEM_WEAR_WIELD  → WEAR_WIELD (checked last in C)
		where = 16
	}
	return where
}

// DoScan implements do_scan() — scan surrounding rooms.
func DoScan(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillScan) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Get current room exits
	room := world.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil {
		return SkillResult{Success: false, MessageToCh: "You are in a void."}
	}

	var scanResult string
	scanResult = "You scan the area...\r\n"

	for dir, exit := range room.Exits {
		if exit.ToRoom > 0 {
			exitRoom := world.GetRoomInWorld(exit.ToRoom)
			if exitRoom != nil {
				exitName := exitRoom.Name
				// Check for players in that room
				players := world.GetPlayersInRoom(exit.ToRoom)
				if len(players) > 0 {
					for _, p := range players {
						scanResult += fmt.Sprintf("%-5s - %s is there.\r\n", strings.ToUpper(dir), p.Name)
					}
				} else {
					scanResult += fmt.Sprintf("%-5s - %s (empty)\r\n", strings.ToUpper(dir), exitName)
				}
			}
		}
	}

	if scanResult == "You scan the area...\r\n" {
		scanResult += "Nothing interesting."
	}

	return SkillResult{Success: true, MessageToCh: scanResult}
}

// DoSharpen implements do_sharpen() — sharpen a weapon.
func DoSharpen(ch *Player, objName string) SkillResult {
	if ch.GetSkill(SkillSharpen) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	obj, found := findItemByName(ch, objName)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	// Check it's a weapon
	if obj.Prototype.TypeFlag != 0 {
		return SkillResult{Success: false, MessageToCh: "You can only sharpen weapons."}
	}

	// Simple sharpen: success based on skill level
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	roll := dprng.Number(1, 100)
	if roll <= ch.GetSkill(SkillSharpen) {
		return SkillResult{
			Success:     true,
			MessageToCh: fmt.Sprintf("You sharpen %s. It looks more deadly!", obj.GetShortDesc()),
		}
	}

	return SkillResult{
		Success:     false,
		MessageToCh: "You fail to sharpen it properly.",
	}
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// findItemByName searches a player's inventory and equipment for an item matching name.
