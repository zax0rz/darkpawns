package engine

// Apply constants matching CircleMUD APPLY_* from structs.h
const (
	ApplyNone         = 0  // APPLY_NONE
	ApplyStr          = 1  // APPLY_STR
	ApplyDex          = 2  // APPLY_DEX
	ApplyInt          = 3  // APPLY_INT
	ApplyWis          = 4  // APPLY_WIS
	ApplyCon          = 5  // APPLY_CON
	ApplyCha          = 7  // APPLY_CHA
	ApplyLevel        = 8  // APPLY_LEVEL
	ApplyAge          = 9  // APPLY_AGE
	ApplyMana         = 10 // APPLY_MANA
	ApplyHit          = 11 // APPLY_HIT
	ApplyMove         = 12 // APPLY_MOVE
	ApplyAC           = 17 // APPLY_AC
	ApplyHitroll      = 18 // APPLY_HITROLL
	ApplyDamroll      = 19 // APPLY_DAMROLL
	ApplySavingPara   = 20 // APPLY_SAVING_PARA
	ApplySavingRod    = 21 // APPLY_SAVING_ROD
	ApplySavingPetri  = 22 // APPLY_SAVING_PETRI
	ApplySavingBreath = 23 // APPLY_SAVING_BREATH
	ApplySavingSpell  = 24 // APPLY_SAVING_SPELL
	ApplySpell        = 26 // APPLY_SPELL

	AFArrayMax = 2 // 64 bits = 2 x 32 bit words
)

// MasterAffect represents a master_affected_type from CircleMUD (handler.c).
// It holds a spell/effect affect with full metadata for iteration and removal.
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
			GetAffects() []interface {
				GetLocation() int
				GetModifier() int
			}
			GetBitvector() uint64
		}
	}
}

// ApplyFunction is the signature for aff_apply_modify style callbacks.
type ApplyFunction func(ch interface{}, loc int, mod int, msg string)

// DefaultApplyModify implements aff_apply_modify from handler.c (lines 136-276).
func DefaultApplyModify(ch interface{}, loc int, mod int, msg string) {
	switch loc {
	case ApplyNone:
		return
	case ApplyStr:
		setStat(ch, "STR", mod)
	case ApplyDex:
		setStat(ch, "DEX", mod)
	case ApplyInt:
		setStat(ch, "INT", mod)
	case ApplyWis:
		setStat(ch, "WIS", mod)
	case ApplyCon:
		setStat(ch, "CON", mod)
	case ApplyCha:
		setStat(ch, "CHA", mod)
	case ApplyMana:
		addMaxStat(ch, "Mana", mod)
	case ApplyHit:
		addMaxStat(ch, "HP", mod)
	case ApplyMove:
		addMaxStat(ch, "Move", mod)
	case ApplyAC:
		addStat(ch, "AC", -mod)
	case ApplyHitroll:
		addStat(ch, "Hitroll", mod)
	case ApplyDamroll:
		addStat(ch, "Damroll", mod)
	case ApplySavingPara:
		addSavingThrow(ch, 0, mod)
	case ApplySavingRod:
		addSavingThrow(ch, 1, mod)
	case ApplySavingPetri:
		addSavingThrow(ch, 2, mod)
	case ApplySavingBreath:
		addSavingThrow(ch, 3, mod)
	case ApplySavingSpell:
		addSavingThrow(ch, 4, mod)
	}
}

func setStat(ch interface{}, name string, mod int) {
	if sm, ok := ch.(StatModifiable); ok {
		cur := sm.GetStat(name)
		sm.SetStat(name, cur+mod)
	}
}

func addMaxStat(ch interface{}, name string, mod int) {
	if sm, ok := ch.(StatModifiable); ok {
		cur := sm.GetMaxStat(name)
		sm.SetMaxStat(name, cur+mod)
	}
}

func addStat(ch interface{}, name string, mod int) {
	if sm, ok := ch.(StatModifiable); ok {
		sm.AddStat(name, mod)
	}
}

