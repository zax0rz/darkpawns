// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"golang.org/x/crypto/bcrypt"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/validation"
	"golang.org/x/time/rate"
)

// LoginAttemptConfig exports the auth config type for convenience.
type LoginAttemptConfig = auth.LoginAttemptConfig

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Development: allow all origins
		if os.Getenv("ENVIRONMENT") == "development" {
			return true
		}

		// Production: validate against allowed origins
		allowedOrigins := []string{
			"https://darkpawns.example.com",
			"https://game.darkpawns.example.com",
			// Add your production domains here
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// No Origin header, could be direct WebSocket connection
			// Allow but log for monitoring
			slog.Warn("WebSocket connection without Origin header", "remote_addr", r.RemoteAddr) // #nosec G706
			return true
		}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		slog.Warn("rejected WebSocket connection from unauthorized origin", "origin", origin) // #nosec G706
		return false
	},
}

// Manager handles all active sessions.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session // keyed by player name
	world        *game.World
	combatEngine *combat.CombatEngine
	shopManager  *systems.ShopManager
	db           db.DB
	hasDB        bool
	loginLimiter    *auth.IPRateLimiter       // Rate limiter for login attempts
	loginAttempts   *auth.LoginAttemptTracker // Lockout tracker (H-15)
	doorManager     *systems.DoorManager

	// Per-IP connection tracking (C5)
	ipConnCount map[string]int
	ipConnMu    sync.Mutex

	// Wizlock state — when true, only immortal players may log in
	wizlockMutex sync.Mutex
	wizlocked    bool
}

// NewManager creates a new session manager.
func NewManager(world *game.World, database *db.DB) *Manager {
	ce := combat.NewCombatEngine()
	ce.Start()

	dm := systems.NewDoorManager()
	if pw := world.GetParsedWorld(); pw != nil {
		dm.LoadDoorsFromWorld(pw)
	}

	m := &Manager{
		sessions:      make(map[string]*Session),
		world:         world,
		combatEngine:  ce,
		shopManager:   systems.NewShopManager(),
		loginLimiter:  auth.NewIPRateLimiter(),
		loginAttempts: auth.NewLoginAttemptTracker(auth.DefaultLoginAttemptConfig()),
		doorManager:   dm,
		ipConnCount:   make(map[string]int),
	}
	if database != nil {
		m.db = *database
		m.hasDB = true
	}

	// Wire MessageSink so that Player.SendMessage routes through Session.send
	world.MessageSink = func(playerName string, msg []byte) {
		s, ok := m.GetSession(playerName)
		if !ok || s == nil {
			return
		}
		// Wrap in JSON event envelope for WebSocket clients
		wrapped, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "text",
				Text: string(msg),
			},
		})
		if err != nil {
			slog.Error("MessageSink marshal error", "error", err)
			return
		}
		select {
		case s.send <- wrapped:
		default:
			slog.Warn("dropping MessageSink message: player channel full",
				"player", playerName,
				"message_preview", truncateStr(string(msg), 120),
			)
		}
	}

	// Wire CloseConnection so game-layer close requests route through the session
	world.CloseConn = func(playerName string) {
		m.UnregisterAndClose(playerName)
	}

	// Wire game-level callbacks
	// HasActiveCharacter allows game.ValidName to check against active sessions.
	game.HasActiveCharacter = func(name string) bool {
		_, ok := m.GetSession(name)
		return ok
	}

	// Load ban list and invalid name list at startup
	if err := game.LoadBanned(); err != nil {
		slog.Warn("Failed to load ban list", "error", err)
	}
	if err := game.ReadInvalidList(); err != nil {
		slog.Warn("Failed to load invalid name list", "error", err)
	}

	return m
}

// SetCombatBroadcastFunc sets the broadcast function for combat messages.
// Must be called after the manager is created and before combat starts.
func (m *Manager) SetCombatBroadcastFunc() {
	m.combatEngine.SetBroadcastFunc(func(roomVNum int, message string, exclude string) {
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "combat",
				Text: message,
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
			return
		}
		m.BroadcastToRoom(roomVNum, msg, exclude)
	})
}

