package game

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// The shopping_* bodies for an NPC actor.
//
// C runs the same four bodies for any body that reaches SPECIAL(shop_keeper)
// (src/shop.c:942-1002): nothing in shopping_buy/sell/value/list requires a
// descriptor. Their direct lines (send_to_char, do_tell) reach a mobile
// nowhere, and shopping_list's only direct output is
// page_string(ch->desc, …) (src/shop.c:924), whose first statement is
// `if (!d) return;` (src/modify.c:430-433) — a silent, safe no-op, not
// undefined behaviour. The room still sees the keeper's say, the broke-temper
// emote and the "$n buys/sells …" acts, and the keeper's stock sort draws
// number(0,5) per unsorted item (src/shop.c:679), so the lane is both visible
// and draw-observable.

// shopNpcSay is C's do_say(keeper, msg, cmd_tell, 0): the room hears the keeper,
// who is excluded from the audience, and the verb follows the final punctuation
// (src/act.comm.c do_say). npcSay is that same body for a mobile.
func (w *World) shopNpcSay(keeper *MobInstance, msg string) {
	w.npcSay(keeper, msg)
}

// shopNpcTell is C's do_tell(keeper, buf) where buf starts with the actor's
// name: the keeper "tells" it, the target lookup fails for a multi-word short
// description, and nothing reaches the room.
func (w *World) shopNpcTell(keeper *MobInstance, msg string) {
	w.NPCTellCommand(keeper, msg)
}

// shopNpcIsOK ports is_ok/is_open/is_ok_char (src/shop.c:74-140) for a mobile
// customer. The keeper's spoken refusals reach the room; the tells reach nobody.
func (w *World) shopNpcIsOK(shop *Shop, keeper, customer *MobInstance) bool {
	hours := TimeSnapshot().Hours
	switch {
	case shop.OpenHour1 > hours:
		w.shopNpcSay(keeper, "Come back later!")
		return false
	case shop.CloseHour1 < hours:
		if shop.OpenHour2 > hours {
			w.shopNpcSay(keeper, "Sorry, we have closed, but come back later.")
			return false
		}
		if shop.CloseHour2 < hours {
			w.shopNpcSay(keeper, "Sorry, come back tomorrow.")
			return false
		}
	}
	if !CanSee(keeper, customer) {
		w.shopNpcSay(keeper, "I don't trade with someone I can't see!")
		return false
	}
	alignment := customer.GetAlignment()
	good, evil := alignment >= 350, alignment <= -350
	trade := shop.WithWho
	if (good && trade&1 != 0) || (evil && trade&2 != 0) || (!good && !evil && trade&4 != 0) {
		w.shopNpcTell(keeper, customer.GetName()+" Get out of here before I call the guards!")
		return false
	}
	// C returns TRUE for an NPC before the class restrictions
	// (src/shop.c:94-95).
	return true
}

// shopNpcTransactionAmt ports transaction_amt (src/shop.c:336-350): a leading
// number is the count and is consumed from the argument, otherwise the count is
// one and the argument is untouched.
func shopNpcTransactionAmt(arg string) (int, string) {
	first, rest := oneArgument(arg)
	if first != "" {
		if n, err := strconv.Atoi(first); err == nil {
			return n, strings.TrimLeft(rest, " ")
		}
	}
	return 1, arg
}

// shopNpcPurchaseObject ports get_purchase_obj (src/shop.c:424-453), including
// its loop that extracts zero-cost stock and retries the same selector.
func (w *World) shopNpcPurchaseObject(shop *Shop, keeper *MobInstance, viewer *MobInstance, arg string, msg bool) *ObjectInstance {
	name := strings.TrimSpace(arg)
	if name == "" {
		return nil
	}
	name = strings.Fields(name)[0]
	for {
		stock, err := w.ShopInventory(shop.VNum, keeper)
		if err != nil {
			slog.Warn("shop stock read failed", "shop", shop.VNum, "error", err)
			return nil
		}
		var found *ObjectInstance
		if strings.HasPrefix(name, "#") {
			found = shopNpcPurchaseByIndex(stock, viewer, name)
		} else {
			found = shopNpcPurchaseByName(stock, viewer, name)
		}
		if found == nil {
			if msg {
				w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageNoSuchItem1], viewer.GetName()))
			}
			return nil
		}
		if found.GetCost() > 0 {
			return found
		}
		if err := w.DiscardShopPurchaseObject(keeper, found); err != nil {
			slog.Warn("shop zero-cost stock removal failed", "obj", found.VNum, "error", err)
			return nil
		}
	}
}

