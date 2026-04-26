// Package game — object location types for the ObjectLocation refactor.
//
// This file introduces the typed location system that replaces the old
// Carrier/EquippedOn/Container interface{} fields on ObjectInstance.
//
// MIGRATION STRATEGY (incremental, not big-bang):
//
// Phase A: Add ObjectLocation type and helper methods. Keep old fields alive.
// Phase B: Add Location field to ObjectInstance, wire MoveObject/ExtractObject.
// Phase C: Update all call sites to use Location instead of old fields.
// Phase D: Remove old fields, add compile-time checks.
//
// Object ownership has 6 distinct states:
//   - Nowhere:       object exists in memory but not in the world (e.g., prototype)
//   - InRoom:        on the ground in a room
//   - InInventory:   in a player's or mob's inventory
//   - Equipped:      equipped on a player or mob in a specific slot
//   - InContainer:   inside another object (bag, corpse, etc.)
//   - InShop:        held by a shop for sale
//
// The "key" for each owner type uses stable identifiers:
//   - Player:  Name (string) — unique per online player
//   - Mob:     MobID (int) — assigned by World.nextMobID
//   - Room:    RoomVNum (int) — static zone data
//   - Shop:    ShopVNum (int) — the mob VNum of the shopkeeper
//   - Container: ContainerObjID (int) — the ObjectInstance.ID of the container

package game

import "fmt"

// ObjectOwnerKind describes the type of entity that owns an object location.
type ObjectOwnerKind uint8

const (
	OwnerNone   ObjectOwnerKind = iota
	OwnerPlayer
	OwnerMob
)

// ObjectLocationKind describes where an object is in the world.
type ObjectLocationKind uint8

const (
	ObjNowhere     ObjectLocationKind = iota // not placed in world
	ObjInRoom                                // on ground
	ObjInInventory                           // in player or mob inventory
	ObjEquipped                              // equipped on player or mob
	ObjInContainer                           // inside another object
	ObjInShop                                // held by a shop
)

// ObjectLocation is a tagged union describing an object's position.
// Exactly one of the key fields should be meaningful depending on Kind.
type ObjectLocation struct {
	Kind          ObjectLocationKind
	OwnerKind     ObjectOwnerKind
	RoomVNum      int    // ObjInRoom: which room
	PlayerName    string // ObjInInventory, ObjEquipped: which player
	MobID         int    // ObjInInventory, ObjEquipped: which mob (World-assigned ID)
	ContainerObjID int   // ObjInContainer: which object contains this
	ShopVNum      int    // ObjInShop: which shop (shopkeeper mob VNum)
	Slot          EquipmentSlot // ObjEquipped: which equipment slot
}

// Convenience constructors.

func LocRoom(roomVNum int) ObjectLocation {
	return ObjectLocation{Kind: ObjInRoom, RoomVNum: roomVNum}
}

func LocNowhere() ObjectLocation {
	return ObjectLocation{Kind: ObjNowhere}
}

func LocInventoryPlayer(name string) ObjectLocation {
	return ObjectLocation{Kind: ObjInInventory, OwnerKind: OwnerPlayer, PlayerName: name}
}

func LocInventoryMob(mobID int) ObjectLocation {
	return ObjectLocation{Kind: ObjInInventory, OwnerKind: OwnerMob, MobID: mobID}
}

func LocEquippedPlayer(name string, slot EquipmentSlot) ObjectLocation {
	return ObjectLocation{Kind: ObjEquipped, OwnerKind: OwnerPlayer, PlayerName: name, Slot: slot}
}

func LocEquippedMob(mobID int, slot EquipmentSlot) ObjectLocation {
	return ObjectLocation{Kind: ObjEquipped, OwnerKind: OwnerMob, MobID: mobID, Slot: slot}
}

func LocContainer(containerObjID int) ObjectLocation {
	return ObjectLocation{Kind: ObjInContainer, ContainerObjID: containerObjID}
}

func LocShop(shopVNum int) ObjectLocation {
	return ObjectLocation{Kind: ObjInShop, ShopVNum: shopVNum}
}

// Query methods — these will replace the old Carrier type-switches.

func (l ObjectLocation) IsNowhere() bool     { return l.Kind == ObjNowhere }
func (l ObjectLocation) IsInRoom() bool      { return l.Kind == ObjInRoom }
func (l ObjectLocation) IsInInventory() bool  { return l.Kind == ObjInInventory }
func (l ObjectLocation) IsEquipped() bool     { return l.Kind == ObjEquipped }
func (l ObjectLocation) IsInContainer() bool  { return l.Kind == ObjInContainer }
func (l ObjectLocation) IsInShop() bool       { return l.Kind == ObjInShop }

func (l ObjectLocation) InRoomOf(vnum int) bool {
	return l.Kind == ObjInRoom && l.RoomVNum == vnum
}

func (l ObjectLocation) InInventoryOfPlayer(name string) bool {
	return l.Kind == ObjInInventory && l.PlayerName == name
}

func (l ObjectLocation) InInventoryOfMob(mobID int) bool {
	return l.Kind == ObjInInventory && l.MobID == mobID
}

