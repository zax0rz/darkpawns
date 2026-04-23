// memory_hooks.go — REM synthesis integration hooks for Python AI system.
//
// Extends the existing narrative memory system with Python AI integration.
// When memory events occur, they're sent to the Python system for:
// - Emotional tagging
// - Forgetting policy evaluation
// - Privacy disclosure analysis
// - REM synthesis (memory consolidation during downtime)
//
// Fire-and-forget: HTTP calls run in goroutines, never blocking game loop.

package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// PythonSystemConfig holds configuration for the Python AI system integration.
type PythonSystemConfig struct {
	BaseURL string        // e.g., "http://localhost:8000"
	Timeout time.Duration // HTTP timeout
	Enabled bool          // Whether to send events to Python system
	APIKey  string        // Optional API key for authentication
}

// MemoryEvent represents a memory event to send to the Python system.
type MemoryEvent struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"` // "mob_kill", "mob_death", "social_interaction"
	AgentName     string    `json:"agent_name"`
	Summary       string    `json:"summary"`
	RoomVNum      int       `json:"room_vnum"`
	RoomName      string    `json:"room_name"`
	RelatedEntity string    `json:"related_entity"`
	RelatedVNum   int       `json:"related_vnum"`
	Valence       int       `json:"valence"`
	Salience      float64   `json:"salience"`
	SocialEventID string    `json:"social_event_id"`
	SessionID     string    `json:"session_id"`
	Timestamp     time.Time `json:"timestamp"`
	RawEventData  string    `json:"raw_event_data,omitempty"` // JSON string of original event
}

// REMSynthesisClient handles communication with the Python AI system.
type REMSynthesisClient struct {
	config    PythonSystemConfig
	client    *http.Client
	endpoints struct {
		memoryEvent    string // POST /api/memory/event
		remConsolidate string // POST /api/rem/consolidate
		retrievalLog   string // POST /api/retrieval/log
	}
}

// NewREMSynthesisClient creates a new client for Python AI system integration.
func NewREMSynthesisClient(config PythonSystemConfig) *REMSynthesisClient {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8000"
	}

	client := &REMSynthesisClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
	client.endpoints.memoryEvent = config.BaseURL + "/api/memory/event"
	client.endpoints.remConsolidate = config.BaseURL + "/api/rem/consolidate"
	client.endpoints.retrievalLog = config.BaseURL + "/api/retrieval/log"

	return client
}

// SendMemoryEvent sends a memory event to the Python system for processing.
// This includes emotional tagging, forgetting policy evaluation, and privacy disclosure.
func (c *REMSynthesisClient) SendMemoryEvent(event *MemoryEvent) error {
	if !c.config.Enabled {
		return nil // silently skip if disabled
	}

	// Generate event ID if not provided
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt_%s_%d", event.AgentName, time.Now().UnixNano())
	}
	event.Timestamp = time.Now()

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal memory event: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoints.memoryEvent, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send memory event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	slog.Info("memory event sent to Python system", "event_type", event.EventType, "agent_name", event.AgentName)
	return nil
}

