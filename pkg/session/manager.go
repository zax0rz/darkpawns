// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"golang.org/x/time/rate"
)

// jwtEffectiveLifetime is the effective token lifetime before the session
// rotates the JWT. The underlying JWT library issues 24h tokens, but the
// session layer treats tokens as expired after this duration and proactively
// refreshes them starting at jwtRefreshWindow before expiry.
const (
	jwtEffectiveLifetime = 1 * time.Hour
	jwtRefreshWindow   = 15 * time.Minute
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
			// H-13: No Origin header in production — reject direct WS connections
			slog.Warn("rejected WebSocket connection without Origin header", "remote_addr", r.RemoteAddr)
			return false
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
	loginLimiter *auth.IPRateLimiter // Rate limiter for login attempts
	doorManager  *systems.DoorManager

	// Per-IP connection tracking (C5)
	ipConnCount map[string]int
	ipConnMu    sync.Mutex

	// Login attempt lockout tracker (H-15)
	loginAttempts *auth.LoginAttemptTracker

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
		sessions:     make(map[string]*Session),
		world:        world,
		combatEngine: ce,
		shopManager:  systems.NewShopManager(),
		loginLimiter:  auth.NewIPRateLimiter(),
		loginAttempts: auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
			Threshold: 10,
			Lockout:   15 * time.Minute,
		}),
		doorManager:  dm,
		ipConnCount:  make(map[string]int),
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
			// Channel full, drop
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

// SetParryDodgeFuncs wires C-11 parry and dodge checks into the combat engine.
func (m *Manager) SetParryDodgeFuncs() {
	m.combatEngine.ParryCheckFunc = func(defenderName string) bool {
		p, ok := m.world.GetPlayer(defenderName)
		if !ok {
			return false
		}
		return game.CheckParry(p)
	}
	m.combatEngine.DodgeCheckFunc = func(defenderName string) bool {
		mob := m.world.GetMobByName(defenderName)
		if mob == nil {
			return false
		}
		return game.CheckNPCDodge(mob)
	}
	// C-10: decrement all player wait states each combat round
	m.combatEngine.OnRoundEnd = func() {
		m.world.ForEachPlayer(func(p *game.Player) {
			p.DecrementWaitState()
		})
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
// If the player is already online, the existing session is forcibly closed
// ("link-dead" takeover, matching original C MUD behavior) and replaced
// with the new session. This prevents players from being locked out when
// their previous connection drops uncleanly and the 60s read-deadline hasn't
// fired yet.
func (m *Manager) Register(playerName string, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if oldSess, exists := m.sessions[playerName]; exists {
		// Notify the old session that it's being taken over, then close it.
		// sendOnce ensures the send channel is closed exactly once, which
		// causes writePump to exit. Closing the conn causes readPump to exit,
		// which calls Unregister — but we've already replaced the session map
		// entry, so the stale Unregister is harmless.
		select {
		case oldSess.send <- []byte("\r\nYour connection has been taken over by a new login.\r\n"):
		default:
			// send buffer full; skip notification rather than block
		}
		oldSess.sendOnce.Do(func() { close(oldSess.send) })
		slog.Info("session takeover", "player", playerName)
	}

	m.sessions[playerName] = s
	s.playerName = playerName
	return nil
}

// Unregister removes a session and saves the player to DB.
// cleanupSession performs all teardown for a session. Idempotent — safe to call
// multiple times for the same session. Both Unregister and UnregisterAndClose
// delegate here to guarantee consistent cleanup ordering.
func (m *Manager) cleanupSession(s *Session, playerName string) {
	// 1. Stop combat
	m.combatEngine.StopCombat(playerName)

	// 2. Broadcast leave message
	if s.player != nil {
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
	}

	// 3. Clean snoop references
	if s.snoopBy != nil {
		s.snoopBy.snooping = nil
	}
	if s.snooping != nil {
		s.snooping.snoopBy = nil
	}

	// 4. Save player to DB
	if m.hasDB && s.player != nil && s.player.ID > 0 {
		if rec, err := db.PlayerToRecord(s.player, nil); err == nil {
			if err := m.db.SavePlayer(rec); err != nil {
				slog.Error("DB save error", "player", playerName, "error", err)
			}
		}
	}

	// 5. Remove from world
	m.world.RemovePlayer(playerName)

	// 6. Close send channel (sync.Once makes this idempotent)
	s.sendOnce.Do(func() { close(s.send) })
}

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

		m.cleanupSession(s, playerName)
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
				slog.Warn("dropping broadcast: channel full",
					"player", name,
					"room", roomVNum,
				)
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

	// H-25: JWT token rotation state
	tokenIssuedAt time.Time // when the current JWT was issued

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
var (
	ErrPlayerAlreadyOnline = fmt.Errorf("player already online")
	ErrNotAuthenticated    = fmt.Errorf("not authenticated")
	ErrUnknownMessageType  = fmt.Errorf("unknown message type")
	ErrInvalidPlayerName   = fmt.Errorf("invalid player name")
	ErrNotInCharCreation   = fmt.Errorf("not in character creation")
)

// Command management methods to implement common.CommandManager interface

// RegisterCommand registers a command handler
