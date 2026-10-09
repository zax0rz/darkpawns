package combat

// DamageBeforeMessage ports src/fight.c:1486-1520. The caller has subtracted
// the adjusted damage; this block must finish before any skill/weapon message.
// True means ROOM_NEUTRAL rescued the victim and damage() must return FALSE.
func DamageBeforeMessage(ch, victim Combatant, dam int, cb *GameCallbacks) bool {
	if ch != nil && ch != victim && !ch.IsNPC() && ch.GetLevel() < 2 && cb != nil && cb.DamageNewbieExp != nil {
		cb.DamageNewbieExp(ch, victim.GetLevel()*dam)
	}
	victim.SetPosition(GetPositionFromHP(victim.GetHP(), victim.GetPosition()))
	if victim.GetPosition() > PosStunned {
		return false
	}
	stop := func(body Combatant) {
		if cb != nil && cb.StopFighting != nil {
			cb.StopFighting(body)
		} else {
			body.StopFighting()
			body.SetPosition(GetPositionFromHP(body.GetHP(), PosStanding))
		}
	}
	if ch != nil && ch.IsNPC() && !victim.IsNPC() && victim.GetLevel() <= 5 {
		stop(ch)
	}
	if victim.IsNPC() || cb == nil || cb.HasRoomFlag == nil || !cb.HasRoomFlag(victim.GetRoom(), "ROOM_NEUTRAL") {
		return false
	}
	if opponent := victim.GetFightingBody(); opponent != nil && opponent.GetFightingBody() == victim {
		stop(opponent)
	}
	victim.Heal(1 - victim.GetHP())
	stop(victim)
	if cb.NeutralRescue != nil {
		cb.NeutralRescue(ch, victim)
	}
	return true
}
