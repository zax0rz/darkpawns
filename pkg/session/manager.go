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
			slog.Warn("WebSocket connection without Origin header", "remote_addr", r.RemoteAddr)
			return true
		}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		slog.Warn("rejected WebSocket connection from unauthorized origin", "origin", origin)
		return false
	},
}

// Manager handles all active sessions.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session // keyed by player name
	world        *game.World
	combatEngine *combat.CombatEngine
	db           db.DB
	hasDB        bool
	loginLimiter *auth.IPRateLimiter // Rate limiter for login attempts
	doorManager  *systems.DoorManager
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
		sessions:     make(map[string]*Session),
		world:        world,
		combatEngine: ce,
		loginLimiter: auth.NewIPRateLimiter(),
		doorManager:  dm,
	}
	if database != nil {
		m.db = *database
		m.hasDB = true
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
		// Save to DB on disconnect
		if m.hasDB && s.player != nil && s.player.ID > 0 {
			if rec, err := db.PlayerToRecord(s.player, nil); err == nil {
				if err := m.db.SavePlayer(rec); err != nil {
					slog.Error("DB save error", "player", playerName, "error", err)
				}
			}
		}
		close(s.send)
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
				// Channel full, drop message
			}
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

	// Agent subscription state — only populated when isAgent==true
	subscribedVars map[string]bool // vars this session subscribed to
	dirtyVars      map[string]bool // vars changed since last flush
	pendingEvents  []interface{}   // queued EVENTS since last flush

	// Character creation state
	charCreating bool
	charName     string
	charSex      int
	charRace     int
	charClass    int
	charHometown int
	charStats    game.CharStats

	// Rate limit: capacity=10, refill=10/sec (token bucket via golang.org/x/time/rate)
	// This protects the server from command floods — it does NOT protect API costs.
	// Agents must implement their own circuit breakers for LLM-level loop detection.
	// See scripts/dp_bot.py for reference implementation.
	limiter *rate.Limiter

	// Temporary data storage for command handlers
	tempData map[string]interface{}

	// Communication state
	lastTeller string   // Last player who told us (for reply)
	snooping  *Session  // Session being snooped (for wizard snoop)
	snoopBy   *Session  // Session that is snooping us
}

// readPump reads messages from the WebSocket.
func (s *Session) readPump() {
	defer func() {
		s.manager.Unregister(s.playerName)
		s.conn.Close()
	}()

	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
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
		s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			s.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
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

	// Apply IP-based rate limiting for login attempts
	ip := auth.GetIPFromRequest(s.request)
	if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
		s.sendError("Too many login attempts. Please try again later.")
		s.conn.Close()
		audit.LogSecurityEvent("rate_limit_exceeded", "Login rate limit exceeded", login.PlayerName, ip)
		return nil
	}

	// Agent auth path — mode="agent" with api_key
	if login.Mode == "agent" && login.APIKey != "" {
		if !s.manager.hasDB {
			s.sendError("agent auth requires database")
			s.conn.Close()
			return nil
		}
		charName, keyID, valid := s.manager.db.ValidateAgentKey(login.APIKey)
		if !valid {
			s.sendError("invalid agent key")
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
		s.conn.Close()
		audit.LogSecurityEvent("invalid_player_name", "Invalid player name format", login.PlayerName, ip)
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
					s.conn.Close()
					return nil
				}
				if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(login.Password)); err != nil {
					s.sendError("Invalid password.")
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
				s.conn.Close()
				return nil
			}
			hashedPwd, err := bcrypt.GenerateFromPassword([]byte(login.Password), bcrypt.DefaultCost)
			if err != nil {
				slog.Error("bcrypt hash error", "error", err)
				s.sendError("Internal error during character creation.")
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
			s.pendingEvents = append(s.pendingEvents, map[string]interface{}{"type": "rate_limited", "command": cmd.Command})
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

// SendMessage sends a message to the client
func (s *Session) SendMessage(message string) error {
	if s.player == nil {
		return fmt.Errorf("no player associated with session")
	}
	s.player.SendMessage(message)
	return nil
}

// Send sends a message to the client (alternative method name)
func (s *Session) Send(message string) {
	if s.player != nil {
		s.player.SendMessage(message)
	}
}

// MarkDirty marks a variable as dirty for agent subscriptions
func (s *Session) MarkDirty(vars ...string) {
	for _, v := range vars {
		s.dirtyVars[v] = true
	}
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
		close(s.send)
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
	return rand.Intn(n)
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
