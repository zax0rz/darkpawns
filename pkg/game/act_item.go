package game

import (
	"fmt"
	"strings"
)

// Item type constants matching src/structs.h
const (
	ITEM_LIGHT       = 1
	ITEM_SCROLL      = 2
	ITEM_WAND        = 3
	ITEM_STAFF       = 4
	ITEM_WEAPON      = 5
	ITEM_FIRE_WEAPON = 6
	ITEM_MISSILE     = 7
	ITEM_TREASURE    = 8
	ITEM_ARMOR       = 9
	ITEM_POTION      = 10
	ITEM_WORN        = 11
	ITEM_OTHER       = 12
	ITEM_TRASH       = 13
	ITEM_TRAP        = 14
	ITEM_CONTAINER   = 15
	ITEM_NOTE        = 16
	ITEM_DRINKCON    = 17
	ITEM_KEY         = 18
	ITEM_FOOD        = 19
	ITEM_MONEY       = 20
	ITEM_PEN         = 21
	ITEM_BOAT        = 22
	ITEM_FOUNTAIN    = 23
	ITEM_VEHICLE     = 24
	ITEM_ONION       = 25
	ITEM_ARMOR_PIECE = 26
	ITEM_TATTOO      = 27
	ITEM_RAWMAT      = 28
	ITEM_WEAPON_PART = 29
	ITEM_TOOL        = 30
	ITEM_GEM         = 31
	ITEM_JEWELRY     = 32
	ITEM_FURNITURE   = 33
	ITEM_BAG         = 35
	ITEM_BACKPACK    = 36
	ITEM_CORPSE      = 37
)

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
// WearFlags is [3]int bitmask from parser (ITEM_WEAR_* constants from structs.h).
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

// ---------------------------------------------------------------------------
// perform_put
// ---------------------------------------------------------------------------
func (w *World) performPut(ch *Player, obj, cont *ObjectInstance) {
	if cont.GetTotalWeight()+obj.GetWeight() > cont.Prototype.Values[contCapacity] {
		w.actToChar(ch, "$p won't fit in $P.", obj, cont)
		return
	}
	ch.Inventory.RemoveItem(obj)
	cont.Contains = append(cont.Contains, obj)
	obj.SetCarrier(nil)
	obj.Container = cont
	w.actToChar(ch, "You put $p in $P.", obj, cont)
	w.actToRoom(ch, "$n puts $p in $P.", obj, cont)
}

