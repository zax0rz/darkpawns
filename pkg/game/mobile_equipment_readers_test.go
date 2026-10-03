package game

import (
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func zoneArmedMob(t *testing.T) (*World, *MobInstance, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 100}}, Mobs: []parser.Mob{{VNum: 300, Keywords: "guard", ShortDesc: "a guard", Level: 30, Position: 8, DefaultPos: 8, Str: 11, Dex: 11, Int: 11, Wis: 11, Con: 11, Cha: 11, AC: 100}}, Objs: []parser.Obj{
		{VNum: 200, Keywords: "sword", ShortDesc: "a sword", TypeFlag: ITEM_WEAPON, Values: [4]int{0, 2, 4, 3}, ExtraFlags: [4]int{1 << itemExtraBless}, WearFlags: [4]int{1 << 13}, LoadPercent: 100},
		{VNum: 201, Keywords: "armor", TypeFlag: ITEM_ARMOR, Values: [4]int{10}, LoadPercent: 100},
		{VNum: 202, Keywords: "lamp", TypeFlag: ITEM_LIGHT, Values: [4]int{0, 0, -1}, LoadPercent: 100},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := NewSpawner(w)
	if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{{Command: "M", Arg1: 300, Arg2: 1, Arg3: 100}, {Command: "E", Arg1: 200, Arg2: 1, Arg3: 16}}}); err != nil {
		t.Fatal(err)
	}
	p := NewPlayer(1, "Tester", 100)
	p.SetHP(1000)
	p.SetMaxHP(1000)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	return w, w.GetMobsInRoom(100)[0], p
}

func TestMobileEquipmentReaderCombatWeapon(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	typ, _, _, bless := w.WireCombatCallbacks().GetWeaponInfo(m.GetName())
	if typ != 3 || !bless {
		t.Fatalf("zone weapon combat info=%d/%v, want slash/blessed", typ, bless)
	}
}

func TestMobileEquipmentReaderHitModifiers(t *testing.T) {
	_, m, _ := zoneArmedMob(t)
	if !m.HitModifiers().WeaponBlessed {
		t.Fatal("HitModifiers missed C WEAR_WIELD weapon")
	}
}

func TestMobileEquipmentReaderFighterParry(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	p.SetFighting(m.GetName())
	var out strings.Builder
	w.MessageSink = func(_ string, b []byte) { out.Write(b) }
	for seed := uint32(1); seed < 100; seed++ {
		rng := dprng.New(seed)
		if rng.Number(1, 101) <= rng.Number(50, 100) {
			dprng.ResetStream(seed)
			mobParry(w, m, p)
			break
		}
	}
	if !strings.Contains(out.String(), "dazzling show of swordplay") {
		t.Fatal("fighter parry missed C WEAR_WIELD weapon")
	}
}

func TestMobileEquipmentReaderPaladinCharge(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	dprng.ResetStream(17)
	hp := p.GetHP()
	mobCharge(w, m, p)
	if p.GetHP() >= hp {
		t.Fatal("paladin charge missed C WEAR_WIELD weapon")
	}
}

func TestMobileEquipmentReaderDisarm(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	p.SetSkill(SkillDisarm, 200)
	p.SetFighting(m.GetName())
	m.SetFighting(p.GetName())
	dprng.ResetStream(1)
	if !DoDisarm(p, m, w).Success {
		t.Fatal("disarm missed C WEAR_WIELD weapon")
	}
	if len(m.Inventory) != 1 || m.Inventory[0].GetVNum() != 200 {
		t.Fatal("disarm did not hand weapon to victim inventory")
	}
}

func TestMobileEquipmentReaderArmor(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	s := NewSpawner(w)
	if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{{Command: "M", Arg1: 300, Arg2: 2, Arg3: 100}, {Command: "E", Arg1: 201, Arg2: 1, Arg3: 5}}}); err != nil {
		t.Fatal(err)
	}
	for _, mob := range w.GetMobsInRoom(100) {
		if mob != m && mob.GetAC() != 70 {
			t.Fatalf("mobile body armor AC=%d want 70", mob.GetAC())
		}
	}
}

