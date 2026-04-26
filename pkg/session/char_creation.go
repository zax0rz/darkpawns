package session

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// handleCharInput processes character creation input from the client.
// Implements the nanny() flow from interpreter.c.
func (s *Session) handleCharInput(data json.RawMessage) error {
	var input CharInputData
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	switch s.charStage {
	case "sex":
		switch input.Choice {
		case "M", "m":
			s.charSex = 0
			s.advanceCharStage("race", "Select your race:", nil)
		case "F", "f":
			s.charSex = 1
			s.advanceCharStage("race", "Select your race:", nil)
		default:
			s.sendCharCreatePrompt("sex", "Invalid choice. Select your sex (M/F):", map[string]string{"M": "Male", "F": "Female"})
		}
	case "race":
		if race, ok := s.getRaceOptions()[input.Choice]; ok {
			s.charRace = race
			s.advanceCharStage("class", "Select your class:", nil)
		} else {
			s.sendCharCreatePrompt("race", "Invalid race. Select your race:", nil)
		}
	case "class":
		if classID, ok := s.getClassOptions()[input.Choice]; ok {
			s.charClass = classID
			s.advanceCharStage("confirm", fmt.Sprintf("Create %s? (Y/N)", s.charName), nil)
		} else {
			s.sendCharCreatePrompt("class", "Invalid class. Select your class:", nil)
		}
	case "confirm":
		switch input.Choice {
		case "Y", "y":
			if err := s.completeCharCreation(); err != nil {
				slog.Error("char creation failed", "error", err)
			}
		case "N", "n":
			s.charCreating = false
			s.charStage = ""
			s.SendMessage("Character creation cancelled.\r\n")
		default:
			s.sendCharCreatePrompt("confirm", fmt.Sprintf("Create %s? (Y/N)", s.charName), nil)
		}
	default:
		return fmt.Errorf("unexpected char creation stage: %s", s.charStage)
	}
	return nil
}

// startCharCreation begins the character creation flow for a new player.
// Called when a player name is not found in DB and new_char is not set.
func (s *Session) startCharCreation(playerName string) {
	s.charCreating = true
	s.charName = playerName

	// Start with sex selection
	s.sendCharCreatePrompt("sex", "Select your sex (M/F):", map[string]string{
		"M": "Male",
		"F": "Female",
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

// sendCharCreateStats sends rolled stats to the client for confirmation.
func (s *Session) sendCharCreateStats(stats game.CharStats) {
	display := CharStatsDisplay{
		Str: stats.Str,
		Int: stats.Int,
		Wis: stats.Wis,
		Dex: stats.Dex,
		Con: stats.Con,
		Cha: stats.Cha,
	}

	data := CharCreateData{
		Stage: "rollstats",
		Prompt: "Your rolled stats:\n" +
			fmt.Sprintf("STR: %d/%d  INT: %d  WIS: %d\n", stats.Str, stats.StrAdd, stats.Int, stats.Wis) +
			fmt.Sprintf("DEX: %d  CON: %d  CHA: %d\n", stats.Dex, stats.Con, stats.Cha) +
			"Accept these stats? (Y/N)",
		Stats: &display,
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

	// Set hometown starting room — C: interpreter.c assigns start rooms per hometown
	// MortalStartRoom=8004, KiroshiStartRoom=18201, AlaozarStartRoom=21258
	switch s.charHometown {
	case 1: // Kiroshi
		s.player.RoomVNum = 18201
	case 2: // Alaozar
		s.player.RoomVNum = 21258
	default: // Mortal
		s.player.RoomVNum = 8004
	}

	// Save to DB if available
	if s.manager.hasDB {
		if r, err := db.PlayerToRecord(s.player, nil); err == nil {
			if err := s.manager.db.CreatePlayer(r); err != nil {
				slog.Error("DB create error during char creation", "error", err)
			} else {
				s.player.ID = r.ID
				// Give starting items
				s.manager.world.GiveStartingItems(s.player)
			}
		}
	} else {
		// Give starting items
		s.manager.world.GiveStartingItems(s.player)
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

	// Clear char creation state
	s.charCreating = false
	s.charName = ""
	s.charSex = 0
	s.charRace = 0
	s.charClass = 0
	s.charHometown = 0
	s.charStats = game.CharStats{}

	// Generate JWT token
	token, err := auth.GenerateJWT(s.player.Name, s.isAgent, s.agentKeyID)
	if err != nil {
		slog.Error("failed to generate JWT token", "error", err)
	}

	// Send welcome with token
	s.sendWelcome(token)

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
func (s *Session) getRaceOptions() map[string]int {
	return map[string]int{
		"0": 0,  // Human
		"1": 1,  // Elf
		"2": 2,  // Dwarf
		"3": 3,  // Halfling
		"4": 4,  // Pixie
		"5": 5,  // Kiroshi
		"6": 6,  // Alaozar
	}
}

// getClassOptions returns available classes for character creation.
func (s *Session) getClassOptions() map[string]int {
	return map[string]int{
		"0": 0,  // Mage
		"1": 1,  // Cleric
		"2": 2,  // Thief
		"3": 3,  // Warrior
		"4": 4,  // Magus
		"5": 5,  // Avatar
		"6": 6,  // Assassin
		"7": 7,  // Paladin
		"8": 8,  // Ninja
		"9": 9,  // Psionic
		"10": 10, // Ranger
		"11": 11, // Mystic
	}
}
