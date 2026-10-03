package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Mob equipment always uses C WEAR_* indices (src/structs.h:390-412),
// including ObjectLocation.Slot. Player EquipmentSlot constants are separate.
const (
	mobWearLight = 0
	mobWearBody  = 5
	mobWearHead  = 6
	mobWearLegs  = 7
	mobWearWield = 16
)

func (m *MobInstance) Equipped(position int) *ObjectInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Equipment[position]
}

func (m *MobInstance) EquipmentSnapshot() map[int]*ObjectInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[int]*ObjectInstance, len(m.Equipment))
	for slot, obj := range m.Equipment {
		out[slot] = obj
	}
	return out
}

// mobileEquipmentEffects carries only the output/combat work that must run
// after World and Mob locks are released. No manager/lifecycle lock is taken.
type mobileEquipmentEffects struct {
	zapped bool
	bad    byte
}

// equipMobileLocked is equip_char's mobile path. Caller holds World.mu when
// world is nonnil and Mob.mu. The object must already be detached by its caller.
func (m *MobInstance) equipMobileLocked(w *World, obj *ObjectInstance, pos int) (mobileEquipmentEffects, error) {
	var effects mobileEquipmentEffects
	if pos < 0 || pos >= NumWears {
		return effects, fmt.Errorf("invalid mobile wear position %d", pos)
	}
	if m.Equipment[pos] != nil {
		return effects, nil
	}
	if obj.Location.Kind != ObjNowhere {
		return effects, fmt.Errorf("mobile equip object is not floating")
	}
	flags := obj.GetExtraFlags()[0]
	alignment := m.alignmentLocked()
	if flags&FlagAntiEvil != 0 && alignment <= -350 || flags&FlagAntiGood != 0 && alignment >= 350 || flags&FlagAntiNeutral != 0 && alignment > -350 && alignment < 350 {
		obj.Location = LocInventoryMob(m.ID)
		m.Inventory = append([]*ObjectInstance{obj}, m.Inventory...)
		effects.zapped = true
		return effects, nil
	}
	// invalid_class is always false for NPCs (class.c:737-764; utils.h:572-598).
	if obj.HasExtraFlag(0, extraFlagTakeName) {
		obj.Runtime.ShortDescOverride = m.GetShortDesc() + "'s " + obj.GetKeywords()
	}
	if m.Equipment == nil {
		m.Equipment = make(map[int]*ObjectInstance)
	}
	m.Equipment[pos] = obj
	obj.Location = LocEquippedMob(m.ID, EquipmentSlot(pos))
	if obj.GetTypeFlag() == ITEM_ARMOR {
		m.changeMobileACLocked(-mobileArmor(obj, pos))
	}
	if w != nil && isLitLightSource(obj) {
		w.adjustRoomLight(m.RoomVNum, 1)
	}
	m.modifyMobileEquipmentLocked(obj, true)
	m.affectTotalLocked()
	stats := m.effectiveAttributesLocked()
	if stats.Str == 0 {
		effects.bad = 's'
	}
	if stats.Int == 0 {
		effects.bad = 'i'
	}
	if stats.Wis == 0 {
		effects.bad = 'w'
	}
	if stats.Cha == 0 {
		effects.bad = 'c'
	}
	if stats.Dex == 0 {
		effects.bad = 'd'
	}
	return effects, nil
}

// unequipMobileLocked is unequip_char: return a floating object, without
// implicitly putting it into inventory (handler.c:754-795).
func (m *MobInstance) unequipMobileLocked(w *World, pos int) *ObjectInstance {
	obj := m.Equipment[pos]
	if obj == nil {
		return nil
	}
	if obj.HasExtraFlag(0, extraFlagTakeName) {
		article := "a"
		if name := obj.GetKeywords(); name != "" && strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
			article = "an"
		}
		obj.Runtime.ShortDescOverride = article + " " + obj.GetKeywords()
	}
	if obj.GetTypeFlag() == ITEM_ARMOR {
		m.changeMobileACLocked(mobileArmor(obj, pos))
	}
	if w != nil && isLitLightSource(obj) {
		w.adjustRoomLight(m.RoomVNum, -1)
	}
	delete(m.Equipment, pos)
	obj.Location = LocNowhere()
	m.modifyMobileEquipmentLocked(obj, false)
	m.affectTotalLocked()
	return obj
}