func TestMobileEquipmentReaderLight(t *testing.T) {
	w, _, _ := zoneArmedMob(t)
	s := NewSpawner(w)
	if err := s.ExecuteZoneReset(&parser.Zone{Commands: []parser.ZoneCommand{{Command: "M", Arg1: 300, Arg2: 2, Arg3: 100}, {Command: "E", Arg1: 202, Arg2: 1, Arg3: 17}}}); err != nil {
		t.Fatal(err)
	}
	r, _ := w.GetRoom(100)
	if r.Light != 1 {
		t.Fatalf("C lit held lamp room light=%d want 1", r.Light)
	}
}

var _ combat.Combatant = (*MobInstance)(nil)

func TestMobileEquipmentRemovalBeforeOwnerRetires(t *testing.T) {
	for _, removal := range []string{"death", "dust", "extraction", "pending"} {
		t.Run(removal, func(t *testing.T) {
			w, m, p := zoneArmedMob(t)
			armor, err := w.SpawnObject(201, -1)
			if err != nil {
				t.Fatal(err)
			}
			lamp, err := w.SpawnObject(202, -1)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.EquipMobileObject(m, armor, mobWearBody); err != nil {
				t.Fatal(err)
			}
			if err := w.EquipMobileObject(m, lamp, 17); err != nil {
				t.Fatal(err)
			}
			switch removal {
			case "death":
				w.Instakill(m, p, combat.TYPE_SLASH)
			case "dust":
				w.Instakill(m, p, 93)
			case "extraction":
				w.ExtractMob(m)
				if !w.HasPendingExtractions() || m.GetAC() != 70 {
					t.Fatal("direct extraction did not retain C equipment until the pending pass")
				}
				w.ExtractPendingChars()
			case "pending":
				m.mu.Lock()
				m.Flags |= 1 << uint(MobFlagExtract)
				m.mu.Unlock()
				w.ExtractPendingChars()
			}
			if len(m.EquipmentSnapshot()) != 0 || m.GetAC() != 100 {
				t.Fatalf("retired owner still has equipment effects: slots=%d AC=%d", len(m.EquipmentSnapshot()), m.GetAC())
			}
			room, _ := w.GetRoom(100)
			if room.Light != 0 {
				t.Fatalf("removed equipped light left room light=%d", room.Light)
			}
			if armor.Location.Kind == ObjEquipped || lamp.Location.Kind == ObjEquipped {
				t.Fatal("removed objects still name retired equipment owner")
			}
		})
	}
}

func TestMobileEquipmentReaderPointsAndAttributes(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 3}, {Location: 12, Modifier: 7}, {Location: 13, Modifier: 9}, {Location: 14, Modifier: 11}, {Location: 18, Modifier: 4}, {Location: 19, Modifier: 5}, {Location: 20, Modifier: -2}})
	mana, hp, move, hit, dam := m.GetMaxMana(), m.GetMaxHP(), m.GetMaxMove(), m.GetHitroll(), m.GetDamroll()
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if m.GetStr() != 14 || m.GetMaxMana() != mana+7 || m.GetMaxHP() != hp+9 || m.GetMaxMove() != move+11 || m.GetHitroll() != hit+4 || m.GetDamroll() != dam+5 || m.GetSavingThrow(0) != -2 {
		t.Fatalf("equipment stat consumers missed live modifiers: str=%d mana=%d hp=%d move=%d hit=%d dam=%d save=%d", m.GetStr(), m.GetMaxMana(), m.GetMaxHP(), m.GetMaxMove(), m.GetHitroll(), m.GetDamroll(), m.GetSavingThrow(0))
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if m.GetStr() != 11 || m.GetMaxMana() != mana || m.GetMaxHP() != hp || m.GetMaxMove() != move || m.GetHitroll() != hit || m.GetDamroll() != dam || m.GetSavingThrow(0) != 0 {
		t.Fatal("unequip failed to undo stat modifiers")
	}
}

