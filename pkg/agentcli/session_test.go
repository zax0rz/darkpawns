package agentcli

import (
	"testing"
)

func TestSessionSummaryIncludesAgentID(t *testing.T) {
	logger := NewSessionLogger()
	logger.Log(LogEntry{
		RoomVnum:   3001,
		HP:         10,
		MaxHP:      10,
		AgentLevel: 1,
		Action:     "look",
	})

	summary := logger.Finalize("TestAgent", "")
	if summary.AgentID != "TestAgent" {
		t.Errorf("expected AgentID to be %q, got %q", "TestAgent", summary.AgentID)
	}
	if summary.Turns != 1 {
		t.Errorf("expected Turns to be 1, got %d", summary.Turns)
	}
}
