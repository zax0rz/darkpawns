// Package common provides shared interfaces and types to break circular dependencies
// between packages like game, engine, session, command, and world.
package common

// ShopManagerInterface defines the interface for shop management.
// This allows game package to work with shops without importing the world package.
type ShopManagerInterface interface {
	// CreateShop creates a new shop
	CreateShop(vnum int, name string, roomVNum int) interface{} // Returns *world.Shop
	
	// GetShop returns a shop by ID
	GetShop(id int) (interface{}, bool) // Returns (*world.Shop, bool)
	
	// GetShopByNPC returns a shop by NPC VNum
	GetShopByNPC(vnum int) (interface{}, bool) // Returns (*world.Shop, bool)
	
	// GetShopsInRoom returns all shops in a room
	GetShopsInRoom(roomVNum int) []interface{} // Returns []*world.Shop
	
	// RemoveShop removes a shop by ID
	RemoveShop(id int) bool
	
	// GetAllShops returns all shops
	GetAllShops() []interface{} // Returns []*world.Shop
	
	// ProcessShopRestock processes restocking for all shops
	ProcessShopRestock(currentTick int)
	
	// FindShopForTransaction finds a shop that can handle a transaction
	FindShopForTransaction(roomVNum, npcVNum int, isBuy bool, itemType int) (interface{}, bool) // Returns (*world.Shop, bool)
}