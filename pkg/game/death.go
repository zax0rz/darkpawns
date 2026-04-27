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
//   Placeholder implementation — should be replaced with proper
//   resurrection mechanics later.

import (
	"context"
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/events"
)

// Con loss probability thresholds — from fight.c die_with_killer()
// These exactly mirror the C logic:
//   level 1-4:  always lose 1 con (no random check)
//   level 5-9:  lose 1 con on !number(0,1) = 50% (ConLossLevel5Chance = 2 means 1-in-2)
//   level 10+:  lose 1 con on !number(0,2) = 33% (ConLossLevel10Chance = 3 means 1-in-3)
//   level 5+:   also get the first check (GET_LEVEL(ch) > 5 && !number(0,3) in C)
// The C code actually does:
//   if (GET_LEVEL(ch) > 5 && !number(0,3)) // 1/4 chance, 3/4 no loss
//   if (GET_LEVEL(ch) > 20 && !number(0,5)) // 1/6 additional chance for second con
// This means:
//   level 1-5:   no CON loss from die_with_killer (the >5 check fails)
//   level 6-20:  !number(0,3) = 25% chance lose 1 con
//   level 21+:   same 25% + additional !number(0,5) = 16.7% chance for 2nd con
const (
	ConLossCheckChance = 4  // !number(0,3) = 3/4 chance skip, 1/4 chance lose 1 con
	ConLossSecondChance = 6 // !number(0,5) = 5/6 chance skip, 1/6 chance lose 2nd con
	ConLossMinLevel     = 6  // Level > 5 → minimum level 6 for any CON loss
	ConLossSecondLevel  = 21 // Level > 20 → minimum level 21 for second CON loss
)

// MortalStartRoom is the vnum of the mortal start room (config.c: mortal_start_room = 8004)
const MortalStartRoom = 8004

// HandleDeath is the DeathFunc set on the combat engine.
// It handles both player and mob death faithfully to the original.
// This is called for combat deaths (die_with_killer).
// Source: fight.c die_with_killer() uses GET_EXP(ch)/37
func (w *World) HandleDeath(victim, killer combat.Combatant, attackType int) {
	if victim.IsNPC() {
		// Capture mob vnum/exp/gold before the mob is removed from the world.
		// Source: fight.c die_with_killer() line 1638 — group_gain(ch, victim)
		mobExp := 0
		mobGold := 0
		mobVNum := 0
		if mob, ok := victim.(*MobInstance); ok && mob.Prototype != nil {
			mobExp = mob.Prototype.Exp
			mobGold = mob.Prototype.Gold
			mobVNum = mob.Prototype.VNum
		}
		// Fire memory hook before removing mob from active list
		killerName := ""
		killerIsNPC := false
		if killer != nil {
			killerName = killer.GetName()
			killerIsNPC = killer.IsNPC()
		}
		roomName := ""
		if room, ok := w.GetRoom(victim.GetRoom()); ok {
			roomName = room.Name
		}
		fireMobKill(&MobKillEvent{
			KillerName:  killerName,
			KillerIsNPC: killerIsNPC,
			VictimName:  victim.GetName(),
			VictimVNum:  mobVNum,
			RoomVNum:    victim.GetRoom(),
			RoomName:    roomName,
		})
		w.handleMobDeath(victim, attackType)
		// Publish typed event bus event
		if w.Events != nil {
// #nosec G104
			w.Events.Publish(context.Background(), events.MobKilledEvent{
				KillerID: killerName,
				MobVNum:  mobVNum,
				RoomVNum: victim.GetRoom(),
			})
		}
		// Award XP and gold to killer and party members — fight.c group_gain() lines 708-830
		w.AwardMobKillXP(killer, mobExp, mobGold)
	} else {
		// Fire player death hook
		killerName := ""
		killerIsNPC := false
		if killer != nil {
			killerName = killer.GetName()
			killerIsNPC = killer.IsNPC()
		}
		roomName := ""
		if room, ok := w.GetRoom(victim.GetRoom()); ok {
			roomName = room.Name
		}
		firePlayerDeath(&PlayerDeathEvent{
			VictimName:  victim.GetName(),
			KillerName:  killerName,
			KillerIsNPC: killerIsNPC,
			RoomVNum:    victim.GetRoom(),
			RoomName:    roomName,
			IsCombat:    true,
		})
		w.handlePlayerDeath(victim, true, attackType, killerName) // combat death with killer
		// Publish typed event bus event
		if w.Events != nil {
// #nosec G104
			w.Events.Publish(context.Background(), events.PlayerKilledEvent{
				KillerID: killerName,
				VictimID: victim.GetName(),
				RoomVNum: victim.GetRoom(),
			})
		}
	}
}

