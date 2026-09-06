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
	snoopMu      sync.RWMutex        // protects the bidirectional snoop links
	sessions     map[string]*Session // keyed by player name
	world        *game.World
	combatEngine *combat.CombatEngine
	shopManager  *game.ShopManager
	pulsePumpMu  sync.RWMutex
	pulsePump    func(int) error
	db           db.Database
	hasDB        bool
	loginLimiter *auth.IPRateLimiter // Rate limiter for login attempts
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
	wizlockLevel int

	// dreamingDir is the path to the dreaming layer's output directory.
	// Agent memory summaries are read from {dreamingDir}/{agent_id}/memory-summary.txt.
	dreamingDir string

	// decisionLog is the write buffer for decision capture (DP-213).
	// nil when decision capture is disabled.
	decisionLog *db.DecisionLogWriter

	// godCrowned is the in-process latch for the first-player-God bootstrap
	// (init_char, db.c:3016). Under DP_FRESH_MUD (the oracle harness path), the
	// MUD is treated as fresh and exactly ONE character per process is crowned —
	// godCrowned flips true on the first crown so subsequent chars are ordinary
	// mortals, matching C's "first player ever" semantics within a single boot.
	// Production uses the persisted CountPlayers()==0 check instead.
	godCrowned atomic.Bool

	// nextConnectionNumber mirrors C's last_desc counter for the dc command.
	// It is protected by m.mu and wraps from 999 back to 1.
	nextConnectionNumber int

	// shutdownRequests carries C do_shutdown's process-level request to the
	// server entrypoint after the command has emitted its player-facing bytes.
	shutdownRequests chan ShutdownRequest
}

// ShutdownRequest describes a C do_shutdown option. Marker is written beside
// the server working directory by cmd/server for reboot/die/pause semantics.
type ShutdownRequest struct {
	Marker string
}

