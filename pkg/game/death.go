package game

// death.go — implements die() / raw_kill() / make_corpse() from fight.c
//
// Original Dark Pawns behavior (fight.c):
//   die(ch):         gain_exp(ch, -(GET_EXP(ch)/3)); raw_kill(ch)
//   raw_kill(ch):    stop_fighting, clear affects, make_corpse, extract_char
//   make_corpse(ch): create container obj named "<name> corpse",
//                    transfer all inventory+equipment+gold into it,
//                    place corpse in room (or mortal_start_room if NOWHERE)
//
// Respawn (not in original — players reconnected or got resurrected):
//   We add a modern respawn: move player to room 8004, heal to full.
//   This is flagged with a TODO so it can be replaced with proper
//   resurrection mechanics later.

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// MortalStartRoom is the vnum of the mortal start room (config.c: mortal_start_room = 8004)
const MortalStartRoom = 8004

// HandleDeath is the DeathFunc set on the combat engine.
// It handles both player and mob death faithfully to the original.
// This is called for combat deaths (die_with_killer).
// Source: fight.c die_with_killer() uses GET_EXP(ch)/37
func (w *World) HandleDeath(victim, killer combat.Combatant, attackType int) {
	if victim.IsNPC() {
		w.handleMobDeath(victim, attackType)
	} else {
		w.handlePlayerDeath(victim, true, attackType) // combat death
	}
}

// HandleNonCombatDeath handles non-combat deaths (bleed-out, legacy).
// Source: fight.c die() uses GET_EXP(ch)/3
func (w *World) HandleNonCombatDeath(victim combat.Combatant) {
	if victim.IsNPC() {
		w.handleMobDeath(victim, -1) // TYPE_UNDEFINED
	} else {
		w.handlePlayerDeath(victim, false, -1) // non-combat death, TYPE_UNDEFINED
	}
}

// handleMobDeath implements raw_kill() for NPCs.
// Original: make_corpse (transfers inventory+equipment+gold), extract_char.
func (w *World) handleMobDeath(victim combat.Combatant, attackType int) {
	roomVNum := victim.GetRoom()

	// Find the MobInstance
	w.mu.Lock()
	var deadMob *MobInstance
	var deadMobID int
	for id, mob := range w.activeMobs {
		if mob == victim {
			deadMob = mob
			deadMobID = id
			break
		}
	}
	w.mu.Unlock()

	if deadMob == nil {
		return
	}

	// make_corpse: create a container object in the room
	// Transfer BOTH inventory items AND all equipped slots into the corpse container
	// Source: fight.c:make_corpse() lines ~383-410
	var equipmentItems []*ObjectInstance
	for _, item := range deadMob.Equipment {
		equipmentItems = append(equipmentItems, item)
	}
	
	// Check for SPELL_DISINTEGRATE (93) - use makeDust instead
	if attackType == 93 { // SPELL_DISINTEGRATE
		w.makeDust(deadMob, deadMob.Inventory, equipmentItems, roomVNum)
	} else {
		corpse := w.makeCorpse(deadMob.GetName(), deadMob.Inventory, equipmentItems, roomVNum, attackType)
		w.AddItemToRoom(corpse, roomVNum)
	}

	// Notify players in room
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		p.SendMessage(fmt.Sprintf("The corpse of %s falls to the ground.\r\n", deadMob.GetShortDesc()))
	}

	// extract_char: remove mob from active list
	w.mu.Lock()
	delete(w.activeMobs, deadMobID)
	w.mu.Unlock()
}

