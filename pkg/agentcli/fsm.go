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

	// In combat but not fighting? Attack.
	if !isInCombat(state) && len(state.Room.Mobs) > 0 {
		for _, mob := range state.Room.Mobs {
			if mob.Fighting {
				return &LLMResponse{
					ActionType: "hit",
					Args:       []string{mob.TargetString},
				}
			}
		}
	}

	return nil
}

func isInCombat(state *GameState) bool {
	if state.Fighting != "" {
		return true
	}
	for _, mob := range state.Room.Mobs {
		if mob.Fighting {
			return true
		}
	}
	return false
}
