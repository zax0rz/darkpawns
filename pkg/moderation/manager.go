package moderation

import (
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	_ "github.com/lib/pq"
)

// Manager handles all moderation operations.
type Manager struct {
	mu    sync.RWMutex
	db    *sql.DB
	hasDB bool

	// In-memory caches for performance
	activePenalties map[string][]PlayerPenalty // player -> penalties
	wordFilters     []WordFilterEntry
	spamConfig      SpamDetectionConfig

	// Spam tracking
	messageHistory map[string][]time.Time // player -> timestamps of recent messages

	// Lifecycle: stop closes to halt the cleanup goroutine; done is closed by
	// the goroutine when it exits; closeOnce guards against double Close.
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// NewManager creates a new moderation manager.
func NewManager(db *sql.DB) *Manager {
	m := &Manager{
		db:              db,
		hasDB:           db != nil,
		activePenalties: make(map[string][]PlayerPenalty),
		wordFilters:     make([]WordFilterEntry, 0),
		messageHistory:  make(map[string][]time.Time),
		spamConfig: SpamDetectionConfig{
			MessagesPerMinute: 10,
			DuplicateWindow:   5 * time.Second,
			Action:            FilterActionWarn,
		},
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	if m.hasDB {
		if err := m.createTables(); err != nil {
			slog.Warn("Failed to create moderation tables", "error", err)
		}
		m.loadActivePenalties()
		m.loadWordFilters()
	}

	// Start cleanup goroutine
	go m.cleanupRoutine()

	return m
}

// createTables creates the necessary moderation tables.
func (m *Manager) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS abuse_reports (
			id SERIAL PRIMARY KEY,
			reporter VARCHAR(32) NOT NULL,
			target VARCHAR(32) NOT NULL,
			report_type VARCHAR(32) NOT NULL,
			description TEXT NOT NULL,
			room_vnum INTEGER DEFAULT 0,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(32) DEFAULT 'pending',
			reviewed_by VARCHAR(32),
			reviewed_at TIMESTAMP,
			resolution TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS admin_log (
			id SERIAL PRIMARY KEY,
			admin VARCHAR(32) NOT NULL,
			action VARCHAR(32) NOT NULL,
			target VARCHAR(32) NOT NULL,
			reason TEXT NOT NULL,
			duration INTERVAL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ip_address VARCHAR(45)
		)`,

		`CREATE TABLE IF NOT EXISTS player_penalties (
			player_name VARCHAR(32) NOT NULL,
			penalty_type VARCHAR(32) NOT NULL,
			issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			expired_at TIMESTAMP,
			status VARCHAR(32) DEFAULT 'active',
			reason TEXT NOT NULL,
			issued_by VARCHAR(32) NOT NULL,
			PRIMARY KEY (player_name, penalty_type, issued_at)
		)`,
		`ALTER TABLE player_penalties ADD COLUMN IF NOT EXISTS expired_at TIMESTAMP`,
		`ALTER TABLE player_penalties ADD COLUMN IF NOT EXISTS status VARCHAR(32) DEFAULT 'active'`,

		`CREATE TABLE IF NOT EXISTS word_filters (
			id SERIAL PRIMARY KEY,
			pattern VARCHAR(255) NOT NULL,
			is_regex BOOLEAN DEFAULT false,
			action VARCHAR(32) NOT NULL,
			created_by VARCHAR(32) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}

// loadActivePenalties loads active penalties from database.
func (m *Manager) loadActivePenalties() {
	if !m.hasDB {
		return
	}

	rows, err := m.db.Query(`
		SELECT player_name, penalty_type, issued_at, expires_at, reason, issued_by
		FROM player_penalties
		WHERE status = 'active' AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		slog.Error("Failed to load penalties", "error", err)
		return
	}
	defer func() { _ = rows.Close() }()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.activePenalties = make(map[string][]PlayerPenalty)

	for rows.Next() {
		var p PlayerPenalty
		var expiresAt sql.NullTime

		if err := rows.Scan(&p.PlayerName, &p.PenaltyType, &p.IssuedAt, &expiresAt, &p.Reason, &p.IssuedBy); err != nil {
			slog.Error("Failed to scan penalty", "error", err)
			continue
		}

		if expiresAt.Valid {
			p.ExpiresAt = &expiresAt.Time
		}

		m.activePenalties[p.PlayerName] = append(m.activePenalties[p.PlayerName], p)
	}
	if err := rows.Err(); err != nil {
		slog.Error("Failed to iterate penalties", "error", err)
	}
}

// loadWordFilters loads word filters from database.
func (m *Manager) loadWordFilters() {
	if !m.hasDB {
		return
	}

	rows, err := m.db.Query(`
		SELECT id, pattern, is_regex, action, created_by, created_at
		FROM word_filters
		ORDER BY created_at DESC
	`)
	if err != nil {
		slog.Error("Failed to load word filters", "error", err)
		return
	}
	defer func() { _ = rows.Close() }()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.wordFilters = make([]WordFilterEntry, 0)

	for rows.Next() {
		var wf WordFilterEntry
		if err := rows.Scan(&wf.ID, &wf.Pattern, &wf.IsRegex, &wf.Action, &wf.CreatedBy, &wf.CreatedAt); err != nil {
			slog.Error("Failed to scan word filter", "error", err)
			continue
		}
		wf.compile()

		m.wordFilters = append(m.wordFilters, wf)
	}
	if err := rows.Err(); err != nil {
		slog.Error("Failed to iterate word filters", "error", err)
	}
}

// cleanupRoutine periodically cleans up expired penalties and old message history.
func (m *Manager) cleanupRoutine() {
	defer close(m.done)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.cleanupExpiredPenalties()
			m.cleanupOldMessageHistory()
		}
	}
}

// Close stops the background cleanup goroutine and waits for it to exit. It is
// safe to call Close multiple times.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		close(m.stop)
		<-m.done
	})
}

// cleanupExpiredPenalties removes expired penalties from memory and database.
func (m *Manager) cleanupExpiredPenalties() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Clean in-memory cache
	for player, penalties := range m.activePenalties {
		var active []PlayerPenalty
		for _, p := range penalties {
			if p.ExpiresAt == nil || p.ExpiresAt.After(now) {
				active = append(active, p)
			}
		}
		m.activePenalties[player] = active
	}

	// Mark database penalties as expired instead of deleting them, preserving
	// an audit trail for admin investigation.
	if m.hasDB {
		_, err := m.db.Exec(`
			UPDATE player_penalties
			SET status = 'expired', expired_at = NOW()
			WHERE expires_at IS NOT NULL AND expires_at <= NOW()
			  AND status != 'expired'
		`)
		if err != nil {
			slog.Error("Failed to mark expired penalties", "error", err)
		}
	}
}

// cleanupOldMessageHistory removes old message timestamps.
func (m *Manager) cleanupOldMessageHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)

	for player, timestamps := range m.messageHistory {
		var recent []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				recent = append(recent, ts)
			}
		}
		m.messageHistory[player] = recent

		// Remove empty entries
		if len(recent) == 0 {
			delete(m.messageHistory, player)
		}
	}
}

// CheckMessage checks a message for filtered words and spam.
// Returns (filtered message, action taken, should block).
func (m *Manager) CheckMessage(playerName, message string) (string, FilterAction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for active mute penalty
	if m.hasPenaltyLocked(playerName, ActionMute) {
		return "", FilterActionBlock, true
	}

	// Check word filters
	filteredMsg := message
	actionTaken := FilterActionLog // Default to just logging

	for _, wf := range m.wordFilters {
		if wf.matches(message) {
			actionTaken = wf.Action

			switch wf.Action {
			case FilterActionCensor:
				filteredMsg = wf.censor(message)
			case FilterActionBlock:
				return "", FilterActionBlock, true
			case FilterActionWarn:
				// Warning will be handled by caller
				filteredMsg = message
			case FilterActionLog:
				// Just log, no modification
				filteredMsg = message
			}

			// For now, apply first matching filter
			break
		}
	}

	// Check for spam
	if m.isSpam(playerName) {
		return filteredMsg, m.spamConfig.Action, m.spamConfig.Action == FilterActionBlock
	}

	return filteredMsg, actionTaken, false
}

// RecordMessage records a message for spam detection.
func (m *Manager) RecordMessage(playerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.messageHistory[playerName] = append(m.messageHistory[playerName], now)
}

// hasPenalty checks if a player has an active penalty of a given type.
// It acquires the read lock and is safe for public callers.
func (m *Manager) hasPenalty(playerName string, penaltyType AdminAction) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hasPenaltyLocked(playerName, penaltyType)
}

// hasPenaltyLocked checks if a player has an active penalty of a given type.
// The caller must hold at least a read lock on m.mu.
func (m *Manager) hasPenaltyLocked(playerName string, penaltyType AdminAction) bool {
	// AddPenalty stores penalties under a lowercase key, so lookup must be
	// normalized to match regardless of the casing the caller supplies.
	penalties, exists := m.activePenalties[strings.ToLower(playerName)]
	if !exists {
		return false
	}

	now := time.Now()
	for _, p := range penalties {
		if p.PenaltyType == penaltyType && (p.ExpiresAt == nil || p.ExpiresAt.After(now)) {
			return true
		}
	}

	return false
}

// isSpam checks if a player is spamming based on message history.
func (m *Manager) isSpam(playerName string) bool {
	timestamps, exists := m.messageHistory[playerName]
	if !exists {
		return false
	}

	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	count := 0
	for _, ts := range timestamps {
		if ts.After(oneMinuteAgo) {
			count++
		}
	}

	return count > m.spamConfig.MessagesPerMinute
}

// matches checks if a word filter matches the message.
func (wf *WordFilterEntry) matches(message string) bool {
	if wf.IsRegex {
		// Lazily compile if the cache was not populated at add/load time
		// (defensive — e.g. a WordFilterEntry built by a test). The hot path
		// goes through entries added via AddWordFilter/loadWordFilters, which
		// pre-compile under the manager lock (DP-819).
		if wf.compiled == nil {
			re, err := regexp.Compile(wf.Pattern)
			if err != nil {
				slog.Error("Invalid regex pattern", "pattern", wf.Pattern, "error", err)
				return false
			}
			wf.compiled = re
		}
		return wf.compiled.MatchString(message)
	}

	return strings.Contains(strings.ToLower(message), strings.ToLower(wf.Pattern))
}

// censor replaces matched patterns with asterisks.
// Matching is case-insensitive (the message is lowercased for exact filters and
// regex filters are applied to the original text), so censoring must also be
// case-insensitive to avoid leaving mixed/upper-case variants uncensored.
func (wf *WordFilterEntry) censor(message string) string {
	// Lazily compile if the cache was not populated at add/load time (DP-819).
	if wf.censored == nil {
		var re *regexp.Regexp
		var err error
		if wf.IsRegex {
			re, err = regexp.Compile(`(?i)` + wf.Pattern)
		} else {
			re, err = regexp.Compile(`(?i)` + regexp.QuoteMeta(wf.Pattern))
		}
		if err != nil {
			return message
		}
		wf.censored = re
	}
	return wf.censored.ReplaceAllStringFunc(message, func(match string) string {
		return strings.Repeat("*", utf8.RuneCountInString(match))
	})
}

// AddReport stores an abuse report (DB if available, always logs). Returns a
// non-nil error if the DB write failed, so callers can warn the admin that the
// report is memory-only and will not survive a restart.
func (m *Manager) AddReport(r AbuseReport) error {
	if m.hasDB {
		_, err := m.db.Exec(
			`
			INSERT INTO abuse_reports (reporter, target, report_type, description, room_vnum, timestamp, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			r.Reporter, r.Target, string(r.ReportType), r.Description, r.RoomVNum, r.Timestamp, string(r.Status),
		)
		if err != nil {
			slog.Error("failed to persist abuse report", "error", err)
			return fmt.Errorf("persist abuse report: %w", err)
		}
	}
	return nil
}

// MaxReportID returns the highest existing abuse report ID, or 0 if there are
// none (or no DB is configured). Used to seed the in-memory report ID counter
// on startup so IDs stay unique across restarts.
func (m *Manager) MaxReportID() (int, error) {
	if !m.hasDB {
		return 0, nil
	}

	var maxID int
	if err := m.db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM abuse_reports`).Scan(&maxID); err != nil {
		return 0, fmt.Errorf("query max report id: %w", err)
	}
	return maxID, nil
}

// ListReports returns all abuse reports from the database, most recent first.
// Returns nil if no DB is configured.
func (m *Manager) ListReports() ([]AbuseReport, error) {
	if !m.hasDB {
		return nil, nil
	}

	rows, err := m.db.Query(`
		SELECT id, reporter, target, report_type, description, room_vnum, timestamp, status, reviewed_by, reviewed_at, resolution
		FROM abuse_reports
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query reports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []AbuseReport
	for rows.Next() {
		var r AbuseReport
		var reportType, status string
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		var resolution sql.NullString

		if err := rows.Scan(
			&r.ID, &r.Reporter, &r.Target, &reportType, &r.Description, &r.RoomVNum,
			&r.Timestamp, &status, &reviewedBy, &reviewedAt, &resolution,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}

		r.ReportType = ReportType(reportType)
		r.Status = ReportStatus(status)
		if reviewedBy.Valid {
			r.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			r.ReviewedAt = &reviewedAt.Time
		}
		if resolution.Valid {
			r.Resolution = resolution.String
		}

		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reports: %w", err)
	}

	return result, nil
}

// AddPenalty stores a penalty (in-memory + DB if available). Returns a non-nil
// error if the DB write failed; the penalty is still applied in memory, but
// callers should warn the admin that it will not survive a restart.
func (m *Manager) AddPenalty(p PlayerPenalty) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	playerName := strings.ToLower(p.PlayerName)
	m.activePenalties[playerName] = append(m.activePenalties[playerName], p)

	if m.hasDB {
		var expiresAt *time.Time
		if p.ExpiresAt != nil {
			expiresAt = p.ExpiresAt
		}
		_, err := m.db.Exec(
			`
			INSERT INTO player_penalties (player_name, penalty_type, issued_at, expires_at, reason, issued_by)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			playerName, p.PenaltyType, p.IssuedAt, expiresAt, p.Reason, p.IssuedBy,
		)
		if err != nil {
			slog.Error("failed to persist penalty", "error", err)
			return fmt.Errorf("persist penalty: %w", err)
		}
	}
	return nil
}

// GetPlayerPenalties returns all penalties for a player.
func (m *Manager) GetPlayerPenalties(playerName string) []PlayerPenalty {
	m.mu.RLock()
	defer m.mu.RUnlock()

	playerName = strings.ToLower(playerName)
	penalties, ok := m.activePenalties[playerName]
	if !ok {
		return nil
	}

	now := time.Now()
	var active []PlayerPenalty
	for _, p := range penalties {
		if p.ExpiresAt == nil || p.ExpiresAt.After(now) {
			active = append(active, p)
		}
	}
	return active
}

// GetAllActivePenalties returns all current active penalties across all players.
func (m *Manager) GetAllActivePenalties() []PlayerPenalty {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	var result []PlayerPenalty
	for _, penalties := range m.activePenalties {
		for _, p := range penalties {
			if p.ExpiresAt == nil || p.ExpiresAt.After(now) {
				result = append(result, p)
			}
		}
	}
	return result
}

// GetWordFilters returns all word filter entries.
func (m *Manager) GetWordFilters() []WordFilterEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]WordFilterEntry, len(m.wordFilters))
	copy(result, m.wordFilters)
	return result
}

// AddWordFilter adds a new word filter entry. Returns a non-nil error if the DB
// write failed; the filter is still active in memory, but callers should warn
// the admin that it will not survive a restart.
func (m *Manager) AddWordFilter(pattern string, isRegex bool, actionStr, createdBy string) error {
	action := FilterActionCensor
	switch strings.ToLower(actionStr) {
	case "censor":
		action = FilterActionCensor
	case "warn":
		action = FilterActionWarn
	case "block":
		action = FilterActionBlock
	case "log":
		action = FilterActionLog
	}

	entry := WordFilterEntry{
		Pattern:   pattern,
		IsRegex:   isRegex,
		Action:    action,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}
	entry.compile()

	// Compute a local ID. In DB mode this is temporary until RETURNING gives the
	// real serial id; in memory-only mode it is the permanent id.
	m.mu.Lock()
	entry.ID = 1
	for _, f := range m.wordFilters {
		if f.ID >= entry.ID {
			entry.ID = f.ID + 1
		}
	}
	m.wordFilters = append(m.wordFilters, entry)
	m.mu.Unlock()

	if m.hasDB {
		// Use RETURNING so the in-memory ID matches the DB row and removals hit
		// the correct row regardless of how loadWordFilters ordered the slice.
		row := m.db.QueryRow(
			`INSERT INTO word_filters (pattern, is_regex, action, created_by)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			pattern, isRegex, action, createdBy,
		)
		if err := row.Scan(&entry.ID); err != nil {
			slog.Error("failed to persist word filter", "error", err)
			return fmt.Errorf("persist word filter: %w", err)
		}
		// Update the in-memory entry with the real DB id.
		m.mu.Lock()
		for i := range m.wordFilters {
			if m.wordFilters[i].Pattern == pattern && m.wordFilters[i].CreatedAt.Equal(entry.CreatedAt) {
				m.wordFilters[i].ID = entry.ID
				break
			}
		}
		m.mu.Unlock()
	}
	return nil
}

// RemoveWordFilter removes a word filter by ID. The DB row is deleted first so
// the in-memory slice stays in sync with the database: if the delete fails, the
// filter remains active in memory (it would otherwise reappear on the next
// restart, since loadWordFilters re-loads the still-present row).
func (m *Manager) RemoveWordFilter(filterID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hasDB {
		_, err := m.db.Exec(`DELETE FROM word_filters WHERE id = $1`, filterID)
		if err != nil {
			slog.Error("failed to delete word filter", "error", err)
			return
		}
	}

	for i, f := range m.wordFilters {
		if f.ID == filterID {
			m.wordFilters = append(m.wordFilters[:i], m.wordFilters[i+1:]...)
			break
		}
	}
}

// GetSpamConfig returns the current spam detection config.
func (m *Manager) GetSpamConfig() SpamDetectionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.spamConfig
}

// SetSpamConfig updates the spam detection config.
func (m *Manager) SetSpamConfig(messagesPerMin int, actionStr string) {
	action := FilterActionWarn
	switch strings.ToLower(actionStr) {
	case "log":
		action = FilterActionLog
	case "warn":
		action = FilterActionWarn
	case "block":
		action = FilterActionBlock
	}

	m.mu.Lock()
	m.spamConfig.MessagesPerMinute = messagesPerMin
	m.spamConfig.Action = action
	m.mu.Unlock()
}

// IsMuted checks if a player is currently muted.
func (m *Manager) IsMuted(playerName string) bool {
	return m.hasPenalty(playerName, ActionMute)
}

// IsBanned checks if a player is currently banned.
func (m *Manager) IsBanned(playerName string) bool {
	return m.hasPenalty(playerName, ActionBan)
}