// shopNpcPurchaseByIndex ports get_hash_obj_vis (src/shop.c:402-421).
func shopNpcPurchaseByIndex(stock []*ObjectInstance, viewer *MobInstance, name string) *ObjectInstance {
	index, err := strconv.Atoi(strings.TrimPrefix(name, "#"))
	if err != nil || index < 1 {
		return nil
	}
	var last *ObjectInstance
	for _, obj := range stock {
		if obj == nil || obj.GetCost() <= 0 || !canSeeObject(viewer, obj) {
			continue
		}
		if last != nil && sameShopObject(last, obj) {
			continue
		}
		index--
		if index == 0 {
			return obj
		}
		last = obj
	}
	return nil
}

// shopNpcPurchaseByName ports get_slide_obj_vis (src/shop.c:344-375).
func shopNpcPurchaseByName(stock []*ObjectInstance, viewer *MobInstance, name string) *ObjectInstance {
	ordinal := 1
	keyword := name
	if before, after, found := strings.Cut(name, "."); found {
		if n, err := strconv.Atoi(before); err == nil && n > 0 {
			ordinal, keyword = n, after
		}
	}
	keyword = strings.Fields(keyword)[0]
	var last *ObjectInstance
	for _, obj := range stock {
		if obj == nil || !canSeeObject(viewer, obj) || !isCName(keyword, obj.GetKeywords()) {
			continue
		}
		if last != nil && sameShopObject(last, obj) {
			continue
		}
		ordinal--
		if ordinal == 0 {
			return obj
		}
		last = obj
	}
	return nil
}

// shopNpcSellingObject ports get_selling_obj (src/shop.c:593-631).
func (w *World) shopNpcSellingObject(shop *Shop, keeper, customer *MobInstance, name string, msg bool) *ObjectInstance {
	var found *ObjectInstance
	for _, obj := range customer.Inventory {
		if obj != nil && canSeeObject(customer, obj) && isCName(name, obj.GetKeywords()) {
			found = obj
			break
		}
	}
	if found == nil {
		if msg {
			w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageNoSuchItem2], customer.GetName()))
		}
		return nil
	}
	if result := shop.TradeWith(found); result != ShopTradeOK {
		if msg {
			message := shop.Messages[ShopMessageDoNotBuy]
			if result == ShopTradeDead {
				message = "%s I don't buy used up wands or staves!"
			}
			w.shopNpcTell(keeper, fmt.Sprintf(message, customer.GetName()))
		}
		return nil
	}
	return found
}

// shopNpcTimesMessage ports times_message (src/shop.c:353-373).
func shopNpcTimesMessage(obj *ObjectInstance, name string, num int) string {
	text := name
	if obj != nil {
		text = obj.GetShortDesc()
	} else if idx := strings.Index(name, "."); idx >= 0 {
		text = name[idx+1:]
	}
	if num > 1 {
		return fmt.Sprintf("%s (x %d)", text, num)
	}
	return text
}

// shopNpcMoveToInventory moves a purchased object into the customer's carried
// list (C: obj_to_char).
func (w *World) shopNpcMoveToInventory(customer *MobInstance, obj *ObjectInstance) error {
	return w.MoveObjectToMobInventoryFront(obj, customer)
}

// shopNpcList ports shopping_list (src/shop.c:876-925) for a mobile actor. Its
// stock sort is the only effect a descriptor-less body can produce: the list
// itself ends in page_string(ch->desc, …), and page_string returns on a NULL
// descriptor (src/modify.c:430-433).
func (w *World) shopNpcList(shop *Shop, keeper, customer *MobInstance) {
	if !w.shopNpcIsOK(shop, keeper, customer) {
		return
	}
	if _, err := w.ShopInventory(shop.VNum, keeper); err != nil {
		slog.Warn("shop stock sort failed", "shop", shop.VNum, "error", err)
	}
	// C then builds the list page and hands it to page_string(NULL), which
	// returns at src/modify.c:432. Nothing is written.
}

// shopNpcValue ports shopping_value (src/shop.c:815-841).
func (w *World) shopNpcValue(shop *Shop, keeper, customer *MobInstance, arg string) {
	if !w.shopNpcIsOK(shop, keeper, customer) {
		return
	}
	if arg == "" {
		w.shopNpcTell(keeper, fmt.Sprintf("%s What do you want me to valuate??", customer.GetName()))
		return
	}
	name, _ := oneArgument(arg)
	obj := w.shopNpcSellingObject(shop, keeper, customer, name, true)
	if obj == nil {
		return
	}
	price := shop.SellPrice(obj.GetCost(), customer.GetCha())
	w.shopNpcTell(keeper, fmt.Sprintf("%s I'll give you %d gold coins for that!", customer.GetName(), price))
}