func mobileArmor(obj *ObjectInstance, pos int) int {
	factor := 1
	switch pos {
	case mobWearBody:
		factor = 3
	case mobWearHead, mobWearLegs:
		factor = 2
	}
	return factor * obj.GetValue(0)
}

func (m *MobInstance) changeMobileACLocked(delta int) {
	ac := 0
	if m.Runtime.ACOverride != nil {
		ac = *m.Runtime.ACOverride
	} else if p := m.Proto(); p != nil {
		ac = p.AC
	}
	ac = mobileSignedPoint(ac+delta, 16)
	m.Runtime.ACOverride = &ac
}

// Non-attribute points are incremental C values. A later Lua/set overwrite is
// effective, not a new base; unequip subtracts its modifier from that value.
// Attribute bounds remain in the shared affect_total recomputation.
func (m *MobInstance) modifyMobileEquipmentLocked(obj *ObjectInstance, add bool) {
	mask := uint64(obj.AffectFlags[0]) | uint64(obj.AffectFlags[1])<<32
	if add {
		m.Affects |= mask
	} else {
		m.Affects &^= mask
	}
	for _, af := range obj.GetAffects() {
		mod := mobileSignedPoint(af.Modifier, 8)
		if !add {
			mod = mobileSignedPoint(-mod, 8)
		}
		switch af.Location {
		case 9:
			m.BirthTime = m.BirthTime.Add(-time.Duration(mod*SECS_PER_MUD_YEAR) * time.Second)
		case 10:
			m.Weight = (m.Weight + mod) & 255
		case 11:
			m.Height = (m.Height + mod) & 255
		case 12:
			m.MaxMana = mobileSignedPoint(m.MaxMana+mod, 16)
		case 13:
			m.MaxHP = mobileSignedPoint(m.MaxHP+mod, 16)
		case 14:
			m.MaxMove = mobileSignedPoint(m.MaxMove+mod, 16)
		case 17:
			m.changeMobileACLocked(mod)
		case 18:
			hit := 0
			if m.Runtime.HitrollOverride != nil {
				hit = *m.Runtime.HitrollOverride
			} else if p := m.Proto(); p != nil {
				hit = 20 - p.THAC0
			}
			hit = mobileSignedPoint(hit+mod, 8)
			m.Runtime.HitrollOverride = &hit
		case 19:
			dam := mobileSignedPoint(m.damrollPointLocked()+mod, 8)
			m.Runtime.DamrollOverride = &dam
			m.Runtime.DamrollBonus = 0
		case 20, 21, 22, 23, 24:
			m.SavingThrows[af.Location-20] = mobileSignedPoint(m.SavingThrows[af.Location-20]+mod, 16)
		case 25:
			m.modifyEquipmentRaceHateLocked(mod)
		}
	}
}

func (w *World) finishMobileEquipment(m *MobInstance, obj *ObjectInstance, effects mobileEquipmentEffects) {
	if effects.zapped {
		w.mobileEquipmentAct(m, obj, "You are zapped by $p and instantly let go of it.")
		Act(w, false, m, nil, obj, nil, "$n is zapped by $p and instantly lets go of it.", "", ToRoom)
		return
	}
	var text string
	switch effects.bad {
	case 's':
		text = "You are too weak to fight!\n\r"
	case 'i', 'w':
		text = "You are too dumb to do much of anything!\n\r"
	case 'c':
		text = "The world hates you!\n\r"
	case 'd':
		text = "You trip over your own feet and hit your head!\n\r"
	}
	if text != "" && w.MobileMessageSink != nil {
		w.MobileMessageSink(m, []byte(text))
	}
	switch effects.bad {
	case 'c':
		var pest *MobInstance
		for _, candidate := range w.GetAllMobs() {
			if candidate.GetVNum() == 7907 && (pest == nil || candidate.ID > pest.ID) {
				pest = candidate
			}
		}
		if pest != nil && pest.GetHunting() == "" {
			pest.SetHunting(m.GetName())
			pest.mu.Lock()
			pest.HuntingMobID = m.ID
			pest.mu.Unlock()
		}
	case 'd':
		w.roomActivitySelfDamage(m, 40, combat.TYPE_SUFFERING)
	}
}

