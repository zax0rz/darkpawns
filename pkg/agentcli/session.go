package agentcli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogEntry represents a single decision turn.
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	RoomVnum    int       `json:"room_vnum"`
	RoomName    string    `json:"room_name,omitempty"`
	HP          int       `json:"hp"`
	MaxHP       int       `json:"max_hp"`
	MobsPresent int       `json:"mobs_present"`
	Fighting    string    `json:"fighting,omitempty"`
	Action      string    `json:"action"`
	Args        []string  `json:"args,omitempty"`
	SayLine     string    `json:"say_line,omitempty"`
	LatencyMs   int64     `json:"latency_ms,omitempty"`
	TokensUsed  int       `json:"tokens_used,omitempty"`
}

// SessionSummary holds aggregate stats for a completed session.
type SessionSummary struct {
	AgentID     string        `json:"agent_id"`
	Turns       int           `json:"turns"`
	Duration    time.Duration `json:"duration"`
	AvgLatencyMs float64      `json:"avg_latency_ms"`

	// Per-session derived stats (computed on finalize).
	RoomsVisited   int `json:"rooms_visited"`
	CombatEncounters int `json:"combat_encounters"`
	GoalsCompleted int   `json:"goals_completed"`
}

// SessionLogger records per-decision log entries and computes summaries.
// Eventually these are sent to the server's Postgres DB for the paper pipeline.
// For now they're held in memory and printed at session end.
type SessionLogger struct {
	entries  []LogEntry
	started  time.Time
	duration time.Duration
}

// NewSessionLogger creates a new session logger.
func NewSessionLogger() *SessionLogger {
	return &SessionLogger{
		entries: make([]LogEntry, 0, 1000),
		started: time.Now(),
	}
}

// Started returns the session start time.
func (s *SessionLogger) Started() time.Time {
	return s.started
}

// Log appends a decision entry.
func (s *SessionLogger) Log(entry LogEntry) {
	entry.Timestamp = time.Now()
	s.entries = append(s.entries, entry)
}

// WriteJSONL exports all log entries as newline-delimited JSON to path.
// Creates parent directories if needed. Returns the number of bytes written.
func (s *SessionLogger) WriteJSONL(path string) (int64, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	var total int64
	for _, entry := range s.entries {
		if err := enc.Encode(entry); err != nil {
			return total, fmt.Errorf("encode entry %d: %w", total, err)
		}
		total++
	}
	return total, nil
}

// Finalize computes summary, exports JSONL to logDir if configured, and returns summary.
func (s *SessionLogger) Finalize(logDir string) *SessionSummary {
	s.duration = time.Since(s.started)

	summary := &SessionSummary{
		Turns:     len(s.entries),
		Duration:  s.duration,
	}

	if len(s.entries) > 0 {
		var totalLatency int64
		rooms := make(map[int]bool)
		combat := 0
		for _, e := range s.entries {
			totalLatency += e.LatencyMs
			rooms[e.RoomVnum] = true
			if e.Action == "hit" || e.Action == "kill" || e.Action == "flee" {
				combat++
			}
		}
		summary.AvgLatencyMs = float64(totalLatency) / float64(len(s.entries))
		summary.RoomsVisited = len(rooms)
		summary.CombatEncounters = combat
	}

	return summary
}

// Entries returns all logged entries.
func (s *SessionLogger) Entries() []LogEntry {
	return s.entries
}
