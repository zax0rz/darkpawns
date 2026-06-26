package agentcli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Daemon holds the persistent connection to the MUD server and exposes
// a Unix socket for CLI commands. This is the "body" — it stays connected
// while the LLM (mind) connects and disconnects.
type Daemon struct {
	cfg     *AgentConfig
	client  *AgentClient
	state   *StateFile
	events  *EventBuffer
	sock    net.Listener
	mu      sync.Mutex
	running bool
}

// DaemonRequest is a CLI→daemon command.
type DaemonRequest struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args,omitempty"`
}

// DaemonResponse is a daemon→CLI response.
type DaemonResponse struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// NewDaemon creates a new daemon for the given character.
func NewDaemon(cfg *AgentConfig) (*Daemon, error) {
	stateFile, err := NewStateFile(cfg.PlayerName)
	if err != nil {
		return nil, fmt.Errorf("state file: %w", err)
	}

	eventBuf, err := NewEventBuffer(cfg.PlayerName)
	if err != nil {
		return nil, fmt.Errorf("event buffer: %w", err)
	}

	return &Daemon{
		cfg:    cfg,
		state:  stateFile,
		events: eventBuf,
	}, nil
}

// socketPath returns the Unix socket path for this character.
func socketPath(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dp-goat", "sock", name+".sock")
}

// Start connects to the MUD and starts listening on the Unix socket.
func (d *Daemon) Start(ctx context.Context) error {
	// Ensure socket directory exists
	sockPath := socketPath(d.cfg.PlayerName)
	if err := os.MkdirAll(filepath.Dir(sockPath), 0o755); err != nil {
		return fmt.Errorf("mkdir socket dir: %w", err)
	}
	// Remove stale socket
	_ = os.Remove(sockPath)

	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	d.mu.Unlock()

	// Connect to MUD server
	client := NewAgentClient(d.cfg)
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect to MUD: %w", err)
	}

	slog.Info("daemon connected to MUD", "player", d.cfg.PlayerName)

	// Load previous state
	if _, err := d.state.Load(); err != nil {
		slog.Warn("failed to load previous state", "error", err)
	}

	// Start Unix socket listener
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("listen socket: %w", err)
	}

	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		_ = client.Close()
		_ = ln.Close()
		_ = os.Remove(sockPath)
		return fmt.Errorf("daemon already running")
	}
	d.client = client
	d.sock = ln
	d.running = true
	d.mu.Unlock()

	slog.Info("daemon listening", "socket", sockPath)

	// Start background loops
	go d.readLoop(ctx)
	go d.acceptLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("daemon shutting down")
	d.Stop()
	return nil
}

// Stop cleanly shuts down the daemon.
func (d *Daemon) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return
	}
	d.running = false

	if d.client != nil {
		_ = d.client.Close()
	}
	if d.sock != nil {
		_ = d.sock.Close()
	}

	// Clean up socket file
	sockPath := socketPath(d.cfg.PlayerName)
	_ = os.Remove(sockPath)

	slog.Info("daemon stopped")
}

// readLoop reads messages from the MUD server and updates state.
func (d *Daemon) readLoop(ctx context.Context) {
	defer func() {
		slog.Info("read loop ended")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if d.client.conn == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		_, raw, err := d.client.conn.ReadMessage()
		if err != nil {
			slog.Warn("read error", "error", err)
			// Trigger reconnection
			d.reconnect(ctx)
			return
		}

		// Parse and handle the message
		if err := d.handleMessage(raw); err != nil {
			slog.Error("handle message", "error", err)
		}
	}
}

// reconnect attempts to reconnect to the MUD server.
func (d *Daemon) reconnect(ctx context.Context) {
	slog.Info("reconnecting...")
	rcfg := DefaultReconnectConfig()
	rcfg.InitialBackoff = 2 * time.Second

	if err := d.client.Reconnect(ctx, rcfg); err != nil {
		slog.Error("reconnect failed", "error", err)
		return
	}

	slog.Info("reconnected, resuming read loop")
	go d.readLoop(ctx)
}

