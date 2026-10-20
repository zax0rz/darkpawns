package session

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ---------------------------------------------------------------------------
// Shop commands: list, buy, sell
// Source: src/shop.c — shopping_list, shopping_buy, shopping_sell
// ---------------------------------------------------------------------------

func findShopKeeperAndMobInRoom(s *Session) (*game.Shop, string, *game.MobInstance) {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return nil, "", nil
	}

	roomVNum := s.player.GetRoomVNum()
	if roomVNum < 0 {
		return nil, "", nil
	}

	// src/interpreter.c:1452-1466 runs mobile specials in room-list order;
	// COrderedRoomMobs mirrors char_to_room's prepend order there.
	mobs := s.manager.world.COrderedRoomMobs(roomVNum)
	if len(mobs) == 0 {
		return nil, "", nil
	}

	// Check each mob — if its VNum matches a shop keeper, return that shop
	for _, mob := range mobs {
		if shop, ok := s.manager.world.GetShopByKeeper(mob.VNum); ok {
			name := mob.GetShortDesc()
			if name == "" {
				name = "The shopkeeper"
			}
			return shop, name, mob
		}
	}

	return nil, "", nil
}

// shopKeeperTellArgument builds the do_tell argument used by shop.c: the
// formatted string begins with the buyer's name, which C resolves through
// get_char_vis (handler.c:1276-1325).
func shopKeeperTellArgument(message, playerName string, values ...any) string {
	args := make([]any, 1, len(values)+1)
	args[0] = playerName
	args = append(args, values...)
	return fmt.Sprintf(message, args...)
}

func sendShopKeeperTell(s *Session, keeper *game.MobInstance, message, playerName string, values ...any) {
	if s == nil || s.manager == nil || s.manager.world == nil || keeper == nil {
		return
	}
	// shopping_buy/sell call do_tell(keeper, buf) at src/shop.c:580, 801;
	// preserve its room-order target resolution and directed-speech side effects.
	s.manager.world.NPCTellCommand(keeper, shopKeeperTellArgument(message, playerName, values...))
}

// shopTimesMessage mirrors shop.c:times_message for the one description and
// count that shopping_buy/shopping_sell expose after a completed transaction.
func shopTimesMessage(description string, count int) string {
	if count > 1 {
		return fmt.Sprintf("%s (x %d)", description, count)
	}
	return description
}

func shopObjectFirstKeyword(keywords string) string {
	if first, _, ok := strings.Cut(keywords, " "); ok {
		return first
	}
	return keywords
}

func shopKeeperSay(keeperName, message string) string {
	if keeperName == "" {
		keeperName = "The shopkeeper"
	} else {
		keeperName = capitalizeFirstRune(keeperName)
	}
	return fmt.Sprintf("%s says, '%s'\r\n", keeperName, message)
}

const (
	shopTradeNoGood = 1 << iota
	shopTradeNoEvil
	shopTradeNoNeutral
	shopTradeNoMage
	shopTradeNoCleric
	shopTradeNoThief
	shopTradeNoWarrior
)

func shopIsOK(s *Session, shop *game.Shop, keeper *game.MobInstance, keeperName string) bool {
	hours := game.TimeSnapshot().Hours
	closedMessage := ""
	if shop.OpenHour1 > hours {
		closedMessage = "Come back later!"
	} else if shop.CloseHour1 < hours {
		if shop.OpenHour2 > hours {
			closedMessage = "Sorry, we have closed, but come back later."
		} else if shop.CloseHour2 < hours {
			closedMessage = "Sorry, come back tomorrow."
		}
	}
	if closedMessage != "" {
		s.Send(shopKeeperSay(keeperName, closedMessage))
		return false
	}

	if !game.CanSee(keeper, s.player) {
		s.Send(shopKeeperSay(keeperName, "I don't trade with someone I can't see!"))
		return false
	}

	// C is_ok_char lets gods and NPCs bypass the trade restrictions after the
	// visibility check. The live command has a player actor, but retain the
	// source ordering at this boundary.
	if s.player.GetLevel() >= game.LVL_GOD || s.player.IsNPC() {
		return true
	}

	trade := shop.WithWho
	if (s.player.IsGood() && trade&shopTradeNoGood != 0) ||
		(s.player.IsEvil() && trade&shopTradeNoEvil != 0) ||
		(s.player.IsNeutral() && trade&shopTradeNoNeutral != 0) {
		sendShopKeeperTell(s, keeper, "%s Get out of here before I call the guards!", s.player.GetName())
		return false
	}

	if (s.player.GetClass() == game.ClassMageUser && trade&shopTradeNoMage != 0) ||
		(s.player.GetClass() == game.ClassCleric && trade&shopTradeNoCleric != 0) ||
		(s.player.GetClass() == game.ClassThief && trade&shopTradeNoThief != 0) ||
		(s.player.GetClass() == game.ClassWarrior && trade&shopTradeNoWarrior != 0) {
		sendShopKeeperTell(s, keeper, "%s We don't serve your kind here!", s.player.GetName())
		return false
	}
	return true
}

