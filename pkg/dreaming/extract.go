package dreaming

import (
	"fmt"
	"time"
)

// ExtractedEvent represents one meaningful event extracted from a session log entry.
type ExtractedEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Kind      string    `json:"kind"` // combat, movement, social, acquisition, death, damage
	AgentID   string    `json:"agent_id"`
	
	// Entity references
	TargetEntity  string `json:"target_entity,omitempty"`
	TargetRoom    int    `json:"target_room,omitempty"`
	TargetRoomName string `json:"target_room_name,omitempty"`
	ItemName       string `json:"item_name,omitempty"`

	// Valence
	Valence int `json:"valence"` // -3 to +3

	// Narrative text for summary
	Narrative string `json:"narrative,omitempty"`
}

// ExtractEvents converts a LogEntry-like struct into memory events.
// In practice this reads from the JSONL session log.
type LogEvent struct {
	Timestamp time.Time `json:"timestamp"`
	RoomVnum  int       `json:"room_vnum"`
	RoomName  string    `json:"room_name,omitempty"`
	HP        int       `json:"hp"`
	MaxHP     int       `json:"max_hp"`
	MobsPresent int     `json:"mobs_present"`
	Fighting  string    `json:"fighting,omitempty"`
	Action    string    `json:"action"`
	Args      []string  `json:"args,omitempty"`
	SayLine   string    `json:"say_line,omitempty"`
	LatencyMs int64     `json:"latency_ms,omitempty"`
}

// ExtractEventsFromEntry analyzes a single log entry and returns any events worth remembering.
func ExtractEventsFromEntry(entry LogEvent, agentID string) []ExtractedEvent {
	var events []ExtractedEvent
	ts := entry.Timestamp

	switch entry.Action {
	case "hit", "kill":
		target := ""
		if len(entry.Args) > 0 {
			target = entry.Args[0]
		}
		events = append(events, ExtractedEvent{
			Timestamp:    ts,
			Kind:         "combat",
			AgentID:      agentID,
			TargetEntity: target,
			TargetRoom:   entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:      1, // mild positive — agent is acting
			Narrative:    fmt.Sprintf("Attacked %s in %s", target, entry.RoomName),
		})

	case "flee":
		events = append(events, ExtractedEvent{
			Timestamp:    ts,
			Kind:         "combat",
			AgentID:      agentID,
			TargetEntity: entry.Fighting,
			TargetRoom:   entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:      -2, // negative — forced retreat
			Narrative:    fmt.Sprintf("Fled from %s at low HP (%d/%d)", entry.Fighting, entry.HP, entry.MaxHP),
		})

	case "say":
		sayText := ""
		if len(entry.Args) > 0 {
			sayText = entry.Args[0]
		} else if entry.SayLine != "" {
			sayText = entry.SayLine
		}
		if sayText != "" {
			events = append(events, ExtractedEvent{
				Timestamp:    ts,
				Kind:         "social",
				AgentID:      agentID,
				TargetRoom:   entry.RoomVnum,
				TargetRoomName: entry.RoomName,
				Valence:      0, // neutral — content matters for exact valence
				Narrative:    fmt.Sprintf("Said \"%s\" in %s", sayText, entry.RoomName),
			})
		}

	case "get":
		item := ""
		if len(entry.Args) > 0 {
			item = entry.Args[0]
		}
		events = append(events, ExtractedEvent{
			Timestamp:    ts,
			Kind:         "acquisition",
			AgentID:      agentID,
			ItemName:     item,
			TargetRoom:   entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:      1,
			Narrative:    fmt.Sprintf("Picked up %s in %s", item, entry.RoomName),
		})

	case "north", "south", "east", "west":
		// Movement is recorded as a room transition, captured by tracking room_vnum changes.
		// The extractor handles this externally by comparing consecutive entries.
	}

	// Track damage taken as negative events.
	if entry.MaxHP > 0 && entry.HP <= entry.MaxHP/4 && entry.Fighting != "" {
		events = append(events, ExtractedEvent{
			Timestamp:    ts,
			Kind:         "damage",
			AgentID:      agentID,
			TargetEntity: entry.Fighting,
			TargetRoom:   entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:      -2,
			Narrative:    fmt.Sprintf("Low HP (%d/%d) while fighting %s", entry.HP, entry.MaxHP, entry.Fighting),
		})
	}

	return events
}
