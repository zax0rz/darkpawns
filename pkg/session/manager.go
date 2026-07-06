// Package session manages WebSocket connections and player sessions.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/events"
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

	// Linkdead reaper thresholds. Structurally faithful to C check_idling()
	// (limits.c IDLE_TO_VOID / IDLE_DISCONNECT), but using wall-clock time
	// so dead TCP sockets are detected and cleaned up quickly.
	linkdeadVoidThreshold      = 60 * time.Second
	linkdeadExtractThreshold   = 5 * time.Minute
	linkdeadVoidRoomVNum       = 1
	linkdeadDisconnectRoomVNum = 3
)

// allowedWebSocketOrigins lists the public origins that may connect without
// presenting an agent key.
var allowedWebSocketOrigins = []string{
	"https://darkpawns.labz0rz.com",
}

// agentKeyHeaderNames and agentKeyQueryParams name where an agent may present
// its API key during the WebSocket handshake. These are configurable for
// deployments where a reverse proxy strips or rewrites headers.
var (
	agentKeyHeaderNames = []string{"X-Agent-Key"}
	agentKeyQueryParams = []string{"agent_key"}
)

func init() {
	if v := os.Getenv("AGENT_KEY_HEADER"); v != "" {
		agentKeyHeaderNames = strings.Split(v, ",")
	}
	if v := os.Getenv("AGENT_KEY_QUERY_PARAMS"); v != "" {
		agentKeyQueryParams = strings.Split(v, ",")
	}
}

// Manager handles all active sessions.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session // keyed by player name
	world        *game.World
	combatEngine *combat.CombatEngine
	shopManager  *systems.ShopManager
	db           db.Database
	hasDB        bool
	loginLimiter *auth.IPRateLimiter // Rate limiter for login attempts
	doorManager  *systems.DoorManager
	upgrader     websocket.Upgrader

	// Per-IP connection tracking (C5)
	ipConnCount map[string]int
	ipConnMu    sync.Mutex

	// Login attempt lockout tracker (H-15)
	loginAttempts *auth.LoginAttemptTracker

	// Account-level login lockout tracker (DP-592)
	accountLockouts *auth.AccountLockoutTracker

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

// checkOrigin validates WebSocket origins. Public origins in the allowlist are
// permitted without further credentials. Connections from private IPs with no
// Origin header must present a valid agent API key (DP-594).
func (m *Manager) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsPrivate() {
			slog.Warn("rejected WebSocket connection without Origin header", "remote_addr", r.RemoteAddr)
			return false
		}

		// Private IP without Origin: require a valid agent key.
		key := findAgentKey(r)
		if key == "" {
			slog.Warn("rejected private WebSocket connection without agent key", "remote_addr", r.RemoteAddr)
			return false
		}
		if m.db == nil {
			slog.Warn("rejected private WebSocket connection: no database to validate agent key", "remote_addr", r.RemoteAddr)
			return false
		}
		if _, _, valid := m.db.ValidateAgentKey(key); !valid {
			slog.Warn("rejected private WebSocket connection: invalid agent key", "remote_addr", r.RemoteAddr)
			return false
		}
		return true
	}

	for _, allowed := range allowedWebSocketOrigins {
		if origin == allowed {
			return true
		}
	}

	// Dev mode: allow any origin from localhost/127.0.0.1 regardless of port
	// or scheme, so local smoke tests and agent harnesses connect without 403.
	if os.Getenv("ENVIRONMENT") == "development" && origin != "" {
		if u, err := url.Parse(origin); err == nil {
			h := strings.ToLower(u.Hostname())
			if h == "localhost" || h == "127.0.0.1" {
				return true
			}
		}
	}

	slog.Warn("rejected WebSocket connection from unauthorized origin", "origin", origin) // #nosec G706
	return false
}

