package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ShopInventory returns the keeper's C-ordered stock after shop.c's lazy
// sort_keeper_objs pass. Listing and buying both run this pass; selling runs
// it from slide_obj after the sold object has left the player's inventory.
func (w *World) ShopInventory(shopVNum int, keeper *MobInstance) ([]*ObjectInstance, error) {
	if keeper == nil {
		return nil, fmt.Errorf("shop keeper is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(shopVNum)
	if !ok {
		return nil, fmt.Errorf("shop %d not found", shopVNum)
	}
	w.sortShopInventoryLocked(shop, keeper)
	return keeper.InventorySnapshot(), nil
}

// BuyShopObject applies shop.c's produced-versus-stocked distinction and
// transfers one selected object to the buyer. Produced stock is cloned;
// finite stock leaves the keeper and decrements SHOP_SORT.
func (w *World) BuyShopObject(shopVNum int, keeper *MobInstance, stock *ObjectInstance, buyer *Player) (*ObjectInstance, error) {
	if keeper == nil || stock == nil || buyer == nil {
		return nil, fmt.Errorf("shop keeper, stock, and buyer are required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(shopVNum)
	if !ok {
		return nil, fmt.Errorf("shop %d not found", shopVNum)
	}
	if stock.Location.Kind != ObjInInventory || stock.Location.OwnerKind != OwnerMob || stock.Location.MobID != keeper.ID {
		return nil, fmt.Errorf("object %d is not carried by shop keeper %d", stock.ID, keeper.ID)
	}

	obj := stock
	if shopProducesObject(w.objs, shop, stock) {
		obj = w.newObjectInstanceLocked(stock.Prototype, -1)
	}
	if err := w.moveObjectLocked(obj, LocInventoryPlayer(buyer.Name)); err != nil {
		if obj != stock {
			w.extractObjectLocked(obj)
		}
		return nil, fmt.Errorf("move purchased shop object to buyer: %w", err)
	}
	if obj == stock {
		shop.LastSort--
	}
	return obj, nil
}

// SellShopObject mirrors shopping_sell's obj_from_char followed by slide_obj.
// C may extract a produced item or randomly discard one in six other items;
// otherwise it joins the keeper's sorted inventory, grouped with identical
// stock for list output.
func (w *World) SellShopObject(shopVNum int, keeper *MobInstance, obj *ObjectInstance, seller *Player) error {
	if keeper == nil || obj == nil || seller == nil {
		return fmt.Errorf("shop keeper, object, and seller are required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	shop, ok := w.shopByVNumLocked(shopVNum)
	if !ok {
		return fmt.Errorf("shop %d not found", shopVNum)
	}
	if obj.Location.Kind != ObjInInventory || obj.Location.OwnerKind != OwnerPlayer || obj.Location.PlayerName != seller.Name {
		return fmt.Errorf("object %d is not carried by seller %s", obj.ID, seller.Name)
	}
	if err := w.moveObjectLocked(obj, LocNowhere()); err != nil {
		return fmt.Errorf("remove sold object from seller: %w", err)
	}
	if shop.LastSort < len(keeper.InventorySnapshot()) {
		w.sortShopInventoryLocked(shop, keeper)
	}
	w.slideShopObjectLocked(shop, keeper, obj)
	return nil
}

// sortShopInventoryLocked ports sort_keeper_objs and slide_obj's grouping
// operation. The caller holds w.mu; each keeper-list mutation holds keeper.mu.
func (w *World) sortShopInventoryLocked(shop *Shop, keeper *MobInstance) {
	keeper.mu.Lock()
	defer keeper.mu.Unlock()

	if shop.LastSort >= len(keeper.Inventory) {
		return
	}
	unsortedCount := len(keeper.Inventory) - shop.LastSort
	unsorted := append([]*ObjectInstance(nil), keeper.Inventory[:unsortedCount]...)
	keeper.Inventory = append([]*ObjectInstance(nil), keeper.Inventory[unsortedCount:]...)
	for _, obj := range unsorted {
		obj.Location = LocNowhere()
	}
	// sort_keeper_objs repeatedly removes the head and prepends to a temporary
	// list, so it processes this unsorted prefix in reverse order.
	for i, j := 0, len(unsorted)-1; i < j; i, j = i+1, j-1 {
		unsorted[i], unsorted[j] = unsorted[j], unsorted[i]
	}

	for _, obj := range unsorted {
		if shopProducesObject(w.objs, shop, obj) && !mobInventoryHasVNum(keeper.Inventory, obj.GetVNum()) {
			prependMobObjectLocked(keeper, obj)
			shop.LastSort++
			continue
		}
		w.slideShopObjectWithKeeperLocked(shop, keeper, obj)
	}
}

func (w *World) slideShopObjectLocked(shop *Shop, keeper *MobInstance, obj *ObjectInstance) {
	keeper.mu.Lock()
	defer keeper.mu.Unlock()
	w.slideShopObjectWithKeeperLocked(shop, keeper, obj)
}

// slideShopObjectWithKeeperLocked follows slide_obj. keeper.mu is held.
func (w *World) slideShopObjectWithKeeperLocked(shop *Shop, keeper *MobInstance, obj *ObjectInstance) {
	if shopProducesObject(w.objs, shop, obj) {
		w.extractObjectLocked(obj)
		return
	}
	// C number(0, 5) == 0 discards this non-produced item.
	if combat.GetRoller().Number(0, 5) == 0 {
		w.extractObjectLocked(obj)
		return
	}

	shop.LastSort++
	obj.Location = LocInventoryMob(keeper.ID)
	for i, existing := range keeper.Inventory {
		if !sameShopObject(obj, existing) {
			continue
		}
		keeper.Inventory = append(keeper.Inventory, nil)
		copy(keeper.Inventory[i+2:], keeper.Inventory[i+1:])
		keeper.Inventory[i+1] = obj
		return
	}
	prependMobObjectLocked(keeper, obj)
}

func prependMobObjectLocked(keeper *MobInstance, obj *ObjectInstance) {
	obj.Location = LocInventoryMob(keeper.ID)
	keeper.Inventory = append(keeper.Inventory, nil)
	copy(keeper.Inventory[1:], keeper.Inventory[:len(keeper.Inventory)-1])
	keeper.Inventory[0] = obj
}

func mobInventoryHasVNum(items []*ObjectInstance, vnum int) bool {
	for _, item := range items {
		if item != nil && item.GetVNum() == vnum {
			return true
		}
	}
	return false
}

func shopProducesObject(prototypes map[int]*parser.Obj, shop *Shop, obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return false
	}
	for _, vnum := range shop.SellTypes {
		proto := prototypes[vnum]
		if proto != nil && sameShopObject(obj, NewObjectInstance(proto, -1)) {
			return true
		}
	}
	return false
}

func sameShopObject(left, right *ObjectInstance) bool {
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
		if leftAffects[i].Location != rightAffects[i].Location || leftAffects[i].Modifier != rightAffects[i].Modifier {
			return false
		}
	}
	return true
}
