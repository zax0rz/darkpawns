//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"strings"
)

// ItemType represents a CircleMUD item type constant.
type ItemType int

// Item type constants matching src/structs.h
const (
	ItemLight       ItemType = 1
	ItemScroll      ItemType = 2
	ItemWand        ItemType = 3
	ItemStaff       ItemType = 4
	ItemWeaponType  ItemType = 5
	ItemFireWeapon  ItemType = 6
	ItemMissile     ItemType = 7
	ItemTreasure    ItemType = 8
	ItemArmor       ItemType = 9
	ItemPotion      ItemType = 10
	ItemWorn        ItemType = 11
	ItemOther       ItemType = 12
	ItemTrash       ItemType = 13
	ItemTrap        ItemType = 14
	ItemContainer   ItemType = 15
	ItemNote        ItemType = 16
	ItemDrinkcon    ItemType = 17
	ItemKey         ItemType = 18
	ItemFood        ItemType = 19
	ItemMoney       ItemType = 20
	ItemPen         ItemType = 21
	ItemBoat        ItemType = 22
	ItemFountain    ItemType = 23
	ItemVehicle     ItemType = 24
	ItemOnion       ItemType = 25
	ItemArmorPiece  ItemType = 26
	ItemTattoo      ItemType = 27
	ItemRawmat      ItemType = 28
	ItemWeaponPart  ItemType = 29
	ItemTool        ItemType = 30
	ItemGem         ItemType = 31
	ItemJewelry     ItemType = 32
	ItemFurniture   ItemType = 33
	ItemBag         ItemType = 35
	ItemBackpack    ItemType = 36
	ItemCorpse      ItemType = 37
)

// Legacy aliases for backward compatibility (typed as int so they work with
// parser.Obj.TypeFlag = int without requiring casts everywhere).
const (
	ITEM_LIGHT       int = 1
	ITEM_SCROLL      int = 2
	ITEM_WAND        int = 3
	ITEM_STAFF       int = 4
	ITEM_WEAPON      int = 5
	ITEM_FIRE_WEAPON int = 6
	ITEM_MISSILE     int = 7
	ITEM_TREASURE    int = 8
	ITEM_ARMOR       int = 9
	ITEM_POTION      int = 10
	ITEM_WORN        int = 11
	ITEM_OTHER       int = 12
	ITEM_TRASH       int = 13
	ITEM_TRAP        int = 14
	ITEM_CONTAINER   int = 15
	ITEM_NOTE        int = 16
	ITEM_DRINKCON    int = 17
	ITEM_KEY         int = 18
	ITEM_FOOD        int = 19
	ITEM_MONEY       int = 20
	ITEM_PEN         int = 21
	ITEM_BOAT        int = 22
	ITEM_FOUNTAIN    int = 23
	ITEM_VEHICLE     int = 24
	ITEM_ONION       int = 25
	ITEM_ARMOR_PIECE int = 26
	ITEM_TATTOO      int = 27
	ITEM_RAWMAT      int = 28
	ITEM_WEAPON_PART int = 29
	ITEM_TOOL        int = 30
	ITEM_GEM         int = 31
	ITEM_JEWELRY     int = 32
	ITEM_FURNITURE   int = 33
	ITEM_BAG         int = 35
	ITEM_BACKPACK    int = 36
	ITEM_CORPSE      int = 37
)

// ItemType helper methods
func (t ItemType) IsContainer() bool { return t == ItemContainer }
func (t ItemType) IsWeapon() bool    { return t == ItemWeaponType || t == ItemFireWeapon || t == ItemMissile }
func (t ItemType) IsArmor() bool     { return t == ItemArmor || t == ItemArmorPiece || t == ItemWorn }
func (t ItemType) IsFood() bool      { return t == ItemFood || t == ItemDrinkcon }
func (t ItemType) IsReadable() bool  { return t == ItemScroll || t == ItemNote || t == ItemPen }

// Container value indices
const (
	contCapacity = iota
	contFlags
	contKey
	contPickproof
)

