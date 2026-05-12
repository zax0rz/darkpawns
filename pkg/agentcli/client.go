package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// AgentClient manages a WebSocket connection to the Dark Pawns game server.
type AgentClient struct {
	Cfg     *AgentConfig
	conn    *WSConn
	state   *GameState
	session *SessionLogger
}

// NewAgentClient creates a new agent client with the given config.
func NewAgentClient(cfg *AgentConfig) *AgentClient {
	return &AgentClient{Cfg: cfg}
}

// GameState holds the latest structured state from the server.
type GameState struct {
	Player struct {
		Name     string `json:"name"`
		Health   int    `json:"health"`
		MaxHealth int   `json:"max_health"`
		Mana     int    `json:"mana"`
		Level    int    `json:"level"`
		Exp      int    `json:"exp"`
	} `json:"player"`
	Room struct {
		Vnum        int      `json:"vnum"`
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Exits       []string `json:"exits"`
		Mobs        []Mob    `json:"mobs"`
		Items       []Item   `json:"items"`
	} `json:"room"`
	Fighting      string           `json:"fighting,omitempty"`
	Inventory     []Item           `json:"inventory,omitempty"`
	Equipment     map[string]Item  `json:"equipment,omitempty"`
	Events        []Event          `json:"events,omitempty"`
	MemorySummary string           `json:"memory_summary,omitempty"`
}

type Mob struct {
	Name         string `json:"name"`
	InstanceID   string `json:"instance_id"`
	TargetString string `json:"target_string"`
	Vnum         int    `json:"vnum"`
	HealthPct    int    `json:"health_pct,omitempty"`
	Fighting     bool   `json:"fighting"`
	Stance       string `json:"stance,omitempty"`
}

type Item struct {
	ID    string   `json:"id,omitempty"`
	Name  string   `json:"name"`
	Count int      `json:"count,omitempty"`
	Flags []string `json:"flags,omitempty"`
}

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Connect establishes a WebSocket connection and authenticates.
func (a *AgentClient) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("ws://%s:%d/ws", a.Cfg.GameHost, a.Cfg.GamePort)
	slog.Debug("connecting", "addr", addr)

	conn, err := Dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	login := map[string]any{
		"type": "login",
		"data": map[string]any{
			"player_name":    "",
			"api_key":        a.Cfg.EffectiveKey(),
			"mode":           "agent",
			"context_budget": a.Cfg.Tier,
			"valence":        a.Cfg.Valence,
		},
	}
	if err := conn.WriteJSON(login); err != nil {
		conn.Close()
		return fmt.Errorf("login send: %w", err)
	}

	var resp struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := conn.ReadJSON(&resp); err != nil {
		conn.Close()
		return fmt.Errorf("login read: %w", err)
	}
	if resp.Type == "error" {
		conn.Close()
		return fmt.Errorf("login rejected: %s", string(resp.Data))
	}

	sub := map[string]any{
		"type": "subscribe",
		"data": map[string]any{
			"variables": []string{
				"HEALTH", "MAX_HEALTH", "MANA", "LEVEL", "EXP",
				"ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS", "ROOM_ITEMS",
				"FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS",
			},
		},
	}
	if err := conn.WriteJSON(sub); err != nil {
		conn.Close()
		return fmt.Errorf("subscribe: %w", err)
	}

	a.conn = conn
	a.state = &GameState{}
	a.session = NewSessionLogger()
	return nil
}

func (a *AgentClient) RunDecisionLoop(ctx context.Context) error {
	if a.conn == nil {
		return fmt.Errorf("not connected")
	}

	msgCh := make(chan []byte, 64)
	go func() {
		defer close(msgCh)
		for {
			_, msg, err := a.conn.ReadMessage()
			if err != nil {
				slog.Warn("read error", "error", err)
				return
			}
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return a.finalizeSession()
		case msg, ok := <-msgCh:
			if !ok {
				return a.finalizeSession()
			}
			if err := a.handleMessage(ctx, msg); err != nil {
				slog.Error("handle message", "error", err)
			}
		}
	}
}

func (a *AgentClient) handleMessage(ctx context.Context, raw []byte) error {
	var env struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	switch env.Type {
	case "vars":
		return a.handleVars(ctx, env.Data)
	case "state":
		return json.Unmarshal(env.Data, a.state)
	case "event":
		slog.Debug("event", "data", string(env.Data))
		return nil
	case "error":
		slog.Error("server error", "msg", string(env.Data))
		return nil
	default:
		return nil
	}
}

