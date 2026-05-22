//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/moderation"
	"golang.org/x/time/rate"
)

// jwtEffectiveLifetime is the effective token lifetime before the session
// rotates the JWT. The underlying JWT library issues 24h tokens, but the
// session layer treats tokens as expired after this duration and proactively
// refreshes them starting at jwtRefreshWindow before expiry.
const (
	jwtEffectiveLifetime = 1 * time.Hour
	jwtRefreshWindow     = 15 * time.Minute
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
			"https://darkpawns.labz0rz.com",
			"https://darkpawns.example.com",
			"https://game.darkpawns.example.com",
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// Allow Docker-internal connections (private IPs) without Origin header.
			// Agent containers connect from 172.x/10.x/192.168.x ranges inside Docker.
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip := net.ParseIP(host); ip != nil && ip.IsPrivate() {
				return true
			}
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

	// Moderation manager for mute/filter/spam checks
	modChecker ModerationChecker

	// Wizlock state — when true, only immortal players may log in
	wizlockMutex sync.Mutex
	wizlocked    bool

	// dreamingDir is the path to the dreaming layer's output directory.
	// Agent memory summaries are read from {dreamingDir}/{agent_id}/memory-summary.txt.
	dreamingDir string

	// decisionLog is the write buffer for decision capture (DP-213).
	// nil when decision capture is disabled.
	decisionLog *db.DecisionLogWriter
}

// ModerationChecker defines the moderation interface the session layer needs.
type ModerationChecker interface {
	// CheckPreCommand is called before every command.
	// Returns (error_message, should_reject). If should_reject, the command is blocked.
	CheckPreCommand(playerName string, command string) (string, bool)

	// CheckMessage filters a communication message for word filters.
	// Returns (filtered_message, should_block).
	CheckMessage(playerName string, message string) (string, bool)

	// RecordMessage records a message for spam detection.
	RecordMessage(playerName string)

	// IsMuted checks if a player is muted.
	IsMuted(playerName string) bool
}

// SetModerationChecker sets the moderation checker on the manager.
func (m *Manager) SetModerationChecker(mc ModerationChecker) {
	m.modChecker = mc
}

