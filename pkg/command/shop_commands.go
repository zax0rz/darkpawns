// Package command implements shop-related commands for Dark Pawns.
package command

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

// ShopCommands provides shop-related command handlers.
type ShopCommands struct {
	shopManager *systems.ShopManager
	world       *game.World
}

// NewShopCommands creates a new ShopCommands instance.
func NewShopCommands(sm *systems.ShopManager, w *game.World) *ShopCommands {
	return &ShopCommands{
		shopManager: sm,
		world:       w,
	}
}

// getPlayer helper function to safely get player from session
func (sc *ShopCommands) getPlayer(s common.CommandSession) (*game.Player, error) {
	if !s.HasPlayer() {
		return nil, fmt.Errorf("you must be logged in to use this command")
	}

	playerInterface := s.GetPlayer()
	if playerInterface == nil {
		return nil, fmt.Errorf("internal error: invalid player object")
	}
	player, ok := playerInterface.(*game.Player)
	if !ok {
		return nil, fmt.Errorf("internal error: invalid player type")
	}
	return player, nil
}

// CmdListShop handles the 'list' command to show shop inventory.
func (sc *ShopCommands) CmdListShop(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Get shop inventory
	inventory := shop.GetInventory()

	if len(inventory) == 0 {
		s.Send("The shop has nothing for sale.\r\n")
		return nil
	}

	// Display shop inventory
	var output strings.Builder
	fmt.Fprintf(&output, "%s's inventory:\r\n", shop.Name)
	output.WriteString("----------------------------------------\r\n")

	for i, item := range inventory {
		price := shop.CalculateSellPrice(item)
		fmt.Fprintf(&output, "%2d) %-30s %5d gold\r\n",
			i+1, item.GetShortDesc(), price)
	}

	output.WriteString("----------------------------------------\r\n")
	fmt.Fprintf(&output, "You have %d gold.\r\n", player.Gold)

	s.Send(output.String())
	return nil
}

// CmdBuy handles the 'buy' command to purchase items from a shop.
func (sc *ShopCommands) CmdBuy(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: buy <item number|item name>")
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Get shop inventory
	inventory := shop.GetInventory()

	if len(inventory) == 0 {
		return fmt.Errorf("the shop has nothing for sale")
	}

	// Try to parse as item number first
	var item common.ObjectInstance
	if itemNum, err := parseItemNumber(args[0]); err == nil {
		// Item number is 1-indexed for display
		if itemNum < 1 || itemNum > len(inventory) {
			return fmt.Errorf("item number %d is not available", itemNum)
		}
		item = inventory[itemNum-1]
	} else {
		// Try to find by name
		var found bool
		item, found = shop.FindItem(strings.Join(args, " "))
		if !found {
			return fmt.Errorf("'%s' is not for sale", strings.Join(args, " "))
		}
	}

	// Process the transaction
	success, message := sc.shopManager.ProcessTransaction(shop, player, item, true)

	if !success {
		return fmt.Errorf("%s", message)
	}

	s.Send(message + "\r\n")
	return nil
}

// CmdSell handles the 'sell' command to sell items to a shop.
func (sc *ShopCommands) CmdSell(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: sell <item name>")
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Find item in player's inventory
	itemName := strings.Join(args, " ")
	item, found := player.Inventory.FindItem(itemName)
	if !found {
		return fmt.Errorf("you don't have '%s'", itemName)
	}

	// Process the transaction
	success, message := sc.shopManager.ProcessTransaction(shop, player, item, false)

	if !success {
		return fmt.Errorf("%s", message)
	}

	s.Send(message + "\r\n")
	return nil
}

// CmdRepair handles the 'repair' command to repair items at a shop.
func (sc *ShopCommands) CmdRepair(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: repair <item name>")
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Find item in player's inventory
	itemName := strings.Join(args, " ")
	item, found := player.Inventory.FindItem(itemName)
	if !found {
		return fmt.Errorf("you don't have '%s'", itemName)
	}

	// Check if shop can repair this type of item
	// For now, assume all shops can repair all items

	// Calculate damage (in a real implementation, we'd track item condition)
	// For now, use a fixed damage value
	damage := 10

	// Process the repair
	success, message := sc.shopManager.ProcessRepair(shop, player, item, damage)

	if !success {
		return fmt.Errorf("%s", message)
	}

	s.Send(message + "\r\n")
	return nil
}

// CmdIdentify handles the 'identify' command to identify items at a shop.
func (sc *ShopCommands) CmdIdentify(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: identify <item name>")
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Find item in player's inventory
	itemName := strings.Join(args, " ")
	item, found := player.Inventory.FindItem(itemName)
	if !found {
		return fmt.Errorf("you don't have '%s'", itemName)
	}

	// Process the identification
	success, message := sc.shopManager.ProcessIdentify(shop, player, item)

	if !success {
		return fmt.Errorf("%s", message)
	}

	s.Send(message + "\r\n")
	return nil
}

// CmdValue handles the 'value' command to check an item's buy/sell price.
func (sc *ShopCommands) CmdValue(s common.CommandSession, args []string) error {
	player, err := sc.getPlayer(s)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: value <item name>")
	}

	// Get player's current room
	roomVNum := s.GetPlayerRoomVNum()
	if roomVNum <= 0 {
		return fmt.Errorf("you are not in a valid room")
	}

	// Get shops in the room
	shops := sc.shopManager.GetShopsInRoomConcrete(roomVNum)
	if len(shops) == 0 {
		return fmt.Errorf("there are no shops here")
	}

	// For now, use the first shop in the room
	shop := shops[0]

	// Find item in player's inventory
	itemName := strings.Join(args, " ")
	item, found := player.Inventory.FindItem(itemName)
	if !found {
		return fmt.Errorf("you don't have '%s'", itemName)
	}

	// Check if shop buys this type of item
	if !shop.CanBuyType(item.GetTypeFlag()) {
		s.Send(fmt.Sprintf("The shopkeeper isn't interested in %s.\r\n", item.GetShortDesc()))
		return nil
	}

	// Calculate prices
	buyPrice := shop.CalculateBuyPrice(item)
	sellPrice := shop.CalculateSellPrice(item)

	s.Send(fmt.Sprintf("%s:\r\n", item.GetShortDesc()))
	s.Send(fmt.Sprintf("  Shop will buy for:  %5d gold\r\n", buyPrice))
	s.Send(fmt.Sprintf("  Shop sells for:     %5d gold\r\n", sellPrice))
	s.Send(fmt.Sprintf("  Base value:         %5d gold\r\n", item.GetCost()))

	return nil
}

// parseItemNumber parses a string as an item number.
func parseItemNumber(s string) (int, error) {
	// Simple implementation - in a real implementation,
	// we'd use strconv.Atoi and handle errors
	var num int
	_, err := fmt.Sscanf(s, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("not a valid number")
	}
	return num, nil
}

// RegisterCommands registers shop commands with the command manager.
func (sc *ShopCommands) RegisterCommands(manager common.CommandManager) {
	// Register shop commands
	manager.RegisterCommand("list", sc.CmdListShop)
	manager.RegisterCommand("buy", sc.CmdBuy)
	manager.RegisterCommand("sell", sc.CmdSell)
	manager.RegisterCommand("repair", sc.CmdRepair)
	manager.RegisterCommand("identify", sc.CmdIdentify)
	manager.RegisterCommand("value", sc.CmdValue)
}
