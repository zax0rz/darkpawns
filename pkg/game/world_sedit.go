package game

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CloneShop returns an independent shop definition suitable for a descriptor
// owned SEDIT working copy. ShopManager owns the live values; SEDIT must not
// expose those slices while a builder is editing them.
func CloneShop(shop Shop) Shop {
	clone := shop
	clone.BuyTypes = append([]int(nil), shop.BuyTypes...)
	clone.BuyWords = append([]string(nil), shop.BuyWords...)
	clone.SellTypes = append([]int(nil), shop.SellTypes...)
	clone.Rooms = append([]int(nil), shop.Rooms...)
	return clone
}

// SnapshotShop returns a deep copy of one shop prototype while holding the
// world lock. Shops are keyed by their virtual number, not by keeper VNUM.
func (w *World) SnapshotShop(vnum int) (Shop, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return Shop{}, false
	}
	shop := sm.GetShopByVNum(vnum)
	if shop == nil {
		return Shop{}, false
	}
	return CloneShop(*shop), true
}

// SnapshotShops returns deep copies ordered by shop VNUM, matching C's
// ascending shop_index walk used by sedit_save_to_disk.
func (w *World) SnapshotShops() []Shop {
	w.mu.RLock()
	defer w.mu.RUnlock()
	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return nil
	}
	shops := make([]Shop, 0, len(sm.shops))
	for _, shop := range sm.shops {
		if shop != nil {
			shops = append(shops, CloneShop(*shop))
		}
	}
	sort.Slice(shops, func(i, j int) bool { return shops[i].VNum < shops[j].VNum })
	return shops
}

// shopToProto maps the live manager shape back to the parser shape used by
// the zone writer and parsed-world bridge.
func shopToProto(shop Shop) parser.ShopProto {
	proto := parser.ShopProto{
		VNum:       shop.VNum,
		Products:   append([]int(nil), shop.SellTypes...),
		BuyProfit:  shop.ProfitBuy,
		SellProfit: shop.ProfitSell,
		BuyTypes:   append([]int(nil), shop.BuyTypes...),
		BuyWords:   append([]string(nil), shop.BuyWords...),
		Messages:   shop.Messages,
		Temper:     shop.Temper,
		Bitvector:  shop.Flags,
		KeeperVNum: shop.KeeperVNum,
		WithWho:    shop.WithWho,
		Rooms:      append([]int(nil), shop.Rooms...),
		OpenHour1:  shop.OpenHour1,
		CloseHour1: shop.CloseHour1,
		OpenHour2:  shop.OpenHour2,
		CloseHour2: shop.CloseHour2,
	}
	return proto
}

// protoToShop maps the parser shape to the live manager shape. The first room
// is retained in RoomVNum for the existing command-layer lookup adapter; the
// full list remains available to the editor and writer.
func protoToShop(proto parser.ShopProto) *Shop {
	shop := &Shop{
		VNum:       proto.VNum,
		KeeperVNum: proto.KeeperVNum,
		BuyTypes:   append([]int(nil), proto.BuyTypes...),
		BuyWords:   append([]string(nil), proto.BuyWords...),
		SellTypes:  append([]int(nil), proto.Products...),
		ProfitBuy:  proto.BuyProfit,
		ProfitSell: proto.SellProfit,
		Flags:      proto.Bitvector,
		Messages:   proto.Messages,
		Temper:     proto.Temper,
		WithWho:    proto.WithWho,
		Rooms:      append([]int(nil), proto.Rooms...),
		OpenHour1:  proto.OpenHour1,
		CloseHour1: proto.CloseHour1,
		OpenHour2:  proto.OpenHour2,
		CloseHour2: proto.CloseHour2,
	}
	if len(proto.Rooms) > 0 {
		shop.RoomVNum = proto.Rooms[0]
	}
	return shop
}

// CommitEditedShop atomically replaces or inserts a shop definition. The
// manager is the live shop authority; parsedData is updated for later exports.
// Rebuilding shopKeepers mirrors C's keeper-special assignment after a shop
// definition changes, including removal of the old keeper when KeeperVNum is
// edited. Existing standing keepers therefore observe the new shop immediately
// through the same manager pointer used by list/buy/sell.
func (w *World) CommitEditedShop(proto parser.ShopProto) bool {
	shop := protoToShop(proto)
	w.mu.Lock()
	defer w.mu.Unlock()
	sm, ok := w.shopManager.(*ShopManager)
	if !ok {
		return false
	}
	if keeper, ok := w.mobs[proto.KeeperVNum]; ok {
		shop.KeeperName = keeper.ShortDesc
	}
	sm.ReplaceShop(shop)
	if w.parsedData != nil {
		index := -1
		for i := range w.parsedData.Shops {
			if w.parsedData.Shops[i].VNum == proto.VNum {
				index = i
				break
			}
		}
		if index < 0 {
			w.parsedData.Shops = append(w.parsedData.Shops, proto)
		} else {
			w.parsedData.Shops[index] = proto
		}
	}
	w.shopKeepers = make(map[int]int, len(sm.shops))
	for _, live := range sm.shops {
		if live != nil && live.KeeperVNum >= 0 {
			w.shopKeepers[live.KeeperVNum] = live.Flags
		}
	}
	return true
}
