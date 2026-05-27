package session

import (
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// classifyCommand returns the command class for the given command string.
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

// determineOutcome classifies the result of a command based on pre/post state changes.
func determineOutcome(pre, post *playerState) string {
	// Room changed = movement
	if pre.room != post.room {
		return "movement"
	}

	// Health changed
	if pre.health != post.health {
		if post.health <= 0 {
			return "agent_died"
		}
		if post.health < pre.health {
			return "combat_hit_taken"
		}
		return "healed"
	}

	// Fighting changed
	if !pre.fighting && post.fighting {
		return "combat_started"
	}
	if pre.fighting && !post.fighting {
		return "combat_ended"
	}

	// Level changed
	if post.level > pre.level {
		return "level_up"
	}

	// Inventory changed
	if post.invCount != pre.invCount {
		if post.invCount > pre.invCount {
			return "item_acquired"
		}
		return "item_dropped"
	}

	// Position changed
	if pre.position != post.position {
		return "position_changed"
	}

	return "no_change"
}

// playerState is a lightweight snapshot of player state for decision capture.
type playerState struct {
	room      int
	health    int
	maxHealth int
	mana      int
	maxMana   int
	move      int
	maxMove   int
	level     int
	fighting  bool
	position  int
	invCount  int
}

// capturePlayerState snapshots the current player state.
func (s *Session) capturePlayerState() playerState {
	if s.player == nil {
		return playerState{}
	}
	return playerState{
		room:      s.player.GetRoom(),
		health:    s.player.GetHP(),
		maxHealth: s.player.GetMaxHP(),
		mana:      s.player.GetMana(),
		maxMana:   s.player.GetMaxMana(),
		move:      s.player.GetMove(),
		maxMove:   s.player.GetMaxMove(),
		level:     s.player.GetLevel(),
		fighting:  s.player.IsFighting(),
		position:  s.player.GetPosition(),
		invCount:  len(s.player.GetInventory()),
	}
}

// captureAndLog captures pre/post state and logs the decision.
// Called from handleCommand after ExecuteCommand returns.
func (s *Session) captureAndLog(cmdStr string, args []string, preState playerState, startTime time.Time, execErr error) {
	if s.manager.decisionLog == nil {
		return
	}

	postState := s.capturePlayerState()
	rawInput := cmdStr
	if len(args) > 0 {
		rawInput = cmdStr + " " + strings.Join(args, " ")
	}

	// Determine outcome category
	outcome := determineOutcome(&preState, &postState)

	// If command returned an error, override outcome
	errMsg := ""
	if execErr != nil {
		outcome = "error"
		errMsg = execErr.Error()
	}

	// Compute zone from world
	zone := 0
	if s.manager.world != nil {
		zone = s.manager.world.GetRoomZone(preState.room)
	}

	record := &db.DecisionRecord{
		SessionID:      s.sessionID(),
		PlayerName:     s.manager.decisionLog.HashPlayerName(s.playerName, s.isAgent),
		IsAgent:        s.isAgent,
		AgentHarness:   s.agentHarness,
		AgentModel:     s.agentModel,
		TurnNumber:     s.commandCount,
		SessionElapsed: time.Since(s.connectedAt).Seconds(),

		Command:      cmdStr,
		CommandClass: classifyCommand(cmdStr),
		Args:         args,
		RawInput:     rawInput,

		PreRoom:      preState.room,
		PreZone:      zone,
		PreHealth:    preState.health,
		PreMaxHealth: preState.maxHealth,
		PreMana:      preState.mana,
		PreMaxMana:   preState.maxMana,
		PreMove:      preState.move,
		PreMaxMove:   preState.maxMove,
		PreLevel:     preState.level,
		PreFighting:  preState.fighting,
		PrePosition:  preState.position,
		PreInvCount:  preState.invCount,

		PostRoom:      postState.room,
		PostZone:      zone,
		PostHealth:    postState.health,
		PostMaxHealth: postState.maxHealth,
		PostMana:      postState.mana,
		PostMaxMana:   postState.maxMana,
		PostMove:      postState.move,
		PostMaxMove:   postState.maxMove,
		PostLevel:     postState.level,
		PostFighting:  postState.fighting,
		PostPosition:  postState.position,
		PostInvCount:  postState.invCount,

		OutcomeCategory: outcome,
		OutcomeText:     errMsg,
		DurationMs:      float64(time.Since(startTime).Microseconds()) / 1000.0,
	}

	s.manager.decisionLog.RecordDecision(record)
	s.commandCount++
}
