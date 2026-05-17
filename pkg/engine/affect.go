package engine

import (
	"time"

	"math/rand/v2"
)

// TickDuration is the real-world duration of one affect tick.
// In C: affect_update() fires every SECS_PER_MUD_HOUR (63s) * PASSES_PER_SEC,
// so each duration unit = 1 mud hour. In Go the gameloop fires every 60s,
// so 1 tick = 60 real seconds.
const TickDuration = 60 * time.Second

// Origin constants — who created this affect?
const (
	OriginNone  = 0
	OriginSpell = 1 // BY_SPELL from handler.c
	OriginItem  = 2 // BY_ITEM from handler.c
)

// Affect represents a temporary effect on a character, mob, or object.
// Unified system: spells, equipment, and items all use this single struct.
//
// In CircleMUD, affects are identified by a spell ID (SPELL_* constant) and
// an application location (APPLY_* constant). The spell ID tells us which
// spell/equipment created the affect; the location tells us which stat to modify.
type Affect struct {
	ID        string // Unique identifier for this affect
	SpellID   int    // SPELL_* or SKILL_* number. 0 = not spell-based.
	Location  int    // APPLY_* constant — which stat this affect modifies
	Duration  int    // Duration in ticks (0 = permanent until removed)
	Magnitude int    // Stat modifier magnitude (positive = buff, negative = debuff)

	// Flags
	Flags uint64 // AFF_* bitvector — status flags to set on the entity

	// Metadata
	Source    string    // Human-readable source name ("bless", "long sword")
	Origin    int       // OriginSpell, OriginItem, or OriginNone
	ObjNum    int       // Object VNUM if from equipment (0 for spells)
	AppliedAt time.Time // When the affect was applied
	ExpiresAt time.Time // When the affect expires (calculated from Duration)

	// Stacking information
	StackID   string // ID for dedup purposes (empty = doesn't deduplicate)
	MaxStacks int    // Maximum number of stacks (0 = infinite, 1 = doesn't stack)

	// Deprecated: Type is an alias for backward compatibility.
	// For status affects, stores the status constant. For stat affects, stores the Location.
	// Use SpellID + Location directly in new code.
	Type int `json:"type"` //nolint:govet // deprecated field for save/load compat
}

// NewAffect creates a new affect with the given parameters.
//   - spellID: SPELL_* or SKILL_* constant that created this affect
//   - location: APPLY_* constant — which stat to modify
//   - duration: ticks (0 = permanent)
//   - magnitude: stat modifier
//   - source: human-readable name
func NewAffect(spellID int, location int, duration int, magnitude int, source string) *Affect {
	now := time.Now()
	affect := &Affect{
		ID:        generateAffectID(),
		SpellID:   spellID,
		Location:  location,
		Duration:  duration,
		Magnitude: magnitude,
		Source:    source,
		AppliedAt: now,
		ExpiresAt: now.Add(time.Duration(duration) * TickDuration),
		Flags:     0,
		Origin:    OriginNone,
		ObjNum:    0,
		StackID:   "",
		MaxStacks: 1,
		Type:      location, // backward compat: stat affects store Location in Type
	}

	// Auto-set StackID for status-flag affects to prevent duplicates.
	if flags, ok := spellStatusFlags[spellID]; ok && flags != 0 {
		affect.Flags = flags
		affect.StackID = spellStackKey(spellID)
		affect.MaxStacks = 1
	}

	return affect
}

// NewAffectDeprecated is backward-compatible with the old NewAffect(affectType, duration, magnitude, source) signature.
// DEPRECATED: Use NewAffect(spellID, location, duration, magnitude, source) instead.
func NewAffectDeprecated(affectType int, duration int, magnitude int, source string) *Affect {
	// Check if this is a status affect (has flags)
	if flags, ok := StatusAffectFlags[affectType]; ok {
		af := NewAffectDirect(0, ApplyNone, duration, magnitude, flags, source)
		af.Type = affectType // backward compat
		return af
	}
	// Otherwise treat as a stat affect — affectType IS the location
	return NewAffect(0, affectType, duration, magnitude, source)
}

