package main

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// mailIdentity resolves mail's persistent integer IDs through the same
// database records used by returning-player login. The online World index is
// intentionally not involved: mail must address players who are offline and
// remain addressable after a server restart.
type mailIdentity struct {
	database db.GameStore

	mu   sync.RWMutex
	byID map[int]string
}

func newMailIdentity(database db.GameStore) (*mailIdentity, error) {
	if database == nil {
		return nil, fmt.Errorf("persistent database is required")
	}

	identity := &mailIdentity{
		database: database,
		byID:     make(map[int]string),
	}
	if err := identity.refresh(); err != nil {
		return nil, err
	}
	return identity, nil
}

// refresh builds the reverse lookup from the authoritative player names and
// records. It is run at boot and on an ID miss so players created after boot
// are still resolved without inventing or guessing an ID.
func (m *mailIdentity) refresh() error {
	names, err := m.database.ListPlayerNames()
	if err != nil {
		return fmt.Errorf("list persistent player names: %w", err)
	}

	byID := make(map[int]string, len(names))
	for _, name := range names {
		record, err := m.database.GetPlayer(name)
		if err != nil {
			return fmt.Errorf("load persistent identity %q: %w", name, err)
		}
		if record == nil {
			return fmt.Errorf("persistent identity %q disappeared during bootstrap", name)
		}
		if record.ID <= 0 || record.Name == "" {
			return fmt.Errorf("persistent identity %q has invalid ID/name (%d, %q)", name, record.ID, record.Name)
		}
		if previous, exists := byID[record.ID]; exists && previous != record.Name {
			return fmt.Errorf("persistent identity ID %d maps to both %q and %q", record.ID, previous, record.Name)
		}
		byID[record.ID] = record.Name
	}

	m.mu.Lock()
	m.byID = byID
	m.mu.Unlock()
	return nil
}

func (m *mailIdentity) remember(record *db.PlayerRecord) error {
	if record == nil || record.ID <= 0 || record.Name == "" {
		return fmt.Errorf("invalid persistent identity")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, exists := m.byID[record.ID]; exists && previous != record.Name {
		return fmt.Errorf("persistent identity ID %d maps to both %q and %q", record.ID, previous, record.Name)
	}
	m.byID[record.ID] = record.Name
	return nil
}

// PlayerRenamed refreshes the cached reverse lookup after the character row's
// name changes. Mail headers resolve their sender and recipient names by ID,
// so the same stable ID must immediately display the new spelling.
func (m *mailIdentity) PlayerRenamed(oldName, newName string) {
	if m == nil || m.database == nil || oldName == "" || newName == "" || oldName == newName {
		return
	}
	record, err := m.database.GetPlayer(newName)
	if err != nil {
		slog.Error("mail identity cache could not follow character rename", "old_name", oldName, "new_name", newName, "error", err)
		return
	}
	if record == nil || record.ID <= 0 {
		slog.Error("mail identity cache could not follow character rename", "old_name", oldName, "new_name", newName, "error", "renamed player record not found")
		return
	}
	m.mu.Lock()
	m.byID[record.ID] = record.Name
	m.mu.Unlock()
}

func (m *mailIdentity) idByName(name string) int {
	record, err := m.database.GetPlayer(name)
	if err != nil {
		slog.Error("mail sender/recipient identity lookup failed", "name", name, "error", err)
		return -1
	}
	if record == nil {
		return -1
	}
	if err := m.remember(record); err != nil {
		slog.Error("mail identity rejected", "name", name, "error", err)
		return -1
	}
	return record.ID
}

func (m *mailIdentity) nameByID(id int) string {
	if id <= 0 {
		return ""
	}

	m.mu.RLock()
	name := m.byID[id]
	m.mu.RUnlock()
	if name != "" {
		return name
	}

	if err := m.refresh(); err != nil {
		slog.Error("mail reverse identity lookup failed", "id", id, "error", err)
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byID[id]
}