// findAgentKey returns the first non-empty agent key from configured headers
// or query parameters.
func findAgentKey(r *http.Request) string {
	for _, h := range agentKeyHeaderNames {
		if v := r.Header.Get(strings.TrimSpace(h)); v != "" {
			return v
		}
	}
	for _, p := range agentKeyQueryParams {
		if v := r.URL.Query().Get(strings.TrimSpace(p)); v != "" {
			return v
		}
	}
	return ""
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
func NewManager(world *game.World, database db.Database) *Manager {
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
	// Guard against the typed-nil interface trap: a nil *db.DB stored in a
	// db.Database interface is itself non-nil. Normalize it to a real nil so
	// the no-database path below is taken instead of dereferencing nil. (DP-589)
	if concreteDB, ok := database.(*db.DB); ok && concreteDB == nil {
		database = nil
	}

	if database != nil {
		m.accountLockouts = auth.NewAccountLockoutTracker(database, auth.AccountLockoutConfig{
			Threshold: 10,
			Lockout:   15 * time.Minute,
		})
	}

	if database != nil {
		m.db = database
		m.hasDB = true
	}

	m.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     m.checkOrigin,
	}

	// Wire moderation checker (in-memory when no DB, DB-backed when available).
	if concreteDB, ok := database.(*db.DB); ok {
		m.modChecker = NewModerationAdapter(moderation.NewManager(concreteDB.SQLDB()))
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
		m.mu.RLock()
		defer m.mu.RUnlock()
		for sessName := range m.sessions {
			if strings.EqualFold(sessName, name) {
				return true
			}
		}
		return false
	}

	// Load ban list and invalid name list at startup
	if err := game.LoadBanned(); err != nil {
		slog.Warn("Failed to load ban list", "error", err)
	}
	if err := game.ReadInvalidList(); err != nil {
		slog.Warn("Failed to load invalid name list", "error", err)
	}

	// Subscribe to PlayerLeveledEvent to trigger Discord level-up notifications
	world.Events.Subscribe(events.PlayerLeveledEvent{}.Type(), func(ctx context.Context, ev events.BusEvent) error {
		if ple, ok := ev.(events.PlayerLeveledEvent); ok {
			isGuest := false
			if s, ok := m.GetSession(ple.PlayerID); ok {
				isGuest = s.isGuest
			}
			if !isGuest {
				sendDiscordNotification(fmt.Sprintf("_**%s has attained circle %d!**_", ple.PlayerID, ple.NewLevel))
			}
		}
		return nil
	})

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

// SetCombatMessageFunc wires the golden-tested DamMessage() path into the live
// combat engine. It builds a combat.GameCallbacks struct with the game-layer
// broadcast/send implementations, wires it into the engine, and keeps the legacy
// package-level hooks as aliases so existing tests continue to pass during the
// multi-PR migration.
// WireCombatCallbacks wires the game-layer character-state callbacks into the
// combat engine. Called once during initialization before SetCombatMessageFunc.
func (m *Manager) WireCombatCallbacks() {
	cb := m.world.WireCombatCallbacks()
	m.combatEngine.SetCallbacks(cb)
}

func (m *Manager) SetCombatMessageFunc() {
	wrap := func(text string) []byte {
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "combat",
				Text: text,
			},
		})
		if err != nil {
			slog.Error("json.Marshal error in combat message", "error", err)
			return nil
		}
		return msg
	}

	broadcast := func(roomVNum int, message string, exclude string) {
		msg := wrap(message)
		if msg == nil {
			return
		}
		excluded := make(map[string]bool)
		for _, name := range strings.Fields(exclude) {
			excluded[name] = true
		}
		m.mu.RLock()
		defer m.mu.RUnlock()
		for name, s := range m.sessions {
			if excluded[name] {
				continue
			}
			if s.player != nil && s.player.GetRoom() == roomVNum {
				select {
				case s.send <- msg:
				default:
					slog.Warn("dropping combat broadcast: channel full", "player", name, "room", roomVNum)
				}
			}
		}
	}

	sendToChar := func(name string, message string) {
		msg := wrap(message)
		if msg == nil {
			return
		}
		if s, ok := m.GetSession(name); ok {
			select {
			case s.send <- msg:
			default:
				slog.Warn("dropping combat message: channel full", "player", name)
			}
		}
	}

	// Reuse the callbacks struct wired by WireCombatCallbacks so PR2 character
	// state hooks are preserved. If no struct exists yet, create one.
	cb := m.combatEngine.Callbacks
	if cb == nil {
		cb = &combat.GameCallbacks{}
	}
	cb.Broadcast = broadcast
	cb.SendToChar = sendToChar
	combat.InitSkillMessages(cb)
	m.combatEngine.SetCallbacks(cb)

	m.combatEngine.MessageFunc = func(attacker, defender combat.Combatant, dam, attackType int) bool {
		combat.DamMessage(dam, attacker, defender, attackType)
		return true
	}
}

