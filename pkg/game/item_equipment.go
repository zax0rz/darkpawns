package game

import (
	"fmt"
	"log/slog"
	"strings"
)

// findEqPos finds the equipment position for an object
func findEqPos(obj *ObjectInstance, arg string) int {
	if arg != "" {
		if pos, ok := eqPosKeywords[strings.ToLower(arg)]; ok {
			return pos
		}
		return -1
	}

	// Auto-detect. C checks every flag in order, so the last matching wear
	// position wins for objects carrying more than one wear flag.
	where := -1
	if canWearObject(obj, eqWearFingerR) {
		where = eqWearFingerR
	}
	if canWearObject(obj, eqWearNeck1) {
		where = eqWearNeck1
	}
	if canWearObject(obj, eqWearBody) {
		where = eqWearBody
	}
	if canWearObject(obj, eqWearHead) {
		where = eqWearHead
	}
	if canWearObject(obj, eqWearLegs) {
		where = eqWearLegs
	}
	if canWearObject(obj, eqWearFeet) {
		where = eqWearFeet
	}
	if canWearObject(obj, eqWearHands) {
		where = eqWearHands
	}
	if canWearObject(obj, eqWearArms) {
		where = eqWearArms
	}
	if canWearObject(obj, eqWearShield) {
		where = eqWearShield
	}
	if canWearObject(obj, eqWearAbout) {
		where = eqWearAbout
	}
	if canWearObject(obj, eqWearWaist) {
		where = eqWearWaist
	}
	if canWearObject(obj, eqWearWristR) {
		where = eqWearWristR
	}
	if canWearObject(obj, eqWearAblegs) {
		where = eqWearAblegs
	}
	if canWearObject(obj, eqWearFace) {
		where = eqWearFace
	}
	if canWearObject(obj, eqWearHover) {
		where = eqWearHover
	}
	if canWearObject(obj, eqWearWield) {
		where = eqWearWield
	}
	return where
}

// wearMessage sends the wear message for an equipment position
func (w *World) wearMessage(ch *Player, obj *ObjectInstance, where int) {
	if where < 0 || where >= len(wearMessages) {
		return
	}
	msg := wearMessages[where]
	// Room message (TO_ROOM)
	w.actToRoom(ch, msg[0], obj, nil)
	// Character message (TO_CHAR)
	w.actToChar(ch, msg[1], obj, nil)
}

func canWearAtPosition(obj *ObjectInstance, where int) bool {
	if where == eqWearHold || where == eqWearHold2 {
		if obj.CanPickUp {
			return true
		}
		if obj.Prototype == nil {
			return false
		}
		for _, flags := range obj.Prototype.WearFlags {
			if flags&1 != 0 { // ITEM_WEAR_TAKE
				return true
			}
		}
		return false
	}
	return canWearObject(obj, where)
}

func objAntiAlign(ch *Player, obj *ObjectInstance) bool {
	flags := obj.GetExtraFlags()[0]
	return flags&FlagAntiEvil != 0 && ch.IsEvil() ||
		flags&FlagAntiGood != 0 && ch.IsGood() ||
		flags&FlagAntiNeutral != 0 && ch.IsNeutral()
}

func objInvalidClass(ch *Player, obj *ObjectInstance) bool {
	flags := obj.GetExtraFlags()[0]
	isSlashWeapon := canWearObject(obj, eqWearWield) && obj.GetValue(3) == 3
	isShield := canWearObject(obj, eqWearShield)
	return InvalidClass(ch.Class, uint32(flags), isSlashWeapon, isShield) // #nosec G115 -- flags is a non-negative Diku extra-flags bitmask that fits in uint32
}