// ---------------------------------------------------------------------------
// do_put
// ---------------------------------------------------------------------------
func (w *World) doPut(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	objDotmode := findAllDots(arg1)
	contDotmode := findAllDots(arg2)

	if arg1 == "" {
		ch.SendMessage("Put what in what?\r\n")
		return true
	}
	if contDotmode != findIndiv {
		ch.SendMessage("You can only put things into one container at a time.\r\n")
		return true
	}
	if arg2 == "" {
		what := "it"
		if objDotmode != findIndiv {
			what = "them"
		}
		ch.SendMessage(fmt.Sprintf("What do you want to put %s in?\r\n", what))
		return true
	}

	// Find container in inventory or room
	var cont *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			cont = obj
			break
		}
	}
	if cont == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.RoomVNum] {
				if isname(arg2, obj.GetKeywords()) {
					cont = obj
					break
				}
			}
		}
	}
	if cont == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg2), arg2))
		return true
	}
	if cont.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "$p is not a container.", cont, nil)
		return true
	}
	if contIsClosed(cont) {
		ch.SendMessage("You'd better open it first!\r\n")
		return true
	}

	if objDotmode == findIndiv {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isname(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You aren't carrying %s %s.\r\n", an(arg1), arg1))
			return true
		}
		if obj == cont {
			ch.SendMessage("You attempt to fold it into itself, but fail.\r\n")
			return true
		}
		w.performPut(ch, obj, cont)
	} else {
		found := false
		for _, obj := range ch.Inventory.Items {
			if obj == cont {
				continue
			}
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if objDotmode == findAll || isname(arg1, obj.GetKeywords()) {
				found = true
				w.performPut(ch, obj, cont)
			}
		}
		if !found {
			if objDotmode == findAll {
				ch.SendMessage("You don't seem to have anything to put in it.\r\n")
			} else {
				ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", arg1))
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// can_take_obj
// ---------------------------------------------------------------------------
func (w *World) canTakeObj(ch *Player, obj *ObjectInstance) bool {
	if len(ch.Inventory.Items) >= ch.Inventory.Capacity {
		w.actToChar(ch, "$p: you can't carry that many items.", obj, nil)
		return false
	}
	if ch.Inventory.GetWeight()+obj.GetWeight() > ch.Inventory.Capacity * 10 {
		w.actToChar(ch, "$p: you can't carry that much weight.", obj, nil)
		return false
	}
	// Check ITEM_WEAR_TAKE flag
	hasTake := false
	for _, wf := range obj.Prototype.WearFlags {
		if wf == 1 {
			hasTake = true
			break
		}
	}
	if !hasTake {
		w.actToChar(ch, "$p: you can't take that!", obj, nil)
		return false
	}
	return true
}

// getCheckMoney handles auto-conversion of money items on pickup
func (w *World) getCheckMoney(ch *Player, obj *ObjectInstance) {
	if obj.GetTypeFlag() == ITEM_MONEY && obj.Prototype.Values[0] > 0 {
		ch.Inventory.RemoveItem(obj)
		amount := obj.Prototype.Values[0]
		if amount > 1 {
			ch.SendMessage(fmt.Sprintf("There were %d coins.\r\n", amount))
		}
		ch.Gold += amount
	}
}

// ---------------------------------------------------------------------------
// perform_get_from_container
// ---------------------------------------------------------------------------
func (w *World) performGetFromContainer(ch *Player, obj, cont *ObjectInstance, mode int) {
	if mode == findObjInv || w.canTakeObj(ch, obj) {
		cont.RemoveFromContainer(obj)
		obj.Container = nil
		ch.Inventory.AddItem(obj)
		w.actToChar(ch, "You get $p from $P.", obj, cont)
		w.actToRoom(ch, "$n gets $p from $P.", obj, cont)
		w.getCheckMoney(ch, obj)
	}
}

// ---------------------------------------------------------------------------
// perform_get_from_room
// ---------------------------------------------------------------------------
func (w *World) performGetFromRoom(ch *Player, obj *ObjectInstance) {
	if w.canTakeObj(ch, obj) {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			w.RemoveItemFromRoom(obj, ch.RoomVNum)
		}
		obj.RoomVNum = -1
		ch.Inventory.AddItem(obj)
		w.actToChar(ch, "You get $p.", obj, nil)
		w.actToRoom(ch, "$n gets $p.", obj, nil)
		w.getCheckMoney(ch, obj)
	}
}

// ---------------------------------------------------------------------------
// do_get
// ---------------------------------------------------------------------------
func (w *World) doGet(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		ch.SendMessage("Get what?\r\n")
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		// get all
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil || len(w.roomItems[ch.RoomVNum]) == 0 {
			ch.SendMessage("There doesn't seem to be anything here.\r\n")
			return true
		}
		items := make([]*ObjectInstance, len(w.roomItems[ch.RoomVNum]))
		copy(items, w.roomItems[ch.RoomVNum])
		for _, obj := range items {
			w.performGetFromRoom(ch, obj)
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("What do you want to get all of?\r\n")
			return true
		}
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil {
			return true
		}
		items := make([]*ObjectInstance, len(w.roomItems[ch.RoomVNum]))
		copy(items, w.roomItems[ch.RoomVNum])
		found := false
		for _, obj := range items {
			if isname(keyword, obj.GetKeywords()) {
				w.performGetFromRoom(ch, obj)
				found = true
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't see any %ss here.\r\n", keyword))
		}
		return true
	}

	// Individual item
	if arg2 == "" {
		// get <item> from room
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room == nil {
			return true
		}
		var obj *ObjectInstance
		for _, o := range w.roomItems[ch.RoomVNum] {
			if isname(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
			return true
		}
		w.performGetFromRoom(ch, obj)
		return true
	}

	// get <item> <container>
	// Find container and determine mode
	var cont *ObjectInstance
	mode := findObjRoom
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			cont = obj
			mode = findObjInv
			break
		}
	}
	if cont == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.RoomVNum] {
				if isname(arg2, obj.GetKeywords()) {
					cont = obj
					break
				}
			}
		}
	}
	if cont == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg2), arg2))
		return true
	}
	if cont.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "$p is not a container.", cont, nil)
		return true
	}
	if contIsClosed(cont) {
		w.actToChar(ch, "$p is closed.", cont, nil)
		return true
	}

	// Find item inside container
	var obj *ObjectInstance
	for _, o := range cont.Contains {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("There doesn't seem to be %s %s in %s.\r\n", an(arg1), arg1, cont.GetShortDesc()))
		return true
	}

	// Perform the get
	w.performGetFromContainer(ch, obj, cont, mode)
	return true
}