// GetCombatEngine returns the combat engine for AI integration.
func (m *Manager) GetCombatEngine() *combat.CombatEngine {
	return m.combatEngine
}

// SetDeathFunc wires the game-layer death handler into the combat engine.
func (m *Manager) SetDeathFunc() {
	m.combatEngine.DeathFunc = func(victim, killer combat.Combatant, attackType int) {
		m.world.HandleDeath(victim, killer, attackType)

		// If victim was a player, send updated room state after respawn
		if !victim.IsNPC() {
			if s, ok := m.GetSession(victim.GetName()); ok {
// #nosec G104
				cmdLook(s, nil)
			}
		}
	}
}

// SetDamageFunc wires health dirty-tracking into the combat engine.
// When a player takes damage in combat, their HEALTH and MAX_HEALTH vars are
// marked dirty so the next flushDirtyVars call will push the update.
func (m *Manager) SetDamageFunc() {
	m.combatEngine.DamageFunc = func(victimName string) {
		if s, ok := m.GetSession(victimName); ok {
			s.markDirty(VarHealth, VarMaxHealth)
		}
	}
}

// SetScriptFightFunc wires the fight trigger into the combat engine.
// After each combat round, if the mob has a fight script, it fires.
func (m *Manager) SetScriptFightFunc() {
	m.combatEngine.ScriptFightFunc = func(mobName string, targetName string, roomVNum int) {
		m.world.FireMobFightScript(mobName, targetName, roomVNum)
	}
}

// HandleWebSocket upgrades HTTP to WebSocket and manages the session.
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}

	ip := auth.GetIPFromRequest(r)

	// Per-IP connection limit (C5)
	m.ipConnMu.Lock()
	if m.ipConnCount[ip] >= 5 {
		m.ipConnMu.Unlock()
// #nosec G104
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections from your IP"))
// #nosec G104
		conn.Close()
		return
	}
	m.ipConnCount[ip]++
	m.ipConnMu.Unlock()

	session := &Session{
		conn:           conn,
		request:        r, // Store the HTTP request for IP extraction
		manager:        m,
		send:           make(chan []byte, 256),
		limiter:        rate.NewLimiter(rate.Limit(10), 10),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		pendingEvents:  nil,
		connectedAt:    time.Now(),
	}

	// Start goroutines for reading and writing
	go session.writePump()
	go session.readPump()
}

// Register adds a session for a player.
func (m *Manager) Register(playerName string, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[playerName]; exists {
		return ErrPlayerAlreadyOnline
	}

	m.sessions[playerName] = s
	s.playerName = playerName
	return nil
}

// Unregister removes a session and saves the player to DB.
func (m *Manager) Unregister(playerName string) {
	m.mu.Lock()
	s, ok := m.sessions[playerName]
	if ok {
		delete(m.sessions, playerName)
	}
	m.mu.Unlock()

	if ok {
		// Decrement per-IP connection count (C5)
		if s.request != nil {
			ip := auth.GetIPFromRequest(s.request)
			m.ipConnMu.Lock()
			m.ipConnCount[ip]--
			if m.ipConnCount[ip] <= 0 {
				delete(m.ipConnCount, ip)
			}
			m.ipConnMu.Unlock()
		}

		// Save to DB on disconnect
		if m.hasDB && s.player != nil && s.player.ID > 0 {
			if rec, err := db.PlayerToRecord(s.player, nil); err == nil {
				if err := m.db.SavePlayer(rec); err != nil {
					slog.Error("DB save error", "player", playerName, "error", err)
				}
			}
		}
		m.combatEngine.StopCombat(playerName)
		s.sendOnce.Do(func() { close(s.send) })
		m.world.RemovePlayer(playerName)
	}
}

// GetSession returns a session by player name.
func (m *Manager) GetSession(playerName string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[playerName]
	return s, ok
}

