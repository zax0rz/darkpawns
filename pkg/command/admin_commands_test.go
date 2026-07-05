package command

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
)

// mockCommandManager satisfies common.CommandManager for report tests.
type mockCommandManager struct {
	mu       sync.RWMutex
	sessions []common.CommandSession
}

func (m *mockCommandManager) RegisterCommand(name string, handler func(common.CommandSession, []string) error) {
}

func (m *mockCommandManager) Sessions() []common.CommandSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions
}
func (m *mockCommandManager) Lock()           { m.mu.Lock() }
func (m *mockCommandManager) Unlock()         { m.mu.Unlock() }
func (m *mockCommandManager) RLock()          { m.mu.RLock() }
func (m *mockCommandManager) RUnlock()        { m.mu.RUnlock() }
func (m *mockCommandManager) Mu() interface{} { return &m.mu }

// mockAdminSession is a mock session with configurable fields.
type mockAdminSession struct {
	mu        sync.Mutex
	messages  []string
	name      string
	level     int
	hasPlayer bool
}

func (m *mockAdminSession) Send(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}
func (m *mockAdminSession) Close()                 {}
func (m *mockAdminSession) GetPlayer() interface{} { return nil }
func (m *mockAdminSession) GetPlayerName() string  { return m.name }
func (m *mockAdminSession) GetPlayerLevel() int    { return m.level }
func (m *mockAdminSession) HasPlayer() bool        { return m.hasPlayer }
func (m *mockAdminSession) IsAuthenticated() bool  { return true }
func (m *mockAdminSession) GetPlayerRoomVNum() int { return 1001 }

// TestCmdReportAssignsSequentialIDs verifies each report gets its own ID and
// that ack/admin messages reference the correct ID even with concurrent calls.
func TestCmdReportAssignsSequentialIDs(t *testing.T) {
	// Reset package-level report state.
	reportsMu.Lock()
	reports = reports[:0]
	reportSeq = 0
	reportsMu.Unlock()

	target := &mockAdminSession{name: "target", hasPlayer: true}
	admin := &mockAdminSession{name: "admin", hasPlayer: true}
	reporter1 := &mockAdminSession{name: "alice", hasPlayer: true}
	reporter2 := &mockAdminSession{name: "bob", hasPlayer: true}

	mgr := &mockCommandManager{sessions: []common.CommandSession{
		target, admin, reporter1, reporter2,
	}}
	ac := NewAdminCommands(mgr, nil)

	var wg sync.WaitGroup
	for _, r := range []*mockAdminSession{reporter1, reporter2} {
		r := r
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ac.cmdReport(r, []string{"target", "harassment", "bad behavior"})
		}()
	}
	wg.Wait()

	ids := make(map[int]bool)
	for _, r := range []*mockAdminSession{reporter1, reporter2} {
		found := false
		for _, msg := range r.messages {
			if strings.HasPrefix(msg, "Thank you for reporting target. Report #") {
				found = true
				var id int
				if _, err := fmt.Sscanf(msg, "Thank you for reporting target. Report #%d has been logged.", &id); err != nil {
					t.Errorf("failed to parse report id from %q: %v", msg, err)
				} else {
					ids[id] = true
				}
			}
		}
		if !found {
			t.Errorf("reporter %s did not receive acknowledgement", r.name)
		}
	}

	if len(ids) != 2 {
		t.Errorf("expected 2 distinct report IDs, got %v", ids)
	}

	adminReports := 0
	for _, msg := range admin.messages {
		if strings.Contains(msg, "REPORT [#") {
			adminReports++
		}
	}
	if adminReports != 2 {
		t.Errorf("expected admin to receive 2 REPORT notifications, got %d", adminReports)
	}
}

func TestParseDuration_ValidDays(t *testing.T) {
	d, err := parseDuration("5d")
	if err != nil {
		t.Fatalf("parseDuration(\"5d\") returned unexpected error: %v", err)
	}
	if want := 120 * time.Hour; d != want {
		t.Errorf("parseDuration(\"5d\") = %v, want %v", d, want)
	}
}

func TestParseDuration_InvalidTrailing(t *testing.T) {
	cases := []string{"5days", "5dd", "5df"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := parseDuration(tc); err == nil {
				t.Errorf("parseDuration(%q) expected error, got nil", tc)
			}
		})
	}
}

func TestParseDuration_TimeUnits(t *testing.T) {
	d, err := parseDuration("2h30m")
	if err != nil {
		t.Fatalf("parseDuration(\"2h30m\") returned unexpected error: %v", err)
	}
	if want := 2*time.Hour + 30*time.Minute; d != want {
		t.Errorf("parseDuration(\"2h30m\") = %v, want %v", d, want)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	if _, err := parseDuration(""); err == nil {
		t.Errorf("parseDuration(\"\") expected error, got nil")
	}
}
