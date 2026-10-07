package grapevine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// gameName is the name this game registers under on the Grapevine
// network. Player presence entries are formatted "Name@GameName".
const gameName = "Dark Pawns"

// Mode selects how much of the Grapevine network the game takes part
// in, via the GRAPEVINE_MODE environment variable.
type Mode int

const (
	// ModeOff disables the integration entirely: no dial, no presence,
	// nothing player-visible.
	ModeOff Mode = iota
	// ModePresence authenticates and answers heartbeats with the player
	// list, but subscribes to no channels and never writes to players.
	// Player-visible bytes are identical to ModeOff.
	ModePresence
	// ModeFull additionally subscribes to the gossip channel and bridges
	// it in both directions.
	ModeFull
)

func (m Mode) String() string {
	switch m {
	case ModePresence:
		return "presence"
	case ModeFull:
		return "full"
	default:
		return "off"
	}
}

// resolveMode applies the GRAPEVINE_MODE decision table:
//
//   - credentials missing            -> off, regardless of mode
//   - mode unset (credentials given) -> presence (safe for classic)
//   - "off" / "presence" / "full"    -> that mode
//   - anything else                  -> presence; the second return
//     value carries the invalid raw value so the caller can warn once.
//
// ModeFromEnv exposes the resolved mode to other packages (MSSP's
// INTERMUD field) without the warning path.
func resolveMode() (Mode, string) {
	if os.Getenv("GRAPEVINE_CLIENT_ID") == "" || os.Getenv("GRAPEVINE_CLIENT_SECRET") == "" {
		return ModeOff, ""
	}
	switch raw := os.Getenv("GRAPEVINE_MODE"); raw {
	case "", "presence":
		return ModePresence, ""
	case "off":
		return ModeOff, ""
	case "full":
		return ModeFull, ""
	default:
		return ModePresence, raw
	}
}

// ModeFromEnv returns the Grapevine mode the current environment
// resolves to.
func ModeFromEnv() Mode {
	mode, _ := resolveMode()
	return mode
}

type Client struct {
	world    *game.World
	conn     *websocket.Conn
	mu       sync.Mutex
	closed   bool
	done     chan struct{}
	send     chan grapevineMessage
	stopOnce sync.Once

	mode Mode
	// reconnectBackoff is the initial delay before reconnecting after a
	// failed attempt; it doubles per consecutive failure up to a cap.
	// Tests shrink it to keep the auth-failure case fast.
	reconnectBackoff time.Duration
}

// maxMessageSize caps inbound relay frames (C4 parity with the session
// pump's /ws read limit).
const maxMessageSize = 16384