// handlePlayerDeath implements die()/die_with_killer() + raw_kill() for players.
// Original:
//   die_with_killer(): gain_exp(ch, -(GET_EXP(ch)/37))  (combat death)
//   die():             gain_exp(ch, -(GET_EXP(ch)/3))   (non-combat death)
//   raw_kill(): stop_fighting, make_corpse, extract_char
// Modern addition: respawn at MortalStartRoom, heal to full.
func (w *World) handlePlayerDeath(victim combat.Combatant, isCombatDeath bool, attackType int) {
	roomVNum := victim.GetRoom()

	// Find the Player
	player, ok := victim.(*Player)
	if !ok {
		return
	}

	// EXP loss based on death type
	// die_with_killer(): GET_EXP(ch)/37 (combat death) - fight.c line 590
	// die(): GET_EXP(ch)/3 (non-combat death) - fight.c line 628
	var expLoss int
	if isCombatDeath {
		expLoss = player.Exp / 37
	} else {
		expLoss = player.Exp / 3
	}
	player.mu.Lock()
	player.Exp -= expLoss
	if player.Exp < 0 {
		player.Exp = 0
	}
	player.mu.Unlock()

	if expLoss > 0 {
		player.SendMessage(fmt.Sprintf("You lose %d experience points.\r\n", expLoss))
	}

	// make_corpse: transfer inventory and equipment to corpse
	var inventoryItems []*ObjectInstance
	var equipmentItems []*ObjectInstance
	
	if player.Inventory != nil {
		// Get all items from inventory
		inventoryItems = player.Inventory.FindItems("")
		// Clear inventory and update item states
		for _, item := range inventoryItems {
			item.Carrier = nil
			item.Container = nil
			item.EquippedOn = nil
			item.EquipPosition = -1
		}
		player.Inventory.Clear()
	}
	
	if player.Equipment != nil {
		// Get all equipped items
		equipped := player.Equipment.GetEquippedItems()
		for _, item := range equipped {
			equipmentItems = append(equipmentItems, item)
			item.EquippedOn = nil
			item.EquipPosition = -1
			item.Carrier = nil
			item.Container = nil
		}
		// Clear equipment slots
		player.Equipment.mu.Lock()
		player.Equipment.Slots = make(map[EquipmentSlot]*ObjectInstance)
		player.Equipment.mu.Unlock()
	}

	// Check for SPELL_DISINTEGRATE (93) - use makeDust instead
	if attackType == 93 { // SPELL_DISINTEGRATE
		w.makeDust(player, inventoryItems, equipmentItems, roomVNum)
	} else {
		corpse := w.makeCorpse(player.Name, inventoryItems, equipmentItems, roomVNum, attackType)
		w.AddItemToRoom(corpse, roomVNum)
	}

	// Notify room
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if p != player {
			p.SendMessage(fmt.Sprintf("The lifeless body of %s crumples to the ground.\r\n", player.Name))
		}
	}

	// Respawn: move to MortalStartRoom, heal to full
	// TODO: In original, players reconnected or got resurrected. This is a
	// modern convenience. Replace with proper resurrection flow later.
	player.SetRoom(MortalStartRoom)
	player.Heal(9999)
	player.StopFighting()

	player.SendMessage("\r\nYou feel your soul wrenched from your body...\r\n")
	player.SendMessage(fmt.Sprintf("You lost %d experience.\r\n", expLoss))
	player.SendMessage("\r\nYou awaken in the temple.\r\n\r\n")
}

// CorpseAttackType describes what killed the victim, for corpse descriptions.
// Source: fight.c:283-370
type CorpseAttackType int

