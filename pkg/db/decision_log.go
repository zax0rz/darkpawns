package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// createDecisionLogTables creates the decision_log and combat_log tables.
func (db *DB) createDecisionLogTables() error {
	query := `
	-- Decision log: one row per command executed by any player/agent.
	CREATE TABLE IF NOT EXISTS decision_log (
		id                BIGSERIAL,
		ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
		session_id        TEXT NOT NULL,
		player_name       TEXT NOT NULL,
		is_agent          BOOLEAN NOT NULL DEFAULT false,
		agent_harness     TEXT,
		agent_model       TEXT,
		turn_number       INT,
		session_elapsed   REAL,

		-- Command
		command           TEXT NOT NULL,
		command_class     TEXT,
		args              TEXT[],
		raw_input         TEXT,

		-- Decision source
		decision_source   TEXT,
		prompt_tokens     INT,
		completion_tokens INT,

		-- Pre-state
		pre_room          INT,
		pre_zone          INT,
		pre_health        INT,
		pre_max_health    INT,
		pre_mana          INT,
		pre_max_mana      INT,
		pre_move          INT,
		pre_max_move      INT,
		pre_level         INT,
		pre_fighting      BOOLEAN,
		pre_position      INT,
		pre_inv_count     INT,

		-- Post-state
		post_room         INT,
		post_zone         INT,
		post_health       INT,
		post_max_health   INT,
		post_mana         INT,
		post_max_mana     INT,
		post_move         INT,
		post_max_move     INT,
		post_level        INT,
		post_fighting     BOOLEAN,
		post_position     INT,
		post_inv_count    INT,

		-- Outcome
		outcome_category  TEXT NOT NULL,
		outcome_value     INT,
		outcome_text      TEXT,
		duration_ms       REAL,

		-- Partition key
		PRIMARY KEY (id, ts)
	) PARTITION BY RANGE (ts);

	-- Combat log: per-round detail during fights.
	CREATE TABLE IF NOT EXISTS combat_log (
		id                BIGSERIAL,
		ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
		decision_id       BIGINT,
		session_id        TEXT NOT NULL,
		round_number      INT,
		attacker_name     TEXT,
		attacker_vnum     INT,
		defender_name     TEXT,
		defender_vnum     INT,
		attack_type       TEXT,
		damage            INT,
		outcome           TEXT,

		-- Target state at time of attack
		target_health     INT,
		target_max_health INT,
		target_level      INT,
		target_position   INT,
		target_count      INT,

		PRIMARY KEY (id, ts)
	) PARTITION BY RANGE (ts);

	-- Indexes for common query patterns
	CREATE INDEX IF NOT EXISTS idx_decision_session ON decision_log(session_id, ts);
	CREATE INDEX IF NOT EXISTS idx_decision_player ON decision_log(player_name, ts);
	CREATE INDEX IF NOT EXISTS idx_decision_agent ON decision_log(is_agent, ts) WHERE is_agent = true;
	CREATE INDEX IF NOT EXISTS idx_decision_command ON decision_log(command, ts);
	CREATE INDEX IF NOT EXISTS idx_decision_outcome ON decision_log(outcome_category, ts);
	CREATE INDEX IF NOT EXISTS idx_decision_harness ON decision_log(agent_harness, agent_model, ts) WHERE is_agent = true;

	CREATE INDEX IF NOT EXISTS idx_combat_decision ON combat_log(decision_id);
	CREATE INDEX IF NOT EXISTS idx_combat_session ON combat_log(session_id, ts);
	`

	_, err := db.conn.Exec(query)
	return err
}

