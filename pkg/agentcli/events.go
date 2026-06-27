package agentcli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AgentEvent represents a single game event with sequence number.
// Named AgentEvent to avoid collision with client.go's Event type.
type AgentEvent struct {
	Seq       uint64          `json:"seq"`
	Timestamp time.Time       `json:"ts"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}

// tellEvent is a parsed tell message from the event buffer.
type tellEvent struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

// EventBuffer is an append-only event log with sequence tracking.
// Events are stored as JSONL (one JSON object per line) in
// ~/.dp-goat/events/<name>.jsonl.
type EventBuffer struct {
	mu       sync.Mutex
	path     string
	nextSeq  uint64
	events   []AgentEvent
	maxCache int // max events to keep in memory (default: 1000)
}

// NewEventBuffer creates an event buffer for the given character.
func NewEventBuffer(name string) (*EventBuffer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".dp-goat", "events")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir events: %w", err)
	}
	eb := &EventBuffer{
		path:     filepath.Join(dir, name+".jsonl"),
		events:   make([]AgentEvent, 0, 256),
		maxCache: 1000,
		nextSeq:  1, // start at 1 so Since(0) returns all events
	}

	// Load existing events to recover sequence number
	if err := eb.loadExisting(); err != nil {
		return nil, err
	}

	return eb, nil
}

// loadExisting reads the JSONL file to find the highest seq number.
func (eb *EventBuffer) loadExisting() error {
	f, err := os.Open(eb.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh start
		}
		return fmt.Errorf("open events: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var lastSeq uint64
	var count int
	for dec.More() {
		var ev AgentEvent
		if err := dec.Decode(&ev); err != nil {
			slog.Warn("corrupt event entry, stopping", "error", err)
			break
		}
		if ev.Seq > lastSeq {
			lastSeq = ev.Seq
		}
		count++
	}

	eb.nextSeq = lastSeq + 1
	slog.Debug("event buffer loaded", "events", count, "next_seq", eb.nextSeq)
	return nil
}

// Append adds an event to the buffer and persists it to disk.
func (eb *EventBuffer) Append(eventType string, data interface{}) (uint64, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	rawData, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("marshal event data: %w", err)
	}

	ev := AgentEvent{
		Seq:       eb.nextSeq,
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Data:      rawData,
	}
	eb.nextSeq++

	// Append to in-memory cache
	eb.events = append(eb.events, ev)
	if len(eb.events) > eb.maxCache {
		eb.events = eb.events[len(eb.events)-eb.maxCache:]
	}

	// Append to disk (JSONL)
	line, err := json.Marshal(ev)
	if err != nil {
		return ev.Seq, fmt.Errorf("marshal event: %w", err)
	}
	line = append(line, '\n')

	f, err := os.OpenFile(eb.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return ev.Seq, fmt.Errorf("open events: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return ev.Seq, fmt.Errorf("write event: %w", err)
	}

	return ev.Seq, nil
}

// Since returns all events with seq > afterSeq.
func (eb *EventBuffer) Since(afterSeq uint64) []AgentEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	var result []AgentEvent
	for _, ev := range eb.events {
		if ev.Seq > afterSeq {
			result = append(result, ev)
		}
	}
	return result
}

// Recent returns the last N events.
func (eb *EventBuffer) Recent(n int) []AgentEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if n >= len(eb.events) {
		result := make([]AgentEvent, len(eb.events))
		copy(result, eb.events)
		return result
	}
	result := make([]AgentEvent, n)
	copy(result, eb.events[len(eb.events)-n:])
	return result
}

// NextSeq returns the next sequence number that will be assigned.
func (eb *EventBuffer) NextSeq() uint64 {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	return eb.nextSeq
}

// CompactionWindow groups events into a compact summary for LLM context.
// Returns a human-readable summary of what happened since afterSeq.
func (eb *EventBuffer) CompactionWindow(afterSeq uint64) string {
	events := eb.Since(afterSeq)
	if len(events) == 0 {
		return "Nothing happened while you were away."
	}

	// Simple compaction: group by type, count occurrences
	typeCounts := make(map[string]int)
	var lastRoom string
	var lastTell *tellEvent

	for _, ev := range events {
		typeCounts[ev.Type]++

		// Extract room name from room events
		if ev.Type == "room_enter" || ev.Type == "state" {
			var room struct {
				Name string `json:"ROOM_NAME"`
			}
			if json.Unmarshal(ev.Data, &room) == nil && room.Name != "" {
				lastRoom = room.Name
			}
		}

		// Extract tells
		if ev.Type == "tell" {
			var t tellEvent
			if json.Unmarshal(ev.Data, &t) == nil {
				tCopy := t
				lastTell = &tCopy
			}
		}
	}

	// Build summary
	summary := fmt.Sprintf("%d events since last check-in.", len(events))
	if lastRoom != "" {
		summary += fmt.Sprintf(" You're in %s.", lastRoom)
	}
	if lastTell != nil {
		summary += fmt.Sprintf(" %s told you: %q", lastTell.From, lastTell.Message)
	}

	// Notable event types
	delete(typeCounts, "state") // state updates are noisy
	if len(typeCounts) > 0 {
		summary += " Events: "
		first := true
		for typ, count := range typeCounts {
			if !first {
				summary += ", "
			}
			if count == 1 {
				summary += typ
			} else {
				summary += fmt.Sprintf("%s(x%d)", typ, count)
			}
			first = false
		}
	}

	return summary
}

// Truncate clears the event buffer and removes the file.
// Used after a successful context handoff to the mind.
func (eb *EventBuffer) Truncate() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.events = eb.events[:0]
	eb.nextSeq = 1

	if err := os.Remove(eb.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove events: %w", err)
	}
	return nil
}