func TestMobileEquipmentReaderLightMoves(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	lamp, err := w.SpawnObject(202, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(m, lamp, 17); err != nil {
		t.Fatal(err)
	}
	m.SetRoom(-1)
	room, _ := w.GetRoom(100)
	if room.Light != 0 {
		t.Fatalf("departed mobile left light=%d", room.Light)
	}
	m.SetRoom(100)
	room, _ = w.GetRoom(100)
	if room.Light != 1 || !mobHasLight(m) {
		t.Fatalf("arriving mobile light=%d has_light=%v", room.Light, mobHasLight(m))
	}
}

func TestMobileEquipmentBoundaryNameAndFlags(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	proto := *obj.Prototype
	proto.Keywords = "apple armor"
	proto.ExtraFlags[0] |= 1 << extraFlagTakeName
	obj.Prototype = &proto
	obj.AffectFlags[0] = 1 << 7
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if obj.GetShortDesc() != "a guard's apple armor" || !m.HasAffect(7) {
		t.Fatalf("equip omitted name or flags: %q", obj.GetShortDesc())
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if obj.GetShortDesc() != "an apple armor" || m.HasAffect(7) {
		t.Fatalf("unequip omitted name or flags: %q", obj.GetShortDesc())
	}
}

func TestMobileEquipmentBoundaryAlignmentRefusal(t *testing.T) {
	for _, tc := range []struct {
		name            string
		flag, alignment int
	}{{"evil", FlagAntiEvil, -350}, {"good", FlagAntiGood, 350}, {"neutral", FlagAntiNeutral, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			w, m, _ := zoneArmedMob(t)
			obj, err := w.SpawnObject(201, -1)
			if err != nil {
				t.Fatal(err)
			}
			proto := *obj.Prototype
			proto.ExtraFlags[0] |= tc.flag
			obj.Prototype = &proto
			m.SetAlignment(tc.alignment)
			if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
				t.Fatal(err)
			}
			if m.Equipped(mobWearHead) != nil || obj.Location != LocInventoryMob(m.ID) || len(m.Inventory) == 0 || m.Inventory[0] != obj || m.GetAC() != 100 {
				t.Fatal("alignment refusal attached equipment or lost inventory ownership")
			}
		})
	}
}

func TestMobileEquipmentReaderFighterHeadbutt(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	p.SetPosition(combat.PosSleeping)
	m.mu.Lock()
	m.CurrentHP = 200
	m.mu.Unlock()
	before := m.GetHP()
	dprng.ResetStream(1)
	mobHeadbutt(w, m, p)
	if m.GetHP() != before-m.GetLevel()/3 {
		t.Fatalf("fighter head equipment not read: HP=%d want=%d", m.GetHP(), before-m.GetLevel()/3)
	}
}

func TestMobileEquipmentReaderPaladinDisarm(t *testing.T) {
	w, target, _ := zoneArmedMob(t)
	attacker, err := w.SpawnMob(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	target.SetFighting(attacker.GetName())
	attacker.SetFighting(target.GetName())
	weapon := target.Equipped(mobWearWield)
	dprng.ResetStream(1)
	mobDisarm(w, attacker, target)
	if target.Equipped(mobWearWield) != nil || weapon.Location != LocInventoryMob(target.ID) {
		t.Fatal("paladin disarm missed C WEAR_WIELD or unequip path")
	}
}

func TestMobileEquipmentBoundaryBadStats(t *testing.T) {
	t.Run("dexterity", func(t *testing.T) {
		w, m, _ := zoneArmedMob(t)
		m.mu.Lock()
		m.Dex = 0
		m.effectiveAttributes = nil
		m.mu.Unlock()
		obj, err := w.SpawnObject(201, -1)
		if err != nil {
			t.Fatal(err)
		}
		before := m.GetHP()
		if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
			t.Fatal(err)
		}
		if m.GetHP() != before-40 {
			t.Fatalf("zero DEX did not self-damage: HP=%d want=%d", m.GetHP(), before-40)
		}
	})
	t.Run("charisma prey identity", func(t *testing.T) {
		w, m, _ := zoneArmedMob(t)
		proto := *m.Proto()
		proto.VNum = 7907
		proto.ShortDesc = "pestilence"
		w.mu.Lock()
		w.mobs[7907] = &proto
		w.mu.Unlock()
		pest, err := w.SpawnMob(7907, 100)
		if err != nil {
			t.Fatal(err)
		}
		m.mu.Lock()
		m.Cha = 0
		m.effectiveAttributes = nil
		m.mu.Unlock()
		obj, err := w.SpawnObject(201, -1)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
			t.Fatal(err)
		}
		if pest.GetHunting() != m.GetName() || pest.HuntingMobID != m.ID {
			t.Fatal("zero CHA did not retain exact mobile prey")
		}
		eng := &normalCheckerCombatEngine{}
		w.SetCombatEngine(eng)
		w.huntVictim(pest)
		if len(eng.initialAttacks) != 1 || eng.initialAttacks[0] != [2]string{pest.GetName(), m.GetName()} {
			t.Fatalf("mobile prey was not hunted: %#v", eng.initialAttacks)
		}
	})
}