func shopListKeywordMatches(keyword, keywords string) bool {
	if keyword == "" || keywords == "" {
		return keyword == ""
	}
	keyword = strings.ToLower(keyword)
	// src/handler.c:isname() accepts a complete name token only. It does
	// not accept an abbreviation that ends in the middle of an alphabetic
	// token: "pepper" matches "pepper", while "pep" does not.
	for _, name := range strings.Fields(strings.ToLower(keywords)) {
		if name == keyword {
			return true
		}
	}
	return false
}

func shopObjectsSame(left, right *game.ObjectInstance) bool {
	if left == nil || right == nil || left.GetVNum() != right.GetVNum() || left.GetCost() != right.GetCost() {
		return false
	}
	if left.GetExtraFlags() != right.GetExtraFlags() {
		return false
	}
	leftAffects, rightAffects := left.GetAffects(), right.GetAffects()
	if len(leftAffects) != len(rightAffects) {
		return false
	}
	for i := range leftAffects {
		if leftAffects[i] != rightAffects[i] {
			return false
		}
	}
	return true
}

func shopObjectIsProduced(s *Session, shop *game.Shop, obj *game.ObjectInstance) bool {
	for _, vnum := range shop.SellTypes {
		proto, ok := s.manager.world.GetObjPrototype(vnum)
		if !ok {
			continue
		}
		if shopObjectsSame(obj, game.NewObjectInstance(proto, -1)) {
			return true
		}
	}
	return false
}

// findShopPurchaseObject follows get_slide_obj_vis/get_hash_obj_vis: matches
// complete keyword tokens, ignores invisible or zero-cost stock, and counts
// identical objects as one listed group for ordinal selection.
func findShopPurchaseObject(name string, inventory []*game.ObjectInstance, viewer *game.Player) *game.ObjectInstance {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	// get_purchase_obj calls one_argument before selecting either # or slide
	// syntax, so a trailing word after a hash selector is ignored by C.
	name = strings.Fields(name)[0]
	if strings.HasPrefix(name, "#") {
		index, err := strconv.Atoi(strings.TrimPrefix(name, "#"))
		if err != nil || index < 1 {
			return nil
		}
		last, group := (*game.ObjectInstance)(nil), 0
		for _, obj := range inventory {
			if obj == nil || obj.GetCost() <= 0 || !game.CanSeeObject(viewer, obj) {
				continue
			}
			if last != nil && shopObjectsSame(last, obj) {
				continue
			}
			group++
			if group == index {
				return obj
			}
			last = obj
		}
		return nil
	}

	ordinal := 1
	keyword := name
	if before, after, found := strings.Cut(name, "."); found {
		if n, err := strconv.Atoi(before); err == nil && n > 0 {
			ordinal, keyword = n, after
		}
	}
	fields := strings.Fields(keyword)
	if len(fields) == 0 {
		return nil
	}
	keyword = fields[0]
	last, group := (*game.ObjectInstance)(nil), 0
	for _, obj := range inventory {
		// get_slide_obj_vis can select zero-cost objects. get_purchase_obj
		// extracts them and repeats the search; the caller mirrors that loop.
		if obj == nil || !game.CanSeeObject(viewer, obj) || !shopListKeywordMatches(keyword, obj.GetKeywords()) {
			continue
		}
		if last != nil && shopObjectsSame(last, obj) {
			continue
		}
		group++
		if group == ordinal {
			return obj
		}
		last = obj
	}
	return nil
}