// BroadcastToRoom sends a message to all players in a room.
func (m *Manager) BroadcastToRoom(roomVNum int, message []byte, excludePlayer string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, s := range m.sessions {
		if name == excludePlayer {
			continue
		}
		if s.player != nil && s.player.GetRoom() == roomVNum {
			select {
			case s.send <- message:
			default:
				// Channel full, drop message — log for observability
				msgPreview := string(message)
				if len(msgPreview) > 120 {
					msgPreview = msgPreview[:120] + "..."
				}
				slog.Warn("dropping broadcast message: player channel full",
					"player", name,
					"room", roomVNum,
					"message_preview", truncateStr(string(message), 120),
				)
		}
	}
}

// Session represents a single WebSocket connection.
type Session struct {
	conn          *websocket.Conn
	request       *http.Request // Store the original HTTP request for IP extraction
	manager       *Manager
	send          chan []byte
	player        *game.Player
	playerName    string
	authenticated bool

	// Agent auth — set on login when mode="agent"
	isAgent     bool
	agentKeyID  int64
	connectedAt time.Time // set on session creation, used for sessionID()

	// agentMu protects all agent-related state from concurrent access.
	// readPump goroutine and combat ticker goroutine (via DamageFunc) both
	// call markDirty/flushDirtyVars which touch the maps below.
	agentMu        sync.Mutex
	subscribedVars map[string]bool // vars this session subscribed to
	dirtyVars      map[string]bool // vars changed since last flush
	pendingEvents  []interface{}   // queued EVENTS since last flush

	// Character creation state
	charCreating bool
	charStage    string // current stage in creation flow (sex, race, class, confirm)
	charName     string
	charSex      int
	charRace     int
	charClass    int
	charHometown int
	charStats    game.CharStats

	// Character switch state (wizard commands)
	isSwitched       bool
	switchedOriginal *game.Player
	switchedMob      *game.MobInstance
	switchedPlayer   *game.Player

	// Rate limit: capacity=10, refill=10/sec (token bucket via golang.org/x/time/rate)
	// This protects the server from command floods — it does NOT protect API costs.
	// Agents must implement their own circuit breakers for LLM-level loop detection.
	// See scripts/dp_bot.py for reference implementation.
	limiter *rate.Limiter

	// Temporary data storage for command handlers
	tempData map[string]interface{}

	// Infobar / display state (from act.display.c)
	screenSize  int // terminal height in lines; 0 = unset (defaults to 25)
	infobarMode int // InfobarOff (0) or InfobarOn (1)

	// Communication state
	lastTeller string   // Last player who told us (for reply)
	snooping  *Session  // Session being snooped (for wizard snoop)
	snoopBy   *Session  // Session that is snooping us

	// idleTicsSet tracks whether the idle timeout counter has been set
	// for pre-login sessions. Used by CheckIdlePasswords().
	idleTicsSet bool

	// sendOnce ensures s.send is closed exactly once across all disconnect paths.
	sendOnce sync.Once
}

// readPump reads messages from the WebSocket.
func (s *Session) readPump() {
	defer func() {
		s.manager.Unregister(s.playerName)
// #nosec G104
		s.conn.Close()
	}()

	s.conn.SetReadLimit(16384) // 16KB max message size (C4)
// #nosec G104
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
// #nosec G104
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err)
			}
			break
		}

		if err := s.handleMessage(message); err != nil {
			slog.Error("handle message error", "error", err)
			s.sendError(err.Error())
		}
	}
}

// writePump writes messages to the WebSocket.
func (s *Session) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
// #nosec G104
		s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
// #nosec G104
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
// #nosec G104
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
// #nosec G104
			s.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
// #nosec G104
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages.
func (s *Session) handleMessage(data []byte) error {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	switch msg.Type {
	case MsgLogin:
		return s.handleLogin(msg.Data)
	case MsgCommand:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleCommand(msg.Data)
	case MsgSubscribe:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleSubscribe(msg.Data)
	case MsgCharInput:
		if s.charCreating {
			return s.handleCharInput(msg.Data)
		}
		return ErrNotInCharCreation
	default:
		return ErrUnknownMessageType
	}
}