// EnsureDecisionLogPartitions creates monthly partitions for decision_log and
// combat_log for the current and next month. Safe to call on every boot.
func (db *DB) EnsureDecisionLogPartitions() error {
	now := time.Now()
	var errs []error
	for i := 0; i < 2; i++ {
		month := now.AddDate(0, i, 0)
		start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)

		// Decision log partition
		dlName := fmt.Sprintf("decision_log_%s", start.Format("2006_01"))
		if err := db.ensurePartition("decision_log", dlName, start, end); err != nil {
			slog.Warn("failed to create decision_log partition", "table", dlName, "error", err)
			errs = append(errs, fmt.Errorf("decision_log partition %s: %w", dlName, err))
		}

		// Combat log partition
		clName := fmt.Sprintf("combat_log_%s", start.Format("2006_01"))
		if err := db.ensurePartition("combat_log", clName, start, end); err != nil {
			slog.Warn("failed to create combat_log partition", "table", clName, "error", err)
			errs = append(errs, fmt.Errorf("combat_log partition %s: %w", clName, err))
		}
	}
	return errors.Join(errs...)
}

func (db *DB) ensurePartition(parent, name string, start, end time.Time) error {
	var exists bool
	err := db.conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)`, name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check partition exists: %w", err)
	}
	if exists {
		return nil
	}

	_, err = db.conn.Exec(fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')`,
		name, parent, start.Format("2006-01-02"), end.Format("2006-01-02"),
	))
	return err
}

// ---------------------------------------------------------------------------
// Decision record types
// ---------------------------------------------------------------------------

// DecisionRecord represents a single command execution for the decision log.
type DecisionRecord struct {
	SessionID      string
	PlayerName     string
	IsAgent        bool
	AgentHarness   string
	AgentModel     string
	TurnNumber     int
	SessionElapsed float64

	Command      string
	CommandClass string
	Args         []string
	RawInput     string

	DecisionSource   string
	PromptTokens     int
	CompletionTokens int

	PreRoom      int
	PreZone      int
	PreHealth    int
	PreMaxHealth int
	PreMana      int
	PreMaxMana   int
	PreMove      int
	PreMaxMove   int
	PreLevel     int
	PreFighting  bool
	PrePosition  int
	PreInvCount  int

	PostRoom      int
	PostZone      int
	PostHealth    int
	PostMaxHealth int
	PostMana      int
	PostMaxMana   int
	PostMove      int
	PostMaxMove   int
	PostLevel     int
	PostFighting  bool
	PostPosition  int
	PostInvCount  int

	OutcomeCategory string
	OutcomeValue    int
	OutcomeText     string
	DurationMs      float64
}

// CombatRecord represents a single combat round for the combat log.
type CombatRecord struct {
	DecisionID   int64
	SessionID    string
	RoundNumber  int
	AttackerName string
	AttackerVnum int
	DefenderName string
	DefenderVnum int
	AttackType   string
	Damage       int
	Outcome      string

	TargetHealth    int
	TargetMaxHealth int
	TargetLevel     int
	TargetPosition  int
	TargetCount     int
}

// ---------------------------------------------------------------------------
// Write buffer
// ---------------------------------------------------------------------------

const (
	flushInterval  = 1 * time.Second
	flushBatchSize = 100
)

// DecisionLogWriter buffers decision and combat records and flushes them to
// PostgreSQL in batches. Safe for concurrent use.
type DecisionLogWriter struct {
	db          *DB
	mu          sync.Mutex
	flushMu     sync.Mutex
	decisions   []*DecisionRecord
	combat      []*CombatRecord
	flushTicker *time.Ticker
	stopCh      chan struct{}
	stopOnce    sync.Once
	flushWG     sync.WaitGroup
}

// NewDecisionLogWriter creates a writer and starts the background flush loop.
func (db *DB) NewDecisionLogWriter() *DecisionLogWriter {
	dlw := &DecisionLogWriter{
		db:        db,
		decisions: make([]*DecisionRecord, 0, flushBatchSize),
		combat:    make([]*CombatRecord, 0, flushBatchSize),
		stopCh:    make(chan struct{}),
	}
	dlw.flushWG.Add(1)
	go dlw.flushLoop()
	return dlw
}