// ---------------------------------------------------------------------------
// perform_drop
// ---------------------------------------------------------------------------
func (w *World) performDrop(ch *Player, obj *ObjectInstance) {
	if objHasFlag(obj, 1<<0) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, "You can't let go of $p!!  Yeech!", obj, nil)
		return
	}
	ch.Inventory.RemoveItem(obj)
	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil {
		w.roomItems[ch.RoomVNum] = append(w.roomItems[ch.RoomVNum], obj)
	}
	obj.RoomVNum = ch.GetRoomVNum()
	obj.SetCarrier(nil)
	obj.Container = nil
	w.actToChar(ch, "You drop $p.", obj, nil)
	w.actToRoom(ch, "$n drops $p.", obj, nil)
}

// ---------------------------------------------------------------------------
// do_drop
// ---------------------------------------------------------------------------
func (w *World) doDrop(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Drop what?\r\n")
		return true
	}
	arg1 := parts[0]
	sname := "drop"

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be carrying anything.\r\n")
			return true
		}
		for _, obj := range ch.Inventory.Items {
			w.performDrop(ch, obj)
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage(fmt.Sprintf("What do you want to %s all of?\r\n", sname))
			return true
		}
		found := false
		for _, obj := range ch.Inventory.Items {
			if isname(keyword, obj.GetKeywords()) {
				w.performDrop(ch, obj)
				found = true
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual drop
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	w.performDrop(ch, obj)
	return true
}

// ---------------------------------------------------------------------------
// perform_give
// ---------------------------------------------------------------------------
func (w *World) performGive(ch *Player, vict *Player, obj *ObjectInstance) {
	if objHasFlag(obj, 1<<0) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, "You can't let go of $p!!  Yeech!", obj, nil)
		return
	}
	if len(vict.Inventory.Items) >= vict.Inventory.Capacity {
		w.actToChar(ch, "$N seems to have $S hands full.", vict, obj)
		return
	}
	if obj.GetWeight()+vict.Inventory.GetWeight() > vict.Inventory.Capacity * 10 {
		w.actToChar(ch, "$E can't carry that much weight.", vict, nil)
		return
	}
	ch.Inventory.RemoveItem(obj)
	vict.Inventory.AddItem(obj)
	w.actToChar(ch, "You give $p to $N.", obj, vict)
	actToVictim(ch, vict, "$n gives you $p.", obj, nil)
	w.actToRoomExclude(ch, vict, "$n gives $p to $N.", obj, vict)
}

// ---------------------------------------------------------------------------
// give_find_vict
// ---------------------------------------------------------------------------
func (w *World) giveFindVict(ch *Player, arg string) *Player {
	if arg == "" {
		ch.SendMessage("To who?\r\n")
		return nil
	}
	vict := w.FindPlayerInRoom(ch.GetRoomVNum(), arg)
	if vict == nil {
		ch.SendMessage("There doesn't seem to be anyone here by that name.\r\n")
		return nil
	}
	if vict == ch {
		ch.SendMessage("What's the point of that?\r\n")
		return nil
	}
	return vict
}

// ---------------------------------------------------------------------------
// perform_give_gold
// ---------------------------------------------------------------------------
func (w *World) performGiveGold(ch *Player, vict *Player, amount int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh ... we are jolly funny today, eh?\r\n")
		return
	}
	if ch.Gold < amount && ch.GetLevel() < lvlGod {
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}
	ch.SendMessage("Ok.\r\n")
	actToVictim(ch, vict, "$n gives you %d gold coins.", nil, nil)
	w.actToRoomExclude(ch, vict, "$n gives %s to $N.", nil, vict)
	if ch.GetLevel() < lvlGod {
		ch.Gold -= amount
	}
	vict.Gold += amount
}

