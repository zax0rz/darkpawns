package game

import (
	"log/slog"
	"time"
)

// MobDreamAdapter gives point_update's dream() call a mobile context
// (src/limits.c:489-491). C reads the shared dummy_mob player_specials, so
// GET_LAST_DEATH is 0 and the level switch runs. send_to_char is a no-op for a
// descriptorless mobile; a switched body's descriptor receives those lines.
// The act() calls are C's FALSE hide-invisible flag, so the room sees them.
type MobDreamAdapter struct {
	m *MobInstance
	w *World
}

func (a *MobDreamAdapter) GetLevel() int        { return a.m.GetLevel() }
func (a *MobDreamAdapter) GetLastDeath() int64  { return a.w.NPCDummyLastDeath() }
func (a *MobDreamAdapter) SetLastDeath(t int64) { a.w.SetNPCDummyLastDeath(t) }
func (a *MobDreamAdapter) HasAffect(bit int) bool {
	return bit >= 0 && bit < 64 && a.m.IsAffected(bit)
}

func (a *MobDreamAdapter) RemoveAffect(bit int) {
	if bit >= 0 && bit < 64 {
		a.m.RemoveAffected(bit)
	}
}

func (a *MobDreamAdapter) SendToChar(msg string) { a.m.SendMessage(msg) }
func (a *MobDreamAdapter) SendToRoom(msg string) {
	Act(a.w, false, a.m, nil, nil, nil, msg, "", ToRoom)
}

func (a *MobDreamAdapter) WakeUp() { a.w.mobWake(a.m) }

func (a *MobDreamAdapter) MoveToRoom(roomVNum int) {
	if err := a.w.MobTransfer(a.m, roomVNum); err != nil {
		slog.Warn("MobTransfer failed in dream", "mob", a.m.GetName(), "room", roomVNum, "error", err)
	}
}

func (a *MobDreamAdapter) CurrentTime() int64 { return time.Now().Unix() }

// mobWake is do_wake's self path for a mobile (src/act.movement.c:869-879).
// AFF_SLEEP refuses; otherwise the body sits up and the room sees the act.
func (w *World) mobWake(m *MobInstance) {
	if m.IsAffected(AffSleep) {
		m.SendMessage("You can't wake up!\r\n")
		Act(w, true, m, nil, nil, nil, "$n tosses and turns uncomfortably.", "", ToRoom)
		return
	}
	m.SendMessage("You awaken, and sit up.\r\n")
	Act(w, true, m, nil, nil, nil, "$n awakens.", "", ToRoom)
	m.SetPosition(PosSitting)
}
