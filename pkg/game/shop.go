package game

// Shop represents a shopkeeper's shop, matching the CircleMUD shop_data structure.
// Source: src/shop.h
type Shop struct {
	KeeperVNum int     // VNUM of the NPC shopkeeper
	BuyTypes   []int   // item types this shop buys (list of TypeFlag values)
	SellTypes  []int   // VNUMs of item prototypes this shop sells
	ProfitBuy  float64 // markup factor when player buys (e.g., 1.20 = 120%)
	ProfitSell float64 // markup factor when player sells (e.g., 0.80 = 80%)
	Flags      int     // shop behavior flags (optional)
	KeeperName string  // name of keeper for message formatting
}

// ShopManager holds all shops and provides lookup methods.
type ShopManager struct {
	shops []*Shop
}

// NewShopManager creates a new ShopManager.
func NewShopManager() *ShopManager {
	return &ShopManager{shops: make([]*Shop, 0)}
}

// AddShop adds a shop to the manager.
func (sm *ShopManager) AddShop(s *Shop) {
	sm.shops = append(sm.shops, s)
}

// GetShopByKeeper returns the first shop run by the given NPC VNUM.
func (sm *ShopManager) GetShopByKeeper(vnum int) *Shop {
	for _, s := range sm.shops {
		if s.KeeperVNum == vnum {
			return s
		}
	}
	return nil
}

// GetShopsByKeeper returns all shops run by the given NPC VNUM (usually just one).
func (sm *ShopManager) GetShopsByKeeper(vnum int) []*Shop {
	var result []*Shop
	for _, s := range sm.shops {
		if s.KeeperVNum == vnum {
			result = append(result, s)
		}
	}
	return result
}

// GetAllShops returns all registered shops.
func (sm *ShopManager) GetAllShops() []*Shop {
	return sm.shops
}

// CreateShop implements common.ShopManager.CreateShop.
func (sm *ShopManager) CreateShop(vnum int, name string, roomVNum int) interface{} {
	shop := &Shop{
		KeeperVNum: vnum,
		KeeperName: name,
		BuyTypes:   make([]int, 0),
		SellTypes:  make([]int, 0),
		ProfitBuy:  1.0,
		ProfitSell: 1.0,
	}
	sm.AddShop(shop)
	return shop
}

// GetShop implements common.ShopManager.GetShop (stub — shops don't have IDs).
func (sm *ShopManager) GetShop(id int) (interface{}, bool) {
	if id >= 0 && id < len(sm.shops) {
		return sm.shops[id], true
	}
	return nil, false
}

// GetShopByNPC implements common.ShopManager.GetShopByNPC.
func (sm *ShopManager) GetShopByNPC(vnum int) (interface{}, bool) {
	shop := sm.GetShopByKeeper(vnum)
	if shop == nil {
		return nil, false
	}
	return shop, true
}

// GetShopsInRoom implements common.ShopManager.GetShopsInRoom (stub).
func (sm *ShopManager) GetShopsInRoom(roomVNum int) []interface{} {
	return nil
}

// BuyPrice calculates the price a player pays to buy an item from the shop.
// Matches src/shop.c buy_price() exactly:
//
//	price = (int)(GET_OBJ_COST(obj) * SHOP_BUYPROFIT(shop_nr))
//	if (GET_CHA(ch)) price -= price*(GET_CHA(ch)*.005)
//	return MAX(MAX(price, 1), GET_OBJ_COST(obj))
func (s *Shop) BuyPrice(itemCost int, cha int) int {
	price := float64(itemCost) * s.ProfitBuy

	// CHA discount: price -= price * (CHA * 0.005)
	if cha > 0 {
		price -= price * (float64(cha) * 0.005)
	}

	// C: MAX(MAX(price, 1), GET_OBJ_COST(obj)) — buy price is at least item cost
	if price < 1 {
		price = 1
	}
	if float64(itemCost) > price {
		price = float64(itemCost)
	}
	return int(price)
}

// SellPrice calculates the price a shop pays the player for an item.
// Matches src/shop.c sell_price() exactly:
//
//	price = (int)(GET_OBJ_COST(obj) * SHOP_SELLPROFIT(shop_nr))
//	if (GET_CHA(ch)) price += price*(GET_CHA(ch)*.005)
//	if ((bprice = buy_price(ch, obj, shop_nr)) < price) price = bprice
//	return MIN(MAX(1, price), GET_OBJ_COST(obj))
func (s *Shop) SellPrice(itemCost int, cha int) int {
	price := float64(itemCost) * s.ProfitSell

	// CHARISMA modifier: price += price * (CHA * 0.005)
	if cha > 0 {
		price += price * (float64(cha) * 0.005)
	}

	// C: if buy_price < price, cap at buy_price
	buyPrice := s.BuyPrice(itemCost, cha)
	if buyPrice < int(price) {
		price = float64(buyPrice)
	}

	// C: MIN(MAX(1, price), GET_OBJ_COST(obj))
	if price < 1 {
		price = 1
	}
	if price > float64(itemCost) {
		price = float64(itemCost)
	}
	return int(price)
}

// WillBuyType returns true if the shop buys items of the given type.
func (s *Shop) WillBuyType(itemType int) bool {
	for _, t := range s.BuyTypes {
		if t == itemType {
			return true
		}
	}
	return false
}

// HasSellItem returns true if the shop sells an item with the given prototype VNUM.
func (s *Shop) HasSellItem(vnum int) bool {
	for _, v := range s.SellTypes {
		if v == vnum {
			return true
		}
	}
	return false
}
