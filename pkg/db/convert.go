package db

import (
	"encoding/json"
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// PlayerToRecord converts a *game.Player to a *PlayerRecord for saving.
func PlayerToRecord(p *game.Player, worldObjs map[int]*game.ObjectInstance) (*PlayerRecord, error) {
	invBytes, err := json.Marshal(inventoryVnums(p.Inventory))
	if err != nil {
		return nil, fmt.Errorf("serialize inventory: %w", err)
	}
	eqBytes, err := json.Marshal(equipmentVnums(p.Equipment))
	if err != nil {
		return nil, fmt.Errorf("serialize equipment: %w", err)
	}

	return &PlayerRecord{
		ID:        p.ID,
		Name:      p.Name,
		RoomVNum:  p.GetRoom(),
		Level:     p.Level,
		Exp:       p.Exp,
		Health:    p.Health,
		MaxHealth: p.MaxHealth,
		Mana:      p.Mana,
		MaxMana:   p.MaxMana,
		Strength:  p.Strength,
		Class:     p.Class,
		Race:      p.Race,
		StatStr:   p.Stats.Str,
		StatInt:   p.Stats.Int,
		StatWis:   p.Stats.Wis,
		StatDex:   p.Stats.Dex,
		StatCon:   p.Stats.Con,
		StatCha:   p.Stats.Cha,
		Inventory: invBytes,
		Equipment: eqBytes,
	}, nil
}

// RecordToPlayer restores a *game.Player from a *PlayerRecord.
func RecordToPlayer(r *PlayerRecord, world *game.World) (*game.Player, error) {
	p := game.NewCharacter(r.ID, r.Name, r.Class, r.Race)

	// Override rolled stats with saved values
	p.Stats = game.CharStats{
		Str: r.StatStr, Int: r.StatInt, Wis: r.StatWis,
		Dex: r.StatDex, Con: r.StatCon, Cha: r.StatCha,
	}
	p.Strength = r.StatStr
	p.Level    = r.Level
	p.Exp      = r.Exp
	p.Health   = r.Health
	p.MaxHealth = r.MaxHealth
	p.Mana     = r.Mana
	p.MaxMana  = r.MaxMana
	p.SetRoom(r.RoomVNum)
	p.ID       = r.ID

	// Restore inventory from vnums
	var invVnums []int
	if len(r.Inventory) > 0 {
		if err := json.Unmarshal(r.Inventory, &invVnums); err == nil {
			for _, vnum := range invVnums {
				if proto, ok := world.GetObjPrototype(vnum); ok {
					obj := game.NewObjectInstance(proto, -1)
					p.Inventory.AddItem(obj)
				}
			}
		}
	}

	// Restore equipment from slot->vnum map
	var eqMap map[string]int
	if len(r.Equipment) > 0 {
		if err := json.Unmarshal(r.Equipment, &eqMap); err == nil {
			for slotName, vnum := range eqMap {
				slot, ok := game.ParseEquipmentSlot(slotName)
				if !ok {
					continue
				}
				if proto, ok := world.GetObjPrototype(vnum); ok {
					obj := game.NewObjectInstance(proto, -1)
					obj.EquipPosition = int(slot)
					p.Equipment.Slots[slot] = obj
				}
			}
		}
	}

	return p, nil
}

// inventoryVnums returns a slice of vnums for inventory serialization.
func inventoryVnums(inv *game.Inventory) []int {
	if inv == nil {
		return []int{}
	}
	items := inv.FindItems("")
	vnums := make([]int, 0, len(items))
	for _, item := range items {
		vnums = append(vnums, item.VNum)
	}
	return vnums
}

// equipmentVnums returns a slot->vnum map for equipment serialization.
func equipmentVnums(eq *game.Equipment) map[string]int {
	result := make(map[string]int)
	if eq == nil {
		return result
	}
	for slot, item := range eq.GetEquippedItems() {
		result[slot.String()] = item.VNum
	}
	return result
}