// handleLogin authenticates a player.
func (s *Session) handleLogin(data json.RawMessage) error {
	var login LoginData
	if err := json.Unmarshal(data, &login); err != nil {
		return err
	}

	// Apply IP-based rate limiting and lockout enforcement (H-12 + H-15)
	ip := auth.GetIPFromRequest(s.request)

	// H-15: Check if IP is locked out from previous failed attempts
	if locked, remaining := s.manager.loginAttempts.IsLocked(ip); locked {
		s.sendError(fmt.Sprintf("Too many failed login attempts. Try again in %d minutes.", int(remaining.Minutes())+1))
// #nosec G104
		s.conn.Close()
		audit.LogSecurityEvent("login_locked_out", "IP locked out from failed login attempts", login.PlayerName, ip)
		return nil
	}

	// H-12: Per-second rate limiting (uses actual TCP IP, not spoofable headers)
	if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
		s.sendError("Too many login attempts. Please try again later.")
// #nosec G104
		s.conn.Close()
		audit.LogSecurityEvent("rate_limit_exceeded", "Login rate limit exceeded", login.PlayerName, ip)
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
					s.manager.loginAttempts.RecordFailure(ip) // H-15: failed attempt (no password)
					return nil
				}
				if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(login.Password)); err != nil {
					s.sendError("Invalid password.")
// #nosec G104
					s.conn.Close()
					audit.LogSecurityEvent("login_failed", "Invalid password", login.PlayerName, ip)
					s.manager.loginAttempts.RecordFailure(ip) // H-15: track failed attempt
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
		s.manager.loginAttempts.RecordSuccess(ip) // H-15: clear failure history on success

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
	// Flush dirty vars for agents after every command dispatch
	if s.isAgent {
		s.flushDirtyVars()
	}
	return err
}

