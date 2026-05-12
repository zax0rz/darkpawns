package agentcli

import "fmt"

// SystemPrompt is the agent's identity and output format instructions.
// Injected at the start of every session.
const SystemPrompt = `You are connected to Dark Pawns, a persistent MUD (Multi-User Dungeon).
You receive structured game state and must respond with exactly one action per turn.

## Output format

Line 1: JSON action (REQUIRED)
Line 2: In-game speech (OPTIONAL — plain text, omit if nothing to add)
Remaining: Terminal commentary (OPTIONAL)

Valid JSON actions:
  {"command": "hit", "args": ["target_name"]}
  {"command": "north"}
  {"command": "south"}
  {"command": "east"}
  {"command": "west"}
  {"command": "flee"}
  {"command": "get", "args": ["item_name"]}
  {"command": "say", "args": ["message"]}
  {"command": "look"}
  {"command": "kill", "args": ["target_name"]}
  {"command": "cast", "args": ["spell_name"]}
  {"command": "open", "args": ["direction"]}
  {"command": "close", "args": ["direction"]}

One action per turn. No filler. No preamble.`

// BuildPrompt constructs the user-facing state prompt for the LLM.
func BuildPrompt(state *GameState) string {
	roomDesc := fmt.Sprintf("Room: %s (vnum %d)", state.Room.Name, state.Room.Vnum)

	mobs := "Mobs here:"
	if len(state.Room.Mobs) == 0 {
		mobs += " none"
	} else {
		for _, m := range state.Room.Mobs {
			status := ""
			if m.Fighting {
				status = " [fighting]"
			}
			mobs += fmt.Sprintf("\n  %s %s (hp:%d%%)%s",
				m.TargetString, m.Name, m.HealthPct, status)
		}
	}

	exits := fmt.Sprintf("Exits: %v", state.Room.Exits)

	hp := state.Player.Health
	maxHP := state.Player.MaxHealth
	health := fmt.Sprintf("HP: %d/%d", hp, maxHP)

	fighting := ""
	if state.Fighting != "" {
		fighting = fmt.Sprintf("\nFighting: %s", state.Fighting)
	}

	return fmt.Sprintf("%s\n\n%s\n%s\n%s%s\n\nWhat do you do?",
		roomDesc, mobs, exits, health, fighting)
}
