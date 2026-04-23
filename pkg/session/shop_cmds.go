package session

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Shop commands: list, buy, sell
// Source: src/shop.c — shopping_list, shopping_buy, shopping_sell
// ---------------------------------------------------------------------------

// findShopKeeperInRoom scans mobs in the current room for a shopkeeper NPC
// and returns the matching Shop if found.
func findShopKeeperInRoom(s *Session) (*game.Shop, string) {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return nil, ""
	}

	roomVNum := s.player.GetRoomVNum()
	if roomVNum < 0 {
		return nil, ""
	}

	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	if len(mobs) == 0 {
		return nil, ""
	}

	// Check each mob — if its VNum matches a shop keeper, return that shop
	for _, mob := range mobs {
		if shop, ok := s.manager.world.GetShopByKeeper(mob.VNum); ok {
			name := mob.GetShortDesc()
			if name == "" {
				name = "The shopkeeper"
			}
			return shop, name
		}
	}

	return nil, ""
}

// cmdList lists items for sale at the shop.
// Usage: list [keyword]
func cmdList(s *Session, args []string) error {
	shop, keeperName := findShopKeeperInRoom(s)
	if shop == nil {
		s.Send("There is no shop here.")
		return nil
	}

	keyword := ""
	if len(args) > 0 {
		keyword = strings.ToLower(strings.Join(args, " "))
	}

	if len(shop.SellTypes) == 0 {
		s.Send(fmt.Sprintf("%s has nothing for sale right now.", keeperName))
		return nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%s has the following items for sale:", keeperName))
	lines = append(lines, "----------------------------------------")
	lines = append(lines, fmt.Sprintf(" %-4s %-48s %6s", "##", "Item", "Cost"))

	found := false
	index := 0
	for _, vnum := range shop.SellTypes {
		proto, ok := s.manager.world.GetObjPrototype(vnum)
		if !ok {
			continue
		}

		// Filter by keyword if given
		if keyword != "" {
			if !strings.Contains(strings.ToLower(proto.Keywords), keyword) &&
				!strings.Contains(strings.ToLower(proto.ShortDesc), keyword) {
				continue
			}
		}

		index++
		price := shop.BuyPrice(proto.Cost, s.player.Stats.Cha)
		lines = append(lines, fmt.Sprintf(" %2d)  %-48s %6d", index, proto.ShortDesc, price))
		found = true
	}

	if !found {
		if keyword != "" {
			s.Send("The shop has nothing like that.")
		} else {
			s.Send(fmt.Sprintf("%s has nothing for sale.", keeperName))
		}
		return nil
	}

	s.Send(strings.Join(lines, "\r\n"))
	return nil
}

// cmdBuy buys an item from the shop.
// Usage: buy <item> [count]
func cmdBuy(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Buy what?")
		return nil
	}

	shop, keeperName := findShopKeeperInRoom(s)
	if shop == nil {
		s.Send("There is no shop here.")
		return nil
	}

	// Parse the item name and optional count
	// "buy sword 3" — count is last arg if it's a number
	itemName := strings.Join(args, " ")
	count := 1
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[len(args)-1]); err == nil && n > 0 {
			count = n
			itemName = strings.Join(args[:len(args)-1], " ")
		}
	}

	// Find the item prototype in the shop's sell list by keyword match
	var matchedProto *parser.Obj
	for _, vnum := range shop.SellTypes {
		proto, ok := s.manager.world.GetObjPrototype(vnum)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(proto.Keywords), strings.ToLower(itemName)) ||
			strings.Contains(strings.ToLower(proto.ShortDesc), strings.ToLower(itemName)) {
			matchedProto = proto
			break
		}
	}

	if matchedProto == nil {
		s.Send(fmt.Sprintf("%s says, 'I don't have that item.'", keeperName))
		return nil
	}

	// Calculate price per item
	pricePerItem := shop.BuyPrice(matchedProto.Cost, s.player.Stats.Cha)
	totalPrice := pricePerItem * count

	// Check if player can afford
	if s.player.Gold < totalPrice {
		s.Send("You can't afford it!")
		return nil
	}

	// Buy up to count items (stop if inventory is full or gold runs out)
	bought := 0
	for i := 0; i < count; i++ {
		if s.player.Inventory.IsFull() {
			if bought == 0 {
				s.Send("You can't carry any more items.")
				return nil
			}
			break
		}

		// Check gold before creating item
		if s.player.Gold < pricePerItem {
			if bought == 0 {
				s.Send("You can't afford it!")
				return nil
			}
			break
		}

		// Create item
		item := game.NewObjectInstance(matchedProto, -1)
		if err := s.player.Inventory.AddItem(item); err != nil {
			break
		}
		s.player.Gold -= pricePerItem
		bought++
	}

	if bought > 0 {
		totalSpent := pricePerItem * bought
		if bought == 1 {
			s.Send(fmt.Sprintf("You buy %s for %d gold pieces.", matchedProto.ShortDesc, totalSpent))
		} else {
			s.Send(fmt.Sprintf("You buy %s (x%d) for %d gold pieces.", matchedProto.ShortDesc, bought, totalSpent))
		}
		s.markDirty(VarInventory)
	}

	return nil
}

// cmdSell sells an item to the shop.
// Usage: sell <item>  or  sell all
func cmdSell(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Sell what?")
		return nil
	}

	shop, keeperName := findShopKeeperInRoom(s)
	if shop == nil {
		s.Send("There is no shop here.")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Handle "sell all"
	if strings.EqualFold(itemName, "all") {
		return cmdSellAll(s, shop, keeperName)
	}

	// Find item in player's inventory by keyword
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.Send(fmt.Sprintf("%s says, 'You don't have that item.'", keeperName))
		return nil
	}

	// Check if shop buys this item type
	if !shop.WillBuyType(item.GetTypeFlag()) {
		s.Send("The shopkeeper doesn't want that.")
		return nil
	}

	// Calculate sell price
	price := shop.SellPrice(item.GetCost(), s.player.Stats.Cha)

	// Remove item from player, add gold
	if s.player.Inventory.RemoveItem(item) {
		s.player.Gold += price
		s.Send(fmt.Sprintf("You sell %s for %d gold pieces.", item.GetShortDesc(), price))
		s.markDirty(VarInventory)
	} else {
		s.Send("You can't sell that.")
	}

	return nil
}

// cmdSellAll sells all items the shop will buy from the player's inventory.
func cmdSellAll(s *Session, shop *game.Shop, keeperName string) error {
	items := s.player.Inventory.FindItems("")
	if len(items) == 0 {
		s.Send("You have nothing to sell.")
		return nil
	}

	sold := 0
	totalGold := 0
	var soldNames []string

	for _, item := range items {
		if !shop.WillBuyType(item.GetTypeFlag()) {
			continue
		}

		price := shop.SellPrice(item.GetCost(), s.player.Stats.Cha)
		if s.player.Inventory.RemoveItem(item) {
			s.player.Gold += price
			totalGold += price
			sold++
			soldNames = append(soldNames, item.GetShortDesc())
		}
	}

	if sold == 0 {
		s.Send("The shopkeeper doesn't want anything you have.")
		return nil
	}

	if sold == 1 {
		s.Send(fmt.Sprintf("You sell %s for %d gold pieces.", soldNames[0], totalGold))
	} else {
		s.Send(fmt.Sprintf("You sell %d items for %d gold pieces.", sold, totalGold))
	}
	s.markDirty(VarInventory)
	return nil
}