func moneyDesc(amount int) string {
	if amount == 0 {
		return "nothing"
	}
	return fmt.Sprintf("%d gold coin%s", amount, map[bool]string{true: "s", false: ""}[amount != 1])
}

// ---------------------------------------------------------------------------
// do_give
// ---------------------------------------------------------------------------
func (w *World) doGive(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Give what to who?\r\n")
		return true
	}

	arg1 := parts[0]

	// Check if first arg is a number (gold)
	if isNumber(arg1) {
		amount := atoi(arg1)
		if len(parts) < 2 {
			ch.SendMessage("Give what to who?\r\n")
			return true
		}
		// Check for "coins" or "coin" keyword
		arg2 := parts[1]
		if strings.EqualFold(arg2, "coins") || strings.EqualFold(arg2, "coin") {
			victName := ""
			if len(parts) > 2 {
				victName = parts[2]
			}
			vict := w.giveFindVict(ch, victName)
			if vict != nil {
				w.performGiveGold(ch, vict, amount)
			}
			return true
		}
		// Just a number wasn't coins
		ch.SendMessage("You can't give more than one item at a time.\r\n")
		return true
	}

	// Give object
	victName := ""
	if len(parts) > 1 {
		victName = parts[1]
	}
	vict := w.giveFindVict(ch, victName)
	if vict == nil {
		return true
	}

	dotmode := findAllDots(arg1)

	if dotmode == findIndiv {
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isname(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
			return true
		}
		w.performGive(ch, vict, obj)
	} else {
		if dotmode == findAlldot && len(parts) > 0 {
			// Strip all. prefix
			keyword := arg1[4:]
			if keyword == "" && len(parts) > 1 {
				keyword = parts[0]
			}
		}
		if len(ch.Inventory.Items) == 0 {
			ch.SendMessage("You don't seem to be holding anything.\r\n")
			return true
		}
		for _, obj := range ch.Inventory.Items {
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if dotmode == findAll || isname(arg1, obj.GetKeywords()) {
				w.performGive(ch, vict, obj)
			}
		}
	}
	return true
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