// stripControlChars removes terminal control characters from relay text
// before it reaches player terminals. Peer-MUD players author broadcast
// content; the shared relay authenticates connections, not content. The
// session package applies the identical rule to every local speech path
// (sanitize.go) — this brings the one inbound remote path to parity.
func stripControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0x20 && r != 0x7f {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func NewClient(world *game.World) *Client {
	return &Client{
		world:            world,
		done:             make(chan struct{}),
		send:             make(chan grapevineMessage, 256),
		reconnectBackoff: 10 * time.Second,
	}
}

// Start launches the Grapevine WebSocket connection loop in a non-blocking goroutine.
func (c *Client) Start() {
	clientID := os.Getenv("GRAPEVINE_CLIENT_ID")
	clientSecret := os.Getenv("GRAPEVINE_CLIENT_SECRET")
	url := os.Getenv("GRAPEVINE_URL")
	if url == "" {
		url = "wss://grapevine.haus/socket"
	}

	if clientID == "" || clientSecret == "" {
		slog.Warn("Grapevine: client ID or secret not configured. Grapevine integration is disabled (running offline).")
		return
	}

	mode, invalid := resolveMode()
	if invalid != "" {
		slog.Warn("Grapevine: invalid GRAPEVINE_MODE, falling back to presence", "value", invalid)
	}
	if mode == ModeOff {
		slog.Info("Grapevine: disabled by GRAPEVINE_MODE=off")
		return
	}
	c.mode = mode
	slog.Info("Grapevine: starting", "mode", mode.String())

	go c.writeLoop()
	go c.connectLoop(url, clientID, clientSecret)
}

func (c *Client) connectLoop(url, clientID, clientSecret string) {
	const maxBackoff = 5 * time.Minute
	backoff := c.reconnectBackoff

	for {
		c.mu.Lock()
		closed := c.closed
		c.mu.Unlock()
		if closed {
			return
		}

		slog.Info("Grapevine: attempting to connect to socket", "url", url)
		dialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}
		conn, _, err := dialer.Dial(url, nil)
		if err != nil {
			slog.Error("Grapevine: connection dial failed", "error", err, "retry_in", backoff.String())
			if !c.sleepBackoff(backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		// The relay is a third-party network: an oversized or malicious frame
		// must not translate into unbounded local allocation. Same cap the
		// session pump applies to /ws (C4).
		conn.SetReadLimit(maxMessageSize)

		slog.Info("Grapevine: socket connected, starting handshake")
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		authed, _ := c.runConnection(conn, clientID, clientSecret)

		// Cleanup on disconnect: drop the connection and clear the gossip
		// bridge so local gossip never enqueues into a dead socket.
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		_ = conn.Close()
		c.world.SetOnGossip(nil)

		slog.Info("Grapevine: disconnected, reconnecting in background...")
		// A session that authenticated earns a prompt reconnect; a failed
		// one backs off on the doubling schedule so a rejected credential
		// never hot-loops against the relay.
		if authed {
			backoff = c.reconnectBackoff
		}
		if !c.sleepBackoff(backoff) {
			return
		}
		if !authed {
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

// sleepBackoff waits out a reconnect delay, returning false if the
// client was stopped meanwhile.
func (c *Client) sleepBackoff(d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-c.done:
		return false
	}
}

// runConnection authenticates on a fresh socket and serves it until it
// drops. It reports whether authentication succeeded and whether the
// failure was an authentication rejection (bad status or close 4000).
func (c *Client) runConnection(conn *websocket.Conn, clientID, clientSecret string) (authed, authFailed bool) {
	if err := c.authenticate(clientID, clientSecret); err != nil {
		slog.Error("Grapevine: authentication handshake failed", "error", err)
		return false, true
	}
	return c.readLoop(conn)
}

type grapevineMessage struct {
	Event   string          `json:"event"`
	Status  string          `json:"status,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

func (c *Client) authenticate(clientID, clientSecret string) error {
	// Protocol (grapevine.haus/docs): "supports" is required and must
	// contain "channels" — unknown support options get the socket
	// disconnected. Channel subscriptions ride the auth payload; there
	// is no separate subscribe step. Presence mode subscribes to
	// nothing, full mode to gossip.
	channels := []string{}
	if c.mode == ModeFull {
		channels = []string{"gossip"}
	}
	payload := map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"supports":      []string{"channels"},
		"channels":      channels,
		"version":       "1.0.0",
		"user_agent":    gameName,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := grapevineMessage{
		Event:   "authenticate",
		Payload: data,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteJSON(msg)
}

func (c *Client) sendGossip(senderName, message string) {
	payload := map[string]interface{}{
		"channel": "gossip",
		"name":    senderName,
		"message": message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Grapevine: failed to marshal outbound gossip", "error", err)
		return
	}
	c.enqueue(grapevineMessage{
		Event:   "channels/send",
		Payload: data,
	})
}

// sendHeartbeat answers a relay heartbeat with the current player list.
// Presence on Grapevine is heartbeat-driven: three missed replies close
// the socket (4001). The list holds only the characters a level-1 mortal
// could see in `who` (World.VisiblePlayersForMortal) — a wizinvis
// immortal must never be revealed to an external network.
func (c *Client) sendHeartbeat() {
	players := c.world.VisiblePlayersForMortal()
	names := make([]string, 0, len(players))
	for _, p := range players {
		names = append(names, p.Name+"@"+gameName)
	}
	sort.Strings(names)

	payload := map[string]interface{}{
		"players": names,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Grapevine: failed to marshal heartbeat", "error", err)
		return
	}
	c.enqueue(grapevineMessage{
		Event:   "heartbeat",
		Payload: data,
	})
}

// enqueue queues an outbound message for the single background writer.
// If the queue is full, the message is dropped to apply backpressure
// instead of spawning unbounded goroutines.
func (c *Client) enqueue(msg grapevineMessage) {
	select {
	case c.send <- msg:
	default:
		slog.Warn("Grapevine: outbound message queue full, dropping message", "event", msg.Event)
	}
}

// writeLoop is the single writer goroutine that drains the send queue.
func (c *Client) writeLoop() {
	for {
		select {
		case msg := <-c.send:
			c.writeMessage(msg)
		case <-c.done:
			return
		}
	}
}

func (c *Client) writeMessage(msg grapevineMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil || c.closed {
		return
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.conn.WriteJSON(msg); err != nil {
		slog.Error("Grapevine: failed to send outbound payload", "event", msg.Event, "error", err)
	}
}

type grapevineBroadcast struct {
	Channel string `json:"channel"`
	Game    string `json:"game"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// readLoop serves one connected socket until it drops. The relay puts
// the auth verdict in a top-level "status" field (a sibling of
// "payload", not inside it). The gossip bridge is installed only after
// a successful auth in full mode, and heartbeat replies are the only
// unsolicited traffic presence mode ever sends.
func (c *Client) readLoop(conn *websocket.Conn) (authed, authFailed bool) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == 4000 {
				// 4000 = authentication failed at the relay.
				slog.Error("Grapevine: relay closed the socket: authentication failed (close code 4000)")
				return authed, true
			}
			slog.Warn("Grapevine: read loop connection closed", "error", err)
			return authed, false
		}

		var msg grapevineMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Error("Grapevine: failed to unmarshal message", "error", err)
			continue
		}

		switch msg.Event {
		case "authenticate":
			if msg.Status != "success" {
				slog.Error("Grapevine: authentication rejected by relay", "status", msg.Status)
				return authed, true
			}
			authed = true
			slog.Info("Grapevine: authenticated with relay", "mode", c.mode.String())
			if c.mode == ModeFull {
				// Wire the local gossip callback only now: before a
				// successful auth there is no bridge.
				c.world.SetOnGossip(func(senderName string, msg string) {
					c.sendGossip(senderName, msg)
				})
			}
		case "heartbeat":
			c.sendHeartbeat()
		case "channels/broadcast":
			// Presence mode never subscribes, so it should never see a
			// broadcast; if one arrives anyway it reaches no player.
			if c.mode != ModeFull || !authed {
				continue
			}
			var bc grapevineBroadcast
			if err := json.Unmarshal(msg.Payload, &bc); err != nil {
				slog.Error("Grapevine: failed to unmarshal broadcast payload", "error", err)
				continue
			}
			if bc.Channel == "gossip" {
				// Sanitize peer-authored fields: control characters must not
				// reach player terminals (terminal injection, CWE-150).
				name, game_ := stripControlChars(bc.Name), stripControlChars(bc.Game)
				message := stripControlChars(bc.Message)
				// Format message with premium deep-purple ANSI styling
				formatted := fmt.Sprintf("\x1B[1;35m[Grapevine] %s@%s gossips, '%s'\033[0m\r\n", name, game_, message)

				// Broadcast to all active MUD players who can hear gossip
				players := c.world.AllPlayers()
				for _, p := range players {
					if p.IsNPC() {
						continue
					}
					// Check NO_GOSSIP flag / PrfNoGossip from other_helpers.go
					if p.Flags&(1<<game.PrfNoGossip) != 0 {
						continue
					}
					// Check deaf flag
					if p.Flags&(1<<game.PrfDeaf) != 0 {
						continue
					}
					p.SendMessage(formatted)
				}
			}
		case "restart":
			// The relay announces a restart with an expected downtime;
			// the socket may drop at any point after this. Log it and let
			// the reconnect loop handle the drop.
			var rp struct {
				Downtime int `json:"downtime"`
			}
			if err := json.Unmarshal(msg.Payload, &rp); err == nil {
				slog.Warn("Grapevine: relay restart announced", "downtime_seconds", rp.Downtime)
			} else {
				slog.Warn("Grapevine: relay restart announced")
			}
		}
	}
}

func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		if c.conn != nil {
			_ = c.conn.Close()
		}
		c.mu.Unlock()
		c.world.SetOnGossip(nil)
		close(c.done)
	})
}
