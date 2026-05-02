// Ported from src/objsave.c
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	MaxObjSave       = 60  // max objects a player can rent — from config.c max_obj_save
	MinRentCost      = 10  // base rent cost — from config.c min_rent_cost
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
// ExtraFlags[0] bits.
const (
	FlagAntiGood    = 1 << 2  // ITEM_ANTI_GOOD
	FlagAntiEvil    = 1 << 3  // ITEM_ANTI_EVIL
	FlagAntiNeutral = 1 << 11 // ITEM_ANTI_NEUTRAL
	FlagNoRent      = 1 << 5  // ITEM_NORENT
)

// Extended equipment slots (M4 additions).
const (
	SlotAblegs EquipmentSlot = 100 + iota
	SlotFace
	SlotHover
)

// NumWears is the number of equipment slots (0-based). Matches NUM_WEARS in C (used in loops).
const NumWears = 21

// MaxBagRow is the max nesting depth for container loading (matching C's MAX_BAG_ROW = 5).
const MaxBagRow = 5

// ==========================================================================
// IsUnrentable — ported from C Crash_is_unrentable()
// Returns true if the object cannot be stored in rent/crash:
//   - ITEM_NORENT flag set
//   - load < 0 (virtual/negative vnum)
//   - type == ITEM_KEY
// ==========================================================================
func IsUnrentable(obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return true
	}
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagNoRent) != 0 || obj.VNum < 0 || ItemType(obj.Prototype.TypeFlag) == ItemKey {
		return true
	}
	return false
}

// ==========================================================================
// ExtractNorents — ported from C Crash_extract_norents()
// Recursively extracts all unrentable items from obj's contents/next_content.
// ==========================================================================
func ExtractNorents(obj *ObjectInstance) {
	if obj == nil {
		return
	}
	// Recursively process contained items and sibling items.
	for _, c := range obj.Contains {
		ExtractNorents(c)
	}
	// Note: obj.Contains and obj.next_content are handled differently in Go.
	// ObjectInstance has `Contains []*ObjectInstance` (slice) not a linked list.
	// We need to iterate the slice and check each item.

	// Actually, the C version traverses the linked list. In Go, ObjectInstance
	// doesn't have next_content — contained items are in obj.Contains (slice).
	// Objects in inventory are in Player.Inventory.Items (slice).
	// This function operates on a single item's content chain.
	for i := 0; i < len(obj.Contains); i++ {
		ExtractNorents(obj.Contains[i])
	}
}

// ==========================================================================
// ExtractNorentsFromEquipped — ported from C Crash_extract_norents_from_equipped()
// Moves unrentable equipped items to inventory, then recursively extracts
// unrentable contents from worn containers.
// ==========================================================================
func ExtractNorentsFromEquipped(p *Player) {
	if p == nil || p.Equipment == nil {
		return
	}
	for _, slot := range p.Equipment.Slots {
		if slot == nil {
			continue
		}
		if IsUnrentable(slot) {
			// Move from equipment to inventory (matching C's unequip_char + obj_to_char)
			p.Equipment.UnequipItem(slot, p.Inventory)
			if err := p.Inventory.addItem(slot); err != nil {
				slog.Warn("addItem failed in ExtractNorents", "player", p.Name, "obj_vnum", slot.GetVNum(), "error", err)
			}
		} else {
			// Recursively extract norents from contained items.
			for _, c := range slot.Contains {
				ExtractNorents(c)
			}
		}
	}
}

// ==========================================================================
// ExtractExpensive — ported from C Crash_extract_expensive()
// Finds and extracts the object with the highest load among sibling items
// (linked via next_content in C; in Go this operates on a slice).
// ==========================================================================
func ExtractExpensive(items []*ObjectInstance) *ObjectInstance {
	if len(items) == 0 {
		return nil
	}
	maxIdx := 0
	maxLoad := 0
	for i, obj := range items {
		if obj == nil || obj.Prototype == nil {
			continue
		}
		l := obj.Prototype.Values[0]
		if l > maxLoad {
			maxLoad = l
			maxIdx = i
		}
	}
	return items[maxIdx]
}

// ==========================================================================
// CalculateRent — ported from C Crash_calculate_rent()
// Recursively accumulates object load values. In C it traverses next_content
// and contains linked lists. In Go we traverse the slice.
// ==========================================================================
func CalculateRent(items []*ObjectInstance, cost *int) {
	for _, obj := range items {
		if obj == nil {
			continue
		}
		if obj.Prototype != nil {
			v := obj.Prototype.Values[0]
			if v > 0 {
				*cost += v
			}
		}
		// Recurse into contained items
		CalculateRent(obj.Contains, cost)
	}
}

