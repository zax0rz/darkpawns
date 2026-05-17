package engine

// --- Old helper functions ---
// These are retained temporarily for backward compatibility during migration.
// They will be removed or rewritten in Phase 2.

// MasterAffect represents a master_affected_type from CircleMUD (handler.c).
// DEPRECATED: Use Affect directly. Retained for migration compatibility.
type MasterAffect struct {
	Type      int    // Spell/skill type (SPELL_XXX or SKILL_XXX)
	Duration  int    // Duration in ticks
	Location  int    // APPLY_XXX constant
	Modifier  int    // Magnitude of the modifier
	Bitvector uint64 // AFF_XXX flag(s) affected
	ByType    int    // BY_SPELL, BY_ITEM, etc.
	ObjNum    int    // Object VNUM if from item (0 for spells)
}

// ByType constants from structs.h / handler.c
const (
	BySpell = 1 // BY_SPELL
	ByItem  = 2 // BY_ITEM
)

// StatModifiable is the interface characters must implement for affect support.
// DEPRECATED: Use Affectable instead. Retained for migration compatibility.
type StatModifiable interface {
	GetStat(name string) int
	SetStat(name string, val int)
	GetMaxStat(name string) int
	SetMaxStat(name string, val int)
	AddStat(name string, delta int)
	GetSavingThrow(idx int) int
	SetSavingThrow(idx int, val int)
	GetAffectBitVector() uint64
	SetAffectBitVector(v uint64)
	SetAffectBit(bit uint64, val bool)
	GetMasterAffects() []*MasterAffect
	SetMasterAffects(affects []*MasterAffect)
	AddMasterAffect(af *MasterAffect)
	RemoveMasterAffect(af *MasterAffect)
	GetEquipment() interface {
		GetItems() []interface {
			GetAffects() []interface{ GetLocation() int; GetModifier() int }
			GetBitvector() uint64
		}
	}
}

// EquipAffectData holds location/modifier/bitvector for one equipment affect.
type EquipAffectData struct {
	Location  int
	Modifier  int
	Bitvector uint64
}

// EquipAffectProvider is an optional interface for characters with equipment.
// If AffectTotal's target implements this, equipment affects are included
// in the strip-and-reapply cycle (matching C's affect_total).
type EquipAffectProvider interface {
	GetEquipAffects() []EquipAffectData
}

// ApplyFunction is the signature for aff_apply_modify style callbacks.
type ApplyFunction func(ch interface{}, loc int, mod int, msg string)

// DefaultApplyModify implements aff_apply_modify from handler.c (lines 136-276).
// DEPRECATED: Use applyLocationToStat table in affect.go instead.
func DefaultApplyModify(ch interface{}, loc int, mod int, msg string) {
	if sm, ok := ch.(StatModifiable); ok {
		switch loc {
		case ApplyNone:
			return
		case ApplyStr:
			sm.SetStat("STR", sm.GetStat("STR")+mod)
		case ApplyDex:
			sm.SetStat("DEX", sm.GetStat("DEX")+mod)
		case ApplyInt:
			sm.SetStat("INT", sm.GetStat("INT")+mod)
		case ApplyWis:
			sm.SetStat("WIS", sm.GetStat("WIS")+mod)
		case ApplyCon:
			sm.SetStat("CON", sm.GetStat("CON")+mod)
		case ApplyCha:
			sm.SetStat("CHA", sm.GetStat("CHA")+mod)
		case ApplyMana:
			sm.SetMaxStat("Mana", sm.GetMaxStat("Mana")+mod)
		case ApplyHit:
			sm.SetMaxStat("HP", sm.GetMaxStat("HP")+mod)
		case ApplyMove:
			sm.SetMaxStat("Move", sm.GetMaxStat("Move")+mod)
		case ApplyAC:
			sm.SetStat("AC", sm.GetStat("AC")-mod)
		case ApplyHitroll:
			sm.SetStat("Hitroll", sm.GetStat("Hitroll")+mod)
		case ApplyDamroll:
			sm.SetStat("Damroll", sm.GetStat("Damroll")+mod)
		case ApplySavingPara:
			sm.SetSavingThrow(0, sm.GetSavingThrow(0)+mod)
		case ApplySavingRod:
			sm.SetSavingThrow(1, sm.GetSavingThrow(1)+mod)
		case ApplySavingPetri:
			sm.SetSavingThrow(2, sm.GetSavingThrow(2)+mod)
		case ApplySavingBreath:
			sm.SetSavingThrow(3, sm.GetSavingThrow(3)+mod)
		case ApplySavingSpell:
			sm.SetSavingThrow(4, sm.GetSavingThrow(4)+mod)
		}
	}
}

