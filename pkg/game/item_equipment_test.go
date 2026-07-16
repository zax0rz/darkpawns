package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newWieldableItem(vnum int, shortDesc, keywords string, twoHanded bool) *ObjectInstance {
	obj := NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		WearFlags: [4]int{1<<0 | 1<<13}, // ITEM_WEAR_TAKE | ITEM_WEAR_WIELD
	}, -1)
	if twoHanded {
		obj.SetExtraFlag(0, extraFlagTwoHanded)
	}
	return obj
}

func newHoldableItem(vnum int, shortDesc, keywords string) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: shortDesc,
		Keywords:  keywords,
		WearFlags: [4]int{1<<0 | 1<<14}, // ITEM_WEAR_TAKE | ITEM_WEAR_HOLD
	}, -1)
}

// TestPerformWearTwoHanded verifies the two-handed-weapon checks use
// ITEM_TWO_HANDED (bit 28), not bit 3 (ITEM_NODONATE's position) —
// src/structs.h:468-496.
func TestPerformWearTwoHanded(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	held := newHoldableItem(4100, "a wooden buckler", "buckler")
	twoHanded := newWieldableItem(4102, "a greatsword", "greatsword", true)

	// A two-handed weapon must be rejected while something is held.
	if err := w.EquipItem(ch, held, eqWearHold); err != nil {
		t.Fatalf("EquipItem held: %v", err)
	}
	if err := ch.Inventory.AddItem(twoHanded); err != nil {
		t.Fatalf("AddItem twoHanded: %v", err)
	}
	w.performWear(ch, twoHanded, eqWearWield)
	if w.IsEquipped(ch, eqWearWield) {
		t.Errorf("two-handed weapon should not wield while a hand is occupied")
	}
	if msg := lastMsg(); !strings.Contains(msg, "Both hands must be free") {
		t.Errorf("two-handed-wield-blocked message: got %q", msg)
	}

	// Symmetric case: a two-handed weapon already wielded blocks holding.
	if err := w.UnequipItem(ch, eqWearHold); err != nil {
		t.Fatalf("UnequipItem held: %v", err)
	}
	if err := w.EquipItem(ch, twoHanded, eqWearWield); err != nil {
		t.Fatalf("EquipItem twoHanded: %v", err)
	}
	lastMsg()
	w.performWear(ch, held, eqWearHold)
	if w.IsEquipped(ch, eqWearHold) {
		t.Errorf("holding should be blocked while a two-handed weapon is wielded")
	}
	if msg := lastMsg(); !strings.Contains(msg, "Both your hands are occupied") {
		t.Errorf("two-handed-hold-blocked message: got %q", msg)
	}
}

// TestDoWearAllSkipsUnseenItems verifies the wear-all loop skips items the
// wearer can't see (CAN_SEE_OBJ — act.item.c:1611), not items with some
// unrelated extra-flag bit set.
func TestDoWearAllSkipsUnseenItems(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)

	visible := newWieldableItem(4110, "a steel sword", "sword", false)
	if err := ch.Inventory.AddItem(visible); err != nil {
		t.Fatalf("AddItem visible: %v", err)
	}
	invisible := newWieldableItem(4111, "an invisible dagger", "dagger", false)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if err := ch.Inventory.AddItem(invisible); err != nil {
		t.Fatalf("AddItem invisible: %v", err)
	}

	w.DoWear(ch, "all")

	if _, ok := ch.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Errorf("visible item should have been worn")
	}
	if _, ok := ch.Inventory.FindItem("dagger"); !ok {
		t.Errorf("invisible item the wearer can't see should be skipped by wear-all")
	}
}

