package moderation

import (
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

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
			reason TEXT NOT NULL,
			issued_by VARCHAR(32) NOT NULL,
			PRIMARY KEY (player_name, penalty_type, issued_at)
		)`,

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
		WHERE expires_at IS NULL OR expires_at > NOW()
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

		m.wordFilters = append(m.wordFilters, wf)
	}
}

// cleanupRoutine periodically cleans up expired penalties and old message history.
func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredPenalties()
		m.cleanupOldMessageHistory()
	}
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

	// Clean database if available
	if m.hasDB {
		_, err := m.db.Exec(`
			DELETE FROM player_penalties
			WHERE expires_at IS NOT NULL AND expires_at <= NOW()
		`)
		if err != nil {
			slog.Error("Failed to clean expired penalties", "error", err)
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
	if m.hasPenalty(playerName, ActionMute) {
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
func (m *Manager) hasPenalty(playerName string, penaltyType AdminAction) bool {
	penalties, exists := m.activePenalties[playerName]
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
		re, err := regexp.Compile(wf.Pattern)
		if err != nil {
			slog.Error("Invalid regex pattern", "pattern", wf.Pattern, "error", err)
			return false
		}
		return re.MatchString(strings.ToLower(message))
	}

	return strings.Contains(strings.ToLower(message), strings.ToLower(wf.Pattern))
}

// censor replaces matched patterns with asterisks.
func (wf *WordFilterEntry) censor(message string) string {
	if wf.IsRegex {
		re, err := regexp.Compile(wf.Pattern)
		if err != nil {
			return message
		}
		return re.ReplaceAllStringFunc(message, func(match string) string {
			return strings.Repeat("*", len(match))
		})
	}

	return strings.ReplaceAll(message, wf.Pattern, strings.Repeat("*", len(wf.Pattern)))
}

// AddPenalty stores a penalty (in-memory + DB if available).
func (m *Manager) AddPenalty(p PlayerPenalty) {
	m.mu.Lock()
	defer m.mu.Unlock()

	playerName := strings.ToLower(p.PlayerName)
	m.activePenalties[playerName] = append(m.activePenalties[playerName], p)

	if m.hasDB {
		var expiresAt *time.Time
		if p.ExpiresAt != nil {
			expiresAt = p.ExpiresAt
		}
		_, err := m.db.Exec(`
			INSERT INTO player_penalties (player_name, penalty_type, issued_at, expires_at, reason, issued_by)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			playerName, p.PenaltyType, p.IssuedAt, expiresAt, p.Reason, p.IssuedBy,
		)
		if err != nil {
			slog.Error("failed to persist penalty", "error", err)
		}
	}
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

// AddWordFilter adds a new word filter entry.
func (m *Manager) AddWordFilter(pattern string, isRegex bool, actionStr, createdBy string) {
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

	m.mu.Lock()
	nextID := len(m.wordFilters) + 1
	if len(m.wordFilters) > 0 {
		nextID = m.wordFilters[len(m.wordFilters)-1].ID + 1
	}
	m.wordFilters = append(m.wordFilters, WordFilterEntry{
		ID:        nextID,
		Pattern:   pattern,
		IsRegex:   isRegex,
		Action:    action,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	})
	m.mu.Unlock()

	if m.hasDB {
		_, err := m.db.Exec(`
			INSERT INTO word_filters (pattern, is_regex, action, created_by)
			VALUES ($1, $2, $3, $4)`,
			pattern, isRegex, action, createdBy,
		)
		if err != nil {
			slog.Error("failed to persist word filter", "error", err)
		}
	}
}

// RemoveWordFilter removes a word filter by ID.
func (m *Manager) RemoveWordFilter(filterID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, f := range m.wordFilters {
		if f.ID == filterID {
			m.wordFilters = append(m.wordFilters[:i], m.wordFilters[i+1:]...)
			break
		}
	}

	if m.hasDB {
		_, err := m.db.Exec(`DELETE FROM word_filters WHERE id = $1`, filterID)
		if err != nil {
			slog.Error("failed to delete word filter", "error", err)
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
