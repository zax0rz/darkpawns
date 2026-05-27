package agentcli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateFile manages persistent state for a character on disk.
// State is written on every significant update so the daemon can
// recover its state after a restart without re-querying the server.
type StateFile struct {
	mu       sync.Mutex
	path     string
	state    *GameState
	lastSave time.Time
}

// NewStateFile creates a state file manager for the given character.
// State is stored at ~/.dp-goat/state/<name>.json.
func NewStateFile(name string) (*StateFile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".dp-goat", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir state: %w", err)
	}
	return &StateFile{
		path:  filepath.Join(dir, name+".json"),
		state: &GameState{},
	}, nil
}

// Load reads the state file from disk. Returns nil state (not error) if
// the file doesn't exist — first run is not an error.
func (sf *StateFile) Load() (*GameState, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	data, err := os.ReadFile(sf.path)
	if err != nil {
		if os.IsNotExist(err) {
			return sf.state, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, sf.state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return sf.state, nil
}

// Save writes the current state to disk atomically (write to temp, rename).
func (sf *StateFile) Save(state *GameState) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.state = state
	sf.lastSave = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Atomic write: temp file + rename
	tmp := sf.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write state tmp: %w", err)
	}
	if err := os.Rename(tmp, sf.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}

	slog.Debug("state saved", "path", sf.path, "bytes", len(data))
	return nil
}

// Get returns the current in-memory state (no disk read).
func (sf *StateFile) Get() *GameState {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return sf.state
}

// Update modifies the state via a callback and saves to disk.
func (sf *StateFile) Update(fn func(*GameState)) error {
	sf.mu.Lock()
	fn(sf.state)
	sf.mu.Unlock()
	return sf.Save(sf.state)
}

// Path returns the state file path.
func (sf *StateFile) Path() string {
	return sf.path
}
