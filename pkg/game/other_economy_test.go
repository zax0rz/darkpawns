package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestDoUseWandConsumesInstanceChargeAndUsesDefaultLevel proves the two
// object-magic mechanics that are otherwise not visible in the scenario's
// player-facing transcript: charges mutate on the object instance, and a
// wand with value(0)==0 uses C's DEFAULT_WAND_LVL (12).
func TestDoUseWandConsumesInstanceChargeAndUsesDefaultLevel(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	caster := NewPlayer(1, "Caster", 1001)
	target := NewPlayer(2, "Target", 1001)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatalf("AddPlayer caster: %v", err)
	}
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}

	wand := NewObjectInstance(&parser.Obj{
		VNum:      8053,
		Keywords:  "wand",
		ShortDesc: "a test wand",
		TypeFlag:  ITEM_WAND,
		Values:    [4]int{0, 2, 1, spells.SpellHaste},
	}, -1)
	if err := caster.Inventory.AddItem(wand); err != nil {
		t.Fatalf("AddItem wand: %v", err)
	}
	if err := w.EquipItem(caster, wand, eqWearHold); err != nil {
		t.Fatalf("EquipItem wand: %v", err)
	}

	if !w.DoUse(caster, "wand Target") {
		t.Fatal("DoUse returned false")
	}
	if got := wand.GetValue(2); got != 0 {
		t.Errorf("wand current charges = %d, want 0 after one use", got)
	}
	if !target.IsAffected(affHaste) {
		t.Fatal("wand should apply haste to the target")
	}

	var duration int
	for _, affect := range target.ActiveAffects {
		if affect != nil && affect.SpellID == spells.SpellHaste {
			duration = affect.Duration
			break
		}
	}
	if duration != 12 { // SpellHaste uses the effective cast level directly.
		t.Errorf("haste duration = %d, want 12 from default wand level 12", duration)
	}
}

func TestDoUseWandPassesManualSpellArgumentToControlWeather(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	caster := NewPlayer(1, "Caster", 1001)
	target := NewPlayer(2, "Worse", 1001)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatalf("AddPlayer caster: %v", err)
	}
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}

	wand := NewObjectInstance(&parser.Obj{
		VNum:      8053,
		Keywords:  "wand",
		ShortDesc: "a weather wand",
		TypeFlag:  ITEM_WAND,
		Values:    [4]int{12, 2, 1, spells.SpellControlWeather},
	}, -1)
	if err := caster.Inventory.AddItem(wand); err != nil {
		t.Fatalf("AddItem wand: %v", err)
	}
	if err := w.EquipItem(caster, wand, eqWearHold); err != nil {
		t.Fatalf("EquipItem wand: %v", err)
	}

	weatherMu.RLock()
	before := weatherInfo.Change
	weatherMu.RUnlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		weatherInfo.Change = before
		weatherMu.Unlock()
	})
	if !w.DoUse(caster, "wand Worse") {
		t.Fatal("DoUse returned false")
	}
	if got := TimeWeatherSnapshot().Weather.Change; got >= before {
		t.Fatalf("weather change = %d, want less than %d after C's manual-spell argument 'worse'", got, before)
	}
}