// Container flags
const (
	contCloseable    = 1 << 0
	contClosed       = 1 << 1
	contLocked       = 1 << 2
	contPickproofBit = 1 << 3
)

// dotMode constants
const (
	findIndiv = iota
	findAll
	findAlldot
)

// Constants for find types
const (
	findObjInv   = 1
	findObjRoom  = 2
	findObjEquip = 4
)

// SCMD constants for give/drop/junk/donate
const (
	scmdDrop = iota
	scmdJunk
	scmdDonate
)

// SCMD for drink/eat
const (
	scmdDrink = iota
	scmdSip
	scmdEat
	scmdTaste
)

// SCMD for pour/fill
const (
	scmdPour = iota
	scmdFill
)

// Equipment position constants matching C WEAR_*
const (
	eqWearLight = iota
	eqWearFingerR
	eqWearFingerL
	eqWearNeck1
	eqWearNeck2
	eqWearBody
	eqWearHead
	eqWearLegs
	eqWearFeet
	eqWearHands
	eqWearArms
	eqWearShield
	eqWearAbout
	eqWearWaist
	eqWearWristR
	eqWearWristL
	eqWearWield
	eqWearHold
	eqWearHold2
	eqWearAblegs
	eqWearFace
	eqWearHover
	eqWearMax
)

// eqPosKeywords maps body part names to equipment positions
var eqPosKeywords = map[string]int{
	"finger": eqWearFingerR,
	"neck":   eqWearNeck1,
	"body":   eqWearBody,
	"head":   eqWearHead,
	"legs":   eqWearLegs,
	"feet":   eqWearFeet,
	"hands":  eqWearHands,
	"arms":   eqWearArms,
	"shield": eqWearShield,
	"about":  eqWearAbout,
	"waist":  eqWearWaist,
	"wrist":  eqWearWristR,
	"ablegs": eqWearAblegs,
	"face":   eqWearFace,
	"hover":  eqWearHover,
	"wield":  eqWearWield,
	"hold":   eqWearHold,
}

// wearMessages maps eq pos index to [room_msg, char_msg]
var wearMessages = [][]string{
	{"$n slides $p on to $s right ring finger.", "You slide $p on to your right ring finger."},
	{"$n slides $p on to $s left ring finger.", "You slide $p on to your left ring finger."},
	{"$n wears $p around $s neck.", "You wear $p around your neck."},
	{"$n wears $p around $s neck.", "You wear $p around your neck."},
	{"$n wears $p on $s body.", "You wear $p on your body."},
	{"$n wears $p on $s head.", "You wear $p on your head."},
	{"$n puts $p on $s legs.", "You put $p on your legs."},
	{"$n wears $p on $s feet.", "You wear $p on your feet."},
	{"$n puts $p on $s hands.", "You put $p on your hands."},
	{"$n wears $p on $s arms.", "You wear $p on your arms."},
	{"$n straps $p around $s arm as a shield.", "You start to use $p as a shield."},
	{"$n wears $p about $s body.", "You wear $p around your body."},
	{"$n wears $p around $s waist.", "You wear $p around your waist."},
	{"$n puts $p on around $s right wrist.", "You put $p on around your right wrist."},
	{"$n puts $p on around $s left wrist.", "You put $p on around your left wrist."},
	{"$n wields $p.", "You wield $p."},
	{"$n grabs $p.", "You grab $p."},
	{"$n grabs $p.", "You grab $p."},
	{"$n wears $p about $s legs.", "You wear $p about your legs."},
	{"$n wears $p on $s face.", "You wear $p on your face."},
	{"$n sets $p afloat by $s head.", "You set $p afloat near your head."},
}