// GetCombatEngine returns the combat engine for AI integration.
// GetShopManager returns the session manager's shop manager.
func (m *Manager) GetShopManager() *systems.ShopManager {
	return m.shopManager
}

func (m *Manager) GetCombatEngine() *combat.CombatEngine {
	return m.combatEngine
}

// GetBanManager returns the ban manager for checking host bans.
func (m *Manager) GetBanManager() *game.BanManager {
	return m.world.Bans
}

// SetDeathFunc wires the game-layer death handler into the combat engine.
func (m *Manager) SetDeathFunc() {
	m.combatEngine.DeathFunc = func(victim, killer combat.Combatant, attackType int) {
		m.world.HandleDeath(victim, killer, attackType)

		// If victim was a player, send updated room state after respawn
		if !victim.IsNPC() {
			isGuest := false
			if s, ok := m.GetSession(victim.GetName()); ok {
				isGuest = s.isGuest
				if err := cmdLook(s, nil); err != nil {
					slog.Error("cmdLook failed after death", "player", victim.GetName(), "error", err)
				}
			}
			if !isGuest {
				if killer != nil && killer.GetName() != "" {
					sendDiscordNotification(fmt.Sprintf("_**%s was slain by %s!**_", victim.GetName(), killer.GetName()))
				} else {
					sendDiscordNotification(fmt.Sprintf("_**%s has met their demise!**_", victim.GetName()))
				}
			}
			return
		}

		// Auto-loot: if killer has autoloot enabled, loot the corpse
		if killer != nil && !killer.IsNPC() {
			if player, ok := m.world.GetPlayer(killer.GetName()); ok && IsAutoLootEnabled(player) {
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
			s.flushDirtyVars()
		}

		// Proactively find any player session fighting this victim to update their target display in real-time
		m.mu.RLock()
		sessions := make([]*Session, 0, len(m.sessions))
		for _, s := range m.sessions {
			if s.wantsStructuredData || s.isAgent {
				sessions = append(sessions, s)
			}
		}
		m.mu.RUnlock()

		for _, s := range sessions {
			if target, fighting := m.combatEngine.GetCombatTarget(s.playerName); fighting && target.GetName() == victimName {
				s.markDirty(VarFighting)
				s.flushDirtyVars()
			}
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

// SetFleeHooks wires wimpy auto-flee into the combat engine (DP-389).
// Must be called after WireCombatCallbacks() so the GameCallbacks struct exists.
func (m *Manager) SetFleeHooks() {
	cb := m.combatEngine.Callbacks
	if cb == nil {
		cb = &combat.GameCallbacks{}
		m.combatEngine.SetCallbacks(cb)
	}
	cb.DoFlee = func(name string) {
		s, ok := m.GetSession(name)
		if !ok || s == nil {
			return
		}
		if err := cmdFlee(s); err != nil {
			slog.Error("DoFlee failed", "player", name, "error", err)
		}
	}
	cb.DoRetreat = func(name string) {
		s, ok := m.GetSession(name)
		if !ok || s == nil {
			return
		}
		if err := cmdFlee(s); err != nil {
			slog.Error("DoRetreat failed", "player", name, "error", err)
		}
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
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}

	ip := auth.GetIPFromRequest(r)

	// Ban check: BanAll → reject before creating a session (DP-418)
	ipBanLevel := 0
	if bm := m.GetBanManager(); bm != nil {
		ipBanLevel = bm.IsBanned(ip)
	}
	if ipBanLevel == game.BanAll {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "your site has been banned"))
		_ = conn.Close()
		slog.Warn("WebSocket: BanAll connection rejected", "ip", ip)
		return
	}

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

	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		banLevel:            ipBanLevel, // BanNew/BanSelect enforced at login (DP-418)
		conn:                conn,
		request:             r, // Store the HTTP request for IP extraction
		manager:             m,
		send:                make(chan []byte, 256),
		limiter:             rate.NewLimiter(rate.Limit(10), 10),
		subscribedVars:      make(map[string]bool),
		dirtyVars:           make(map[string]bool),
		pendingEvents:       nil,
		connectedAt:         time.Now(),
		wantsStructuredData: true,
		sessionCtx:          ctx,
		cancelFunc:          cancel,
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
//	m.mu  →  w.mu  = FORBIDDEN (causes deadlock)
//	w.mu  →  m.mu  = OK (never happens in practice)
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
			oldSess.takeOverPending.Store(true)
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
				if !exists || !oldSess.takeOverPending.Load() {
					// Session responded or disconnected — cancel new login
					m.mu.Unlock()
					return fmt.Errorf("player %s is already online and active", playerName)
				}
			}
			// Timeout — proceed with takeover
			oldSess.takeOverPending.Store(false)
		}

		// Notify the old session that it's being taken over, then close it.
		// sendOnce ensures the send channel is closed exactly once, which
		// causes writePump to exit. Closing the conn causes readPump to exit,
		// which calls Unregister — but we've already replaced the session map
		// entry, so the stale Unregister is harmless.
		oldSess.sendMu.RLock()
		if !oldSess.sendClosed {
			select {
			case oldSess.send <- []byte("\r\nYour connection has been taken over by a new login.\r\n"):
			default:
				// send buffer full; skip notification rather than block
			}
		}
		oldSess.sendMu.RUnlock()
		oldSess.CloseSend()

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

	if s.authenticated && !s.isGuest {
		sendDiscordNotification(fmt.Sprintf("_**%s has stepped into the shadows of the world.**_", playerName))
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

	if s.authenticated && !s.isGuest {
		sendDiscordNotification(fmt.Sprintf("_**%s has retreated back into the mist.**_", playerName))
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
			slog.Warn(
				"auto-return on disconnect",
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
	if m.hasDB && s.player != nil && s.player.ID > 0 && !s.isGuest {
		if rec, err := db.PlayerToRecord(s.player, nil); err == nil {
			if err := m.db.SavePlayer(rec); err != nil {
				slog.Error("DB save error", "player", playerName, "error", err)
			}
		}
	}

	// 5. Remove from world
	m.world.RemovePlayer(playerName)

	// 6. Close send channel (guarded + sync.Once makes this idempotent)
	s.CloseSend()

	// 7. Cancel session context if defined
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
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

// ReapLinkdeadSessions checks for authenticated sessions that have not sent an
// inbound message in linkdeadVoidThreshold or linkdeadExtractThreshold. It
// mirrors the C check_idling() two-stage behaviour, but uses wall-clock time
// so dead TCP sockets are cleaned up quickly (DP-902).
//
// Stage 1 (>60s): move the player to the void room (vnum 1) and remember the
// original room in WasInRoom.
// Stage 2 (>5m): move the player to the disconnect room (vnum 3), save, and
// close the WebSocket — this triggers readPump/writePump exit → Unregister.
func (m *Manager) ReapLinkdeadSessions() {
	m.mu.RLock()
	var toVoid []*Session
	var toExtract []*Session
	now := time.Now().UnixNano()
	for _, s := range m.sessions {
		if !s.authenticated || s.player == nil {
			continue
		}
		last := s.lastActive.Load()
		if last == 0 {
			continue
		}
		elapsed := time.Duration(now - last)
		if elapsed > linkdeadExtractThreshold {
			toExtract = append(toExtract, s)
		} else if elapsed > linkdeadVoidThreshold {
			toVoid = append(toVoid, s)
		}
	}
	m.mu.RUnlock()

	for _, s := range toVoid {
		s.moveToVoid()
	}
	for _, s := range toExtract {
		s.extractLinkdead()
	}
}

// moveToVoid moves a linkdead player to the void room (vnum 1), remembering
// the original room so they can be returned when they send a command.
// Equivalent to the first branch of C check_idling() (limits.c:426-437).
func (s *Session) moveToVoid() {
	p := s.player
	if p == nil {
		return
	}

	// Re-check activity: the session may have become active while we were
	// iterating under the read lock.
	if time.Since(time.Unix(0, s.lastActive.Load())) <= linkdeadVoidThreshold {
		return
	}

	wasIn := p.GetWasInRoom()
	roomVNum := p.GetRoom()
	level := p.GetLevel()

	// Only mortal players who are not already voided.
	if level >= game.LVL_IMMORT || wasIn != 0 || roomVNum <= 0 || roomVNum == linkdeadVoidRoomVNum {
		return
	}

	p.SetWasInRoom(roomVNum)

	if err := s.manager.world.PlayerTransfer(p, linkdeadVoidRoomVNum); err != nil {
		slog.Warn("linkdead reaper: PlayerTransfer to void failed", "player", s.playerName, "error", err)
		return
	}

	s.sendText("You have been idle, and are pulled into a void.\r\n")
	s.manager.world.SendToRoom(roomVNum, fmt.Sprintf("%s disappears into the void.\r\n", p.Name))
	slog.Info("linkdead reaper: moved to void", "player", s.playerName, "room", roomVNum)
}

// extractLinkdead moves a long-idle player to the disconnect room (vnum 3),
// saves them, and closes the WebSocket so the pump defers run Unregister.
// Equivalent to the second branch of C check_idling() (limits.c:438-451).
func (s *Session) extractLinkdead() {
	p := s.player
	playerName := s.playerName
	if p == nil || playerName == "" {
		return
	}

	// Re-check activity: the session may have become active while we were
	// iterating under the read lock.
	if time.Since(time.Unix(0, s.lastActive.Load())) <= linkdeadExtractThreshold {
		return
	}

	elapsed := time.Since(time.Unix(0, s.lastActive.Load()))
	slog.Warn(
		"reaping linkdead session",
		"player", playerName,
		"idle", elapsed.Round(time.Second),
	)

	if err := s.manager.world.PlayerTransfer(p, linkdeadDisconnectRoomVNum); err != nil {
		slog.Warn("linkdead reaper: PlayerTransfer to disconnect room failed", "player", playerName, "error", err)
	}

	// Save before closing the connection.
	if s.manager.hasDB && p.ID > 0 && !s.isGuest {
		if rec, err := db.PlayerToRecord(p, nil); err == nil {
			if err := s.manager.db.SavePlayer(rec); err != nil {
				slog.Error("linkdead reaper: DB save error", "player", playerName, "error", err)
			}
		}
	}

	// Close the underlying connection. For WebSocket this triggers the pump
	// defers to run Unregister; for telnet and other transports there is no
	// pump defer, so we Unregister directly after closing.
	s.Close()
	if s.conn == nil {
		s.manager.Unregister(playerName)
	}
}

// GetSession returns a session by player name.
func (m *Manager) GetSession(playerName string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[playerName]
	return s, ok
}

// SessionCount returns the number of active player sessions.
func (m *Manager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
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
				slog.Warn(
					"dropping broadcast: channel full",
					"player", name,
					"room", roomVNum,
				)
			}
		}
	}
}

// Session represents a single WebSocket connection.
type Session struct {
	conn                 *websocket.Conn
	request              *http.Request // Store the original HTTP request for IP extraction
	remoteIP             string        // Store IP directly for non-HTTP (Telnet) sessions
	manager              *Manager
	send                 chan []byte
	player               *game.Player
	playerName           string
	authenticated        bool
	isGuest              bool
	connCountDecremented bool // C5: prevents double-decrement of IP connection count
	banLevel             int  // ban level from IsBanned (BanNew or BanSelect); 0 = no ban

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
	agentMu             sync.Mutex
	subscribedVars      map[string]bool // vars this session subscribed to
	dirtyVars           map[string]bool // vars changed since last flush
	pendingEvents       []interface{}   // queued EVENTS since last flush
	wantsStructuredData bool

	// Character creation state
	charCreating         bool
	charStage            string // current stage in creation flow (color, sex, race, class, hometown, stats_roll)
	charName             string
	charPassword         string // hashed password during creation
	charPasswordSupplied bool   // auth layer already collected the password; nanny skips its prompt (DP-909)
	charColor            bool   // ANSI color preference
	charSex              int
	charRace             int
	charClass            int
	charHometown         int
	charStats            game.CharStats

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
	//
	// Atomic to avoid data race between Register (m.mu) and readPump (no lock).
	takeOverPending atomic.Bool
	takeOverAt      time.Time

	// lastActive is the Unix-nano timestamp of the most recent inbound message.
	// Updated by readPump on every successful ReadMessage and used by the
	// linkdead reaper to detect dead TCP sockets (DP-902).
	lastActive atomic.Int64

	// Force-command safety state
	IsForced             bool      // true while executing a forced command (prevents transitive force)
	ForcedPrivilegeLevel int       // target's privilege level during forced execution (0 = not forced)
	LastForceTime        time.Time // last time this session was the target of a force command

	// idleTicsSet tracks whether the idle timeout counter has been set
	// for pre-login sessions. Used by CheckIdlePasswords().
	idleTicsSet bool

	// sendOnce ensures s.send is closed exactly once across all disconnect paths.
	sendOnce sync.Once

	// sendMu makes sends on s.send mutually exclusive with closing it. A send on
	// a closed channel panics even inside a select, so SendMessage takes RLock
	// (concurrent sends are safe) and the close path takes the exclusive Lock and
	// flips sendClosed. This prevents a use-after-close panic when a caller holds
	// a session reference across a concurrent disconnect (e.g. admin kick).
	// sendMu is a leaf lock: never acquire another lock while holding it.
	sendMu     sync.RWMutex
	sendClosed bool

	// msgSeq is a monotonically incrementing sequence number stamped on every
	// outbound message. Used by the dp-goat daemon for event tracking and
	// reconnection replay. Zero is never sent (first message gets seq=1).
	msgSeq uint64

	// sessionCtx is the long-running connection context.
	sessionCtx context.Context
	// cancelFunc triggers cancellation of the entire session context on disconnect.
	cancelFunc context.CancelFunc

	// closeFunc is an optional transport-specific close callback. WebSocket
	// sessions use s.conn directly; telnet/other transports set this so that
	// s.Close() can tear down the underlying connection (DP-928).
	closeFunc func()
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

// SetWantsStructuredData sets whether this session receives structured updates.
func (s *Session) SetWantsStructuredData(val bool) {
	s.agentMu.Lock()
	alreadySet := s.wantsStructuredData
	s.wantsStructuredData = val
	if val && !alreadySet {
		// Automatically subscribe to all standard variables for structured clients
		for _, v := range AllVariables {
			if v != VarEvents {
				s.subscribedVars[v] = true
			}
		}
	}
	s.agentMu.Unlock()

	// Send initial dump if newly enabled and the session is authenticated
	if val && !alreadySet && s.IsAuthenticated() {
		s.sendFullVarDump()
	}
}

// WantsStructuredData returns whether this session receives structured updates.
func (s *Session) WantsStructuredData() bool {
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	return s.wantsStructuredData
}

// ShutdownGracefully drains and shuts down all active sessions gracefully.
func (m *Manager) ShutdownGracefully(timeout time.Duration) {
	m.mu.Lock()
	sessions := make(map[string]*Session)
	for name, s := range m.sessions {
		sessions[name] = s
		delete(m.sessions, name)
	}
	m.mu.Unlock()

	if len(sessions) == 0 {
		return
	}

	slog.Info("session manager: shutting down active sessions", "count", len(sessions))

	// 1. Notify players of the shutdown
	for _, s := range sessions {
		s.sendText("\r\n\x1b[1;31m!! The MUD server is performing a graceful shutdown for maintenance. Your state has been saved. !!\x1b[0m\r\n")
	}

	// 2. Wait a brief moment for messages to flush
	time.Sleep(1 * time.Second)

	// 3. Unregister and cleanup each session sequentially with safety timeouts
	for name, s := range sessions {
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error(
						"CRITICAL PANIC in session shutdown cleanup",
						"player", name,
						"recover", r,
					)
				}
				close(done)
			}()
			m.cleanupSession(s, name)
		}()

		select {
		case <-done:
			// Cleanup completed successfully
		case <-time.After(1 * time.Second):
			slog.Error("session cleanup timed out during graceful shutdown", "player", name)
			if s.cancelFunc != nil {
				s.cancelFunc()
			}
		}
	}
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
