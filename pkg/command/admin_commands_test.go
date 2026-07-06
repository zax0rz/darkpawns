package command

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// mockCommandManager satisfies common.CommandManager for report tests.
type mockCommandManager struct {
	mu       sync.RWMutex
	sessions []common.CommandSession
}

func (m *mockCommandManager) RegisterCommand(name string, handler func(common.CommandSession, []string) error, minLevel int) {
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

// ---------------------------------------------------------------------------
// Fake moderation DB driver — used to exercise initReports() against a
// moderation.Manager without a live Postgres instance.
// ---------------------------------------------------------------------------

var (
	fakeAdminDBMu        sync.Mutex
	fakeAdminMaxReportID int
	fakeAdminReportRows  []moderation.AbuseReport
)

func init() {
	sql.Register("fakeadmindb", fakeAdminDBDriver{})
}

func setFakeAdminMaxReportID(id int) {
	fakeAdminDBMu.Lock()
	defer fakeAdminDBMu.Unlock()
	fakeAdminMaxReportID = id
}

func setFakeAdminReportRows(rows []moderation.AbuseReport) {
	fakeAdminDBMu.Lock()
	defer fakeAdminDBMu.Unlock()
	fakeAdminReportRows = rows
}

type fakeAdminDBDriver struct{}

func (fakeAdminDBDriver) Open(name string) (driver.Conn, error) {
	return &fakeAdminDBConn{}, nil
}

type fakeAdminDBConn struct{}

func (c *fakeAdminDBConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("fakeAdminDBConn: Prepare not implemented")
}
func (c *fakeAdminDBConn) Close() error { return nil }
func (c *fakeAdminDBConn) Begin() (driver.Tx, error) {
	return nil, errors.New("fakeAdminDBConn: Begin not implemented")
}
func (c *fakeAdminDBConn) CheckNamedValue(nv *driver.NamedValue) error { return nil }

// ExecContext handles the moderation manager's CREATE TABLE IF NOT EXISTS
// calls on startup; every statement just "succeeds".
func (c *fakeAdminDBConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.ResultNoRows, nil
}

func (c *fakeAdminDBConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	fakeAdminDBMu.Lock()
	defer fakeAdminDBMu.Unlock()

	switch {
	case strings.Contains(query, "player_penalties"):
		return &fakeEmptyRows{cols: []string{"player_name", "penalty_type", "issued_at", "expires_at", "reason", "issued_by"}}, nil
	case strings.Contains(query, "word_filters"):
		return &fakeEmptyRows{cols: []string{"id", "pattern", "is_regex", "action", "created_by", "created_at"}}, nil
	case strings.Contains(query, "MAX(id)"):
		return &fakeMaxIDRows{value: fakeAdminMaxReportID}, nil
	case strings.Contains(query, "abuse_reports"):
		data := make([]moderation.AbuseReport, len(fakeAdminReportRows))
		copy(data, fakeAdminReportRows)
		return &fakeAdminReportRows2{data: data}, nil
	}
	return nil, fmt.Errorf("fakeAdminDBConn: unexpected query: %s", query)
}

// fakeEmptyRows serves zero rows for a fixed column set (used for the
// penalty/word-filter loads that initReports doesn't exercise directly).
type fakeEmptyRows struct{ cols []string }

func (r *fakeEmptyRows) Columns() []string              { return r.cols }
func (r *fakeEmptyRows) Close() error                   { return nil }
func (r *fakeEmptyRows) Next(dest []driver.Value) error { return io.EOF }

type fakeMaxIDRows struct {
	value  int
	served bool
}

func (r *fakeMaxIDRows) Columns() []string { return []string{"coalesce"} }
func (r *fakeMaxIDRows) Close() error      { return nil }
func (r *fakeMaxIDRows) Next(dest []driver.Value) error {
	if r.served {
		return io.EOF
	}
	dest[0] = int64(r.value)
	r.served = true
	return nil
}

// fakeAdminReportRows2 mirrors the abuse_reports column order ListReports
// scans (see pkg/moderation/manager.go).
type fakeAdminReportRows2 struct {
	data []moderation.AbuseReport
	idx  int
}

func (r *fakeAdminReportRows2) Columns() []string {
	return []string{
		"id", "reporter", "target", "report_type", "description",
		"room_vnum", "timestamp", "status", "reviewed_by", "reviewed_at", "resolution",
	}
}
func (r *fakeAdminReportRows2) Close() error { return nil }
func (r *fakeAdminReportRows2) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	rep := r.data[r.idx]
	r.idx++

	dest[0] = int64(rep.ID)
	dest[1] = rep.Reporter
	dest[2] = rep.Target
	dest[3] = string(rep.ReportType)
	dest[4] = rep.Description
	dest[5] = int64(rep.RoomVNum)
	dest[6] = rep.Timestamp
	dest[7] = string(rep.Status)
	if rep.ReviewedBy == "" {
		dest[8] = nil
	} else {
		dest[8] = rep.ReviewedBy
	}
	if rep.ReviewedAt == nil {
		dest[9] = nil
	} else {
		dest[9] = *rep.ReviewedAt
	}
	if rep.Resolution == "" {
		dest[10] = nil
	} else {
		dest[10] = rep.Resolution
	}
	return nil
}

// TestInitReports_LoadsFromDB verifies that constructing AdminCommands with a
// moderation manager backed by a DB seeds reportSeq from the max stored
// report ID and populates the in-memory reports slice from the DB, so both
// survive a process restart (DP-711, DP-709).
func TestInitReports_LoadsFromDB(t *testing.T) {
	reportsMu.Lock()
	reports = reports[:0]
	reportSeq = 0
	reportsMu.Unlock()

	setFakeAdminMaxReportID(7)
	now := time.Now()
	setFakeAdminReportRows([]moderation.AbuseReport{
		{
			ID: 7, Reporter: "alice", Target: "bob", ReportType: moderation.ReportTypeHarassment,
			Description: "rude", Timestamp: now, Status: moderation.ReportStatusPending,
		},
		{
			ID: 3, Reporter: "carol", Target: "dave", ReportType: moderation.ReportTypeSpam,
			Description: "spam", Timestamp: now, Status: moderation.ReportStatusResolved,
		},
	})

	db, err := sql.Open("fakeadmindb", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mod := moderation.NewManager(db)

	mgr := &mockCommandManager{}
	NewAdminCommands(mgr, mod)

	reportsMu.RLock()
	defer reportsMu.RUnlock()

	if reportSeq != 7 {
		t.Errorf("reportSeq = %d, want 7 (seeded from MaxReportID)", reportSeq)
	}
	if len(reports) != 2 {
		t.Fatalf("reports = %d entries, want 2 loaded from DB", len(reports))
	}

	foundResolved := false
	for _, r := range reports {
		if r.ID == 3 {
			foundResolved = true
			if !r.Resolved {
				t.Error("report #3 (status=resolved) should map to Resolved=true")
			}
		}
		if r.ID == 7 && r.Resolved {
			t.Error("report #7 (status=pending) should map to Resolved=false")
		}
	}
	if !foundResolved {
		t.Error("expected report #3 to be loaded from DB")
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