// NewAffectDirect creates an affect with explicit flags and stack settings.
// Used by equipment and item code that needs full control over the affect.
func NewAffectDirect(spellID int, location int, duration int, magnitude int, flags uint64, source string) *Affect {
	now := time.Now()
	affect := &Affect{
		ID:        generateAffectID(),
		SpellID:   spellID,
		Location:  location,
		Duration:  duration,
		Magnitude: magnitude,
		Source:    source,
		AppliedAt: now,
		ExpiresAt: now.Add(time.Duration(duration) * TickDuration),
		Flags:     flags,
		Origin:    OriginNone,
		ObjNum:    0,
		StackID:   "",
		MaxStacks: 1,
		Type:      location, // backward compat
	}

	if flags != 0 {
		affect.StackID = spellStackKey(spellID)
		affect.MaxStacks = 1
	}

	return affect
}

// spellStackKey generates a stack dedup key for a spell.
func spellStackKey(spellID int) string {
	return "spell_" + string(rune(spellID))
}

// Type returns the deprecated AffectType value.
// For status affects, returns the status constant (AffectBlind, etc.).
// For stat affects, returns the Location (APPLY_*) constant.
// DEPRECATED: Use SpellID + Location directly.
func (a *Affect) GetType() int {
	if a.Flags != 0 {
		// Status affect — return the status constant
		for affType, flags := range StatusAffectFlags {
			if a.Flags&flags != 0 {
				return affType
			}
		}
	}
	// Stat affect — return the location
	return a.Location
}

// SetType sets the deprecated AffectType value.
// DEPRECATED: Set SpellID + Location directly instead.
func (a *Affect) SetType(v int) {
	if flags, ok := StatusAffectFlags[v]; ok {
		a.Flags = flags
	} else {
		a.Location = v
	}
}

// IsExpired checks if the affect has expired
func (a *Affect) IsExpired() bool {
	if a.Duration == 0 {
		return false // Permanent
	}
	return time.Now().After(a.ExpiresAt)
}

// Tick reduces the duration by one tick and returns true if expired.
// Each tick represents one mud hour (60 real seconds).
func (a *Affect) Tick() bool {
	if a.Duration == 0 {
		return false // Permanent, never expires
	}

	a.Duration--
	if a.Duration <= 0 {
		return true
	}

	a.ExpiresAt = a.ExpiresAt.Add(-TickDuration)
	return false
}

// GetRemainingDuration returns the remaining duration in ticks
func (a *Affect) GetRemainingDuration() int {
	return a.Duration
}

// SetFlag sets a specific flag
func (a *Affect) SetFlag(flag uint64) {
	a.Flags |= flag
}

// ClearFlag clears a specific flag
func (a *Affect) ClearFlag(flag uint64) {
	a.Flags &^= flag
}

// HasFlag checks if a specific flag is set
func (a *Affect) HasFlag(flag uint64) bool {
	return a.Flags&flag != 0
}

