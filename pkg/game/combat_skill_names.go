package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// combatSkillName maps C SKILL_* numbers used by pkg/combat callbacks to the
// game-layer skill keys stored on Player.SkillManager.
func combatSkillName(skillNum int) string {
	switch skillNum {
	case combat.SKILL_RETREAT:
		return SkillRetreat
	case combat.SKILL_ESCAPE:
		return SkillEscape
	case combat.SKILL_PARRY:
		return SkillParry
	default:
		return ""
	}
}