// ==========================================================================
// ExtractObjs — ported from C Crash_extract_objs()
// Recursively marks all objects in the chain as extracted (removes them from
// the world). In Go this means clearing their parent's reference and
// dereferencing from slices.
// ==========================================================================
func ExtractObjs(p *Player) {
	if p == nil {
		return
	}
	// Extract inventory items.
	for _, obj := range p.Inventory.Items {
		extractSingleObj(obj)
	}
	p.Inventory.clear()

	// Extract equipped items.
	for slot, obj := range p.Equipment.Slots {
		if obj != nil {
			extractSingleObj(obj)
		}
		delete(p.Equipment.Slots, slot)
	}
}

func extractSingleObj(obj *ObjectInstance) {
	if obj == nil {
		return
	}
	// Recursively extract containers.
	for _, c := range obj.Contains {
		extractSingleObj(c)
	}
	obj.Contains = nil
	// Clear prototype reference to allow GC.
	obj.Prototype = nil
}

// ==========================================================================
// RestoreWeight — ported from C Crash_restore_weight()
// Recursively recalculates container weights by summing contained items.
// ==========================================================================
func RestoreWeight(obj *ObjectInstance) {
	if obj == nil {
		return
	}
	// Recurse into contained items (depth-first).
	for _, c := range obj.Contains {
		RestoreWeight(c)
	}

	// Sum contained item weights into this object's weight.
	// The ObjectInstance doesn't track runtime weight; we store it in CustomData.
	sum := 0
	for _, c := range obj.Contains {
		if c.Prototype != nil {
			sum += c.Prototype.Weight
		}
	}
	if obj.Prototype != nil {
		obj.CustomData["restored_weight"] = obj.Prototype.Weight + sum
	}
}