// TestPerformRemoveNoDrop verifies performRemove's NODROP check uses
// ITEM_NODROP (bit 7), not bit 0 (ITEM_GLOW).
func TestPerformRemoveNoDrop(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	cursed := newWieldableItem(4120, "a cursed blade", "blade", false)
	cursed.SetExtraFlag(0, extraFlagNoDrop)
	if err := w.EquipItem(ch, cursed, eqWearWield); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}

	w.performRemove(ch, eqWearWield)
	if msg := lastMsg(); !strings.Contains(msg, "You can't remove") || !strings.Contains(msg, "CURSED") {
		t.Errorf("NODROP remove message: got %q", msg)
	}
	if _, ok := ch.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Errorf("NODROP item should remain equipped")
	}
}

func TestCWearSlotEquipAndRemove(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	sword := newWieldableItem(4130, "a short sword", "sword", false)
	if err := ch.Inventory.AddItem(sword); err != nil {
		t.Fatalf("AddItem sword: %v", err)
	}
	if err := w.EquipItem(ch, sword, eqWearWield); err != nil {
		t.Fatalf("EquipItem sword: %v", err)
	}
	if !w.IsEquipped(ch, eqWearWield) {
		t.Fatal("eqWearWield should map to an occupied SlotWield")
	}
	if got, ok := ch.Equipment.GetItemInSlot(SlotWield); !ok || got != sword {
		t.Fatalf("SlotWield = (%v, %v), want sword, true", got, ok)
	}

	w.performRemove(ch, eqWearWield)
	if w.IsEquipped(ch, eqWearWield) {
		t.Fatal("performRemove(eqWearWield) left the item equipped")
	}
	if got, ok := ch.Inventory.FindItem("sword"); !ok || got != sword {
		t.Fatalf("removed sword = (%v, %v), want sword in inventory", got, ok)
	}
}

func TestTakeNameEquipAndUnequipOrdering(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	proto := &parser.Obj{
		VNum:       4135,
		ShortDesc:  "a frayed tunic",
		Keywords:   "tunic",
		WearFlags:  [4]int{1<<0 | 1<<3}, // ITEM_WEAR_TAKE | ITEM_WEAR_BODY
		ExtraFlags: [4]int{1 << extraFlagTakeName},
	}
	tunic := NewObjectInstance(proto, -1)
	if err := ch.Inventory.AddItem(tunic); err != nil {
		t.Fatalf("AddItem tunic: %v", err)
	}

	w.DoWear(ch, "tunic")
	if got, want := lastMsg(), "You wear a frayed tunic on your body.\r\n"; got != want {
		t.Fatalf("wear message = %q, want pre-rename %q", got, want)
	}
	if got, want := tunic.GetShortDesc(), "Tester's tunic"; got != want {
		t.Fatalf("equipped short description = %q, want %q", got, want)
	}
	if got := tunic.GetKeywords(); got != "tunic" {
		t.Fatalf("equip changed targeting keywords to %q", got)
	}
	if got := proto.ShortDesc; got != "a frayed tunic" {
		t.Fatalf("equip mutated prototype short description to %q", got)
	}
	if got := NewObjectInstance(proto, -1).GetShortDesc(); got != "a frayed tunic" {
		t.Fatalf("fresh instance inherited runtime rename %q", got)
	}

	// Targeting still uses the original keyword even though the displayed name
	// changed while equipped.
	w.DoRemove(ch, "tunic")
	if got, want := lastMsg(), "You stop using a tunic.\r\n"; got != want {
		t.Fatalf("remove message = %q, want post-rename %q", got, want)
	}
	if got, want := tunic.GetShortDesc(), "a tunic"; got != want {
		t.Fatalf("unequipped short description = %q, want %q", got, want)
	}
	if got, found := ch.Inventory.FindItem("tunic"); !found || got != tunic {
		t.Fatal("removed take-name item was not targetable in inventory by its original keyword")
	}
}

