package game

import (
	"fmt"
	"testing"
)

// TestVampireConsumableDepth proves the PLR_VAMPIRE gates, not AFF_VAMPIRE.
// C: src/act.item.c:955-1030,1091-1154; src/constants.c:967-984.
func TestVampireConsumableDepth(t *testing.T) {
	weatherMu.Lock()
	original := weatherInfo.Sunlight
	weatherMu.Unlock()
	originalNumber := consumableNumber
	t.Cleanup(func() {
		weatherMu.Lock()
		weatherInfo.Sunlight = original
		weatherMu.Unlock()
		consumableNumber = originalNumber
	})
	for _, sun := range []int{SunRise, SunLight, SunSet, SunDark} {
		weatherMu.Lock()
		weatherInfo.Sunlight = sun
		weatherMu.Unlock()
		for _, vampire := range []bool{false, true} {
			for _, transformed := range []bool{false, true} {
				for _, liq := range []int{LiqWater, LiqBeer, LiqBlood} {
					for _, sub := range []int{scmdDrink, scmdSip} {
						t.Run(fmt.Sprintf("drink/sun%d/plr%t/aff%t/liq%d/sub%d", sun, vampire, transformed, liq, sub), func(t *testing.T) {
							proto := makeDrinkconProto(9801, 20, 20, liq, 0)
							w, ch, out := newConsumableTestWorld(t, proto)
							ch.SetPlrFlag(PlrVampire, vampire)
							ch.SetAffect(affVampire, transformed)
							ch.SetCondition(CondFull, 10)
							ch.SetCondition(CondThirst, 10)
							ch.SetCondition(CondDrunk, 0)
							ch.TatTimer = 7
							obj := NewObjectInstance(proto, -1)
							registerTransferObject(w, obj)
							if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
								t.Fatal(err)
							}
							draws := 0
							consumableNumber = func(lo, hi int) int {
								draws++
								if lo != 3 || hi != 8 {
									t.Fatalf("draw bounds %d,%d", lo, hi)
								}
								return 4
							}
							w.DoDrink(ch, nil, "drink", "skin", sub)
							amount, wantDraws := 4, 1
							if liq == LiqBeer {
								amount, wantDraws = 5, 0
							}
							if sub == scmdSip {
								amount, wantDraws = 0, 0
							}
							blocked := vampire && liq != LiqBlood && (sun == SunSet || sun == SunDark)
							full, thirst, drunk := 10, 10, 0
							da, fa, ta := GetDrinkAffects(liq)
							drunk += da * amount / 4
							if !blocked {
								full += fa * amount / 4
								thirst += ta * amount / 4
							}
							msg := out()
							want := fmt.Sprintf("You drink the %s.\r\n", DrinkName(liq))
							if sub == scmdSip {
								want = fmt.Sprintf("It tastes like %s.\r\n", DrinkName(liq))
							}
							if blocked {
								want += fmt.Sprintf("The vampirism in your body is not satiated by mere %s...\r\n", DrinkName(liq))
							}
							if msg != want {
								t.Fatalf("bytes = %q, want %q", msg, want)
							}
							if ch.GetCondition(CondFull) != full || ch.GetCondition(CondThirst) != thirst || ch.GetCondition(CondDrunk) != drunk {
								t.Fatalf("conditions = %d/%d/%d want %d/%d/%d", ch.GetCondition(CondFull), ch.GetCondition(CondThirst), ch.GetCondition(CondDrunk), full, thirst, drunk)
							}
							if obj.GetValue(1) != 20-amount || obj.GetWeight() != 20-amount || draws != wantDraws || ch.TatTimer != 7 || obj.GetTimer() != 0 {
								t.Fatalf("consumption: liquid=%d weight=%d draws=%d cooldown=%d", obj.GetValue(1), obj.GetWeight(), draws, ch.TatTimer)
							}
							if proto.Values[1] != 20 || proto.Weight != 20 {
								t.Fatal("prototype mutated")
							}
						})
					}
				}
				for _, sub := range []int{scmdEat, scmdTaste} {
					t.Run(fmt.Sprintf("eat/sun%d/plr%t/aff%t/sub%d", sun, vampire, transformed, sub), func(t *testing.T) {
						proto := makeFoodProto(9802, 4, 0)
						w, ch, out := newConsumableTestWorld(t, proto)
						ch.SetPlrFlag(PlrVampire, vampire)
						ch.SetAffect(affVampire, transformed)
						ch.SetCondition(CondFull, 10)
						ch.TatTimer = 7
						obj := NewObjectInstance(proto, -1)
						registerTransferObject(w, obj)
						if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
							t.Fatal(err)
						}
						w.DoEat(ch, nil, "eat", "bread", sub)
						blocked := vampire && (sun == SunSet || sun == SunDark)
						wantFull := 10
						if sub == scmdEat && !blocked {
							wantFull += 4
						}
						want := "You eat a loaf of bread.\r\n"
						if sub == scmdTaste {
							want = "You nibble a little bit of a loaf of bread.\r\n"
						}
						if blocked {
							want += "The vampirism in your body is not satiated by mere food...\r\n"
						}
						if msg := out(); msg != want {
							t.Fatalf("bytes=%q want %q", msg, want)
						}
						if ch.GetCondition(CondFull) != wantFull {
							t.Fatalf("full=%d want %d", ch.GetCondition(CondFull), wantFull)
						}
						count := 0
						if sub == scmdTaste {
							count = 1
						}
						if ch.Inventory.GetItemCount() != count || (sub == scmdTaste && obj.GetValue(0) != 3) || ch.TatTimer != 7 {
							t.Fatal("food consumption/cooldown")
						}
						if proto.Values[0] != 4 {
							t.Fatal("prototype mutated")
						}
					})
				}
			}
		}
	}
}
