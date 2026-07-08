package agentcli

import "testing"

// TestFSMDecision_AutoAttacksFightingMobWhenPlayerIdle is a regression test for
// DP-755. The original code guarded the auto-attack loop with !isInCombat(state),
// which made the loop unreachable dead code (isInCombat is true whenever any mob
// is fighting). After the fix the loop is gated on the player's own Fighting
// flag: when the player is idle but a mob is fighting, the FSM should target it.
func TestFSMDecision_AutoAttacksFightingMobWhenPlayerIdle(t *testing.T) {
	state := &GameState{}
	state.Player.Health = 100
	state.Player.MaxHealth = 100
	state.Fighting = "" // player not engaged
	state.Room.Mobs = []Mob{
		{Name: "goblin", TargetString: "goblin.1001", Fighting: true},
	}

	resp := FSMDecision(state)
	if resp == nil {
		t.Fatal("expected auto-attack response, got nil")
	}
	if resp.ActionType != "hit" {
		t.Errorf("ActionType = %q, want %q", resp.ActionType, "hit")
	}
	if len(resp.Args) != 1 || resp.Args[0] != "goblin.1001" {
		t.Errorf("Args = %v, want [goblin.1001]", resp.Args)
	}
}

// TestFSMDecision_NoAttackWhenPlayerAlreadyEngaged confirms the FSM does not
// override when the player is already fighting — the LLM (or the auto_attack
// behavior) owns that case.
func TestFSMDecision_NoAttackWhenPlayerAlreadyEngaged(t *testing.T) {
	state := &GameState{}
	state.Player.Health = 100
	state.Player.MaxHealth = 100
	state.Fighting = "goblin" // player already engaged
	state.Room.Mobs = []Mob{
		{Name: "goblin", Fighting: true},
	}

	if resp := FSMDecision(state); resp != nil {
		t.Errorf("expected nil response when already fighting, got %+v", resp)
	}
}

// TestFSMDecision_TargetFallsBackToName when TargetString is empty.
func TestFSMDecision_TargetFallsBackToName(t *testing.T) {
	state := &GameState{}
	state.Player.Health = 100
	state.Player.MaxHealth = 100
	state.Room.Mobs = []Mob{
		{Name: "orc", Fighting: true}, // no TargetString
	}

	resp := FSMDecision(state)
	if resp == nil || len(resp.Args) != 1 || resp.Args[0] != "orc" {
		t.Fatalf("expected target to fall back to mob Name, got %+v", resp)
	}
}

// TestFSMDecision_FleesAtLowHP keeps the flee priority intact.
func TestFSMDecision_FleesAtLowHP(t *testing.T) {
	state := &GameState{}
	state.Player.Health = 10
	state.Player.MaxHealth = 100
	state.Room.Mobs = []Mob{{Name: "dragon", Fighting: true}}

	resp := FSMDecision(state)
	if resp == nil || resp.ActionType != "flee" {
		t.Fatalf("expected flee at low HP, got %+v", resp)
	}
}
