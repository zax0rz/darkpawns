package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// MockDatabase is a thread-safe, memory-backed struct fully satisfying the db.Database interface.
type MockDatabase struct {
	mu           sync.RWMutex
	players      map[string]*db.PlayerRecord
	nextPlayerID int

	agentKeys      map[string]string // rawKey -> characterName
	agentKeysByID  map[int64]string  // keyID -> characterName
	nextAgentKeyID int64

	memories     []*db.NarrativeMemory
	nextMemoryID int64

	summaries map[string][]string // agentName -> summaries
}

// NewMockDatabase creates an initialized MockDatabase instance.
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		players:        make(map[string]*db.PlayerRecord),
		nextPlayerID:   1,
		agentKeys:      make(map[string]string),
		agentKeysByID:  make(map[int64]string),
		nextAgentKeyID: 1,
		memories:       make([]*db.NarrativeMemory, 0),
		nextMemoryID:   1,
		summaries:      make(map[string][]string),
	}
}

// Close satisfies db.Database.
func (m *MockDatabase) Close() error {
	return nil
}

// GetPlayer satisfies db.Database.
func (m *MockDatabase) GetPlayer(name string) (*db.PlayerRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.players[name]
	if !ok {
		return nil, nil
	}
	copyP := *p
	return &copyP, nil
}

// CreatePlayer satisfies db.Database.
func (m *MockDatabase) CreatePlayer(p *db.PlayerRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.players[p.Name]; ok {
		return fmt.Errorf("duplicate key value violates unique constraint")
	}
	p.ID = m.nextPlayerID
	m.nextPlayerID++
	m.players[p.Name] = p
	return nil
}

// SavePlayer satisfies db.Database.
func (m *MockDatabase) SavePlayer(p *db.PlayerRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players[p.Name] = p
	return nil
}

// UpdatePassword satisfies db.Database.
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

// Exec satisfies db.Database.
func (m *MockDatabase) Exec(query string, args ...interface{}) (interface{}, error) {
	return nil, nil
}

// CreateAgentKey satisfies db.Database.
func (m *MockDatabase) CreateAgentKey(characterName string) (rawKey string, id int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	rawKey = "dp_" + hex.EncodeToString(buf)

	id = m.nextAgentKeyID
	m.nextAgentKeyID++

	m.agentKeys[rawKey] = characterName
	m.agentKeysByID[id] = characterName
	return rawKey, id, nil
}

// ValidateAgentKey satisfies db.Database.
func (m *MockDatabase) ValidateAgentKey(rawKey string) (characterName string, keyID int64, valid bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	characterName, ok := m.agentKeys[rawKey]
	if !ok {
		return "", 0, false
	}
	for id, name := range m.agentKeysByID {
		if name == characterName {
			return characterName, id, true
		}
	}
	return characterName, 1, true
}

// EnsureDecisionLogPartitions satisfies db.Database.
func (m *MockDatabase) EnsureDecisionLogPartitions() error {
	return nil
}

// NewDecisionLogWriter satisfies db.Database.
func (m *MockDatabase) NewDecisionLogWriter() *db.DecisionLogWriter {
	return nil
}

// InitNarrativeMemory satisfies db.Database.
func (m *MockDatabase) InitNarrativeMemory() error {
	return nil
}

// WriteNarrativeMemory satisfies db.Database.
func (m *MockDatabase) WriteNarrativeMemory(mem *db.NarrativeMemory) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mem.ID = m.nextMemoryID
	m.nextMemoryID++
	m.memories = append(m.memories, mem)
	return mem.ID, nil
}

// BootstrapMemories satisfies db.Database.
func (m *MockDatabase) BootstrapMemories(agentName string, limit int) ([]*db.NarrativeMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*db.NarrativeMemory
	count := 0
	for i := len(m.memories) - 1; i >= 0; i-- {
		mem := m.memories[i]
		if mem.AgentName == agentName {
			out = append(out, mem)
			count++
			if count >= limit {
				break
			}
		}
	}
	return out, nil
}

// RecentMemories satisfies db.Database.
func (m *MockDatabase) RecentMemories(agentName, sessionID string) ([]*db.NarrativeMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*db.NarrativeMemory
	for _, mem := range m.memories {
		if mem.AgentName == agentName && mem.SessionID == sessionID {
			out = append(out, mem)
		}
	}
	return out, nil
}

// SocialEventMemories satisfies db.Database.
func (m *MockDatabase) SocialEventMemories(socialEventID string) ([]*db.NarrativeMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*db.NarrativeMemory
	for _, mem := range m.memories {
		if mem.SocialEventID == socialEventID {
			out = append(out, mem)
		}
	}
	return out, nil
}

// WriteSessionSummary satisfies db.Database.
func (m *MockDatabase) WriteSessionSummary(agentName, sessionID, summary string, eventCount int, start, end time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaries[agentName] = append(m.summaries[agentName], summary)
	return nil
}

// GetSessionSummaries satisfies db.Database.
func (m *MockDatabase) GetSessionSummaries(agentName string, limit int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sums := m.summaries[agentName]
	if len(sums) == 0 {
		return nil, nil
	}
	start := len(sums) - limit
	if start < 0 {
		start = 0
	}
	return sums[start:], nil
}

// DecayStaleMemories satisfies db.Database.
func (m *MockDatabase) DecayStaleMemories(cutoffDays int) (decayed, pruned int, err error) {
	return 0, 0, nil
}
