package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// observationBodyActor adapts a dispatched character body to the shared act
// and visibility boundary. Both live command bodies are Player or MobInstance.
func observationBodyActor(body combat.Combatant) Actor {
	return asActor(body)
}

func observationBodyAffected(body combat.Combatant, bit int) bool {
	if affected, ok := body.(interface{ IsAffected(int) bool }); ok {
		return affected.IsAffected(bit)
	}
	return false
}

// C player preferences are absent on NPC bodies. The zero value here mirrors
// the C NPC path for PRF_* checks; body-owned affects still come from
// MobInstance.IsAffected.
func observationBodyFlags(body combat.Combatant) uint64 {
	if player, ok := body.(*Player); ok && player != nil {
		return player.GetFlags()
	}
	return 0
}

func observationBodyHolyLight(body combat.Combatant) bool {
	if player, ok := body.(*Player); ok && player != nil {
		return player.GetHolyLight()
	}
	return false
}

func observationBodyAutoExit(body combat.Combatant) bool {
	if player, ok := body.(*Player); ok && player != nil {
		return player.GetAutoExit()
	}
	return false
}

func observationBodyRoomFlags(body combat.Combatant) bool {
	if player, ok := body.(*Player); ok && player != nil {
		return player.GetRoomFlags()
	}
	return false
}

func observationBodyInventory(body combat.Combatant) []*ObjectInstance {
	switch character := body.(type) {
	case *Player:
		if character != nil {
			return character.GetInventory()
		}
	case *MobInstance:
		if character != nil {
			return character.InventorySnapshot()
		}
	}
	return nil
}

func observationBodyEquipped(body combat.Combatant, cPosition int) (*ObjectInstance, bool) {
	if cPosition < 0 || cPosition >= len(cWearToGoSlot) {
		return nil, false
	}
	switch character := body.(type) {
	case *Player:
		if character == nil || character.Equipment == nil {
			return nil, false
		}
		return character.Equipment.GetItemInSlot(cWearToGoSlot[cPosition])
	case *MobInstance:
		if character == nil {
			return nil, false
		}
		item, ok := character.EquipmentSnapshot()[cPosition]
		return item, ok
	default:
		return nil, false
	}
}
