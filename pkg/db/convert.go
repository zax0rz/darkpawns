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
			restoreSavedItems(p, world, invItems, false)
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
			restoreSavedItems(p, world, eqItems, true)
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

// inventorySaveData returns the save list for everything carried, each
// container followed by its contents (see appendSaveTree).
func inventorySaveData(inv *game.Inventory) []game.SaveItemData {
	result := make([]game.SaveItemData, 0)
	if inv == nil {
		return result
	}
	for _, item := range inv.FindItems("") {
		appendSaveTree(&result, item, 0, 0)
	}
	return result
}

// equipmentSaveData returns the save list for everything worn, preserving
// each item's slot, each container followed by its contents.
func equipmentSaveData(eq *game.Equipment) []game.SaveItemData {
	result := make([]game.SaveItemData, 0)
	if eq == nil {
		return result
	}
	for slot, item := range eq.GetEquippedItems() {
		cPos, ok := game.SlotToCWearPos(slot)
		locate := 0
		if ok {
			locate = cPos + 1
		}
		appendSaveTree(&result, item, locate, 0)
	}
	return result
}

// appendSaveTree appends obj and then, depth first, everything inside it. C's
// Crash_save writes a container's contents with the container (objsave.c),
// so a rented bag comes back full. ContainerIndex names the containing
// object by its 1-based position in the same list; 0 means carried or worn
// directly. Contents keep the container's own order.
func appendSaveTree(out *[]game.SaveItemData, obj *game.ObjectInstance, locate, parent int) {
	item := game.SaveItemData{
		VNum:           obj.GetVNum(),
		Count:          1,
		Locate:         locate,
		State:          obj.GetSaveState(),
		ContainerIndex: parent,
	}
	if parent > 0 {
		item.ContainerVNum = (*out)[parent-1].VNum
	}
	*out = append(*out, item)
	self := len(*out)
	for _, contained := range obj.Contains {
		appendSaveTree(out, contained, 0, self)
	}
}

// restoreSavedItems rebuilds one saved list (inventory or equipment). An item
// whose container is listed before it goes back inside that container; one
// whose container could not be rebuilt is carried instead, as C's Crash_load
// does with the contents of a lost container. Objects are created through the
// world so they have IDs: moving anything into or out of a restored container
// resolves the container by ID.
func restoreSavedItems(p *game.Player, world *game.World, items []game.SaveItemData, equipped bool) {
	restored := make([]*game.ObjectInstance, len(items))
	for i, item := range items {
		obj := restoreSavedObject(p, world, item)
		if obj == nil {
			continue
		}
		restored[i] = obj
		if parent := item.ContainerIndex - 1; parent >= 0 && parent < i && restored[parent] != nil {
			container := restored[parent]
			container.Contains = append(container.Contains, obj)
			obj.Location = game.LocContainer(container.ID)
			continue
		}
		if equipped && item.ContainerIndex == 0 {
			if slot, ok := game.CWearPosToSlot(item.Locate - 1); ok {
				obj.Location = game.LocEquippedPlayer(p.Name, slot)
				p.Equipment.Slots[slot] = obj
				continue
			}
		}
		obj.Location = game.LocInventoryPlayer(p.Name)
		if p.Inventory.RestoreItem(obj) {
			slog.Warn("restored item over inventory capacity",
				"player", p.Name, "vnum", obj.VNum)
		}
	}
}

// restoreSavedObject rebuilds one saved object, or returns nil when it can no
// longer be built (its prototype is gone, or its synthetic state is invalid).
func restoreSavedObject(p *game.Player, world *game.World, item game.SaveItemData) *game.ObjectInstance {
	if item.VNum == -1 {
		obj, ok := restorePersistedMail(p, world, item)
		if !ok {
			return nil
		}
		return obj
	}
	obj, err := world.SpawnObject(item.VNum, -1)
	if err != nil {
		return nil
	}
	for k, v := range item.State {
		obj.CustomData[k] = v
	}
	if item.State != nil {
		obj.MigrateCustomData()
	}
	return obj
}
