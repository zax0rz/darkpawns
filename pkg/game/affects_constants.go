package game

import "github.com/zax0rz/darkpawns/pkg/engine"

// C-style bit index positions for player and mob affects.
// Source: structs.h AFF_* constants.
const (
	affBlind           = 0
	affInvisible       = 1
	affDetectAlign     = 2
	affDetectInvisible = 3
	affDetectMagic     = 4
	affSenseLife       = 5
	affWaterWalk       = 6
	affSanctuary       = 7
	affGroup           = 8
	affCurse           = 9
	affInfravision     = 10
	affPoison          = 11
	affProtectEvil     = 12
	affProtectGood     = 13
	affSleep           = 14
	affNoTrack         = 15
	affFleshAlter      = 16
	affDodge           = 17
	affSneak           = 18
	affHide            = 19
	affBerserk         = 20
	affCharm           = 21
	affFollow          = 22
	affWimpy           = 23
	affKujiKiri        = 24
	affCutthroat       = 25
	affFly             = 26
	affWerewolf        = 27
	affVampire         = 28
	affMount           = 29
	affInvuln          = 30
	affFlaming         = 31
	affNothing         = 32
	affHaste           = 33
	affSlow            = 34
	affDream           = 35
	affWaterBreathe    = 36
	affMetalskin       = 37
	affRobbed          = 38
)

// Exported constants for compatibility with outside packages (e.g. command)
const (
	AffSleep   = affSleep
	AffHide    = affHide
	affMounted = affMount
)

// AffBitToEngineFlag maps C bit positions to engine flag uint64 values.
var AffBitToEngineFlag = map[int]uint64{
	affBlind:           engine.AFFBlind,
	affInvisible:       engine.AFFInvisible,
	affDetectAlign:     engine.AFFDetectAlign,
	affDetectInvisible: engine.AFFDetectInvisible,
	affDetectMagic:     engine.AFFDetectMagic,
	affSenseLife:       engine.AFFSenseLife,
	affWaterWalk:       engine.AFFWaterwalk,
	affSanctuary:       engine.AFFSanctuary,
	affGroup:           engine.AFFGroup,
	affCurse:           engine.AFFCurse,
	affInfravision:     engine.AFFInfrared,
	affPoison:          engine.AFFPoison,
	affProtectEvil:     engine.AFFProtectionEvil,
	affProtectGood:     engine.AFFProtectionGood,
	affSleep:           engine.AFFSleep,
	affNoTrack:         engine.AFFNoTrack,
	affFleshAlter:      engine.AFFFleshAlter,
	affDodge:           engine.AFFDodge,
	affSneak:           engine.AFFSneak,
	affHide:            engine.AFFHide,
	affBerserk:         engine.AFFBerserk,
	affCharm:           engine.AFFCharm,
	affFollow:          engine.AFFFollow,
	affWimpy:           engine.AFFWimpy,
	affKujiKiri:        engine.AFFKujiKiri,
	affCutthroat:       engine.AFFCutthroat,
	affFly:             engine.AFFFlying,
	affWerewolf:        engine.AFFWerewolf,
	affVampire:         engine.AFFVampire,
	affMount:           engine.AFFMount,
	affInvuln:          engine.AFFInvuln,
	affFlaming:         engine.AFFFlaming,
	affNothing:         engine.AFFNothing,
	affHaste:           engine.AFFHaste,
	affSlow:            engine.AFFSlow,
	affDream:           engine.AFFDream,
	affWaterBreathe:    engine.AFFWaterBreathing,
	affMetalskin:       engine.AFFMetalskin,
	affRobbed:          engine.AFFRobbed,
}

// EngineFlagToAffBit maps engine status flags back to C bit positions.
var EngineFlagToAffBit = map[uint64]int{
	engine.AFFBlind:           affBlind,
	engine.AFFInvisible:       affInvisible,
	engine.AFFDetectAlign:     affDetectAlign,
	engine.AFFDetectInvisible: affDetectInvisible,
	engine.AFFDetectMagic:     affDetectMagic,
	engine.AFFSenseLife:       affSenseLife,
	engine.AFFWaterwalk:       affWaterWalk,
	engine.AFFSanctuary:       affSanctuary,
	engine.AFFGroup:           affGroup,
	engine.AFFCurse:           affCurse,
	engine.AFFInfrared:        affInfravision,
	engine.AFFPoison:          affPoison,
	engine.AFFProtectionEvil:  affProtectEvil,
	engine.AFFProtectionGood:  affProtectGood,
	engine.AFFSleep:           affSleep,
	engine.AFFNoTrack:         affNoTrack,
	engine.AFFFleshAlter:      affFleshAlter,
	engine.AFFDodge:           affDodge,
	engine.AFFSneak:           affSneak,
	engine.AFFHide:            affHide,
	engine.AFFBerserk:         affBerserk,
	engine.AFFCharm:           affCharm,
	engine.AFFFollow:          affFollow,
	engine.AFFWimpy:           affWimpy,
	engine.AFFKujiKiri:        affKujiKiri,
	engine.AFFCutthroat:       affCutthroat,
	engine.AFFFlying:          affFly,
	engine.AFFWerewolf:        affWerewolf,
	engine.AFFVampire:         affVampire,
	engine.AFFMount:           affMount,
	engine.AFFInvuln:          affInvuln,
	engine.AFFFlaming:         affFlaming,
	engine.AFFNothing:         affNothing,
	engine.AFFHaste:           affHaste,
	engine.AFFSlow:            affSlow,
	engine.AFFDream:           affDream,
	engine.AFFWaterBreathing:  affWaterBreathe,
	engine.AFFMetalskin:       affMetalskin,
	engine.AFFRobbed:          affRobbed,
}
