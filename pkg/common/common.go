// Package common provides shared interfaces and types to break circular dependencies.
package common

// ShopManager defines the interface for shop management.
type ShopManager interface {
	CreateShop(vnum int, name string, roomVNum int) interface{}
	GetShop(id int) (interface{}, bool)
	GetShopByNPC(vnum int) (interface{}, bool)
	GetShopsInRoom(roomVNum int) []interface{}
}

// ObjectInstance defines the interface for object instances.
type ObjectInstance interface {
	GetCost() int
	GetTypeFlag() int
	GetShortDesc() string
	GetLongDesc() string
	GetKeywords() string
	GetWeight() int
	GetVNum() int
	GetRoomVNum() int
	SetRoomVNum(int)
	IsContainer() bool
	IsWearable() bool
	IsWeapon() bool
	IsArmor() bool
}

// Session defines the interface for game sessions.
type Session interface {
	GetPlayer() interface{}
	SendText(string)
	IsAuthenticated() bool
	GetPlayerName() string
}

// Affectable defines entities that can have affects applied to them.
type Affectable interface {
	GetAffects() interface{}
	SetAffects(interface{})
	GetName() string
	GetID() int
	GetStrength() int
	SetStrength(int)
	GetDexterity() int
	SetDexterity(int)
	GetIntelligence() int
	SetIntelligence(int)
	GetWisdom() int
	SetWisdom(int)
	GetConstitution() int
	SetConstitution(int)
	GetCharisma() int
	SetCharisma(int)
	GetHitRoll() int
	SetHitRoll(int)
	GetDamageRoll() int
	SetDamageRoll(int)
	GetArmorClass() int
	SetArmorClass(int)
	GetTHAC0() int
	SetTHAC0(int)
	GetHP() int
	SetHP(int)
	GetMaxHP() int
	SetMaxHP(int)
	GetMana() int
	SetMana(int)
	GetMaxMana() int
	SetMaxMana(int)
	GetMovement() int
	SetMovement(int)
	HasStatusFlag(flag uint64) bool
	SetStatusFlag(flag uint64)
	ClearStatusFlag(flag uint64)
	SendMessage(string)
}
