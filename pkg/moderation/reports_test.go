package moderation

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
)

// fakeReportsDriver is a minimal database/sql/driver implementation that only
// knows how to answer the two read queries MaxReportID and ListReports issue,
// so their scanning logic (including nullable columns) can be exercised
// without a live Postgres instance.
var (
	fakeReportsMu      sync.Mutex
	fakeReportsFail    bool
	fakeMaxReportIDVal int
	fakeReportRowsVal  []AbuseReport
)

func init() {
	sql.Register("fakereportsdb", fakeReportsDriver{})
}

func setFakeReportsFail(fail bool) {
	fakeReportsMu.Lock()
	defer fakeReportsMu.Unlock()
	fakeReportsFail = fail
}

func setFakeMaxReportID(id int) {
	fakeReportsMu.Lock()
	defer fakeReportsMu.Unlock()
	fakeMaxReportIDVal = id
}

func setFakeReportRows(rows []AbuseReport) {
	fakeReportsMu.Lock()
	defer fakeReportsMu.Unlock()
	fakeReportRowsVal = rows
}

type fakeReportsDriver struct{}

func (fakeReportsDriver) Open(name string) (driver.Conn, error) {
	return &fakeReportsConn{}, nil
}

type fakeReportsConn struct{}

func (c *fakeReportsConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("fakeReportsConn: Prepare not implemented")
}
func (c *fakeReportsConn) Close() error { return nil }
func (c *fakeReportsConn) Begin() (driver.Tx, error) {
	return nil, errors.New("fakeReportsConn: Begin not implemented")
}

func (c *fakeReportsConn) CheckNamedValue(nv *driver.NamedValue) error { return nil }

func (c *fakeReportsConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	fakeReportsMu.Lock()
	defer fakeReportsMu.Unlock()

	if fakeReportsFail {
		return nil, errors.New("simulated query error")
	}

	switch {
	case strings.Contains(query, "MAX(id)"):
		return &fakeMaxIDRows{value: fakeMaxReportIDVal}, nil
	case strings.Contains(query, "FROM abuse_reports"):
		data := make([]AbuseReport, len(fakeReportRowsVal))
		copy(data, fakeReportRowsVal)
		return &fakeReportRows{data: data}, nil
	}
	return nil, fmt.Errorf("fakeReportsConn: unexpected query: %s", query)
}

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

type fakeReportRows struct {
	data []AbuseReport
	idx  int
}

func (r *fakeReportRows) Columns() []string {
	return []string{
		"id", "reporter", "target", "report_type", "description",
		"room_vnum", "timestamp", "status", "reviewed_by", "reviewed_at", "resolution",
	}
}
func (r *fakeReportRows) Close() error { return nil }
func (r *fakeReportRows) Next(dest []driver.Value) error {
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

func newFakeReportsManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("fakereportsdb", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Manager{db: db, hasDB: true}
}

func TestMaxReportID_NoDB(t *testing.T) {
	m := NewManager(nil)
	maxID, err := m.MaxReportID()
	if err != nil {
		t.Fatalf("MaxReportID() error = %v, want nil", err)
	}
	if maxID != 0 {
		t.Errorf("MaxReportID() = %d, want 0 with no DB", maxID)
	}
}

func TestMaxReportID_ReturnsMax(t *testing.T) {
	setFakeReportsFail(false)
	setFakeMaxReportID(42)
	m := newFakeReportsManager(t)

	maxID, err := m.MaxReportID()
	if err != nil {
		t.Fatalf("MaxReportID() error = %v", err)
	}
	if maxID != 42 {
		t.Errorf("MaxReportID() = %d, want 42", maxID)
	}
}

func TestMaxReportID_DBError(t *testing.T) {
	setFakeReportsFail(true)
	defer setFakeReportsFail(false)
	m := newFakeReportsManager(t)

	if _, err := m.MaxReportID(); err == nil {
		t.Fatal("expected error when the query fails, got nil")
	}
}

func TestListReports_NoDB(t *testing.T) {
	m := NewManager(nil)
	reports, err := m.ListReports()
	if err != nil {
		t.Fatalf("ListReports() error = %v, want nil", err)
	}
	if reports != nil {
		t.Errorf("ListReports() = %v, want nil with no DB", reports)
	}
}

func TestListReports_ReturnsReportsWithNullableFields(t *testing.T) {
	setFakeReportsFail(false)
	reviewedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	setFakeReportRows([]AbuseReport{
		{
			ID: 2, Reporter: "alice", Target: "bob", ReportType: ReportTypeHarassment,
			Description: "rude", RoomVNum: 3001, Timestamp: reviewedAt, Status: ReportStatusResolved,
			ReviewedBy: "admin", ReviewedAt: &reviewedAt, Resolution: "warned",
		},
		{
			ID: 1, Reporter: "carol", Target: "dave", ReportType: ReportTypeSpam,
			Description: "spam", RoomVNum: 3002, Timestamp: reviewedAt, Status: ReportStatusPending,
		},
	})
	m := newFakeReportsManager(t)

	got, err := m.ListReports()
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListReports() returned %d reports, want 2", len(got))
	}

	if got[0].ID != 2 || got[0].Status != ReportStatusResolved || got[0].ReviewedBy != "admin" || got[0].ReviewedAt == nil {
		t.Errorf("ListReports()[0] = %+v, want reviewed report with ReviewedBy/ReviewedAt set", got[0])
	}
	if got[1].ID != 1 || got[1].Status != ReportStatusPending || got[1].ReviewedBy != "" || got[1].ReviewedAt != nil {
		t.Errorf("ListReports()[1] = %+v, want pending report with empty ReviewedBy/nil ReviewedAt", got[1])
	}
}

func TestListReports_DBError(t *testing.T) {
	setFakeReportsFail(true)
	defer setFakeReportsFail(false)
	m := newFakeReportsManager(t)

	if _, err := m.ListReports(); err == nil {
		t.Fatal("expected error when the query fails, got nil")
	}
}
