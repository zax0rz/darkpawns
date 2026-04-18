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
func (w *World) HandleDeath(victim, killer combat.Combatant) {
	if victim.IsNPC() {
		w.handleMobDeath(victim)
	} else {
		w.handlePlayerDeath(victim)
	}
}

// handleMobDeath implements raw_kill() for NPCs.
// Original: make_corpse (transfers inventory+equipment+gold), extract_char.
func (w *World) handleMobDeath(victim combat.Combatant) {
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
	corpse := w.makeCorpse(deadMob.GetName(), deadMob.Inventory, nil, roomVNum)
	w.AddItemToRoom(corpse, roomVNum)

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

// handlePlayerDeath implements die() + raw_kill() for players.
// Original:
//   die():     gain_exp(ch, -(GET_EXP(ch)/3))
//   raw_kill(): stop_fighting, make_corpse, extract_char
// Modern addition: respawn at MortalStartRoom, heal to full.
func (w *World) handlePlayerDeath(victim combat.Combatant) {
	roomVNum := victim.GetRoom()

	// Find the Player
	player, ok := victim.(*Player)
	if !ok {
		return
	}

	// die(): lose EXP/3 — from fight.c: gain_exp(ch, -(GET_EXP(ch)/3))
	expLoss := player.Exp / 3
	player.mu.Lock()
	player.Exp -= expLoss
	if player.Exp < 0 {
		player.Exp = 0
	}
	player.mu.Unlock()

	if expLoss > 0 {
		player.SendMessage(fmt.Sprintf("You lose %d experience points.\r\n", expLoss))
	}

	// make_corpse: note inventory/equipment use *parser.Obj at this stage.
	// We record item counts for the corpse description but don't transfer
	// ObjectInstances (they don't exist yet for player items).
	// TODO: when player inventory migrates to ObjectInstance, transfer items here.
	invCount := 0
	if player.Inventory != nil {
		invCount = len(player.Inventory.Items)
		player.Inventory.Items = nil
	}
	if player.Equipment != nil {
		for slot := range player.Equipment.Slots {
			delete(player.Equipment.Slots, slot)
		}
	}
	_ = invCount // will be used when items transfer to corpse

	corpse := w.makeCorpse(player.Name, nil, nil, roomVNum)
	w.AddItemToRoom(corpse, roomVNum)

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

// makeCorpse creates a corpse container object, faithfully to make_corpse() in fight.c.
// The corpse is an ObjectInstance with ITEM_NODONATE, containing the victim's inventory.
// Corpse description matches the original "The corpse of <name> is lying here."
// (Attack-type-specific descriptions are a Phase 3 enhancement when we have spell types.)
func (w *World) makeCorpse(name string, inventory []*ObjectInstance, equipment []*ObjectInstance, roomVNum int) *ObjectInstance {
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
	corpse.CustomData["long_desc"] = fmt.Sprintf("The corpse of %s is lying here.", name)

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