// alreadyWearing messages per equipment position
var alreadyWearing = []string{
	"YOU SHOULD NEVER SEE THIS MESSAGE.  PLEASE REPORT.\r\n",
	"YOU SHOULD NEVER SEE THIS MESSAGE.  PLEASE REPORT.\r\n",
	"You're already wearing something on both of your ring fingers.\r\n",
	"YOU SHOULD NEVER SEE THIS MESSAGE.  PLEASE REPORT.\r\n",
	"You can't wear anything else around your neck.\r\n",
	"You're already wearing something on your body.\r\n",
	"You're already wearing something on your head.\r\n",
	"You're already wearing something on your legs.\r\n",
	"You're already wearing something on your feet.\r\n",
	"You're already wearing something on your hands.\r\n",
	"You're already wearing something on your arms.\r\n",
	"You're already using a shield.\r\n",
	"You're already wearing something about your body.\r\n",
	"You already have something around your waist.\r\n",
	"YOU SHOULD NEVER SEE THIS MESSAGE.  PLEASE REPORT.\r\n",
	"You're already wearing something around both of your wrists.\r\n",
	"You're already wielding a weapon.\r\n",
	"You're already holding something.\r\n",
	"You're already holding something.\r\n",
	"You're already wearing something about your legs.\n\r",
	"You're already wearing something on your face.\n\r",
	"Something is already hovering near your head.\n\r",
}

// Drink names from C drinks[] table
var drinks = []string{
	"water",
	"beer",
	"wine",
	"ale",
	"dark ale",
	"whisky",
	"lemonade",
	"firebreather",
	"local speciality",
	"slime mold juice",
	"milk",
	"tea",
	"coffee",
	"blood",
	"salt water",
	"clear water",
	"skunk essence",
	"cocoa",
	"elvish wine",
	"dwarven spirits",
	"green dragon",
	"liquid fire",
	"sake",
	"battery acid",
	"lab reagent",
	"ichor",
	"oil",
	"healing potion",
	"mana potion",
	"white wine",
	"champagne",
	"mead",
	"rose wine",
	"spring water",
	"holy water",
	"ratafee",
	"mountain dew",
}

// drinkAff[liqIdx][0]=drunk, [1]=full, [2]=thirst
var drinkAff = [][]int{
	{0, 1, 10}, // water
	{3, 2, 5},  // beer
	{5, 3, 3},  // wine
	{3, 2, 5},  // ale
	{3, 2, 5},  // dark ale
	{6, 2, 1},  // whisky
	{0, 1, 8},  // lemonade
	{8, 1, 1},  // firebreather
	{3, 3, 3},  // local speciality
	{0, 4, -8}, // slime mold juice
	{0, 3, 6},  // milk
	{0, 1, 6},  // tea
	{0, 1, 6},  // coffee
	{0, 2, 6},  // blood
	{0, 1, -1}, // salt water
	{0, 0, 13}, // clear water
	{10, 2, 3}, // skunk essence
	{0, 2, 7},  // cocoa
	{7, 4, 3},  // elvish wine
	{5, 5, 2},  // dwarven spirits
	{10, 2, 5}, // green dragon
	{10, 0, 0}, // liquid fire
	{6, 2, 1},  // sake
	{0, 0, 0},  // battery acid
	{0, 0, 0},  // lab reagent
	{0, 0, 0},  // ichor
	{0, 0, 0},  // oil
	{0, 0, 0},  // healing potion
	{0, 0, 0},  // mana potion
	{5, 2, 5},  // white wine
	{6, 1, 6},  // champagne
	{6, 3, 5},  // mead
	{3, 1, 8},  // rose wine
	{0, 0, 12}, // spring water
	{0, 0, 12}, // holy water
	{3, 3, 3},  // ratafee
	{0, 0, 12}, // mountain dew
}

// Container flag helpers
func contIsCloseable(obj *ObjectInstance) bool {
	return obj.Prototype.Values[contFlags]&contCloseable != 0
}
func contIsClosed(obj *ObjectInstance) bool {
	return obj.Prototype.Values[contFlags]&contClosed != 0
}

// IsContainerClosed is the exported version of contIsClosed for session layer use.
func IsContainerClosed(obj *ObjectInstance) bool {
	return contIsClosed(obj)
}

