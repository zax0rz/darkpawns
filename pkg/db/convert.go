package db

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// PlayerToRecord converts a *game.Player to a *PlayerRecord for saving.
func PlayerToRecord(p *game.Player, worldObjs map[int]*game.ObjectInstance) (*PlayerRecord, error) {
	invBytes, err := json.Marshal(inventorySaveData(p.Inventory))
	if err != nil {
		return nil, fmt.Errorf("serialize inventory: %w", err)
	}
	eqBytes, err := json.Marshal(equipmentSaveData(p.Equipment))
	if err != nil {
		return nil, fmt.Errorf("serialize equipment: %w", err)
	}

	return &PlayerRecord{
		ID:         p.ID,
		Name:       p.Name,
		RoomVNum:   p.GetRoom(),
		Level:      p.Level,
		Exp:        p.Exp,
		Health:     p.Health,
		MaxHealth:  p.MaxHealth,
		Mana:       p.Mana,
		MaxMana:    p.MaxMana,
		Move:       p.Move,
		MaxMove:    p.MaxMove,
		Strength:   p.Strength,
		Class:      p.Class,
		Race:       p.Race,
		StatStr:    p.Stats.Str,
		StatStrAdd: p.Stats.StrAdd,
		StatInt:    p.Stats.Int,
		StatWis:    p.Stats.Wis,
		StatDex:    p.Stats.Dex,
		StatCon:    p.Stats.Con,
		StatCha:    p.Stats.Cha,
		Hunger:     p.Hunger,
		Thirst:     p.Thirst,
		Drunk:      p.Drunk,
		Hometown:   p.Hometown,
		Inventory:  invBytes,
		Equipment:  eqBytes,
	}, nil
}

// RecordToPlayer restores a *game.Player from a *PlayerRecord.
func RecordToPlayer(r *PlayerRecord, world *game.World) (*game.Player, error) {
	p := game.NewCharacter(r.ID, r.Name, r.Class, r.Race)

	// Override rolled stats with saved values
	p.Stats = game.CharStats{
		Str:    r.StatStr,
		StrAdd: r.StatStrAdd,
		Int:    r.StatInt,
		Wis:    r.StatWis,
		Dex:    r.StatDex,
		Con:    r.StatCon,
		Cha:    r.StatCha,
	}
	p.Strength = r.StatStr
	p.Level = r.Level
	p.Exp = r.Exp
	p.Health = r.Health
	p.MaxHealth = r.MaxHealth
	p.Mana = r.Mana
	p.MaxMana = r.MaxMana
	p.Move = r.Move
	p.MaxMove = r.MaxMove
	p.Hunger = r.Hunger
	p.Thirst = r.Thirst
	p.Drunk = r.Drunk
	p.Hometown = r.Hometown
	p.SetRoom(r.RoomVNum)
	p.ID = r.ID

	// Restore inventory — try new SaveItemData format first, fall back to legacy []int.
	if len(r.Inventory) > 0 {
		var invItems []game.SaveItemData
		if err := json.Unmarshal(r.Inventory, &invItems); err == nil {
			for _, item := range invItems {
				if proto, ok := world.GetObjPrototype(item.VNum); ok {
					obj := game.NewObjectInstance(proto, -1)
					if item.State != nil {
						for k, v := range item.State {
							obj.CustomData[k] = v
						}
						obj.MigrateCustomData()
					}
					if p.Inventory.RestoreItem(obj) {
						slog.Warn("restored item over inventory capacity",
							"player", p.Name, "vnum", obj.VNum)
					}
				}
			}
		} else {
			// Legacy format: plain []int of vnums
			var invVnums []int
			if err := json.Unmarshal(r.Inventory, &invVnums); err == nil {
				for _, vnum := range invVnums {
					if proto, ok := world.GetObjPrototype(vnum); ok {
						obj := game.NewObjectInstance(proto, -1)
						if p.Inventory.RestoreItem(obj) {
							slog.Warn("restored item over inventory capacity",
								"player", p.Name, "vnum", obj.VNum)
						}
					}
				}
			}
		}
	}

	// Restore equipment — try new SaveItemData format first, fall back to legacy map[string]int.
	if len(r.Equipment) > 0 {
		var eqItems []game.SaveItemData
		if err := json.Unmarshal(r.Equipment, &eqItems); err == nil {
			for _, item := range eqItems {
				if proto, ok := world.GetObjPrototype(item.VNum); ok {
					obj := game.NewObjectInstance(proto, -1)
					if item.State != nil {
						for k, v := range item.State {
							obj.CustomData[k] = v
						}
						obj.MigrateCustomData()
					}
					slot, ok := game.CWearPosToSlot(item.Locate - 1)
					if !ok {
						if p.Inventory.RestoreItem(obj) {
							slog.Warn("restored item over inventory capacity",
								"player", p.Name, "vnum", obj.VNum)
						}
						continue
					}
					obj.Location = game.LocEquippedPlayer(p.Name, slot)
					p.Equipment.Slots[slot] = obj
				}
			}
		} else {
			// Legacy format: map[string]int of slot name -> vnum
			var eqMap map[string]int
			if err := json.Unmarshal(r.Equipment, &eqMap); err == nil {
				for slotName, vnum := range eqMap {
					slot, ok := game.ParseEquipmentSlot(slotName)
					if !ok {
						continue
					}
					if proto, ok := world.GetObjPrototype(vnum); ok {
						obj := game.NewObjectInstance(proto, -1)
						obj.Location = game.LocEquippedPlayer(p.Name, slot)
						p.Equipment.Slots[slot] = obj
					}
				}
			}
		}
	}

	return p, nil
}

// inventorySaveData returns SaveItemData for each inventory item, preserving state.
func inventorySaveData(inv *game.Inventory) []game.SaveItemData {
	if inv == nil {
		return []game.SaveItemData{}
	}
	items := inv.FindItems("")
	result := make([]game.SaveItemData, 0, len(items))
	for _, item := range items {
		vnum := item.GetVNum()
		result = append(result, game.SaveItemData{
			VNum:   vnum,
			Count:  1,
			Locate: 0,
			State:  item.GetSaveState(),
		})
	}
	return result
}

// equipmentSaveData returns SaveItemData for each equipped item, preserving slot and state.
func equipmentSaveData(eq *game.Equipment) []game.SaveItemData {
	if eq == nil {
		return []game.SaveItemData{}
	}
	result := make([]game.SaveItemData, 0)
	for slot, item := range eq.GetEquippedItems() {
		cPos, ok := game.SlotToCWearPos(slot)
		locate := 0
		if ok {
			locate = cPos + 1
		}
		result = append(result, game.SaveItemData{
			VNum:   item.GetVNum(),
			Count:  1,
			Locate: locate,
			State:  item.GetSaveState(),
		})
	}
	return result
}