// performWear equips an item at a given position
func (w *World) performWear(ch *Player, obj *ObjectInstance, where int) {
	if !canWearAtPosition(obj, where) || where == eqWearLight {
		w.actToChar(ch, "You can't wear $p there.", obj, nil)
		return
	}

	// For finger, neck, wrist: try secondary if primary full
	if where == eqWearFingerR {
		// Check if slot is occupied, try other finger
		if w.IsEquipped(ch, eqWearFingerR) {
			where = eqWearFingerL
		}
	}
	if where == eqWearNeck1 {
		if w.IsEquipped(ch, eqWearNeck1) {
			where = eqWearNeck2
		}
	}
	if where == eqWearWristR {
		if w.IsEquipped(ch, eqWearWristR) {
			where = eqWearWristL
		}
	}

	if w.IsEquipped(ch, where) {
		if where >= 0 && where < len(alreadyWearing) {
			ch.SendMessage(alreadyWearing[where])
		}
		return
	}

	// Wielding checks
	switch where {
	case eqWearWield:
		if !canWearObject(obj, eqWearWield) {
			ch.SendMessage("You can't wield that.\r\n")
			return
		}
		if ch.IsAffected(affFleshAlter) {
			ch.SendMessage("Your flesh is altered, you can't wield anything!\r\n")
			return
		}
		if obj.GetWeight() > ch.MaxWieldWeight() {
			ch.SendMessage("It is too heavy for you to use.\r\n")
			return
		}
		// Check for two-handed
		if obj.HasExtraFlag(0, extraFlagTwoHanded) && (w.IsEquipped(ch, eqWearHold) || w.IsEquipped(ch, eqWearShield)) {
			ch.SendMessage("Both hands must be free to wield that.\r\n")
			return
		}
	case eqWearHold, eqWearShield:
		if w.IsEquipped(ch, eqWearWield) {
			// Check if wielded weapon is two-handed
			wpn := w.GetEquipped(ch, eqWearWield)
			if wpn != nil && wpn.HasExtraFlag(0, extraFlagTwoHanded) {
				ch.SendMessage("Both your hands are occupied with your weapon at the moment.\r\n")
				return
			}
		}
	}

	invalidClass := objInvalidClass(ch, obj)
	if !invalidClass {
		w.wearMessage(ch, obj, where)
	}
	if objAntiAlign(ch, obj) {
		w.actToChar(ch, "You are zapped by $p and instantly let go of it.", obj, nil)
		w.actToRoom(ch, "$n is zapped by $p and instantly lets go of it.", obj, nil)
		return
	}
	if invalidClass {
		w.actToChar(ch, "You cannot use $p.", obj, nil)
		return
	}

	if err := w.EquipItem(ch, obj, where); err != nil {
		slog.Warn("equip failed during wear", "player", ch.Name, "slot", where, "error", err)
		ch.SendMessage("You can't wear that right now.\r\n")
		return
	}
}

// IsEquipped checks if a character has something equipped in a slot (0-based eq pos)
func (w *World) IsEquipped(ch *Player, slot int) bool {
	if ch.Equipment == nil {
		return false
	}
	goSlot, ok := cWearSlot(slot)
	if !ok {
		return false
	}
	_, found := ch.Equipment.GetItemInSlot(goSlot)
	return found
}

// GetEquipped returns the item in a given slot
func (w *World) GetEquipped(ch *Player, slot int) *ObjectInstance {
	if ch.Equipment == nil {
		return nil
	}
	goSlot, ok := cWearSlot(slot)
	if !ok {
		return nil
	}
	item, found := ch.Equipment.GetItemInSlot(goSlot)
	if !found {
		return nil
	}
	return item
}

// EquipItem equips an item at the given slot.
func (w *World) EquipItem(ch *Player, obj *ObjectInstance, slot int) error {
	if ch == nil || ch.Equipment == nil {
		return fmt.Errorf("character has no equipment")
	}
	goSlot, ok := cWearSlot(slot)
	if !ok {
		return fmt.Errorf("c wear position %d has no Go equipment slot", slot)
	}

	removed := ch.Inventory != nil && ch.Inventory.RemoveItem(obj)
	if err := ch.Equipment.SetSlot(goSlot, obj); err != nil {
		if removed {
			if rbErr := ch.Inventory.AddItem(obj); rbErr != nil {
				slog.Error("rollback after failed equip: restore to inventory failed", "player", ch.Name, "obj_vnum", obj.VNum, "error", rbErr)
			}
		}
		return err
	}
	obj.Location = LocEquippedPlayer(ch.Name, goSlot)
	if obj.HasExtraFlag(0, extraFlagTakeName) {
		obj.Runtime.ShortDescOverride = fmt.Sprintf("%s's %s", ch.Name, obj.GetKeywords())
	}
	return nil
}

// UnequipItem removes an item from a slot.
func (w *World) UnequipItem(ch *Player, slot int) error {
	if ch == nil || ch.Equipment == nil {
		return fmt.Errorf("character has no equipment")
	}
	goSlot, ok := cWearSlot(slot)
	if !ok {
		return fmt.Errorf("c wear position %d has no Go equipment slot", slot)
	}
	if obj, found := ch.Equipment.GetItemInSlot(goSlot); found && obj.HasExtraFlag(0, extraFlagTakeName) {
		keywords := obj.GetKeywords()
		obj.Runtime.ShortDescOverride = an(keywords) + " " + keywords
	}
	return ch.Equipment.Unequip(goSlot, ch.Inventory)
}

