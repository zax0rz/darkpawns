package agentcli

// FSMDecision implements combat survival logic that NEVER delegates to the LLM.
// Returns an action to override the LLM with, or nil to let the LLM decide.
func FSMDecision(state *GameState) *LLMResponse {
	if state == nil {
		return nil
	}

	hp := state.Player.Health
	maxHP := state.Player.MaxHealth

	// Flee at low HP.
	if maxHP > 0 && hp*100/maxHP < 25 {
		return &LLMResponse{ActionType: "flee"}
	}

	// Note: there used to be an "attack aggressive mob" block here gated on
	// !isInCombat(state). It was dead code: isInCombat returns true if any mob
	// has Fighting==true, so the inner `if mob.Fighting` loop could never fire.
	// Auto-attacking mobs that are fighting the player is handled by the
	// auto_attack behavior in behavior.go (DP-755).

	return nil
}
