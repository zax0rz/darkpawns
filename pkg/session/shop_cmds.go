package session

import (
	"fmt"
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
		if shop, ok := s.manager.world.GetShopByKeeper(mob.VNum); ok && mob.GetPosition() > game.PosSleeping && s.manager.world.ShopAllowsRoom(shop, roomVNum) {
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
		s.manager.world.NPCSayCommand(keeper, closedMessage)
		return false
	}

	if !game.CanSee(keeper, s.player) {
		s.manager.world.NPCSayCommand(keeper, "I don't trade with someone I can't see!")
		return false
	}

	// C is_ok_char lets gods and NPCs bypass the trade restrictions after the
	// visibility check. The live command has a player actor, but retain the
	// source ordering at this boundary.
	if s.player.GetLevel() >= game.LVL_GOD && !s.player.IsNPC() {
		return true
	}

	trade := shop.WithWho
	if (s.player.IsGood() && trade&shopTradeNoGood != 0) ||
		(s.player.IsEvil() && trade&shopTradeNoEvil != 0) ||
		(s.player.IsNeutral() && trade&shopTradeNoNeutral != 0) {
		sendShopKeeperTell(s, keeper, "%s Get out of here before I call the guards!", s.player.GetName())
		return false
	}

	if s.player.IsNPC() {
		return true
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
		index, valid := game.ShopNumber(strings.TrimPrefix(name, "#"))
		if !valid || index < 1 {
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

	keyword := name
	ordinal := game.GetNumber(&keyword)
	if ordinal <= 0 {
		return nil
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
	if (obj.GetTypeFlag() == game.ITEM_WAND || obj.GetTypeFlag() == game.ITEM_STAFF) && obj.GetValue(2) < obj.GetValue(1) {
		description += " (partially used)"
	}
	description = capitalizeFirstRune(description)
	return fmt.Sprintf(" %2d)  %s%-48s %6d\n\r", index, available, description, shop.BuyPrice(obj.GetCost(), s.player.GetCha()))
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
	output.WriteString(" ##   Available   Item                                               Cost\n\r")
	output.WriteString("-------------------------------------------------------------------------\n\r")

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
		pageLiteralString(s, output.String())
	}
	return nil
}

// shopTransactionAmount mirrors transaction_amt (src/shop.c:337-350).
// C is_number accepts digits only (src/interpreter.c:1175-1182), so a signed
// token remains an item name. Counts precede the name; trailing words do not
// supply a count.
func shopTransactionAmount(argument string) (int, string) {
	return game.ShopTransactionAmount(argument)
}

// cmdBuy follows shopping_buy, src/shop.c:472-590. Every hand-over checks the
// next object's price/count/weight, and an affordable prefix is still bought.
func cmdBuy(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}
	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}
	inventory, err := s.manager.world.ShopInventory(shop.VNum, keeper)
	if err != nil {
		return err
	}
	count, itemName := shopTransactionAmount(strings.Join(args, " "))
	if itemName == "" || count == 0 {
		sendShopKeeperTell(s, keeper, "%s What do you want to buy??", s.player.GetName())
		return nil
	}
	stock, err := shopPurchaseObject(s.manager.world, shop.VNum, keeper, itemName, inventory, s.player)
	if err != nil {
		return err
	}
	if stock == nil {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageNoSuchItem1], s.player.GetName())
		return nil
	}
	price := func(obj *game.ObjectInstance) int { return shop.BuyPrice(obj.GetCost(), s.player.GetCha()) }
	pays := s.player.GetLevel() < game.LVL_GOD
	if pays && s.player.GetGold() < price(stock) {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageMissingCash2], s.player.GetName())
		switch shop.Temper {
		case 0:
			s.manager.world.NPCSocialCommand(keeper, "puke", s.player.GetName())
		case 1:
			s.manager.world.NPCEmoteCommand(keeper, "smokes on his joint.")
		}
		return nil
	}
	if s.player.Inventory.GetItemCount()+1 > s.player.MaxCarryItems() {
		s.Send(fmt.Sprintf("%s: You can't carry any more items.\n\r", shopObjectFirstKeyword(stock.GetKeywords())))
		return nil
	}
	if s.player.CarriedWeight()+stock.GetWeight() > s.player.MaxCarryWeight() {
		s.Send(fmt.Sprintf("%s: You can't carry that much weight.\n\r", shopObjectFirstKeyword(stock.GetKeywords())))
		return nil
	}
	bought, goldamt := 0, 0
	var lastBought *game.ObjectInstance
	for stock != nil && (!pays || s.player.GetGold() >= price(stock)) &&
		s.player.Inventory.GetItemCount() < s.player.MaxCarryItems() && bought < count &&
		s.player.CarriedWeight()+stock.GetWeight() <= s.player.MaxCarryWeight() {
		cost := price(stock)
		if pays && !s.player.SpendGold(cost) {
			break
		}
		item, moveErr := s.manager.world.BuyShopObject(shop.VNum, keeper, stock, s.player)
		if moveErr != nil {
			if pays {
				s.player.AddGold(cost)
			}
			return moveErr
		}
		lastBought = item
		bought++
		goldamt += cost
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
	if bought < count {
		switch {
		case stock == nil || !shopObjectsSame(lastBought, stock):
			sendShopKeeperTell(s, keeper, "%s I only have %d to sell you.", s.player.GetName(), bought)
		case s.player.GetGold() < price(stock):
			sendShopKeeperTell(s, keeper, "%s You can only afford %d.", s.player.GetName(), bought)
		case s.player.Inventory.GetItemCount() >= s.player.MaxCarryItems():
			sendShopKeeperTell(s, keeper, "%s You can only hold %d.", s.player.GetName(), bought)
		case s.player.CarriedWeight()+stock.GetWeight() > s.player.MaxCarryWeight():
			sendShopKeeperTell(s, keeper, "%s You can only carry %d.", s.player.GetName(), bought)
		}
	}
	if pays {
		s.manager.world.CreditShopPurchase(shop.VNum, keeper, goldamt)
	}
	if bought > 0 {
		times := shopTimesMessage(lastBought.GetShortDesc(), bought)
		game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n buys "+times+".", "", game.ToRoom)
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageBuy], s.player.GetName(), goldamt)
		s.Send("You now have " + times + ".\n\r")
		s.markDirty(VarInventory)
	}
	return nil
}