// AffModify implements affect_modify from handler.c line 280.
// DEPRECATED: Use AffectManager.applyAffectImmediate instead.
func AffModify(ch interface{}, loc int, mod int, bitv uint64, add bool, applyFn ApplyFunction) {
	if applyFn == nil {
		applyFn = DefaultApplyModify
	}
	if sm, ok := ch.(StatModifiable); ok && bitv != 0 {
		if add {
			sm.SetAffectBit(bitv, true)
		} else {
			sm.SetAffectBit(bitv, false)
			mod = -mod
		}
	} else if !add {
		mod = -mod
	}
	applyFn(ch, loc, mod, "affect_modify")
}

// AffectModifyAR implements affect_modify_ar from handler.c line 294.
// DEPRECATED: Use AffectManager.applyAffectImmediate instead.
func AffectModifyAR(ch interface{}, loc int, mod int, bitv uint64, add bool, applyFn ApplyFunction) {
	if applyFn == nil {
		applyFn = DefaultApplyModify
	}
	if sm, ok := ch.(StatModifiable); ok && bitv != 0 {
		if add {
			sm.SetAffectBit(bitv, true)
		} else {
			sm.SetAffectBit(bitv, false)
			mod = -mod
		}
	} else if !add {
		mod = -mod
	}
	applyFn(ch, loc, mod, "affect_modify_ar")
}

// AffectTotal implements affect_total from handler.c lines 314-373.
// DEPRECATED: Use AffectManager.RecalculateStats instead.
func AffectTotal(ch interface{}, applyFn ApplyFunction) {
	if applyFn == nil {
		applyFn = DefaultApplyModify
	}
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}

	masterAffects := sm.GetMasterAffects()

	var equipAffects []EquipAffectData
	if ep, ok := ch.(EquipAffectProvider); ok {
		equipAffects = ep.GetEquipAffects()
	}

	for _, ea := range equipAffects {
		AffectModifyAR(ch, ea.Location, ea.Modifier, ea.Bitvector, false, applyFn)
	}
	for _, af := range masterAffects {
		AffModify(ch, af.Location, af.Modifier, af.Bitvector, false, applyFn)
	}

	for _, ea := range equipAffects {
		AffectModifyAR(ch, ea.Location, ea.Modifier, ea.Bitvector, true, applyFn)
	}
	for _, af := range masterAffects {
		AffModify(ch, af.Location, af.Modifier, af.Bitvector, true, applyFn)
	}

	isNPC := false
	if npc, ok := ch.(interface{ IsNPC() bool }); ok {
		isNPC = npc.IsNPC()
	}
	maxStat := 18
	if isNPC {
		maxStat = 25
	}

	for _, s := range []string{"DEX", "INT", "WIS", "CON"} {
		v := sm.GetStat(s)
		if v < 0 {
			sm.SetStat(s, 0)
		} else if v > maxStat {
			sm.SetStat(s, maxStat)
		}
	}

	str := sm.GetStat("STR")
	if str < 0 {
		sm.SetStat("STR", 0)
	} else if str > 18 && !isNPC {
		strAdd := sm.GetStat("StrAdd")
		i := strAdd + ((str - 18) * 10)
		if i > 100 {
			i = 100
		}
		sm.SetStat("STR", 18)
		sm.SetStat("StrAdd", i)
	} else if str > maxStat {
		sm.SetStat("STR", maxStat)
	}

	align := sm.GetStat("Alignment")
	if align < -1000 {
		sm.SetStat("Alignment", -1000)
	} else if align > 1000 {
		sm.SetStat("Alignment", 1000)
	}
}

