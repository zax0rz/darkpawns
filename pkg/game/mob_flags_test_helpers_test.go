package game

import "github.com/zax0rz/darkpawns/pkg/parser"

// setProtoWithFlags swaps a test mobile's prototype and copies its action
// flags into the instance, as read_mobile does at load: MOB_FLAGGED reads
// the instance's bitmask, not the prototype's.
func setProtoWithFlags(m *MobInstance, p *parser.Mob) {
	m.SetProto(p)
	m.mu.Lock()
	m.Flags = actionFlagBits(p.ActionFlags)
	m.mu.Unlock()
}
