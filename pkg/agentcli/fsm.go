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

	// If the player is not already engaged but a mob in the room is fighting,
	// auto-attack the first such mob. The previous guard (!isInCombat(state))
	// conflated the player's combat state with mob states, which made this loop
	// unreachable dead code: isInCombat returns true if any mob has
	// Fighting==true, so the inner `if mob.Fighting` could never fire. The
	// correct condition is the player's own Fighting flag (DP-755).
	if state.Fighting == "" {
		for _, mob := range state.Room.Mobs {
			if mob.Fighting {
				target := mob.TargetString
				if target == "" {
					target = mob.Name
				}
				return &LLMResponse{
					ActionType: "hit",
					Args:       []string{target},
				}
			}
		}
	}

	return nil
}
