package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// combatCanSee mirrors CAN_SEE(observer, subject) for damage()'s jail-guard
// branch. Unlike the generic character visibility helper, this includes the
// observer room's light state and deliberately ignores AFF_HIDE, matching
// src/utils.h:515-530.
func (w *World) combatCanSee(observer, subject combat.Combatant) bool {
	if w == nil || observer == nil || subject == nil {
		return false
	}
	if observer == subject {
		return true
	}

	observerState, ok := observer.(visibilitySubject)
	if !ok {
		return false
	}
	subjectState, ok := subject.(visibilitySubject)
	if !ok {
		return false
	}

	// GET_REAL_LEVEL is checked before HOLYLIGHT in C's CAN_SEE macro.
	if invisLevel, ok := subject.(invisLevelSubject); ok && observerState.GetLevel() < invisLevel.GetInvisLevel() {
		return false
	}
	lightOK := !observerState.IsAffected(affBlind) &&
		(!w.IsRoomDark(observerState.GetRoom()) || observerState.IsAffected(affInfravision))
	if lightOK && (!subjectState.IsAffected(affInvisible) || observerState.IsAffected(affDetectInvisible)) {
		return true
	}
	if holyLight, ok := observer.(holyLightSubject); ok {
		return holyLight.GetHolyLight()
	}
	return false
}