// ---------------------------------------------------------------------------
// do_drink
// ---------------------------------------------------------------------------
func (w *World) doDrink(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Drink from what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Find drink container in inventory first, then room
	var temp *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			temp = obj
			break
		}
	}
	onGround := false
	if temp == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, obj := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, obj.GetKeywords()) {
					temp = obj
					onGround = true
					break
				}
			}
		}
	}
	if temp == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if temp.GetTypeFlag() != ITEM_DRINKCON && temp.GetTypeFlag() != ITEM_FOUNTAIN {
		ch.SendMessage("You can't drink from that!\r\n")
		return true
	}
	if onGround && temp.GetTypeFlag() == ITEM_DRINKCON {
		ch.SendMessage("You have to be holding that to drink from it.\r\n")
		return true
	}

	// Condition checks (simplified)
	liqType := temp.Prototype.Values[2]
	if liqType < 0 || liqType >= len(drinks) {
		liqType = 0
	}

	if temp.Prototype.Values[1] <= 0 {
		ch.SendMessage("It's empty.\r\n")
		return true
	}

	ch.SendMessage(fmt.Sprintf("You drink the %s.\r\n", drinks[liqType]))

	drunkAff := drinkAff[liqType][0]
	fullAff := drinkAff[liqType][1]
	thirstAff := drinkAff[liqType][2]

	// Calculate amount to drink
	var amount int
	if drunkAff > 0 {
		amount = 3 + len(drinks)/2 // approximate: number(3,8) or condition-based
		if amount > temp.Prototype.Values[1] {
			amount = temp.Prototype.Values[1]
		}
	} else {
		amount = 3 + len(drinks)/2
		if amount > temp.Prototype.Values[1] {
			amount = temp.Prototype.Values[1]
		}
	}

	w.actToRoom(ch, "$n drinks $p.", temp, nil)

	// Reduce weight
	weightLoss := amount
	if weightLoss > temp.GetWeight() {
		weightLoss = temp.GetWeight()
	}
	_ = weightLoss // weight tracking simplified

	// Update condition
	_ = drunkAff
	_ = fullAff
	_ = thirstAff

	// Empty the container
	temp.Prototype.Values[1] -= amount
	if temp.Prototype.Values[1] <= 0 {
		temp.Prototype.Values[1] = 0
		temp.Prototype.Values[2] = 0
		temp.Prototype.Values[3] = 0
	}

	ch.SendMessage("You feel refreshed.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_eat
// ---------------------------------------------------------------------------
func (w *World) doEat(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Eat what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Find food in inventory
	var food *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			food = obj
			break
		}
	}
	if food == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	if food.GetTypeFlag() != ITEM_FOOD && ch.GetLevel() < lvlGod {
		ch.SendMessage("You can't eat THAT!\r\n")
		return true
	}

	foodVal := 0
	if len(food.Prototype.Values) > 0 {
		foodVal = food.Prototype.Values[0]
	}

	w.actToChar(ch, "You eat $p.", food, nil)
	w.actToRoom(ch, "$n eats $p.", food, nil)

	_ = foodVal

	// Consume the food
	ch.Inventory.RemoveItem(food)
	ch.SendMessage("That was good!\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_pour
// ---------------------------------------------------------------------------
func (w *World) doPour(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) == 0 {
		w.actToChar(ch, "From what do you want to pour?", nil, nil)
		return true
	}
	arg1 := strings.TrimSpace(parts[0])
	arg2 := ""
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		w.actToChar(ch, "From what do you want to pour?", nil, nil)
		return true
	}

	// Find from_obj in inventory
	var fromObj *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg1, obj.GetKeywords()) {
			fromObj = obj
			break
		}
	}
	if fromObj == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if fromObj.GetTypeFlag() != ITEM_DRINKCON {
		w.actToChar(ch, "You can't pour from that!", nil, nil)
		return true
	}
	if fromObj.Prototype.Values[1] <= 0 {
		w.actToChar(ch, "The $p is empty.", fromObj, nil)
		return true
	}
	if arg2 == "" {
		w.actToChar(ch, "Where do you want it?  Out or in what?", nil, nil)
		return true
	}

	if strings.EqualFold(arg2, "out") {
		// Pour out
		w.actToRoom(ch, "$n empties $p.", fromObj, nil)
		w.actToChar(ch, "You empty $p.", fromObj, nil)
		fromObj.Prototype.Values[1] = 0
		fromObj.Prototype.Values[2] = 0
		fromObj.Prototype.Values[3] = 0
		return true
	}

	// Pour into another container
	var toObj *ObjectInstance
	for _, obj := range ch.Inventory.Items {
		if isname(arg2, obj.GetKeywords()) {
			toObj = obj
			break
		}
	}
	if toObj == nil {
		w.actToChar(ch, "You can't find it!", nil, nil)
		return true
	}
	if toObj.GetTypeFlag() != ITEM_DRINKCON && toObj.GetTypeFlag() != ITEM_FOUNTAIN {
		w.actToChar(ch, "You can't pour anything into that.", nil, nil)
		return true
	}
	if toObj == fromObj {
		w.actToChar(ch, "A most unproductive effort.", nil, nil)
		return true
	}
	if toObj.Prototype.Values[1] != 0 && toObj.Prototype.Values[2] != fromObj.Prototype.Values[2] {
		w.actToChar(ch, "There is already another liquid in it!", nil, nil)
		return true
	}
	if toObj.Prototype.Values[1] >= toObj.Prototype.Values[0] {
		w.actToChar(ch, "There is no room for more.", nil, nil)
		return true
	}

	w.actToChar(ch, fmt.Sprintf("You pour the %s into the %s.", drinks[fromObj.Prototype.Values[2]], arg2), nil, nil)

	// Perform the pour
	toObj.Prototype.Values[2] = fromObj.Prototype.Values[2]
	amount := toObj.Prototype.Values[0] - toObj.Prototype.Values[1]
	fromObj.Prototype.Values[1] -= amount
	toObj.Prototype.Values[1] = toObj.Prototype.Values[0]

	if fromObj.Prototype.Values[1] < 0 {
		toObj.Prototype.Values[1] += fromObj.Prototype.Values[1]
		fromObj.Prototype.Values[1] = 0
		fromObj.Prototype.Values[2] = 0
		fromObj.Prototype.Values[3] = 0
	}

	// Poison carries over
	if fromObj.Prototype.Values[3] != 0 {
		toObj.Prototype.Values[3] = 1
	}

	return true
}

