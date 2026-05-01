package game

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
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
		VNum:    corpse.VNum,
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

// DoCutthroat implements do_cutthroat() — attempt instant kill from behind.
func DoCutthroat(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillCutthroat) == 0 {
		return SkillResult{Success: false, MessageToCh: "You don't know how!"}
	}

	if target.GetHP() <= 0 {
		return SkillResult{Success: false, MessageToCh: "They're already dead!"}
	}

	// Skill check: D100 vs skill
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	roll := rand.Intn(100) + 1
	if roll > ch.GetSkill(SkillCutthroat) {
		return SkillResult{
			Success:     false,
			MessageToCh: "Your attempt fails!",
		}
	}

	// Instant kill: set target to -1 HP
	damage := target.GetHP() + 1
	target.TakeDamage(damage)

	return SkillResult{
		Success:     true,
		Damage:      damage,
		MessageToCh: "You slash their throat!",
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
	damage := rand.Intn(ch.GetLevel()) + 1

	return SkillResult{
		Success:     true,
		Damage:      damage,
		MessageToCh: fmt.Sprintf("You strike %s!", target.GetName()),
		MessageToVict: fmt.Sprintf("%s strikes you!", ch.Name),
		MessageToRoom: fmt.Sprintf("%s strikes %s!", ch.Name, target.GetName()),
	}
}

// DoCompare implements do_compare() — compare weapons or armor.
func DoCompare(ch *Player, objName1, objName2 string, compareToEquipped bool) SkillResult {
	// Find the first object
	obj1, found := findItemByName(ch, objName1)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	// Find the second object
	if compareToEquipped {
		// Compare against equipped weapon
		result := fmt.Sprintf("Comparing %s with your weapon...", obj1.GetShortDesc())
		return SkillResult{Success: true, MessageToCh: result}
	}

	obj2, found := findItemByName(ch, objName2)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	result := fmt.Sprintf("%s vs %s: comparing...", obj1.GetShortDesc(), obj2.GetShortDesc())
	return SkillResult{Success: true, MessageToCh: result}
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
	roll := rand.Intn(100) + 1
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