func (l ObjectLocation) InContainerOf(objID int) bool {
	return l.Kind == ObjInContainer && l.ContainerObjID == objID
}

// Validate checks that the ObjectLocation is internally consistent.
func (l ObjectLocation) Validate() error {
	switch l.Kind {
	case ObjNowhere:
		if l.OwnerKind != OwnerNone || l.RoomVNum != 0 || l.PlayerName != "" || l.MobID != 0 {
			return fmt.Errorf("ObjNowhere should have no owner or location fields")
		}
	case ObjInRoom:
		if l.RoomVNum <= 0 {
			return fmt.Errorf("ObjInRoom requires positive RoomVNum, got %d", l.RoomVNum)
		}
		if l.OwnerKind != OwnerNone {
			return fmt.Errorf("ObjInRoom should not have an owner kind")
		}
		if l.PlayerName != "" {
			return fmt.Errorf("ObjInRoom should not have a player name")
		}
		if l.MobID != 0 {
			return fmt.Errorf("ObjInRoom should not have a MobID")
		}
		if l.ContainerObjID != 0 {
			return fmt.Errorf("ObjInRoom should not have a ContainerObjID")
		}
	case ObjInInventory:
		if l.OwnerKind == OwnerPlayer && l.PlayerName == "" {
			return fmt.Errorf("OwnerPlayer requires non-empty PlayerName")
		}
		if l.OwnerKind == OwnerMob && l.MobID <= 0 {
			return fmt.Errorf("OwnerMob requires positive MobID, got %d", l.MobID)
		}
		if l.OwnerKind == OwnerPlayer && l.MobID != 0 {
			return fmt.Errorf("ObjInInventory with OwnerPlayer should not have MobID")
		}
		if l.OwnerKind == OwnerMob && l.PlayerName != "" {
			return fmt.Errorf("ObjInInventory with OwnerMob should not have PlayerName")
		}
		if l.RoomVNum != 0 {
			return fmt.Errorf("ObjInInventory should not have RoomVNum")
		}
		if l.ContainerObjID != 0 {
			return fmt.Errorf("ObjInInventory should not have ContainerObjID")
		}
		if l.ShopVNum != 0 {
			return fmt.Errorf("ObjInInventory should not have ShopVNum")
		}
	case ObjEquipped:
		if l.OwnerKind == OwnerPlayer && l.PlayerName == "" {
			return fmt.Errorf("OwnerPlayer requires non-empty PlayerName")
		}
		if l.OwnerKind == OwnerMob && l.MobID <= 0 {
			return fmt.Errorf("OwnerMob requires positive MobID, got %d", l.MobID)
		}
		if l.OwnerKind == OwnerPlayer && l.MobID != 0 {
			return fmt.Errorf("ObjEquipped with OwnerPlayer should not have MobID")
		}
		if l.OwnerKind == OwnerMob && l.PlayerName != "" {
			return fmt.Errorf("ObjEquipped with OwnerMob should not have PlayerName")
		}
		if l.Slot < 0 {
			return fmt.Errorf("ObjEquipped requires non-negative Slot, got %d", l.Slot)
		}
		if l.RoomVNum != 0 {
			return fmt.Errorf("ObjEquipped should not have RoomVNum")
		}
		if l.ContainerObjID != 0 {
			return fmt.Errorf("ObjEquipped should not have ContainerObjID")
		}
		if l.ShopVNum != 0 {
			return fmt.Errorf("ObjEquipped should not have ShopVNum")
		}
	case ObjInContainer:
		if l.ContainerObjID <= 0 {
			return fmt.Errorf("ObjInContainer requires positive ContainerObjID")
		}
		if l.OwnerKind != OwnerNone {
			return fmt.Errorf("ObjInContainer should not have an owner kind")
		}
		if l.RoomVNum != 0 {
			return fmt.Errorf("ObjInContainer should not have RoomVNum")
		}
		if l.PlayerName != "" {
			return fmt.Errorf("ObjInContainer should not have PlayerName")
		}
		if l.MobID != 0 {
			return fmt.Errorf("ObjInContainer should not have MobID")
		}
		if l.ShopVNum != 0 {
			return fmt.Errorf("ObjInContainer should not have ShopVNum")
		}
	case ObjInShop:
		if l.ShopVNum <= 0 {
			return fmt.Errorf("ObjInShop requires positive ShopVNum")
		}
		if l.OwnerKind != OwnerNone {
			return fmt.Errorf("ObjInShop should not have an owner kind")
		}
		if l.PlayerName != "" {
			return fmt.Errorf("ObjInShop should not have PlayerName")
		}
		if l.MobID != 0 {
			return fmt.Errorf("ObjInShop should not have MobID")
		}
		if l.ContainerObjID != 0 {
			return fmt.Errorf("ObjInShop should not have ContainerObjID")
		}
	}
	return nil
}

func (l ObjectLocation) OwnerIsPlayer() bool { return l.OwnerKind == OwnerPlayer }
func (l ObjectLocation) OwnerIsMob() bool    { return l.OwnerKind == OwnerMob }
func (l ObjectLocation) IsZero() bool        { return l.Kind == ObjNowhere }