func (a *AgentClient) handleVars(ctx context.Context, data json.RawMessage) error {
	var vars struct {
		HEALTH     int      `json:"HEALTH"`
		MAX_HEALTH int      `json:"MAX_HEALTH"`
		MANA       int      `json:"MANA"`
		LEVEL      int      `json:"LEVEL"`
		EXP        int      `json:"EXP"`
		ROOM_VNUM  int      `json:"ROOM_VNUM"`
		ROOM_NAME  string   `json:"ROOM_NAME"`
		ROOM_EXITS []string `json:"ROOM_EXITS"`
		ROOM_MOBS  []Mob    `json:"ROOM_MOBS"`
		ROOM_ITEMS []Item   `json:"ROOM_ITEMS,omitempty"`
		FIGHTING   string   `json:"FIGHTING"`
	}
	if err := json.Unmarshal(data, &vars); err != nil {
		return fmt.Errorf("parse vars: %w", err)
	}

	a.state.Player.Health = vars.HEALTH
	a.state.Player.MaxHealth = vars.MAX_HEALTH
	a.state.Player.Mana = vars.MANA
	a.state.Player.Level = vars.LEVEL
	a.state.Player.Exp = vars.EXP
	a.state.Room.Vnum = vars.ROOM_VNUM
	a.state.Room.Name = vars.ROOM_NAME
	a.state.Room.Exits = vars.ROOM_EXITS
	a.state.Room.Mobs = vars.ROOM_MOBS
	a.state.Room.Items = vars.ROOM_ITEMS
	a.state.Fighting = vars.FIGHTING

	if action := FSMDecision(a.state); action != nil {
		return a.executeAction(ctx, action, 0)
	}
	return a.llmDecision(ctx)
}

func (a *AgentClient) llmDecision(ctx context.Context) error {
	msg := []map[string]string{
		{"role": "system", "content": SystemPrompt},
	}
	if a.state.MemorySummary != "" {
		msg = append(msg, map[string]string{
			"role":    "system",
			"content": a.state.MemorySummary,
		})
	}
	msg = append(msg, map[string]string{
		"role":    "user",
		"content": BuildPrompt(a.state),
	})

	temp := a.Cfg.Temperature
	start := time.Now()

	resp, err := CallLLM(a.Cfg.LiteLLM, "sk-lab…-key", a.Cfg.ModelFast, msg, 30*time.Second, temp)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		slog.Warn("llm error, trying fallback", "error", err)
		start = time.Now()
		resp, err = CallLLM(a.Cfg.LiteLLM, "sk-lab…-key", a.Cfg.ModelFallback, msg, 60*time.Second, temp)
		latencyMs = time.Since(start).Milliseconds()
		if err != nil {
			return fmt.Errorf("llm fallback failed: %w", err)
		}
	}

	return a.executeAction(ctx, resp, latencyMs)
}

func (a *AgentClient) executeAction(ctx context.Context, action *LLMResponse, latencyMs int64) error {
	cmd := map[string]any{
		"type": "command",
		"data": map[string]any{
			"command": action.ActionType,
			"args":    action.Args,
		},
	}
	if err := a.conn.WriteJSON(cmd); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	a.session.Log(LogEntry{
		RoomVnum:  a.state.Room.Vnum,
		RoomName:  a.state.Room.Name,
		HP:        a.state.Player.Health,
		MaxHP:     a.state.Player.MaxHealth,
		Fighting:  a.state.Fighting,
		Action:    action.ActionType,
		Args:      action.Args,
		SayLine:   action.SayLine,
		LatencyMs: latencyMs,
	})
	return nil
}

// PushCommand sends a pre-constructed action to the server for one-shot exec mode.
// It logs the command but does not call the LLM or FSM.
func (a *AgentClient) PushCommand(ctx context.Context, action *LLMResponse) error {
	cmd := map[string]any{
		"type": "command",
		"data": map[string]any{
			"command": action.ActionType,
			"args":    action.Args,
		},
	}
	if err := a.conn.WriteJSON(cmd); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	a.session.Log(LogEntry{
		RoomVnum:  a.state.Room.Vnum,
		RoomName:  a.state.Room.Name,
		HP:        a.state.Player.Health,
		MaxHP:     a.state.Player.MaxHealth,
		Fighting:  a.state.Fighting,
		Action:    action.ActionType,
		Args:      action.Args,
		SayLine:   action.SayLine,
	})
	return nil
}

func (a *AgentClient) finalizeSession() error {
	if a.session != nil {
		summary := a.session.Finalize(a.Cfg.LogDir)

		// Export JSONL if log directory is configured.
		if a.Cfg.LogDir != "" {
			ts := a.session.Started().Format("2006-01-02-150405")
			logPath := filepath.Join(a.Cfg.LogDir, "sessions", a.state.Player.Name, ts+".jsonl")
			if n, err := a.session.WriteJSONL(logPath); err != nil {
				slog.Warn("write jsonl", "error", err)
			} else {
				// Also write summary JSON alongside the log.
				sumPath := filepath.Join(a.Cfg.LogDir, "summaries", a.state.Player.Name, ts+".json")
				writeJSON(sumPath, summary)
				slog.Info("session saved", "entries", n, "path", logPath)
			}
		}

		slog.Info("session done",
			"turns", summary.Turns,
			"duration", summary.Duration.Round(time.Second),
			"avg_latency_ms", summary.AvgLatencyMs,
		)
	}
	if a.conn != nil {
		a.conn.Close()
	}
	return nil
}

func (a *AgentClient) Close() error {
	return a.finalizeSession()
}

// writeJSON is a helper to marshal v to path as indented JSON.
func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		slog.Warn("marshal json", "error", err)
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Warn("mkdir", "error", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("write", "error", err)
	}
}
