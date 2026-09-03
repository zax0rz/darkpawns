package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

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

	if got := DoSharpen(ch, ""); got.MessageToCh != "Sharpen what?" {
		t.Fatalf("empty argument message = %q", got.MessageToCh)
	}
	if got := DoSharpen(ch, "missing"); got.MessageToCh != "Sharpen what?" {
		t.Fatalf("missing object message = %q", got.MessageToCh)
	}

	nonWeapon := sharpenTestObject(1, "bread", ITEM_FOOD, [4]int{})
	ch.Inventory.Items = append(ch.Inventory.Items, nonWeapon)
	if got := DoSharpen(ch, "bread"); got.MessageToCh != "This weapon can not be sharpened." {
		t.Fatalf("non-weapon message = %q", got.MessageToCh)
	}

	magic := sharpenTestObject(2, "magic sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	magic.SetExtraFlag(0, itemExtraMagic)
	ch.Inventory.Items = append(ch.Inventory.Items, magic)
	if got := DoSharpen(ch, "magic"); got.MessageToCh != "This weapon can not be sharpened any further." {
		t.Fatalf("magic weapon message = %q", got.MessageToCh)
	}

	equipped := sharpenTestObject(3, "equipped sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	if err := ch.Equipment.SetSlot(SlotWield, equipped); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}
	if got := DoSharpen(ch, "equipped"); got.MessageToCh != "Sharpen what?" {
		t.Fatalf("equipped-only message = %q", got.MessageToCh)
	}
}

func TestDoSharpenRejectsNonSlashAndAlreadyAffected(t *testing.T) {
	ch := NewPlayer(1, "Sharpener", 1001)
	ch.SetSkill(SkillSharpen, 100)

	nonSlash := sharpenTestObject(4, "blunt", ITEM_WEAPON, [4]int{0, 0, 0, 11})
	ch.Inventory.Items = append(ch.Inventory.Items, nonSlash)
	if got := DoSharpen(ch, "blunt"); got.MessageToCh != "This weapon can not be sharpened." {
		t.Fatalf("non-slash weapon message = %q", got.MessageToCh)
	}

	affected := sharpenTestObject(5, "affected", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	affected.Prototype.Affects = []parser.ObjAffect{{Location: ApplyDamroll, Modifier: 2}}
	ch.Inventory.Items = append(ch.Inventory.Items, affected)
	if got := DoSharpen(ch, "affected"); got.MessageToCh != "This weapon can not be sharpened any further." {
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
			ch := NewPlayer(1, "Sharpener", 1001)
			ch.SetLevel(test.level)
			ch.SetSkill(SkillSharpen, 100)
			weapon := sharpenTestObject(10+test.level, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
			ch.Inventory.Items = append(ch.Inventory.Items, weapon)

			dprng.ResetStream(seed)
			result := DoSharpen(ch, "sword")
			if !result.Success {
				t.Fatalf("result = %#v", result)
			}
			if result.MessageToCh != "You sharpen it to perfection!" {
				t.Errorf("actor message = %q", result.MessageToCh)
			}
			if result.MessageToRoom != "Sharpener sharpens a test object to perfection." {
				t.Errorf("room message = %q", result.MessageToRoom)
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

	ch := NewPlayer(1, "Sharpener", 1001)
	weapon := sharpenTestObject(20, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.Items = append(ch.Inventory.Items, weapon)

	dprng.ResetStream(seed)
	result := DoSharpen(ch, "sword")
	if result.Success {
		t.Fatal("zero-skill sharpen unexpectedly succeeded")
	}
	if result.MessageToCh != "You damage it trying to sharpen it!" || result.MessageToRoom != "Sharpener damages a test object trying to sharpen it!" {
		t.Fatalf("failure messages = ch %q room %q", result.MessageToCh, result.MessageToRoom)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillSharpen || !result.MessageToChAfterRoom || !result.DeferredImproveAfterActor {
		t.Fatalf("failure ordering contract = %#v", result)
	}
	if affects := weapon.GetAffects(); len(affects) != 1 || affects[0].Location != ApplyDamroll || affects[0].Modifier != -1 {
		t.Fatalf("failure affects = %#v, want damroll -1", affects)
	}
}

func TestDoSharpenFightingGateFollowsObjectGates(t *testing.T) {
	ch := NewPlayer(1, "Sharpener", 1001)
	ch.SetPosition(combat.PosFighting)
	ch.SetFighting("opponent")
	weapon := sharpenTestObject(30, "sword", ITEM_WEAPON, [4]int{0, 0, 0, 3})
	ch.Inventory.Items = append(ch.Inventory.Items, weapon)

	result := DoSharpen(ch, "sword")
	if result.MessageToCh != "You're too busy to be sharpening anything!" {
		t.Fatalf("fighting message = %q", result.MessageToCh)
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
	ch.Inventory.Items = append(ch.Inventory.Items, weapon)

	dprng.ResetStream(seed)
	DoSharpen(ch, "sword")
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("draw after sharpen = %d, want %d after one number(1,101)", got, wantNext)
	}
}