// MasterAffectToChar implements master_affect_to_char from handler.c lines 377-396.
// DEPRECATED: Use AffectManager.ApplyAffect directly.
func MasterAffectToChar(ch interface{}, af *Affect, byType int, objNum int) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}

	masterAf := &MasterAffect{
		Type:      int(af.SpellID),
		Duration:  af.Duration,
		Location:  af.Location,
		Modifier:  af.Magnitude,
		Bitvector: af.Flags,
		ByType:    byType,
		ObjNum:    objNum,
	}

	sm.AddMasterAffect(masterAf)
	AffModify(ch, masterAf.Location, masterAf.Modifier, masterAf.Bitvector, true, nil)
	AffectTotal(ch, nil)
}

// AffectToChar implements affect_to_char from handler.c line 400.
// DEPRECATED: Use AffectManager.ApplyAffect directly.
func AffectToChar(ch interface{}, af *Affect) {
	MasterAffectToChar(ch, af, BySpell, 0)
}

// AffectToChar2 implements affect_to_char2 from handler.c lines 405-427.
// DEPRECATED: Use AffectManager.ApplyAffect directly.
func AffectToChar2(ch interface{}, af *MasterAffect) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}

	masterAf := &MasterAffect{
		Type:      af.Type,
		Duration:  af.Duration,
		Location:  af.Location,
		Modifier:  af.Modifier,
		Bitvector: af.Bitvector,
		ByType:    BySpell,
		ObjNum:    0,
	}
	sm.AddMasterAffect(masterAf)
	AffModify(ch, masterAf.Location, masterAf.Modifier, masterAf.Bitvector, true, nil)
	AffectTotal(ch, nil)
}

// AffectRemove implements affect_remove from handler.c lines 428-438.
// DEPRECATED: Use AffectManager.RemoveAffect directly.
func AffectRemove(ch interface{}, af *MasterAffect) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}
	AffModify(ch, af.Location, af.Modifier, af.Bitvector, false, nil)
	sm.RemoveMasterAffect(af)
	AffectTotal(ch, nil)
}

// AffectFromChar implements affect_from_char from handler.c lines 443-452.
// DEPRECATED: Use AffectManager.RemoveAffectsBySpell directly.
func AffectFromChar(ch interface{}, spellType int) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}
	affects := sm.GetMasterAffects()
	for _, af := range affects {
		if af.Type == spellType {
			AffectRemove(ch, af)
		}
	}
}

// AffectedBySpell implements affected_by_spell from handler.c lines 460-469.
// DEPRECATED: Use AffectManager.HasAffectBySpell directly.
func AffectedBySpell(ch interface{}, spellType int) bool {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return false
	}
	for _, af := range sm.GetMasterAffects() {
		if af.Type == spellType {
			return true
		}
	}
	return false
}

// AffectJoin implements affect_join from handler.c lines 473-499.
// DEPRECATED: Use AffectManager directly.
func AffectJoin(ch interface{}, af *MasterAffect, addDur, avgDur, addMod, avgMod bool) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}
	for _, existing := range sm.GetMasterAffects() {
		if existing.Type == af.Type && existing.Location == af.Location {
			if addDur {
				af.Duration += existing.Duration
			}
			if avgDur {
				af.Duration >>= 1
			}
			if addMod {
				af.Modifier += existing.Modifier
			}
			if avgMod {
				af.Modifier >>= 1
			}
			AffectRemove(ch, existing)
			AffectToChar2(ch, af)
			return
		}
	}
	AffectToChar2(ch, af)
}