// EquipMobileObject is the floating-object equip_char boundary. World lock
// precedes Mob lock; callback/Act/self-damage work follows both releases.
func (w *World) EquipMobileObject(m *MobInstance, obj *ObjectInstance, pos int) error {
	w.mu.Lock()
	m.mu.Lock()
	effects, err := m.equipMobileLocked(w, obj, pos)
	m.mu.Unlock()
	w.mu.Unlock()
	if err == nil {
		w.finishMobileEquipment(m, obj, effects)
	}
	return err
}

func (m *MobInstance) modifyEquipmentRaceHateLocked(mod int) {
	if mod <= 0 {
		for i, v := range m.RaceHates {
			if v == -mod {
				m.RaceHates[i] = -1
				return
			}
		}
		if mod < 0 {
			return
		}
	}
	for i, v := range m.RaceHates {
		if v == -1 {
			m.RaceHates[i] = mod
			return
		}
	}
}

// C affect_total strips and reapplies every equipment modifier in WEAR order.
// This matters even for points: signed-byte -128 negates back to -128 in C.
func (m *MobInstance) refreshMobileEquipmentFlagsLocked() {
	for pos := 0; pos < NumWears; pos++ {
		if obj := m.Equipment[pos]; obj != nil {
			m.modifyMobileEquipmentLocked(obj, false)
		}
	}
	for pos := 0; pos < NumWears; pos++ {
		if obj := m.Equipment[pos]; obj != nil {
			m.modifyMobileEquipmentLocked(obj, true)
		}
	}
}

func (w *World) equipMobileFromInventory(m *MobInstance, obj *ObjectInstance, pos int) bool {
	w.mu.Lock()
	if obj.Location.Kind == ObjInInventory {
		if _, err := w.detachObjectLocked(obj); err != nil {
			w.mu.Unlock()
			return false
		}
		obj.Location = LocNowhere()
	}
	m.mu.Lock()
	effects, err := m.equipMobileLocked(w, obj, pos)
	equipped := m.Equipment[pos] == obj
	m.mu.Unlock()
	w.mu.Unlock()
	if err == nil {
		w.finishMobileEquipment(m, obj, effects)
	}
	return err == nil && equipped
}

// C points use signed bytes/shorts; preserve assignment wrap without an
// unchecked narrowing cast (src/structs.h:903-917,935; R1a).
func mobileSignedPoint(value, width int) int {
	mask := (1 << uint(width)) - 1
	value &= mask
	if value&(1<<uint(width-1)) != 0 {
		value -= mask + 1
	}
	return value
}

// dropMobilePossessionsLocked is extract_char_final's inventory-first then
// ascending-WEAR order. World.mu precedes Mob.mu; obj_to_room changes no light.
func (w *World) dropMobilePossessionsLocked(m *MobInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	drop := func(obj *ObjectInstance) {
		if m.RoomVNum >= 0 {
			w.roomItems[m.RoomVNum] = append([]*ObjectInstance{obj}, w.roomItems[m.RoomVNum]...)
			obj.Location = LocRoom(m.RoomVNum)
			obj.RoomVNum = m.RoomVNum
		} else {
			obj.Location = LocNowhere()
			obj.RoomVNum = -1
		}
	}
	for _, obj := range m.Inventory {
		drop(obj)
	}
	m.Inventory = nil
	m.Flags &^= 1 << uint(MobFlagExtract)
	m.SetAlive(false)
	for slot := 0; slot < NumWears; slot++ {
		if obj := m.unequipMobileLocked(w, slot); obj != nil {
			drop(obj)
		}
	}
}

// mobileEquipmentAct preserves act's position gate, capitalization and CRLF,
// delivering only to the descriptor of the concrete NPC body.
func (w *World) mobileEquipmentAct(m *MobInstance, obj *ObjectInstance, format string) {
	actDeliver(w, false, m, nil, obj, nil, format, "", ToChar, func(_ Actor, line string) {
		if w.MobileMessageSink != nil {
			w.MobileMessageSink(m, []byte(line))
		}
	})
}
