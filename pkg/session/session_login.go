// Package session manages WebSocket connections and player sessions.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

func (s *Session) handleLogin(data json.RawMessage) error {
	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return err
	}

	// Apply IP-based rate limiting for login attempts
	ip := s.RemoteIP()
	if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
		s.sendError("Too many login attempts. Please try again later.")
		s.Close()
		audit.LogSecurityEvent("rate_limit_exceeded", "Login rate limit exceeded", login.PlayerName, ip)
		return nil
	}

	// H-15: Check login attempt lockout BEFORE auth attempt
	if locked, remaining := s.manager.loginAttempts.IsLocked(ip); locked {
		mins := int(remaining.Minutes()) + 1
		s.sendError(fmt.Sprintf("Too many failed login attempts. Try again in %d minutes.", mins))
		s.Close()
		audit.LogSecurityEvent("login_locked_out", "Login locked out due to repeated failures", login.PlayerName, ip)
		return nil
	}

	// Agent identity declaration — agents play by the same rules as humans.
	// Server tags the session for observation, but gameplay is identical.
	if login.IsAgent {
		s.isAgent = true
		s.agentHarness = login.Harness
		s.agentModel = login.Model
		s.agentVersion = login.Version
		slog.InfoContext(
			s.sessionCtx,
			"agent identity declared",
			s.logAttrs(
				slog.String("harness", login.Harness),
				slog.String("model", login.Model),
			)...,
		)
	}

	if login.PlayerName == "" {
		return ErrInvalidPlayerName
	}

	if strings.HasPrefix(strings.ToLower(login.PlayerName), "guest") {
		// Bypasses DB password authentication & character creation completely!
		guestName := login.PlayerName
		if strings.EqualFold(guestName, "guest") {
			// Generate dynamic unique name Guest_XXXX
			guestName = fmt.Sprintf("Guest_%d", time.Now().UnixNano()%10000)
		}
		// Avoid duplicate names for active sessions
		for {
			if _, ok := s.manager.GetSession(guestName); ok {
				guestName = fmt.Sprintf("Guest_%d", time.Now().UnixNano()%10000)
			} else {
				break
			}
		}

		s.player = game.NewCharacter(0, guestName, game.ClassWarrior, game.RaceHuman)
		s.player.Stats = game.RollRealAbils(game.ClassWarrior, game.RaceHuman)
		s.player.Sex = 0 // Male
		s.player.Hometown = 1
		s.player.RoomVNum = game.MortalStartRoom // 8004
		s.player.MaxHealth = 100
		s.player.Health = 100
		s.player.MaxMana = 20
		s.player.Mana = 20
		s.player.MaxMove = 100
		s.player.Move = 100
		game.GiveStartingSkills(s.player)
		grantClassSpells(s.player)

		s.authenticated = true
		s.isGuest = true
		s.playerName = guestName

		s.manager.loginAttempts.RecordSuccess(ip)
		if err := s.manager.Register(guestName, s); err != nil {
			return err
		}

		if err := s.manager.world.AddPlayer(s.player); err != nil {
			s.manager.Unregister(guestName)
			return err
		}

		s.manager.world.GiveStartingItems(s.player)

		// Look around so they see the room on entry!
		if err := ExecuteCommand(s, "look", nil); err != nil {
			slog.ErrorContext(s.sessionCtx, "look command failed on entry for guest", s.logAttrs(slog.Any("error", err))...)
		}

		// Generate a dummy JWT token for WebSocket client auth checks
		token, err := auth.GenerateJWT(guestName, s.isAgent, s.agentKeyID, "")
		if err != nil {
			slog.ErrorContext(s.sessionCtx, "failed to generate JWT token for guest", s.logAttrs(slog.Any("error", err))...)
		}
		s.tokenIssuedAt = time.Now()

		s.sendWelcome(token)

		// Broadcast to room
		enterMsg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "enter",
				Text: s.player.Name + " has arrived.",
			},
		})
		if err == nil {
			s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)
		}

		return nil
	}

	// Validate player name
	if !validation.IsValidPlayerName(login.PlayerName) {
		s.sendError("Invalid player name. Names must be 2-32 characters and contain only letters, numbers, spaces, dots, dashes, and underscores.")
		s.Close()
		audit.LogSecurityEvent("invalid_player_name", "Invalid player name format", login.PlayerName, ip)
		return nil
	}

	if login.Password == "" {
		s.sendError("Password required.")
		s.Close()
		return nil
	}

	// Load from DB if available
	if s.manager.hasDB {
		rec, err := s.manager.db.GetPlayer(login.PlayerName)
		if err != nil {
			slog.ErrorContext(s.sessionCtx, "DB load error", s.logAttrs(slog.Any("error", err))...)
		}

		if rec != nil && !login.NewChar {
			// Returning player — verify password
			if rec.Password != "" {
				if login.Password == "" {
					s.sendError("Password required.")
					s.Close()
					return nil
				}
				if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(login.Password)); err != nil {
					s.manager.loginAttempts.RecordFailure(ip)
					s.sendError("Invalid password.")
					s.Close()
					audit.LogSecurityEvent("login_failed", "Invalid password", login.PlayerName, ip)
					return nil
				}
			}
			p, err := db.RecordToPlayer(rec, s.manager.world)
			if err != nil {
				slog.ErrorContext(s.sessionCtx, "RecordToPlayer error", s.logAttrs(slog.Any("error", err))...)
				// Fall back to character creation (only check profanity list/length without active check)
				if !game.ValidNameNoActive(login.PlayerName) {
					s.sendError("Invalid player name. Please choose another.")
					s.Close()
					return nil
				}
				s.startNewCharFlow(login.PlayerName, login.Password)
				return nil
			}
			if aliases, aErr := game.ReadAliases(p.Name); aErr == nil {
				p.Aliases = aliases
			}
			s.player = p
			s.authenticated = true
		} else {
			// New character or player doesn't exist — start stateful creation flow

			// Block new char creation from BanNew/BanSelect sites (DP-418)
			if s.banLevel == game.BanNew || s.banLevel == game.BanSelect {
				s.sendError("New character creation is not allowed from your site.")
				s.Close()
				return nil
			}

			if rec != nil && login.NewChar {
				// Name already exists — reject
				s.sendError(fmt.Sprintf("A character named '%s' already exists. Please choose a different name.", login.PlayerName))
				return nil
			}

			// Validate player name for character creation (checks format, profanity, and online duplicates)
			if !game.ValidName(login.PlayerName) {
				s.sendError("Invalid player name. Please choose another.")
				s.Close()
				return nil
			}

			s.startNewCharFlow(login.PlayerName, login.Password)
			return nil
		}
	} else {
		// No DB - start creation flow statefully
		// Validate player name for character creation without active check (to allow no-DB test takeover)
		if !game.ValidNameNoActive(login.PlayerName) {
			s.sendError("Invalid player name. Please choose another.")
			s.Close()
			return nil
		}
		s.startNewCharFlow(login.PlayerName, login.Password)
		return nil
	}

	// Check if player is banned before entering the game
	if s.authenticated && s.player != nil && s.manager.modChecker != nil {
		if errMsg, banned := s.manager.modChecker.CheckPreCommand(s.player.Name, ""); banned {
			s.sendError(errMsg)
			s.Close()
			slog.WarnContext(s.sessionCtx, "banned player denied entry", s.logAttrs(slog.String("ip", ip))...)
			return nil
		}
	}

	// If we created a player directly (not through char creation), proceed with registration
	if s.authenticated && s.player != nil {
		// Returning players: reconcile known spells with class/level. Saved
		// characters predate spell-granting (or leveled up while offline), so
		// fill in any spells they now qualify for.
		grantClassSpells(s.player)

		s.manager.loginAttempts.RecordSuccess(ip)
		if err := s.manager.Register(login.PlayerName, s); err != nil {
			return err
		}

		if err := s.manager.world.AddPlayer(s.player); err != nil {
			s.manager.Unregister(login.PlayerName)
			return err
		}

		// Look around so they see the room on entry!
		if err := ExecuteCommand(s, "look", nil); err != nil {
			slog.ErrorContext(s.sessionCtx, "look command failed on entry", s.logAttrs(slog.Any("error", err))...)
		}

		// Generate JWT token for API access
		token, err := auth.GenerateJWT(login.PlayerName, s.isAgent, s.agentKeyID, "")
		if err != nil {
			slog.ErrorContext(s.sessionCtx, "failed to generate JWT token", s.logAttrs(slog.Any("error", err))...)
		}
		s.tokenIssuedAt = time.Now()

		// Send welcome with token
		s.sendWelcome(token)

		// Agents get a full variable dump + memory bootstrap + dreaming summary immediately after login.
		// Human structured sessions also get a full variable dump to populate their status bars/UI immediately.
		if s.isAgent || s.wantsStructuredData {
			s.sendFullVarDump()
			if s.isAgent {
				s.SendMemoryBootstrap()
				s.SendMemorySummary()
			}
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
			slog.ErrorContext(s.sessionCtx, "json.Marshal error", s.logAttrs(slog.Any("error", err))...)
			return nil
		}
		s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)
	}

	return nil
}