// Helper function to generate a unique affect ID
func generateAffectID() string {
	return "aff_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random alphanumeric string of the given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

// --- APPLY_* location constants (from CircleMUD structs.h) ---

const (
	ApplyNone         = 0
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

	// Regen locations (extended from C for item affects)
	ApplyManaRegen = 27
	ApplyHitRegen  = 28
	ApplyMoveRegen = 29
)

// --- AFF_* status flag bit positions (from CircleMUD structs.h) ---

const (
	AFFNone              uint64 = 0
	AFFBlind             uint64 = 1 << 0  // AFF_BLIND
	AFFInvisible         uint64 = 1 << 1  // AFF_INVISIBLE
	AFFDetectInvisible   uint64 = 1 << 2  // AFF_DETECT_INVIS
	AFFDetectMagic       uint64 = 1 << 3  // AFF_DETECT_MAGIC
	AFFSanctuary         uint64 = 1 << 4  // AFF_SANCTUARY
	AFFFlying            uint64 = 1 << 5  // AFF_FLYING
	AFFFloating          uint64 = 1 << 6  // AFF_FLOATING
	AFFPassDoor          uint64 = 1 << 7  // AFF_PASSDOOR
	AFFSneak             uint64 = 1 << 8  // AFF_SNEAK
	AFFHide              uint64 = 1 << 9  // AFF_HIDE
	AFFCharm             uint64 = 1 << 10 // AFF_CHARM
	AFFPoison            uint64 = 1 << 11 // AFF_POISON
	AFFSleep             uint64 = 1 << 12 // AFF_SLEEP
	AFFStunned           uint64 = 1 << 13 // AFF_STUNNED
	AFFParalyzed         uint64 = 1 << 14 // AFF_PARALYZED
	AFFFlaming           uint64 = 1 << 15 // AFF_FLAMING
	AFFHaste             uint64 = 1 << 16 // AFF_HASTE
	AFFSlow              uint64 = 1 << 17 // AFF_SLOW
	AFFProtectionEvil    uint64 = 1 << 18 // AFF_PROTECT_EVIL
	AFFProtectionGood    uint64 = 1 << 19 // AFF_PROTECT_GOOD
	AFFFear              uint64 = 1 << 20 // AFF_FEAR
	AFFCurse             uint64 = 1 << 21 // AFF_CURSE
	AFFSilence           uint64 = 1 << 22 // AFF_SILENCE
	AFFWaterBreathing    uint64 = 1 << 23 // AFF_WATER_BREATH
	AFFRegeneration      uint64 = 1 << 24 // AFF_REGENERATION
	AFFInfrared          uint64 = 1 << 25 // AFF_INFRARED
	AFFUltraviolet       uint64 = 1 << 26 // AFF_ULTRAVIOLET
	AFFDetectAlign       uint64 = 1 << 27 // AFF_DETECT_ALIGN
	AFFSenseLife         uint64 = 1 << 28 // AFF_SENSE_LIFE
	AFFDream             uint64 = 1 << 29 // AFF_DREAM
	AFFMindBar           uint64 = 1 << 30 // AFF_MIND_BAR
	AFFWaterwalk         uint64 = 1 << 31 // AFF_WATERWALK
	AFFMetalskin         uint64 = 1 << 32 // AFF_METALSKIN
	AFFInvuln            uint64 = 1 << 33 // AFF_INVULN
)

// --- Location → stat name table ---

// applyLocationToStat maps APPLY_* constants to entity stat names.
// Used by AffectManager for table-driven stat modification.
var applyLocationToStat = map[int]string{
	ApplyStr:          "STR",
	ApplyDex:          "DEX",
	ApplyInt:          "INT",
	ApplyWis:          "WIS",
	ApplyCon:          "CON",
	ApplyCha:          "CHA",
	ApplyMana:         "Mana",
	ApplyHit:          "HP",
	ApplyMove:         "Move",
	ApplyAC:           "AC",
	ApplyHitroll:      "Hitroll",
	ApplyDamroll:      "Damroll",
	ApplySavingPara:   "SavingPara",
	ApplySavingRod:    "SavingRod",
	ApplySavingPetri:  "SavingPetri",
	ApplySavingBreath: "SavingBreath",
	ApplySavingSpell:  "SavingSpell",
}

// --- Spell → default status flags table ---

// spellStatusFlags maps SPELL_* constants to their default AFF_* flags.
// When NewAffect is called with a known spellID, the flags are set automatically.
var spellStatusFlags = map[int]uint64{
	// These will be populated when we migrate spell constants.
	// For now, affects created via NewAffectDirect can set flags explicitly.
}

// StatusAffectFlags maps legacy AffectType enum values to AFF_* flags.
// Used by restoreAffects to reconstruct flags from old save files.
var StatusAffectFlags = map[int]uint64{
	100: AFFBlind,
	101: AFFInvisible,
	102: AFFDetectInvisible,
	103: AFFDetectMagic,
	104: AFFSanctuary,
	105: AFFFlying,
	106: AFFFloating,
	107: AFFPassDoor,
	108: AFFSneak,
	109: AFFHide,
	110: AFFCharm,
	111: AFFPoison,
	112: AFFSleep,
	113: AFFStunned,
	114: AFFParalyzed,
	115: AFFFlaming,
	116: AFFHaste,
	117: AFFSlow,
	118: AFFProtectionEvil,
	119: AFFProtectionGood,
	120: AFFFear,
	121: AFFCurse,
	122: AFFSilence,
	123: AFFWaterBreathing,
	124: AFFRegeneration,
	125: AFFInfrared,
	126: AFFUltraviolet,
	127: AFFDetectAlign,
	128: AFFSenseLife,
	129: AFFDream,
	130: AFFMindBar,
	131: AFFWaterwalk,
	132: AFFMetalskin,
	133: AFFInvuln,
}

// NewAffectCompat is backward-compatible with the old NewAffect(affectType, duration, magnitude, source) signature.
// DEPRECATED: Use NewAffect(spellID, location, duration, magnitude, source) instead.
func NewAffectCompat(affectType int, duration int, magnitude int, source string) *Affect {
	// Check if this is a status affect (has flags)
	if flags, ok := StatusAffectFlags[affectType]; ok {
		return NewAffectDirect(0, ApplyNone, duration, magnitude, flags, source)
	}
	// Otherwise treat as a stat affect — affectType IS the location
	return NewAffect(0, affectType, duration, magnitude, source)
}