// sendWelcome sends the initial game state to the player.
func (s *Session) sendWelcome(token string) {
	room, _ := s.manager.world.GetRoom(s.player.GetRoom())

	state := StateData{
		Player: PlayerState{
			Name:      s.player.Name,
			Health:    s.player.Health,
			MaxHealth: s.player.MaxHealth,
			Level:     s.player.Level,
			Class:     game.ClassNames[s.player.Class],
			Race:      game.RaceNames[s.player.Race],
			Str:       s.player.Stats.Str,
			Int:       s.player.Stats.Int,
			Wis:       s.player.Stats.Wis,
			Dex:       s.player.Stats.Dex,
			Con:       s.player.Stats.Con,
			Cha:       s.player.Stats.Cha,
		},
		Room: RoomState{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Exits:       getExitNames(room.Exits),
			Doors:       getDoorInfo(s.manager.doorManager, room.VNum, room.Exits),
		},
		Token: token,
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgState,
		Data: state,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.send <- msg
}

// sendError sends an error message to the player.
func (s *Session) sendError(text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgError,
		Data: ErrorData{Message: text},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	select {
	case s.send <- msg:
	default:
	}
}

func getExitNames(exits map[string]parser.Exit) []string {
	var names []string
	for dir := range exits {
		names = append(names, dir)
	}
	return names
}

func getDoorInfo(dm *systems.DoorManager, roomVNum int, exits map[string]parser.Exit) []DoorInfo {
	if dm == nil {
		return nil
	}
	var doors []DoorInfo
	for dir := range exits {
		door, ok := dm.GetDoor(roomVNum, dir)
		if !ok {
			continue
		}
		if !door.CanSee() {
			continue
		}
		doors = append(doors, DoorInfo{
			Direction: dir,
			Closed:    door.Closed,
			Locked:    door.Locked,
		})
	}
	if len(doors) == 0 {
		return nil
	}
	return doors
}

// GetPlayer returns the player associated with this session
func (s *Session) GetPlayer() *game.Player {
	return s.player
}

// GetPlayerInterface returns the player as interface{} for common.CommandSession
func (s *Session) GetPlayerInterface() interface{} {
	return s.player
}

// SendMessage sends a message to the client.
// Routes through Session.send (which writePump reads) — not through Player.Send.
func (s *Session) SendMessage(message string) error {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "text",
			Text: message,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	select {
	case s.send <- msg:
	default:
		// Channel full, drop message
	}
	return nil
}

// Send sends a text message to the client (alternative method name).
// Routes through Session.send directly — not through Player.Send.
func (s *Session) Send(message string) {
	_ = s.SendMessage(message)
}

// MarkDirty marks a variable as dirty for agent subscriptions.
// Deprecated: prefer markDirty (unexported) which uses the agent mutex.
func (s *Session) MarkDirty(vars ...string) {
	s.markDirty(vars...)
}

// GetManager returns the session manager (needed for some admin commands)
func (s *Session) GetManager() interface{} {
	return s.manager
}

// GetWorld returns the game world.
func (s *Session) GetWorld() *game.World {
	return s.manager.world
}

// GetCombatEngine returns the combat engine.
func (s *Session) GetCombatEngine() interface{} {
	return s.manager.combatEngine
}

// GetPlayerName returns the name of the player associated with this session
func (s *Session) GetPlayerName() string {
	if s.player != nil {
		return s.player.Name
	}
	return s.playerName
}

// IsAuthenticated returns whether the session is authenticated
func (s *Session) IsAuthenticated() bool {
	return s.authenticated
}

// GetPlayerRoomVNum returns the room VNum where the player is located
func (s *Session) GetPlayerRoomVNum() int {
	if s.player != nil {
		return s.player.GetRoomVNum()
	}
	return 0
}

// HasPlayer returns true if the session has a player associated with it
func (s *Session) HasPlayer() bool {
	return s.player != nil
}

// NewSession creates a bare session not associated with any WebSocket (for telnet/embed use).
func (m *Manager) NewSession() *Session {
	return &Session{
		manager:        m,
		send:           make(chan []byte, 256),
		limiter:        rate.NewLimiter(rate.Limit(10), 10),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
	}
}

// Manager returns the session manager that owns this session.
func (s *Session) Manager() *Manager {
	return s.manager
}

// PlayerName returns the player name associated with this session.
func (s *Session) PlayerName() string {
	return s.playerName
}

// CloseSend closes the session's outgoing message channel.
func (s *Session) CloseSend() {
	if s.send != nil {
		s.sendOnce.Do(func() { close(s.send) })
	}
}

// SendChannel returns the session's outgoing message channel (for telnet/embed).
func (s *Session) SendChannel() <-chan []byte {
	return s.send
}

// HandleMessage is the exported version of handleMessage (for telnet/embed).
func (s *Session) HandleMessage(data []byte) error {
	return s.handleMessage(data)
}

// Close closes the session
func (s *Session) Close() {
	// Close the connection only; channel close is handled by Unregister()
	if s.conn != nil {
// #nosec G104
		s.conn.Close()
	}
}

// SetTempData stores temporary data in the session
func (s *Session) SetTempData(key string, value interface{}) {
	if s.tempData == nil {
		s.tempData = make(map[string]interface{})
	}
	s.tempData[key] = value
}

// GetTempData retrieves temporary data from the session
func (s *Session) GetTempData(key string) interface{} {
	if s.tempData == nil {
		return nil
	}
	return s.tempData[key]
}

// ClearTempData removes temporary data from the session
func (s *Session) ClearTempData(key string) {
	if s.tempData != nil {
		delete(s.tempData, key)
	}
}

// RandomInt generates a random integer in range [0, n)
func (s *Session) RandomInt(n int) int {
	if n <= 0 {
		return 0
	}
	// Use math/rand for randomness
	// Note: In production, you might want to use a cryptographically secure random source
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	return rand.Intn(n)
}

// truncateStr returns s truncated to maxLen characters with "..." appended if needed.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Errors
var (
	ErrPlayerAlreadyOnline = fmt.Errorf("player already online")
	ErrNotAuthenticated    = fmt.Errorf("not authenticated")
	ErrUnknownMessageType  = fmt.Errorf("unknown message type")
	ErrInvalidPlayerName   = fmt.Errorf("invalid player name")
	ErrNotInCharCreation   = fmt.Errorf("not in character creation")
)

// Command management methods to implement common.CommandManager interface

// RegisterCommand registers a command handler
func (m *Manager) RegisterCommand(name string, handler func(common.CommandSession, []string) error) {
	// This is a stub implementation
	// In a real implementation, this would register the command with the session manager
	slog.Debug("RegisterCommand called (stub)", "name", name)
}

// Sessions returns all active sessions
func (m *Manager) Sessions() []common.CommandSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]common.CommandSession, 0, len(m.sessions))
	for _, sess := range m.sessions {
		// Create a wrapper that implements common.CommandSession
		wrapper := &commandSessionWrapper{session: sess}
		sessions = append(sessions, wrapper)
	}
	return sessions
}

