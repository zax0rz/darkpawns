package session

import "testing"

func TestClassifyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		// movement
		{cmd: "north", want: "movement"},
		{cmd: "south", want: "movement"},
		{cmd: "east", want: "movement"},
		{cmd: "west", want: "movement"},
		{cmd: "up", want: "movement"},
		{cmd: "down", want: "movement"},
		{cmd: "n", want: "movement"},
		{cmd: "s", want: "movement"},
		{cmd: "e", want: "movement"},
		{cmd: "w", want: "movement"},
		{cmd: "u", want: "movement"},
		{cmd: "d", want: "movement"},
		{cmd: "follow", want: "movement"},
		{cmd: "order", want: "movement"},
		{cmd: "group", want: "movement"},
		{cmd: "dismiss", want: "movement"},
		{cmd: "leave", want: "movement"},
		{cmd: "stand", want: "movement"},

		// info
		{cmd: "look", want: "info"},
		{cmd: "examine", want: "info"},
		{cmd: "scan", want: "info"},
		{cmd: "who", want: "info"},
		{cmd: "where", want: "info"},
		{cmd: "score", want: "info"},
		{cmd: "inventory", want: "info"},
		{cmd: "skills", want: "info"},
		{cmd: "help", want: "info"},

		// inventory
		{cmd: "buy", want: "inventory"},
		{cmd: "sell", want: "inventory"},
		{cmd: "get", want: "inventory"},
		{cmd: "drop", want: "inventory"},
		{cmd: "wear", want: "inventory"},
		{cmd: "remove", want: "inventory"},

		// combat
		{cmd: "kill", want: "combat"},
		{cmd: "hit", want: "combat"},
		{cmd: "flee", want: "combat"},
		{cmd: "consider", want: "combat"},
		{cmd: "backstab", want: "combat"},

		// social
		{cmd: "say", want: "social"},
		{cmd: "tell", want: "social"},
		{cmd: "shout", want: "social"},
		{cmd: "reply", want: "social"},

		// magic
		{cmd: "cast", want: "magic"},
		{cmd: "activate", want: "magic"},
		{cmd: "recall", want: "magic"},

		// system
		{cmd: "quit", want: "system"},
		{cmd: "save", want: "system"},
		{cmd: "password", want: "system"},
		{cmd: "toggle", want: "system"},

		// other
		{cmd: "", want: "other"},
		{cmd: "xyzzy", want: "other"},
		{cmd: "sit", want: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := classifyCommand(tt.cmd); got != tt.want {
				t.Errorf("classifyCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestClassifyCommand_CaseSensitive(t *testing.T) {
	// The function does a direct switch on the exact string, so uppercase
	// should fall through to "other".
	if got := classifyCommand("North"); got != "other" {
		t.Errorf("classifyCommand(\"North\") = %q, want \"other\"", got)
	}
	if got := classifyCommand("LOOK"); got != "other" {
		t.Errorf("classifyCommand(\"LOOK\") = %q, want \"other\"", got)
	}
}

func TestDetermineOutcome(t *testing.T) {
	base := &playerState{room: 1, health: 100, fighting: false, level: 5, invCount: 3, position: 8}

	tests := []struct {
		name string
		pre  *playerState
		post *playerState
		want string
	}{
		{
			name: "no change",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: false, level: 5, invCount: 3, position: 8},
			want: "no_change",
		},
		{
			name: "movement",
			pre:  base,
			post: &playerState{room: 2, health: 100, fighting: false, level: 5, invCount: 3, position: 8},
			want: "movement",
		},
		{
			name: "agent died",
			pre:  base,
			post: &playerState{room: 1, health: 0, fighting: false, level: 5, invCount: 3, position: 8},
			want: "agent_died",
		},
		{
			name: "combat hit taken",
			pre:  base,
			post: &playerState{room: 1, health: 70, fighting: false, level: 5, invCount: 3, position: 8},
			want: "combat_hit_taken",
		},
		{
			name: "healed",
			pre:  &playerState{room: 1, health: 50, maxHealth: 100},
			post: &playerState{room: 1, health: 80, maxHealth: 100},
			want: "healed",
		},
		{
			name: "combat started",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: true, level: 5, invCount: 3, position: 8},
			want: "combat_started",
		},
		{
			name: "combat ended",
			pre:  &playerState{room: 1, health: 100, fighting: true, level: 5},
			post: &playerState{room: 1, health: 100, fighting: false, level: 5},
			want: "combat_ended",
		},
		{
			name: "level up",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: false, level: 6, invCount: 3, position: 8},
			want: "level_up",
		},
		{
			name: "item acquired",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: false, level: 5, invCount: 4, position: 8},
			want: "item_acquired",
		},
		{
			name: "item dropped",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: false, level: 5, invCount: 2, position: 8},
			want: "item_dropped",
		},
		{
			name: "position changed",
			pre:  base,
			post: &playerState{room: 1, health: 100, fighting: false, level: 5, invCount: 3, position: 4},
			want: "position_changed",
		},
		{
			name: "movement takes priority over position",
			pre:  base,
			post: &playerState{room: 2, health: 100, fighting: false, level: 5, invCount: 3, position: 4},
			want: "movement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineOutcome(tt.pre, tt.post); got != tt.want {
				t.Errorf("determineOutcome = %q, want %q", got, tt.want)
			}
		})
	}
}