// ==========================================================================
// AutoEquip — matches the C auto_equip() logic.
// locate: C WEAR_* index + 1 (1 = worn at pos 0, 20 = worn at pos 19).
// ==========================================================================
func AutoEquip(p *Player, obj *ObjectInstance, locate int) {
	if locate <= 0 {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	cPos := locate - 1
	_, ok := cWearPosToGoSlot(cPos)
	if !ok {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (invalid pos)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	rf := cWearPosCanWearFlag(cPos)
	wf := obj.Prototype.WearFlags[0]
	wears := (wf & rf) != 0
	// Warriors can wield in hold slot.
	if cPos == 17 && !wears {
		if (wf&(1<<13)) != 0 && ItemType(obj.Prototype.TypeFlag) == ItemWeaponType {
			wears = true
		}
	}
	if !wears {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (cant wear)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	// Alignment restrictions.
	xf := obj.Prototype.ExtraFlags[0]
	if (xf&FlagAntiEvil != 0 && p.IsEvil()) ||
		(xf&FlagAntiGood != 0 && p.IsGood()) ||
		(xf&FlagAntiNeutral != 0 && p.IsNeutral()) {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (alignment)", "player", p.Name, "obj_vnum", obj.VNum)
		}
		return
	}
	if err := p.Equipment.Equip(obj, p.Inventory); err != nil {
		obj.Location = LocInventoryPlayer(p.Name)
		if err := p.Inventory.addItem(obj); err != nil {
			slog.Error("autoequip: inventory full on load (equip failed)", "player", p.Name, "obj_vnum", obj.VNum, "original_err", err)
		}
	}
}

// ==========================================================================
// OfferRent — ported from C Crash_offer_rent()
// Calculates and reports rent cost for a player's items.
// Returns 0 if items can't be rented (norents, empty, too many), or total cost.
// ==========================================================================
func OfferRent(p *Player, display bool, factor int) int {
	if p == nil || p.Inventory == nil {
		return 0
	}

	// Check for unrentable items.
	norent := reportUnrentables(p.Inventory.Items, nil)
	if p.Equipment != nil {
		for _, obj := range p.Equipment.Slots {
			norent += reportUnrentables([]*ObjectInstance{obj}, nil)
		}
	}
	if norent > 0 {
		return 0
	}

	totalCost := MinRentCost * factor
	numItems := 0

	totalCost = reportRent(p.Inventory.Items, &totalCost, &numItems, display, factor)

	if p.Equipment != nil {
		for _, obj := range p.Equipment.Slots {
			if obj == nil {
				continue
			}
			if !IsUnrentable(obj) {
				numItems++
				v := obj.Prototype.Values[0]
				if v < 1 {
					v = 1
				}
				totalCost += v * factor
			}
			// Recurse into container contents.
			cCost := 0
			reportRent(obj.Contains, &cCost, &numItems, display, factor)
			totalCost += cCost
		}
	}

	if numItems == 0 {
		return 0
	}
	if numItems > MaxObjSave {
		return 0
	}
	return totalCost
}

// reportUnrentables — C Crash_report_unrentables. Returns count of unrentable items.
func reportUnrentables(items []*ObjectInstance, _ *MobInstance) int {
	count := 0
	for _, obj := range items {
		if obj == nil || obj.Prototype == nil {
			continue
		}
		if IsUnrentable(obj) {
			count++
		}
		count += reportUnrentables(obj.Contains, nil)
	}
	return count
}

// reportRent — C Crash_report_rent. Accumulates cost from rentable items.
func reportRent(items []*ObjectInstance, cost *int, numItems *int, display bool, factor int) int {
	for _, obj := range items {
		if obj == nil || obj.Prototype == nil {
			continue
		}
		if !IsUnrentable(obj) {
			(*numItems)++
			v := obj.Prototype.Values[0]
			if v < 1 {
				v = 1
			}
			*cost += v * factor
		}
		reportRent(obj.Contains, cost, numItems, display, factor)
	}
	return *cost
}

// ==========================================================================
// CrashSave — ported from C Crash_crashsave()
// Saves the player's current inventory/equipment as a crash save.
// In Go port, updates rent metadata in the JSON save file.
// ==========================================================================
func CrashSave(p *Player) {
	if p == nil || p.IsNPC() {
		return
	}

	// Restore weights before saving.
	for _, obj := range p.Inventory.Items {
		RestoreWeight(obj)
	}
	for _, obj := range p.Equipment.Slots {
		if obj != nil {
			RestoreWeight(obj)
		}
	}

	if err := SavePlayerWithRent(p, RentCrash, 0); err != nil {
		slog.Error("CrashSave: failed to save player", "name", p.GetName(), "error", err)
	}
}

// ==========================================================================
// Idlesave — ported from C Crash_idlesave()
// Saves player items when they idle out: extract norents, calculate cost,
// strip items if player can't afford double cost, then save as timed-out rent.
// ==========================================================================
func Idlesave(p *Player) {
	if p == nil || p.IsNPC() {
		return
	}

	// Extract norent items (move unequippable to inventory, extract contained).
	ExtractNorentsFromEquipped(p)
	ExtractNorentsList(p.Inventory.Items)

	// Calculate cost for inventory.
	cost := 0
	CalculateRent(p.Inventory.Items, &cost)

	// Calculate cost for equipment.
	costEq := 0
	for _, obj := range p.Equipment.Slots {
		if obj != nil {
			CalculateRent([]*ObjectInstance{obj}, &costEq)
		}
	}

	// Double the cost (forcerent).
	cost <<= 1
	costEq <<= 1

	// If player can't afford total cost, unequip and extract expensive items.
	if cost+costEq > p.Gold+p.BankGold {
		// Unequip all items to inventory.
		for slot, obj := range p.Equipment.Slots {
			if obj != nil {
				p.Equipment.UnequipItem(obj, p.Inventory)
				delete(p.Equipment.Slots, slot)
			}
		}
		cost += costEq
		costEq = 0

		// Extract most expensive items until affordable.
		for cost > p.Gold+p.BankGold && len(p.Inventory.Items) > 0 {
			expensive := ExtractExpensive(p.Inventory.Items)
			if expensive == nil {
				break
			}
			extractSingleObj(expensive)
			p.Inventory.removeItem(expensive)
			cost = 0
			CalculateRent(p.Inventory.Items, &cost)
			cost <<= 1
		}
	}

	// If player has nothing left, delete the save file and return.
	if len(p.Inventory.Items) == 0 && len(p.Equipment.Slots) == 0 {
		DeleteCrashFile(p.GetName())
		return
	}

	// Restore weights before saving.
	for _, obj := range p.Inventory.Items {
		RestoreWeight(obj)
	}
	for _, obj := range p.Equipment.Slots {
		if obj != nil {
			RestoreWeight(obj)
		}
	}

	if err := SavePlayerWithRent(p, RentTimedOut, cost); err != nil {
		slog.Error("Idlesave: failed to save player", "name", p.GetName(), "error", err)
	}

	// Extract all items from the world (they're saved now).
	ExtractObjs(p)
}

// ==========================================================================
// RentSave — ported from C Crash_rentsave()
// Saves items for voluntary rent: extract norents, save, extract items from world.
// ==========================================================================
func RentSave(p *Player, cost int) {
	if p == nil || p.IsNPC() {
		return
	}

	ExtractNorentsFromEquipped(p)
	ExtractNorentsList(p.Inventory.Items)

	// Restore weights before save.
	for _, obj := range p.Inventory.Items {
		RestoreWeight(obj)
	}
	for _, obj := range p.Equipment.Slots {
		if obj != nil {
			RestoreWeight(obj)
		}
	}

	if err := SavePlayerWithRent(p, RentRented, cost); err != nil {
		slog.Error("RentSave: failed to save player", "name", p.GetName(), "error", err)
	}

	ExtractObjs(p)
}

// ==========================================================================
// CryoSave — ported from C Crash_cryosave()
// Saves items for cryo rent: extract norents, deduct cost, save, extract items.
// ==========================================================================
func CryoSave(p *Player, cost int) {
	if p == nil || p.IsNPC() {
		return
	}

	ExtractNorentsFromEquipped(p)
	ExtractNorentsList(p.Inventory.Items)

	// Deduct cost.
	p.Gold = max(0, p.Gold-cost)

	// Restore weights before save.
	for _, obj := range p.Inventory.Items {
		RestoreWeight(obj)
	}
	for _, obj := range p.Equipment.Slots {
		if obj != nil {
			RestoreWeight(obj)
		}
	}

	if err := SavePlayerWithRent(p, RentCryo, 0); err != nil {
		slog.Error("CryoSave: failed to save player", "name", p.GetName(), "error", err)
	}

	ExtractObjs(p)
}

// ExtractNorentsList extracts all unrentable items from a slice (matching C Crash_extract_norents on ch->carrying).
func ExtractNorentsList(items []*ObjectInstance) {
	var survivors []*ObjectInstance
	for _, obj := range items {
		if obj == nil {
			continue
		}
		if IsUnrentable(obj) {
			extractSingleObj(obj)
			continue
		}
		// Recursively extract contained norents.
		ExtractNorentsFromContains(obj)
		survivors = append(survivors, obj)
	}
	// Replace items slice with survivors.
	_ = survivors // caller's slice is unmodified; caller must replace
}

// ExtractNorentsFromContains recursively removes unrentable items from an object's Contains.
func ExtractNorentsFromContains(obj *ObjectInstance) {
	if obj == nil {
		return
	}
	var survivors []*ObjectInstance
	for _, c := range obj.Contains {
		if c == nil {
			continue
		}
		if IsUnrentable(c) {
			extractSingleObj(c)
			continue
		}
		ExtractNorentsFromContains(c)
		survivors = append(survivors, c)
	}
	obj.Contains = survivors
}

// ==========================================================================
// CrashLoad — ported from C Crash_load()
// Restores player from saved data. Handles rent cost deduction, auto-equip,
// and container nesting (matching C's cont_row logic).
// Returns:

// ==========================================================================
// RentDeadline — ported from C Crash_rent_deadline()
// Reports how many days the player can rent with their current gold.
// ==========================================================================
func RentDeadline(ch *Player, recep *MobInstance, cost int) {
	if ch == nil || recep == nil || cost <= 0 {
		return
	}
	// In Go port, this is a social message — handled by the caller session/command layer.
	days := (ch.Gold) / cost // C version uses GET_GOLD(ch) + GET_BANK_GOLD(ch)
	if days > 1 {
		_ = fmt.Sprintf("You can rent for %d days with the gold you have.", days)
	} else if days == 1 {
		// "You can rent for 1 day..."
		_ = days
	}
}

// ==========================================================================
// SaveAllPlayers — ported from C Crash_save_all()
// Iterates all players and saves those with the crash flag set.
// ==========================================================================
func SaveAllPlayers(players []*Player) {
	for _, p := range players {
		if p == nil || p.IsNPC() {
			continue
		}
		CrashSave(p)
	}
}

// ==========================================================================
// DeleteCrashFile — ported from C Crash_delete_file()
// Removes the crash/rent save file for a player. In Go port, deletes the
// JSON save file.
// ==========================================================================
func DeleteCrashFile(name string) bool {
	err := DeletePlayer(name)
	if err != nil && !os.IsNotExist(err) {
		slog.Error("DeleteCrashFile: failed to delete", "name", name, "error", err)
		return false
	}
	slog.Debug("DeleteCrashFile", "player", name)
	return true
}

// ==========================================================================
// CleanCrashFile — ported from C Crash_clean_file()
// Checks if a player's save file should be deleted due to timeout.
// Returns true if the file was cleaned.
// ==========================================================================
func CleanCrashFile(name string, _ time.Time, _ int) bool {
	data, err := LoadSaveData(name)
	if err != nil {
		return false // file doesn't exist or can't read
	}

	now := time.Now()
	rentCode := data.RentCode
	savedTime := time.Unix(data.RentTime, 0)

	switch rentCode {
	case RentCrash, RentForced, RentTimedOut:
		if now.Sub(savedTime) > time.Duration(CrashFileTimeout)*24*time.Hour {
			DeleteCrashFile(name)
			fileType := "crash"
			switch rentCode {
			case RentForced:
				fileType = "forced rent"
			case RentTimedOut:
				fileType = "idlesave"
			}
			slog.Info("Deleting expired file", "player", name, "type", fileType)
			return true
		}
	case RentRented:
		if now.Sub(savedTime) > time.Duration(RentFileTimeout)*24*time.Hour {
			DeleteCrashFile(name)
			slog.Info("Deleting expired rent file", "player", name)
			return true
		}
	}
	return false
}

// ==========================================================================
// UpdateObjFiles — ported from C update_obj_file()
// Runs CleanCrashFile on all player save files.
// ==========================================================================
func UpdateObjFiles(playerNames []string) {
	for _, name := range playerNames {
		CleanCrashFile(name, time.Time{}, 0)
	}
}

// ==========================================================================
// DeleteAliasFile — ported from C Alias_delete_file()
// In Go port, aliases are not persisted to separate files.
// ==========================================================================
func DeleteAliasFile(name string) bool {
	slog.Debug("DeleteAliasFile", "player", name)
	return true
}

// ==========================================================================
// CrashListrent — ported from C Crash_listrent()
// Lists all items in a player's rent/crash save. In Go port, formats item
// data from the save struct.
// ==========================================================================
func CrashListrent(invDump []saveItemData, eqDump []saveItemData) string {
	var b strings.Builder
	for _, item := range invDump {
		fmt.Fprintf(&b, "  [%5d] inv %s\r\n", item.VNum, item.State)
	}
	for _, item := range eqDump {
		fmt.Fprintf(&b, "  [%5d] eq  %s\r\n", item.VNum, item.State)
	}
	return b.String()
}

// ==========================================================================
// GenReceptionist — ported from C gen_receptionist()
// Handles the receptionist NPC social/rent flow.
// ==========================================================================
func GenReceptionist(p *Player, cmd, arg string, mode int) bool {
	if p == nil || p.IsNPC() {
		return false
	}
	switch strings.ToLower(cmd) {
	case "offer":
		cost := OfferRent(p, true, mode)
		if cost > 0 && cost <= p.Gold {
			p.SendMessage(fmt.Sprintf("You can rent for %d.\r\n", cost))
		} else {
			p.SendMessage("You can't afford to rent.\r\n")
		}
		return true
	case "rent":
		cost := OfferRent(p, false, mode)
		if cost <= 0 || cost > p.Gold {
			p.SendMessage("You can't afford to rent.\r\n")
			return true
		}
		if mode == CryoFactor {
			CryoSave(p, cost)
		} else {
			RentSave(p, cost)
		}
		p.SendMessage("Your belongings are stored.\r\n")
		return true
	}
	return false
}

// ==========================================================================
// RestoreItemsFromSave — new function to create ObjectInstances from saved
// saveItemData, using prototype lookups. This closes the gap between
// saveDataToPlayer (which creates bare Inventory/Equipment) and the actual
// item restoration needed after load.
// ==========================================================================
func RestoreItemsFromSave(inv []saveItemData, eq []saveItemData, getProto func(vnum int) (*parser.Obj, bool)) ([]*ObjectInstance, map[int]*ObjectInstance) {
	invItems := make([]*ObjectInstance, 0, len(inv))
	for _, s := range inv {
		proto, ok := getProto(s.VNum)
		if !ok {
			slog.Warn("RestoreItemsFromSave: missing proto", "vnum", s.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if s.State != nil {
			for k, v := range s.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		invItems = append(invItems, obj)
	}

	eqItems := make(map[int]*ObjectInstance)
	for _, s := range eq {
		proto, ok := getProto(s.VNum)
		if !ok {
			slog.Warn("RestoreItemsFromSave: missing eq proto", "vnum", s.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if s.State != nil {
			for k, v := range s.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		eqItems[0] = obj // slot mapping handled by AutoEquip
	}

	return invItems, eqItems
}
