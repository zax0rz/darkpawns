package dreaming

import (
	"fmt"
	"math"
	"strings"
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

// LogEvent represents a single line from a JSONL session log.
type LogEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	RoomVnum    int       `json:"room_vnum"`
	RoomName    string    `json:"room_name,omitempty"`
	HP          int       `json:"hp"`
	MaxHP       int       `json:"max_hp"`
	AgentLevel  int       `json:"agent_level,omitempty"` // agent's current level
	MobsPresent int       `json:"mobs_present"`
	Fighting    string    `json:"fighting,omitempty"`
	MobLevel    int       `json:"mob_level,omitempty"` // level of mob being fought (if known)
	Action      string    `json:"action"`
	Args        []string  `json:"args,omitempty"`
	SayLine     string    `json:"say_line,omitempty"`
	LatencyMs   int64     `json:"latency_ms,omitempty"`

	// Social context
	SocialTarget string `json:"social_target,omitempty"` // who was interacted with
	SocialType   string `json:"social_type,omitempty"`   // gift, betrayal, cooperation, etc.

	// Item context
	ItemLevel int `json:"item_level,omitempty"` // approximate item quality (0-100)

	// Outcome tracking
	CombatResult string `json:"combat_result,omitempty"` // "won", "lost", "fled", ""
}

// --- Content-aware valence heuristics ---

// KillValence: how good was this kill?
//
//   - Rat (mobLevel << agentLevel): +0, barely worth remembering
//   - Challenging mob (mobLevel ~ agentLevel): +1, solid fight
//   - Tough mob (mobLevel > agentLevel): +2, hard-won
//   - Boss / dragon (mobLevel >> agentLevel): +3, epic
//   - Unknown mob level: default +1 (mild positive — agent acted)
func KillValence(agentLevel, mobLevel int) int {
	if agentLevel <= 0 || mobLevel <= 0 {
		return 1 // unknown levels, default
	}

	diff := mobLevel - agentLevel
	switch {
	case diff <= -10:
		return 0 // trivial — a rat at level 20
	case diff <= -3:
		return 1 // easy but not trivial
	case diff <= 3:
		return 2 // challenging — worthy opponent
	default:
		return 3 // epic — mob outlevels the agent
	}
}

// fleeValence: how bad was this retreat?
//
//   - Fleeing at full HP: -3, cowardly or something went very wrong
//   - Fleeing at low HP: -1, tactical retreat, expected
//   - Fleeing at critical HP (<20%): 0, survival instinct, barely negative
func fleeValence(hpPct int) int {
	switch {
	case hpPct >= 80:
		return -3 // fled while barely scratched — something is wrong
	case hpPct >= 40:
		return -2 // fled at half health — embarrassing but understandable
	case hpPct >= 20:
		return -1 // fled while hurt — tactical retreat
	default:
		return 0 // fled at death's door — survival, not failure
	}
}

// socialValence: how emotionally significant was this social event?
//
// Social valence is context-dependent. A gift to a friend is positive.
// A betrayal is deeply negative. Cooperation is mildly positive.
func socialValence(socialType, sayLine string) int {
	low := strings.ToLower(socialType)
	switch low {
	case "betrayal", "backstab", "attack_ally":
		return -3
	case "gift", "give":
		return 2
	case "cooperation", "assist", "heal_ally":
		return 1
	case "insult", "threaten":
		return -2
	case "greet", "introduce":
		return 1
	case "trade", "exchange":
		return 1
	default:
		// Heuristic: positive words in speech → mild positive
		if sayLine != "" {
			return speechValence(sayLine)
		}
		return 0
	}
}

// speechValence does simple keyword-based sentiment on say lines.
// Not perfect, but captures obvious positive/negative speech.
func speechValence(text string) int {
	lower := strings.ToLower(text)
	positive := []string{"thank", "thanks", "help", "please", "friend", "ally", "good", "great", "well done", "nice", "hello", "hi"}
	negative := []string{"hate", "kill", "die", "stupid", "idiot", "enemy", "betray", "traitor", "fool", "curse"}

	pos, neg := 0, 0
	for _, w := range positive {
		if strings.Contains(lower, w) {
			pos++
		}
	}
	for _, w := range negative {
		if strings.Contains(lower, w) {
			neg++
		}
	}

	switch {
	case pos > neg:
		return 1
	case neg > pos:
		return -1
	default:
		return 0
	}
}

// acquisitionValence: how important was this item?
//
// Items are hard to evaluate without game knowledge, but we can use
// ItemLevel as a proxy. Higher-level items are more memorable.
func acquisitionValence(itemLevel int) int {
	switch {
	case itemLevel >= 80:
		return 3 // legendary find
	case itemLevel >= 50:
		return 2 // valuable acquisition
	case itemLevel >= 20:
		return 1 // useful item
	default:
		return 1 // default mild positive
	}
}