// handleMessage processes an incoming MUD server message.
func (d *Daemon) handleMessage(raw []byte) error {
	var env struct {
		Type string          `json:"type"`
		Seq  uint64          `json:"seq,omitempty"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Track sequence number
	if env.Seq > 0 {
		d.client.conn.seqMu.Lock()
		d.client.conn.lastSeq = env.Seq
		d.client.conn.seqMu.Unlock()
	}

	switch env.Type {
	case "vars":
		return d.handleVars(env.Data)
	case "state":
		return d.handleState(env.Data)
	case "event":
		return d.handleEvent(env.Data)
	case "error":
		slog.Error("server error", "msg", string(env.Data))
		return nil
	default:
		return nil
	}
}

// handleVars processes a variable subscription update.
func (d *Daemon) handleVars(data json.RawMessage) error {
	var vars struct {
		HEALTH     int      `json:"HEALTH"`
		MAX_HEALTH int      `json:"MAX_HEALTH"`
		MANA       int      `json:"MANA"`
		MAX_MANA   int      `json:"MAX_MANA"`
		MOVE       int      `json:"MOVE"`
		MAX_MOVE   int      `json:"MAX_MOVE"`
		LEVEL      int      `json:"LEVEL"`
		EXP        int      `json:"EXP"`
		ROOM_VNUM  int      `json:"ROOM_VNUM"`
		ROOM_NAME  string   `json:"ROOM_NAME"`
		ROOM_EXITS []string `json:"ROOM_EXITS"`
		ROOM_MOBS  []Mob    `json:"ROOM_MOBS"`
		ROOM_ITEMS []Item   `json:"ROOM_ITEMS,omitempty"`
		FIGHTING   string   `json:"FIGHTING"`
		GOLD       int      `json:"GOLD"`
		POSITION   string   `json:"POSITION"`
		INVENTORY  []Item   `json:"INVENTORY,omitempty"`
	}
	if err := json.Unmarshal(data, &vars); err != nil {
		return fmt.Errorf("parse vars: %w", err)
	}

	state := d.state.Get()
	state.Player.Health = vars.HEALTH
	state.Player.MaxHealth = vars.MAX_HEALTH
	state.Player.Mana = vars.MANA
	state.Player.Level = vars.LEVEL
	state.Player.Exp = vars.EXP
	state.Player.Gold = vars.GOLD
	state.Room.Vnum = vars.ROOM_VNUM
	state.Room.Name = vars.ROOM_NAME
	state.Room.Exits = vars.ROOM_EXITS
	state.Room.Mobs = vars.ROOM_MOBS
	state.Room.Items = vars.ROOM_ITEMS
	state.Fighting = vars.FIGHTING
	state.Inventory = vars.INVENTORY

	// Persist state
	if err := d.state.Save(state); err != nil {
		slog.Warn("save state", "error", err)
	}

	// Buffer event
	_, _ = d.events.Append("vars", vars)

	return nil
}

// handleState processes a full state snapshot from the server.
func (d *Daemon) handleState(data json.RawMessage) error {
	state := d.state.Get()
	if err := json.Unmarshal(data, state); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}

	if err := d.state.Save(state); err != nil {
		slog.Warn("save state", "error", err)
	}

	_, _ = d.events.Append("state", state)
	return nil
}

// handleEvent processes a game event (text message, combat, etc.).
func (d *Daemon) handleEvent(data json.RawMessage) error {
	// Extract event type from the data
	var evt struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil // not all events have structured data
	}

	_, _ = d.events.Append(evt.Type, data)
	return nil
}

// acceptLoop handles incoming CLI connections on the Unix socket.
func (d *Daemon) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := d.sock.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Warn("accept error", "error", err)
				continue
			}
		}

		go d.handleCLI(conn)
	}
}

// handleCLI processes a single CLI request over the Unix socket.
func (d *Daemon) handleCLI(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("close conn", "error", err)
		}
	}()

	scanner := bufio.NewScanner(conn)
	// Increase buffer for large inventory/room descriptions
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		var req DaemonRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			d.sendResponse(conn, DaemonResponse{OK: false, Error: "invalid request"})
			continue
		}

		resp := d.executeCommand(req)
		d.sendResponse(conn, resp)
	}
}

// sendResponse writes a JSON response to the CLI connection.
func (d *Daemon) sendResponse(conn net.Conn, resp DaemonResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = conn.Write(data)
}

// executeCommand routes a CLI request to the appropriate handler.
func (d *Daemon) executeCommand(req DaemonRequest) DaemonResponse {
	switch req.Cmd {
	case "status":
		return d.cmdStatus()
	case "look":
		return d.cmdLook(req.Args)
	case "move":
		return d.cmdMove(req.Args)
	case "say":
		return d.cmdSay(req.Args)
	case "tell":
		return d.cmdTell(req.Args)
	case "kill":
		return d.cmdKill(req.Args)
	case "flee":
		return d.cmdFlee()
	case "get":
		return d.cmdGet(req.Args)
	case "drop":
		return d.cmdDrop(req.Args)
	case "inventory":
		return d.cmdInventory()
	case "score":
		return d.cmdScore()
	case "events":
		return d.cmdEvents(req.Args)
	case "context":
		return d.cmdContext()
	case "raw":
		return d.cmdRaw(req.Args)
	default:
		return DaemonResponse{OK: false, Error: fmt.Sprintf("unknown command: %s", req.Cmd)}
	}
}

// cmdStatus returns a quick status summary.
func (d *Daemon) cmdStatus() DaemonResponse {
	state := d.state.Get()
	data, _ := json.Marshal(map[string]interface{}{
		"player":   state.Player,
		"room":     state.Room,
		"fighting": state.Fighting,
		"gold":     state.Player.Gold,
	})
	return DaemonResponse{OK: true, Data: data}
}

// cmdLook sends a look command and returns the result.
func (d *Daemon) cmdLook(args []string) DaemonResponse {
	return d.sendCommand("look", args)
}

// cmdMove sends a movement command.
func (d *Daemon) cmdMove(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "direction required"}
	}
	return d.sendCommand(args[0], nil)
}

// cmdSay sends a say command.
func (d *Daemon) cmdSay(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "message required"}
	}
	return d.sendCommand("say", args)
}

// cmdTell sends a tell command.
func (d *Daemon) cmdTell(args []string) DaemonResponse {
	if len(args) < 2 {
		return DaemonResponse{OK: false, Error: "usage: tell <player> <message>"}
	}
	target := args[0]
	msg := args[1:]
	return d.sendCommand("tell", append([]string{target}, msg...))
}

// cmdKill sends a kill command.
func (d *Daemon) cmdKill(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "target required"}
	}
	return d.sendCommand("kill", args)
}

// cmdFlee sends a flee command.
func (d *Daemon) cmdFlee() DaemonResponse {
	return d.sendCommand("flee", nil)
}

// cmdGet sends a get command.
func (d *Daemon) cmdGet(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "item required"}
	}
	return d.sendCommand("get", args)
}

// cmdDrop sends a drop command.
func (d *Daemon) cmdDrop(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "item required"}
	}
	return d.sendCommand("drop", args)
}

// cmdInventory returns the current inventory.
func (d *Daemon) cmdInventory() DaemonResponse {
	state := d.state.Get()
	data, _ := json.Marshal(map[string]interface{}{
		"inventory": state.Inventory,
		"equipment": state.Equipment,
	})
	return DaemonResponse{OK: true, Data: data}
}

// cmdScore returns character stats.
func (d *Daemon) cmdScore() DaemonResponse {
	state := d.state.Get()
	data, _ := json.Marshal(state.Player)
	return DaemonResponse{OK: true, Data: data}
}

// cmdEvents returns buffered events.
func (d *Daemon) cmdEvents(args []string) DaemonResponse {
	var since uint64
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &since); err != nil {
			return DaemonResponse{OK: false, Error: fmt.Sprintf("invalid event seq: %s", args[0])}
		}
	}
	events := d.events.Since(since)
	data, _ := json.Marshal(events)
	return DaemonResponse{OK: true, Data: data}
}

// cmdContext returns a full context packet for the LLM mind.
func (d *Daemon) cmdContext() DaemonResponse {
	state := d.state.Get()
	summary := d.events.CompactionWindow(0)
	recent := d.events.Recent(10)

	data, _ := json.Marshal(map[string]interface{}{
		"state":   state,
		"summary": summary,
		"events":  recent,
	})
	return DaemonResponse{OK: true, Data: data}
}

// cmdRaw sends a raw command to the MUD server.
func (d *Daemon) cmdRaw(args []string) DaemonResponse {
	if len(args) == 0 {
		return DaemonResponse{OK: false, Error: "command required"}
	}
	return d.sendCommand(args[0], args[1:])
}

// sendCommand sends a command to the MUD server and waits briefly for a response.
func (d *Daemon) sendCommand(cmd string, args []string) DaemonResponse {
	d.mu.Lock()
	if d.client == nil || d.client.conn == nil {
		d.mu.Unlock()
		return DaemonResponse{OK: false, Error: "not connected"}
	}
	d.mu.Unlock()

	// Send command
	command := map[string]any{
		"type": "command",
		"data": map[string]any{
			"command": cmd,
			"args":    args,
		},
	}
	if err := d.client.conn.WriteJSON(command); err != nil {
		return DaemonResponse{OK: false, Error: fmt.Sprintf("send: %v", err)}
	}

	// For commands that expect a response, wait briefly
	// In practice, the state subscription handles async updates
	return DaemonResponse{OK: true, Data: json.RawMessage(`{"status":"sent"}`)}
}