// ---------------------------------------------------------------------------
// find_eq_pos
// ---------------------------------------------------------------------------
func findEqPos(obj *ObjectInstance, arg string) int {
	if arg != "" {
		if pos, ok := eqPosKeywords[strings.ToLower(arg)]; ok {
			return pos
		}
		return -1
	}

	// Auto-detect
	if canWearObject(obj, eqWearFingerR) {
		return eqWearFingerR
	}
	if canWearObject(obj, eqWearNeck1) {
		return eqWearNeck1
	}
	if canWearObject(obj, eqWearBody) {
		return eqWearBody
	}
	if canWearObject(obj, eqWearHead) {
		return eqWearHead
	}
	if canWearObject(obj, eqWearLegs) {
		return eqWearLegs
	}
	if canWearObject(obj, eqWearFeet) {
		return eqWearFeet
	}
	if canWearObject(obj, eqWearHands) {
		return eqWearHands
	}
	if canWearObject(obj, eqWearArms) {
		return eqWearArms
	}
	if canWearObject(obj, eqWearShield) {
		return eqWearShield
	}
	if canWearObject(obj, eqWearAbout) {
		return eqWearAbout
	}
	if canWearObject(obj, eqWearWaist) {
		return eqWearWaist
	}
	if canWearObject(obj, eqWearWristR) {
		return eqWearWristR
	}
	if canWearObject(obj, eqWearAblegs) {
		return eqWearAblegs
	}
	if canWearObject(obj, eqWearFace) {
		return eqWearFace
	}
	if canWearObject(obj, eqWearHover) {
		return eqWearHover
	}
	if canWearObject(obj, eqWearWield) {
		return eqWearWield
	}
	if canWearObject(obj, eqWearHold) {
		return eqWearHold
	}
	return -1
}

// ---------------------------------------------------------------------------
// wear_message
// ---------------------------------------------------------------------------
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

