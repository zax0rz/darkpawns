// Package game provides write methods for mutating in-memory world data.
// These methods hold the world write lock and modify runtime state only —
// they do NOT persist changes to disk (persistence is a future phase).
package game

import "github.com/zax0rz/darkpawns/pkg/parser"

// SetRoomName updates a room's name. Returns false if the room doesn't exist.
func (w *World) SetRoomName(vnum int, name string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	room.Name = name
	return true
}

// SetRoomDescription updates a room's description. Returns false if the room doesn't exist.
func (w *World) SetRoomDescription(vnum int, desc string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	room.Description = desc
	return true
}

// SetMobShortDesc updates a mob's short description. Returns false if the mob doesn't exist.
func (w *World) SetMobShortDesc(vnum int, desc string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.ShortDesc = desc
	return true
}

// SetMobLongDesc updates a mob's long description. Returns false if the mob doesn't exist.
func (w *World) SetMobLongDesc(vnum int, desc string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.LongDesc = desc
	return true
}

// SetMobLevel updates a mob's level. Returns false if the mob doesn't exist.
func (w *World) SetMobLevel(vnum int, level int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Level = level
	return true
}

// SetMobAC updates a mob's armor class. Returns false if the mob doesn't exist.
func (w *World) SetMobAC(vnum int, ac int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.AC = ac
	return true
}

// SetMobHP updates a mob's hit point dice roll. Returns false if the mob doesn't exist.
func (w *World) SetMobHP(vnum int, numDice, sizeDice, addHP int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.HP.Num = numDice
	mob.HP.Sides = sizeDice
	mob.HP.Plus = addHP
	return true
}

// SetMobGold updates a mob's gold. Returns false if the mob doesn't exist.
func (w *World) SetMobGold(vnum int, gold int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	if gold < 0 { gold = 0 }
	mob.Gold = gold
	return true
}

// SetMobExp updates a mob's experience value. Returns false if the mob doesn't exist.
func (w *World) SetMobExp(vnum int, exp int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	if exp < 0 { exp = 0 }
	mob.Exp = exp
	return true
}

// SetMobAlignment updates a mob's alignment. Returns false if the mob doesn't exist.
func (w *World) SetMobAlignment(vnum int, alignment int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	if alignment < -1000 { alignment = -1000 } else if alignment > 1000 { alignment = 1000 }
	mob.Alignment = alignment
	return true
}

// SetObjShortDesc updates an object's short description. Returns false if the object doesn't exist.
func (w *World) SetObjShortDesc(vnum int, desc string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.ShortDesc = desc
	return true
}

// SetObjLongDesc updates an object's long description. Returns false if the object doesn't exist.
func (w *World) SetObjLongDesc(vnum int, desc string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.LongDesc = desc
	return true
}

// SetObjWeight updates an object's weight. Returns false if the object doesn't exist.
func (w *World) SetObjWeight(vnum int, weight int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	if weight < 0 { weight = 0 }
	obj.Weight = weight
	return true
}

// SetObjCost updates an object's cost. Returns false if the object doesn't exist.
func (w *World) SetObjCost(vnum int, cost int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	if cost < 0 { cost = 0 }
	obj.Cost = cost
	return true
}

// --------------------------------------------------------------------------
// Room write methods
// --------------------------------------------------------------------------

// SetRoomFlags sets a room's flag bitmasks. Returns false if the room doesn't exist.
func (w *World) SetRoomFlags(vnum int, flags []string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	room.Flags = flags
	return true
}

// SetRoomSector sets a room's sector type. Returns false if the room doesn't exist.
func (w *World) SetRoomSector(vnum int, sector int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	room.Sector = sector
	return true
}

// SetRoomExit sets or creates an exit in a room for the given direction.
// Returns false if the room doesn't exist.
func (w *World) SetRoomExit(vnum int, direction string, toRoom int, key int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	exit, exists := room.Exits[direction]
	if !exists {
		exit = parser.Exit{Direction: direction}
	}
	exit.ToRoom = toRoom
	exit.Key = key
	room.Exits[direction] = exit
	return true
}

// SetRoomExtraDescs sets a room's extra descriptions. Returns false if the room doesn't exist.
func (w *World) SetRoomExtraDescs(vnum int, descs []parser.ExtraDesc) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	room, ok := w.rooms[vnum]
	if !ok {
		return false
	}
	room.ExtraDescs = descs
	return true
}

// --------------------------------------------------------------------------
// Mob write methods
// --------------------------------------------------------------------------

// SetMobKeywords updates a mob's keywords. Returns false if the mob doesn't exist.
func (w *World) SetMobKeywords(vnum int, keywords string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Keywords = keywords
	return true
}

// SetMobActionFlags updates a mob's action flags. Returns false if the mob doesn't exist.
func (w *World) SetMobActionFlags(vnum int, flags []string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.ActionFlags = flags
	return true
}

