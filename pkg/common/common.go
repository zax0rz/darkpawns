// Package common provides shared interfaces and types to break circular dependencies.
package common

// ShopManager defines the interface for shop management.
// Deprecated: Use ShopManagerInterface from shop.go instead. It exposes the
// full API including removal, restock, and transaction lookup.
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