func contIsLocked(obj *ObjectInstance) bool {
	return obj.Prototype.Values[contFlags]&contLocked != 0
}
func contSetClosed(obj *ObjectInstance, val bool) {
	if val {
		obj.Prototype.Values[contFlags] |= contClosed
	} else {
		obj.Prototype.Values[contFlags] &^= contClosed
	}
}
func contSetLocked(obj *ObjectInstance, val bool) {
	if val {
		obj.Prototype.Values[contFlags] |= contLocked
	} else {
		obj.Prototype.Values[contFlags] &^= contLocked
	}
}

// drinkLiquidIndex maps a drink name to its index
func drinkLiquidIndex(name string) int {
	for i, d := range drinks {
		if strings.EqualFold(d, name) {
			return i
		}
	}
	return 0
}

// wearBitForPosition returns the wear flag bit for a given eq position
func wearBitForPosition(where int) int {
	switch where {
	case eqWearFingerR, eqWearFingerL:
		return 1 << 1 // finger
	case eqWearNeck1, eqWearNeck2:
		return 1 << 2 // neck
	case eqWearBody:
		return 1 << 3 // body
	case eqWearHead:
		return 1 << 4 // head
	case eqWearLegs:
		return 1 << 5 // legs
	case eqWearFeet:
		return 1 << 6 // feet
	case eqWearHands:
		return 1 << 7 // hands
	case eqWearArms:
		return 1 << 8 // arms
	case eqWearShield:
		return 1 << 9 // shield
	case eqWearAbout:
		return 1 << 10 // about
	case eqWearWaist:
		return 1 << 11 // waist
	case eqWearWristR, eqWearWristL:
		return 1 << 12 // wrist
	case eqWearWield:
		return 1 << 13 // wield
	case eqWearHold, eqWearHold2:
		return 1 << 14 // hold
	case eqWearAblegs:
		return 1 << 16 // ablegs
	case eqWearFace:
		return 1 << 17 // face
	case eqWearHover:
		return 1 << 18 // hover
	default:
		return 0
	}
}

// canWearObject checks if object can be worn in given position.
// WearFlags is [4]int bitmask from parser (ITEM_WEAR_* constants from structs.h).
// Source: act.item.c can_take_obj() / wear checks.
func canWearObject(obj *ObjectInstance, where int) bool {
	bit := wearBitForPosition(where)
	if bit == 0 && where != 0 {
		return false
	}
	// WearFlags stores bitmasks; OR all slots together and check for bit.
	var wearMask int
	for _, wf := range obj.Prototype.WearFlags {
		wearMask |= wf
	}
	return wearMask&bit != 0
}

func objHasFlag(obj *ObjectInstance, bit int) bool {
	if obj == nil || obj.Prototype == nil {
		return false
	}
	return (obj.Prototype.ExtraFlags[0] & bit) != 0
}

// isname checks if str matches keywords in a space-separated namelist
func isname(str, namelist string) bool {
	if namelist == "" {
		return false
	}
	words := strings.Fields(namelist)
	for _, w := range words {
		if strings.Contains(strings.ToLower(w), strings.ToLower(str)) {
			return true
		}
	}
	return false
}

// findAllDots returns the dot mode for an argument
func findAllDots(arg string) int {
	if arg == "all" || arg == "all." {
		return findAll
	}
	if strings.HasPrefix(arg, "all.") {
		return findAlldot
	}
	return findIndiv
}

func an(s string) string {
	if s == "" {
		return "a"
	}
	c := strings.ToLower(s)[0]
	if c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u' {
		return "an"
	}
	return "a"
}

// removeFromSlice removes an item from a pointer slice
func removeFromSlice(items []*ObjectInstance, obj *ObjectInstance) []*ObjectInstance {
	for i, item := range items {
		if item == obj {
			return append(items[:i], items[i+1:]...)
		}
	}
	return items
}

// FindPlayerInRoom finds a player by name in a specific room
func (w *World) FindPlayerInRoom(vnum int, name string) *Player {
	for _, p := range w.GetPlayersInRoom(vnum) {
		if strings.EqualFold(p.Name, name) || strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(name)) {
			return p
		}
	}
	return nil
}

