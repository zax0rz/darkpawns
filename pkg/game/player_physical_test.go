package game

// Worn APPLY_AGE / APPLY_CHAR_WEIGHT / APPLY_CHAR_HEIGHT proof
// (src/handler.c:160-166). The stored fields keep the base; the getters fold
// every current modifier with C's sbyte modifiers and ubyte wrap
// (src/structs.h:736-739, 885-886). Serialization stores the base exactly as
// char_to_store does after unwinding every modifier (src/db.c:2531-2637).

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func physicalItem(location, modifier int, wearBit int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{VNum: 14425, Keywords: "vest", WearFlags: [4]int{1 | wearBit}, Weight: 1, Affects: []parser.ObjAffect{{Location: location, Modifier: modifier}}}, -1)
}

func physicalPlayer() *Player {
	p := NewPlayer(1, "Physical", 1001)
	p.Weight = 180
	p.Height = 175
	p.Birth = time.Now().Unix() - int64(17*SECS_PER_MUD_YEAR)
	p.CopyBaseAttributes()
	return p
}

func TestPhysicalAffectsEquipRemoveAllLocations(t *testing.T) {
	p := physicalPlayer()
	baseWeight, baseHeight, baseBirth := p.Weight, p.Height, p.Birth
	baseYears := Age(baseBirth).Year

	ageItem := physicalItem(engine.ApplyAge, 10, 1<<3)
	weightItem := physicalItem(engine.ApplyCharWeight, -10, 1<<4)
	heightItem := physicalItem(engine.ApplyCharHeight, 3, 1<<6)
	for _, item := range []*ObjectInstance{ageItem, weightItem, heightItem} {
		if err := p.Equipment.Equip(item, p.Inventory); err != nil {
			t.Fatalf("Equip: %v", err)
		}
	}

	// Repeated rebuilds between reads must not drift the folded values.
	for i := 0; i < 3; i++ {
		p.AffectTotal()
		if got, want := p.GetWeight(), baseWeight-10; got != want {
			t.Fatalf("weight pass %d = %d, want %d", i, got, want)
		}
		if got, want := p.GetHeight(), baseHeight+3; got != want {
			t.Fatalf("height pass %d = %d, want %d", i, got, want)
		}
		if got, want := p.EffectiveBirth(), baseBirth-10*int64(SECS_PER_MUD_YEAR); got != want {
			t.Fatalf("birth pass %d = %d, want %d", i, got, want)
		}
		if years, _, _, _ := p.GetAgeParts(); years != baseYears+10 {
			t.Fatalf("age years pass %d = %d, want %d", i, years, baseYears+10)
		}
	}

	for _, item := range []*ObjectInstance{ageItem, weightItem, heightItem} {
		if !p.Equipment.UnequipItem(item, p.Inventory) {
			t.Fatalf("UnequipItem failed for %v", item)
		}
	}
	p.AffectTotal()
	if got := p.GetWeight(); got != baseWeight {
		t.Fatalf("weight after remove = %d, want base %d", got, baseWeight)
	}
	if got := p.GetHeight(); got != baseHeight {
		t.Fatalf("height after remove = %d, want base %d", got, baseHeight)
	}
	if got := p.EffectiveBirth(); got != baseBirth {
		t.Fatalf("birth after remove = %d, want base %d", got, baseBirth)
	}
}

func TestPhysicalAffectsWrap(t *testing.T) {
	p := physicalPlayer()
	p.Weight = 250
	p.Height = 10

	// C adds each modifier onto the ubyte field: 250+30 wraps to 24, and
	// 10-20 wraps to 246 (src/structs.h:885-886).
	if err := p.Equipment.Equip(attrItem(engine.ApplyCharWeight, 30), p.Inventory); err != nil {
		t.Fatal(err)
	}
	if got, want := p.GetWeight(), 24; got != want {
		t.Fatalf("wrapped weight = %d, want %d", got, want)
	}
	if !p.Equipment.UnequipItem(attrItem(engine.ApplyCharWeight, 30), p.Inventory) {
		// UnequipItem matches by pointer; re-find via the worn set instead.
		for _, item := range COrderedWorn(p) {
			if item != nil {
				p.Equipment.UnequipItem(item, p.Inventory)
			}
		}
	}
	if got := p.GetWeight(); got != 250 {
		t.Fatalf("weight after unwrap = %d, want 250", got)
	}

	if err := p.Equipment.Equip(attrItem(engine.ApplyCharHeight, -20), p.Inventory); err != nil {
		t.Fatal(err)
	}
	if got, want := p.GetHeight(), 246; got != want {
		t.Fatalf("wrapped height = %d, want %d", got, want)
	}
}