// shopPurchaseObject mirrors get_purchase_obj, including its side effect of
// extracting selected zero-cost stock before retrying the same selector.
func shopPurchaseObject(world *game.World, shopVNum int, keeper *game.MobInstance, name string, inventory []*game.ObjectInstance, viewer *game.Player) (*game.ObjectInstance, error) {
	for {
		stock := findShopPurchaseObject(name, inventory, viewer)
		if stock == nil || stock.GetCost() > 0 {
			return stock, nil
		}
		if err := world.DiscardShopPurchaseObject(keeper, stock); err != nil {
			return nil, err
		}
		var err error
		inventory, err = world.ShopInventory(shopVNum, keeper)
		if err != nil {
			return nil, err
		}
	}
}

func capitalizeFirstRune(value string) string {
	r, size := utf8.DecodeRuneInString(value)
	if size == 0 {
		return value
	}
	return string(unicode.ToUpper(r)) + value[size:]
}

func shopListLine(s *Session, shop *game.Shop, obj *game.ObjectInstance, count, index int) string {
	available := fmt.Sprintf("%5d       ", count)
	if shopObjectIsProduced(s, shop, obj) {
		available = "Unlimited   "
	}
	// src/shop.c:858-862: a drink container with liquid in it lists " of <drink>"
	// after its short description, and CAP() then uppercases the whole line.
	description := obj.GetShortDesc()
	if obj.GetTypeFlag() == game.ITEM_DRINKCON && obj.GetValue(1) != 0 {
		liquid := obj.GetValue(2)
		if liquid >= 0 && liquid < len(game.LiquidNames) {
			description += " of " + game.LiquidNames[liquid]
		}
	}
	description = capitalizeFirstRune(description)
	return fmt.Sprintf(" %2d)  %s%-48s %6d\r\n", index, available, description, shop.BuyPrice(obj.GetCost(), s.player.GetCha()))
}

// cmdList lists items for sale at the shop.
// Usage: list [keyword]
func cmdList(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		// No shop special-proc → C routes buy/sell/list to do_not_here
		// (act.other.c:208): "Sorry, but you cannot do that here!" — the
		// command is registered under do_not_here and only a shop spec_proc
		// intercepts it; with no shop the fallback fires regardless of args.
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}

	keyword := ""
	if len(args) > 0 {
		// shopping_list calls one_argument(), so only the first word is
		// consumed by C's isname() matcher.
		keyword = strings.ToLower(args[0])
	}

	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}
	inventory, err := s.manager.world.ShopInventory(shop.VNum, keeper)
	if err != nil {
		return err
	}

	var last *game.ObjectInstance
	count, index := 0, 0
	found := false
	var output strings.Builder
	output.WriteString(" ##   Available   Item                                               Cost\r\n")
	output.WriteString("-------------------------------------------------------------------------\r\n")

	flush := func() {
		if last == nil {
			return
		}
		index++
		if keyword == "" || shopListKeywordMatches(keyword, last.GetKeywords()) {
			output.WriteString(shopListLine(s, shop, last, count, index))
			found = true
		}
	}
	for _, obj := range inventory {
		if obj == nil || obj.GetCost() <= 0 || !game.CanSeeObject(s.player, obj) {
			continue
		}
		if last == nil {
			last, count = obj, 1
		} else if shopObjectsSame(last, obj) {
			count++
		} else {
			flush()
			last, count = obj, 1
		}
	}
	// Preserve src/shop.c:910-925 exactly: the final group increments the
	// ordinal before C checks found, and is only appended when the command has
	// already found an earlier matching group (or has no keyword). Therefore a
	// keyword that matches only the final group still receives the fixed
	// "none are for sale" response.
	index++

	if last == nil {
		s.Send("Currently, there is nothing for sale.\r\n")
	} else if keyword != "" && !found {
		// src/shop.c:920 — this is a fixed C string, not a keeper-name
		// interpolation.
		s.Send("Presently, none of those are for sale.\r\n")
	} else {
		if keyword == "" || shopListKeywordMatches(keyword, last.GetKeywords()) {
			output.WriteString(shopListLine(s, shop, last, count, index))
		}
		s.Send(output.String())
	}
	return nil
}

