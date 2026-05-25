package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// handleCharInput processes character creation input from the client.
// Implements the nanny() flow from interpreter.c.
// Stages: color → sex → race → class → hometown → stats_roll
func (s *Session) handleCharInput(data json.RawMessage) error {
	var input CharInputData
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	switch s.charStage {
	case "color":
		switch input.Choice {
		case "Y", "y":
			s.charColor = true
			s.advanceCharStage("sex", "Select your sex (M/F):", map[string]string{"M": "Male", "F": "Female"})
		case "N", "n":
			s.charColor = false
			s.advanceCharStage("sex", "Select your sex (M/F):", map[string]string{"M": "Male", "F": "Female"})
		default:
			s.sendCharCreatePrompt("color", "Invalid choice. Do you want ANSI color? (Y/N):", map[string]string{"Y": "Yes", "N": "No"})
		}

	case "sex":
		switch input.Choice {
		case "M", "m":
			s.charSex = 0
			s.advanceCharStage("race", "Select your race:", s.getRaceOptions())
		case "F", "f":
			s.charSex = 1
			s.advanceCharStage("race", "Select your race:", s.getRaceOptions())
		default:
			s.sendCharCreatePrompt("sex", "Invalid choice. Select your sex (M/F):", map[string]string{"M": "Male", "F": "Female"})
		}

	case "race":
		if raceStr, ok := s.getRaceOptions()[input.Choice]; ok {
			if race, err := strconv.Atoi(input.Choice); err == nil {
				s.charRace = race
			}
			_ = raceStr
			s.advanceCharStage("class", "Select your class:", s.getClassOptions(s.charRace))
		} else {
			s.sendCharCreatePrompt("race", "Invalid race. Select your race:", s.getRaceOptions())
		}

	case "class":
		if _, ok := s.getClassOptions(s.charRace)[input.Choice]; ok {
			if classID, err := strconv.Atoi(input.Choice); err == nil {
				s.charClass = classID
			}
			// Roll initial stats for display
			s.charStats = game.RollRealAbils(s.charClass, s.charRace)
			s.advanceCharStage("hometown", "Choose your hometown:", map[string]string{
				"K": "Kir Drax'in — The Main City. New players should choose this.",
				"O": "Kir-Oshi — The Port City.",
				"A": "Alaozar — The Holy City.",
			})
		} else {
			s.sendCharCreatePrompt("class", "Invalid class. Select your class:", s.getClassOptions(s.charRace))
		}

	case "hometown":
		switch input.Choice {
		case "K", "k":
			s.charHometown = 1
			s.sendStatsRollPrompt()
		case "O", "o":
			s.charHometown = 2
			s.sendStatsRollPrompt()
		case "A", "a":
			s.charHometown = 3
			s.sendStatsRollPrompt()
		default:
			s.sendCharCreatePrompt("hometown", "Invalid choice. Choose your hometown:", map[string]string{
				"K": "Kir Drax'in — The Main City. New players should choose this.",
				"O": "Kir-Oshi — The Port City.",
				"A": "Alaozar — The Holy City.",
			})
		}

	case "stats_roll":
		switch input.Choice {
		case "Y", "y":
			if err := s.completeCharCreation(); err != nil {
				slog.Error("char creation failed", "error", err)
			}
		case "N", "n":
			// Reroll stats and stay at stats_roll stage
			s.charStats = game.RollRealAbils(s.charClass, s.charRace)
			s.sendStatsRollPrompt()
		default:
			s.sendStatsRollPrompt()
		}

	default:
		return fmt.Errorf("unexpected char creation stage: %s", s.charStage)
	}
	return nil
}

func getAbilName(score int) string {
	abilNames := []string{
		"terrible",      // 0
		"terrible",      // 1
		"awful",         // 2
		"awful",         // 3
		"bad",           // 4
		"bad",           // 5
		"poor",          // 6
		"poor",          // 7
		"below average", // 8
		"average",       // 9
		"average",       // 10
		"decent",        // 11
		"decent",        // 12
		"good",          // 13
		"good",          // 14
		"very good",     // 15
		"very good",     // 16
		"excellent",     // 17
		"excellent",     // 18
		"superior",      // 19
		"godlike",       // 20
		"godlike",       // 21
		"godlike",       // 22
		"godlike",       // 23
		"godlike",       // 24
		"godlike",       // 25
	}
	if score < 0 {
		return "terrible"
	}
	if score >= len(abilNames) {
		return "godlike"
	}
	return abilNames[score]
}