// FindMobInRoom finds a mob by name in a specific room
func (w *World) FindMobInRoom(vnum int, name string) *MobInstance {
	for _, m := range w.GetMobsInRoom(vnum) {
		if strings.HasPrefix(strings.ToLower(m.GetName()), strings.ToLower(name)) {
			return m
		}
	}
	return nil
}

// actToChar sends an act-string to the character
func (w *World) actToChar(ch *Player, msg string, obj1, obj2 interface{}) {
	s := msg
	if obj1 != nil {
		if o, ok := obj1.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
		if o, ok := obj1.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	if obj2 != nil {
		if o, ok := obj2.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
		if o, ok := obj2.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	s = strings.ReplaceAll(s, "$n", "you")
	s = strings.ReplaceAll(s, "$e", "you")
	s = strings.ReplaceAll(s, "$s", "your")
	s = strings.ReplaceAll(s, "$m", "you")
	s = strings.ReplaceAll(s, "$F", "it")
	ch.SendMessage(s + "\r\n")
}

// actToRoom sends an act-string to the room (excluding ch)
func (w *World) actToRoom(ch *Player, msg string, obj1, obj2 interface{}) {
	s := msg
	if obj1 != nil {
		if o, ok := obj1.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
		if o, ok := obj1.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	if obj2 != nil {
		if o, ok := obj2.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
		if o, ok := obj2.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	s = strings.ReplaceAll(s, "$n", ch.Name)
	s = strings.ReplaceAll(s, "$e", "he")
	s = strings.ReplaceAll(s, "$s", "his")
	s = strings.ReplaceAll(s, "$m", "him")
	s = strings.ReplaceAll(s, "$F", "it")
	w.roomMessage(ch.GetRoomVNum(), s)
}

// actToVictim sends an act-string to the victim only
func actToVictim(ch, vict *Player, msg string, obj1, obj2 interface{}) {
	s := msg
	if obj1 != nil {
		if o, ok := obj1.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
		if o, ok := obj1.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	if obj2 != nil {
		if o, ok := obj2.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
	}
	s = strings.ReplaceAll(s, "$n", ch.Name)
	s = strings.ReplaceAll(s, "$e", "he")
	s = strings.ReplaceAll(s, "$s", "his")
	s = strings.ReplaceAll(s, "$m", "him")
	s = strings.ReplaceAll(s, "$N", "you")
	s = strings.ReplaceAll(s, "$E", "you")
	s = strings.ReplaceAll(s, "$S", "your")
	s = strings.ReplaceAll(s, "$M", "you")
	s = strings.ReplaceAll(s, "$F", "it")
	vict.SendMessage(s + "\r\n")
}

// actToRoomExclude sends to room excluding ch and vict
func (w *World) actToRoomExclude(ch, vict *Player, msg string, obj1, obj2 interface{}) {
	s := msg
	if obj1 != nil {
		if o, ok := obj1.(*ObjectInstance); ok {
			s = strings.ReplaceAll(s, "$p", o.GetShortDesc())
			s = strings.ReplaceAll(s, "$P", o.GetShortDesc())
		}
	}
	if obj2 != nil {
		if o, ok := obj2.(*Player); ok {
			s = strings.ReplaceAll(s, "$N", o.Name)
			s = strings.ReplaceAll(s, "$E", "them")
			s = strings.ReplaceAll(s, "$S", "their")
			s = strings.ReplaceAll(s, "$M", "them")
		}
	}
	s = strings.ReplaceAll(s, "$n", ch.Name)
	s = strings.ReplaceAll(s, "$e", "he")
	s = strings.ReplaceAll(s, "$s", "his")
	s = strings.ReplaceAll(s, "$m", "him")
	s = strings.ReplaceAll(s, "$F", "it")
	w.roomMessageExcludeTwo(ch.GetRoomVNum(), s, ch.Name, vict.Name)
}

// moneyDesc describes an amount of money
func moneyDesc(amount int) string {
	if amount == 0 {
		return "nothing"
	}
	return fmt.Sprintf("%d gold coin%s", amount, map[bool]string{true: "s", false: ""}[amount != 1])
}

// atoi converts string to int
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}