// shopSellingObject follows get_selling_obj (src/shop.c:594-631): resolve
// carried/visible/ordinal names before applying trade_with's refusal gates.
func shopSellingObject(s *Session, shop *game.Shop, keeper *game.MobInstance, name string, message bool) *game.ObjectInstance {
	item := s.manager.world.FindCarriedVis(s.player, name)
	refusal := ""
	if item == nil {
		refusal = shop.Messages[game.ShopMessageNoSuchItem2]
	} else if shop.TradeWith(item) == game.ShopTradeNotOK {
		refusal = shop.Messages[game.ShopMessageDoNotBuy]
	} else if shop.TradeWith(item) == game.ShopTradeDead {
		refusal = "%s I don't buy used up wands or staves!"
	} else {
		return item
	}
	if message {
		sendShopKeeperTell(s, keeper, refusal, s.player.GetName())
	}
	return nil
}

// cmdSell follows shopping_sell (src/shop.c:733-812). C sells a counted
// prefix of matching carried objects; "all" is an ordinary object keyword.
func cmdSell(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}
	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}
	count, argument := shopTransactionAmount(strings.Join(args, " "))
	if count == 0 || argument == "" {
		sendShopKeeperTell(s, keeper, "%s What do you want to sell??", s.player.GetName())
		return nil
	}
	name, _ := game.OneArgument(argument)
	item := shopSellingObject(s, shop, keeper, name, true)
	if item == nil {
		return nil
	}
	description := item.GetShortDesc()
	price := func(obj *game.ObjectInstance) int { return shop.SellPrice(obj.GetCost(), s.player.GetCha()) }
	canPay := func(obj *game.ObjectInstance) bool {
		account, cash, ok := s.manager.world.SnapshotShopAccount(shop.VNum, keeper)
		return ok && cash+account.Bank >= price(obj)
	}
	if !canPay(item) {
		sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageMissingCash1], s.player.GetName())
		return nil
	}
	sold, goldamt := 0, 0
	for item != nil && canPay(item) && sold < count {
		cost := price(item)
		if !s.manager.world.TryPayShopSale(shop.VNum, keeper, cost) {
			break
		}
		if err := s.manager.world.SellShopObject(shop.VNum, keeper, item, s.player); err != nil {
			s.manager.world.RefundShopSale(shop.VNum, keeper, cost)
			return err
		}
		sold++
		goldamt += cost
		item = shopSellingObject(s, shop, keeper, name, false)
	}
	if sold < count {
		if item == nil {
			sendShopKeeperTell(s, keeper, "%s You only have %d of those.", s.player.GetName(), sold)
		} else if !canPay(item) {
			sendShopKeeperTell(s, keeper, "%s I can only afford to buy %d of those.", s.player.GetName(), sold)
		}
	}
	s.player.AddGold(goldamt)
	if dot := strings.IndexByte(description, '.'); dot >= 0 {
		description = description[dot+1:]
	}
	times := shopTimesMessage(description, sold)
	game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n sells "+times+".", "", game.ToRoom)
	sendShopKeeperTell(s, keeper, shop.Messages[game.ShopMessageSell], s.player.GetName(), goldamt)
	s.Send("The shopkeeper now has " + times + ".\n\r")
	s.manager.world.RebalanceShopBank(shop.VNum, keeper)
	s.markDirty(VarInventory)
	return nil
}

// cmdValue follows shopping_value (src/shop.c:816-841). Object specials such
// as identifier still run first; ordinary keepers quote without changing state.
func cmdValue(s *Session, args []string) error {
	shop, keeperName, keeper := findShopKeeperAndMobInRoom(s)
	if shop == nil {
		s.Send("Sorry, but you cannot do that here!\r\n")
		return nil
	}
	if !shopIsOK(s, shop, keeper, keeperName) {
		return nil
	}
	if len(args) == 0 {
		sendShopKeeperTell(s, keeper, "%s What do you want me to valuate??", s.player.GetName())
		return nil
	}
	name, _ := game.OneArgument(strings.Join(args, " "))
	if obj := shopSellingObject(s, shop, keeper, name, true); obj != nil {
		sendShopKeeperTell(s, keeper, "%s I'll give you %d gold coins for that!", s.player.GetName(), shop.SellPrice(obj.GetCost(), s.player.GetCha()))
	}
	return nil
}
