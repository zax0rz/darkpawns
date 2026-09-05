package session

import (
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

// cmdPrompt is the C prompt alias for do_display.
// Source: src/interpreter.c:619 -> src/act.other.c:1024-1082.
func cmdPrompt(s *Session, args []string) error {
	s.manager.world.ExecDisplay(s.player, strings.Join(args, " "))
	return nil
}
