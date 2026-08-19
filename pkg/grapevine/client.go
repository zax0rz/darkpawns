package grapevine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type Client struct {
	world    *game.World
	conn     *websocket.Conn
	mu       sync.Mutex
	closed   bool
	done     chan struct{}
	send     chan grapevineMessage
	stopOnce sync.Once
}

func NewClient(world *game.World) *Client {
	return &Client{
		world: world,
		done:  make(chan struct{}),
		send:  make(chan grapevineMessage, 256),
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

	go c.writeLoop()
	go c.connectLoop(url, clientID, clientSecret)
}

func (c *Client) connectLoop(url, clientID, clientSecret string) {
	backoff := 10 * time.Second
	maxBackoff := 5 * time.Minute

	for {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		slog.Info("Grapevine: attempting to connect to socket", "url", url)
		dialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}
		conn, _, err := dialer.Dial(url, nil)
		if err != nil {
			slog.Error("Grapevine: connection dial failed", "error", err, "retry_in", backoff.String())
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		slog.Info("Grapevine: socket connected, starting handshake")
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		// Reset backoff on successful connection
		backoff = 10 * time.Second

		if err := c.authenticate(clientID, clientSecret); err != nil {
			slog.Error("Grapevine: authentication handshake failed", "error", err)
			_ = conn.Close()
			continue
		}

		// Subscribe to gossip channel
		if err := c.subscribe("gossip"); err != nil {
			slog.Error("Grapevine: channel subscription failed", "error", err)
			_ = conn.Close()
			continue
		}

		// Wire local gossip callback
		c.world.OnGossip = func(senderName string, msg string) {
			c.sendGossip(senderName, msg)
		}

		// Start heartbeat ticker
		heartbeatDone := make(chan struct{})
		go c.presenceTicker(heartbeatDone)

		// Run read loop
		c.readLoop()

		// Cleanup on disconnect
		close(heartbeatDone)
		c.mu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.world.OnGossip = nil
		c.mu.Unlock()

		slog.Info("Grapevine: disconnected, reconnecting in background...")
		time.Sleep(backoff)
	}
}

type grapevineMessage struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func (c *Client) authenticate(clientID, clientSecret string) error {
	payload := map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
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

func (c *Client) subscribe(channel string) error {
	payload := map[string]interface{}{
		"channel": channel,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := grapevineMessage{
		Event:   "channels/subscribe",
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

func (c *Client) presenceTicker(done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial presence ping
	c.sendPresence()

	for {
		select {
		case <-ticker.C:
			c.sendPresence()
		case <-done:
			return
		}
	}
}

func (c *Client) sendPresence() {
	players := c.world.GetAllPlayers()
	names := make([]string, 0)
	for _, p := range players {
		if !p.IsNPC() {
			names = append(names, p.Name)
		}
	}

	payload := map[string]interface{}{
		"players": names,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Grapevine: failed to marshal presence", "error", err)
		return
	}
	c.enqueue(grapevineMessage{
		Event:   "players/sign-in",
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

func (c *Client) readLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("Grapevine: read loop connection closed", "error", err)
			return
		}

		var msg grapevineMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Error("Grapevine: failed to unmarshal message", "error", err)
			continue
		}

		switch msg.Event {
		case "authenticate":
			var status struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal(msg.Payload, &status); err == nil {
				slog.Info("Grapevine: handshake status", "status", status.Status)
			}
		case "channels/broadcast":
			var bc grapevineBroadcast
			if err := json.Unmarshal(msg.Payload, &bc); err != nil {
				slog.Error("Grapevine: failed to unmarshal broadcast payload", "error", err)
				continue
			}
			if bc.Channel == "gossip" {
				// Format message with premium deep-purple ANSI styling
				formatted := fmt.Sprintf("\x1B[1;35m[Grapevine] %s@%s gossips, '%s'\033[0m\r\n", bc.Name, bc.Game, bc.Message)

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
		close(c.done)
	})
}
