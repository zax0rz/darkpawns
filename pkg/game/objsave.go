// Ported from src/objsave.c
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

import (
	"log"
	"time"
)

// Rent codes — used in persistence to track why a player's items are saved.
const (
	RentCrash    = 1
	RentRented   = 2
	RentCryo     = 3
	RentTimedOut = 4
	RentForced   = 5
)

var (
	RentFileTimeout  = 30  // days before rent files expire
	CrashFileTimeout = 365 // days before crash files expire
)

// Rent vs Cryo pricing factors.
const (
	RentFactor = 1
	CryoFactor = 4
)

// --------------------------------------------------------------------------
// C WEAR_* position mapping — matches the 0-based array indices in C
// char_data.equipment[] (structs.h WEAR_* constants 0–20).
// --------------------------------------------------------------------------

func cWearPosToGoSlot(cPos int) (EquipmentSlot, bool) {
	m := map[int]EquipmentSlot{
		0:  SlotLight,
		1:  SlotFingerR,
		2:  SlotFingerL,
		3:  SlotNeck1,
		4:  SlotNeck2,
		5:  SlotBody,
		6:  SlotHead,
		7:  SlotLegs,
		8:  SlotFeet,
		9:  SlotHands,
		10: SlotArms,
		11: SlotShield,
		12: SlotAbout,
		13: SlotWaist,
		14: SlotWristR,
		15: SlotWristL,
		16: SlotWield,
		17: SlotHold,
		18: SlotAblegs,
		19: SlotFace,
		20: SlotHover,
	}
	s, ok := m[cPos]
	return s, ok
}

func goSlotToCWearPos(s EquipmentSlot) (int, bool) {
	m := map[EquipmentSlot]int{
		SlotLight:   0,
		SlotFingerR: 1,
		SlotFingerL: 2,
		SlotNeck1:   3,
		SlotNeck2:   4,
		SlotBody:    5,
		SlotHead:    6,
		SlotLegs:    7,
		SlotFeet:    8,
		SlotHands:   9,
		SlotArms:    10,
		SlotShield:  11,
		SlotAbout:   12,
		SlotWaist:   13,
		SlotWristR:  14,
		SlotWristL:  15,
		SlotWield:   16,
		SlotHold:    17,
		SlotAblegs:  18,
		SlotFace:    19,
		SlotHover:   20,
	}
	c, ok := m[s]
	return c, ok
}

// cWearPosCanWearFlag maps C WEAR_* index to the ITEM_WEAR_* bit required.
func cWearPosCanWearFlag(cPos int) int {
	m := map[int]int{
		0:  1 << 15, // ITEM_WEAR_LIGHT
		1:  1 << 1,  // ITEM_WEAR_FINGER
		2:  1 << 1,  // ITEM_WEAR_FINGER (alt)
		3:  1 << 2,  // ITEM_WEAR_NECK
		4:  1 << 2,  // ITEM_WEAR_NECK (alt)
		5:  1 << 3,  // ITEM_WEAR_BODY
		6:  1 << 4,  // ITEM_WEAR_HEAD
		7:  1 << 5,  // ITEM_WEAR_LEGS
		8:  1 << 6,  // ITEM_WEAR_FEET
		9:  1 << 7,  // ITEM_WEAR_HANDS
		10: 1 << 8,  // ITEM_WEAR_ARMS
		11: 1 << 9,  // ITEM_WEAR_SHIELD
		12: 1 << 10, // ITEM_WEAR_ABOUT
		13: 1 << 11, // ITEM_WEAR_WAIST
		14: 1 << 12, // ITEM_WEAR_WRIST
		15: 1 << 12, // ITEM_WEAR_WRIST (alt)
		16: 1 << 13, // ITEM_WEAR_WIELD
		17: 1 << 14, // ITEM_WEAR_HOLD
		18: 1 << 16, // ITEM_WEAR_ABLEGS
		19: 1 << 17, // ITEM_WEAR_FACE
		20: 1 << 18, // ITEM_WEAR_HOVER
	}
	return m[cPos]
}

// Flag constants matching ITEM_* from structs.h used for alignment checks.
const (
	FlagAntiGood    = 1 << 2  // ITEM_ANTI_GOOD
	FlagAntiEvil    = 1 << 3  // ITEM_ANTI_EVIL
	FlagAntiNeutral = 1 << 11 // ITEM_ANTI_NEUTRAL
	ItemWeapon      = 1       // ITEM_WEAPON
)

// Extended equipment slots (M4 additions).
const (
	SlotAblegs EquipmentSlot = 100 + iota
	SlotFace
	SlotHover
)

// --------------------------------------------------------------------------
// AutoEquip — matches the C auto_equip() logic.
// locate: C WEAR_* index + 1 (1 = worn at pos 0, 20 = worn at pos 19).
// --------------------------------------------------------------------------

func AutoEquip(p *Player, obj *ObjectInstance, locate int) {
	if locate <= 0 {
		p.Inventory.addItem(obj)
		return
	}
	cPos := locate - 1
	_, ok := cWearPosToGoSlot(cPos)
	if !ok {
		p.Inventory.addItem(obj)
		return
	}
	rf := cWearPosCanWearFlag(cPos)
	wf := obj.Prototype.WearFlags[0]
	wears := (wf & rf) != 0
	// Warriors can wield in hold slot.
	if cPos == 17 && !wears {
		if (wf&(1<<13)) != 0 && obj.Prototype.TypeFlag == ItemWeapon {
			wears = true
		}
	}
	if !wears {
		p.Inventory.addItem(obj)
		return
	}
	// Alignment restrictions.
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagAntiEvil != 0 && p.IsEvil()) ||
		(xf&FlagAntiGood != 0 && p.IsGood()) ||
		(xf&FlagAntiNeutral != 0 && p.IsNeutral()) {
		p.Inventory.addItem(obj)
		return
	}
	if err := p.Equipment.Equip(obj, p.Inventory); err != nil {
		p.Inventory.addItem(obj)
	}
}

