// Package game provides write methods for mutating in-memory world data.
// These methods hold the world write lock and modify runtime state only —
// they do NOT persist changes to disk (persistence is a future phase).
package game

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
