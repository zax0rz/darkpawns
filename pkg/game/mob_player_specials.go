package game

import "sync"

// C points every mobile at the one file-scope dummy_mob player-specials
// record (src/db.c:104, 1101, 1281). The condition slots therefore belong to
// the shared record, not to an individual MobInstance. load_mobiles initializes
// each slot to -1; because all prototypes share the pointer, the runtime value
// is the same -1 sentinel for every NPC.
var npcDummyConditions = struct {
	sync.RWMutex
	values [3]int
}{values: [3]int{-1, -1, -1}}

// GetCondition mirrors GET_COND for an NPC, which reads dummy_mob's shared
// player-specials conditions (src/utils.h:328).
func (m *MobInstance) GetCondition(condition int) int {
	if condition < 0 || condition >= len(npcDummyConditions.values) {
		return 0
	}
	npcDummyConditions.RLock()
	defer npcDummyConditions.RUnlock()
	return npcDummyConditions.values[condition]
}

// SetCondition mirrors C's direct GET_COND lvalue writes. This intentionally
// writes shared state: every MobInstance reads the same dummy_mob slots.
func (m *MobInstance) SetCondition(condition, value int) {
	if condition < 0 || condition >= len(npcDummyConditions.values) {
		return
	}
	npcDummyConditions.Lock()
	npcDummyConditions.values[condition] = value
	npcDummyConditions.Unlock()
}

func gainMobDummyCondition(condition, amount int) (current int, intoxicated bool, changed bool) {
	if condition < 0 || condition >= len(npcDummyConditions.values) {
		return 0, false, false
	}
	npcDummyConditions.Lock()
	defer npcDummyConditions.Unlock()
	current = npcDummyConditions.values[condition]
	if current == -1 {
		return current, false, false
	}
	intoxicated = npcDummyConditions.values[CondDrunk] > 0
	current = max(0, min(48, current+amount))
	npcDummyConditions.values[condition] = current
	return current, intoxicated, true
}

func addMobDummyConditionIfInRange(condition, lowerBound, upperBound, amount int) bool {
	if condition < 0 || condition >= len(npcDummyConditions.values) {
		return false
	}
	npcDummyConditions.Lock()
	defer npcDummyConditions.Unlock()
	current := npcDummyConditions.values[condition]
	if current < lowerBound || current >= upperBound {
		return false
	}
	npcDummyConditions.values[condition] = current + amount
	return true
}