// HandleNonCombatDeath handles non-combat deaths (bleed-out, legacy).
// Source: fight.c die() uses GET_EXP(ch)/3
func (w *World) HandleNonCombatDeath(victim combat.Combatant) {
	if victim.IsNPC() {
		w.handleMobDeath(victim, -1) // TYPE_UNDEFINED
	} else {
		w.handlePlayerDeath(victim, false, -1, "") // non-combat death, TYPE_UNDEFINED, no killer
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

	// Cancel all pending events for this mob.
	// Source: events.c event_cancel() — in original, extract_char would
	// clean up any pending events tied to the character.
	if w.EventQueue != nil {
		w.EventQueue.CancelBySource(deadMobID)
	}

	// make_corpse: create a container object in the room
	// Transfer BOTH inventory items AND all equipped slots into the corpse container
	// Source: fight.c:make_corpse() lines ~383-410
	var inventoryItems []*ObjectInstance
	for _, item := range deadMob.Inventory {
		inventoryItems = append(inventoryItems, item)
	}
	var equipmentItems []*ObjectInstance
	for _, item := range deadMob.Equipment {
		equipmentItems = append(equipmentItems, item)
	}

	// Transfer gold into corpse (as money objects)
	mobGold := deadMob.GetGold()

	// Check for SPELL_DISINTEGRATE (93) - use makeDust instead
	if attackType == 93 { // SPELL_DISINTEGRATE
		w.makeDust(deadMob, inventoryItems, equipmentItems, roomVNum, mobGold)
	} else {
		corpse := w.makeCorpse(deadMob.GetName(), deadMob.GetSex(), inventoryItems, equipmentItems, roomVNum, attackType, mobGold)
// #nosec G104
		w.MoveObjectToRoom(corpse, roomVNum)
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
//
//	die_with_killer(): gain_exp(ch, -(GET_EXP(ch)/37))  (combat death)
//	die():             gain_exp(ch, -(GET_EXP(ch)/3))   (non-combat death)
//	raw_kill(): stop_fighting, make_corpse, extract_char
//
// CON loss (die_with_killer only, fight.c lines 598-607):
//   if GET_LEVEL(ch) > 5 && !number(0,3):  lose 1 con (1/4 chance)
//   if GET_LEVEL(ch) > 20 && !number(0,5): lose 1 more con (1/6 additional chance)
//
// Modern addition: respawn at MortalStartRoom, heal to full.
func (w *World) handlePlayerDeath(victim combat.Combatant, isCombatDeath bool, attackType int, killerName string) {
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

	// CON loss — fight.c die_with_killer() lines 598-607
	// Only applies to combat deaths (die_with_killer path)
	if isCombatDeath && player.Level > ConLossMinLevel-1 {
		// GET_LEVEL(ch) > 5 means level 6+
		if number(0, ConLossCheckChance-1) == 0 {
			player.Stats.Con--
			if player.Stats.Con < 1 {
				player.Stats.Con = 1
			}
			// GET_LEVEL(ch) > 20 means level 21+
			if player.Level >= ConLossSecondLevel && number(0, ConLossSecondChance-1) == 0 {
				player.Stats.Con--
				if player.Stats.Con < 1 {
					player.Stats.Con = 1
				}
			}
			// affect_total(ch) — sends updated stats message
			player.SendMessage(fmt.Sprintf(
				"You lose some constitution! Your Constitution is now %d.\r\n",
				player.Stats.Con))
		}
	}

	// make_corpse: transfer inventory and equipment to corpse
	var inventoryItems []*ObjectInstance
	var equipmentItems []*ObjectInstance

	if player.Inventory != nil {
		// Get all items from inventory
		inventoryItems = player.Inventory.FindItems("")
		// makeCorpse's MoveObjectToContainer will handle detach from old location
		player.Inventory.clear()
	}

	if player.Equipment != nil {
		// Get all equipped items
		// makeCorpse's MoveObjectToContainer will handle detach from equipment
		equipped := player.Equipment.GetEquippedItems()
		for _, item := range equipped {
			equipmentItems = append(equipmentItems, item)
		}
		// Clear equipment slots
		player.Equipment.mu.Lock()
		player.Equipment.Slots = make(map[EquipmentSlot]*ObjectInstance)
		player.Equipment.mu.Unlock()
	}

	// Transfer gold — fight.c make_corpse() line 410: GET_GOLD(ch) -> create_money -> obj_to_obj(corpse)
	player.mu.Lock()
	playerGold := player.Gold
	player.Gold = 0
	player.mu.Unlock()

	// Check for SPELL_DISINTEGRATE (93) - use makeDust instead
	if attackType == 93 { // SPELL_DISINTEGRATE
		w.makeDust(player, inventoryItems, equipmentItems, roomVNum, playerGold)
	} else {
		corpse := w.makeCorpse(player.Name, player.Sex, inventoryItems, equipmentItems, roomVNum, attackType, playerGold)
// #nosec G104
		w.MoveObjectToRoom(corpse, roomVNum)
	}

	// Notify room
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if p != player {
			p.SendMessage(fmt.Sprintf("The lifeless body of %s crumples to the ground.\r\n", player.Name))
		}
	}

	// Respawn: move to MortalStartRoom, heal to full
	// In the original C code, players could reconnect or get resurrected. This is a
	// modern convenience. Replace with proper resurrection flow later.
	player.SetRoom(MortalStartRoom)
	player.SetPosition(combat.PosStanding)
	player.Heal(9999)
	player.StopFighting()

	player.SendMessage("\r\nYou feel your soul wrenched from your body...\r\n")
	player.SendMessage(fmt.Sprintf("You lost %d experience.\r\n", expLoss))
	player.SendMessage("\r\nYou awaken in the temple.\r\n\r\n")
}

// Attack type constants (TYPE_*) from spells.h:266-283
// Weapon types: TYPE_HIT(300) through TYPE_SUFFERING(399)
const (
	TypeUndefined = -1
	TypeHit      = 300
	TypeSting    = 301
	TypeWhip     = 302
	TypeSlash    = 303
	TypeBite     = 304
	TypeBludgeon = 305
	TypeCrush    = 306
	TypePound    = 307
	TypeClaw     = 308
	TypeMaul     = 309
	TypeThrash   = 310
	TypePierce   = 311
	TypeBlast    = 312
	TypePunch    = 313
	TypeStab     = 314
	TypeSuffering = 399
)

// Skill numeric IDs from spells.h — used in attack type resolution.
// Note: skills.go uses string identifiers; these numeric values match C source.
// Only defined here because attackTypeToCorpseAttack receives int (C-style numeric type).
const (
	SkillBackstabNum   = 131
	SkillBashNum       = 132
	SkillKickNum       = 134
	SkillPunchNum      = 136
	SkillBiteNum       = 150
	SkillHeadbuttNum   = 141
	SkillSmackheadsNum = 145
	SkillSlugNum       = 146
	SkillSerpentKickNum = 156
	SkillCircleNum     = 173
	SkillDisembowelNum = 184
	SkillSleeperNum    = 187
	SkillNeckbreakNum  = 190
	SkillDragonKickNum  = 222
	SkillTigerPunchNum  = 223
)

// CorpseAttackType describes what killed the victim, for corpse descriptions.
// Source: fight.c:283-370
type CorpseAttackType int

const (
	AttackUndefined   CorpseAttackType = iota // TYPE_UNDEFINED: "The corpse of X is lying here."
	AttackFire                                // fire spells: "The charred corpse of X is lying here, still smoking."
	AttackCold                                // chill touch: "The frozen corpse of X is thawing here."
	AttackBlast                               // COLOR_SPRAY/DISRUPT: "A blasted corpse lies here in pieces."
	AttackEnergyDrain                         // ENERGY_DRAIN: "A dried up husk of a corpse is lying here."
	AttackLightning                           // LIGHTNING_BOLT: "The shocked looking corpse of X is lying here."
	AttackPsiblast                            // PSIBLAST: "The corpse of X is lying here, brains exploded everywhere."
	AttackSlash                               // TYPE_SLASH/SKILL_BITE: "The hacked up, bloody corpse of X is lying here."
	AttackDisembowel                          // SKILL_DISEMBOWEL: "The corpse of X is lying here, guts spilled everywhere."
	AttackDrowning                            // SPELL_DROWNING: "The bloated, waterlogged corpse of X is lying here."
	AttackPetrify                             // SPELL_PETRIFY: "The corpse of X is here, frozen in stone."
	AttackCrush                               // TYPE_CRUSH/MAUL: "The crushed, barely recognizable corpse of X is lying here."
	AttackBruised                             // BASH/KICK/PUNCH etc: "The bruised, battered corpse of X is lying here."
	AttackPierce                              // TYPE_PIERCE/STAB: "The well-ventilated corpse of X is lying here."
	AttackNeckBreak                           // SKILL_NECKBREAK: "The corpse of X is lying here, his/her neck snapped in two."
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
	case 21, 83: // SPELL_ENERGY_DRAIN, SPELL_SOUL_LEECH (was incorrectly 94 before fix)
		return AttackEnergyDrain
	case 30, 6, 31: // SPELL_LIGHTNING_BOLT, SPELL_CALL_LIGHTNING (was incorrectly 15 before fix), SPELL_SHOCKING_GRASP
		return AttackLightning
	case 34: // SPELL_PSIBLAST
		return AttackPsiblast
	case 93: // SPELL_DISINTEGRATE - handled separately in makeDust
		return AttackUndefined // Should not reach here for disintegrate
	case 103: // SPELL_DROWNING (was incorrectly 24 before fix)
		return AttackDrowning
	case 35: // SPELL_PETRIFY
		return AttackPetrify
	case TypeBludgeon, TypePound, TypePunch, TypeWhip,
		SkillBashNum, SkillKickNum, SkillPunchNum, SkillDragonKickNum, SkillTigerPunchNum,
		SkillHeadbuttNum, SkillSmackheadsNum, SkillSlugNum, SkillSerpentKickNum:
		return AttackBruised
	case SkillBiteNum, TypeBite, TypeClaw, TypeSlash,
		SkillBackstabNum, SkillCircleNum:
		return AttackSlash
	case SkillDisembowelNum:
		return AttackDisembowel
	case SkillNeckbreakNum:
		return AttackNeckBreak
	case TypeCrush, TypeMaul, TypeThrash:
		return AttackCrush
	case TypePierce, TypeStab:
		return AttackPierce
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
// createMoneyObject creates a gold coin/money object with the given amount.
// Source: handler.c create_money() — creates a container-type object representing coins.
// In the original, this is a special object without a prototype VNum.
func (w *World) createMoneyObject(amount int) *ObjectInstance {
	if amount <= 0 {
		return nil
	}
	// Build description based on amount (matching handler.c create_money())
	var shortDesc, longDesc, name string
	if amount == 1 {
		name = "coin gold"
		shortDesc = "a gold coin"
		longDesc = "One miserable gold coin is lying here."
	} else {
		name = "coins gold"
		shortDesc = createMoneyDesc(amount)
		longDesc = fmt.Sprintf("%s is lying here.", capitalize(createMoneyDesc(amount)))
	}

	money := &ObjectInstance{
		Prototype: nil, // synthetic object, no prototype
		VNum:      -1,
		RoomVNum:  -1,
		Contains:  make([]*ObjectInstance, 0),
		CustomData: map[string]interface{}{
			"is_money":    true,
			"money_amount": amount,
		},
	}
	money.Runtime.Name = name
	money.Runtime.ShortDesc = shortDesc
	money.Runtime.LongDesc = longDesc

	w.mu.Lock()
	money.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[money.ID] = money
	w.mu.Unlock()

	return money
}

// createMoneyDesc wraps the amount for the short description (like handler.c money_desc())
func createMoneyDesc(amount int) string {
	if amount == 1 {
		return "a gold coin"
	}
	// Simplified version of money_desc() — C has specific ranges
	return fmt.Sprintf("a pile of %d gold coins", amount)
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 32
	}
	return string(r)
}

// genderPronoun returns the possessive pronoun for the given sex value.
func genderPronoun(sex int) string {
	switch sex {
	case 1:
		return "her"
	case 2:
		return "its"
	default:
		return "his"
	}
}

func (w *World) makeCorpse(name string, sex int, inventory []*ObjectInstance, equipment []*ObjectInstance, roomVNum int, attackType int, gold int) *ObjectInstance {
	corpse := &ObjectInstance{
		Prototype: nil, // synthetic object, no prototype vnum
		VNum:      -1,
		RoomVNum:  roomVNum,
		Contains:  make([]*ObjectInstance, 0),
		CustomData: map[string]interface{}{
			"is_corpse":   true,
			"corpse_name": name,
			// OBJ_VAL(3) = 1 in original (corpse identifier)
			"corpse_id": 1,
		},
	}

	// Name and descriptions — from make_corpse() in fight.c
	corpse.Runtime.Name = fmt.Sprintf("%s corpse", name)
	corpse.Runtime.ShortDesc = fmt.Sprintf("the corpse of %s", name)
	// Convert attack type to corpse description
	corpseAttackType := attackTypeToCorpseAttack(attackType)
	gender := genderPronoun(sex)
	corpse.Runtime.LongDesc = corpseAttackLongDesc(name, corpseAttackType, gender)

	// Give the corpse a unique ID before MoveObjectToContainer (needs valid ContainerObjID)
	w.mu.Lock()
	corpse.ID = w.nextObjID
	w.nextObjID++
	w.objectInstances[corpse.ID] = corpse
	w.mu.Unlock()

	// Copy inventory slice to avoid mutation issues during MoveObjectToContainer
	// (MoveObjectToContainer's detach will modify the mob's Inventory backing slice)
	invCopy := make([]*ObjectInstance, len(inventory))
	copy(invCopy, inventory)
	equipCopy := make([]*ObjectInstance, len(equipment))
	copy(equipCopy, equipment)

	// Transfer inventory into corpse (obj_to_obj in original)
	for _, item := range invCopy {
		if item != nil {
			_ = w.MoveObjectToContainer(item, corpse)
		}
	}
	for _, item := range equipCopy {
		if item != nil {
			_ = w.MoveObjectToContainer(item, corpse)
		}
	}

	// Transfer gold — fight.c make_corpse() lines 406-413:
	// if (GET_GOLD(ch) > 0) { money = create_money(GET_GOLD(ch)); obj_to_obj(money, corpse); GET_GOLD(ch) = 0; }
	if gold > 0 {
		moneyObj := w.createMoneyObject(gold)
		_ = w.MoveObjectToContainer(moneyObj, corpse)
	}

	return corpse
}

// makeDust implements make_dust() for SPELL_DISINTEGRATE
// Source: fight.c lines 433-480
func (w *World) makeDust(victim interface{}, inventory []*ObjectInstance, equipment []*ObjectInstance, roomVNum int, gold int) {
	// Scatter ALL inventory items directly to room floor
	for _, item := range inventory {
		if item != nil {
			_ = w.MoveObjectToRoom(item, roomVNum)
		}
	}

	// Scatter ALL equipment directly to room floor
	for _, item := range equipment {
		if item != nil {
			_ = w.MoveObjectToRoom(item, roomVNum)
		}
	}

	// Scatter gold as money objects to room floor (original make_dust also drops gold)
	if gold > 0 {
		moneyObj := w.createMoneyObject(gold)
		_ = w.MoveObjectToRoom(moneyObj, roomVNum)
	}

	// Create ash object
	ash := &ObjectInstance{
		Prototype: nil, // synthetic object
		VNum:      -1,
		RoomVNum:  roomVNum,
		Contains:  make([]*ObjectInstance, 0),
		CustomData: map[string]interface{}{
			"name":       "a pile of ash",
			"short_desc": "a pile of ash",
			"long_desc":  "A small pile of ash is all that remains.",
			"is_ash":     true,
		},
	}
// #nosec G104
	w.MoveObjectToRoom(ash, roomVNum)

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
