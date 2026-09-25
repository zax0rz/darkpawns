// Package session manages WebSocket connections and player sessions.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

// timingDecoyHash is a valid bcrypt hash of a value no caller can produce. It
// gives the locked-account path the same cost as a real comparison (DP-1281).
const timingDecoyHash = `$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy` // #nosec G101 -- not a credential; a fixed decoy hash for constant-time behaviour

// guestSeq is a monotonic counter for generated guest names so two guests
// never share a "Guest_NNNN" name (DP-912). The previous scheme derived the
// suffix from time.Now().UnixNano()%10000, which collided for sequential
// logins (a name freed by disconnect could be reassigned) and raced under
// concurrency. A counter is unique across both sequential and concurrent
// logins. Note: C has no "guest" login — this is a Go-only affordance, so
// uniqueness (not fidelity to any C scheme) is the goal.
var guestSeq atomic.Int64

func (s *Session) handleLogin(data json.RawMessage) error {
	if s.authenticated || s.SendClosed() {
		return ErrNotInCharCreation
	}
	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return err
	}

	// Apply IP-based rate limiting for login attempts
	ip := s.RemoteIP()
	if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
		s.sendError("Too many login attempts. Please try again later.")
		s.CloseSend()
		audit.LogSecurityEvent("rate_limit_exceeded", "Login rate limit exceeded", login.PlayerName, ip)
		return nil
	}

	// H-15: Check login attempt lockout BEFORE auth attempt
	if locked, remaining := s.manager.loginAttempts.IsLocked(ip); locked {
		mins := int(remaining.Minutes()) + 1
		s.sendError(fmt.Sprintf("Too many failed login attempts. Try again in %d minutes.", mins))
		s.CloseSend()
		audit.LogSecurityEvent("login_locked_out", "Login locked out due to repeated failures", login.PlayerName, ip)
		return nil
	}

	if login.PlayerName == "" {
		s.CloseSend()
		return ErrInvalidPlayerName
	}

	if strings.HasPrefix(strings.ToLower(login.PlayerName), "guest") {
		// Bypasses DB password authentication & character creation completely!
		guestName := login.PlayerName
		if strings.EqualFold(guestName, "guest") {
			// Generate a unique name from a monotonic counter (DP-912).
			guestName = fmt.Sprintf("Guest_%d", guestSeq.Add(1))
		}
		// Belt-and-suspenders: if the (extremely unlikely, counter-wrap) name
		// is already live, keep incrementing until free. The counter makes the
		// common sequential case collision-free without this loop.
		for {
			if _, ok := s.manager.GetSession(guestName); ok {
				guestName = fmt.Sprintf("Guest_%d", guestSeq.Add(1))
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

		// Guest entry renders the room exactly once, via sendWelcome's shared
		// observation below (welcome text first, then the room — matching C's
		// welcome-before-entry-look order). A prior explicit cmdLook here
		// rendered the room a second time on both transports.

		// Generate a dummy JWT token for WebSocket client auth checks
		token, err := auth.GenerateJWT(guestName, "")
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
		s.restartNameEntry()
		audit.LogSecurityEvent("invalid_player_name", "Invalid player name format", login.PlayerName, ip)
		return nil
	}

	// Load from DB if available
	if s.manager.hasDB {
		rec, err := s.manager.db.GetPlayer(login.PlayerName)
		if err != nil {
			return s.abortEntry(fmt.Errorf("load character: %w", err))
		}

		if rec != nil {
			login.PlayerName = rec.Name // Identity lookup and all subsequent accounting use the stored name.
			// DP-592: Account-level lockout check for returning players.
			if s.manager.accountLockouts != nil {
				if locked, remaining := s.manager.accountLockouts.IsLocked(login.PlayerName); locked {
					if login.Password != "" {
						// Mask timing difference against password check (DP-1281)
						_ = bcrypt.CompareHashAndPassword([]byte(timingDecoyHash), []byte(login.Password))
					}
					mins := int(remaining.Minutes()) + 1
					s.sendError(fmt.Sprintf("Account locked due to too many failed login attempts. Try again in %d minutes.", mins))
					s.CloseSend()
					audit.LogSecurityEvent("account_locked", "Account locked due to repeated failures", login.PlayerName, ip)
					return nil
				}
			}

			// C CON_GET_NAME selects the password state. Legacy structured clients
			// may supply a password with login; interactive transports send only a name.
			if login.Password == "" && s.charStage != "login_password" {
				s.charCreating = true
				s.charStage = "login_password"
				s.charName = rec.Name
				s.sendCharCreatePromptWithSecret("login_password", "Password: ", nil, true)
				return nil
			}
			if login.Password == "" {
				s.CloseSend()
				return nil
			}
			if rec.Password != "" && bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(login.Password)) != nil {
				s.manager.loginAttempts.RecordFailure(ip)
				if s.manager.accountLockouts != nil {
					if newlyLocked := s.manager.accountLockouts.RecordFailure(rec.Name); newlyLocked {
						_, remaining := s.manager.accountLockouts.IsLocked(rec.Name)
						mins := int(remaining.Minutes()) + 1
						s.sendError(fmt.Sprintf("Account locked due to too many failed login attempts. Try again in %d minutes.", mins))
						s.CloseSend()
						audit.LogSecurityEvent("account_locked", "Account locked after threshold failures", rec.Name, ip)
						return nil
					}
				}
				if s.loginFailures.Add(1) >= 3 { // C config.c max_bad_pws.
					s.sendCharCreatePrompt("closing", "Wrong password... disconnecting.\r\n", nil)
					s.CloseSend()
				} else {
					s.charCreating = true
					s.charStage = "login_password"
					s.charName = rec.Name
					s.sendCharCreatePromptWithSecret("login_password", "Wrong password.\r\nPassword: ", nil, true)
				}
				return nil
			}
			p, err := db.RecordToPlayer(rec, s.manager.world)
			if err != nil {
				return s.abortEntry(fmt.Errorf("restore character: %w", err))
			}
			if aliases, aErr := game.ReadAliases(p.Name); aErr == nil {
				p.Aliases = aliases
			}
			s.charCreating = false
			s.charStage = ""
			s.charPassword = ""
			s.loginFailures.Store(0)
			s.player = p
			s.olcZone = rec.OlcZone
			s.authenticated = true
			s.menuPasswordHash = rec.Password
		} else {
			// A record removed during password entry must never become creation.
			if s.charStage == "login_password" {
				return s.abortEntry(fmt.Errorf("character is no longer available"))
			}
			// Unknown name — start stateful creation flow.

			// Block new char creation from BanNew/BanSelect sites (DP-418)
			if s.banLevel == game.BanNew || s.banLevel == game.BanSelect {
				s.sendError("New character creation is not allowed from your site.")
				s.CloseSend()
				return nil
			}

			// Validate player name for character creation (checks format, profanity, and online duplicates)
			if !game.ValidName(login.PlayerName) {
				s.restartNameEntry()
				return nil
			}

			s.startNewCharFlow(login.PlayerName)
			return nil
		}
	} else {
		// No DB - start creation flow statefully
		// Validate player name for character creation without active check (to allow no-DB test takeover)
		if !game.ValidNameNoActive(login.PlayerName) {
			s.restartNameEntry()
			return nil
		}
		s.startNewCharFlow(login.PlayerName)
		return nil
	}

	// Check if player is banned before entering the game
	if s.authenticated && s.player != nil && s.manager.modChecker != nil {
		if errMsg, banned := s.manager.modChecker.CheckPreCommand(s.player.Name, ""); banned {
			s.sendError(errMsg)
			s.CloseSend()
			slog.WarnContext(s.sessionCtx, "banned player denied entry", s.logAttrs(slog.String("ip", ip))...)
			return nil
		}
	}

	// Returning characters stop at the post-MOTD menu. World registration and
	// room entry happen only after option 1 is selected.
	if s.authenticated && s.player != nil {
		s.manager.loginAttempts.RecordSuccess(ip)
		if s.manager.accountLockouts != nil {
			s.manager.accountLockouts.RecordSuccess(login.PlayerName)
		}
		// C checks for another copy of the character before the MOTD
		// (interpreter.c:1914-1916).
		if s.performDupeCheck() {
			return nil
		}
		s.startReturningMenu(s.menuPasswordHash)
		return nil
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

	// C's process_input exposes the raw line to an active snooper before any
	// editor, wait-state, alias, or command routing consumes it.
	if cmd.RawLine != "" {
		// Telnet has the original line; use it so a snooper sees `/h` rather
		// than a reconstructed `/ h` while a descriptor editor is active.
		s.forwardSnoopInput(cmd.RawLine, "", nil)
	} else {
		s.forwardSnoopInput(cmd.Command, cmd.RawArgs, cmd.Args)
	}

	// A descriptor room editor owns the complete next line, including its
	// transient improved-editor buffer. This check precedes ordinary command
	// routing just like interpreter.c's CON_REDIT dispatch.
	if s.player != nil && s.isRoomEditing() {
		line := cmd.RawLine
		if line == "" {
			line = commandInputLine(cmd.Command, cmd.Args)
		}
		s.handleReditInput(line)
		return nil
	}

	// An object OLC editor owns the complete next line, including its transient
	// improved-editor buffer, mirroring C's CON_OEDIT case in interpreter.c
	// which calls oedit_parse instead of the command interpreter. Like
	// CON_REDIT it precedes the generic string-editor route so the editor's
	// buffered output flushes in the same turn.
	if s.player != nil && s.isOeditEditing() {
		line := cmd.RawLine
		if line == "" {
			line = commandInputLine(cmd.Command, cmd.Args)
		}
		s.handleOeditInput(line)
		return nil
	}

	// ZEDIT and SEDIT sessions own the complete next line, mirroring
	// interpreter.c's CON_ZEDIT and CON_SEDIT dispatch. Neither has an
	// improved string editor, but both must still win over ordinary command
	// routing so a bare line such as "q" cannot become the quaff command.
	if s.player != nil && (s.isZoneEditing() || s.isSeditEditing()) {
		line := cmd.RawLine
		if line == "" {
			line = commandInputLine(cmd.Command, cmd.Args)
		}
		if s.isZoneEditing() {
			s.handleZeditInput(line)
		} else {
			s.handleSeditInput(line)
		}
		return nil
	}

	// A descriptor string editor owns the complete next line. In particular,
	// slash commands must not be rebuilt as "/ h" from tokenized JSON args;
	// telnet supplies RawLine and direct/WebSocket clients can still use the
	// faithful fallback reconstruction.
	if s.player != nil && s.isTextEditing() {
		line := cmd.RawLine
		if line == "" {
			line = commandInputLine(cmd.Command, cmd.Args)
		}
		s.handleTextEditInput(line)
		return nil
	}

	// A medit (CON_MEDIT) session owns the complete next line while no
	// descriptor string editor is active, mirroring C's CON_MEDIT case in
	// interpreter.c which calls medit_parse instead of the command
	// interpreter.
	if s.player != nil && s.isMeditEditing() {
		line := cmd.RawLine
		if line == "" {
			line = commandInputLine(cmd.Command, cmd.Args)
		}
		s.handleMeditInput(line)
		return nil
	}

	// Clan plan writes use the same PLR_WRITING flag as
	// notes/mail and are completed by the generic string editor equivalent.
	if s.player != nil && s.player.ClanPlanWriting {
		line := cmd.Command
		if len(cmd.Args) > 0 {
			line += " " + strings.Join(cmd.Args, " ")
		}
		if s.manager.world.HandleClanPlanInput(s.player, line) {
			return nil
		}
	}

	// WriteMagic intercept: a board post uses the same PLR_WRITING flag as
	// notes/mail, but has its own C string target (mail_to = board + magic).
	// This must precede the generic PLR_WRITING branch or board lines would be
	// misrouted into the note writer.
	if s.player != nil && s.player.WriteMagic != 0 && s.manager.world.Boards != nil {
		line := cmd.Command
		if len(cmd.Args) > 0 {
			if strings.HasPrefix(line, "/") {
				line += strings.Join(cmd.Args, " ")
			} else {
				line += " " + strings.Join(cmd.Args, " ")
			}
		}
		switch line {
		case "@", "/s":
			s.manager.world.Boards.FinalizeBoardWrite(s.player.WriteMagic, s.player)
			s.player.WriteMagic = 0
			s.player.SetPlrFlag(game.PlrWriting, false)
		case "/a":
			// C's playing_string_cleanup deliberately keeps the post and
			// tells the author to remove it from the board.
			s.manager.world.Boards.AbortBoardWrite(s.player.WriteMagic)
			s.player.WriteMagic = 0
			s.player.SetPlrFlag(game.PlrWriting, false)
			s.player.SendMessage("Post not aborted, use REMOVE <post #>.\r\n")
		default:
			if strings.HasPrefix(line, "/e") {
				rest := strings.TrimSpace(line[2:])
				fields := strings.Fields(rest)
				if len(fields) >= 2 {
					lineNumber, err := strconv.Atoi(fields[0])
					text := strings.TrimSpace(rest[len(fields[0]):])
					if err == nil && s.manager.world.Boards.ReviseBoardLine(s.player.WriteMagic, lineNumber, text) {
						s.player.SendMessage("Line changed.\r\n")
					}
					return nil
				}
			}
			s.manager.world.Boards.AppendBoardLine(s.player.WriteMagic, line)
		}
		return nil
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

	// Token bucket rate limit: 10 cmd/sec per session
	if !s.limiter.Allow() {
		s.sendError("rate limit exceeded — slow down")
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

	startTime := time.Now()

	// C-faithful per-pulse command-drain gate (DP-1201; port of comm.c:603).
	// A command issued while wait>0 — or while earlier commands are still
	// draining — is NOT executed now and NOT rejected: it is appended to the
	// session's drain queue and runs later, one per heartbeat pulse, with no
	// message (the C delay). tryExecuteNow does the atomic wait>0 / queue-empty
	// check + append under inputMu, preserving strict FIFO. Only the wait==0
	// fast path (queue empty) executes immediately here. This is placed after
	// the PLR_WRITING / board / rate-limit intercepts (which have their own
	// game-loop routing in C) and does not touch sendCharInput/sendPagerInput.
	// Internal ExecuteCommand callers (order/force) bypass handleCommand and
	// stay immediate.
	if s.player != nil && s.tryExecuteNow(cmd.Command, cmd.Args, cmd.RawArgs) {
		return nil
	}

	err := executeCommandRaw(s, cmd.Command, cmd.Args, true, cmd.RawArgs)

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

	// H-25: Proactive JWT refresh — if token is within refresh window,
	// generate a new one and push it to the client.
	s.maybeRefreshToken()

	// Flush dirty vars for structured sessions after every command dispatch
	if s.wantsStructuredData {
		s.flushDirtyVars()
	}
	return err
}

// sendWelcome sends the initial game state to the player.