// SetMobAffectFlags updates a mob's affect flags. Returns false if the mob doesn't exist.
func (w *World) SetMobAffectFlags(vnum int, flags []string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.AffectFlags = flags
	return true
}

// SetMobStr updates a mob's strength. Returns false if the mob doesn't exist.
func (w *World) SetMobStr(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Str = val
	return true
}

// SetMobInt updates a mob's intelligence. Returns false if the mob doesn't exist.
func (w *World) SetMobInt(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Int = val
	return true
}

// SetMobWis updates a mob's wisdom. Returns false if the mob doesn't exist.
func (w *World) SetMobWis(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Wis = val
	return true
}

// SetMobDex updates a mob's dexterity. Returns false if the mob doesn't exist.
func (w *World) SetMobDex(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Dex = val
	return true
}

// SetMobCon updates a mob's constitution. Returns false if the mob doesn't exist.
func (w *World) SetMobCon(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Con = val
	return true
}

// SetMobCha updates a mob's charisma. Returns false if the mob doesn't exist.
func (w *World) SetMobCha(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Cha = val
	return true
}

// SetMobTHAC0 updates a mob's THAC0. Returns false if the mob doesn't exist.
func (w *World) SetMobTHAC0(vnum int, val int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.THAC0 = val
	return true
}

// SetMobDamage updates a mob's damage dice roll. Returns false if the mob doesn't exist.
func (w *World) SetMobDamage(vnum int, numDice, sizeDice, addHP int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Damage.Num = numDice
	mob.Damage.Sides = sizeDice
	mob.Damage.Plus = addHP
	return true
}

// SetMobPosition updates a mob's position. Returns false if the mob doesn't exist.
func (w *World) SetMobPosition(vnum int, pos int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Position = pos
	return true
}

// SetMobDefaultPos updates a mob's default position. Returns false if the mob doesn't exist.
func (w *World) SetMobDefaultPos(vnum int, pos int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.DefaultPos = pos
	return true
}

// SetMobSex updates a mob's sex. Returns false if the mob doesn't exist.
func (w *World) SetMobSex(vnum int, sex int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Sex = sex
	return true
}

// SetMobRace updates a mob's race. Returns false if the mob doesn't exist.
func (w *World) SetMobRace(vnum int, race int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok {
		return false
	}
	mob.Race = race
	return true
}

// --------------------------------------------------------------------------
// Object write methods
// --------------------------------------------------------------------------

// SetObjKeywords updates an object's keywords. Returns false if the object doesn't exist.
func (w *World) SetObjKeywords(vnum int, keywords string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.Keywords = keywords
	return true
}

// SetObjTypeFlag updates an object's type flag. Returns false if the object doesn't exist.
func (w *World) SetObjTypeFlag(vnum int, typeFlag int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.TypeFlag = typeFlag
	return true
}

// SetObjValues updates an object's values array. Returns false if the object doesn't exist.
func (w *World) SetObjValues(vnum int, values [4]int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.Values = values
	return true
}

// SetObjWearFlags updates an object's wear flags. Returns false if the object doesn't exist.
func (w *World) SetObjWearFlags(vnum int, flags [4]int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.WearFlags = flags
	return true
}

// SetObjExtraFlags updates an object's extra flags. Returns false if the object doesn't exist.
func (w *World) SetObjExtraFlags(vnum int, flags [4]int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.ExtraFlags = flags
	return true
}

// SetObjAffects updates an object's affects. Returns false if the object doesn't exist.
func (w *World) SetObjAffects(vnum int, affects []parser.ObjAffect) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.Affects = affects
	return true
}

// SetObjExtraDescs sets an object's extra descriptions. Returns false if the object doesn't exist.
func (w *World) SetObjExtraDescs(vnum int, descs []parser.ExtraDesc) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok {
		return false
	}
	obj.ExtraDescs = descs
	return true
}

// --------------------------------------------------------------------------
// Shop write methods
// --------------------------------------------------------------------------

// SetShopBuyTypes sets the buy types for the shop run by the given keeper NPC.
// Returns false if no shop exists for that keeper.
func (w *World) SetShopBuyTypes(keeperVNum int, buyTypes []int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.BuyTypes = buyTypes
	return true
}

// SetShopSellTypes sets the sell types for the shop run by the given keeper NPC.
// Returns false if no shop exists for that keeper.
func (w *World) SetShopSellTypes(keeperVNum int, sellTypes []int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.SellTypes = sellTypes
	return true
}

// SetShopProfit sets the buy and sell profit multipliers for the shop run by
// the given keeper NPC. Returns false if no shop exists for that keeper.
func (w *World) SetShopProfit(keeperVNum int, buyProfit, sellProfit float64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	shop := sm.GetShopByKeeper(keeperVNum)
	if shop == nil {
		return false
	}
	shop.ProfitBuy = buyProfit
	shop.ProfitSell = sellProfit
	return true
}