// shopNpcBuyObject performs one hand-over of shopping_buy's loop: produced
// stock is cloned, finite stock leaves the keeper and decrements SHOP_SORT
// (src/shop.c:536-545).
func (w *World) shopNpcBuyObject(shop *Shop, keeper, customer *MobInstance, stock *ObjectInstance) (*ObjectInstance, error) {
	if stock == nil {
		return nil, fmt.Errorf("purchase stock is required")
	}
	obj := stock
	if shopProducesObject(w.objs, shop, stock) {
		w.mu.Lock()
		obj = w.newObjectInstanceLocked(stock.Prototype, -1)
		w.mu.Unlock()
	}
	if err := w.shopNpcMoveToInventory(customer, obj); err != nil {
		if obj != stock {
			w.ExtractObject(obj, customer.GetRoomVNum())
		}
		return nil, fmt.Errorf("move purchased object to customer: %w", err)
	}
	if obj == stock {
		shop.LastSort--
	}
	return obj, nil
}

// shopNpcSellObject performs one hand-over of shopping_sell's loop: the object
// leaves the seller and slides into the keeper's grouped stock
// (src/shop.c:776-777).
func (w *World) shopNpcSellObject(shop *Shop, keeper *MobInstance, obj *ObjectInstance) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.moveObjectLocked(obj, LocNowhere()); err != nil {
		return fmt.Errorf("remove sold object from customer: %w", err)
	}
	w.slideShopObjectLocked(shop, keeper, obj)
	return nil
}

// shopNpcBuy ports shopping_buy (src/shop.c:471-590) for a mobile actor.
func (w *World) shopNpcBuy(shop *Shop, keeper, customer *MobInstance, arg string) {
	if !w.shopNpcIsOK(shop, keeper, customer) {
		return
	}
	if _, err := w.ShopInventory(shop.VNum, keeper); err != nil {
		slog.Warn("shop stock sort failed", "shop", shop.VNum, "error", err)
		return
	}
	buynum, arg := shopNpcTransactionAmt(arg)
	if buynum < 0 {
		w.shopNpcTell(keeper, customer.GetName()+" A negative amount?  Try selling me something.")
		return
	}
	if arg == "" || buynum == 0 {
		w.shopNpcTell(keeper, customer.GetName()+" What do you want to buy??")
		return
	}
	obj := w.shopNpcPurchaseObject(shop, keeper, customer, arg, true)
	if obj == nil {
		return
	}
	cha := customer.GetCha()
	pays := customer.GetLevel() < LVL_GOD
	if pays && shop.BuyPrice(obj.GetCost(), cha) > customer.GetGold() {
		w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageMissingCash2], customer.GetName()))
		switch shop.Temper {
		case 0:
			w.npcSocial(keeper, "puke", customer.GetName())
		case 1:
			w.npcSocial(keeper, "emote", "smokes on his joint.")
		}
		return
	}
	if len(customer.Inventory)+1 > mobMaxCarryCount(customer) {
		// C: send_to_char(fname(obj->name) + ": You can't carry any more
		// items.\n\r", ch) — a descriptor-less body receives nothing.
		return
	}
	if mobCarriedWeight(customer)+obj.GetWeight() > mobMaxCarryWeight(customer) {
		return
	}

	bought, goldamt := 0, 0
	var lastBought *ObjectInstance
	for obj != nil && (customer.GetGold() >= shop.BuyPrice(obj.GetCost(), cha) || !pays) &&
		len(customer.Inventory) < mobMaxCarryCount(customer) && bought < buynum &&
		mobCarriedWeight(customer)+obj.GetWeight() <= mobMaxCarryWeight(customer) {
		bought++
		item, err := w.shopNpcBuyObject(shop, keeper, customer, obj)
		if err != nil {
			slog.Warn("shop purchase hand-over failed", "shop", shop.VNum, "error", err)
			break
		}
		goldamt += shop.BuyPrice(obj.GetCost(), cha)
		if pays {
			customer.updateGold(func(coins int) (int, bool) {
				return coins - shop.BuyPrice(item.GetCost(), cha), true
			})
		}
		lastBought = item
		obj = w.shopNpcPurchaseObject(shop, keeper, customer, arg, false)
		if obj == nil || !sameShopObject(lastBought, obj) {
			break
		}
	}

	if bought < buynum {
		cost := 0
		if obj != nil {
			cost = shop.BuyPrice(obj.GetCost(), cha)
		}
		switch {
		case obj == nil || (lastBought != nil && !sameShopObject(lastBought, obj)):
			w.shopNpcTell(keeper, fmt.Sprintf("%s I only have %d to sell you.", customer.GetName(), bought))
		case customer.GetGold() < cost:
			w.shopNpcTell(keeper, fmt.Sprintf("%s You can only afford %d.", customer.GetName(), bought))
		case len(customer.Inventory) >= mobMaxCarryCount(customer):
			w.shopNpcTell(keeper, fmt.Sprintf("%s You can only hold %d.", customer.GetName(), bought))
		case mobCarriedWeight(customer)+obj.GetWeight() > mobMaxCarryWeight(customer):
			w.shopNpcTell(keeper, fmt.Sprintf("%s You can only carry %d.", customer.GetName(), bought))
		default:
			w.shopNpcTell(keeper, fmt.Sprintf("%s Something screwy only gave you %d.", customer.GetName(), bought))
		}
	}
	if pays {
		w.CreditShopPurchase(shop.VNum, keeper, goldamt)
	}
	// C: sprintf(tempstr, "%s", times_message(ch->carrying, 0, bought)) — the
	// head of the buyer's carried list, which obj_to_char front-inserted. With
	// an empty carried list C would pass a NULL name into strchr() (undefined
	// behaviour, R1a); the port simply omits the act there.
	if carried := customer.InventorySnapshot(); len(carried) > 0 {
		times := shopNpcTimesMessage(carried[0], "", bought)
		Act(w, false, customer, nil, nil, nil, "$n buys "+times+".", "", ToRoom)
	}
	// C then tells the keeper and sends "You now have %s.\n\r" to the buyer;
	// neither reaches a descriptor-less pet.
	w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageBuy], customer.GetName(), goldamt))
}