// commandSessionWrapper wraps a Session to implement common.CommandSession
type commandSessionWrapper struct {
	session *Session
}

func (w *commandSessionWrapper) Send(msg string) {
	w.session.Send(msg)
}

func (w *commandSessionWrapper) Close() {
	w.session.Close()
}

func (w *commandSessionWrapper) GetPlayer() interface{} {
	return w.session.GetPlayer()
}

func (w *commandSessionWrapper) GetPlayerName() string {
	return w.session.GetPlayerName()
}

func (w *commandSessionWrapper) GetPlayerRoomVNum() int {
	return w.session.GetPlayerRoomVNum()
}

func (w *commandSessionWrapper) IsAuthenticated() bool {
	return w.session.IsAuthenticated()
}

func (w *commandSessionWrapper) HasPlayer() bool {
	return w.session.HasPlayer()
}

// Lock locks the manager mutex
func (m *Manager) Lock() {
	m.mu.Lock()
}

// Unlock unlocks the manager mutex
func (m *Manager) Unlock() {
	m.mu.Unlock()
}

// RLock locks the manager mutex for reading
func (m *Manager) RLock() {
	m.mu.RLock()
}

// RUnlock unlocks the manager mutex for reading
func (m *Manager) RUnlock() {
	m.mu.RUnlock()
}

// Mu returns the mutex for synchronization
func (m *Manager) Mu() interface{} {
	return &m.mu
}

// ---------------------------------------------------------------------------
// Session lifecycle and communication — ported from comm.c
// ---------------------------------------------------------------------------

// UnregisterAndClose removes a session for the specified player and cleans up
// all associated resources. This is the Go equivalent of close_socket() from
// comm.c. It handles:
//   - Flushing queues (input/output)
//   - Closing the WebSocket connection
//   - Saving player state if CON_PLAYING
//   - Notifying the room of departure
//   - Removing from the sessions map
//   - Freeing compression/showstr state (not applicable in Go version)
func (m *Manager) UnregisterAndClose(playerName string) {
	m.mu.Lock()
	s, ok := m.sessions[playerName]
	if ok {
		delete(m.sessions, playerName)
	}
	m.mu.Unlock()

	if !ok || s == nil {
		slog.Warn("unregister and close: session not found", "player", playerName)
		return
	}

	// Flush any pending output
	s.FlushQueues()

	// Save player state
	if s.player != nil {
		// Notify room of departure
		leaveMsg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "leave",
				Text: s.player.Name + " has left the game.",
			},
		})
		if err == nil {
			m.BroadcastToRoom(s.player.GetRoom(), leaveMsg, s.player.Name)
		}

		// Save to DB
		if m.hasDB && s.player.ID > 0 {
			if rec, err := db.PlayerToRecord(s.player, nil); err == nil {
				if err := m.db.SavePlayer(rec); err != nil {
					slog.Error("DB save error on disconnect", "player", playerName, "error", err)
				}
			}
		}

		// Remove from world
		m.world.RemovePlayer(playerName)
	}

	// Close the WebSocket connection
	if s.conn != nil {
// #nosec G104
		s.conn.Close()
	}

	m.combatEngine.StopCombat(playerName)

	// Close the send channel to stop the write pump
	s.sendOnce.Do(func() { close(s.send) })

	// Clean up snooping state
	if s.snoopBy != nil {
		s.snoopBy.snooping = nil
	}
	if s.snooping != nil {
		s.snooping.snoopBy = nil
	}

	slog.Info("session closed", "player", playerName)
}

