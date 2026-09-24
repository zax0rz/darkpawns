package session

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newClockControlSession(t *testing.T) (*Session, *int) {
	t.Helper()
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	manager := NewManager(world, nil)
	t.Cleanup(manager.Stop)
	pumped := new(int)
	manager.SetPulsePump(func(n int) error {
		*pumped += n
		return nil
	})
	return &Session{manager: manager}, pumped
}

func TestHandleClockControlIsDPClockOnlyAndDrawNeutral(t *testing.T) {
	s, pumped := newClockControlSession(t)

	if s.HandleClockControl("~dpclock pulse 40") {
		t.Fatal("control intercepted with DP_CLOCK unset")
	}
	if *pumped != 0 {
		t.Fatalf("pumped %d pulses with DP_CLOCK unset", *pumped)
	}

	t.Setenv("DP_CLOCK", "1")
	if !s.HandleClockControl("~dpclock pulse 40") {
		t.Fatal("valid control was not intercepted")
	}
	if *pumped != 40 {
		t.Fatalf("pumped %d pulses, want 40", *pumped)
	}
	if s.HandleClockControl("~dpclock pulse 0") {
		t.Fatal("invalid pulse count was intercepted")
	}
}

// The browser client sends every typed line as a command message with no raw
// line, before and after login alike; the control must still reach the pump.
func TestClockControlArrivesAsWebSocketCommand(t *testing.T) {
	s, pumped := newClockControlSession(t)
	t.Setenv("DP_CLOCK", "1")

	data, err := json.Marshal(CommandData{Command: "~dpclock pulse 7"})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(ClientMessage{Type: MsgCommand, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleMessage(frame); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	if *pumped != 7 {
		t.Fatalf("pumped %d pulses, want 7", *pumped)
	}
}
