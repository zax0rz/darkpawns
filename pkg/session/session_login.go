// Package session manages WebSocket connections and player sessions.
package session

import (
	"fmt"
	"log/slog"
	"time"
	"encoding/json"

	"golang.org/x/crypto/bcrypt"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
)

func (s *Session) handleLogin(data json.RawMessage) error {
	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return err
	}

	// Apply IP-based rate limiting for login attempts
	ip := auth.GetIPFromRequest(s.request)
	if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
		s.sendError("Too many login attempts. Please try again later.")
// #nosec G104
		s.conn.Close()
		audit.LogSecurityEvent("rate_limit_exceeded", "Login rate limit exceeded", login.PlayerName, ip)
		return nil
	}

	// H-15: Check login attempt lockout BEFORE auth attempt
	if locked, remaining := s.manager.loginAttempts.IsLocked(ip); locked {
		mins := int(remaining.Minutes()) + 1
		s.sendError(fmt.Sprintf("Too many failed login attempts. Try again in %d minutes.", mins))
// #nosec G104
		s.conn.Close()
		audit.LogSecurityEvent("login_locked_out", "Login locked out due to repeated failures", login.PlayerName, ip)
		return nil
	}

	// Agent auth path — mode="agent" with api_key
	if login.Mode == "agent" && login.APIKey != "" {
		if !s.manager.hasDB {
			s.sendError("agent auth requires database")
// #nosec G104
			s.conn.Close()
			return nil
		}
		charName, keyID, valid := s.manager.db.ValidateAgentKey(login.APIKey)
		if !valid {
			s.sendError("invalid agent key")
// #nosec G104
			s.conn.Close()
			return nil
		}
		// Use character name from the key — ignore login.PlayerName for security
		login.PlayerName = charName
		s.isAgent = true
		s.agentKeyID = keyID
	}

	if login.PlayerName == "" {
		return ErrInvalidPlayerName
	}

	// Validate player name
	if !validation.IsValidPlayerName(login.PlayerName) {
		s.sendError("Invalid player name. Names must be 2-32 characters and contain only letters, numbers, spaces, dots, dashes, and underscores.")
// #nosec G104
		s.conn.Close()
		audit.LogSecurityEvent("invalid_player_name", "Invalid player name format", login.PlayerName, ip)
		return nil
	}

	// Check against invalid name list (profanity filter) — from game/ban.c
	if !game.ValidName(login.PlayerName) {
		s.sendError("Invalid player name. Please choose another.")
// #nosec G104
		s.conn.Close()
		return nil
	}

	// Load from DB if available
	if s.manager.hasDB {
		rec, err := s.manager.db.GetPlayer(login.PlayerName)
		if err != nil {
			slog.Error("DB load error", "player", login.PlayerName, "error", err)
		}

		if rec != nil && !login.NewChar {
			// Returning player — verify password
			if rec.Password != "" {
				if login.Password == "" {
					s.sendError("Password required.")
// #nosec G104
					s.conn.Close()
					return nil
				}
				if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(login.Password)); err != nil {
					s.manager.loginAttempts.RecordFailure(ip)
					s.sendError("Invalid password.")
// #nosec G104
					s.conn.Close()
					audit.LogSecurityEvent("login_failed", "Invalid password", login.PlayerName, ip)
					return nil
				}
			}
			p, err := db.RecordToPlayer(rec, s.manager.world)
			if err != nil {
				slog.Error("RecordToPlayer error", "error", err)
				// Fall back to character creation
				s.startCharCreation(login.PlayerName)
				return nil
			}
			s.player = p
			s.authenticated = true
		} else {
			// New character — require password
			if login.Password == "" {
				s.sendError("Password required for new characters.")
// #nosec G104
				s.conn.Close()
				return nil
			}
			hashedPwd, err := bcrypt.GenerateFromPassword([]byte(login.Password), bcrypt.DefaultCost)
			if err != nil {
				slog.Error("bcrypt hash error", "error", err)
				s.sendError("Internal error during character creation.")
// #nosec G104
				s.conn.Close()
				return nil
			}
			s.player = game.NewCharacter(0, login.PlayerName, login.Class, login.Race)
			// Save immediately to get an ID
			if r, err := db.PlayerToRecord(s.player, nil); err == nil {
				r.Password = string(hashedPwd)
				if err := s.manager.db.CreatePlayer(r); err != nil {
					slog.Error("DB create error", "error", err)
				} else {
					s.player.ID = r.ID
					// Give starting items and skills — do_start() from class.c
					s.manager.world.GiveStartingItems(s.player)
					game.GiveStartingSkills(s.player)
				}
			}
			s.authenticated = true
		}
	} else {
		// No DB - always create new character
		s.player = game.NewCharacter(0, login.PlayerName, login.Class, login.Race)
		// Give starting items and skills — do_start() from class.c
		s.manager.world.GiveStartingItems(s.player)
		game.GiveStartingSkills(s.player)
		s.authenticated = true
	}

	// If we created a player directly (not through char creation), proceed with registration
	if s.authenticated && s.player != nil {
		s.manager.loginAttempts.RecordSuccess(ip)
		if err := s.manager.Register(login.PlayerName, s); err != nil {
			return err
		}

		if err := s.manager.world.AddPlayer(s.player); err != nil {
			s.manager.Unregister(login.PlayerName)
			return err
		}

		// Generate JWT token for API access
		token, err := auth.GenerateJWT(login.PlayerName, s.isAgent, s.agentKeyID)
		if err != nil {
			slog.Error("failed to generate JWT token", "error", err)
		}
		s.tokenIssuedAt = time.Now()

		// Send welcome with token
		s.sendWelcome(token)

		// Agents get a full variable dump + memory bootstrap immediately after login
		if s.isAgent {
			s.sendFullVarDump()
			s.SendMemoryBootstrap()
		}

		// Broadcast to room
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
	}

	return nil
}

// handleCommand processes game commands.
func (s *Session) handleCommand(data json.RawMessage) error {
	var cmd CommandData
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	// Token bucket rate limit: 10 cmd/sec per session
	if !s.limiter.Allow() {
		s.sendError("rate limit exceeded — slow down")
		if s.isAgent {
			s.agentMu.Lock()
			s.pendingEvents = append(s.pendingEvents, map[string]interface{}{"type": "rate_limited", "command": cmd.Command})
			s.agentMu.Unlock()
			s.markDirty(VarEvents)
			s.flushDirtyVars()
		}
		return nil
	}

	err := ExecuteCommand(s, cmd.Command, cmd.Args)

	// H-25: Proactive JWT refresh — if token is within refresh window,
	// generate a new one and push it to the client.
	s.maybeRefreshToken()

	// Flush dirty vars for agents after every command dispatch
	if s.isAgent {
		s.flushDirtyVars()
	}
	return err
}

// sendWelcome sends the initial game state to the player.
