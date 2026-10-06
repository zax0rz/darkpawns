package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/metrics"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// ApplyRangedProjectileDamage is do_shoot's direct HP/update_pos/die boundary.
// It deliberately bypasses damage's protection, messages and killer bookkeeping.
// C: src/act.offensive.c:938-946,967-973; src/fight.c:186-203,625-631.
func (w *World) ApplyRangedProjectileDamage(victim combat.Combatant, damage int) bool {
	switch v := victim.(type) {
	case *Player:
		if damage > 0 {
			metrics.DamageTaken("player", damage)
		}
		v.mu.Lock()
		v.Health -= damage
		v.mu.Unlock()
	case *MobInstance:
		if damage > 0 {
			metrics.DamageTaken("mob", damage)
		}
		v.mu.Lock()
		v.CurrentHP -= damage
		v.mu.Unlock()
	}
	victim.SetPosition(combat.GetPositionFromHP(victim.GetHP(), victim.GetPosition()))
	if victim.GetPosition() != combat.PosDead {
		return false
	}
	switch v := victim.(type) {
	case *Player:
		// gain_exp's negative branch only: same mortal gate, cap and floor.
		// Keeping the mutation under the body lock avoids GainExp's unlocked field writes.
		v.mu.Lock()
		if v.Level >= 1 && v.Level < LVL_IMMORT {
			v.Exp = max(0, v.Exp-min(v.Exp/3, maxExpLoss))
		}
		v.mu.Unlock()
	case *MobInstance:
		v.mu.Lock()
		exp := 0
		if v.Runtime.ExpOverride != nil {
			exp = *v.Runtime.ExpOverride
		} else if proto := v.Proto(); proto != nil {
			exp = proto.Exp
		}
		exp -= exp / 3
		v.Runtime.ExpOverride = &exp
		v.mu.Unlock()
	}
	w.RawKillCombatant(victim, combat.TYPE_UNDEFINED)
	return true
}

// TransferRangedVictim performs bare char_from_room/char_to_room for an
// unfighting shoot target. Unlike wizard transfers, it preserves posture.
// C: src/handler.c:504-543; src/act.offensive.c:948-950.
func (w *World) TransferRangedVictim(mob *MobInstance, to int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.activeMobs[mob.ID] != mob {
		return fmt.Errorf("ranged victim is no longer live")
	}
	if w.rooms[to] == nil {
		return fmt.Errorf("ranged destination does not exist")
	}
	// The command's target-fighting refusal guarantees no char_from_room fight
	// teardown. Do not apply wizard-transfer's unconditional stop/stand behavior.
	mob.moveRoomLocked(w, to)
	mob.mu.Lock()
	w.nextRoomEntrySequence++
	mob.RoomEntrySequence = w.nextRoomEntrySequence
	mob.mu.Unlock()
	// char_from_room calls circle_check at NOWHERE (no effect); char_to_room
	// decrements the first destination circle (src/handler.c:527,553;
	// src/utils.c:877-893; src/spells.h:37).
	for _, obj := range w.roomItems[to] {
		if obj.GetVNum() == spells.CocVnum {
			obj.SetTimer(obj.GetTimer() - 1)
			break
		}
	}
	return nil
}