// GetModerationChecker returns the current moderation checker.
func (m *Manager) GetModerationChecker() ModerationChecker {
	return m.modChecker
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
		loginLimiter: auth.NewIPRateLimiter(),
		loginAttempts: auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
			Threshold: 10,
			Lockout:   15 * time.Minute,
		}),
		doorManager: dm,
		ipConnCount: make(map[string]int),
	}
	if database != nil {
		m.db = *database
		m.hasDB = true
	}

	// Wire moderation checker (in-memory when no DB, DB-backed when available).
	if database != nil {
		m.modChecker = NewModerationAdapter(moderation.NewManager(database.SQLDB()))
	} else {
		m.modChecker = NewModerationAdapter(moderation.NewManager(nil))
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
			slog.Warn("MessageSink channel full — dropping message", "player", playerName)
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
// GetShopManager returns the session manager's shop manager.
func (m *Manager) GetShopManager() *systems.ShopManager {
	return m.shopManager
}

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
				if err := cmdLook(s, nil); err != nil {
					slog.Error("cmdLook failed after death", "player", victim.GetName(), "error", err)
				}
			}
			return
		}

		// Auto-loot: if killer has autoloot enabled, loot the corpse
		if killer != nil && !killer.IsNPC() && IsAutoLootEnabled(killer.GetName()) {
			if player, ok := m.world.GetPlayer(killer.GetName()); ok {
				// Use doGet to transfer items from corpse to player inventory
				items := m.world.GetItemsInRoom(victim.GetRoom())
				for _, item := range items {
					if item.IsCorpse && len(item.Contains) > 0 {
						m.world.DoGet(player, "all "+item.GetShortDesc())
						break
					}
				}
				// Refresh killer's UI
				if s, ok := m.GetSession(killer.GetName()); ok {
					s.markDirty(VarInventory, VarRoomItems)
				}
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

// SetScriptDeathFunc wires the death trigger into the combat engine.
// When a mob dies, if it has a death script, it fires.
func (m *Manager) SetScriptDeathFunc() {
	m.combatEngine.ScriptDeathFunc = func(victimName string, killerName string, roomVNum int) {
		m.world.FireMobDeathScript(victimName, killerName, roomVNum)
	}
}

// OnRoundEnd decrements all player wait states each combat round.
// Set after combat engine initialization.
func (m *Manager) SetOnRoundEnd() {
	m.combatEngine.OnRoundEnd = func() {
		m.world.ForEachPlayer(func(p *game.Player) {
			p.DecrementWaitState()
		})
	}
}

// SetCommandExecFunc wires the session layer's command dispatch into the game
// world so that doOrder can execute commands on charmed followers.
func (m *Manager) SetCommandExecFunc() {
	m.world.CommandExecFunc = func(ch *game.Player, command string) bool {
		 sess, ok := m.GetSession(ch.GetName())
		if !ok || sess == nil {
			return false
		}
		if err := ExecuteCommand(sess, command, nil); err != nil {
			slog.Error("order command failed", "player", ch.GetName(), "command", command, "error", err)
			return false
		}
		return true
	}
}

// SetDreamingDir sets the path to the dreaming layer's output directory.
// Agent memory summaries are read from {dir}/{agent_id}/memory-summary.txt.
func (m *Manager) SetDreamingDir(dir string) {
	m.dreamingDir = dir
}

// SetDecisionLog enables decision capture with the given writer.
func (m *Manager) SetDecisionLog(dlw *db.DecisionLogWriter) {
	m.decisionLog = dlw
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
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections from your IP"))
		_ = conn.Close()
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
// Register associates a player name with a session. It is safe to call
// concurrently — m.mu serialises session-map mutations.
//
// LOCK ORDERING: Register must NEVER call methods that acquire w.mu while
// holding m.mu. This caused a deadlock (LRN-20260519-001): RemovePlayer
// acquires w.mu, so it must be called AFTER releasing m.mu. Rule:
//
//   m.mu  →  w.mu  = FORBIDDEN (causes deadlock)
//   w.mu  →  m.mu  = OK (never happens in practice)
//
// If you need both locks, always acquire w.mu first.
func (m *Manager) Register(playerName string, s *Session) error {
	// RemovePlayer (called below for takeover) acquires w.mu. To avoid the
	// m.mu → w.mu lock ordering that caused a deadlock during char creation,
	// we record whether removal is needed under m.mu, release m.mu, then call
	// RemovePlayer separately so the two locks are never held simultaneously.
	var needsWorldRemove bool

	m.mu.Lock()
	if oldSess, exists := m.sessions[playerName]; exists {
		// DP-GOAT P0-3: Session handoff grace period
		// Give the old session a brief window to prove it's alive before
		// forcible takeover. During this window the old session's readPump
		// can clear takeOverPending by handling an incoming message.
		if s.isAgent && oldSess.isAgent {
			// Agent-to-agent: wait for old session to respond or timeout
			oldSess.takeOverPending = true
			oldSess.takeOverAt = time.Now().Add(5 * time.Second)
			select {
			case oldSess.send <- []byte("\r\n*** New connection detected. Send any command within 5 seconds to keep this session. ***\r\n"):
			default:
			}

			// Poll for old session to clear takeOverPending or timeout
			for time.Now().Before(oldSess.takeOverAt) {
				m.mu.Unlock()
				time.Sleep(200 * time.Millisecond)
				m.mu.Lock()
				// Re-acquire oldSess reference (it may have been removed)
				oldSess, exists = m.sessions[playerName]
				if !exists || !oldSess.takeOverPending {
					// Session responded or disconnected — cancel new login
					m.mu.Unlock()
					return fmt.Errorf("player %s is already online and active", playerName)
				}
			}
			// Timeout — proceed with takeover
			oldSess.takeOverPending = false
		}

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

		needsWorldRemove = oldSess.player != nil
		slog.Info("session takeover", "player", playerName)
	}

	m.sessions[playerName] = s
	s.playerName = playerName
	m.mu.Unlock()

	// Remove the old player from the world AFTER releasing m.mu.
	// This call acquires w.mu independently; no nested locking with m.mu.
	// The caller (completeCharCreation / handleLogin) will call AddPlayer
	// after Register returns, so the window where the player is absent from
	// the world is intentional and bounded.
	if needsWorldRemove {
		m.world.RemovePlayer(playerName)
	}

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

	// 3b. M-16: Auto-return from switched body on disconnect
	if s.isSwitched {
		if s.switchedOriginal != nil {
			slog.Warn("auto-return on disconnect",
				"wizard", s.switchedOriginal.Name,
				"from_mob", s.switchedMob != nil,
				"from_player", s.switchedPlayer != nil,
			)
			s.player = s.switchedOriginal
		}
		s.isSwitched = false
		s.switchedOriginal = nil
		s.switchedOriginalLevel = 0
		s.switchedMob = nil
		s.switchedPlayer = nil
	}

	// 3c. Cancel any in-progress mail writing
	if s.player != nil {
		game.CancelMailWriting(s.player.ID)
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
	connCountDecremented bool // C5: prevents double-decrement of IP connection count

	// Agent identity — set on login when is_agent=true.
	// Harness+Model is the agent identity. Same combo = same agent across sessions.
	isAgent      bool
	agentHarness string    // e.g. "openclaw", "claude-code"
	agentModel   string    // e.g. "mimo-v2.5-base"
	agentVersion string    // harness version
	agentKeyID   int64     // legacy: kept for backward compat, deprecated
	connectedAt  time.Time // set on session creation, used for sessionID()

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
	charStage    string // current stage in creation flow (color, sex, race, class, hometown, stats_roll)
	charName     string
	charPassword string // hashed password during creation
	charColor    bool   // ANSI color preference
	charSex      int
	charRace     int
	charClass    int
	charHometown int
	charStats    game.CharStats

	// Character switch state (wizard commands)
	isSwitched            bool
	switchedOriginal      *game.Player
	switchedOriginalLevel int       // M-16: wizard's real level for permission gating
	switchedStartTime     time.Time // M-16: when switch began
	switchedMob           *game.MobInstance
	switchedPlayer        *game.Player

	// Rate limit: capacity=10, refill=10/sec (token bucket via golang.org/x/time/rate)
	// This protects the server from command floods — it does NOT protect API costs.
	// Agents must implement their own circuit breakers for LLM-level loop detection.
	// See scripts/dp_bot.py for reference implementation.
	limiter *rate.Limiter

	// Decision capture: incremented per command for turn_number in decision log
	commandCount int

	// Temporary data storage for command handlers
	tempData map[string]interface{}

	// Infobar / display state (from act.display.c)
	screenSize  int //nolint:unused // terminal height in lines; 0 = unset (defaults to 25)
	infobarMode int //nolint:unused // InfobarOff (0) or InfobarOn (1)

	// Communication state
	lastTeller string   // Last player who told us (for reply)
	snooping   *Session // Session being snooped (for wizard snoop)
	snoopBy    *Session // Session that is snooping us

	// DP-GOAT P0-3: Session handoff grace period
	// When a new agent login arrives for a character that already has a session,
	// takeOverPending is set to true and takeOverAt marks the deadline. The old
	// session's readPump clears takeOverPending on any incoming message to prove
	// it's still alive. If it doesn't respond in time, the new login takes over.
	takeOverPending bool
	takeOverAt      time.Time

	// Force-command safety state
	IsForced             bool      // true while executing a forced command (prevents transitive force)
	ForcedPrivilegeLevel int       // target's privilege level during forced execution (0 = not forced)
	LastForceTime        time.Time // last time this session was the target of a force command

	// idleTicsSet tracks whether the idle timeout counter has been set
	// for pre-login sessions. Used by CheckIdlePasswords().
	idleTicsSet bool

	// sendOnce ensures s.send is closed exactly once across all disconnect paths.
	sendOnce sync.Once

	// msgSeq is a monotonically incrementing sequence number stamped on every
	// outbound message. Used by the dp-goat daemon for event tracking and
	// reconnection replay. Zero is never sent (first message gets seq=1).
	msgSeq uint64
}

// LiveAgentSession is already defined in admin package.
// GetLiveAgentSessions returns info about all active agent sessions.
// Implements admin.LiveSessionProvider interface.
// Safe to call concurrently.
func (m *Manager) GetLiveAgentSessions() []admin.LiveAgentSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]admin.LiveAgentSession, 0)
	for _, s := range m.sessions {
		if s.isAgent {
			info := admin.LiveAgentSession{
				PlayerName:  s.playerName,
				Harness:     s.agentHarness,
				Model:       s.agentModel,
				Version:     s.agentVersion,
				ConnectedAt: s.connectedAt.Format(time.RFC3339),
			}
			if s.player != nil {
				info.RoomVNum = s.player.GetRoom()
				info.Level = s.player.GetLevel()
			}
			sessions = append(sessions, info)
		}
	}
	return sessions
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
