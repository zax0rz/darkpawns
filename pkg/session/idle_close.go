package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// closeIdleDescriptor ports limits.c:442-444 / comm.c:2092-2148. Called
// after the room transfer with no world/player lock. Lifecycle ownership lasts
// only through the real close, and is released before RentOut/extraction.
func (m *Manager) closeIdleDescriptor(p *game.Player) {
	m.playerLifecycleMu.Lock()
	defer m.playerLifecycleMu.Unlock()
	s := m.attachedBody(p)
	if s == nil {
		return
	} // original/linkdead bodies have no descriptor to close
	if s.idleTransportClosed.Swap(true) {
		return
	}
	s.discardHeartbeatOutput()
	s.Close()
	if s.transportDone != nil {
		s.DetachTransport()
	}
	s.cancelTextEdit()
	s.cancelRoomEdit()
	s.cancelMedit()
	s.cancelOedit()
	s.cancelSedit()
	s.saveCharacter("idle close", game.LoadRoomNowhere)
	p.SetLinkless(true)
	game.Act(m.world, true, p, nil, nil, nil, "$n has lost $s link.", "", game.ToRoom)
	game.MudLog(fmt.Sprintf("Closing link to: %s.", p.GetName()), game.MudlogNormal, max(game.LVL_IMMORT, p.GetInvisLevel()), true)
	s.leaveBroadcastHandled = true
	if s.activePCSwitch() {
		m.detachPCSwitch(s, false)
	}
}
