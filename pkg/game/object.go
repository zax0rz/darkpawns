// Package game manages the game world state and player interactions.
package game

import (
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ObjectInstance represents a spawned object in the world.
type ObjectInstance struct {
	// Link to prototype
	Prototype *parser.Obj
	VNum      int

	// Location
	RoomVNum int // -1 if not in a room
	Carrier  interface{} // *MobInstance or *Player
	Container *ObjectInstance // if inside another object
	
	// Equipment state
	EquippedOn   interface{} // *MobInstance or *Player
	EquipPosition int // -1 if not equipped

	// Contents (for containers)
	Contains []*ObjectInstance

	// Custom state
	CustomData map[string]interface{}
}

// NewObjectInstance creates a new object instance from a prototype.
func NewObjectInstance(proto *parser.Obj, roomVNum int) *ObjectInstance {
	return &ObjectInstance{
		Prototype:  proto,
		VNum:       proto.VNum,
		RoomVNum:   roomVNum,
		Contains:   make([]*ObjectInstance, 0),
		CustomData: make(map[string]interface{}),
		EquipPosition: -1,
	}
}

// GetShortDesc returns the object's short description.
func (o *ObjectInstance) GetShortDesc() string {
	if o.Prototype != nil {
		return o.Prototype.ShortDesc
	}
	return "a generic object"
}

// GetLongDesc returns the object's long description.
func (o *ObjectInstance) GetLongDesc() string {
	if o.Prototype != nil {
		return o.Prototype.LongDesc
	}
	return "A generic object lies here."
}

// GetKeywords returns the object's keywords.
func (o *ObjectInstance) GetKeywords() string {
	if o.Prototype != nil {
		return o.Prototype.Keywords
	}
	return "object generic"
}

// GetWeight returns the object's weight.
func (o *ObjectInstance) GetWeight() int {
	if o.Prototype != nil {
		return o.Prototype.Weight
	}
	return 1
}

// GetCost returns the object's cost.
func (o *ObjectInstance) GetCost() int {
	if o.Prototype != nil {
		return o.Prototype.Cost
	}
	return 0
}

// GetTypeFlag returns the object's type flag.
func (o *ObjectInstance) GetTypeFlag() int {
	if o.Prototype != nil {
		return o.Prototype.TypeFlag
	}
	return 0
}

// IsContainer returns true if the object can contain other objects.
func (o *ObjectInstance) IsContainer() bool {
	// Check type flag for container types
	// Type 1 is container in CircleMUD
	return o.GetTypeFlag() == 1
}

// IsWearable returns true if the object can be worn.
func (o *ObjectInstance) IsWearable() bool {
	// Check wear flags
	if o.Prototype == nil {
		return false
	}
	
	// If any wear flag is non-zero, it's wearable
	for _, flag := range o.Prototype.WearFlags {
		if flag != 0 {
			return true
		}
	}
	return false
}

// IsWeapon returns true if the object is a weapon.
func (o *ObjectInstance) IsWeapon() bool {
	// Type 5 is weapon in CircleMUD
	return o.GetTypeFlag() == 5
}

// IsArmor returns true if the object is armor.
func (o *ObjectInstance) IsArmor() bool {
	// Type 9 is armor in CircleMUD
	return o.GetTypeFlag() == 9
}

// AddToContainer adds an object to this container.
func (o *ObjectInstance) AddToContainer(obj *ObjectInstance) bool {
	if !o.IsContainer() {
		return false
	}
	
	obj.Container = o
	o.Contains = append(o.Contains, obj)
	return true
}

// RemoveFromContainer removes an object from this container.
func (o *ObjectInstance) RemoveFromContainer(obj *ObjectInstance) bool {
	for i, item := range o.Contains {
		if item == obj {
			o.Contains = append(o.Contains[:i], o.Contains[i+1:]...)
			obj.Container = nil
			return true
		}
	}
	return false
}

// GetContents returns all objects inside this container.
func (o *ObjectInstance) GetContents() []*ObjectInstance {
	return o.Contains
}

// GetTotalWeight returns the total weight including contents.
func (o *ObjectInstance) GetTotalWeight() int {
	total := o.GetWeight()
	for _, item := range o.Contains {
		total += item.GetTotalWeight()
	}
	return total
}

// GetAffects returns the object's affect modifiers.
func (o *ObjectInstance) GetAffects() []parser.ObjAffect {
	if o.Prototype != nil {
		return o.Prototype.Affects
	}
	return nil
}

// GetExtraDescs returns the object's extra descriptions.
func (o *ObjectInstance) GetExtraDescs() []parser.ExtraDesc {
	if o.Prototype != nil {
		return o.Prototype.ExtraDescs
	}
	return nil
}

// GetExtraDesc returns an extra description matching the given keyword.
func (o *ObjectInstance) GetExtraDesc(keyword string) string {
	if o.Prototype == nil {
		return ""
	}
	
	for _, ed := range o.Prototype.ExtraDescs {
		// Simple keyword matching - in reality would need to parse keywords
		if ed.Keywords == keyword {
			return ed.Description
		}
	}
	return ""
}

// SetCustomData sets custom data on the object.
func (o *ObjectInstance) SetCustomData(key string, value interface{}) {
	if o.CustomData == nil {
		o.CustomData = make(map[string]interface{})
	}
	o.CustomData[key] = value
}

// GetCustomData gets custom data from the object.
func (o *ObjectInstance) GetCustomData(key string) interface{} {
	if o.CustomData == nil {
		return nil
	}
	return o.CustomData[key]
}

// Scripting interface implementations

func (o *ObjectInstance) GetVNum() int {
	return o.VNum
}

func (o *ObjectInstance) GetRoomVNum() int {
	return o.RoomVNum
}

func (o *ObjectInstance) SetRoomVNum(roomVNum int) {
	o.RoomVNum = roomVNum
}

func (o *ObjectInstance) GetCarrier() interface{} {
	return o.Carrier
}

func (o *ObjectInstance) SetCarrier(carrier interface{}) {
	o.Carrier = carrier
}

func (o *ObjectInstance) GetTimer() int {
	// TODO: Add timer field to ObjectInstance
	return 0
}

func (o *ObjectInstance) SetTimer(timer int) {
	// TODO: Add timer field to ObjectInstance
}