// FindEquippedVis resolves an EQUIPPED object by keyword, mirroring C do_use's
// SCMD_USE lookup (act.other.c:908-936): WEAR_HOLD is checked first, then every
// worn slot (the last keyword match wins). It never touches inventory — a
// wand/staff must be held or worn to be used.
func (w *World) FindEquippedVis(ch *Player, arg string) *ObjectInstance {
	if ch == nil || ch.Equipment == nil {
		return nil
	}
	if held := w.GetEquipped(ch, eqWearHold); held != nil && canSeeObject(ch, held) && isnameWithAbbrevs(arg, held.GetKeywords()) {
		return held
	}
	var match *ObjectInstance
	for _, item := range ch.Equipment.GetEquippedItems() {
		if item != nil && canSeeObject(ch, item) && isnameWithAbbrevs(arg, item.GetKeywords()) {
			match = item // last match wins, per C's slot loop
		}
	}
	return match
}

// FindCarriedVis resolves an object in the player's inventory by keyword,
// mirroring C get_obj_in_list_vis(ch, arg, ch->carrying): per-word prefix match
// with "N.name" handling, never the short description. Exported entry point for
// session command handlers that reproduce carrying-list lookups.
func (w *World) FindCarriedVis(ch *Player, arg string) *ObjectInstance {
	return getObjInInvVis(ch, arg)
}

// HeldItemVis resolves the WEAR_HOLD item by keyword — C do_use
// (act.other.c:897-910) checks the held item with isname BEFORE falling back
// to the carrying list, so a held matching item wins the quaff/recite/use
// target selection. Returns nil when nothing held or no keyword match.
func (w *World) HeldItemVis(ch *Player, arg string) *ObjectInstance {
	if ch == nil || ch.Equipment == nil {
		return nil
	}
	name := strings.TrimSpace(arg)
	if name == "" {
		return nil
	}
	item, ok := ch.Equipment.GetItemInSlot(SlotHold)
	if !ok || item == nil {
		return nil
	}
	if !isnameWithAbbrevs(name, item.GetKeywords()) {
		return nil
	}
	return item
}

func getObjInInvVis(ch *Player, arg string) *ObjectInstance {
	if ch == nil || ch.Inventory == nil {
		return nil
	}
	name := strings.TrimSpace(arg)
	number := GetNumber(&name)
	if number <= 0 || name == "" {
		return nil
	}

	found := 0
	for _, obj := range ch.Inventory.FindItems("") {
		if !isnameWithAbbrevs(name, obj.GetKeywords()) {
			continue
		}
		if !canSeeObject(ch, obj) && obj.GetTypeFlag() != ITEM_LIGHT {
			continue
		}
		found++
		if found == number {
			return obj
		}
	}
	return nil
}

func getObjInEquipVis(ch *Player, arg string) (*ObjectInstance, int) {
	if ch == nil || ch.Equipment == nil {
		return nil, -1
	}
	for where := 0; where < len(cWearToGoSlot); where++ {
		slot, ok := cWearSlot(where)
		if !ok {
			continue
		}
		obj, found := ch.Equipment.GetItemInSlot(slot)
		if found && canSeeObject(ch, obj) && isnameWithAbbrevs(arg, obj.GetKeywords()) {
			return obj, where
		}
	}
	return nil, -1
}

// DoWear handles wear <item> [position], wear all, and wear all.<keyword>.
func (w *World) DoWear(ch *Player, arg string) {
	// C do_wear parses with two_arguments (interpreter.c): fill words dropped,
	// tokens lowercased.
	arg1, arg2 := twoArguments(arg)
	if arg1 == "" {
		ch.SendMessage("Wear what?\r\n")
		return
	}

	dotmode := findAllDots(arg1)
	if arg2 != "" && dotmode != findIndiv {
		ch.SendMessage("You can't specify the same body location for more than one item!\r\n")
		return
	}

	if dotmode == findAll {
		itemsWorn := 0
		for _, obj := range ch.Inventory.FindItems("") {
			if !canSeeObject(ch, obj) {
				continue
			}
			if where := findEqPos(obj, ""); where >= 0 {
				itemsWorn++
				w.performWear(ch, obj, where)
			}
		}
		if itemsWorn == 0 {
			ch.SendMessage("You don't seem to have anything wearable.\r\n")
		}
		return
	}

	if dotmode == findAlldot {
		keyword := strings.TrimPrefix(arg1, "all.")
		if keyword == "" {
			ch.SendMessage("Wear all of what?\r\n")
			return
		}
		found := false
		for _, obj := range ch.Inventory.FindItems("") {
			if !canSeeObject(ch, obj) || !isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				continue
			}
			found = true
			if where := findEqPos(obj, ""); where >= 0 {
				w.performWear(ch, obj, where)
			} else {
				w.actToChar(ch, "You can't wear $p.", obj, nil)
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return
	}

	obj := getObjInInvVis(ch, arg1)
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return
	}
	where := findEqPos(obj, arg2)
	if where >= 0 {
		w.performWear(ch, obj, where)
	} else if arg2 == "" {
		w.actToChar(ch, "You can't wear $p.", obj, nil)
	} else {
		ch.SendMessage(fmt.Sprintf("'%s'?  What part of your body is THAT?\r\n", arg2))
	}
}