// isLoopback reports whether remoteAddr resolves to the loopback interface
// (127.0.0.0/8 or ::1). It tolerates missing ports.
func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// checkOrigin validates WebSocket origins. Public origins in the allowlist are
// permitted without further credentials. Machine-local connections are always
// trusted. Connections from private IPs with no Origin header must present a
// valid agent API key (DP-594).
func (m *Manager) checkOrigin(r *http.Request) bool {
	// Machine-local connections are always trusted; this covers CI smoke tests
	// and local agent harnesses regardless of what Origin header they send.
	if isLoopback(r.RemoteAddr) {
		return true
	}

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

	// Allow any origin from localhost/127.0.0.1 regardless of port or scheme.
	if u, err := url.Parse(origin); err == nil {
		h := strings.ToLower(u.Hostname())
		if h == "localhost" || h == "127.0.0.1" {
			return true
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

	// House-control uses the C get_name_by_id()/get_id_by_name() seam. Keep it
	// backed by the live world for both the admin command and ordinary house
	// commands; leaving the injected callbacks nil makes every build/list path
	// diverge even when the player is online.
	game.RegisterHousePlayerLookup(
		func(id int64) string {
			player := world.GetPlayerByID(int(id))
			if player == nil {
				return ""
			}
			return player.GetName()
		},
		func(name string) int64 {
			for _, player := range world.GetPlayers() {
				if strings.EqualFold(player.GetName(), name) {
					return int64(player.GetID())
				}
			}
			return -1
		},
	)

	shopManager := game.NewShopManager()
	if existing, ok := world.GetShopManager().(*game.ShopManager); ok && existing != nil {
		shopManager = existing
	}
	m := &Manager{
		sessions:         make(map[string]*Session),
		world:            world,
		combatEngine:     ce,
		shopManager:      shopManager,
		shutdownRequests: make(chan ShutdownRequest, 1),
		loginLimiter:     auth.NewIPRateLimiter(),
		loginAttempts: auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
			Threshold: 10,
			Lockout:   15 * time.Minute,
		}),
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
		s.notePlayerOutput()
		s.forwardSnoopOutput(string(msg))
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

	// C's do_simple_move calls look_at_room before follower recursion. Keep the
	// renderer in session while letting the game transaction own that ordering.
	world.MovementLook = func(player *game.Player) {
		s, ok := m.GetSession(player.Name)
		if !ok || s == nil {
			return
		}
		if err := cmdMovementLook(s); err != nil {
			slog.Error("movement look failed", "player", player.Name, "error", err)
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

// allocateConnectionNumber mirrors C's last_desc counter. The value is
// assigned when the transport session is created, before character login.
func (m *Manager) allocateConnectionNumber() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextConnectionNumber++
	if m.nextConnectionNumber == 1000 {
		m.nextConnectionNumber = 1
	}
	return m.nextConnectionNumber
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
	// enqueueCombatMessage shares the player-output framing used by the world
	// MessageSink. Combat callbacks historically wrote directly to s.send; the
	// session queue keeps both sources in one C-style flush boundary.
	enqueueCombatMessage := func(s *Session, message string) {
		if s == nil {
			return
		}
		s.notePlayerOutput()
		msg := wrap(message)
		if msg == nil {
			return
		}
		select {
		case s.send <- msg:
		default:
			slog.Warn("dropping combat message: channel full", "player", s.playerName)
		}
	}

	broadcast := func(roomVNum int, message string, exclude string) {
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
				enqueueCombatMessage(s, message)
			}
		}
	}

	sendToChar := func(name string, message string) {
		if s, ok := m.GetSession(name); ok {
			enqueueCombatMessage(s, message)
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
	if err := combat.InitEmbeddedFightMessages(cb); err != nil {
		slog.Error("loading combat messages", "error", err)
		combat.InitSkillMessages(cb)
	}
	m.combatEngine.SetCallbacks(cb)

	m.combatEngine.MessageFunc = func(attacker, defender combat.Combatant, dam, attackType int) bool {
		combat.SendWeaponMessage(dam, attacker, defender, attackType)
		return true
	}
}

// GetCombatEngine returns the combat engine for AI integration.
// GetShopManager returns the session manager's shop manager.
func (m *Manager) GetShopManager() *game.ShopManager {
	return m.shopManager
}

func (m *Manager) GetCombatEngine() *combat.CombatEngine {
	return m.combatEngine
}

// SetPulsePump installs the DP_CLOCK-only synchronous heartbeat driver.
func (m *Manager) SetPulsePump(pump func(int) error) {
	m.pulsePumpMu.Lock()
	defer m.pulsePumpMu.Unlock()
	m.pulsePump = pump
}

// ExtractPendingChars drains the world's C-style deferred extraction pass and
// returns connected victims to the main menu. C does this from heartbeat(),
// after raw_kill() has finished its synchronous death bytes; a linkless
// descriptor receives no menu because extract_char_final() has nothing to
// write to.
func (m *Manager) ExtractPendingChars() {
	extracted := m.world.ExtractPendingPlayers()
	for _, player := range extracted {
		m.mu.RLock()
		var victim *Session
		for _, s := range m.sessions {
			if s.player == player {
				victim = s
				break
			}
		}
		m.mu.RUnlock()
		if victim == nil || !victim.hasTransport() {
			continue
		}
		victim.showMainMenu()
	}
}

// PumpPulses advances the deterministic heartbeat without routing through the
// command interpreter. After the heartbeats return, every session that
// received player-bound output during the pump gets its prompt — C's
// process_output flushes every player every game-loop pass and each flush
// carries the prompt at its tail (comm.c:632-648, 1624-1640).
func (m *Manager) PumpPulses(n int) error {
	return m.PumpPulsesFrom(nil, n)
}

// PumpPulsesFrom is PumpPulses with the session whose control line drove the
// pump. The DP_CLOCK control is consumed inside process_input before the line
// enters C's input queue, so it does not clear has_prompt; pulse output is
// therefore flushed with the existing prompt's leading CRLF (comm.c:607,
// 1620-1643; tools/oracle-seam/dp-determinism.patch).
func (m *Manager) PumpPulsesFrom(_ *Session, n int) error {
	m.pulsePumpMu.RLock()
	pump := m.pulsePump
	m.pulsePumpMu.RUnlock()
	if pump == nil {
		return fmt.Errorf("pulse pump is not configured")
	}
	if err := pump(n); err != nil {
		return err
	}
	m.flushAsyncPrompts()
	return nil
}

// flushAsyncPrompts emits the trailing prompt for sessions that received
// game output outside their own command path (pumped pulse output delivered
// to idle players).
func (m *Manager) flushAsyncPrompts() {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()
	for _, s := range sessions {
		if s.outputSincePrompt.Load() > 0 && !s.IsCharCreating() && !s.IsMenuActive() && !s.IsPaging() {
			s.SendPrompt()
		}
	}
}

// Stop halts the manager's background workers and waits for them to exit.
// Tests must register this with t.Cleanup after NewManager so combat rounds
// cannot outlive their manager and race later package-global callback wiring.
func (m *Manager) Stop() {
	m.combatEngine.Stop()
}

// RequestShutdown hands an accepted do_shutdown option to the process
// lifecycle owner. A second request cannot replace the first C shutdown arm.
func (m *Manager) RequestShutdown(marker string) {
	select {
	case m.shutdownRequests <- ShutdownRequest{Marker: marker}:
	default:
		slog.Warn("shutdown request already pending")
	}
}

// ShutdownRequests returns the process-level shutdown request stream.
func (m *Manager) ShutdownRequests() <-chan ShutdownRequest {
	return m.shutdownRequests
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

// SetMobSpecialFunc wires native MOB_SPEC procedures into the combat engine.
// C perform_violence() invokes an assigned mob special after the mob's
// ordinary attack loop; mobile_activity() deliberately skips fighters, so the
// combat seam is the only correct call path for combat-time specials.
func (m *Manager) SetMobSpecialFunc() {
	m.combatEngine.MobSpecialFunc = func(mob combat.Combatant) bool {
		instance, ok := mob.(*game.MobInstance)
		if !ok || instance == nil || !instance.HasFlag("spec") {
			return false
		}
		spec := game.GetMobSpec(instance.GetVNum())
		if spec == nil {
			return false
		}
		return spec(m.world, nil, instance, "", "")
	}
}

// SetScriptDeathFunc wires the death trigger into the combat engine.
// When a mob dies, if it has a death script, it fires.
func (m *Manager) SetScriptDeathFunc() {
	m.combatEngine.ScriptDeathFunc = func(victimName string, killerName string, roomVNum int) {
		m.world.FireMobDeathScript(victimName, killerName, roomVNum)
	}
}

// SetOnRoundEnd wires the combat engine's per-round-end callback. Player wait
// states used to be decremented here (once per combat round), but the faithful
// command-drain port (DP-1201) moved that decrement to the heartbeat's per-pulse
// OnDrainInput (Manager.DrainInputQueues), matching comm.c:603 where wait
// decrements once per loop pass, not once per combat round. The seam is kept so
// CombatEngine.PerformRound's OnRoundEnd dispatch is unchanged, and to host any
// future round-granular work. The mob wait-state decrement
// (combat/engine.go:429-433) is separate and still runs.
func (m *Manager) SetOnRoundEnd() {
	m.combatEngine.OnRoundEnd = func() {}
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

// shouldCrownFirstPlayer reports whether the character currently being created
// should be crowned God (init_char first-player block, db.c:3016). Two paths:
//
//  1. Harness path: DP_FRESH_MUD is set → treat the MUD as fresh and crown the
//     FIRST character created this process, gated by the in-process godCrowned
//     latch (exactly one crown per boot). The harness runs Go with a dead DB, so
//     CountPlayers can't be used there; the env + latch stand in for "no players
//     yet."
//
//  2. Production path: DP_FRESH_MUD unset → crown iff the persisted player
//     store is empty (CountPlayers == 0). Only the very first player ever is
//     crowned; subsequent ones see a non-empty store and are ordinary mortals.
//
// DP_FRESH_MUD is a Go-side test control over initial store state (the
// DP_SEED/DP_CLOCK category — an external input), NOT a gameplay-output
// injection; the harness sets it ONLY for God-fixture scenarios so existing
// scenarios (combat-death, backstab-opener) still get ordinary mortals.
func (m *Manager) shouldCrownFirstPlayer() bool {
	if os.Getenv("DP_FRESH_MUD") != "" {
		// Harness path: crown exactly one, then the latch sticks.
		return !m.godCrowned.Swap(true)
	}
	// Production path: persisted store must be empty. No DB → can't be fresh
	// (there's nothing to be "first" against); behave as a normal mortal MUD.
	if !m.hasDB {
		return false
	}
	count, err := m.db.CountPlayers()
	if err != nil {
		slog.Warn("CountPlayers failed during char-creation God check; treating as non-fresh", "error", err)
		return false
	}
	return count == 0
}

// queuedInput is a single buffered command awaiting the per-pulse drain
// (DP-1201; port of comm.c:603 game_loop input queue).
type queuedInput struct {
	cmd     string
	args    []string
	rawArgs string
	aliased bool
}

// tryExecuteNow is the single player-input funnel gate (called from
// handleCommand in session_login.go). It mirrors comm.c:603: a command issued
// while wait>0 is NOT rejected — it stays queued and drains later, with no
// message. It returns true when the command was enqueued (caller returns nil,
// emitting nothing — the C delay), and false when the command should execute
// immediately (the wait==0 fast path).
//
// The wait>0 / queue-empty check and the append are one atomic critical section
// under inputMu. This preserves strict FIFO order: once the queue is non-empty,
// every new command appends to the tail, so a freshly-typed command can never
// jump ahead of a still-draining queue (e.g. if [A,B] are queued from a wait,
// A drains without setting a new wait, then a freshly-typed C arrives while
// wait==0 — C must still go behind B). The fast path is taken only when BOTH
// wait==0 AND the queue is empty.
func (s *Session) tryExecuteNow(cmd string, args []string, rawArgText ...string) bool {
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	if (s.player != nil && s.player.GetWaitState() > 0) || len(s.inputQueue) > 0 {
		rawArgs := ""
		if len(rawArgText) > 0 {
			rawArgs = rawArgText[0]
		}
		s.inputQueue = append(s.inputQueue, queuedInput{cmd: cmd, args: args, rawArgs: rawArgs})
		return true
	}
	return false
}

// enqueueInput appends a command to the tail of the drain queue regardless of
// wait state. Used by tests to pre-seed the queue without going through the
// funnel gate.
func (s *Session) enqueueInput(cmd string, args []string) {
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	s.inputQueue = append(s.inputQueue, queuedInput{cmd: cmd, args: args})
}

// prependAliasedInputs puts the commands generated by a complex alias ahead
// of any already-buffered player input. The aliased marker mirrors C's
// game-loop flag: queued expansions must not recursively expand aliases when
// they are drained.
func (s *Session) prependAliasedInputs(commands []string) {
	if len(commands) == 0 {
		return
	}
	inputs := make([]queuedInput, 0, len(commands)+len(s.inputQueue))
	for _, command := range commands {
		cmd, args := splitCommandInput(command)
		inputs = append(inputs, queuedInput{cmd: cmd, args: args, aliased: true})
	}
	inputs = append(inputs, s.inputQueue...)
	s.inputQueue = inputs
}

// dequeueInput pops the head of the drain queue. Returns ok=false when empty.
func (s *Session) dequeueInput() (queuedInput, bool) {
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	if len(s.inputQueue) == 0 {
		return queuedInput{}, false
	}
	head := s.inputQueue[0]
	s.inputQueue = s.inputQueue[1:]
	return head, true
}

// queueLen returns the current drain-queue depth (test helper).
func (s *Session) queueLen() int {
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	return len(s.inputQueue)
}

// DrainInputQueues is the heartbeat's per-pulse command-drain step (wired as
// OnDrainInput, dispatched at the TOP of heartbeat before OnPerformViolence —
// comm.c drains commands before perform_violence within a pass). Per session it:
//  1. decrements the player's wait by one pulse (comm.c:603 `--wait`); and
//  2. if wait<=0 and the queue is non-empty, dequeues exactly one command
//     (the drain is an `if`, not a `while` — one command per pulse).
//
// ExecuteCommand is deferred to a lock-free pass. Go's RWMutex is not
// reentrant, and command handlers reach GetSession/broadcasts that re-acquire
// m.mu — so mirroring the DamageFunc pattern (manager.go SetDamageFunc), we
// collect the drainable (session, cmd, args) tuples under m.mu.RLock, release
// the lock, then ExecuteCommand each outside it. DecrementWaitState is safe
// under the lock (it touches only Player.WaitState under p.mu, never m.mu).
func (m *Manager) DrainInputQueues() {
	type drainJob struct {
		s       *Session
		cmd     string
		args    []string
		rawArgs string
		aliased bool
	}

	m.mu.RLock()
	jobs := make([]drainJob, 0)
	for _, s := range m.sessions {
		if s.player == nil {
			continue
		}
		// Fact 2: wait decrements once per pulse.
		s.player.DecrementWaitState()
		// Fact 3: one command drains per pulse (if, not while).
		if s.player.GetWaitState() <= 0 {
			if input, ok := s.dequeueInput(); ok {
				jobs = append(jobs, drainJob{s: s, cmd: input.cmd, args: input.args, rawArgs: input.rawArgs, aliased: input.aliased})
			}
		}
	}
	m.mu.RUnlock()

	for _, job := range jobs {
		if err := executeCommandRaw(job.s, job.cmd, job.args, !job.aliased, job.rawArgs); err != nil {
			slog.Error("drained command failed",
				"player", job.s.playerName, "command", job.cmd, "error", err)
		}
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
		if err := cmdRetreat(s); err != nil {
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
		transportDone:       make(chan struct{}),
		connectionNumber:    m.allocateConnectionNumber(),
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
	if s.player != nil && !s.leaveBroadcastHandled {
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
	m.snoopMu.Lock()
	if s.snoopBy != nil {
		s.snoopBy.snooping = nil
	}
	if s.snooping != nil {
		s.snooping.snoopBy = nil
	}
	s.snoopBy = nil
	s.snooping = nil
	m.snoopMu.Unlock()

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

// HandleTelnetDisconnect preserves an authenticated playing character after
// an unexpected TCP EOF. C close_socket() saves the character and clears its
// descriptor, but leaves it in character_list so directed speech can report
// that the target is linkless. The linkdead reaper later owns extraction.
// It returns true when the session was retained as linkdead; orderly quits
// and pre-auth disconnects return false and use normal cleanup.
func (m *Manager) HandleTelnetDisconnect(s *Session) bool {
	if s == nil || !s.authenticated || s.player == nil || s.SendClosed() {
		return false
	}

	p := s.player
	p.SetLinkless(true)
	game.Act(m.world, true, p, nil, nil, nil, "$n has lost $s link.", "", game.ToRoom)

	if m.hasDB && p.ID > 0 && !s.isGuest {
		if rec, err := db.PlayerToRecord(p, nil); err == nil {
			if err := m.db.SavePlayer(rec); err != nil {
				slog.Error("linkdead save error", "player", s.playerName, "error", err)
			}
		}
	}

	s.DetachTransport()
	return true
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

// closeDuplicateSessions implements C do_quit's IDNUM anti-dupe sweep. Each
// duplicate is fully cleaned up (including its save) before the quitting
// session performs its final save, so the selected quit equipment policy wins.
func (m *Manager) closeDuplicateSessions(quitting *Session) {
	if quitting == nil || quitting.player == nil || quitting.player.GetID() <= 0 {
		return
	}

	type duplicate struct {
		name    string
		session *Session
	}
	id := quitting.player.GetID()
	m.mu.RLock()
	duplicates := make([]duplicate, 0)
	for name, candidate := range m.sessions {
		if candidate != quitting && candidate != nil && candidate.player != nil && candidate.player.GetID() == id {
			duplicates = append(duplicates, duplicate{name: name, session: candidate})
		}
	}
	m.mu.RUnlock()

	for _, duplicate := range duplicates {
		m.mu.Lock()
		current, ok := m.sessions[duplicate.name]
		if ok && current == duplicate.session {
			delete(m.sessions, duplicate.name)
		}
		m.mu.Unlock()
		if !ok || current != duplicate.session {
			continue
		}
		m.cleanupSession(duplicate.session, duplicate.name)
		duplicate.session.Close()
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
	// Some callers hand over a raw text line instead of a marshaled
	// ServerMessage envelope. Every transport renders only envelope frames
	// (the telnet writeLoop json.Unmarshals each send), so raw payloads were
	// silently dropped for every client — wrap them at the sink.
	if !json.Valid(message) {
		if envelope, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{Type: "text", Text: string(message)},
		}); err == nil {
			message = envelope
		}
	}

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
	remoteIP             string        // Store remote IP directly for non-HTTP (Telnet) sessions
	manager              *Manager
	send                 chan []byte
	player               *game.Player
	playerName           string
	authenticated        bool
	isGuest              bool
	connCountDecremented bool // C5: prevents double-decrement of IP connection count
	banLevel             int  // ban level from IsBanned (BanNew or BanSelect); 0 = no ban

	// outputSincePrompt counts player-bound messages enqueued since the last
	// prompt. C's process_output appends "\r\n" + make_prompt to every output
	// flush (comm.c:1624-1640); this counter lets SendPrompt reproduce that
	// trailing framing and lets the post-pulse sweep find sessions that owe an
	// async prompt after pumped heartbeat output.
	outputSincePrompt atomic.Int64

	// Agent identity — set on login when is_agent=true.
	// Harness+Model is the agent identity. Same combo = same agent across sessions.
	isAgent          bool
	agentHarness     string    // e.g. "openclaw", "claude-code"
	agentModel       string    // e.g. "mimo-v2.5-base"
	agentVersion     string    // harness version
	agentKeyID       int64     // legacy: kept for backward compat, deprecated
	connectedAt      time.Time // set on session creation, used for sessionID()
	connectionNumber int       // C descriptor number, used by do_dc
	olcZone          int       // C GET_OLC_ZONE; zero until an OLC zone is assigned

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

	// Post-MOTD main menu state. This is separate from character creation
	// because returning players pass through the same menu before world entry.
	menuActive           bool
	menuStage            string
	menuDescription      string
	menuDescriptionDraft []string
	menuPasswordHash     string
	menuNewPasswordHash  string

	// Output pager state (DP-1195; port of the Buselli pager, modify.c:346-527).
	// While pagerCount > 0, every input line routes to handlePagerInput instead
	// of the command interpreter — mirroring C's showstr_count routing
	// (comm.c:617). pagerPages holds the pre-split page byte slices,
	// pagerPage is the 0-based current page (C's showstr_page). Telnet/plain-text
	// only; structured-data clients receive whole text and never enter this mode.
	pagerPages [][]byte
	pagerPage  int
	pagerCount int

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

	// C-faithful per-pulse command-drain queue (DP-1201; port of comm.c:603
	// game_loop). A command issued while wait>0 is NOT rejected — it stays
	// queued here and drains one per heartbeat pulse once wait reaches 0, with
	// no message (the C delay). tryExecuteNow is the single funnel at
	// handleCommand (session_login.go); DrainInputQueues drains from the
	// heartbeat's OnDrainInput. inputMu guards the slice; the queue is only
	// ever appended at the funnel and popped at the drain, and the atomic
	// wait>0/empty check + append under inputMu preserves strict FIFO order
	// (nothing can jump a draining queue).
	inputMu    sync.Mutex
	inputQueue []queuedInput

	// Temporary data storage for command handlers
	tempData map[string]interface{}

	// Infobar / display state (from act.display.c)
	screenSize                          int //nolint:unused // terminal height in lines; 0 = unset (defaults to 25)
	infobarMode                         int //nolint:unused // InfobarOff (0) or InfobarOn (1)
	infobarLastHit, infobarLastMaxHit   int
	infobarLastMana, infobarLastMaxMana int
	infobarLastMove, infobarLastMaxMove int
	infobarLastExp, infobarLastGold     int
	// leaveBroadcastHandled is set by an orderly quit after it applies C's
	// invisibility gate, preventing generic disconnect cleanup from announcing
	// the same departure a second time.
	leaveBroadcastHandled bool

	// Communication state
	snooping *Session // Session being snooped (for wizard snoop)
	snoopBy  *Session // Session that is snooping us

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

	// transportDone is closed when a transport disappears while an
	// authenticated player remains linkdead in the world. The send channel is
	// deliberately left open until the linkdead session is reaped.
	transportDone chan struct{}
	transportOnce sync.Once
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
	m.shutdownGracefully(timeout, true)
}

// ShutdownGracefullyWithoutNotice is used after C do_shutdown has already
// broadcast its exact shutdown text and forced the all-save continuation.
func (m *Manager) ShutdownGracefullyWithoutNotice(timeout time.Duration) {
	m.shutdownGracefully(timeout, false)
}

func (m *Manager) shutdownGracefully(timeout time.Duration, notify bool) {
	m.Stop()

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

	// 1. Notify players of an external shutdown. A command-triggered C
	// shutdown already emitted its own global bytes and save notices.
	if notify {
		for _, s := range sessions {
			s.sendText("\r\n\x1b[1;31m!! The MUD server is performing a graceful shutdown for maintenance. Your state has been saved. !!\x1b[0m\r\n")
		}
		// 2. Wait a brief moment for messages to flush
		time.Sleep(1 * time.Second)
	}

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
