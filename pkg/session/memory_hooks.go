// memory_hooks.go — Manager-side narrative memory hook registration.
//
// Manager knows which sessions are agents (s.isAgent). World does not.
// These hooks bridge the two: World fires events, Manager checks isAgent
// and writes to narrative memory when appropriate.
//
// Fire-and-forget: all DB writes run in goroutines, never blocking game loop.

package session

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// RegisterMemoryHooks wires narrative memory callbacks into the World.
// Call this once after NewManager(), before the server starts accepting connections.
func (m *Manager) RegisterMemoryHooks() {
	m.world.SetMobKillHook(func(evt *game.MobKillEvent) {
		m.onMobKill(evt)
	})
	m.world.SetPlayerDeathHook(func(evt *game.PlayerDeathEvent) {
		m.onPlayerDeath(evt)
	})
}

// onMobKill is called when any mob dies. Writes a narrative memory only if
// the killer is an active agent session.
func (m *Manager) onMobKill(evt *game.MobKillEvent) {
	if !m.hasDB || evt.KillerName == "" || evt.KillerIsNPC {
		return // only care about player/agent kills
	}

	m.mu.RLock()
	s, ok := m.sessions[evt.KillerName]
	m.mu.RUnlock()
	if !ok || !s.isAgent {
		return // killer is not an active agent session
	}

	// Compute valence: +1 neutral kill, +2 if it was dangerous (can refine later)
	valence := 1
	salience := db.SalienceScore(valence, false, false)

	mem := &db.NarrativeMemory{
		AgentName:     evt.KillerName,
		EventType:     db.NarrEventMobKill,
		Summary:       fmt.Sprintf("Killed %s (vnum %d) in %s (room %d).", evt.VictimName, evt.VictimVNum, evt.RoomName, evt.RoomVNum),
		RoomVNum:      evt.RoomVNum,
		RoomName:      evt.RoomName,
		RelatedEntity: evt.VictimName,
		RelatedVNum:   evt.VictimVNum,
		Valence:       valence,
		Salience:      salience,
		SessionID:     s.sessionID(),
	}

	id, err := m.db.WriteNarrativeMemory(mem)
	if err != nil {
		log.Printf("[MEMORY] failed to write mob_kill for %s: %v", evt.KillerName, err)
		return
	}
	log.Printf("[MEMORY] mob_kill recorded for agent %s (id=%d): %s", evt.KillerName, id, mem.Summary)
}

// onPlayerDeath is called when any player/agent dies.
func (m *Manager) onPlayerDeath(evt *game.PlayerDeathEvent) {
	if !m.hasDB {
		return
	}

	m.mu.RLock()
	s, ok := m.sessions[evt.VictimName]
	m.mu.RUnlock()
	if !ok || !s.isAgent {
		return // only write memories for agent deaths
	}

	// Death is always negative valence.
	// Killed by NPC = -2. (Bad, painful, but expected.)
	// Killed by player (future) = -3. (Catastrophic — someone actively hunted you.)
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
		AgentName:     evt.VictimName,
		EventType:     db.NarrEventMobDeath,
		Summary:       fmt.Sprintf("Killed by %s in %s (room %d). Lost experience.", killerDesc, evt.RoomName, evt.RoomVNum),
		RoomVNum:      evt.RoomVNum,
		RoomName:      evt.RoomName,
		RelatedEntity: killerDesc,
		Valence:       valence,
		Salience:      salience,
		SessionID:     s.sessionID(),
	}

	id, err := m.db.WriteNarrativeMemory(mem)
	if err != nil {
		log.Printf("[MEMORY] failed to write mob_death for %s: %v", evt.VictimName, err)
		return
	}
	log.Printf("[MEMORY] mob_death recorded for agent %s (id=%d): %s", evt.VictimName, id, mem.Summary)
}

// sessionID returns a stable session identifier for the current connection.
// Uses connect timestamp — good enough for now, revisit if we need UUIDs.
func (s *Session) sessionID() string {
	return fmt.Sprintf("%s-%d", s.playerName, s.connectedAt.UnixNano())
}

// SendMemoryBootstrap fetches this agent's narrative memories and sends them
// as a dedicated "memory_bootstrap" message. Called on agent login after sendFullVarDump().
//
// Message format:
//
//	{"type": "memory_bootstrap", "data": {"block": "<formatted text>", "count": N}}
//
// Context budget (from PHASE4-AGENT-PROTOCOL.md):
//
//	Default: 15 memories + last 3 session summaries ("medium" tier)
//	Agents can request more in auth message (future: context_budget field)
func (s *Session) SendMemoryBootstrap() {
	if !s.manager.hasDB {
		return
	}

	// Medium tier: 15 memories + 3 session summaries
	memories, err := s.manager.db.BootstrapMemories(s.playerName, 15)
	if err != nil {
		log.Printf("[MEMORY] bootstrap fetch error for %s: %v", s.playerName, err)
		return
	}

	summaries, err := s.manager.db.GetSessionSummaries(s.playerName, 3)
	if err != nil {
		log.Printf("[MEMORY] summary fetch error for %s: %v", s.playerName, err)
		// Non-fatal: continue with just memories
	}

	if len(memories) == 0 && len(summaries) == 0 {
		// No memories yet — still send the message so agent knows bootstrap completed
		log.Printf("[MEMORY] no memories for agent %s (first session)", s.playerName)
	}

	block := db.BootstrapBlock(memories, summaries)

	// Send as a structured message — agent reads data.block into LLM context
	msg, err := json.Marshal(map[string]interface{}{
		"type": "memory_bootstrap",
		"data": map[string]interface{}{
			"block":     block,
			"count":     len(memories),
			"summaries": len(summaries),
		},
	})
	if err != nil {
		log.Printf("json.Marshal failed in SendMemoryBootstrap: %v", err)
		return
	}
	select {
	case s.send <- msg:
		log.Printf("[MEMORY] sent bootstrap to agent %s: %d memories, %d summaries", s.playerName, len(memories), len(summaries))
	default:
		log.Printf("[MEMORY] bootstrap send blocked for %s (channel full)", s.playerName)
	}
}

var _ = time.Now // ensure time import used
