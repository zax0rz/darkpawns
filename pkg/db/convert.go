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

	charData, err := game.EncodeCharacterData(p)
	if err != nil {
		return nil, err
	}

	roomVNum := p.GetRoom()
	if p.GetFlags()&(1<<uint(game.PlrLoadroom)) != 0 {
		roomVNum = p.GetLoadRoom()
	}

	return &PlayerRecord{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		Title:         p.Title,
		RoomVNum:      roomVNum,
		Level:         p.Level,
		Exp:           p.Exp,
		Health:        p.Health,
		MaxHealth:     p.MaxHealth,
		Mana:          p.Mana,
		MaxMana:       p.MaxMana,
		Move:          p.Move,
		MaxMove:       p.MaxMove,
		Strength:      p.Strength,
		Class:         p.Class,
		Race:          p.Race,
		StatStr:       p.Stats.Str,
		StatStrAdd:    p.Stats.StrAdd,
		StatInt:       p.Stats.Int,
		StatWis:       p.Stats.Wis,
		StatDex:       p.Stats.Dex,
		StatCon:       p.Stats.Con,
		StatCha:       p.Stats.Cha,
		Hunger:        p.Hunger,
		Thirst:        p.Thirst,
		Drunk:         p.Drunk,
		Hometown:      p.Hometown,
		Inventory:     invBytes,
		Equipment:     eqBytes,
		CharacterData: charData,
	}, nil
}

// RecordToPlayer restores a *game.Player from a *PlayerRecord.
func RecordToPlayer(r *PlayerRecord, world *game.World) (*game.Player, error) {
	stats := game.CharStats{
		Str:    r.StatStr,
		StrAdd: r.StatStrAdd,
		Int:    r.StatInt,
		Wis:    r.StatWis,
		Dex:    r.StatDex,
		Con:    r.StatCon,
		Cha:    r.StatCha,
	}
	p := game.RestoreCharacterWithStats(r.ID, r.Name, r.Class, r.Race, stats)
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
	// The DB schema's existing room_vnum column is the compatible persistence
	// seam for C's selected load room; no new save-format field is introduced.
	p.SetLoadRoom(r.RoomVNum)
	p.ID = r.ID
	p.Description = r.Description
	if r.Title != "" {
		p.Title = r.Title
	}
	p.Inventory.SetCapacity(p.Stats.Str, p.Stats.StrAdd, p.Stats.Dex, p.Level)

	// Restore inventory — try new SaveItemData format first, fall back to legacy []int.
	if len(r.Inventory) > 0 {
		var invItems []game.SaveItemData
		if err := json.Unmarshal(r.Inventory, &invItems); err == nil {
			for _, item := range invItems {
				if item.VNum == -1 {
					if obj, ok := restorePersistedMail(p, world, item); ok {
						if p.Inventory.RestoreItem(obj) {
							slog.Warn("restored item over inventory capacity",
								"player", p.Name, "vnum", obj.VNum)
						}
					}
					continue
				}
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

	// Everything the columns above do not hold: sex, gold, bank gold,
	// alignment, flags and preferences, skills, affects, practices and the
	// rest of C's char_file_u (DP-1314).
	if err := game.ApplyCharacterData(p, r.CharacterData); err != nil {
		return nil, err
	}
	return p, nil
}

// restorePersistedMail reconstructs the one synthetic inventory object that
// has an established persisted representation. VNum -1 is not sufficient to
// identify mail because other synthetic objects also use it; mail_text is the
// existing state discriminator written by ObjectInstance.GetSaveState.
func restorePersistedMail(p *game.Player, world *game.World, item game.SaveItemData) (*game.ObjectInstance, bool) {
	mailText, ok := item.State["mail_text"].(string)
	if !ok || mailText == "" {
		slog.Warn("skipping persisted synthetic object with invalid mail state",
			"player", p.Name, "vnum", item.VNum, "has_state", item.State != nil)
		return nil, false
	}

	obj := world.CreateMailObject(p, mailText)
	if obj == nil {
		slog.Warn("skipping persisted mail object that could not be constructed",
			"player", p.Name, "vnum", item.VNum)
		return nil, false
	}
	obj.Location = game.LocInventoryPlayer(p.Name)
	return obj, true
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