func TestMobileEquipmentBoundaryPointWidths(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	m.SetMaxHP(32760)
	m.Weight = 254
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 18, Modifier: 130}, {Location: 19, Modifier: 130}, {Location: 13, Modifier: 10}, {Location: 10, Modifier: 10}})
	hit, dam := m.GetHitroll(), m.GetDamroll()
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if m.GetHitroll() != hit-126 || m.GetDamroll() != dam-126 || m.GetMaxHP() != -32766 || m.Weight != 8 {
		t.Fatal("mobile equipment ignored C point widths")
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if m.GetHitroll() != hit || m.GetDamroll() != dam || m.GetMaxHP() != 32760 || m.Weight != 254 {
		t.Fatal("point-width unequip did not restore effective points")
	}
	if err := w.MoveObject(obj, LocNowhere()); err != nil {
		t.Fatal(err)
	}
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 13, Modifier: 128}})
	m.SetMaxHP(1000)
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if m.GetMaxHP() != 616 {
		t.Fatalf("C -128 modifier plus affect_total = %d, want 616", m.GetMaxHP())
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if m.GetMaxHP() != 488 {
		t.Fatalf("C -128 inverse assignment = %d, want 488", m.GetMaxHP())
	}
}

func TestMobileEquipmentBoundaryObjectExtraction(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(m, obj, mobWearBody); err != nil {
		t.Fatal(err)
	}
	w.ExtractObject(obj, 100)
	if m.GetAC() != 100 || m.Equipped(mobWearBody) != nil || obj.Location.Kind != ObjNowhere {
		t.Fatal("object extraction omitted mobile unequip effects")
	}
}

func TestMobileEquipmentBoundaryLuaAndRemoval(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	proto := *obj.Prototype
	proto.WearFlags[0] = 1 << 4 // ITEM_WEAR_HEAD, find_eq_pos -> WEAR_HEAD=6
	obj.Prototype = &proto
	if err := w.MoveObjectToMobInventory(obj, m); err != nil {
		t.Fatal(err)
	}
	NewWorldScriptableAdapter(w).EquipCharObj(scripting.CharRef{NPC: true, ID: m.ID}, scripting.ObjRef{ID: obj.ID})
	if m.GetAC() != 80 || m.Equipped(mobWearHead) != obj || obj.Location != LocEquippedMob(m.ID, EquipmentSlot(mobWearHead)) {
		t.Fatal("Lua equip bypassed shared mobile effects")
	}
	if removed := m.UnequipItem(mobWearHead); removed != obj || m.GetAC() != 100 || obj.Location != LocInventoryMob(m.ID) {
		t.Fatal("Lua-equipped item did not undo through shared unequip")
	}
}

func TestMobileEquipmentBoundaryAttributeWidth(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 1, Modifier: 128}, {Location: 6, Modifier: 128}})
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if m.GetStr() != 0 || m.GetCha() != -117 {
		t.Fatalf("mobile attribute bytes did not wrap before limits: STR=%d CHA=%d", m.GetStr(), m.GetCha())
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if m.GetStr() != 11 || m.GetCha() != 11 {
		t.Fatal("attribute-width unequip did not restore base copy")
	}
}

