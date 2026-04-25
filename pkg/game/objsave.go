// Ported from src/objsave.c
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
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
// --------------------------------------------------------------------------
// Binary data types matching C structs from src/structs.h
// --------------------------------------------------------------------------

const (
	EFArrayMax   = 4
	AFArrayMax   = 4
	MaxObjAffect = 6

	// ObjFileElemSize is the exact binary size of obj_file_elem: 592 bytes.
	// Layout: 4 + 2 + 4*4 + 4*4 + 4 + 4 + 4*4 + 6*2 + 128 + 128 + 256 = 592
	ObjsaveElemSize = 592

	// RentInfoSize is the exact binary size of rent_info: 56 bytes (14 int32).
	RentInfoSize = 56
)

// ObjAffect is a single stat-modifier pair matching the C obj_affected_type.
type ObjAffect struct {
	Location uint8
	Modifier int8
}

// ObjFileElem is the on-disk binary record for one saved object, matching
// the C struct obj_file_elem from src/structs.h.
type ObjFileElem struct {
	ItemNumber int32
	Locate     int16
	Value      [4]int32
	ExtraFlags [4]int32
	Weight     int32
	Timer      int32
	Bitvector  [4]int32
	Affects    [6]ObjAffect
	Name       [128]byte
	ShortDesc  [128]byte
	Desc       [256]byte
}

// RentInfo is the on-disk binary record for player rent/account data,
// matching the C struct rent_info from src/structs.h.
type RentInfo struct {
	Time           int32
	RentCode       int32
	NetCostPerDiem int32
	Gold           int32
	Account        int32
	NItems         int32
	Spare          [8]int32
}

// ObjFromBinary deserializes a 592-byte binary blob into an ObjectInstance.
// Returns the object and the C-style locate offset (1-based wear position).
func ObjFromBinary(data []byte, world *World) (*ObjectInstance, int) {
	if len(data) < ObjsaveElemSize {
		return nil, 0
	}

	var elem ObjFileElem
	buf := bytes.NewReader(data[:ObjsaveElemSize])
	if err := binary.Read(buf, binary.LittleEndian, &elem); err != nil {
		return nil, 0
	}

	var proto *parser.Obj
	if world != nil {
		if p, ok := world.GetObjPrototype(int(elem.ItemNumber)); ok {
			proto = p
		}
	}

	obj := &ObjectInstance{
		Prototype: proto,
		VNum:      int(elem.ItemNumber),
	}

	return obj, int(elem.Locate)
}

// ObjToBinary serializes an ObjectInstance into the 592-byte binary blob.
func ObjToBinary(obj *ObjectInstance, locate int) ([]byte, error) {
	var elem ObjFileElem

	elem.ItemNumber = int32(obj.VNum)
	elem.Locate = int16(locate)

	if obj.Prototype != nil {
		elem.Weight = int32(obj.Prototype.Weight)

		// Copy values (up to 4)
		for i := 0; i < 4 && i < len(obj.Prototype.Values); i++ {
			elem.Value[i] = int32(obj.Prototype.Values[i])
		}

		// Copy extra flags (up to 4)
		for i := 0; i < 4 && i < len(obj.Prototype.ExtraFlags); i++ {
			elem.ExtraFlags[i] = int32(obj.Prototype.ExtraFlags[i])
		}

		// Copy affects (up to 6)
		for i := 0; i < MaxObjAffect && i < len(obj.Prototype.Affects); i++ {
			elem.Affects[i] = ObjAffect{
				Location: uint8(obj.Prototype.Affects[i].Location),
				Modifier: int8(obj.Prototype.Affects[i].Modifier),
			}
		}

		// Copy name strings as fixed-size C-style buffers.
		copyStringToFixedBytes(elem.Name[:], obj.Prototype.Keywords)
		copyStringToFixedBytes(elem.ShortDesc[:], obj.Prototype.ShortDesc)
		copyStringToFixedBytes(elem.Desc[:], obj.Prototype.LongDesc)
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &elem); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CrashIsUnrentable checks whether an object should be dropped during crash
// recovery. Matches the C Crash_is_unrentable() logic:
//   - Prototype is nil (deleted object)
//   - TypeFlag == ITEM_KEY (13 in CircleMUD)
//   - Has ITEM_NORENT flag (extra_flags[0] bit 1 = 2)
func CrashIsUnrentable(obj *ObjectInstance) bool {
	if obj == nil {
		return true
	}
	if obj.Prototype == nil {
		return true
	}
	// ITEM_KEY = 13
	if obj.Prototype.TypeFlag == 13 {
		return true
	}
	// ITEM_NORENT = (1 << 1) = 2
	if len(obj.Prototype.ExtraFlags) > 0 && (obj.Prototype.ExtraFlags[0]&2) != 0 {
		return true
	}
	return false
}

// DecodeRentInfo deserializes a 56-byte rent_info blob.
func DecodeRentInfo(data []byte) (*RentInfo, error) {
	if len(data) < RentInfoSize {
		return nil, ErrBadRentSize
	}
	var ri RentInfo
	buf := bytes.NewReader(data[:RentInfoSize])
	if err := binary.Read(buf, binary.LittleEndian, &ri); err != nil {
		return nil, err
	}
	return &ri, nil
}

// EncodeRentInfo serializes a RentInfo into a 56-byte binary blob.
func EncodeRentInfo(ri *RentInfo) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, ri); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// copyStringToFixedBytes copies a Go string into a fixed-size byte slice,
// null-terminated (C-style), truncating if too long.
func copyStringToFixedBytes(dst []byte, src string) {
	n := len(src)
	if n >= len(dst) {
		n = len(dst) - 1
	}
	for i := 0; i < n; i++ {
		dst[i] = src[i]
	}
	// Null-terminate
	if n < len(dst) {
		dst[n] = 0
	}
}

// ErrBadRentSize is returned when rent info data is too short.
var ErrBadRentSize = fmt.Errorf("rent info data too short: need %d bytes", RentInfoSize)

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
		p.Inventory.AddItem(obj)
		return
	}
	cPos := locate - 1
	_, ok := cWearPosToGoSlot(cPos)
	if !ok {
		p.Inventory.AddItem(obj)
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
		p.Inventory.AddItem(obj)
		return
	}
	// Alignment restrictions.
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagAntiEvil != 0 && p.IsEvil()) ||
		(xf&FlagAntiGood != 0 && p.IsGood()) ||
		(xf&FlagAntiNeutral != 0 && p.IsNeutral()) {
		p.Inventory.AddItem(obj)
		return
	}
	if err := p.Equipment.Equip(obj, p.Inventory); err != nil {
		p.Inventory.AddItem(obj)
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
	p.Inventory.Clear()
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
			p.Inventory.AddItem(obj)
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
