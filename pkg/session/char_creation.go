package session

import (
	"encoding/json"
	"fmt"
	"log"

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

	// Drive the state machine based on current stage
	// We need to track which stage we're in. For now, implement a simple flow.
	// TODO: Implement full state machine with stage tracking

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
		log.Printf("json.Marshal error: %v", err)
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
		log.Printf("json.Marshal error: %v", err)
		return
	}

	s.send <- msg
}

// completeCharCreation finalizes character creation and enters the world.
func (s *Session) completeCharCreation() error {
	// Create the player with collected attributes
	s.player = game.NewCharacter(0, s.charName, s.charClass, s.charRace)
	s.player.Stats = s.charStats

	// Set sex (for Phase 3 display)
	// TODO: Store sex when Phase 3 implements display

	// Set hometown (for starting room)
	// TODO: Set starting room based on hometown

	// Save to DB if available
	if s.manager.hasDB {
		if r, err := db.PlayerToRecord(s.player, nil); err == nil {
			if err := s.manager.db.CreatePlayer(r); err != nil {
				log.Printf("DB create error during char creation: %v", err)
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
		log.Printf("Failed to generate JWT token: %v", err)
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
		log.Printf("json.Marshal error: %v", err)
		return nil
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)

	return nil
}