// FlushQueues drains any pending input/output for a session.
// In the WebSocket Go version this is a no-op for input (handled by readPump), but we
// keep the method for compatibility with the flush_queues() semantics.
// Ported from comm.c:flush_queues().
func (s *Session) FlushQueues() {
	// Drain the send channel (pending output)
	for {
		select {
		case <-s.send:
		default:
			return
		}
	}
}

// SendToAll sends a text message to all connected, playing sessions.
// Ported from comm.c:send_to_all().
func (m *Manager) SendToAll(message string) {
	if message == "" {
		return
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "broadcast",
			Text: message,
		},
	})
	if err != nil {
		slog.Error("SendToAll marshal error", "error", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.player == nil || !s.authenticated {
			continue
		}
		select {
		case s.send <- msg:
		default:
			slog.Debug("SendToAll: dropping message to full channel", "player", s.playerName)
		}
	}
}

// SendToOutdoor sends a message to all playing sessions whose characters are
// awake and in an outdoor room (Sector > 0, i.e. not SECT_INSIDE).
// Ported from comm.c:send_to_outdoor().
func (m *Manager) SendToOutdoor(message string) {
	if message == "" {
		return
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "outdoor",
			Text: message,
		},
	})
	if err != nil {
		slog.Error("SendToOutdoor marshal error", "error", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.player == nil || !s.authenticated {
			continue
		}
		// AWAKE check: position >= PosStanding
		if s.player.GetPosition() < combat.PosStanding {
			continue
		}
		// OUTSIDE check: sector type != INSIDE (0)
		roomVNum := s.player.GetRoom()
		if room, ok := m.world.GetRoom(roomVNum); ok && room.Sector == 0 {
			continue // SECT_INSIDE
		}
		select {
		case s.send <- msg:
		default:
			slog.Debug("SendToOutdoor: dropping message to full channel", "player", s.playerName)
		}
	}
}

// CheckIdlePasswords checks for idle pre-login sessions (not yet fully connected)
// and disconnects them if they have been idle for more than one tick cycle.
// Ported from comm.c:check_idle_passwords().
//
// In the Go WebSocket version, a session is considered "pre-login" if authenticated is false
// (i.e. they haven't completed login yet). The idleTics counter is checked:
// - First idle tick: increment counter
// - Second idle tick: send timeout message and mark for close
func (m *Manager) CheckIdlePasswords() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var toDelete []string

	for name, s := range m.sessions {
		// Only check pre-login sessions (not yet authenticated)
		if s.authenticated {
			continue
		}

		if !s.idleTicsSet {
			s.idleTicsSet = true
			continue
		}

		// Timed out
		timeoutMsg, err := json.Marshal(ServerMessage{
			Type: MsgError,
			Data: ErrorData{Message: "\r\nTimed out... goodbye.\r\n"},
		})
		if err == nil {
			select {
			case s.send <- timeoutMsg:
			default:
			}
		}

		// Close the connection
		if s.conn != nil {
// #nosec G104
			s.conn.Close()
		}

		toDelete = append(toDelete, name)
	}

	for _, name := range toDelete {
		// Close channel and remove
		if s, ok := m.sessions[name]; ok {
			s.sendOnce.Do(func() { close(s.send) })
			delete(m.sessions, name)
		}
	}

	if len(toDelete) > 0 {
		slog.Info("timed out idle pre-login sessions", "count", len(toDelete))
	}
}

// CountSessions returns the number of connected and playing sessions.
// Implements engine.UsageCounter for record_usage().
func (m *Manager) CountSessions() (connected int, playing int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		connected++
		if s.authenticated && s.player != nil {
			playing++
		}
	}
	return
}

// IsWizlocked returns whether the game is in wizard-only login mode.
func (m *Manager) IsWizlocked() bool {
	m.wizlockMutex.Lock()
	defer m.wizlockMutex.Unlock()
	return m.wizlocked
}

// SetWizlock sets or clears wizard-only login mode.
func (m *Manager) SetWizlock(locked bool) {
	m.wizlockMutex.Lock()
	defer m.wizlockMutex.Unlock()
	m.wizlocked = locked
}

// HasDB returns whether a database backend is configured.
func (m *Manager) HasDB() bool {
	return m.hasDB
}

