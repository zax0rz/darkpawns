package dreaming

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// NodeKind categorizes memory graph nodes.
type NodeKind string

const (
	NodeKindEvent  NodeKind = "event"
	NodeKindEntity NodeKind = "entity"
	NodeKindRoom   NodeKind = "room"
	NodeKindItem   NodeKind = "item"
)

// EdgeKind describes the relationship between two nodes.
type EdgeKind string

const (
	EdgeKindOccurredIn     EdgeKind = "occurred_in"      // event → room
	EdgeKindInvolved       EdgeKind = "involved"          // event → entity
	EdgeKindTransitionedTo EdgeKind = "transitioned_to"   // room → room (sequential movement)
	EdgeKindKilled         EdgeKind = "killed"            // entity → entity
	EdgeKindTookFrom       EdgeKind = "took_from"         // entity → item
	EdgeKindFought         EdgeKind = "fought"            // entity ↔ entity (mutual combat)
	EdgeKindSocial         EdgeKind = "social"            // entity → entity (speech, cooperation)
	EdgeKindSimilarTo      EdgeKind = "similar_to"        // event → event (consolidation)
)

// Node in the memory graph.
type Node struct {
	ID        string    `json:"id"`
	Kind      NodeKind  `json:"kind"`
	Label     string    `json:"label"`
	Salience  float64   `json:"salience"`   // 0.0–1.0, decays over time
	Valence   int       `json:"valence"`    // -3 to +3
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	VisitCount int      `json:"visit_count"`
}

// Edge in the memory graph.
type Edge struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Kind      EdgeKind  `json:"kind"`
	Weight    float64   `json:"weight"`     // reinforcement weight
	CreatedAt time.Time `json:"created_at"`
}

// MemoryGraph holds the consolidated narrative memory for one agent.
type MemoryGraph struct {
	mu     sync.RWMutex
	Nodes  map[string]*Node `json:"nodes"`
	Edges  []Edge           `json:"edges"`
	config GraphConfig
}

// GraphConfig controls consolidation behavior.
type GraphConfig struct {
	DecayRate        float64       // fraction of salience lost per consolidation cycle (default 0.1)
	PruneThreshold   float64       // salience below this → prune (default 0.05)
	ReinforceBonus   float64       // salience bonus for re-encountering (default 0.2)
	ConsolidateEvery time.Duration // min time between consolidation runs (default 1h)
}

// DefaultGraphConfig returns sensible defaults.
func DefaultGraphConfig() GraphConfig {
	return GraphConfig{
		DecayRate:        0.1,
		PruneThreshold:   0.05,
		ReinforceBonus:   0.2,
		ConsolidateEvery: time.Hour,
	}
}

// NewMemoryGraph creates an empty memory graph.
func NewMemoryGraph(cfg GraphConfig) *MemoryGraph {
	return &MemoryGraph{
		Nodes:  make(map[string]*Node),
		Edges:  make([]Edge, 0, 1000),
		config: cfg,
	}
}

// AddOrReinforceNode finds or creates a node, reinforcing it if it exists.
func (g *MemoryGraph) AddOrReinforceNode(id string, kind NodeKind, label string, valence int) *Node {
	g.mu.Lock()
	defer g.mu.Unlock()

	if existing, ok := g.Nodes[id]; ok {
		existing.VisitCount++
		existing.Salience = clamp(existing.Salience+g.config.ReinforceBonus, 0, 1)
		existing.UpdatedAt = time.Now()
		// Valence blends toward most recent.
		existing.Valence = blendValence(existing.Valence, valence, existing.VisitCount)
		return existing
	}

	n := &Node{
		ID:        id,
		Kind:      kind,
		Label:     label,
		Salience:  1.0,
		Valence:   valence,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		VisitCount: 1,
	}
	g.Nodes[id] = n
	return n
}

// AddEdge creates a directed edge between two nodes.
func (g *MemoryGraph) AddEdge(from, to string, kind EdgeKind) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check for existing edge and reinforce it.
	for i, e := range g.Edges {
		if e.From == from && e.To == to && e.Kind == kind {
			g.Edges[i].Weight += 0.1
			return
		}
	}

	g.Edges = append(g.Edges, Edge{
		From:      from,
		To:        to,
		Kind:      kind,
		Weight:    1.0,
		CreatedAt: time.Now(),
	})
}

// Consolidate runs one decay/prune cycle.
// Returns count of pruned nodes.
func (g *MemoryGraph) Consolidate() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Decay all node salience.
	for _, n := range g.Nodes {
		n.Salience -= g.config.DecayRate
		if n.Salience < 0 {
			n.Salience = 0
		}
		n.UpdatedAt = time.Now()
	}

	// Prune nodes below threshold (keep edges for now, they'll be cleaned on next read).
	pruned := 0
	for id, n := range g.Nodes {
		if n.Salience < g.config.PruneThreshold {
			delete(g.Nodes, id)
			pruned++
		}
	}

	// Clean orphaned edges.
	valid := make([]Edge, 0, len(g.Edges))
	for _, e := range g.Edges {
		if _, hasFrom := g.Nodes[e.From]; hasFrom {
			if _, hasTo := g.Nodes[e.To]; hasTo {
				valid = append(valid, e)
			}
		}
	}
	g.Edges = valid

	return pruned
}