func TestTakeNameUsesWholeKeywordStringAndArticle(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	amulet := NewObjectInstance(&parser.Obj{
		VNum:       4136,
		ShortDesc:  "a cloudy charm",
		Keywords:   "ivory amulet charm",
		WearFlags:  [4]int{1<<0 | 1<<3},
		ExtraFlags: [4]int{1 << extraFlagTakeName},
	}, -1)

	if err := w.EquipItem(ch, amulet, eqWearBody); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}
	if got, want := amulet.GetShortDesc(), "Tester's ivory amulet charm"; got != want {
		t.Fatalf("equipped multi-keyword description = %q, want %q", got, want)
	}
	if err := w.UnequipItem(ch, eqWearBody); err != nil {
		t.Fatalf("UnequipItem: %v", err)
	}
	if got, want := amulet.GetShortDesc(), "an ivory amulet charm"; got != want {
		t.Fatalf("unequipped multi-keyword description = %q, want %q", got, want)
	}
}

func TestNonTakeNameDescriptionUnchangedByEquipCycle(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	sword := newWieldableItem(4137, "a short sword", "sword", false)

	if err := w.EquipItem(ch, sword, eqWearWield); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}
	if got := sword.GetShortDesc(); got != "a short sword" {
		t.Fatalf("equipped ordinary item description = %q", got)
	}
	if err := w.UnequipItem(ch, eqWearWield); err != nil {
		t.Fatalf("UnequipItem: %v", err)
	}
	if got := sword.GetShortDesc(); got != "a short sword" {
		t.Fatalf("unequipped ordinary item description = %q", got)
	}
}

func TestEquipmentCommandRoomMessages(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	actor := NewPlayer(1, "Eqactor", 1001)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	outputs := map[string]*strings.Builder{
		actor.Name:    {},
		observer.Name: {},
	}
	w.MessageSink = func(name string, msg []byte) { outputs[name].Write(msg) }
	assertMessages := func(wantActor, wantObserver string) {
		t.Helper()
		if got := outputs[actor.Name].String(); got != wantActor {
			t.Errorf("actor message = %q, want %q", got, wantActor)
		}
		if got := outputs[observer.Name].String(); got != wantObserver {
			t.Errorf("observer message = %q, want %q", got, wantObserver)
		}
		outputs[actor.Name].Reset()
		outputs[observer.Name].Reset()
	}

	tunic := NewObjectInstance(&parser.Obj{
		VNum: 4140, ShortDesc: "a frayed tunic", Keywords: "tunic", WearFlags: [4]int{1<<0 | 1<<3},
	}, -1)
	sword := newWieldableItem(4141, "a short sword", "sword", false)
	torch := newHoldableItem(4142, "a brass torch", "torch")
	for _, obj := range []*ObjectInstance{tunic, sword, torch} {
		if err := actor.Inventory.AddItem(obj); err != nil {
			t.Fatalf("AddItem %s: %v", obj.GetKeywords(), err)
		}
	}

	w.DoWear(actor, "tunic")
	assertMessages("You wear a frayed tunic on your body.\r\n", "Eqactor wears a frayed tunic on his body.\r\n")
	w.DoWield(actor, "sword")
	assertMessages("You wield a short sword.\r\n", "Eqactor wields a short sword.\r\n")
	w.DoGrab(actor, "torch")
	assertMessages("You grab a brass torch.\r\n", "Eqactor grabs a brass torch.\r\n")
	w.DoRemove(actor, "sword")
	assertMessages("You stop using a short sword.\r\n", "Eqactor stops using a short sword.\r\n")
}

func TestDoGrabRejectsNonHoldableItem(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	tunic := NewObjectInstance(&parser.Obj{
		VNum: 4150, ShortDesc: "a frayed tunic", Keywords: "tunic", TypeFlag: ITEM_ARMOR,
		WearFlags: [4]int{1<<0 | 1<<3},
	}, -1)
	if err := ch.Inventory.AddItem(tunic); err != nil {
		t.Fatalf("AddItem tunic: %v", err)
	}

	w.DoGrab(ch, "tunic")
	if got := lastMsg(); got != "You can't hold that.\r\n" {
		t.Fatalf("hold rejection = %q, want %q", got, "You can't hold that.\r\n")
	}
	if got, ok := ch.Inventory.FindItem("tunic"); !ok || got != tunic {
		t.Fatal("rejected tunic should remain in inventory")
	}
}