// shopNpcSell ports shopping_sell (src/shop.c:732-812) for a mobile actor.
func (w *World) shopNpcSell(shop *Shop, keeper, customer *MobInstance, arg string) {
	if !w.shopNpcIsOK(shop, keeper, customer) {
		return
	}
	sellnum, arg := shopNpcTransactionAmt(arg)
	if sellnum < 0 {
		w.shopNpcTell(keeper, customer.GetName()+" A negative amount?  Try buying something.")
		return
	}
	if arg == "" || sellnum == 0 {
		w.shopNpcTell(keeper, customer.GetName()+" What do you want to sell??")
		return
	}
	name, _ := oneArgument(arg)
	obj := w.shopNpcSellingObject(shop, keeper, customer, name, true)
	if obj == nil {
		return
	}
	shortd := obj.GetShortDesc()
	cha := customer.GetCha()

	if !w.shopCanPaySale(shop, keeper, shop.SellPrice(obj.GetCost(), cha)) {
		w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageMissingCash1], customer.GetName()))
		return
	}

	sold, goldamt := 0, 0
	for obj != nil && w.shopCanPaySale(shop, keeper, shop.SellPrice(obj.GetCost(), cha)) && sold < sellnum {
		sold++
		price := shop.SellPrice(obj.GetCost(), cha)
		goldamt += price
		if !w.TryPayShopSale(shop.VNum, keeper, price) {
			break
		}
		if err := w.shopNpcSellObject(shop, keeper, obj); err != nil {
			slog.Warn("shop sale hand-over failed", "shop", shop.VNum, "error", err)
			break
		}
		obj = w.shopNpcSellingObject(shop, keeper, customer, name, false)
	}

	if sold < sellnum {
		switch {
		case obj == nil:
			w.shopNpcTell(keeper, fmt.Sprintf("%s You only have %d of those.", customer.GetName(), sold))
		case !w.shopCanPaySale(shop, keeper, shop.SellPrice(obj.GetCost(), cha)):
			w.shopNpcTell(keeper, fmt.Sprintf("%s I can only afford to buy %d of those.", customer.GetName(), sold))
		default:
			w.shopNpcTell(keeper, fmt.Sprintf("%s Something really screwy made me buy %d.", customer.GetName(), sold))
		}
	}
	customer.updateGold(func(coins int) (int, bool) { return coins + goldamt, true })
	if sold > 0 {
		times := shopNpcTimesMessage(nil, shortd, sold)
		Act(w, false, customer, nil, nil, nil, "$n sells "+times+".", "", ToRoom)
	}
	// C then tells the keeper and sends "The shopkeeper now has %s.\\n\\r" to
	// the seller; neither reaches a descriptor-less pet.
	w.shopNpcTell(keeper, fmt.Sprintf(shop.Messages[ShopMessageSell], customer.GetName(), goldamt))
	w.RebalanceShopBank(shop.VNum, keeper)
}

// shopCanPaySale reports whether the keeper's coins plus bank cover price, the
// gate shopping_sell evaluates before every hand-over (src/shop.c:763, 768).
func (w *World) shopCanPaySale(shop *Shop, keeper *MobInstance, price int) bool {
	return keeper.GetGold()+shop.Bank >= price
}