// TestDoUseEquipdQuirkStaffTap pins mag_objectmagic's equipd scan for a held
// staff (src/spell_parser.c:569-609). do_use's second argument is `buf`; when it
// names the WIELDED weapon (slot 16) equipd == WEAR_HOLD (17) and the staff
// takes the switch ITEM_STAFF tap branch instead of the held-staff "radiates"
// else branch. Naming the staff itself (hold slot 17 -> equipd 18) or giving no
// second argument (equipd 0) takes the else branch.
func TestDoUseEquipdQuirkStaffTap(t *testing.T) {
	newCase := func(t *testing.T) (*World, *Player, map[string]string) {
		w, a, _, out := tattooDepthWorld(t)
		staff := NewObjectInstance(&parser.Obj{
			VNum: 8054, Keywords: "staff", ShortDesc: "a test staff",
			TypeFlag: ITEM_STAFF, Values: [4]int{0, 0, 1, spells.SpellSleep},
		}, -1)
		sword := NewObjectInstance(&parser.Obj{
			VNum: 8037, Keywords: "sword", ShortDesc: "a short sword",
			TypeFlag: ITEM_WEAPON, Values: [4]int{0, 1, 6, 3},
		}, -1)
		for _, obj := range []*ObjectInstance{staff, sword} {
			if err := a.Inventory.AddItem(obj); err != nil {
				t.Fatalf("AddItem: %v", err)
			}
		}
		if err := w.EquipItem(a, staff, eqWearHold); err != nil {
			t.Fatalf("EquipItem staff: %v", err)
		}
		if err := w.EquipItem(a, sword, eqWearWield); err != nil {
			t.Fatalf("EquipItem sword: %v", err)
		}
		return w, a, out
	}

	tapLines := "You tap a test staff three times on the ground."
	elseLine := "Your a test staff radiates an ethereal glow that lights the room."

	t.Run("second arg names wielded weapon taps", func(t *testing.T) {
		w, a, out := newCase(t)
		if !w.DoUse(a, "staff sword") {
			t.Fatal("DoUse returned false")
		}
		if !strings.Contains(out["Actor"], tapLines) {
			t.Fatalf("actor output = %q, want the C tap line", out["Actor"])
		}
		if strings.Contains(out["Actor"], elseLine) {
			t.Fatalf("actor output = %q took the held-staff else branch", out["Actor"])
		}
	})
	t.Run("naming the staff takes the else branch", func(t *testing.T) {
		w, a, out := newCase(t)
		if !w.DoUse(a, "staff staff") {
			t.Fatal("DoUse returned false")
		}
		if !strings.Contains(out["Actor"], elseLine) {
			t.Fatalf("actor output = %q, want the held-staff else branch", out["Actor"])
		}
	})
	t.Run("no second arg takes the else branch", func(t *testing.T) {
		w, a, out := newCase(t)
		if !w.DoUse(a, "staff") {
			t.Fatal("DoUse returned false")
		}
		if !strings.Contains(out["Actor"], elseLine) {
			t.Fatalf("actor output = %q, want the held-staff else branch", out["Actor"])
		}
	})
}

// TestDoUseEquipdQuirkWandPoint pins the matching wand branch: naming the
// wielded weapon makes a held wand take the switch ITEM_WAND point branch, whose
// missing-target arm prints "At what should $p be pointed?" (src/spell_parser.c:641).
func TestDoUseEquipdQuirkWandPoint(t *testing.T) {
	w, a, _, out := tattooDepthWorld(t)
	wand := NewObjectInstance(&parser.Obj{
		VNum: 8053, Keywords: "wand", ShortDesc: "a test wand",
		TypeFlag: ITEM_WAND, Values: [4]int{0, 2, 1, spells.SpellHaste},
	}, -1)
	sword := NewObjectInstance(&parser.Obj{
		VNum: 8037, Keywords: "sword", ShortDesc: "a short sword",
		TypeFlag: ITEM_WEAPON, Values: [4]int{0, 1, 6, 3},
	}, -1)
	for _, obj := range []*ObjectInstance{wand, sword} {
		if err := a.Inventory.AddItem(obj); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
	}
	if err := w.EquipItem(a, wand, eqWearHold); err != nil {
		t.Fatalf("EquipItem wand: %v", err)
	}
	if err := w.EquipItem(a, sword, eqWearWield); err != nil {
		t.Fatalf("EquipItem sword: %v", err)
	}
	if !w.DoUse(a, "wand sword") {
		t.Fatal("DoUse returned false")
	}
	if !strings.Contains(out["Actor"], "At what should a test wand be pointed?") {
		t.Fatalf("actor output = %q, want the C point prompt", out["Actor"])
	}
}

