package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestForceTargetsUsesDescriptorOrderForAllAndRoomOrderForRoom(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Start", Zone: 1},
		{VNum: 1002, Name: "Target", Zone: 1},
	}}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mgr := NewManager(w, nil)

	god := game.NewPlayer(1, "God", 1002)
	god.Level = LVL_IMPL
	alpha := game.NewPlayer(2, "Alpha", 1001)
	alpha.Level = 1
	beta := game.NewPlayer(3, "Beta", 1001)
	beta.Level = 1
	for _, p := range []*game.Player{god, alpha, beta} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", p.Name, err)
		}
	}

	// Move Beta into room 1002 first, then Alpha, so world[1002].people order
	// (most recently arrived first) is Alpha, Beta — opposite to descriptor_list
	// order (newest connectionNumber first: Beta, Alpha).
	if err := w.PlayerTransfer(beta, 1002); err != nil {
		t.Fatalf("PlayerTransfer beta: %v", err)
	}
	if err := w.PlayerTransfer(alpha, 1002); err != nil {
		t.Fatalf("PlayerTransfer alpha: %v", err)
	}

	godSess := &Session{manager: mgr, player: god, playerName: god.Name, authenticated: true, connectionNumber: 1}
	alphaSess := &Session{manager: mgr, player: alpha, playerName: alpha.Name, authenticated: true, connectionNumber: 2}
	betaSess := &Session{manager: mgr, player: beta, playerName: beta.Name, authenticated: true, connectionNumber: 3}

	mgr.mu.Lock()
	mgr.sessions[god.Name] = godSess
	mgr.sessions[alpha.Name] = alphaSess
	mgr.sessions[beta.Name] = betaSess
	mgr.mu.Unlock()

	for range 20 {
		allTargets := forceTargets(godSess, false)
		if len(allTargets) != 2 || allTargets[0].playerName != "Beta" || allTargets[1].playerName != "Alpha" {
			t.Fatalf("force all order = %v, want [Beta Alpha]", sessionNames(allTargets))
		}
		roomTargets := forceTargets(godSess, true)
		if len(roomTargets) != 2 || roomTargets[0].playerName != "Alpha" || roomTargets[1].playerName != "Beta" {
			t.Fatalf("force room order = %v, want [Alpha Beta]", sessionNames(roomTargets))
		}
	}
}

func sessionNames(sessions []*Session) []string {
	out := make([]string, len(sessions))
	for i, s := range sessions {
		out[i] = s.playerName
	}
	return out
}