// ---------------------------------------------------------------------------
// perform_wear
// ---------------------------------------------------------------------------
func (w *World) performWear(ch *Player, obj *ObjectInstance, where int) {
	if !canWearObject(obj, where) || where == eqWearLight {
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
	if where == eqWearWield {
		if !canWearObject(obj, eqWearWield) {
			ch.SendMessage("You can't wield that.\r\n")
			return
		}
		if obj.GetWeight() > 50 { // simplified str_app check
			ch.SendMessage("It is too heavy for you to use.\r\n")
			return
		}
		// Check for two-handed
		if objHasFlag(obj, 1<<3) && (w.IsEquipped(ch, eqWearHold) || w.IsEquipped(ch, eqWearShield)) {
			ch.SendMessage("Both hands must be free to wield that.\r\n")
			return
		}
	} else if where == eqWearHold || where == eqWearShield {
		if w.IsEquipped(ch, eqWearWield) {
			// Check if wielded weapon is two-handed
			wpn := w.GetEquipped(ch, eqWearWield)
			if wpn != nil && objHasFlag(wpn, 1<<3) {
				ch.SendMessage("Both your hands are occupied with your weapon at the moment.\r\n")
				return
			}
		}
	}

	// Remove from inventory and equip
	ch.Inventory.RemoveItem(obj)
	w.EquipItem(ch, obj, where)
	w.wearMessage(ch, obj, where)
}

// IsEquipped checks if a character has something equipped in a slot (0-based eq pos)
func (w *World) IsEquipped(ch *Player, slot int) bool {
	if ch.Equipment == nil {
		return false
	}
	_, found := ch.Equipment.GetItemInSlot(EquipmentSlot(slot))
	return found
}

// GetEquipped returns the item in a given slot
func (w *World) GetEquipped(ch *Player, slot int) *ObjectInstance {
	if ch.Equipment == nil {
		return nil
	}
	item, found := ch.Equipment.GetItemInSlot(EquipmentSlot(slot))
	if !found {
		return nil
	}
	return item
}

// EquipItem equips an item at the given slot
func (w *World) EquipItem(ch *Player, obj *ObjectInstance, slot int) {
	if ch.Equipment == nil {
		return
	}
	ch.Equipment.Equip(obj, ch.Inventory)
}

// UnequipItem removes an item from a slot
func (w *World) UnequipItem(ch *Player, slot int) {
	if ch.Equipment == nil {
		return
	}
	ch.Equipment.Unequip(EquipmentSlot(slot), ch.Inventory)
}

// ---------------------------------------------------------------------------
// do_wear
// ---------------------------------------------------------------------------
func (w *World) doWear(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.SplitN(arg, " ", 2)
	arg1 := ""
	arg2 := ""
	if len(parts) > 0 {
		arg1 = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		arg2 = strings.TrimSpace(parts[1])
	}

	if arg1 == "" {
		ch.SendMessage("Wear what?\r\n")
		return true
	}

	dotmode := findAllDots(arg1)

	if arg2 != "" && dotmode != findIndiv {
		ch.SendMessage("You can't specify the same body location for more than one item!\r\n")
		return true
	}

	if dotmode == findAll {
		items := ch.Inventory.Items
		if len(items) == 0 {
			ch.SendMessage("You don't seem to have anything to wear.\r\n")
			return true
		}
		for _, obj := range items {
			if objHasFlag(obj, 1<<4) {
				continue
			}
			if where := findEqPos(obj, ""); where >= 0 {
				w.performWear(ch, obj, where)
			} else {
				w.actToChar(ch, "You can't wear $p.", obj, nil)
			}
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("Wear all of what?\r\n")
			return true
		}
		found := false
		items := ch.Inventory.Items
		for _, obj := range items {
			if isname(keyword, obj.GetKeywords()) {
				if where := findEqPos(obj, ""); where >= 0 {
					w.performWear(ch, obj, where)
					found = true
				} else {
					w.actToChar(ch, "You can't wear $p.", obj, nil)
				}
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
		return true
	}
	if where := findEqPos(obj, arg2); where >= 0 {
		w.performWear(ch, obj, where)
	} else if arg2 == "" {
		w.actToChar(ch, "You can't wear $p.", obj, nil)
	}
	return true
}

// ---------------------------------------------------------------------------
// perform_remove
// ---------------------------------------------------------------------------
func (w *World) performRemove(ch *Player, pos int) {
	if ch.Equipment == nil {
		return
	}
	obj, found := ch.Equipment.GetItemInSlot(EquipmentSlot(pos))
	if !found {
		return
	}
	if objHasFlag(obj, 1<<0) {
		w.actToChar(ch, "You can't remove $p, it must be CURSED!", obj, nil)
		return
	}
	if len(ch.Inventory.Items) >= ch.Inventory.Capacity {
		w.actToChar(ch, "$p: you can't carry that many items!", obj, nil)
		return
	}

	// Unequip
	w.UnequipItem(ch, pos)
	ch.Inventory.AddItem(obj)

	w.actToChar(ch, "You stop using $p.", obj, nil)
	w.actToRoom(ch, "$n stops using $p.", obj, nil)
}

// ---------------------------------------------------------------------------
// do_remove
// ---------------------------------------------------------------------------
func (w *World) doRemove(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Remove what?\r\n")
		return true
	}
	arg1 := parts[0]

	dotmode := findAllDots(arg1)

	if dotmode == findAll {
		found := false
		for i := 0; i < eqWearMax; i++ {
			if w.IsEquipped(ch, i) {
				w.performRemove(ch, i)
				found = true
			}
		}
		if !found {
			ch.SendMessage("You're not using anything.\r\n")
		}
		return true
	}

	if dotmode == findAlldot {
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage("Remove all of what?\r\n")
			return true
		}
		found := false
		for i := 0; i < eqWearMax; i++ {
			if w.IsEquipped(ch, i) {
				obj, _ := ch.Equipment.GetItemInSlot(EquipmentSlot(i))
				if obj != nil && isname(keyword, obj.GetKeywords()) {
					w.performRemove(ch, i)
					found = true
				}
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to be using any %ss.\r\n", keyword))
		}
		return true
	}

	// Individual remove
	for i := 0; i < eqWearMax; i++ {
		if w.IsEquipped(ch, i) {
			obj, _ := ch.Equipment.GetItemInSlot(EquipmentSlot(i))
			if obj != nil && isname(arg1, obj.GetKeywords()) {
				w.performRemove(ch, i)
				return true
			}
		}
	}
	ch.SendMessage(fmt.Sprintf("You don't seem to be using %s %s.\r\n", an(arg1), arg1))
	return true
}

// ---------------------------------------------------------------------------
// Container door commands: do_open, do_close, do_lock, do_unlock
// These handle opening/closing/locking/unlocking containers.
// Room exits are handled by doGenDoor in act_movement.go.
// ---------------------------------------------------------------------------

// DOOR_IS_OPENABLE for containers
func doorIsOpenable(obj *ObjectInstance) bool {
	return obj.GetTypeFlag() == ITEM_CONTAINER && contIsCloseable(obj)
}

// doOpen handles opening containers
func (w *World) doOpen(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Open what?\r\n")
		return true
	}
	arg1 := parts[0]

	// Look for container in inventory or room
	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if !doorIsOpenable(obj) {
		w.actToChar(ch, "You can't open $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("It's already open!\r\n")
		return true
	}
	if contIsLocked(obj) {
		ch.SendMessage("It seems to be locked.\r\n")
		return true
	}

	contSetClosed(obj, false)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n opens $p.", obj, nil)
	return true
}

// doClose handles closing containers
func (w *World) doClose(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Close what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if !doorIsOpenable(obj) {
		w.actToChar(ch, "You can't close $p!", obj, nil)
		return true
	}
	if contIsClosed(obj) {
		ch.SendMessage("It's already closed!\r\n")
		return true
	}

	contSetClosed(obj, true)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n closes $p.", obj, nil)
	return true
}

// doUnlock handles unlocking containers
func (w *World) doUnlock(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Unlock what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if obj.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "That's not a container.", nil, nil)
		return true
	}
	if !contIsCloseable(obj) {
		w.actToChar(ch, "You can't unlock $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("It's not closed.\r\n")
		return true
	}
	if !contIsLocked(obj) {
		ch.SendMessage("It's not locked.\r\n")
		return true
	}

	// Check for key in inventory
	keyVNum := obj.Prototype.Values[contKey]
	if keyVNum > 0 {
		hasKey := false
		for _, inv := range ch.Inventory.Items {
			if inv.GetTypeFlag() == ITEM_KEY && inv.GetVNum() == keyVNum {
				hasKey = true
				break
			}
		}
		if !hasKey {
			w.actToChar(ch, "You don't have the key for $p.", obj, nil)
			return true
		}
	}

	if contIsLocked(obj) {
		contSetLocked(obj, false)
	}

	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n unlocks $p.", obj, nil)
	return true
}

// doLock handles locking containers
func (w *World) doLock(ch *Player, me *MobInstance, cmd, arg string) bool {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		ch.SendMessage("Lock what?\r\n")
		return true
	}
	arg1 := parts[0]

	var obj *ObjectInstance
	for _, o := range ch.Inventory.Items {
		if isname(arg1, o.GetKeywords()) {
			obj = o
			break
		}
	}
	if obj == nil {
		room := w.GetRoomInWorld(ch.GetRoomVNum())
		if room != nil {
			for _, o := range w.roomItems[ch.RoomVNum] {
				if isname(arg1, o.GetKeywords()) {
					obj = o
					break
				}
			}
		}
	}
	if obj == nil {
		ch.SendMessage(fmt.Sprintf("You don't see %s %s here.\r\n", an(arg1), arg1))
		return true
	}
	if obj.GetTypeFlag() != ITEM_CONTAINER {
		w.actToChar(ch, "That's not a container.", nil, nil)
		return true
	}
	if !contIsCloseable(obj) {
		w.actToChar(ch, "You can't lock $p!", obj, nil)
		return true
	}
	if !contIsClosed(obj) {
		ch.SendMessage("You'd better close it first.\r\n")
		return true
	}
	if contIsLocked(obj) {
		ch.SendMessage("It's already locked!\r\n")
		return true
	}

	// Check for key in inventory
	keyVNum := obj.Prototype.Values[contKey]
	if keyVNum > 0 {
		hasKey := false
		for _, inv := range ch.Inventory.Items {
			if inv.GetTypeFlag() == ITEM_KEY && inv.GetVNum() == keyVNum {
				hasKey = true
				break
			}
		}
		if !hasKey {
			w.actToChar(ch, "You don't have the key for $p.", obj, nil)
			return true
		}
	}

	contSetLocked(obj, true)
	ch.SendMessage("Ok.\r\n")
	w.actToRoom(ch, "$n locks $p.", obj, nil)
	return true
}