// sendStatsRollPrompt displays the current rolled stats and asks to keep or reroll.
func (s *Session) sendStatsRollPrompt() {
	stats := &CharStatsDisplay{
		Str: s.charStats.Str,
		Int: s.charStats.Int,
		Wis: s.charStats.Wis,
		Dex: s.charStats.Dex,
		Con: s.charStats.Con,
		Cha: s.charStats.Cha,
	}
	prompt := fmt.Sprintf(
		"Your ability scores:\r\n  Str: (%s)  Dex: (%s)  Int: (%s)\r\n  Wis: (%s)  Con: (%s)  Cha: (%s)\r\n\r\nPress Y to keep these stats, or N to reroll:",
		getAbilName(stats.Str), getAbilName(stats.Dex), getAbilName(stats.Int),
		getAbilName(stats.Wis), getAbilName(stats.Con), getAbilName(stats.Cha),
	)
	data := CharCreateData{
		Stage:   "stats_roll",
		Prompt:  prompt,
		Options: map[string]string{"Y": "Keep", "N": "Reroll"},
		Stats:   stats,
	}
	s.charStage = "stats_roll"
	msg, err := json.Marshal(ServerMessage{
		Type: MsgCharCreate,
		Data: data,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.send <- msg
}

// startCharCreation begins the character creation flow for a new player.
func (s *Session) startCharCreation(playerName string) {
	s.charCreating = true
	s.charName = playerName

	// Start with color selection — must set charStage so handleCharInput knows where we are
	s.charStage = "color"
	s.sendCharCreatePrompt("color", "Do you want ANSI color? (Y/N):", map[string]string{
		"Y": "Yes",
		"N": "No",
	})
}

// sendCharCreatePrompt sends a character creation prompt to the client.
func (s *Session) sendCharCreatePrompt(stage, prompt string, options map[string]string) {
	data := CharCreateData{
		Stage:   stage,
		Prompt:  prompt,
		Options: options,
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgCharCreate,
		Data: data,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}

	s.send <- msg
}

// completeCharCreation finalizes character creation and enters the world.
func (s *Session) completeCharCreation() error {
	// Create the player with collected attributes
	s.player = game.NewCharacter(0, s.charName, s.charClass, s.charRace)
	s.player.Stats = s.charStats

	// Set sex
	s.player.Sex = s.charSex

	// Set hometown
	s.player.Hometown = s.charHometown

	// Set hometown starting room
	// K=Kir Draxin/8004, O=Kir-Oshi/18201, A=Alaozar/21258
	// C source: config.c — mortal_start_room=8004, kiroshi_start_room=18201, alaozar_start_room=21258
	switch s.charHometown {
	case 1: // Kir Drax'in (main city, new players)
		s.player.RoomVNum = 8004
	case 2: // Kir-Oshi (port city)
		s.player.RoomVNum = 18201
	case 3: // Alaozar (holy city)
		s.player.RoomVNum = 21258
	default: // Mortal start
		s.player.RoomVNum = 8004
	}

	// Save to DB if available
	if s.manager.hasDB {
		if r, err := db.PlayerToRecord(s.player, nil); err == nil {
			// Apply the hashed password collected during login
			r.Password = s.charPassword
			if err := s.manager.db.CreatePlayer(r); err != nil {
				// Check for constraint violation (player already exists)
				if isUniqueConstraintError(err) {
					slog.Warn("duplicate character name, rejecting creation",
						"player", s.charName)
					return fmt.Errorf("a character named '%s' already exists", s.charName)
				}
				slog.Error("DB create error during char creation", "error", err)
				return fmt.Errorf("failed to save character: %w", err)
			} else {
				s.player.ID = r.ID
				// Give starting items
				s.manager.world.GiveStartingItems(s.player)
				game.GiveStartingSkills(s.player)
			}
		}
	} else {
		// Give starting items
		s.manager.world.GiveStartingItems(s.player)
		game.GiveStartingSkills(s.player)
	}

	// Register and add to world
	s.authenticated = true
	s.playerName = s.charName

	if err := s.manager.Register(s.charName, s); err != nil {
		return err
	}

	if err := s.manager.world.AddPlayer(s.player); err != nil {
		s.manager.Unregister(s.charName)
		return err
	}
	slog.Info("completeCharCreation: player added to world", "player", s.player.Name, "room", s.player.GetRoom())

	// Safety: catch any panic in the rest of char creation
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC in completeCharCreation", "recover", r, "player", s.player.Name)
		}
	}()

	// Clear char creation state
	s.charCreating = false
	s.charStage = ""
	s.charName = ""
	s.charPassword = ""
	s.charColor = false
	s.charSex = 0
	s.charRace = 0
	s.charClass = 0
	s.charHometown = 0
	s.charStats = game.CharStats{}
	slog.Info("completeCharCreation: state cleared")

	// Generate JWT token
	token, err := auth.GenerateJWT(s.player.Name, s.isAgent, s.agentKeyID, "")
	if err != nil {
		slog.Error("failed to generate JWT token", "error", err)
	}

	// Send welcome with token
	s.sendWelcome(token)

	// Mirror the agent initialization that handleLogin sends for returning players.
	// Without this, the agent harness never receives type:vars / memory_bootstrap /
	// memory_summary and stays in its initialization wait loop until it times out,
	// discarding all subsequent command responses.
	if s.isAgent {
		s.sendFullVarDump()
		s.SendMemoryBootstrap()
		s.SendMemorySummary()
	}

	// Broadcast arrival
	enterMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			Text: s.player.Name + " has arrived.",
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)

	return nil
}

// advanceCharStage moves to the next char creation stage.
func (s *Session) advanceCharStage(stage, prompt string, options map[string]string) {
	s.charStage = stage
	s.sendCharCreatePrompt(stage, prompt, options)
}

// getRaceOptions returns available races for character creation.
func (s *Session) getRaceOptions() map[string]string {
	return map[string]string{
		"0": "Human",
		"1": "Elf",
		"2": "Dwarf",
		"3": "Halfling",
		"4": "Minotaur",
		"5": "Rakshasa",
		"6": "Ssaur",
	}
}

// getClassOptions returns available classes for character creation, filtered by race.
// Matches valid_user_class_choice() from interpreter.c.
func (s *Session) getClassOptions(race int) map[string]string {
	// Base classes available to all races
	opts := map[string]string{
		"0": "Magic-user",
		"1": "Cleric",
		"2": "Thief",
		"3": "Warrior",
		"9": "Psionic",
	}
	// Ninja is human-only
	if race == game.RaceHuman {
		opts["8"] = "Ninja"
	}
	return opts
}

// isUniqueConstraintError checks if a DB error is a PostgreSQL unique constraint violation.
// C source: — pure Go helper, no C equivalent
func isUniqueConstraintError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}
