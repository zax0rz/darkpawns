package spells

// SavingThrowType matches SAVING_PARA / SAVING_ROD etc. from src/spells.h
type SavingThrowType int

const (
	SaveParalysis SavingThrowType = 0
	SaveRodStaff  SavingThrowType = 1
	SavePetrify   SavingThrowType = 2
	SaveBreath    SavingThrowType = 3
	SaveSpell     SavingThrowType = 4
	SaveCount                    = 5
)

// MagRoutine constants — bitmask values matching C MAG_* defines
// Named RoutineXxx to avoid clash with function names like MagDamage()
type MagRoutine int

const (
	RoutineDamage    MagRoutine = 1 << 0
	RoutineAffects   MagRoutine = 1 << 1
	RoutineUnaffects MagRoutine = 1 << 2
	RoutinePoints    MagRoutine = 1 << 3
	RoutineAlterObjs MagRoutine = 1 << 4
	RoutineGroups    MagRoutine = 1 << 5
	RoutineMasses    MagRoutine = 1 << 6
	RoutineAreas     MagRoutine = 1 << 7
	RoutineSummons   MagRoutine = 1 << 8
	RoutineCreations MagRoutine = 1 << 9
	RoutineManual    MagRoutine = 1 << 10
)

// TargetFlags bitmask constants — matching C TAR_* defines
type TargetFlags int

const (
	TarIgnore     TargetFlags = 1 << 0
	TarCharRoom   TargetFlags = 1 << 1
	TarCharWorld  TargetFlags = 1 << 2
	TarFightSelf  TargetFlags = 1 << 3
	TarFightVict  TargetFlags = 1 << 4
	TarSelfOnly   TargetFlags = 1 << 5
	TarNotSelf    TargetFlags = 1 << 6
	TarObjInv     TargetFlags = 1 << 7
	TarObjRoom    TargetFlags = 1 << 8
	TarObjWorld   TargetFlags = 1 << 9
	TarObjEquip   TargetFlags = 1 << 10
)

// CastType constants
type CastType int

const (
	CastSpell  CastType = 0
	CastPotion CastType = 1
	CastWand   CastType = 2
	CastStaff  CastType = 3
	CastScroll CastType = 4
)

// Position constants — matching C POS_* defines
type Position int

const (
	PosDead            Position = 0
	PosMortallyWounded Position = 1
	PosIncap           Position = 2
	PosStunned         Position = 3
	PosSleeping        Position = 4
	PosResting         Position = 5
	PosSitting         Position = 6
	PosFighting        Position = 7
	PosStanding        Position = 8
)

// SpellRoutines holds routine and target flags for a spell.
type SpellRoutines struct {
	Routines MagRoutine
	Violent  bool
	Targets  TargetFlags
}

// SpellInfo holds the spell template data matching C spell_info_type.
type SpellInfo struct {
	MinPosition Position
	ManaMin     int
	ManaMax     int
	ManaChange  int
	MinLevel    [12]int
	Routines    SpellRoutines
}

// spellInfoTable maps spell number -> SpellInfo.
var spellInfoTable = make(map[int]*SpellInfo)

// GetSpellInfo returns the SpellInfo for a given spell number.
func GetSpellInfo(spellNum int) *SpellInfo {
	return spellInfoTable[spellNum]
}

// SetSpellInfo sets the spell template for a given spell number.
func SetSpellInfo(spellNum int, info *SpellInfo) {
	if spellNum >= 0 {
		spellInfoTable[spellNum] = info
	}
}

// HasRoutine checks if a spell's routines include the given routine bit.
func (si *SpellInfo) HasRoutine(r MagRoutine) bool {
	if si == nil {
		return false
	}
	return si.Routines.Routines&r != 0
}

// HasTarget checks if a spell's targets include the given target flag.
func (si *SpellInfo) HasTarget(t TargetFlags) bool {
	if si == nil {
		return false
	}
	return si.Routines.Targets&t != 0
}

// GetManaCost calculates mana cost for a spell given caster level.
func (si *SpellInfo) GetManaCost(level int) int {
	if si == nil {
		return 0
	}
	cost := si.ManaMax - (si.ManaChange * level)
	if cost < si.ManaMin {
		cost = si.ManaMin
	}
	if cost < 0 {
		cost = 0
	}
	return cost
}

// IsViolent returns true if the spell is considered violent (can't cast in peaceful rooms).
func (si *SpellInfo) IsViolent() bool {
	return si != nil && si.Routines.Violent
}

// setupSpellInfo assigns a spell's template data.
// Mirrors the C spell_info[].field = value pattern in mag_assign_spells().
func setupSpellInfo(spellNum int, minPos Position, manaMin, manaMax, manaChange int, r MagRoutine, violent bool, t TargetFlags) {
	SetSpellInfo(spellNum, &SpellInfo{
		MinPosition: minPos,
		ManaMin:     manaMin,
		ManaMax:     manaMax,
		ManaChange:  manaChange,
		Routines: SpellRoutines{
			Routines: r,
			Violent:  violent,
			Targets:  t,
		},
	})
}

// setSpellLevel sets the minimum level for a given spell and class.
func setSpellLevel(spellNum, class, level int) {
	si := GetSpellInfo(spellNum)
	if si == nil {
		return
	}
	if class >= 0 && class < 12 {
		si.MinLevel[class] = level
	}
}

// AttackType holds the singular and plural forms for an attack type name.
type AttackType struct {
	Singular string
	Plural   string
}

// AttackTypes table — matching C attack_hit_types from custom spell parser
var AttackTypes = []AttackType{
	{},
	{"hit", "hits"},
	{"pound", "pounds"},
	{"smash", "smashes"},
	{"punch", "punches"},
	{"kick", "kicks"},
	{"blast", "blasts"},
	{"pierce", "pierces"},
	{"slash", "slashes"},
	{"chop", "chops"},
	{"claw", "claws"},
	{"bite", "bites"},
	{"sting", "stings"},
	{"scratch", "scratches"},
	{"stab", "stabs"},
	{"crush", "crushes"},
	{"whip", "whips"},
	{"burn", "burns"},
	{"freeze", "freezes"},
	{"shock", "shocks"},
	{"slam", "slams"},
	{"maul", "mauls"},
	{"trample", "tramples"},
	{"wound", "wounds"},
	{"thrust", "thrusts"},
	{"cleave", "cleaves"},
	{"rend", "rends"},
	{"choke", "chokes"},
	{"shatter", "shatters"},
	{"fracture", "fractures"},
	{"sear", "sears"},
	{"corrode", "corrodes"},
	{"dissolve", "dissolves"},
	{"electrocute", "electrocutes"},
}
