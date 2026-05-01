package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// Condition constants — from structs.h
const (
	CondDrunk  = 0
	CondFull   = 1
	CondThirst = 2
)

// Equipment affect location constants — from structs.h:525-527
const (
	ApplyHitRegen  = 26
	ApplyManaRegen = 27
	ApplyMoveRegen = 28
)

// Level/immortal constants — from structs.h LVL_*
const (
	LVL_IMMORT = 50
	LVL_IMPL   = 54
	LVL_GOD    = 50

	// Idle time limits — from limits.c
	IDLE_TO_VOID     = 20 // cycles before being pulled into void
	IDLE_DISCONNECT  = 30 // cycles before forced disconnect
	MAX_TITLE_LENGTH = 80 // from structs.h
)

// maxExpGain and maxExpLoss cap XP per single kill/death.
// Source: limits.c extern int max_exp_gain, max_exp_loss
var (
	maxExpGain = 100000
	maxExpLoss = 50000
)

// Titles is the class title string array.
// Source: class.c:1087-1099 titles[NUM_CLASSES]
var Titles = []string{
	"the Mage", "the Cleric", "the Thief", "the Warrior", "the Magus",
	"the Avatar", "the Assassin", "the Paladin", "the Ninja", "the Psionic",
	"the Ranger", "the Mystic",
}

// Position constants re-exported from combat package for use within game package.
// Source: structs.h POS_* constants (structs.h:130-138), ported to pkg/combat/formulas.go
const (
	PosDead      = combat.PosDead
	PosMortally  = combat.PosMortally
	PosIncap     = combat.PosIncap
	PosStunned   = combat.PosStunned
	PosSleeping  = combat.PosSleeping
	PosResting   = combat.PosResting
	PosSitting   = combat.PosSitting
	PosFighting  = combat.PosFighting
	PosStanding  = combat.PosStanding
)

// FieldObject represents a field object entry.
// Source: limits.c field_object_data_t field_objs[NUM_FOS]
type FieldObject struct {
	ObjVNum       int
	WornOffObjNum int
	WearOffMsg    string
}

// fieldObjs is the field objects list.
var fieldObjs []FieldObject

// Affect bit constants — from structs.h:321,335,341
const (
	AffPoison    = 11 // AFF_POISON — structs.h:321
	AffCutthroat = 25 // AFF_CUTTHROAT — structs.h:335
	AffFlaming   = 31 // AFF_FLAMING — structs.h:341
)

// ---------------------------------------------------------------------------
// isMystic — from src/utils.h IS_MYSTIC() macro
// ---------------------------------------------------------------------------
func isMystic(p *Player) bool {
	if p == nil {
		return false
	}
	return p.Class == ClassMystic
}

// ---------------------------------------------------------------------------
// isVeteran — from utils.c:358-362
// ---------------------------------------------------------------------------
// playing_time(ch).day >= 30 && GET_KILLS(ch) >= 10000
func isVeteran(p *Player) bool {
	if p == nil {
		return false
	}
	if p.PlayedDuration <= 0 {
		return false
	}
	pt := PlayingTime(p.ConnectedAt, p.PlayedDuration)
	return pt.Day >= 30 && p.Kills >= 10000
}

// ---------------------------------------------------------------------------
// roomHasFlag — checks room flags
// ---------------------------------------------------------------------------
func (w *World) roomHasFlag(roomVNum int, flag string) bool {
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// ManaGain — from limits.c mana_gain() (lines 59-125)
// ---------------------------------------------------------------------------