const (
	AttackUndefined  CorpseAttackType = iota // TYPE_UNDEFINED: "The corpse of X is lying here."
	AttackFire                               // fire spells: "The charred corpse of X is lying here, still smoking."
	AttackCold                               // chill touch: "The frozen corpse of X is thawing here."
	AttackBlast                              // COLOR_SPRAY/DISRUPT: "A blasted corpse lies here in pieces."
	AttackEnergyDrain                        // ENERGY_DRAIN: "A dried up husk of a corpse is lying here."
	AttackLightning                          // LIGHTNING_BOLT: "The shocked looking corpse of X is lying here."
	AttackPsiblast                           // PSIBLAST: "The corpse of X is lying here, brains exploded everywhere."
	AttackSlash                              // TYPE_SLASH/SKILL_BITE: "The hacked up, bloody corpse of X is lying here."
	AttackDisembowel                         // SKILL_DISEMBOWEL: "The corpse of X is lying here, guts spilled everywhere."
	AttackDrowning                           // SPELL_DROWNING: "The bloated, waterlogged corpse of X is lying here."
	AttackPetrify                            // SPELL_PETRIFY: "The corpse of X is here, frozen in stone."
	AttackCrush                              // TYPE_CRUSH/MAUL: "The crushed, barely recognizable corpse of X is lying here."
	AttackBruised                            // BASH/KICK/PUNCH etc: "The bruised, battered corpse of X is lying here."
	AttackPierce                             // TYPE_PIERCE/STAB: "The well-ventilated corpse of X is lying here."
	AttackNeckBreak                          // SKILL_NECKBREAK: "The corpse of X is lying here, his/her neck snapped in two."
)

// attackTypeToCorpseAttack converts numeric attack type to CorpseAttackType
// Source: fight.c:283-370
// Note: Many TYPE_ and SKILL_ constants are not yet defined in Go.
// We use approximate values based on spell numbers from spells.go.
func attackTypeToCorpseAttack(attackType int) CorpseAttackType {
	switch attackType {
	case 5, 26, 58, 96: // SPELL_BURNING_HANDS, FIREBALL, HELLFIRE, FLAMESTRIKE
		return AttackFire
	case 8: // SPELL_CHILL_TOUCH
		return AttackCold
	case 10, 92: // SPELL_COLOR_SPRAY, SPELL_DISRUPT
		return AttackBlast
	case 21, 94: // SPELL_ENERGY_DRAIN, SPELL_SOUL_LEECH
		return AttackEnergyDrain
	case 30, 15, 31: // SPELL_LIGHTNING_BOLT, SPELL_CALL_LIGHTNING, SPELL_SHOCKING_GRASP
		return AttackLightning
	case 34: // SPELL_PSIBLAST
		return AttackPsiblast
	case 93: // SPELL_DISINTEGRATE - handled separately in makeDust
		return AttackUndefined // Should not reach here for disintegrate
	case 24: // SPELL_DROWNING
		return AttackDrowning
	case 35: // SPELL_PETRIFY
		return AttackPetrify
	// TODO: Add TYPE_ and SKILL_ constants when they're defined
	default:
		// Check if it's a negative number (TYPE_UNDEFINED or non-combat death)
		if attackType < 0 {
			return AttackUndefined
		}
		// For now, treat all other attack types as undefined
		return AttackUndefined
	}
}

// corpseAttackLongDesc returns the long description for a corpse based on attack type.
// Source: fight.c:283-370
func corpseAttackLongDesc(name string, attackType CorpseAttackType, gender string) string {
	switch attackType {
	case AttackFire:
		return fmt.Sprintf("The charred corpse of %s is lying here, still smoking.", name)
	case AttackCold:
		return fmt.Sprintf("The frozen corpse of %s is thawing here.", name)
	case AttackBlast:
		return "A blasted corpse lies here in pieces."
	case AttackEnergyDrain:
		return "A dried up husk of a corpse is lying here."
	case AttackLightning:
		return fmt.Sprintf("The shocked looking corpse of %s is lying here.", name)
	case AttackPsiblast:
		return fmt.Sprintf("The corpse of %s is lying here, brains exploded everywhere.", name)
	case AttackSlash:
		return fmt.Sprintf("The hacked up, bloody corpse of %s is lying here.", name)
	case AttackDisembowel:
		return fmt.Sprintf("The corpse of %s is lying here, guts spilled everywhere.", name)
	case AttackDrowning:
		return fmt.Sprintf("The bloated, waterlogged corpse of %s is lying here.", name)
	case AttackPetrify:
		return fmt.Sprintf("The corpse of %s is here, frozen in stone.", name)
	case AttackCrush:
		return fmt.Sprintf("The crushed, barely recognizable corpse of %s is lying here.", name)
	case AttackBruised:
		return fmt.Sprintf("The bruised, battered corpse of %s is lying here.", name)
	case AttackPierce:
		return fmt.Sprintf("The well-ventilated corpse of %s is lying here.", name)
	case AttackNeckBreak:
		return fmt.Sprintf("The corpse of %s is lying here, %s neck snapped in two.", name, gender)
	default: // AttackUndefined
		return fmt.Sprintf("The corpse of %s is lying here.", name)
	}
}