func addSavingThrow(ch interface{}, idx int, mod int) {
	if sm, ok := ch.(StatModifiable); ok {
		cur := sm.GetSavingThrow(idx)
		sm.SetSavingThrow(idx, cur+mod)
	}
}

// AffModify implements affect_modify from handler.c line 280.
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
// For item affects with array-based bitvectors. Since Go has uint64, we treat
// bitv as the raw bitvector value.
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
// Recalculates all affects for a character by removing and re-applying.
func AffectTotal(ch interface{}, applyFn ApplyFunction) {
	if applyFn == nil {
		applyFn = DefaultApplyModify
	}
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}

	// Phase 1: Remove all affect modifiers from equipment
	// Phase 2: Remove all affect modifiers from master affects
	// Phase 3: Remove tattoo effects
	// Phase 4: Reset abils to base real_abils
	// Phase 5: Re-apply all in reverse order
	// Phase 6: Clamp values

	// For now, the simplified model is: iterate all affects, strip, reapply.
	// The Player struct will need to implement these methods.

	// Actually modify the ch's actual fields by calling DefaultApplyModify
	// in the right order (remove eq, remove affects, reapply eq, reapply affects)
	masterAffects := sm.GetMasterAffects()
	if masterAffects == nil {
		return
	}

	// Remove pass
	for _, af := range masterAffects {
		AffModify(ch, af.Location, af.Modifier, af.Bitvector, false, applyFn)
	}

	// Re-apply pass
	for _, af := range masterAffects {
		AffModify(ch, af.Location, af.Modifier, af.Bitvector, true, applyFn)
	}
}

// MasterAffectToChar implements master_affect_to_char from handler.c lines 377-396.
// Creates a new MasterAffect from an Affect and appends it to the character's list.
func MasterAffectToChar(ch interface{}, af *Affect, byType int, objNum int) {
	sm, ok := ch.(StatModifiable)
	if !ok {
		return
	}

	masterAf := &MasterAffect{
		Type:      int(af.Type),
		Duration:  af.Duration,
		Location:  applyLocFromAffectType(af.Type),
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
// Wrapper — passes BY_SPELL, obj_num=0.
func AffectToChar(ch interface{}, af *Affect) {
	MasterAffectToChar(ch, af, BySpell, 0)
}

// AffectToChar2 implements affect_to_char2 from handler.c lines 405-427.
// Copies a MasterAffect to the character's list with BY_SPELL origin.
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
// Removes all affects of the given type from the character.
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
				af.Duration >>= 1 // divide by 2
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
	// No existing affect found — just add
	AffectToChar2(ch, af)
}

// applyLocFromAffectType maps engine.AffectType to APPLY_* constants.
func applyLocFromAffectType(affType AffectType) int {
	switch affType {
	case AffectStrength:
		return ApplyStr
	case AffectDexterity:
		return ApplyDex
	case AffectIntelligence:
		return ApplyInt
	case AffectWisdom:
		return ApplyWis
	case AffectConstitution:
		return ApplyCon
	case AffectCharisma:
		return ApplyCha
	case AffectHitRoll:
		return ApplyHitroll
	case AffectDamageRoll:
		return ApplyDamroll
	case AffectArmorClass:
		return ApplyAC
	case AffectHP, AffectMaxHP:
		return ApplyHit
	case AffectMana, AffectMaxMana:
		return ApplyMana
	case AffectMovement:
		return ApplyMove
	default:
		return ApplyNone
	}
}

// SetBitVector sets a bit in a bitmask (for uint64 wraparound).
func SetBitVector(bv *uint64, bit uint) {
	*bv |= 1 << bit
}

// RemoveBitVector clears a bit in a bitmask.
func RemoveBitVector(bv *uint64, bit uint) {
	*bv &^= 1 << bit
}

// IsSetBitVector checks if a bit is set in a bitmask.
func IsSetBitVector(bv uint64, bit uint) bool {
	return bv&(1<<bit) != 0
}
