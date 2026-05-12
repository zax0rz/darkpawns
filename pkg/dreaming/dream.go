package dreaming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DreamConfig controls the dreaming cycle.
type DreamConfig struct {
	SessionDir   string // path to data/sessions/{agent_id}/
	OutputDir    string // path to write consolidated memory
	AgentID      string // agent to dream for
	GraphConfig  GraphConfig
	DryRun       bool // if true, print what would happen without writing
}

// DreamResult summarizes a dreaming run.
type DreamResult struct {
	AgentID        string `json:"agent_id"`
	SessionFiles   int    `json:"session_files_processed"`
	EventsExtracted int   `json:"events_extracted"`
	NodesBefore    int    `json:"nodes_before_consolidation"`
	NodesAfter     int    `json:"nodes_after_consolidation"`
	Pruned         int    `json:"nodes_pruned"`
	SummaryTokens  int    `json:"summary_tokens_estimated"`
}

// RunDream executes one dreaming cycle: read sessions, extract events, build graph, consolidate.
func RunDream(cfg DreamConfig) (*DreamResult, error) {
	graph := NewMemoryGraph(cfg.GraphConfig)

	// 1. Load existing graph if available.
	graphFile := filepath.Join(cfg.OutputDir, cfg.AgentID, "memory-graph.json")
	if data, err := os.ReadFile(graphFile); err == nil {
		if err := json.Unmarshal(data, graph); err != nil {
			return nil, fmt.Errorf("load existing graph: %w", err)
		}
		// Reinitialize mutex after unmarshal.
		graph.mu = sync.RWMutex{}
	}

	// 2. Find and process session logs.
	sessionPath := filepath.Join(cfg.SessionDir, cfg.AgentID)
	entries, err := os.ReadDir(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("read sessions: %w", err)
	}

	// Sort by name (timestamps) so we process in chronological order.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var allEvents []ExtractedEvent
	sessionFiles := 0
	lastRoom := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		fileEvents, err := processSessionFile(filepath.Join(sessionPath, entry.Name()), cfg.AgentID, &lastRoom)
		if err != nil {
			return nil, fmt.Errorf("process %s: %w", entry.Name(), err)
		}
		allEvents = append(allEvents, fileEvents...)
		sessionFiles++
	}

	result := &DreamResult{
		AgentID:         cfg.AgentID,
		SessionFiles:    sessionFiles,
		EventsExtracted: len(allEvents),
	}

	if cfg.DryRun {
		result.NodesBefore = len(graph.Nodes)
		result.NodesAfter = len(graph.Nodes)
		return result, nil
	}

	// 3. Inject events into graph.
	result.NodesBefore = len(graph.Nodes)

	for _, ev := range allEvents {
		eventID := fmt.Sprintf("%s-%s-%d", cfg.AgentID, ev.Kind, ev.Timestamp.UnixNano())
		graph.AddOrReinforceNode(eventID, NodeKindEvent, ev.Narrative, ev.Valence)

		// Link event → room.
		roomID := fmt.Sprintf("room-%d", ev.TargetRoom)
		graph.AddOrReinforceNode(roomID, NodeKindRoom, ev.TargetRoomName, 0)
		graph.AddEdge(eventID, roomID, EdgeKindOccurredIn)

		// Link event → entity.
		if ev.TargetEntity != "" {
			entID := fmt.Sprintf("entity-%s", ev.TargetEntity)
			graph.AddOrReinforceNode(entID, NodeKindEntity, ev.TargetEntity, ev.Valence)
			graph.AddEdge(eventID, entID, EdgeKindInvolved)
		}

		// Link event → item.
		if ev.ItemName != "" {
			itemID := fmt.Sprintf("item-%s", ev.ItemName)
			graph.AddOrReinforceNode(itemID, NodeKindItem, ev.ItemName, 0)
			graph.AddEdge(eventID, itemID, EdgeKindTookFrom)
		}
	}

	// 4. Consolidate (decay + prune).
	result.Pruned = graph.Consolidate()
	result.NodesAfter = len(graph.Nodes)

	// 5. Generate summary.
	summary := graph.BuildSummary(500) // ~500 tokens of memory summary
	result.SummaryTokens = estimateTokens(summary)

	// 6. Write output.
	if err := os.MkdirAll(filepath.Dir(graphFile), 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	// Write the full graph.
	graphData, _ := json.MarshalIndent(graph, "", "  ")
	if err := os.WriteFile(graphFile, graphData, 0644); err != nil {
		return nil, fmt.Errorf("write graph: %w", err)
	}

	// Write the summary separately for quick inspection.
	summaryFile := filepath.Join(cfg.OutputDir, cfg.AgentID, "memory-summary.txt")
	if err := os.WriteFile(summaryFile, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("write summary: %w", err)
	}

	// Write the dream result as JSON.
	resultFile := filepath.Join(cfg.OutputDir, cfg.AgentID, "dream-result.json")
	resultData, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile(resultFile, resultData, 0644)

	return result, nil
}

// processSessionFile reads a JSONL file and returns extracted events.
func processSessionFile(path, agentID string, lastRoom *int) ([]ExtractedEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []ExtractedEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry LogEvent
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed lines
		}

		// Detect room transitions.
		if entry.RoomVnum != 0 && entry.RoomVnum != *lastRoom {
			if *lastRoom != 0 {
				events = append(events, ExtractedEvent{
					Timestamp:      entry.Timestamp,
					Kind:           "movement",
					AgentID:        agentID,
					TargetRoom:     entry.RoomVnum,
					TargetRoomName: entry.RoomName,
					Valence:        0,
					Narrative:      fmt.Sprintf("Moved to %s", entry.RoomName),
				})
			}
			*lastRoom = entry.RoomVnum
		}

		entryEvents := ExtractEventsFromEntry(entry, agentID)
		events = append(events, entryEvents...)
	}

	return events, scanner.Err()
}

func estimateTokens(s string) int {
	// Rough estimate: ~4 chars per token for LLM context.
	return len(s) / 4
}