// TriggerREMSynthesis triggers REM consolidation for an agent during downtime.
// Called by the scheduler when the agent is inactive.
func (c *REMSynthesisClient) TriggerREMSynthesis(agentName string) error {
	if !c.config.Enabled {
		return nil
	}

	payload := map[string]string{
		"agent_name": agentName,
		"trigger":    "downtime",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal REM trigger: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoints.remConsolidate, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create REM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("trigger REM synthesis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	slog.Info("synthesis triggered for agent", "agent_name", agentName)
	return nil
}

// LogRetrieval logs when a memory is accessed (retrieved) by the AI system.
// Used for tracking memory access patterns and importance.
func (c *REMSynthesisClient) LogRetrieval(agentName, memoryID, context string) error {
	if !c.config.Enabled {
		return nil
	}

	payload := map[string]string{
		"agent_name": agentName,
		"memory_id":  memoryID,
		"context":    context,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal retrieval log: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoints.retrievalLog, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create retrieval log request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("log retrieval: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// ConvertNarrativeMemoryToEvent converts a database NarrativeMemory to a MemoryEvent.
func ConvertNarrativeMemoryToEvent(mem *db.NarrativeMemory, rawEvent interface{}) *MemoryEvent {
	event := &MemoryEvent{
		EventID:       fmt.Sprintf("mem_%d", mem.ID),
		EventType:     mem.EventType,
		AgentName:     mem.AgentName,
		Summary:       mem.Summary,
		RoomVNum:      mem.RoomVNum,
		RoomName:      mem.RoomName,
		RelatedEntity: mem.RelatedEntity,
		RelatedVNum:   mem.RelatedVNum,
		Valence:       mem.Valence,
		Salience:      mem.Salience,
		SocialEventID: mem.SocialEventID,
		SessionID:     mem.SessionID,
		Timestamp:     mem.CreatedAt,
	}

	// Add raw event data if provided
	if rawEvent != nil {
		if data, err := json.Marshal(rawEvent); err == nil {
			event.RawEventData = string(data)
		}
	}

	return event
}

// ConvertGameEventToMemoryEvent converts a game event to a MemoryEvent.
func ConvertGameEventToMemoryEvent(eventType string, agentName, summary string, roomVNum int, roomName, relatedEntity string, relatedVNum, valence int, salience float64, socialEventID, sessionID string) *MemoryEvent {
	return &MemoryEvent{
		EventID:       fmt.Sprintf("game_%s_%d", agentName, time.Now().UnixNano()),
		EventType:     eventType,
		AgentName:     agentName,
		Summary:       summary,
		RoomVNum:      roomVNum,
		RoomName:      roomName,
		RelatedEntity: relatedEntity,
		RelatedVNum:   relatedVNum,
		Valence:       valence,
		Salience:      salience,
		SocialEventID: socialEventID,
		SessionID:     sessionID,
		Timestamp:     time.Now(),
	}
}

// HookManager manages the integration between Go server and Python AI system.
type HookManager struct {
	client *REMSynthesisClient
	db     *db.DB
}

// NewHookManager creates a new hook manager.
func NewHookManager(db *db.DB, config PythonSystemConfig) *HookManager {
	return &HookManager{
		client: NewREMSynthesisClient(config),
		db:     db,
	}
}

// OnMobKill handles mob kill events.
func (hm *HookManager) OnMobKill(evt *game.MobKillEvent, agentName, sessionID string) {
	// First write to database (existing behavior)
	valence := 1
	salience := db.SalienceScore(valence, false, false)

	mem := &db.NarrativeMemory{
		AgentName:     agentName,
		EventType:     db.NarrEventMobKill,
		Summary:       fmt.Sprintf("Killed %s (vnum %d) in %s (room %d).", evt.VictimName, evt.VictimVNum, evt.RoomName, evt.RoomVNum),
		RoomVNum:      evt.RoomVNum,
		RoomName:      evt.RoomName,
		RelatedEntity: evt.VictimName,
		RelatedVNum:   evt.VictimVNum,
		Valence:       valence,
		Salience:      salience,
		SessionID:     sessionID,
	}

	id, err := hm.db.WriteNarrativeMemory(mem)
	if err != nil {
		slog.Error("failed to write mob_kill", "agent_name", agentName, "error", err)
		return
	}
	mem.ID = id

	// Send to Python system
	event := ConvertNarrativeMemoryToEvent(mem, evt)
	go func() {
		if err := hm.client.SendMemoryEvent(event); err != nil {
			slog.Error("failed to send mob_kill to Python system", "error", err)
		}
	}()
}

// OnPlayerDeath handles player death events.
func (hm *HookManager) OnPlayerDeath(evt *game.PlayerDeathEvent, agentName, sessionID string) {
	valence := -2
	if !evt.KillerIsNPC && evt.KillerName != "" {
		valence = -3
	}
	salience := db.SalienceScore(valence, false, false)

	killerDesc := evt.KillerName
	if killerDesc == "" {
		killerDesc = "unknown"
	}

	mem := &db.NarrativeMemory{
		AgentName:     agentName,
		EventType:     db.NarrEventMobDeath,
		Summary:       fmt.Sprintf("Killed by %s in %s (room %d). Lost experience.", killerDesc, evt.RoomName, evt.RoomVNum),
		RoomVNum:      evt.RoomVNum,
		RoomName:      evt.RoomName,
		RelatedEntity: killerDesc,
		Valence:       valence,
		Salience:      salience,
		SessionID:     sessionID,
	}

	id, err := hm.db.WriteNarrativeMemory(mem)
	if err != nil {
		slog.Error("failed to write mob_death", "agent_name", agentName, "error", err)
		return
	}
	mem.ID = id

	// Send to Python system
	event := ConvertNarrativeMemoryToEvent(mem, evt)
	go func() {
		if err := hm.client.SendMemoryEvent(event); err != nil {
			slog.Error("failed to send mob_death to Python system", "error", err)
		}
	}()
}

// OnSocialInteraction handles social interaction events.
func (hm *HookManager) OnSocialInteraction(agentName, otherEntity, interactionType, summary, sessionID string, roomVNum int, roomName string) {
	valence := 0                                       // neutral by default, can be adjusted
	salience := db.SalienceScore(valence, true, false) // social events get bonus

	mem := &db.NarrativeMemory{
		AgentName:     agentName,
		EventType:     db.NarrEventPlayerEncounter,
		Summary:       summary,
		RoomVNum:      roomVNum,
		RoomName:      roomName,
		RelatedEntity: otherEntity,
		Valence:       valence,
		Salience:      salience,
		SocialEventID: fmt.Sprintf("social_%s_%s_%d", agentName, otherEntity, time.Now().UnixNano()),
		SessionID:     sessionID,
	}

	id, err := hm.db.WriteNarrativeMemory(mem)
	if err != nil {
		slog.Error("failed to write social interaction", "agent_name", agentName, "error", err)
		return
	}
	mem.ID = id

	// Send to Python system
	event := ConvertNarrativeMemoryToEvent(mem, nil)
	go func() {
		if err := hm.client.SendMemoryEvent(event); err != nil {
			slog.Error("failed to send social interaction to Python system", "error", err)
		}
	}()
}

// OnMemoryRetrieved logs when a memory is accessed.
func (hm *HookManager) OnMemoryRetrieved(agentName, memoryID, context string) {
	go func() {
		if err := hm.client.LogRetrieval(agentName, memoryID, context); err != nil {
			slog.Error("failed to log retrieval", "error", err)
		}
	}()
}

// TriggerAgentREMSynthesis triggers REM consolidation for an agent.
func (hm *HookManager) TriggerAgentREMSynthesis(agentName string) {
	go func() {
		if err := hm.client.TriggerREMSynthesis(agentName); err != nil {
			slog.Error("failed to trigger REM synthesis", "error", err)
		}
	}()
}
