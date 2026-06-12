package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

type mockDatabase struct {
	getPlayerFunc      func(name string) (*db.PlayerRecord, error)
	createAgentKeyFunc func(characterName string) (string, int64, error)
}

func (m *mockDatabase) Close() error { return nil }
func (m *mockDatabase) GetPlayer(name string) (*db.PlayerRecord, error) {
	if m.getPlayerFunc != nil {
		return m.getPlayerFunc(name)
	}
	return &db.PlayerRecord{Name: name}, nil
}
func (m *mockDatabase) CreatePlayer(p *db.PlayerRecord) error          { return nil }
func (m *mockDatabase) SavePlayer(p *db.PlayerRecord) error            { return nil }
func (m *mockDatabase) UpdatePassword(playerID int, hash string) error { return nil }
func (m *mockDatabase) Exec(query string, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockDatabase) CreateAgentKey(characterName string) (string, int64, error) {
	if m.createAgentKeyFunc != nil {
		return m.createAgentKeyFunc(characterName)
	}
	return "dp_testkey123", 42, nil
}

func (m *mockDatabase) ValidateAgentKey(rawKey string) (string, int64, bool) {
	return "", 0, false
}
func (m *mockDatabase) EnsureDecisionLogPartitions() error          { return nil }
func (m *mockDatabase) NewDecisionLogWriter() *db.DecisionLogWriter { return nil }
func (m *mockDatabase) InitNarrativeMemory() error                  { return nil }
func (m *mockDatabase) WriteNarrativeMemory(mem *db.NarrativeMemory) (int64, error) {
	return 0, nil
}

func (m *mockDatabase) BootstrapMemories(agentName string, limit int) ([]*db.NarrativeMemory, error) {
	return nil, nil
}

func (m *mockDatabase) RecentMemories(agentName, sessionID string) ([]*db.NarrativeMemory, error) {
	return nil, nil
}

func (m *mockDatabase) SocialEventMemories(socialEventID string) ([]*db.NarrativeMemory, error) {
	return nil, nil
}

func (m *mockDatabase) WriteSessionSummary(agentName, sessionID, summary string, eventCount int, start, end time.Time) error {
	return nil
}

func (m *mockDatabase) GetSessionSummaries(agentName string, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockDatabase) DecayStaleMemories(cutoffDays int) (int, int, error) {
	return 0, 0, nil
}

func TestRunWithDBSuccess(t *testing.T) {
	mock := &mockDatabase{}
	err := runWithDB("Aidan", mock)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRunWithDBGetPlayerError(t *testing.T) {
	mock := &mockDatabase{
		getPlayerFunc: func(name string) (*db.PlayerRecord, error) {
			return nil, errors.New("database connection timeout")
		},
	}
	err := runWithDB("Aidan", mock)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get player") || !strings.Contains(err.Error(), "database connection timeout") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunWithDBCreateKeyError(t *testing.T) {
	mock := &mockDatabase{
		createAgentKeyFunc: func(name string) (string, int64, error) {
			return "", 0, errors.New("key gen error")
		},
	}
	err := runWithDB("Aidan", mock)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create agent key") || !strings.Contains(err.Error(), "key gen error") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunReturnsErrorOnBadDSN(t *testing.T) {
	err := run("Aidan", "invalid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
	if !strings.Contains(err.Error(), "connect to database") {
		t.Errorf("expected 'connect to database' error, got %v", err)
	}
}

func TestMainRequiresDatabaseURL(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "BE_CRASHY=1", "DATABASE_URL=")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected process to fail when DATABASE_URL is missing, but it succeeded")
	}
	if !strings.Contains(string(output), "DATABASE_URL environment variable is required") {
		t.Errorf("unexpected output: %s", string(output))
	}
}

// TestHelperProcess runs main() in a subprocess
func TestHelperProcess(t *testing.T) {
	if os.Getenv("BE_CRASHY") != "1" {
		return
	}
	os.Args = []string{os.Args[0], "-name", "test"}
	main()
}