// TestUseWandPointSelfIgnoresActionDescription pins C's self-point room act:
// the tch == ch branch always prints the plain "$n points $p at $mself."
// (src/spell_parser.c:614-616); only the at-someone-else branches substitute
// obj->action_description (src/spell_parser.c:620-624). Reverting the branch to
// use the action description fails this test.
func TestUseWandPointSelfIgnoresActionDescription(t *testing.T) {
	w, a, _, out := tattooDepthWorld(t)
	wand := NewObjectInstance(&parser.Obj{
		VNum: 8053, Keywords: "wand", ShortDesc: "a glowing wand",
		TypeFlag: ITEM_WAND, Values: [4]int{0, 2, 1, spells.SpellHaste},
		ActionDesc: "%s's wand erupts in a custom glow.",
	}, -1)
	if err := a.Inventory.AddItem(wand); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := w.EquipItem(a, wand, eqWearHold); err != nil {
		t.Fatalf("EquipItem wand: %v", err)
	}
	// Target the caster itself.
	w.useWandPoint(a, wand, "Actor", spells.SpellHaste, 12)
	if !strings.Contains(out["Observer"], "points a glowing wand at") {
		t.Fatalf("observer output = %q, want the plain self-point act", out["Observer"])
	}
	if strings.Contains(out["Observer"], "custom glow") {
		t.Fatalf("observer output = %q used the action description for a self point", out["Observer"])
	}
}

// TestDoUseEquipdQuirkWholeRemainder pins that C's equipd scan uses the whole
// half_chop remainder as `buf` with isname (src/act.other.c:901;
// src/spell_parser.c:569-572). isname("long sword", "long sword") is 0
// (src/handler.c:81-105 compares each namelist word against the START of str,
// and str continues after "long"), so a multi-word second argument does NOT set
// equipd and the held-staff else branch runs; the single word "sword" does
// match and taps.
func TestDoUseEquipdQuirkWholeRemainder(t *testing.T) {
	newCase := func(t *testing.T) (*World, *Player, map[string]string) {
		w, a, _, out := tattooDepthWorld(t)
		staff := NewObjectInstance(&parser.Obj{
			VNum: 8054, Keywords: "staff", ShortDesc: "a test staff",
			TypeFlag: ITEM_STAFF, Values: [4]int{0, 0, 1, spells.SpellSleep},
		}, -1)
		sword := NewObjectInstance(&parser.Obj{
			VNum: 8037, Keywords: "long sword", ShortDesc: "a long sword",
			TypeFlag: ITEM_WEAPON, Values: [4]int{0, 1, 6, 3},
		}, -1)
		for _, obj := range []*ObjectInstance{staff, sword} {
			if err := a.Inventory.AddItem(obj); err != nil {
				t.Fatalf("AddItem: %v", err)
			}
		}
		if err := w.EquipItem(a, staff, eqWearHold); err != nil {
			t.Fatalf("EquipItem staff: %v", err)
		}
		if err := w.EquipItem(a, sword, eqWearWield); err != nil {
			t.Fatalf("EquipItem sword: %v", err)
		}
		return w, a, out
	}

	t.Run("multi-word remainder is not a match", func(t *testing.T) {
		w, a, out := newCase(t)
		if !w.DoUse(a, "staff long sword") {
			t.Fatal("DoUse returned false")
		}
		if !strings.Contains(out["Actor"], "radiates an ethereal glow") {
			t.Fatalf("actor output = %q, want the held-staff else branch", out["Actor"])
		}
	})
	t.Run("single sword word matches the wielded weapon", func(t *testing.T) {
		w, a, out := newCase(t)
		if !w.DoUse(a, "staff sword") {
			t.Fatal("DoUse returned false")
		}
		if !strings.Contains(out["Actor"], "You tap a test staff three times on the ground.") {
			t.Fatalf("actor output = %q, want the tap line", out["Actor"])
		}
	})
}
