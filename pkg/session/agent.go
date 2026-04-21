package session

import "encoding/json"

// sendFullVarDump sends all agent variables to the session.
// TODO: Phase 4.2 — Agent #2 fills in real variable values.
// For now this sends an empty vars map so agents know auth succeeded.
func (s *Session) sendFullVarDump() {
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgVars,
		Data: map[string]interface{}{},
	})
	s.send <- msg
}
