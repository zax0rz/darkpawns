package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func sharpenTestWorld(t *testing.T) (*World, *Player, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Sharpener", 1001)
	obs := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer ch: %v", err)
	}
	if err := w.AddPlayer(obs); err != nil {
		t.Fatalf("AddPlayer obs: %v", err)
	}
	var obsOut strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == obs.Name {
			obsOut.Write(msg)
		}
	}
	return w, ch, obs, &obsOut
}

func sharpenTestObject(vnum int, keywords string, typeFlag int, values [4]int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:       vnum,
		Keywords:   keywords,
		ShortDesc:  "a test object",
		TypeFlag:   typeFlag,
		Values:     values,
		ExtraFlags: [4]int{},
	}, -1)
}

func TestDoSharpenUsesCObjectGatesAndCarriedOnly(t *testing.T) {
	ch := NewPlayer(1, "Sharpener", 1001)
	ch.SetSkill(SkillSharpen, 100)

	if got := DoSharpen(ch, "", nil); got.MessageToCh != "Sharpen what?\r\n" || !got.NoMessageToChTerminator {
		t.Fatalf("empty argument message = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}
	if got := DoSharpen(ch, "missing", nil); got.MessageToCh != "Sharpen what?\r\n" {
		t.Fatalf("missing object message = %q", got.MessageToCh)
	}

	nonWeapon := sharpenTestObject(1, "bread", ITEM_FOOD, [4]int{})
	ch.Inventory.RestoreItem(nonWeapon)
	if got := DoSharpen(ch, "bread", nil); got.MessageToCh != "This weapon can not be sharpened.\n\r" || !got.NoMessageToChTerminator {
		t.Fatalf("non-weapon message = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}

	magic := sharpenTestObject(2, "magic sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	magic.SetExtraFlag(0, itemExtraMagic)
	ch.Inventory.RestoreItem(magic)
	if got := DoSharpen(ch, "magic", nil); got.MessageToCh != "This weapon can not be sharpened any further.\n\r" || !got.NoMessageToChTerminator {
		t.Fatalf("magic weapon message = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}

	equipped := sharpenTestObject(3, "equipped sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	if err := ch.Equipment.SetSlot(SlotWield, equipped); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}
	if got := DoSharpen(ch, "equipped", nil); got.MessageToCh != "Sharpen what?\r\n" {
		t.Fatalf("equipped-only message = %q", got.MessageToCh)
	}
}

func TestDoSharpenRejectsNonSlashAndAlreadyAffected(t *testing.T) {
	ch := NewPlayer(1, "Sharpener", 1001)
	ch.SetSkill(SkillSharpen, 100)

	nonSlash := sharpenTestObject(4, "blunt", ITEM_WEAPON, [4]int{0, 0, 0, 11})
	ch.Inventory.RestoreItem(nonSlash)
	if got := DoSharpen(ch, "blunt", nil); got.MessageToCh != "This weapon can not be sharpened.\n\r" {
		t.Fatalf("non-slash weapon message = %q", got.MessageToCh)
	}

	affected := sharpenTestObject(5, "affected", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	affected.Prototype.Affects = []parser.ObjAffect{{Location: ApplyDamroll, Modifier: 2}}
	ch.Inventory.RestoreItem(affected)
	if got := DoSharpen(ch, "affected", nil); got.MessageToCh != "This weapon can not be sharpened any further.\n\r" {
		t.Fatalf("affected weapon message = %q", got.MessageToCh)
	}
}

func TestDoSharpenMutatesAndScalesSuccessfulWeapon(t *testing.T) {
	seed := uint32(1)
	for ; seed < 1000; seed++ {
		if dprng.New(seed).Number(1, 101) < 100 {
			break
		}
	}
	if seed == 1000 {
		t.Fatal("could not find deterministic sharpen success seed")
	}

	for _, test := range []struct {
		level int
		want  int
	}{
		{level: 1, want: 1},
		{level: 26, want: 2},
		{level: 30, want: 3},
		{level: 31, want: 2},
	} {
		t.Run("level", func(t *testing.T) {
			w, ch, _, obsOut := sharpenTestWorld(t)
			ch.SetLevel(test.level)
			ch.SetSkill(SkillSharpen, 100)
			weapon := sharpenTestObject(10+test.level, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
			ch.Inventory.RestoreItem(weapon)

			dprng.ResetStream(seed)
			result := DoSharpen(ch, "sword", w)
			if !result.Success {
				t.Fatalf("result = %#v", result)
			}
			if result.MessageToCh != "You sharpen it to perfection!\r\n" || !result.NoMessageToChTerminator {
				t.Errorf("actor message = %q (noTerm=%v)", result.MessageToCh, result.NoMessageToChTerminator)
			}
			if got := obsOut.String(); got != "Sharpener sharpens a test object to perfection.\r\n" {
				t.Errorf("room message = %q", got)
			}
			affects := weapon.GetAffects()
			if len(affects) != 1 || affects[0].Location != ApplyDamroll || affects[0].Modifier != test.want {
				t.Fatalf("affects = %#v, want damroll %+d", affects, test.want)
			}
		})
	}
}

func TestDoSharpenFailureMutatesEvenWithZeroSkill(t *testing.T) {
	seed := uint32(1)
	for ; seed < 1000; seed++ {
		if dprng.New(seed).Number(1, 101) >= 1 {
			break
		}
	}
	if seed == 1000 {
		t.Fatal("could not find deterministic sharpen failure seed")
	}

	w, ch, _, obsOut := sharpenTestWorld(t)
	weapon := sharpenTestObject(20, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.RestoreItem(weapon)

	dprng.ResetStream(seed)
	result := DoSharpen(ch, "sword", w)
	if result.Success {
		t.Fatal("zero-skill sharpen unexpectedly succeeded")
	}
	if result.MessageToCh != "You damage it trying to sharpen it!\r\n" || obsOut.String() != "Sharpener damages a test object trying to sharpen it!\r\n" {
		t.Fatalf("failure messages = ch %q room %q", result.MessageToCh, obsOut.String())
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillSharpen || !result.DeferredImproveAfterActor {
		t.Fatalf("failure ordering contract = %#v", result)
	}
	if affects := weapon.GetAffects(); len(affects) != 1 || affects[0].Location != ApplyDamroll || affects[0].Modifier != -1 {
		t.Fatalf("failure affects = %#v, want damroll -1", affects)
	}
}

func TestDoSharpenFightingGateFollowsObjectGates(t *testing.T) {
	ch := NewPlayer(1, "Sharpener", 1001)
	ch.SetPosition(combat.PosFighting)
	ch.SetFightingBody(NewPlayer(99999, "opponent", 1001))
	weapon := sharpenTestObject(30, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.RestoreItem(weapon)

	result := DoSharpen(ch, "sword", nil)
	if result.MessageToCh != "You're too busy to be sharpening anything!\n\r" || !result.NoMessageToChTerminator {
		t.Fatalf("fighting message = %q (noTerm=%v)", result.MessageToCh, result.NoMessageToChTerminator)
	}
	if len(weapon.GetAffects()) != 0 {
		t.Fatalf("fighting gate mutated weapon: %#v", weapon.GetAffects())
	}
}

func TestDoSharpenConsumesOneCPercentDraw(t *testing.T) {
	seed := uint32(17)
	reference := dprng.New(seed)
	reference.Number(1, 101)
	wantNext := reference.Number(0, 999)

	ch := NewPlayer(1, "Sharpener", 1001)
	weapon := sharpenTestObject(40, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.RestoreItem(weapon)

	dprng.ResetStream(seed)
	DoSharpen(ch, "sword", nil)
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("draw after sharpen = %d, want %d after one number(1,101)", got, wantNext)
	}
}

func TestDoSharpenInvisibleUsesActRoomVisibility(t *testing.T) {
	w, ch, _, obsOut := sharpenTestWorld(t)
	ch.AddAffect(engine.NewAffectDirect(0, engine.ApplyNone, 1, 0, engine.AFFInvisible, "test"))
	weapon := sharpenTestObject(41, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.RestoreItem(weapon)

	DoSharpen(ch, "sword", w)
	if got := obsOut.String(); got != "Someone damages a test object trying to sharpen it!\r\n" {
		t.Fatalf("invisible sharpen room act = %q, want Someone...", got)
	}
}

func TestCompareAndPointExactTerminatorsAndOrder(t *testing.T) {
	w, ch, obs, _ := sharpenTestWorld(t)
	var events []string
	w.MessageSink = func(name string, msg []byte) {
		events = append(events, name+":"+string(msg))
	}

	// Pin compare exact \n\r vs \r\n terminators (src/new_cmds.c:1969-1992).
	if got := DoCompare(ch, "missing", "other"); got.MessageToCh != "Looks like you don't have those objects..\n\r" || !got.NoMessageToChTerminator {
		t.Fatalf("compare missing = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}
	sw := sharpenTestObject(50, "sword", ITEM_WEAPON, [4]int{0, 1, 4, 3})
	ch.Inventory.RestoreItem(sw)
	if got := DoCompare(ch, "sword", "sword"); got.MessageToCh != "They're the same thing!\n\r" || !got.NoMessageToChTerminator {
		t.Fatalf("compare same = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}
	ch.SetFightingBody(obs)
	if got := DoCompare(ch, "sword", "sword"); got.MessageToCh != "You're pretty busy right now!\n\r" || !got.NoMessageToChTerminator {
		t.Fatalf("compare fighting = %q (noTerm=%v)", got.MessageToCh, got.NoMessageToChTerminator)
	}
	ch.SetFightingBody(nil)

	// Pin point emit order: TO_NOTVICT/TO_ROOM -> TO_CHAR -> TO_VICT
	// (src/new_cmds.c:2557-2562) and hide_invisible=TRUE on TO_VICT.
	obs2 := NewPlayer(3, "Third", 1001)
	if err := w.AddPlayer(obs2); err != nil {
		t.Fatalf("AddPlayer obs2: %v", err)
	}
	DoPoint(ch, "Watcher", w)
	want := []string{
		"Third:Sharpener points at Watcher.\r\n",
		"Sharpener:You point at Watcher.\r\n",
		"Watcher:Sharpener points at you.\r\n",
	}
	if len(events) != len(want) {
		t.Fatalf("point events = %#v, want %#v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, events[i], want[i])
		}
	}

	// Invisible pointer: $n renders as "Someone" for observers/victim who
	// cannot see the pointer, in C order TO_NOTVICT -> TO_CHAR -> TO_VICT.
	events = nil
	ch.AddAffect(engine.NewAffectDirect(0, engine.ApplyNone, 1, 0, engine.AFFInvisible, "test"))
	DoPoint(ch, "Watcher", w)
	wantInvis := []string{
		"Third:Someone points at Watcher.\r\n",
		"Sharpener:You point at Watcher.\r\n",
		"Watcher:Someone points at you.\r\n",
	}
	if len(events) != len(wantInvis) {
		t.Fatalf("invisible point events = %#v, want %#v", events, wantInvis)
	}
	for i := range wantInvis {
		if events[i] != wantInvis[i] {
			t.Fatalf("invis event[%d] = %q, want %q", i, events[i], wantInvis[i])
		}
	}
}