func TestPhysicalAffectsMultipleItemsAndRemovalOrder(t *testing.T) {
	p := physicalPlayer()
	first := physicalItem(engine.ApplyCharWeight, 30, 1<<3)
	second := physicalItem(engine.ApplyCharWeight, 50, 1<<4)
	negative := physicalItem(engine.ApplyCharWeight, -5, 1<<6)
	for _, item := range []*ObjectInstance{first, second, negative} {
		if err := p.Equipment.Equip(item, p.Inventory); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := p.GetWeight(), p.Weight+75; got != want {
		t.Fatalf("stacked weight = %d, want %d", got, want)
	}
	// Removal order must not matter: each item removes exactly its own
	// modifier (C unequip_char negates that item's affects).
	if !p.Equipment.UnequipItem(second, p.Inventory) {
		t.Fatal("remove second failed")
	}
	if got, want := p.GetWeight(), p.Weight+25; got != want {
		t.Fatalf("after removing second = %d, want %d", got, want)
	}
	if !p.Equipment.UnequipItem(first, p.Inventory) {
		t.Fatal("remove first failed")
	}
	if got, want := p.GetWeight(), p.Weight-5; got != want {
		t.Fatalf("after removing first = %d, want %d", got, want)
	}
	if !p.Equipment.UnequipItem(negative, p.Inventory) {
		t.Fatal("remove negative failed")
	}
	if got := p.GetWeight(); got != p.Weight {
		t.Fatalf("after removing all = %d, want base %d", got, p.Weight)
	}
}

func TestPhysicalAffectsTimedAffectFolds(t *testing.T) {
	p := physicalPlayer()
	base := p.Weight
	// C affect_modify reaches the same switch for timed affects
	// (src/handler.c:280-292), so a timed APPLY_CHAR_WEIGHT folds too.
	p.AddAffect(engine.NewAffectDirect(1, engine.ApplyCharWeight, 1, 12, 0, "test"))
	if got, want := p.GetWeight(), base+12; got != want {
		t.Fatalf("timed weight = %d, want %d", got, want)
	}
	p.RemoveAffectBySpell(1)
	if got := p.GetWeight(); got != base {
		t.Fatalf("after affect expiry = %d, want base %d", got, base)
	}
}

// char_to_store unwinds every modifier before writing, so the persisted
// weight, height, and birth are the base values, and a load + re-equip
// rebuilds the effective values without double-counting
// (src/db.c:2531-2637; src/objsave.c:344-443).
func TestPhysicalAffectsSerializationBoundary(t *testing.T) {
	p := physicalPlayer()
	baseWeight, baseHeight, baseBirth := p.Weight, p.Height, p.Birth
	if err := p.Equipment.Equip(physicalItem(engine.ApplyCharWeight, 20, 1<<3), p.Inventory); err != nil {
		t.Fatal(err)
	}
	if err := p.Equipment.Equip(physicalItem(engine.ApplyCharHeight, -15, 1<<4), p.Inventory); err != nil {
		t.Fatal(err)
	}
	if err := p.Equipment.Equip(physicalItem(engine.ApplyAge, -2, 1<<6), p.Inventory); err != nil {
		t.Fatal(err)
	}
	if got, want := p.GetWeight(), baseWeight+20; got != want {
		t.Fatalf("equipped weight = %d, want %d", got, want)
	}

	data, err := SerializePlayer(p)
	if err != nil {
		t.Fatalf("SerializePlayer: %v", err)
	}
	loaded, err := DeserializePlayer(data)
	if err != nil {
		t.Fatalf("DeserializePlayer: %v", err)
	}
	if loaded.Weight != baseWeight || loaded.Height != baseHeight || loaded.Birth != baseBirth {
		t.Fatalf("serialized base = %d/%d/%d, want base %d/%d/%d (char_to_store unwinds modifiers)",
			loaded.Weight, loaded.Height, loaded.Birth, baseWeight, baseHeight, baseBirth)
	}

	// Re-equipping after load must reproduce the effective values exactly
	// once: the loaded base plus the current modifiers.
	loaded.CopyBaseAttributes()
	if err := loaded.Equipment.Equip(physicalItem(engine.ApplyCharWeight, 20, 1<<3), loaded.Inventory); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Equipment.Equip(physicalItem(engine.ApplyCharHeight, -15, 1<<4), loaded.Inventory); err != nil {
		t.Fatal(err)
	}
	if got, want := loaded.GetWeight(), baseWeight+20; got != want {
		t.Fatalf("post-load weight = %d, want %d", got, want)
	}
	if got, want := loaded.GetHeight(), baseHeight-15; got != want {
		t.Fatalf("post-load height = %d, want %d", got, want)
	}
	_ = baseBirth
}