// cmdBuy buys an item from the shop.
// Usage: buy <item> [count]
func cmdBuy(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		// No shop spec-proc → do_not_here fires first (act.other.c:208),
		// before any argument parsing; see cmdList.
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}
	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}

	// shopping_buy sorts keeper stock immediately after is_ok, even when the
	// command has no item argument (src/shop.c:482-486).
	inventory, err := s.manager.world.ShopInventory(shop.VNum, keeper)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		s.Send("Buy what?")
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

	stock, err := shopPurchaseObject(s.manager.world, shop.VNum, keeper, itemName, inventory, s.player)
	if err != nil {
		return err
	}
	if stock == nil {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageNoSuchItem1], s.player.GetName())
		return nil
	}
	itemDescription := stock.GetShortDesc()

	// Calculate price per item
	pricePerItem := shop.BuyPrice(stock.GetCost(), s.player.GetCha())
	totalPrice := pricePerItem * count
	pays := s.player.GetLevel() < game.LVL_GOD

	// Check if player can afford
	if pays && s.player.GetGold() < totalPrice {
		s.Send("You can't afford it!")
		return nil
	}

	// src/shop.c:520-532 checks count before weight and emits fname(obj->name)
	// followed by the exact LFCR-terminated refusal. Keep this ahead of object
	// creation so the rejected purchase has no registry or inventory effects.
	if s.player.Inventory.GetItemCount()+1 > s.player.MaxCarryItems() {
		s.Send(fmt.Sprintf("%s: You can't carry any more items.\n\r", shopObjectFirstKeyword(stock.GetKeywords())))
		return nil
	}
	if s.player.CarriedWeight()+stock.GetWeight() > s.player.MaxCarryWeight() {
		s.Send(fmt.Sprintf("%s: You can't carry that much weight.\n\r", shopObjectFirstKeyword(stock.GetKeywords())))
		return nil
	}

	// Buy up to count items (stop if inventory is full or gold runs out)
	bought := 0
	var lastBought *game.ObjectInstance
	for i := 0; i < count; i++ {
		if bought > 0 && !shopObjectsSame(lastBought, stock) {
			break
		}
		if s.player.Inventory.IsFull() {
			if bought == 0 {
				s.Send("You can't carry any more items.")
				return nil
			}
			break
		}

		// Check gold before creating item (atomically: concurrent drains were
		// a double-spend, VULN-053)
		if pays && !s.player.SpendGold(pricePerItem) {
			if bought == 0 {
				s.Send("You can't afford it!")
				return nil
			}
			break
		}

		// BuyShopObject clones produced stock and transfers finite stock from
		// the keeper. The atomic spend precedes the hand-over; refund if the
		// object cannot be moved.
		item, err := s.manager.world.BuyShopObject(shop.VNum, keeper, stock, s.player)
		if err != nil {
			if pays {
				s.player.AddGold(pricePerItem)
			}
			break
		}
		lastBought = item
		bought++
		// C calls get_purchase_obj after every hand-over, including the last
		// requested item; this can extract a following zero-cost stock object.
		inventory, err = s.manager.world.ShopInventory(shop.VNum, keeper)
		if err != nil {
			return err
		}
		stock, err = shopPurchaseObject(s.manager.world, shop.VNum, keeper, itemName, inventory, s.player)
		if err != nil {
			return err
		}
		if !shopObjectsSame(lastBought, stock) {
			break
		}
	}

	if bought > 0 {
		totalSpent := pricePerItem * bought
		// C credits the keeper only when the buyer pays; bank-enabled shops
		// sweep coins above 15,000 into their runtime account.
		if pays {
			s.manager.world.CreditShopPurchase(shop.VNum, keeper, totalSpent)
		}
		times := shopTimesMessage(itemDescription, bought)
		game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n buys "+times+".", "", game.ToRoom)
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageBuy], s.player.GetName(), totalSpent)
		s.Send("You now have " + times + ".\n\r")
		s.markDirty(VarInventory)
	}

	return nil
}

