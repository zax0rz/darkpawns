package session

// classifyCommand groups commands for the command metrics (metrics.CommandProcessed).
func classifyCommand(cmd string) string {
	switch cmd {
	case "north", "south", "east", "west", "up", "down", "n", "s", "e", "w", "u", "d":
		return "movement"
	case "look", "examine", "scan", "who", "where", "score", "inventory", "equipped", "skills", "spells", "areas", "help":
		return "info"
	case "buy", "sell", "value", "list", "get", "take", "drop", "put", "give", "wear", "wield", "hold", "remove", "eat", "drink", "use", "quaff", "recite", "zap":
		return "inventory"
	case "kill", "hit", "attack", "backstab", "flee", "kick", "punch", "bash", "rescue", "guard", "disarm", "trip", "circle", "consider", "assess":
		return "combat"
	case "say", "tell", "whisper", "yell", "shout", "gossip", "auction", "clan", "reply", "ask", "talk":
		return "social"
	case "cast", "activate", "recall":
		return "magic"
	case "rent", "quit", "save", "password", "title", "description", "color", "prompt", "alias", "unalias", "toggle", "wimpy", "compact", "brief", "map", "notell", "noshout", "novice":
		return "system"
	case "follow", "order", "group", "dismiss", "leave", "stand":
		return "movement"
	default:
		return "other"
	}
}
