package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C's same_obj compares GET_OBJ_RNUM, GET_OBJ_COST, all four extra-flag words,
// and every affected[] location/modifier (src/shop.c:293-318). The shop
// surface map's three stock-identity rows are exactly those field comparisons:
// the live corpus cannot mutate one field while holding the others, so this
// focused proof pins each comparison. The downstream grouping bytes are owned
// by the shop-listing and shop-trade scenarios.
func TestSameShopObjectComparesCRNUMCostExtraFlagsAndAffects(t *testing.T) {
	proto := &parser.Obj{VNum: 5100, Cost: 10}
	left := NewObjectInstance(proto, -1)
	right := NewObjectInstance(proto, -1)
	if !sameShopObject(left, right) {
		t.Fatal("identical objects must compare equal")
	}

	other := NewObjectInstance(&parser.Obj{VNum: 5101, Cost: 10}, -1)
	if sameShopObject(left, other) {
		t.Fatal("different prototype/vnum must not compare equal")
	}

	cost := 11
	dearer := NewObjectInstance(proto, -1)
	dearer.CostOverride = &cost
	if sameShopObject(left, dearer) {
		t.Fatal("different cost must not compare equal")
	}

	flagged := NewObjectInstance(proto, -1)
	flagged.SetExtraFlags(1)
	if sameShopObject(left, flagged) {
		t.Fatal("different extra flags must not compare equal")
	}

	differentCount := NewObjectInstance(proto, -1)
	differentCount.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 2}})
	if sameShopObject(left, differentCount) {
		t.Fatal("different affect count must not compare equal")
	}

	sameLocation := NewObjectInstance(proto, -1)
	sameLocation.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 2}})
	differentModifier := NewObjectInstance(proto, -1)
	differentModifier.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 3}})
	if sameShopObject(sameLocation, differentModifier) {
		t.Fatal("different affect modifier must not compare equal")
	}

	differentLocation := NewObjectInstance(proto, -1)
	differentLocation.SetAffectsOverride([]parser.ObjAffect{{Location: 2, Modifier: 2}})
	if sameShopObject(sameLocation, differentLocation) {
		t.Fatal("different affect location must not compare equal")
	}

	equal := NewObjectInstance(proto, -1)
	equal.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 2}})
	if !sameShopObject(sameLocation, equal) {
		t.Fatal("equal affects must compare equal")
	}
}

// C's shop_producing returns FALSE for GET_OBJ_RNUM(item) < 0 before comparing
// the shop's product list (src/shop.c:322-332). A prototype-less object is the
// port's representation of that negative rnum.
func TestShopProducesObjectRejectsPrototypeLessObject(t *testing.T) {
	proto := &parser.Obj{VNum: 5100, Cost: 10}
	prototypes := map[int]*parser.Obj{5100: proto}
	shop := &Shop{SellTypes: []int{5100}}

	if !shopProducesObject(prototypes, shop, NewObjectInstance(proto, -1)) {
		t.Fatal("prototyped produced object must be accepted")
	}
	if shopProducesObject(prototypes, shop, &ObjectInstance{VNum: 5100}) {
		t.Fatal("object without a prototype (C rnum < 0) must be rejected")
	}
	notSold := &parser.Obj{VNum: 5101, Cost: 10}
	if shopProducesObject(prototypes, shop, NewObjectInstance(notSold, -1)) {
		t.Fatal("prototype outside the product list must be rejected")
	}
}
