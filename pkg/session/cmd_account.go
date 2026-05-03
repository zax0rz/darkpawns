package session

import (
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// cmdPassword handles password changes.
// Usage: password <old> <new>
func cmdPassword(s *Session, args []string) error {
	if !s.manager.hasDB {
		s.Send("Password management requires a database connection, which is not available.")
		return nil
	}

	if len(args) < 2 {
		s.Send("Usage: password <old> <new>")
		return nil
	}

	oldPass := args[0]
	newPass := args[1]

	if oldPass == newPass {
		s.Send("That's the same as your old password!")
		return nil
	}

	if len(newPass) < 4 {
		s.Send("Password must be at least 4 characters.")
		return nil
	}

	if len(newPass) > 72 {
		s.Send("Password is too long (max 72 characters).")
		return nil
	}

	// Load current player record from DB
	rec, err := s.manager.db.GetPlayer(s.playerName)
	if err != nil {
		slog.Error("password change: failed to load player", "player", s.playerName, "error", err)
		s.Send("An error occurred. Please try again later.")
		return nil
	}
	if rec == nil {
		s.Send("Player record not found.")
		return nil
	}

	// Verify old password if one is set
	if rec.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(oldPass)); err != nil {
			s.Send("Old password is incorrect.")
			return nil
		}
	}

	// Hash new password
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("password change: bcrypt hash error", "error", err)
		s.Send("An error occurred. Please try again later.")
		return nil
	}

	err = s.manager.db.UpdatePassword(rec.ID, string(hashedPwd))
	if err != nil {
		slog.Error("password change: db update failed", "player", s.playerName, "error", err)
		s.Send("Failed to save new password. Please try again later.")
		return nil
	}

	s.Send("Password changed successfully.")
	slog.Info("password changed", "player", s.playerName)
	return nil
}

// cmdPrompt sets or toggles the player's prompt display.
// Usage: prompt              — toggle prompt on/off
//
//	prompt <string>     — set custom prompt (supports %h/%m/%v/%H/%M/%V)
//	prompt all          — show all stats (hp/mana/move)
func cmdPrompt(s *Session, args []string) error {
	arg := strings.Join(args, " ")

	if arg == "" {
		s.player.PromptOn = !s.player.PromptOn
		if s.player.PromptOn {
			s.Send("Prompt now on.")
		} else {
			s.Send("Prompt now off.")
		}
		return nil
	}

	if strings.EqualFold(arg, "all") {
		s.player.PromptStr = "%h/%H hp %m/%M mana %v/%V mv > "
		s.player.PromptOn = true
		s.Send("Prompt set to show all.")
		return nil
	}

	if strings.EqualFold(arg, "off") {
		s.player.PromptOn = false
		s.Send("Prompt now off.")
		return nil
	}

	if strings.EqualFold(arg, "on") {
		s.player.PromptOn = true
		s.Send("Prompt now on.")
		return nil
	}

	s.player.PromptStr = arg
	s.player.PromptOn = true
	s.Send(fmt.Sprintf("Prompt set to: %s", arg))
	return nil
}
