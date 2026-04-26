// Package game manages the game world state and player interactions.
package game

import (
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ObjectInstance represents a spawned object in the world.
type ObjectInstance struct {
	// Instance identity
	ID    int // unique instance ID, assigned by World
	VNum  int // prototype VNum

	// Link to prototype
	Prototype *parser.Obj

	// Location
	RoomVNum  int             // -1 if not in a room

	Location      ObjectLocation

	// Contents (for containers)
	Contains []*ObjectInstance

	// Custom state
	CustomData map[string]interface{}

	// Runtime state — typed replacement for CustomData
	Runtime ObjectRuntimeState

	// Timer — ticks until object decays (0 = permanent/no timer)
	Timer int

	// Runtime flags
	IsCorpse  bool // true if this is a corpse object
	CanPickUp bool // true if ITEM_WEAR_TAKE flag is set

	// Instance-level overrides for enchantment spells.
	// When non-nil, these override the prototype values.
	// Source: src/spells.c spell_enchant_weapon/armor, spell_silken_missile.
	ExtraFlagsOverride [4]int
	AffectsOverride    []parser.ObjAffect
	ValuesOverride     *[4]int // copy-on-write override of prototype Values
}

// NewObjectInstance creates a new object instance from a prototype.
func NewObjectInstance(proto *parser.Obj, roomVNum int) *ObjectInstance {
	obj := &ObjectInstance{
		Prototype:     proto,
		VNum:          proto.VNum,
		RoomVNum:      roomVNum,
		Contains:      make([]*ObjectInstance, 0),
		CustomData:    make(map[string]interface{}),
		Runtime:       ObjectRuntimeState{},
	}
	if roomVNum > 0 {
		obj.Location = LocRoom(roomVNum)
	} else {
		obj.Location = LocNowhere()
	}
	return obj
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

	o.Contains = append(o.Contains, obj)
	return true
}

// RemoveFromContainer removes an object from this container.
func (o *ObjectInstance) RemoveFromContainer(obj *ObjectInstance) bool {
	for i, item := range o.Contains {
		if item == obj {
			o.Contains = append(o.Contains[:i], o.Contains[i+1:]...)
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
	if o.AffectsOverride != nil {
		return o.AffectsOverride
	}
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

// MigrateCustomData copies known keys from CustomData to Runtime and deletes them
// from CustomData. It is safe to call multiple times. Designed for save/load
// backward compatibility during the transition from untyped map to typed struct.
func (o *ObjectInstance) MigrateCustomData() {
	if o.CustomData == nil {
		return
	}

	type strCopy struct {
		key string
		dst *string
	}
	for _, sc := range []strCopy{
		{"name", &o.Runtime.Name},
		{"short_desc", &o.Runtime.ShortDesc},
		{"long_desc", &o.Runtime.LongDesc},
		{"short_desc_override", &o.Runtime.ShortDescOverride},
		{"mold_name", &o.Runtime.MoldName},
		{"mold_desc", &o.Runtime.MoldDesc},
		{"mail_text", &o.Runtime.MailText},
	} {
		if v, ok := o.CustomData[sc.key]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				*sc.dst = s
			}
			delete(o.CustomData, sc.key)
		}
	}

	// Horse state: extract int keys (check in separate block)
	hasCarryW := false
	hasCarryN := false
	hasMove := false
	hasMaxMove := false
	if _, ok := o.CustomData["carryW"]; ok {
		hasCarryW = true
	}
	if _, ok := o.CustomData["carryN"]; ok {
		hasCarryN = true
	}
	if _, ok := o.CustomData["move"]; ok {
		hasMove = true
	}
	if _, ok := o.CustomData["maxMove"]; ok {
		hasMaxMove = true
	}

	if hasCarryW || hasCarryN || hasMove || hasMaxMove {
		if o.Runtime.Horse == nil {
			o.Runtime.Horse = &HorseState{}
		}

		for key, dst := range map[string]*int{
			"carryW":  &o.Runtime.Horse.CarryWeight,
			"carryN":  &o.Runtime.Horse.CarryNumber,
			"move":    &o.Runtime.Horse.Move,
			"maxMove": &o.Runtime.Horse.MaxMove,
		} {
			if v, ok := o.CustomData[key]; ok {
				if iv, ok2 := v.(int); ok2 {
					*dst = iv
				}
				delete(o.CustomData, key)
			}
		}
	}
}

// Scripting interface implementations

func (o *ObjectInstance) GetVNum() int {
	return o.VNum
}

// GetValue returns the object's Values[idx], preferring instance override.
func (o *ObjectInstance) GetValue(idx int) int {
	if o.ValuesOverride != nil && idx >= 0 && idx < len(*o.ValuesOverride) {
		return (*o.ValuesOverride)[idx]
	}
	if o.Prototype == nil || idx < 0 || idx >= len(o.Prototype.Values) {
		return 0
	}
	return o.Prototype.Values[idx]
}

// SetValue sets an object value at the given index. Creates a copy-on-write
// override so the prototype isn't mutated. Used by create_water and similar spells.
func (o *ObjectInstance) SetValue(idx, val int) {
	if o.Prototype == nil || idx < 0 || idx >= len(o.Prototype.Values) {
		return
	}
	// Copy-on-write: create instance values override when first mutating
	if o.ValuesOverride == nil {
		copy := o.Prototype.Values
		o.ValuesOverride = &copy
	}
	o.ValuesOverride[idx] = val
}

func (o *ObjectInstance) GetRoomVNum() int {
	return o.RoomVNum
}

func (o *ObjectInstance) SetRoomVNum(roomVNum int) {
	o.RoomVNum = roomVNum
	if roomVNum > 0 {
		o.Location = LocRoom(roomVNum)
	} else {
		o.Location = LocNowhere()
	}
}

func (o *ObjectInstance) GetTimer() int {
	return o.Timer
}

func (o *ObjectInstance) SetTimer(timer int) {
	o.Timer = timer
}

// GetExtraFlags returns the effective extra flags, preferring instance overrides.
// Returns all 4 flag words as a slice.
func (o *ObjectInstance) GetExtraFlags() [4]int {
	if o.ExtraFlagsOverride != [4]int{} {
		return o.ExtraFlagsOverride
	}
	if o.Prototype != nil {
		return o.Prototype.ExtraFlags
	}
	return [4]int{}
}

// SetExtraFlag sets a bit in the instance-level extra flags override.
// word is the flag word index (0-3), bit is the bit position.
func (o *ObjectInstance) SetExtraFlag(word, bit int) {
	o.ExtraFlagsOverride[word] |= (1 << uint(bit))
}

// HasExtraFlag checks if a bit is set in the effective extra flags.
func (o *ObjectInstance) HasExtraFlag(word, bit int) bool {
	ef := o.GetExtraFlags()
	return ef[word]&(1<<uint(bit)) != 0
}

// SetAffectsOverride sets the instance-level affect overrides.
func (o *ObjectInstance) SetAffectsOverride(affects []parser.ObjAffect) {
	o.AffectsOverride = affects
}
