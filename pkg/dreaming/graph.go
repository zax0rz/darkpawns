package dreaming

import (
	"fmt"
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

// BuildSummary produces a string suitable for LLM context injection.
// Max tokens parameter controls truncation.
func (g *MemoryGraph) BuildSummary(maxTokens int) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.Nodes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Memory\n\n")

	// Sort nodes by salience descending.
	sorted := sortBySalience(g.Nodes)

	// Top-K nodes by salience.
	limit := maxTokens / 20 // rough estimate: ~20 tokens per memory line
	if limit > len(sorted) {
		limit = len(sorted)
	}

	written := 0
	for _, n := range sorted[:limit] {
		line := formatNode(n, g.Edges)
		if b.Len()+len(line) > maxTokens*4 { // rough char-to-token estimate
			b.WriteString("(additional memories consolidated)\n")
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
		written++
	}

	return b.String()
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

func sortBySalience(nodes map[string]*Node) []*Node {
	sorted := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		sorted = append(sorted, n)
	}
	// Simple bubble sort (small n — memory graphs stay under 1000 nodes).
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Salience > sorted[i].Salience {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatNode(n *Node, edges []Edge) string {
	valence := ""
	switch {
	case n.Valence > 0:
		valence = fmt.Sprintf(" [+%d]", n.Valence)
	case n.Valence < 0:
		valence = fmt.Sprintf(" [%d]", n.Valence)
	}
	return fmt.Sprintf("- %s%s (salience: %.2f)", n.Label, valence, n.Salience)
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