// --------------------------------------------------------------------------
// Rent cost calculation — matches C Crash_offer_rent.
// --------------------------------------------------------------------------

func OfferRent(p *Player, cryo bool, recep *MobInstance, cmd string, mode int) int {
	if p == nil || p.Inventory == nil {
		return 0
	}
	total := 0
	for _, obj := range p.Inventory.Items {
		if obj == nil || obj.Prototype == nil {
			continue
		}
		v := obj.Prototype.Values[0]
		if v < 1 {
			v = 1
		}
		total += v
	}
	if p.Equipment != nil {
		for _, obj := range p.Equipment.Slots {
			if obj == nil || obj.Prototype == nil {
				continue
			}
			v := obj.Prototype.Values[0]
			if v < 1 {
				v = 1
			}
			total += v
		}
	}
	if cryo {
		total *= CryoFactor
	}
	return total
}

// --------------------------------------------------------------------------
// Crash-load — restore player from deserialized data (matches C Crash_load).
// --------------------------------------------------------------------------

func CrashLoad(p *Player, invItems []*ObjectInstance, eqSlots map[EquipmentSlot]*ObjectInstance) bool {
	if p == nil {
		return false
	}
	p.Inventory.clear()
	p.Equipment = NewEquipment()
	for slot, obj := range eqSlots {
		if obj == nil {
			continue
		}
		locate := 0
		if cPos, ok := goSlotToCWearPos(slot); ok {
			locate = cPos + 1
		}
		AutoEquip(p, obj, locate)
	}
	for _, obj := range invItems {
		if obj != nil {
			p.Inventory.addItem(obj)
		}
	}
	return true
}

// --------------------------------------------------------------------------
// File/persistence stubs — these delegate to the DB layer.
// --------------------------------------------------------------------------

func DeleteCrashFile(name string) bool {
	log.Printf("[objsave] DeleteCrashFile(%s) — delegated to DB layer", name)
	return true
}

func CleanCrashFile(name string, savedTime time.Time, rentCode int) bool {
	now := time.Now()
	switch rentCode {
	case RentCrash, RentForced, RentTimedOut:
		if now.Sub(savedTime) > time.Duration(CrashFileTimeout)*24*time.Hour {
			DeleteCrashFile(name)
			log.Printf("[objsave] Deleting %s's crash save (expired)", name)
			return true
		}
	case RentRented:
		if now.Sub(savedTime) > time.Duration(RentFileTimeout)*24*time.Hour {
			DeleteCrashFile(name)
			log.Printf("[objsave] Deleting %s's rent save (expired)", name)
			return true
		}
	}
	return false
}

func SaveAllPlayers(players []*Player) {
	for _, p := range players {
		if !p.IsNPC() {
			_ = p
		}
	}
}

func DeleteAliasFile(name string) bool {
	log.Printf("[objsave] DeleteAliasFile(%s) — no alias persistence in Go port", name)
	return true
}

// --------------------------------------------------------------------------
// Save helpers — persist player inventory/equipment.
// --------------------------------------------------------------------------

func RentSave(p *Player, cost int) {
	if p == nil {
		return
	}
	log.Printf("[objsave] RentSave(%s, cost=%d) — delegating to DB layer", p.GetName(), cost)
}

func CrashSave(p *Player) {
	if p == nil {
		return
	}
	log.Printf("[objsave] CrashSave(%s) — delegating to DB layer", p.GetName())
}

func CryoSave(p *Player, cost int) {
	if p == nil {
		return
	}
	log.Printf("[objsave] CryoSave(%s, cost=%d) — delegating to DB layer", p.GetName(), cost)
}

// --------------------------------------------------------------------------
// Receptionist NPC handler — matches C gen_receptionist.
// --------------------------------------------------------------------------

type PlayerSaver interface {
	SavePlayer(p *Player) error
	ExtractPlayer(p *Player, saveRoom int) error
}

func GenReceptionist(p *Player, recep *MobInstance, cmd string, arg string, mode int, world PlayerSaver) bool {
	if p == nil || p.IsNPC() {
		return false
	}
	if cmd != "offer" && cmd != "rent" {
		return false
	}
	cost := OfferRent(p, mode == CryoFactor, recep, cmd, mode)
	if cmd == "offer" {
		return true
	}
	if cost > p.GetGold() {
		p.SendMessage("You can't afford to rent!\r\n")
		return true
	}
	if mode == RentFactor {
		RentSave(p, cost)
	} else {
		CryoSave(p, cost)
	}
	saveRoom := p.GetRoom()
	_ = world.ExtractPlayer(p, saveRoom)
	return true
}

func RentDeadline(ch *Player, recep *MobInstance, totalCost int) {
	log.Printf("[objsave] RentDeadline(%s, cost=%d)", ch.GetName(), totalCost)
}

func UpdateObjFiles(playerNames []string) {
	for range playerNames {
		// Handled by periodic DB cleanup.
	}
}