// DoWield wields one visible carried object.
func (w *World) DoWield(ch *Player, arg string) {
	arg1, _ := oneArgument(arg) // C do_wield: one_argument (fill-skip, lowercase)
	if arg1 == "" {
		ch.SendMessage("Wield what?\r\n")
		return
	}
	obj := getObjInInvVis(ch, arg1)
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return
	}
	if ch.IsAffected(affFleshAlter) {
		ch.SendMessage("Your flesh is altered, you can't wield anything!\r\n")
		return
	}
	w.performWear(ch, obj, eqWearWield)
}

// DoGrab holds one visible carried object.
func (w *World) DoGrab(ch *Player, arg string) {
	arg1, _ := oneArgument(arg) // C do_grab: one_argument (fill-skip, lowercase)
	if arg1 == "" {
		ch.SendMessage("Hold what?\r\n")
		return
	}
	obj := getObjInInvVis(ch, arg1)
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return
	}
	itemType := obj.GetTypeFlag()
	if !canWearObject(obj, eqWearHold) && itemType != ITEM_WAND && itemType != ITEM_STAFF && itemType != ITEM_SCROLL && itemType != ITEM_POTION {
		ch.SendMessage("You can't hold that.\r\n")
		return
	}
	w.performWear(ch, obj, eqWearHold)
}

// performRemove removes an item from an equipment slot
func (w *World) performRemove(ch *Player, pos int) {
	if ch.Equipment == nil {
		return
	}
	slot, ok := cWearSlot(pos)
	if !ok {
		return
	}
	obj, found := ch.Equipment.GetItemInSlot(slot)
	if !found {
		return
	}
	if obj.HasExtraFlag(0, extraFlagNoDrop) {
		w.actToChar(ch, "You can't remove $p, it must be CURSED!", obj, nil)
		return
	}
	if ch.Inventory.GetItemCount() >= ch.Inventory.GetCapacity() {
		w.actToChar(ch, "$p: you can't carry that many items!", obj, nil)
		return
	}

	if err := w.UnequipItem(ch, pos); err != nil {
		slog.Warn("unequip failed during remove", "player", ch.Name, "slot", pos, "error", err)
		ch.SendMessage("You can't remove that right now.\r\n")
		return
	}

	w.actToChar(ch, "You stop using $p.", obj, nil)
	w.actToRoom(ch, "$n stops using $p.", obj, nil)
}

// DoRemove handles remove <item>, remove all, and remove all.<keyword>.
func (w *World) DoRemove(ch *Player, arg string) {
	arg1, _ := oneArgument(arg) // C do_remove: one_argument (fill-skip, lowercase)
	if arg1 == "" {
		ch.SendMessage("Remove what?\r\n")
		return
	}
	dotmode := findAllDots(arg1)
	if arg1 == "all." {
		dotmode = findAlldot
	}

	if dotmode == findAll {
		found := false
		for where := 0; where < len(cWearToGoSlot); where++ {
			if w.IsEquipped(ch, where) {
				w.performRemove(ch, where)
				found = true
			}
		}
		if !found {
			ch.SendMessage("You're not using anything.\r\n")
		}
		return
	}

	if dotmode == findAlldot {
		keyword := strings.TrimPrefix(arg1, "all.")
		if keyword == "" {
			ch.SendMessage("Remove all of what?\r\n")
			return
		}
		found := false
		for where := 0; where < len(cWearToGoSlot); where++ {
			obj := w.GetEquipped(ch, where)
			if obj == nil || !canSeeObject(ch, obj) || !isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				continue
			}
			w.performRemove(ch, where)
			found = true
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to be using any %ss.\r\n", keyword))
		}
		return
	}

	obj, where := getObjInEquipVis(ch, arg1)
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to be using %s %s.\r\n", an(arg1), arg1))
		return
	}
	w.performRemove(ch, where)
}