// BuildSummary produces narrative prose suitable for LLM context injection.
// Events are ordered chronologically and grouped into sessions.
// High-salience events get full sentences; low-salience ones are summarized.
// Max tokens parameter controls truncation.
func (g *MemoryGraph) BuildSummary(maxTokens int) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.Nodes) == 0 {
		return ""
	}

	// Collect all event nodes and sort by creation time.
	events := make([]*Node, 0)
	for _, n := range g.Nodes {
		if n.Kind == NodeKindEvent {
			events = append(events, n)
		}
	}
	if len(events) == 0 {
		return ""
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	// Group into sessions by time gap (>30 min = new session).
	sessions := groupBySession(events, 30*time.Minute)

	var b strings.Builder
	b.WriteString("## Memory\n\n")

	maxChars := maxTokens * 4 // rough char-to-token estimate

	for si, session := range sessions {
		if b.Len() >= maxChars {
			b.WriteString("\n_(additional memories consolidated)_\n")
			break
		}

		// Session header with time range.
		if len(session) > 0 {
			date := session[0].CreatedAt.Format("Jan 2")
			if len(session) > 1 {
				date += " " + session[0].CreatedAt.Format("3:04 PM") + " – " + session[len(session)-1].CreatedAt.Format("3:04 PM")
			} else {
				date += " at " + session[0].CreatedAt.Format("3:04 PM")
			}
			fmt.Fprintf(&b, "### Session %d — %s\n\n", si+1, date)
		}

		// Render each event as narrative prose.
		for _, n := range session {
			if b.Len() >= maxChars {
				break
			}
			line := formatNarrativeEvent(n)
			if line != "" {
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Append entity relationship summary.
	entitySummary := g.buildEntitySummary()
	if entitySummary != "" && b.Len() < maxChars {
		b.WriteString("### Relationships\n\n")
		b.WriteString(entitySummary)
	}

	return b.String()
}

// groupBySession splits event nodes into sessions based on time gaps.
func groupBySession(events []*Node, gap time.Duration) [][]*Node {
	if len(events) == 0 {
		return nil
	}
	var sessions [][]*Node
	current := []*Node{events[0]}

	for i := 1; i < len(events); i++ {
		if events[i].CreatedAt.Sub(events[i-1].CreatedAt) > gap {
		sessions = append(sessions, current)
		current = []*Node{events[i]}
		} else {
			current = append(current, events[i])
		}
	}
	sessions = append(sessions, current)
	return sessions
}

// formatNarrativeEvent renders a single event node as a sentence.
// High-salience events get more detail; low-salience ones get summarized away.
func formatNarrativeEvent(n *Node) string {
	if n.Salience < 0.15 {
		return "" // too faded to include
	}

	// The label IS the narrative (set by extract.go).
	label := n.Label
	if label == "" {
		return ""
	}

	// Add valence context when strongly valenced.
	switch {
	case n.Valence >= 3:
		label += " (a significant moment)"
	case n.Valence == 2:
		label += " (noteworthy)"
	case n.Valence <= -2:
		label += " (a difficult moment)"
	case n.Valence == -1:
		label += " (unpleasant)"
	}

	return label + "."
}

// buildEntitySummary lists known entities and their accumulated valence.
func (g *MemoryGraph) buildEntitySummary() string {
	var entities []*Node
	for _, n := range g.Nodes {
		if n.Kind == NodeKindEntity {
			entities = append(entities, n)
		}
	}
	if len(entities) == 0 {
		return ""
	}

	// Sort by absolute valence (most emotionally charged first).
	sort.Slice(entities, func(i, j int) bool {
		return abs(entities[i].Valence) > abs(entities[j].Valence)
	})

	var b strings.Builder
	for _, n := range entities {
		if n.Valence == 0 && n.VisitCount <= 1 {
			continue // skip neutral one-off encounters
		}
		relationship := "met"
		switch {
		case n.Valence >= 3:
			relationship = "trusted ally"
		case n.Valence >= 2:
			relationship = "friendly"
		case n.Valence == 1:
			relationship = "acquaintance"
		case n.Valence == -1:
			relationship = "unfriendly"
		case n.Valence <= -2:
			relationship = "dangerous"
		}
		fmt.Fprintf(&b, "%s — %s (met %d time%s)\n", n.Label, relationship, n.VisitCount, plural(n.VisitCount))
	}
	return b.String()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- internal helpers ---

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func blendValence(oldVal, newVal int, visits int) int {
	// Weighted average: recent events count more up to a cap.
	w := 1.0 / float64(min(visits, 10))
	blended := float64(oldVal)*(1-w) + float64(newVal)*w
	return int(clamp(blended, -3, 3))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- summary formatting for different node kinds ---

// FormatEntityMemory creates a human-readable entity memory line.
func FormatEntityMemory(name string, valence int, lastEncounter string) string {
	v := ""
	switch {
	case valence > 0:
		v = "friendly"
	case valence < 0:
		v = "hostile"
	default:
		v = "neutral"
	}
	return fmt.Sprintf("%s (%s, last seen %s)", name, v, lastEncounter)
}
