package agentcli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ContextPacket is the full context delivered to the LLM mind when it wakes.
type ContextPacket struct {
	State     *GameState `json:"state"`
	Summary   string     `json:"summary"`              // narrative compaction
	Events    []AgentEvent `json:"events"`              // recent events (raw)
	GeneratedAt time.Time `json:"generated_at"`
	SeqFloor  uint64     `json:"seq_floor"`            // events newer than this are "new"
}

// ContextBuilder generates context packets from state + event buffer.
type ContextBuilder struct {
	maxRecentEvents int // max raw events to include (default: 20)
}

// NewContextBuilder creates a context builder with sensible defaults.
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		maxRecentEvents: 20,
	}
}

// Build generates a context packet from the current state and event buffer.
// seqFloor is the last sequence number the mind saw — events above this are "new".
func (cb *ContextBuilder) Build(state *GameState, events *EventBuffer, seqFloor uint64) *ContextPacket {
	recent := events.Recent(cb.maxRecentEvents)
	summary := cb.CompactNarrative(events, seqFloor, state)

	return &ContextPacket{
		State:     state,
		Summary:   summary,
		Events:    recent,
		GeneratedAt: time.Now().UTC(),
		SeqFloor:  seqFloor,
	}
}

// CompactNarrative produces a human-readable summary of what happened since seqFloor.
// This is what the LLM sees as "what happened while you were away."
func (cb *ContextBuilder) CompactNarrative(events *EventBuffer, seqFloor uint64, state *GameState) string {
	since := events.Since(seqFloor)
	if len(since) == 0 {
		if state.Fighting != "" {
			return fmt.Sprintf("You are fighting %s.", state.Fighting)
		}
		return "Nothing happened while you were away."
	}

	var parts []string

	// Track state changes for narrative
	var lastRoom string
	// Initialize lastRoom from current state so the first room change is captured.
	if state != nil && state.Room.Name != "" {
		lastRoom = state.Room.Name
	}
	var roomTransitions []string
	var combatRounds int
	var tells []tellRecord
	var says []sayRecord
	var itemsGotten []string
	var itemsDropped []string
	var playersEntered []string
	var errors []string
	var death bool
	var combatStarted bool
	var combatEnded bool

	for _, ev := range since {
		switch ev.Type {
		case "vars":
			var vars struct {
				ROOM_NAME string `json:"ROOM_NAME"`
				HEALTH    *int   `json:"HEALTH"`
			}
			if json.Unmarshal(ev.Data, &vars) == nil {
				if vars.ROOM_NAME != "" && vars.ROOM_NAME != lastRoom {
					if lastRoom != "" {
						roomTransitions = append(roomTransitions, vars.ROOM_NAME)
					}
					lastRoom = vars.ROOM_NAME
				}
				if vars.HEALTH != nil && *vars.HEALTH == 0 {
					death = true
				}
			}
		case "state":
			var st struct {
				Room struct {
					Name string `json:"name"`
				} `json:"room"`
			}
			if json.Unmarshal(ev.Data, &st) == nil {
				if st.Room.Name != "" && st.Room.Name != lastRoom {
					if lastRoom != "" {
						roomTransitions = append(roomTransitions, st.Room.Name)
					}
					lastRoom = st.Room.Name
				}
			}
		case "combat_start":
			combatStarted = true
		case "combat_tick":
			combatRounds++
		case "combat_end":
			combatEnded = true
		case "tell":
			var t struct {
				From    string `json:"from"`
				Message string `json:"message"`
			}
			if json.Unmarshal(ev.Data, &t) == nil {
				tells = append(tells, tellRecord{From: t.From, Message: t.Message})
			}
		case "say":
			var s struct {
				From    string `json:"from"`
				Message string `json:"message"`
			}
			if json.Unmarshal(ev.Data, &s) == nil {
				says = append(says, sayRecord{From: s.From, Message: s.Message})
			}
		case "item_get":
			var ig struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(ev.Data, &ig) == nil && ig.Name != "" {
				itemsGotten = append(itemsGotten, ig.Name)
			}
		case "item_drop":
			var id struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(ev.Data, &id) == nil && id.Name != "" {
				itemsDropped = append(itemsDropped, id.Name)
			}
		case "entity_enter":
			var pe struct {
				Name string `json:"name"`
				Type string `json:"entity_type"`
			}
			if json.Unmarshal(ev.Data, &pe) == nil && pe.Type == "player" && pe.Name != "" {
				playersEntered = append(playersEntered, pe.Name)
			}
		case "error":
			var er struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(ev.Data, &er) == nil && er.Message != "" {
				errors = append(errors, er.Message)
			}
		}
	}

	// Build narrative parts
	if death {
		parts = append(parts, "You died.")
	}

	if len(roomTransitions) > 0 {
		if len(roomTransitions) == 1 {
			parts = append(parts, fmt.Sprintf("You moved to %s.", roomTransitions[0]))
		} else {
			path := strings.Join(roomTransitions, " → ")
			parts = append(parts, fmt.Sprintf("You traveled: %s.", path))
		}
	}

	if combatStarted && !combatEnded {
		if state.Fighting != "" {
			parts = append(parts, fmt.Sprintf("You are fighting %s.", state.Fighting))
		} else {
			parts = append(parts, "A combat started.")
		}
	} else if combatStarted && combatEnded {
		parts = append(parts, "You fought a battle.")
	} else if combatRounds > 0 {
		if combatRounds == 1 {
			parts = append(parts, "Combat occurred.")
		} else {
			parts = append(parts, fmt.Sprintf("Combat lasted %d rounds.", combatRounds))
		}
	}

	for _, t := range tells {
		parts = append(parts, fmt.Sprintf("%s told you: %q", t.From, t.Message))
	}

	for _, s := range says {
		parts = append(parts, fmt.Sprintf("%s said: %q", s.From, s.Message))
	}

	if len(itemsGotten) > 0 {
		parts = append(parts, fmt.Sprintf("You picked up: %s.", strings.Join(itemsGotten, ", ")))
	}

	if len(itemsDropped) > 0 {
		parts = append(parts, fmt.Sprintf("You dropped: %s.", strings.Join(itemsDropped, ", ")))
	}

	if len(playersEntered) > 0 {
		parts = append(parts, fmt.Sprintf("Players nearby: %s.", strings.Join(playersEntered, ", ")))
	}

	if len(errors) > 0 {
		if len(errors) <= 3 {
			for _, e := range errors {
				parts = append(parts, fmt.Sprintf("Error: %s", e))
			}
		} else {
			parts = append(parts, fmt.Sprintf("[%d errors occurred]", len(errors)))
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%d events occurred.", len(since))
	}

	return strings.Join(parts, " ")
}

type tellRecord struct {
	From    string
	Message string
}

type sayRecord struct {
	From    string
	Message string
}

// FormatContext formats a ContextPacket as a prompt string for the LLM.
func FormatContext(pkt *ContextPacket) string {
	var b strings.Builder

	// State summary
	if pkt.State != nil {
		s := pkt.State
		b.WriteString(fmt.Sprintf("Room: %s (vnum %d)\n", s.Room.Name, s.Room.Vnum))

		if len(s.Room.Mobs) > 0 {
			b.WriteString("Mobs:\n")
			for _, m := range s.Room.Mobs {
				status := ""
				if m.Fighting {
					status = " [fighting]"
				}
				b.WriteString(fmt.Sprintf("  %s (hp:%d%%)%s\n", m.Name, m.HealthPct, status))
			}
		}

		b.WriteString(fmt.Sprintf("Exits: %v\n", s.Room.Exits))
		b.WriteString(fmt.Sprintf("HP: %d/%d  Mana: %d\n", s.Player.Health, s.Player.MaxHealth, s.Player.Mana))
		b.WriteString(fmt.Sprintf("Level: %d  Exp: %d  Gold: %d\n", s.Player.Level, s.Player.Exp, s.Player.Gold))

		if s.Fighting != "" {
			b.WriteString(fmt.Sprintf("Fighting: %s\n", s.Fighting))
		}
	}

	// Narrative summary
	if pkt.Summary != "" {
		b.WriteString("\nWhat happened:\n")
		b.WriteString(pkt.Summary)
		b.WriteString("\n")
	}

	b.WriteString("\nWhat do you do?")
	return b.String()
}
