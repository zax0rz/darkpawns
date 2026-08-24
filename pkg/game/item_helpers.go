package game

import (
	"strings"
)

// ItemType represents a CircleMUD item type constant.
type ItemType int

// Item type constants matching src/structs.h
const (
	ItemLight      ItemType = 1
	ItemScroll     ItemType = 2
	ItemWand       ItemType = 3
	ItemStaff      ItemType = 4
	ItemWeaponType ItemType = 5
	ItemFireWeapon ItemType = 6
	ItemMissile    ItemType = 7
	ItemTreasure   ItemType = 8
	ItemArmor      ItemType = 9
	ItemPotion     ItemType = 10
	ItemWorn       ItemType = 11
	ItemOther      ItemType = 12
	ItemTrash      ItemType = 13
	ItemTrap       ItemType = 14
	ItemContainer  ItemType = 15
	ItemNote       ItemType = 16
	ItemDrinkcon   ItemType = 17
	ItemKey        ItemType = 18
	ItemFood       ItemType = 19
	ItemMoney      ItemType = 20
	ItemPen        ItemType = 21
	ItemBoat       ItemType = 22
	ItemFountain   ItemType = 23
	ItemVehicle    ItemType = 24
	ItemOnion      ItemType = 25
	ItemArmorPiece ItemType = 26
	ItemTattoo     ItemType = 27
	ItemRawmat     ItemType = 28
	ItemWeaponPart ItemType = 29
	ItemTool       ItemType = 30
	ItemGem        ItemType = 31
	ItemJewelry    ItemType = 32
	ItemFurniture  ItemType = 33
	ItemBag        ItemType = 35
	ItemBackpack   ItemType = 36
	ItemCorpse     ItemType = 37
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

func (t ItemType) IsWeapon() bool {
	return t == ItemWeaponType || t == ItemFireWeapon || t == ItemMissile
}

func (t ItemType) IsArmor() bool    { return t == ItemArmor || t == ItemArmorPiece || t == ItemWorn }
func (t ItemType) IsFood() bool     { return t == ItemFood || t == ItemDrinkcon }
func (t ItemType) IsReadable() bool { return t == ItemScroll || t == ItemNote || t == ItemPen }

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
	contPickproofBit = 1 << 1
	contClosed       = 1 << 2
	contLocked       = 1 << 3
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

// Extra-flag (ITEM_*) bit positions — src/structs.h:468-496. These are bit
// positions within word 0 of ExtraFlags, not pre-shifted masks; use with
// ObjectInstance.HasExtraFlag/SetExtraFlag.
const (
	extraFlagNoDonate  = 3  // ITEM_NODONATE
	extraFlagInvisible = 5  // ITEM_INVISIBLE
	extraFlagNoDrop    = 7  // ITEM_NODROP
	extraFlagTakeName  = 17 // ITEM_TAKE_NAME
	extraFlagTwoHanded = 28 // ITEM_TWO_HANDED
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
}

// wearMessages maps eq pos index to [room_msg, char_msg]
var wearMessages = [][]string{
	{"ITEM_WEAR_TAKE - Tell a god.", "ITEM_WEAR_TAKE - Tell a god."},
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

// Container flag helpers
func contIsClosed(obj *ObjectInstance) bool {
	return obj != nil && obj.GetValue(contFlags)&contClosed != 0
}

// IsContainerClosed is the exported version of contIsClosed for session layer use.
func IsContainerClosed(obj *ObjectInstance) bool {
	return contIsClosed(obj)
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

// legacyActArgs adapts the old loosely typed helper arguments to Act's typed
// victim/primary-object/secondary-object parameters. Keep this only while
// domain migrations retire the legacy helper signatures.
func legacyActArgs(args ...interface{}) (Actor, *ObjectInstance, *ObjectInstance) {
	var victim Actor
	var obj, victObj *ObjectInstance
	for _, arg := range args {
		switch value := arg.(type) {
		case Actor:
			if victim == nil {
				victim = value
			}
		case *ObjectInstance:
			if obj == nil {
				obj = value
			} else if victObj == nil {
				victObj = value
			}
		}
	}
	return victim, obj, victObj
}

func legacyActFormat(msg string) string {
	return strings.TrimRight(msg, "\r\n")
}

// actToChar is a legacy signature kept as a thin adapter to canonical Act.
func (w *World) actToChar(ch *Player, msg string, obj1, obj2 interface{}) {
	victim, obj, victObj := legacyActArgs(obj1, obj2)
	Act(nil, false, ch, victim, obj, victObj, legacyActFormat(msg), "", ToChar)
}

// actToRoom is a legacy signature kept as a thin adapter to canonical Act.
func (w *World) actToRoom(ch *Player, msg string, obj1, obj2 interface{}) {
	victim, obj, victObj := legacyActArgs(obj1, obj2)
	Act(w, false, ch, victim, obj, victObj, legacyActFormat(msg), "", ToRoom)
}

// actToVictim is a legacy signature kept as a thin adapter to canonical Act.
func actToVictim(ch, vict *Player, msg string, obj1, obj2 interface{}) {
	_, obj, victObj := legacyActArgs(obj1, obj2)
	Act(nil, false, ch, vict, obj, victObj, legacyActFormat(msg), "", ToVict)
}

// actToRoomExclude is a legacy signature kept as a thin adapter to Act.
func (w *World) actToRoomExclude(ch, vict *Player, msg string, obj1, obj2 interface{}) {
	_, obj, victObj := legacyActArgs(obj1, obj2)
	Act(w, false, ch, vict, obj, victObj, legacyActFormat(msg), "", ToNotVict)
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