func TestMobileEquipmentReaderLiveWeaponValues(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	weapon := m.Equipped(mobWearWield)
	weapon.SetValue(3, 12)
	typ, _, _, _ := w.WireCombatCallbacks().GetWeaponInfo(m.GetName())
	if typ != 12 {
		t.Fatalf("combat reader ignored live weapon type: %d", typ)
	}
	weapon.SetValue(1, 100)
	weapon.SetValue(2, 1)
	before := p.GetHP()
	dprng.ResetStream(17)
	mobCharge(w, m, p)
	if p.GetHP() > before-200 {
		t.Fatal("paladin charge ignored live weapon dice")
	}
}

func TestMobileEquipmentBoundaryLaterDamrollWrite(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	m.AddDamrollBonus(2)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 19, Modifier: 3}})
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	if m.GetDamrollPoint() != 5 {
		t.Fatal("equipment lost prior point growth")
	}
	m.SetDamroll(50)
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	if m.GetDamrollPoint() != 47 {
		t.Fatalf("unequip did not subtract from later point write: %d", m.GetDamrollPoint())
	}
}

func TestMobileEquipmentReaderSpellSave(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	base := spells.GetSavingThrow(combat.ClassWarrior, m.GetLevel(), spells.SaveSpell)
	var seed uint32
	for candidate := uint32(1); candidate < 10000; candidate++ {
		rng := dprng.New(candidate)
		roll := rng.Number(0, 99)
		if roll > 1 && roll <= base {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("no discriminating saving throw seed")
	}
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	obj.SetAffectsOverride([]parser.ObjAffect{{Location: 24, Modifier: -100}})
	if err := w.EquipMobileObject(m, obj, mobWearHead); err != nil {
		t.Fatal(err)
	}
	dprng.ResetStream(seed)
	if !spells.CheckSavingThrow(m, spells.SaveSpell) {
		t.Fatal("spell save ignored mobile equipment modifier")
	}
	if err := w.MoveObjectToRoom(obj, 100); err != nil {
		t.Fatal(err)
	}
	dprng.ResetStream(seed)
	if spells.CheckSavingThrow(m, spells.SaveSpell) {
		t.Fatal("spell save retained removed equipment modifier")
	}
}

func TestMobileEquipmentBoundaryRetiredResetOwner(t *testing.T) {
	w, m, p := zoneArmedMob(t)
	w.Instakill(m, p, combat.TYPE_SLASH)
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(m, obj, mobWearBody); err != nil {
		t.Fatal(err)
	}
	if m.GetAC() != 70 || m.Equipped(mobWearBody) != obj {
		t.Fatal("later E did not retain the pending C mobile owner")
	}
	if w.countMobInstances(300) != 1 {
		t.Fatal("pending death released C's prototype count too soon")
	}
	if _, ok := NewWorldScriptableAdapter(w).CharFields(scripting.CharRef{NPC: true, ID: m.ID}); !ok {
		t.Fatal("pending extraction discarded C's Lua character handle")
	}
	w.ExtractPendingChars()
	if m.GetAC() != 100 || m.Equipped(mobWearBody) != nil || obj.Location != LocRoom(100) {
		t.Fatal("pending extraction lost equipment attached after death")
	}
	if _, ok := w.GetMobByID(m.ID); ok || w.HasPendingExtractions() {
		t.Fatal("retired owner survived its pending pass")
	}
}

func TestMobileEquipmentConcurrentReaders(t *testing.T) {
	w, m, _ := zoneArmedMob(t)
	obj := m.Equipped(mobWearWield)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			if m.UnequipItem(mobWearWield) != obj {
				t.Error("lost weapon")
			}
			if !m.EquipItem(obj, mobWearWield) {
				t.Error("failed reattach")
			}
		}
	})
	wg.Go(func() {
		for range 100 {
			_ = m.HitModifiers()
			_ = m.EquipmentSnapshot()
			_ = m.GetAC()
			_ = m.GetSavingThrow(4)
			_ = mobHasLight(m)
			_, _ = w.GetRoom(100)
		}
	})
	wg.Wait()
}