// cmdSell sells an item to the shop.
// Usage: sell <item>  or  sell all
func cmdSell(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		// No shop spec-proc → do_not_here fires first (act.other.c:208),
		// before any argument parsing; see cmdList.
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}
	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}

	if len(args) == 0 {
		s.Send("Sell what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Handle "sell all"
	if strings.EqualFold(itemName, "all") {
		return cmdSellAll(s, shop, keeperName, keeper)
	}

	// Find item in player's inventory by keyword
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageNoSuchItem2], s.player.GetName())
		return nil
	}

	// Check if shop buys this item type
	if !shop.WillBuyType(item.GetTypeFlag()) || item.GetCost() < 1 || item.HasExtraFlag(0, 16) { // ITEM_NOSELL, src/structs.h:484.
		s.Send("The shopkeeper doesn't want that.")
		return nil
	}

	// Calculate sell price
	price := shop.SellPrice(item.GetCost(), s.player.GetCha())
	if !s.manager.world.TryPayShopSale(shop.VNum, keeper, price) {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageMissingCash1], s.player.GetName())
		return nil
	}

	// Remove item from player, add gold
	if err := s.manager.world.SellShopObject(shop.VNum, keeper, item, s.player); err == nil {
		s.player.AddGold(price)
		s.manager.world.RebalanceShopBank(shop.VNum, keeper)
		times := shopTimesMessage(item.GetShortDesc(), 1)
		game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n sells "+times+".", "", game.ToRoom)
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageSell], s.player.GetName(), price)
		s.Send("The shopkeeper now has " + times + ".\n\r")
		s.markDirty(VarInventory)
	} else {
		s.manager.world.RefundShopSale(shop.VNum, keeper, price)
		s.Send("You can't sell that.")
	}

	return nil
}

// cmdSellAll sells all items the shop will buy from the player's inventory.
func cmdSellAll(s *Session, shop *game.Shop, keeperName string, keeper *game.MobInstance) error {
	items := s.player.Inventory.FindItems("")
	if len(items) == 0 {
		s.Send("You have nothing to sell.")
		return nil
	}

	sold := 0
	totalGold := 0
	var soldNames []string
	unableToPay := false

	for _, item := range items {
		if !shop.WillBuyType(item.GetTypeFlag()) || item.GetCost() < 1 || item.HasExtraFlag(0, 16) { // ITEM_NOSELL, src/structs.h:484.
			continue
		}

		price := shop.SellPrice(item.GetCost(), s.player.GetCha())
		if !s.manager.world.TryPayShopSale(shop.VNum, keeper, price) {
			unableToPay = true
			break
		}
		if err := s.manager.world.SellShopObject(shop.VNum, keeper, item, s.player); err == nil {
			s.player.AddGold(price)
			totalGold += price
			sold++
			soldNames = append(soldNames, item.GetShortDesc())
		} else {
			s.manager.world.RefundShopSale(shop.VNum, keeper, price)
		}
	}

	if sold == 0 {
		if unableToPay {
			sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageMissingCash1], s.player.GetName())
			return nil
		}
		s.Send("The shopkeeper doesn't want anything you have.")
		return nil
	}
	if unableToPay {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageMissingCash2], s.player.GetName())
	}
	s.manager.world.RebalanceShopBank(shop.VNum, keeper)

	if sold == 1 {
		s.Send(fmt.Sprintf("You sell %s for %d gold pieces.", soldNames[0], totalGold))
	} else {
		s.Send(fmt.Sprintf("You sell %d items for %d gold pieces.", sold, totalGold))
	}
	s.markDirty(VarInventory)
	return nil
}