// NewMockDecisionLogWriter returns a DecisionLogWriter with no database handle
// and no background flush goroutine. It is safe for the construct/record/Stop
// paths used in tests: RecordDecision appends to the in-memory buffer, and
// Stop closes stopCh and runs a final Flush that early-returns when the buffers
// are empty, so it never touches the nil db. It does NOT persist — driving the
// buffer to flushBatchSize would trigger a real Flush that dereferences the nil
// db, so it is intended only for low-volume test scenarios.
func NewMockDecisionLogWriter() *DecisionLogWriter {
	return &DecisionLogWriter{
		decisions: make([]*DecisionRecord, 0, flushBatchSize),
		combat:    make([]*CombatRecord, 0, flushBatchSize),
		stopCh:    make(chan struct{}),
	}
}

// Stop terminates the background flush loop and flushes remaining records.
// Safe to call multiple times; subsequent calls are no-ops.
//
// It signals the loop, waits for it to fully exit so any in-flight Flush the
// loop started runs to completion, then performs a final Flush of whatever the
// loop did not drain. This ordering prevents dropped records on shutdown (DP-767).
func (dlw *DecisionLogWriter) Stop() {
	dlw.stopOnce.Do(func() {
		close(dlw.stopCh)
		dlw.flushWG.Wait()
		dlw.Flush()
	})
}

// RecordDecision appends a decision record to the buffer.
func (dlw *DecisionLogWriter) RecordDecision(r *DecisionRecord) {
	dlw.mu.Lock()
	dlw.decisions = append(dlw.decisions, r)
	shouldFlush := len(dlw.decisions) >= flushBatchSize
	dlw.mu.Unlock()

	if shouldFlush {
		dlw.Flush()
	}
}

// RecordCombat appends a combat record to the buffer.
func (dlw *DecisionLogWriter) RecordCombat(r *CombatRecord) {
	dlw.mu.Lock()
	dlw.combat = append(dlw.combat, r)
	dlw.mu.Unlock()
}

// HashPlayerName returns a pseudonymized name for human players.
// Agent names are stored in plaintext (research subjects).
func (dlw *DecisionLogWriter) HashPlayerName(name string, isAgent bool) string {
	if isAgent {
		return name
	}
	h := sha256.Sum256([]byte("darkpawns_salt:" + name))
	return fmt.Sprintf("player_%s", h[:8])
}

// Flush writes all buffered records to the database. Failed batches are
// requeued at the front of their buffers so transient errors do not silently
// drop telemetry. Flushes are serialized with flushMu.
func (dlw *DecisionLogWriter) Flush() {
	dlw.flushMu.Lock()
	defer dlw.flushMu.Unlock()

	dlw.mu.Lock()
	if len(dlw.decisions) == 0 && len(dlw.combat) == 0 {
		dlw.mu.Unlock()
		return
	}

	decisions := dlw.decisions
	dlw.decisions = make([]*DecisionRecord, 0, flushBatchSize)
	combat := dlw.combat
	dlw.combat = make([]*CombatRecord, 0, flushBatchSize)
	dlw.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if len(decisions) > 0 {
		if err := dlw.flushDecisions(ctx, decisions); err != nil {
			slog.Error("decision_log flush failed", "count", len(decisions), "error", err)
			dlw.mu.Lock()
			dlw.decisions = append(decisions, dlw.decisions...)
			dlw.mu.Unlock()
		}
	}

	if len(combat) > 0 {
		if err := dlw.flushCombat(ctx, combat); err != nil {
			slog.Error("combat_log flush failed", "count", len(combat), "error", err)
			dlw.mu.Lock()
			dlw.combat = append(combat, dlw.combat...)
			dlw.mu.Unlock()
		}
	}
}

func (dlw *DecisionLogWriter) flushLoop() {
	defer dlw.flushWG.Done()
	dlw.flushTicker = time.NewTicker(flushInterval)
	defer dlw.flushTicker.Stop()
	for {
		select {
		case <-dlw.flushTicker.C:
			dlw.Flush()
		case <-dlw.stopCh:
			return
		}
	}
}