// makeCorpse creates a corpse container object, faithfully to make_corpse() in fight.c.
// The corpse is an ObjectInstance with ITEM_NODONATE, containing the victim's inventory.
func (w *World) makeCorpse(name string, inventory []*ObjectInstance, equipment []*ObjectInstance, roomVNum int, attackType int) *ObjectInstance {
	corpse := &ObjectInstance{
		Prototype:     nil, // synthetic object, no prototype vnum
		VNum:          -1,
		RoomVNum:      roomVNum,
		Contains:      make([]*ObjectInstance, 0),
		CustomData:    map[string]interface{}{
			"is_corpse":   true,
			"corpse_name": name,
			// OBJ_VAL(3) = 1 in original (corpse identifier)
			"corpse_id": 1,
		},
		EquipPosition: -1,
	}

	// Name and descriptions — from make_corpse() in fight.c
	corpse.CustomData["name"] = fmt.Sprintf("%s corpse", name)
	corpse.CustomData["short_desc"] = fmt.Sprintf("the corpse of %s", name)
	// Convert attack type to corpse description
	corpseAttackType := attackTypeToCorpseAttack(attackType)
	// For now, use "his" as default gender - TODO: get actual gender from victim
	gender := "his"
	corpse.CustomData["long_desc"] = corpseAttackLongDesc(name, corpseAttackType, gender)

	// Transfer inventory into corpse (obj_to_obj in original)
	for _, item := range inventory {
		if item != nil {
			item.Container = corpse
			item.RoomVNum = -1
			corpse.Contains = append(corpse.Contains, item)
		}
	}
	for _, item := range equipment {
		if item != nil {
			item.Container = corpse
			item.RoomVNum = -1
			corpse.Contains = append(corpse.Contains, item)
		}
	}

	return corpse
}

// makeDust implements make_dust() for SPELL_DISINTEGRATE
// Source: fight.c lines 433-480
func (w *World) makeDust(victim interface{}, inventory []*ObjectInstance, equipment []*ObjectInstance, roomVNum int) {
	// Scatter ALL inventory items directly to room floor
	for _, item := range inventory {
		if item != nil {
			item.Container = nil
			item.RoomVNum = roomVNum
			w.AddItemToRoom(item, roomVNum)
		}
	}
	
	// Scatter ALL equipment directly to room floor
	for _, item := range equipment {
		if item != nil {
			item.Container = nil
			item.RoomVNum = roomVNum
			item.EquippedOn = nil
			item.EquipPosition = -1
			w.AddItemToRoom(item, roomVNum)
		}
	}
	
	// Create ash object
	ash := &ObjectInstance{
		Prototype:     nil, // synthetic object
		VNum:          -1,
		RoomVNum:      roomVNum,
		Contains:      make([]*ObjectInstance, 0),
		CustomData:    map[string]interface{}{
			"name":        "a pile of ash",
			"short_desc":  "a pile of ash",
			"long_desc":   "A small pile of ash is all that remains.",
			"is_ash":      true,
		},
		EquipPosition: -1,
	}
	w.AddItemToRoom(ash, roomVNum)
	
	// Send room message
	victimName := ""
	switch v := victim.(type) {
	case *Player:
		victimName = v.Name
	case *MobInstance:
		victimName = v.GetShortDesc()
	}
	
	if victimName != "" {
		players := w.GetPlayersInRoom(roomVNum)
		for _, p := range players {
			p.SendMessage(fmt.Sprintf("%s is disintegrated! Equipment lies scattered on the ground.\r\n", victimName))
		}
	}
}