// isHeavyCommand checks if the command is categorized as a heavy operation.
func isHeavyCommand(cmd string) bool {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	switch cmd {
	case "cast", "buy", "sell", "group", "split", "rent", "quit", "save":
		return true
	default:
		return false
	}
}

// handleCommand processes game commands.
func (s *Session) handleCommand(data json.RawMessage) error {
	var cmd CommandData
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	// PLR_WRITING intercept: if the player is composing mail or a note,
	// buffer the input instead of parsing commands.
	// C equivalent: nanny() checks PLR_WRITING → calls string_add().
	if s.player != nil && s.player.GetFlags()&(1<<game.PlrWriting) != 0 {
		// Reconstruct the full input line from command + args
		line := cmd.Command
		if len(cmd.Args) > 0 {
			line += " " + strings.Join(cmd.Args, " ")
		}
		// PLR_MAILING set → mail compose; unset → note write (do_write).
		if s.player.GetFlags()&(1<<game.PlrMailing) != 0 {
			game.HandleMailInput(s.player, line) // returns true when mail complete; PLR_WRITING cleared inside
		} else {
			game.HandleNoteInput(s.player, line) // returns true when note complete; PLR_WRITING cleared inside
		}
		return nil
	}

	// WriteMagic intercept: player is composing a board message (DP-423)
	// C equivalent: nanny() / write_message() editor input accumulation.
	if s.player != nil && s.player.WriteMagic != 0 && s.manager.world.Boards != nil {
		line := cmd.Command
		if len(cmd.Args) > 0 {
			line += " " + strings.Join(cmd.Args, " ")
		}
		switch line {
		case "~":
			s.manager.world.Boards.FinalizeBoardWrite(s.player.WriteMagic, s.player)
		case "@":
			s.player.WriteMagic = 0
			s.player.SendMessage("Message aborted.\r\n")
		default:
			s.manager.world.Boards.AppendBoardLine(s.player.WriteMagic, line)
		}
		return nil
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

	// Determine timeout based on command scope (dynamic command contexts)
	timeout := 5 * time.Second
	if isHeavyCommand(cmd.Command) {
		timeout = 15 * time.Second
	}

	parentCtx := s.sessionCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	cmdCtx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	// Capture pre-state for decision log
	preState := s.capturePlayerState()
	startTime := time.Now()

	err := ExecuteCommand(s, cmd.Command, cmd.Args)

	// Emit dynamic execution telemetry warning for slow commands (>500ms)
	elapsed := time.Since(startTime)
	if elapsed > 500*time.Millisecond {
		slog.WarnContext(
			cmdCtx, "slow command execution warning",
			"player", s.playerName,
			"command", cmd.Command,
			"elapsed_ms", elapsed.Milliseconds(),
		)
	}

	// Log decision to database (DP-213)
	s.captureAndLog(cmd.Command, cmd.Args, preState, startTime, err)

	// H-25: Proactive JWT refresh — if token is within refresh window,
	// generate a new one and push it to the client.
	s.maybeRefreshToken()

	// Flush dirty vars for agents and human structured sessions after every command dispatch
	if s.isAgent || s.wantsStructuredData {
		s.flushDirtyVars()
	}
	return err
}

// sendWelcome sends the initial game state to the player.
