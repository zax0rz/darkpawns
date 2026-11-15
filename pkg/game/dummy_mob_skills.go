package game

// DummyMobSkill reads GET_SKILL for an NPC. C points every NPC at the same
// zero-initialized dummy_mob record (src/db.c:104,1281; src/utils.h:343).
func (w *World) DummyMobSkill(skillNum int) int {
	if w == nil {
		return 0
	}
	w.npcDummySkillMu.RLock()
	defer w.npcDummySkillMu.RUnlock()
	if value, ok := w.npcDummySkillOverrides[skillNum]; ok {
		return value
	}
	return w.npcDummySkillDefault
}

// SetDummyMobSkill mirrors SET_SKILL against the shared dummy_mob record.
func (w *World) SetDummyMobSkill(skillNum, value int) {
	if w == nil {
		return
	}
	w.npcDummySkillMu.Lock()
	defer w.npcDummySkillMu.Unlock()
	if value == w.npcDummySkillDefault {
		delete(w.npcDummySkillOverrides, skillNum)
		return
	}
	if w.npcDummySkillOverrides == nil {
		w.npcDummySkillOverrides = make(map[int]int)
	}
	w.npcDummySkillOverrides[skillNum] = value
}

// RestoreDummyMobSkills mirrors do_restore's loop over every skill number
// when an immortal-level target is restored by a greater god.
func (w *World) RestoreDummyMobSkills() {
	if w == nil {
		return
	}
	w.npcDummySkillMu.Lock()
	defer w.npcDummySkillMu.Unlock()
	w.npcDummySkillDefault = 100
	w.npcDummySkillOverrides = nil
}
