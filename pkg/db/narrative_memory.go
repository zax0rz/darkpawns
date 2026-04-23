// Package db — narrative memory layer for Dark Pawns agents.
//
// Architecture (from PHASE4-AGENT-PROTOCOL.md, locked 2026-04-21):
//
//   Postgres agent_narrative_memory  = server-written objective facts.
//   mem0/Qdrant dp_brenda_memory     = agent-written subjective experience.
//
// These are separate systems. Server writes facts ("killed an orc in room 5042").
// Agent writes feelings ("that fight was scrappy — came in at 40% HP, barely won").
// Both feed into LLM context at bootstrap. No duplication when scoped correctly.
//
// Salience decay: nightly cron (scripts/dp_salience_decay.py) halves scores
// older than 30 days, prunes below 0.05. High-valence events decay slower.

package db

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

// NarrativeMemory is one server-written narrative fact about an agent's experience.
// Stored in agent_narrative_memory, bootstrapped into agent LLM context on connect.
type NarrativeMemory struct {
	ID            int64
	AgentName     string  // character name of the agent this memory belongs to
	EventType     string  // "mob_kill", "mob_death", "item_loot", "player_encounter", "room_visit", "session_summary"
	Summary       string  // human-readable narrative sentence, e.g. "Killed an orc in The Sewers (room 5042)"
	RoomVNum      int     // where it happened (0 if not applicable)
	RoomName      string  // denormalized for bootstrap readability
	RelatedEntity string  // mob/player/item name involved
	RelatedVNum   int     // vnum of related mob/obj (0 if n/a)
	Valence       int     // emotional weight: -3 (catastrophic) to +3 (triumphant). 0 = neutral
	Salience      float64 // 0.0–1.0, decayed nightly by dp_salience_decay.py
	SocialEventID string  // UUID linking multiple agents' perspectives on the same event
	SessionID     string  // which play session this came from
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NarrativeEventType constants — server writes these, never invent new ones without
// adding a corresponding hook in pkg/session/manager.go.
const (
	NarrEventMobKill         = "mob_kill"         // agent killed a mob
	NarrEventMobDeath        = "mob_death"        // agent was killed by a mob
	NarrEventPlayerEncounter = "player_encounter" // meaningful interaction with another player/agent
	NarrEventItemLoot        = "item_loot"        // agent looted a significant item
	NarrEventSessionSummary  = "session_summary"  // end-of-session LLM-generated consolidation
	NarrEventRoomVisit       = "room_visit"       // agent visited a notable room (first time, death room, etc.)
)

// InitNarrativeMemory creates the narrative memory tables if they don't exist.
// Called from DB.New() alongside createTables().
func (db *DB) InitNarrativeMemory() error {
	query := `
	CREATE TABLE IF NOT EXISTS agent_narrative_memory (
		id               BIGSERIAL PRIMARY KEY,
		agent_name       VARCHAR(64)   NOT NULL,
		event_type       VARCHAR(32)   NOT NULL,
		summary          TEXT          NOT NULL,
		room_vnum        INTEGER       NOT NULL DEFAULT 0,
		room_name        VARCHAR(128)  NOT NULL DEFAULT '',
		related_entity   VARCHAR(128),   -- e.g., 'mob:giant_rat', 'player:zach'
		related_vnum     INTEGER,         -- vnum of related mob/item, NULL if not applicable
		valence          SMALLINT      NOT NULL DEFAULT 0 CHECK (valence BETWEEN -3 AND 3),
		salience         REAL          NOT NULL DEFAULT 1.0 CHECK (salience BETWEEN 0.0 AND 1.0),
		social_event_id  VARCHAR(64),   -- NULL for non-social events
		session_id       VARCHAR(64)   NOT NULL DEFAULT '',
		created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
		updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_anm_agent_name    ON agent_narrative_memory(agent_name);
	CREATE INDEX IF NOT EXISTS idx_anm_event_type    ON agent_narrative_memory(event_type);
	CREATE INDEX IF NOT EXISTS idx_anm_salience      ON agent_narrative_memory(salience DESC);
	CREATE INDEX IF NOT EXISTS idx_anm_social_event  ON agent_narrative_memory(social_event_id) WHERE social_event_id != '';
	CREATE INDEX IF NOT EXISTS idx_anm_agent_session ON agent_narrative_memory(agent_name, session_id);

	-- Composite for bootstrap query: agent + salience DESC + recency
	CREATE INDEX IF NOT EXISTS idx_anm_bootstrap
		ON agent_narrative_memory(agent_name, salience DESC, created_at DESC);

	-- Session summaries table: LLM-generated consolidations written after each session.
	-- Separate from raw events — summaries are the distillation, events are the log.
	CREATE TABLE IF NOT EXISTS agent_session_summaries (
		id             BIGSERIAL PRIMARY KEY,
		agent_name     VARCHAR(64)  NOT NULL,
		session_id     VARCHAR(64)  NOT NULL UNIQUE,
		summary        TEXT         NOT NULL,
		event_count    INTEGER      NOT NULL DEFAULT 0,
		session_start  TIMESTAMPTZ,
		session_end    TIMESTAMPTZ,
		created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_ass_agent_name ON agent_session_summaries(agent_name);
	CREATE INDEX IF NOT EXISTS idx_ass_session_id ON agent_session_summaries(session_id);
	`

	_, err := db.conn.Exec(query)
	return err
}

// WriteNarrativeMemory inserts one narrative fact. Fire-and-forget safe to call
// from Manager callback hooks — does not block game loop.
func (db *DB) WriteNarrativeMemory(m *NarrativeMemory) (int64, error) {
	var id int64
	err := db.conn.QueryRow(`
		INSERT INTO agent_narrative_memory
			(agent_name, event_type, summary, room_vnum, room_name,
			 related_entity, related_vnum, valence, salience,
			 social_event_id, session_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		m.AgentName, m.EventType, m.Summary, m.RoomVNum, m.RoomName,
		m.RelatedEntity, m.RelatedVNum, m.Valence, m.Salience,
		m.SocialEventID, m.SessionID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write narrative memory: %w", err)
	}
	return id, nil
}

// BootstrapMemories returns the top N memories for an agent, ordered by salience
// then recency. Used to populate the LLM context block on agent connect.
//
// Context budget tiers (from PHASE4-AGENT-PROTOCOL.md):
//
//	small:     5 memories  (~200 tokens)
//	medium:   15 memories  (~600 tokens)
//	large:    30 memories  (~1200 tokens)
//	unlimited: no limit    (not recommended in production)
func (db *DB) BootstrapMemories(agentName string, limit int) ([]*NarrativeMemory, error) {
	rows, err := db.conn.Query(`
		SELECT id, agent_name, event_type, summary, room_vnum, room_name,
		       related_entity, related_vnum, valence, salience,
		       social_event_id, session_id, created_at, updated_at
		FROM agent_narrative_memory
		WHERE agent_name = $1
		  AND salience > 0.05
		ORDER BY salience DESC, created_at DESC
		LIMIT $2`,
		agentName, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("bootstrap memories: %w", err)
	}
	defer rows.Close()
	return scanNarrativeMemories(rows)
}

// RecentMemories returns memories from a specific session — used for session
// consolidation cron (scripts/dp_session_consolidate.py).
func (db *DB) RecentMemories(agentName, sessionID string) ([]*NarrativeMemory, error) {
	rows, err := db.conn.Query(`
		SELECT id, agent_name, event_type, summary, room_vnum, room_name,
		       related_entity, related_vnum, valence, salience,
		       social_event_id, session_id, created_at, updated_at
		FROM agent_narrative_memory
		WHERE agent_name = $1 AND session_id = $2
		ORDER BY created_at ASC`,
		agentName, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("recent memories: %w", err)
	}
	defer rows.Close()
	return scanNarrativeMemories(rows)
}

// SocialEventMemories returns all agent perspectives on a shared social event.
// Used by the research log automation to detect cross-agent memory references.
func (db *DB) SocialEventMemories(socialEventID string) ([]*NarrativeMemory, error) {
	if socialEventID == "" {
		return nil, fmt.Errorf("social_event_id cannot be empty")
	}
	rows, err := db.conn.Query(`
		SELECT id, agent_name, event_type, summary, room_vnum, room_name,
		       related_entity, related_vnum, valence, salience,
		       social_event_id, session_id, created_at, updated_at
		FROM agent_narrative_memory
		WHERE social_event_id = $1
		ORDER BY created_at ASC`,
		socialEventID,
	)
	if err != nil {
		return nil, fmt.Errorf("social event memories: %w", err)
	}
	defer rows.Close()
	return scanNarrativeMemories(rows)
}

// WriteSessionSummary stores a post-session LLM consolidation.
func (db *DB) WriteSessionSummary(agentName, sessionID, summary string, eventCount int, start, end time.Time) error {
	_, err := db.conn.Exec(`
		INSERT INTO agent_session_summaries
			(agent_name, session_id, summary, event_count, session_start, session_end)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (session_id) DO UPDATE
			SET summary = EXCLUDED.summary,
			    event_count = EXCLUDED.event_count,
			    session_end = EXCLUDED.session_end`,
		agentName, sessionID, summary, eventCount, start, end,
	)
	if err != nil {
		return fmt.Errorf("write session summary: %w", err)
	}
	return nil
}

// GetSessionSummaries returns the N most recent session summaries for an agent.
// Included in LLM bootstrap after raw memories (higher-level context).
func (db *DB) GetSessionSummaries(agentName string, limit int) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT summary FROM agent_session_summaries
		WHERE agent_name = $1
		ORDER BY session_end DESC NULLS LAST, created_at DESC
		LIMIT $2`,
		agentName, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get session summaries: %w", err)
	}
	defer rows.Close()

	var summaries []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// DecayStaleMemories is called by the nightly salience decay cron.
// Half-life: 30 days for neutral events (valence 0).
// High-valence events (|valence| >= 2) decay at half the rate.
// Prunes memories below salience 0.05.
// Returns count of decayed records, count pruned.
func (db *DB) DecayStaleMemories(cutoffDays int) (decayed, pruned int, err error) {
	if cutoffDays <= 0 {
		cutoffDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -cutoffDays)

	// Decay: multiply salience by 0.5 for neutral, 0.75 for high-valence
	// (high-valence = |valence| >= 2, per PHASE4-AGENT-PROTOCOL.md spec)
	result, err := db.conn.Exec(`
		UPDATE agent_narrative_memory
		SET salience = CASE
			WHEN ABS(valence) >= 2 THEN salience * 0.75
			ELSE salience * 0.5
		END,
		updated_at = NOW()
		WHERE created_at < $1 AND salience > 0.05`,
		cutoff,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("decay memories: %w", err)
	}
	n, _ := result.RowsAffected()
	decayed = int(n)

	// Prune below floor
	result, err = db.conn.Exec(`
		DELETE FROM agent_narrative_memory WHERE salience <= 0.05`,
	)
	if err != nil {
		return decayed, 0, fmt.Errorf("prune memories: %w", err)
	}
	n, _ = result.RowsAffected()
	pruned = int(n)

	return decayed, pruned, nil
}

// SalienceScore computes the initial salience for a new memory.
// Factors: outcome stakes (valence), social involvement, novelty flag.
// Returns a value in [0.1, 1.0].
//
// Formula from PHASE4-AGENT-PROTOCOL.md:
//
//	base = 0.5
//	+ valence component (high-magnitude events start higher)
//	+ social bonus (social_event_id present = other agents were involved)
//	+ novelty bonus (first time seeing this entity)
func SalienceScore(valence int, hasSocialEvent bool, isNovel bool) float64 {
	base := 0.5
	// Valence contributes up to ±0.3 — catastrophic/triumphant events start salient
	valenceComponent := float64(valence) / 10.0 // -0.3 to +0.3
	base += valenceComponent

	if hasSocialEvent {
		base += 0.15 // social involvement boosts salience
	}
	if isNovel {
		base += 0.1 // first encounter with entity/room
	}

	return math.Max(0.1, math.Min(1.0, base))
}

// BootstrapBlock renders narrative memories as a formatted LLM context block.
// Format matches PHASE4-AGENT-PROTOCOL.md section "Bootstrap Injection Format":
//
//	CHARACTER HISTORY → WORLD KNOWLEDGE → ACTIVE WARNINGS → CURRENT GOALS
//
// (goals are injected by dp_brenda.py, not here — this covers history + knowledge)
func BootstrapBlock(memories []*NarrativeMemory, sessionSummaries []string) string {
	if len(memories) == 0 && len(sessionSummaries) == 0 {
		return ""
	}

	block := "=== CHARACTER HISTORY ===\n"

	if len(sessionSummaries) > 0 {
		block += "\nRecent sessions:\n"
		for _, s := range sessionSummaries {
			block += "- " + s + "\n"
		}
	}

	// Separate memories into warnings (negative valence) and history (neutral/positive)
	var warnings, history []*NarrativeMemory
	for _, m := range memories {
		if m.Valence <= -2 {
			warnings = append(warnings, m)
		} else {
			history = append(history, m)
		}
	}

	if len(history) > 0 {
		block += "\n=== WORLD KNOWLEDGE ===\n"
		for _, m := range history {
			block += fmt.Sprintf("- %s\n", m.Summary)
		}
	}

	if len(warnings) > 0 {
		block += "\n=== ACTIVE WARNINGS ===\n"
		// Per spec: autobiographical context, not directives.
		// "Keldor took your gear. You haven't forgotten." NOT "Do not trust Keldor."
		for _, m := range warnings {
			block += fmt.Sprintf("- %s\n", m.Summary)
		}
	}

	return block
}

// --- internal helpers ---

func scanNarrativeMemories(rows *sql.Rows) ([]*NarrativeMemory, error) {
	var out []*NarrativeMemory
	for rows.Next() {
		m := &NarrativeMemory{}
		err := rows.Scan(
			&m.ID, &m.AgentName, &m.EventType, &m.Summary,
			&m.RoomVNum, &m.RoomName, &m.RelatedEntity, &m.RelatedVNum,
			&m.Valence, &m.Salience, &m.SocialEventID, &m.SessionID,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