// damageValence: how bad was getting hit?
//
// Low HP while fighting is negative, but the closer to death,
// the more traumatic (and memorable). Near-death at 1% HP is
// more impactful than dropping to 60%.
func damageValence(hp, maxHP int) int {
	if maxHP <= 0 {
		return -1
	}
	hpPct := (hp * 100) / maxHP
	switch {
	case hpPct <= 5:
		return -3 // near-death
	case hpPct <= 15:
		return -2 // badly hurt
	case hpPct <= 30:
		return -1 // moderate damage
	default:
		return 0 // not significant enough to remember
	}
}

// ComputeValence is the main entry point for content-aware valence.
// It replaces the old hardcoded valence assignments.
func ComputeValence(entry LogEvent) int {
	hpPct := 100
	if entry.MaxHP > 0 {
		hpPct = (entry.HP * 100) / entry.MaxHP
	}

	switch entry.Action {
	case "hit", "kill":
		return KillValence(entry.AgentLevel, entry.MobLevel)
	case "flee":
		return fleeValence(hpPct)
	case "say":
		return socialValence(entry.SocialType, entry.SayLine)
	case "get":
		return acquisitionValence(entry.ItemLevel)
	case "north", "south", "east", "west", "up", "down":
		return 0 // movement is neutral
	default:
		// Check for low HP damage event
		if entry.MaxHP > 0 && entry.HP <= entry.MaxHP/4 && entry.Fighting != "" {
			return damageValence(entry.HP, entry.MaxHP)
		}
		return 0
	}
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
		valence := KillValence(entry.AgentLevel, entry.MobLevel)

		narrative := fmt.Sprintf("Attacked %s in %s", target, entry.RoomName)
		if entry.Action == "kill" {
			narrative = fmt.Sprintf("Killed %s in %s", target, entry.RoomName)
		}

		events = append(events, ExtractedEvent{
			Timestamp:      ts,
			Kind:           "combat",
			AgentID:        agentID,
			TargetEntity:   target,
			TargetRoom:     entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:        valence,
			Narrative:      narrative,
		})

	case "flee":
		valence := fleeValence(percentHP(entry.HP, entry.MaxHP))
		events = append(events, ExtractedEvent{
			Timestamp:      ts,
			Kind:           "combat",
			AgentID:        agentID,
			TargetEntity:   entry.Fighting,
			TargetRoom:     entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:        valence,
			Narrative:      fmt.Sprintf("Fled from %s at low HP (%d/%d)", entry.Fighting, entry.HP, entry.MaxHP),
		})

	case "say":
		sayText := ""
		if len(entry.Args) > 0 {
			sayText = entry.Args[0]
		} else if entry.SayLine != "" {
			sayText = entry.SayLine
		}
		if sayText != "" {
			valence := socialValence(entry.SocialType, sayText)
			events = append(events, ExtractedEvent{
				Timestamp:      ts,
				Kind:           "social",
				AgentID:        agentID,
				TargetRoom:     entry.RoomVnum,
				TargetRoomName: entry.RoomName,
				Valence:        valence,
				Narrative:      fmt.Sprintf("Said \"%s\" in %s", sayText, entry.RoomName),
			})
		}

	case "get":
		item := ""
		if len(entry.Args) > 0 {
			item = entry.Args[0]
		}
		valence := acquisitionValence(entry.ItemLevel)
		events = append(events, ExtractedEvent{
			Timestamp:      ts,
			Kind:           "acquisition",
			AgentID:        agentID,
			ItemName:       item,
			TargetRoom:     entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:        valence,
			Narrative:      fmt.Sprintf("Picked up %s in %s", item, entry.RoomName),
		})

	case "north", "south", "east", "west":
		// Movement is recorded as a room transition, captured by tracking room_vnum changes.
		// The extractor handles this externally by comparing consecutive entries.
	}

	// Track damage taken as negative events — only when significantly hurt.
	if entry.MaxHP > 0 && entry.HP <= entry.MaxHP/4 && entry.Fighting != "" {
		valence := damageValence(entry.HP, entry.MaxHP)
		events = append(events, ExtractedEvent{
			Timestamp:      ts,
			Kind:           "damage",
			AgentID:        agentID,
			TargetEntity:   entry.Fighting,
			TargetRoom:     entry.RoomVnum,
			TargetRoomName: entry.RoomName,
			Valence:        valence,
			Narrative:      fmt.Sprintf("Low HP (%d/%d) while fighting %s", entry.HP, entry.MaxHP, entry.Fighting),
		})
	}

	return events
}

// percentHP computes HP as a percentage, guarding against division by zero.
func percentHP(hp, maxHP int) int {
	if maxHP <= 0 {
		return 100
	}
	return int(math.Round(float64(hp) / float64(maxHP) * 100))
}
