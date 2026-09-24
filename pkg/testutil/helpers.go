package testutil

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// AssertBehaviorMatchesC compares the output of a Go routine against expected C source behavior.
// If they do not match, it triggers a descriptive test failure.
func AssertBehaviorMatchesC(t *testing.T, description string, goFunc func() string, cExpected string) {
	t.Helper()
	goActual := goFunc()
	if goActual != cExpected {
		t.Errorf("FIDELITY MISMATCH [%s]:\n  Go actual:   %q\n  C expected:  %q", description, goActual, cExpected)
	}
}

// MockDiceRoller provides a way to mock dice-rolling outcomes deterministically.
type MockDiceRoller struct {
	mu     sync.Mutex
	preset []int
	index  int
}

// NewMockDiceRoller creates a MockDiceRoller with optional preset roll sequences.
func NewMockDiceRoller(preset []int) *MockDiceRoller {
	return &MockDiceRoller{preset: preset}
}

// Roll returns the next deterministic roll, or falls back to a neutral 10 if depleted.
func (md *MockDiceRoller) Roll(dice, sides int) int {
	md.mu.Lock()
	defer md.mu.Unlock()
	if md.index < len(md.preset) {
		val := md.preset[md.index]
		md.index++
		return val
	}
	return dice * (sides / 2) // reasonable average fallback
}

// NewTestPlayer constructs a fully populated character player for test runs.
func NewTestPlayer(name string, class, race int) *game.Player {
	p := game.NewCharacter(0, name, class, race)
	p.Health = 100
	p.MaxHealth = 100
	p.Mana = 20
	p.MaxMana = 20
	p.Move = 100
	p.MaxMove = 100
	p.Level = 1
	return p
}

// NewTestWorld builds a minimal in-memory world containing essential starting rooms.
func NewTestWorld() *game.World {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 8004, Name: "A Burning Hut", Zone: 1},
			{VNum: 18201, Name: "Kir-Oshi Docks", Zone: 2},
			{VNum: 21258, Name: "Alaozar Temple", Zone: 3},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{
			{VNum: 8023, Keywords: "club", ShortDesc: "a club", LongDesc: "A wooden club."},
			{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic", LongDesc: "A plain tunic."},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		panic(err)
	}
	return w
}

// MockDatabase is a thread-safe, memory-backed struct fully satisfying the
// db.GameStore interface.
type MockDatabase struct {
	mu           sync.RWMutex
	players      map[string]*db.PlayerRecord
	nextPlayerID int
}

// Compile-time proof that the mock covers the store interface, so a method
// added to it fails here rather than in a downstream package.
var _ db.GameStore = (*MockDatabase)(nil)

// NewMockDatabase creates an initialized MockDatabase instance.
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		players:      make(map[string]*db.PlayerRecord),
		nextPlayerID: 1,
	}
}

// Close satisfies db.GameStore.
func (m *MockDatabase) Close() error {
	return nil
}

// ListPlayerNames satisfies db.GameStore.
func (m *MockDatabase) ListPlayerNames() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.players))
	for name := range m.players {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// CountPlayers satisfies db.GameStore. Used by the first-player-God bootstrap.
func (m *MockDatabase) CountPlayers() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.players), nil
}

// GetPlayer satisfies db.GameStore.
func (m *MockDatabase) GetPlayer(name string) (*db.PlayerRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var p *db.PlayerRecord
	for key, record := range m.players {
		if strings.EqualFold(key, name) {
			if p != nil {
				return nil, db.ErrAmbiguousPlayerName
			}
			p = record
		}
	}
	if p == nil {
		return nil, nil
	}
	copyP := *p
	return &copyP, nil
}

// CreatePlayer satisfies db.GameStore.
func (m *MockDatabase) CreatePlayer(p *db.PlayerRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.players {
		if strings.EqualFold(name, p.Name) {
			return fmt.Errorf("duplicate key value violates unique constraint")
		}
	}
	p.ID = m.nextPlayerID
	m.nextPlayerID++
	m.players[p.Name] = p
	return nil
}

// SavePlayer satisfies db.GameStore.
func (m *MockDatabase) SavePlayer(p *db.PlayerRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players[p.Name] = p
	return nil
}

// GetAccountLockout satisfies db.GameStore.
func (m *MockDatabase) GetAccountLockout(name string) (int, *time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.players[name]
	if !ok {
		return 0, nil, nil
	}
	return p.FailedLoginAttempts, p.LockedUntil, nil
}

// UpdatePassword satisfies db.GameStore.
func (m *MockDatabase) UpdatePassword(playerID int, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.players {
		if p.ID == playerID {
			p.Password = hash
			return nil
		}
	}
	return fmt.Errorf("player not found")
}

// UpdateDescription satisfies db.GameStore.
func (m *MockDatabase) UpdateDescription(playerID int, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.players {
		if p.ID == playerID {
			p.Description = description
			return nil
		}
	}
	return fmt.Errorf("player not found")
}

// DeletePlayer satisfies db.GameStore.
func (m *MockDatabase) DeletePlayer(playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, p := range m.players {
		if p.ID == playerID {
			delete(m.players, name)
			return nil
		}
	}
	return fmt.Errorf("player not found")
}

// RecordLoginFailure satisfies db.GameStore.
func (m *MockDatabase) RecordLoginFailure(name string, threshold int, lockoutDuration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.players[name]
	if !ok {
		return false, nil
	}
	p.FailedLoginAttempts++
	if p.FailedLoginAttempts >= threshold {
		until := time.Now().Add(lockoutDuration)
		p.LockedUntil = &until
		return true, nil
	}
	return false, nil
}

// RecordLoginSuccess satisfies db.GameStore.
func (m *MockDatabase) RecordLoginSuccess(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.players[name]; ok {
		p.FailedLoginAttempts = 0
		p.LockedUntil = nil
	}
	return nil
}

// Exec satisfies db.GameStore.
func (m *MockDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

// WriteSessionSummary satisfies db.GameStore. Summaries are keyed by
// (agentName, sessionID) so distinct sessions never collapse into one list.