func (dlw *DecisionLogWriter) flushDecisions(ctx context.Context, records []*DecisionRecord) error {
	if len(records) == 0 {
		return nil
	}

	query := `INSERT INTO decision_log (
		session_id, player_name, is_agent, agent_harness, agent_model,
		turn_number, session_elapsed,
		command, command_class, args, raw_input,
		decision_source, prompt_tokens, completion_tokens,
		pre_room, pre_zone, pre_health, pre_max_health,
		pre_mana, pre_max_mana, pre_move, pre_max_move,
		pre_level, pre_fighting, pre_position, pre_inv_count,
		post_room, post_zone, post_health, post_max_health,
		post_mana, post_max_mana, post_move, post_max_move,
		post_level, post_fighting, post_position, post_inv_count,
		outcome_category, outcome_value, outcome_text, duration_ms
	) VALUES `

	var sb strings.Builder
	sb.WriteString(query)
	args := make([]interface{}, 0, len(records)*42)
	argIdx := 1

	for i, r := range records {
		if i > 0 {
			sb.WriteString(",\n")
		}
		args = append(
			args,
			r.SessionID, r.PlayerName, r.IsAgent, r.AgentHarness, r.AgentModel,
			r.TurnNumber, r.SessionElapsed,
			r.Command, r.CommandClass, r.Args, r.RawInput,
			r.DecisionSource, r.PromptTokens, r.CompletionTokens,
			r.PreRoom, r.PreZone, r.PreHealth, r.PreMaxHealth,
			r.PreMana, r.PreMaxMana, r.PreMove, r.PreMaxMove,
			r.PreLevel, r.PreFighting, r.PrePosition, r.PreInvCount,
			r.PostRoom, r.PostZone, r.PostHealth, r.PostMaxHealth,
			r.PostMana, r.PostMaxMana, r.PostMove, r.PostMaxMove,
			r.PostLevel, r.PostFighting, r.PostPosition, r.PostInvCount,
			r.OutcomeCategory, r.OutcomeValue, r.OutcomeText, r.DurationMs,
		)

		fmt.Fprintf(
			&sb,
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4,
			argIdx+5, argIdx+6,
			argIdx+7, argIdx+8, argIdx+9, argIdx+10,
			argIdx+11, argIdx+12, argIdx+13,
			argIdx+14, argIdx+15, argIdx+16, argIdx+17,
			argIdx+18, argIdx+19, argIdx+20, argIdx+21,
			argIdx+22, argIdx+23, argIdx+24, argIdx+25,
			argIdx+26, argIdx+27, argIdx+28, argIdx+29,
			argIdx+30, argIdx+31, argIdx+32, argIdx+33,
			argIdx+34, argIdx+35, argIdx+36, argIdx+37,
			argIdx+38, argIdx+39, argIdx+40, argIdx+41,
		)
		argIdx += 42
	}

	_, err := dlw.db.conn.ExecContext(ctx, sb.String(), args...)
	return err
}

func (dlw *DecisionLogWriter) flushCombat(ctx context.Context, records []*CombatRecord) error {
	if len(records) == 0 {
		return nil
	}

	query := `INSERT INTO combat_log (
		decision_id, session_id, round_number,
		attacker_name, attacker_vnum, defender_name, defender_vnum,
		attack_type, damage, outcome,
		target_health, target_max_health, target_level, target_position, target_count
	) VALUES `

	var sb strings.Builder
	sb.WriteString(query)
	args := make([]interface{}, 0, len(records)*15)
	argIdx := 1

	for i, r := range records {
		if i > 0 {
			sb.WriteString(",\n")
		}
		args = append(
			args,
			r.DecisionID, r.SessionID, r.RoundNumber,
			r.AttackerName, r.AttackerVnum, r.DefenderName, r.DefenderVnum,
			r.AttackType, r.Damage, r.Outcome,
			r.TargetHealth, r.TargetMaxHealth, r.TargetLevel, r.TargetPosition, r.TargetCount,
		)

		fmt.Fprintf(
			&sb,
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4,
			argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9,
			argIdx+10, argIdx+11, argIdx+12, argIdx+13, argIdx+14,
		)
		argIdx += 15
	}

	_, err := dlw.db.conn.ExecContext(ctx, sb.String(), args...)
	return err
}
