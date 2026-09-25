package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// GET_OBJ_EXTRA is per object (read_object copies the prototype's): a change
// keeps the prototype's other flags, a prototype flag can be removed, and an
// object can end up with no flags at all.
func TestObjectExtraFlagsStartFromPrototype(t *testing.T) {
	const glow, hum, magic = 0, 1, 6
	newObj := func() *ObjectInstance {
		return NewObjectInstance(&parser.Obj{VNum: 1, ExtraFlags: [4]int{1 << glow, 0, 0, 0}}, 0)
	}

	obj := newObj()
	obj.SetExtraFlag(0, hum)
	if !obj.HasExtraFlag(0, glow) || !obj.HasExtraFlag(0, hum) {
		t.Fatalf("setting HUM on a glowing object: flags %v, want GLOW|HUM", obj.GetExtraFlags())
	}

	obj = newObj()
	obj.RemoveExtraFlag(0, glow)
	if obj.HasExtraFlag(0, glow) {
		t.Fatalf("removing the prototype's GLOW left it set: %v", obj.GetExtraFlags())
	}
	if obj.GetExtraFlags() != ([4]int{}) {
		t.Fatalf("an object whose only flag was removed should have none: %v", obj.GetExtraFlags())
	}

	// spell_enchant_weapon's SET_BIT(GET_OBJ_EXTRA(obj), ITEM_MAGIC) path.
	obj = newObj()
	obj.SetExtraFlags(obj.ExtraFlagWord(0) | 1<<magic)
	if !obj.HasExtraFlag(0, glow) || !obj.HasExtraFlag(0, magic) {
		t.Fatalf("enchanting a glowing object: flags %v, want GLOW|MAGIC", obj.GetExtraFlags())
	}

	obj.ResetExtraFlags()
	if obj.GetExtraFlags() != ([4]int{1 << glow, 0, 0, 0}) {
		t.Fatalf("after ResetExtraFlags the object should read its prototype's flags: %v", obj.GetExtraFlags())
	}
}

// MOB_FLAGGED reads the mobile's own act bits (read_mobile copies the
// prototype's), so a bit set on one mobile at runtime is seen by the name
// lookups and is not seen on another copy.
func TestMobFlagsArePerInstance(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	first := spawnTargetMob(t, w)
	second := spawnTargetMob(t, w)
	first.SetMobFlag(MobFlagHunter)
	if !first.HasFlag("hunter") || !hasMobFlag(first, "HUNTER") {
		t.Fatal("a HUNTER bit set on the instance is not seen by HasFlag/hasMobFlag")
	}
	if second.HasFlag("hunter") {
		t.Fatal("setting HUNTER on one mobile changed another copy")
	}
	first.ClearMobFlag(MobFlagHunter)
	if first.HasFlag("hunter") {
		t.Fatal("clearing HUNTER on the instance left HasFlag true")
	}
}
