package moderation

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// These tests run the moderation store against real databases: SQLite always
// (it is the dependency-free default), PostgreSQL when DATABASE_URL points at a
// disposable database. The fake-driver tests elsewhere in this package prove
// that statements are issued; these prove the statements are valid SQL on both
// dialects, that ids autoincrement, and that the schema is idempotent. Without
// them the four tables the previous SQLite pass missed rot straight back into
// "no such function: NOW" and "no such table: word_filters" on every boot.
//
// A note on timestamps, because it shapes what these tests can assert. The
// moderation schema declares TIMESTAMP (without time zone) on all four tables.
// PostgreSQL drops the offset on write, and lib/pq reads a naive TIMESTAMP back
// as UTC, so on a host whose zone is not UTC the instant of a stored value
// shifts by the host's offset. That is pre-existing behaviour and not something
// this port introduces: pkg/db's locked_until has the same shape and its backend
// test fails identically on a non-UTC host, and pkg/command renders "expires in"
// from the same instants. These tests therefore assert the wall-clock value,
// which is what a timezone-naive column does store, and only assert instants
// where the backend preserves them.

// moderationBackends lists the backends for one test run. Same shape as
// pkg/db's gameStoreBackends.
type moderationBackend struct {
	name string
	dsn  string
}

func moderationBackends(t *testing.T) []moderationBackend {
	t.Helper()
	backends := []moderationBackend{{"sqlite", "sqlite://" + filepath.Join(t.TempDir(), "moderation.db")}}
	if pg := os.Getenv("DATABASE_URL"); pg != "" {
		backends = append(backends, moderationBackend{"postgres", pg})
	}
	return backends
}

// moderationTables is every table this package owns. The tables are created and
// queried by this package, through the connection pkg/db opened: that shared
// connection is why the dialect has to be threaded in rather than re-derived.
var moderationTables = []string{"abuse_reports", "admin_log", "player_penalties", "word_filters"}

// newModerationManager opens a real connection through pkg/db and builds the
// manager exactly as cmd/server does, handle and dialect together.
func newModerationManager(t *testing.T, dsn string) *Manager {
	t.Helper()
	store, err := db.New(dsn)
	if err != nil {
		t.Fatalf("db.New(%q): %v", dsn, err)
	}
	t.Cleanup(func() { _ = store.Close() })
	m := NewManager(store.SQLDB(), store.Dialect())
	t.Cleanup(m.Close)
	return m
}

// tableExists and columnExists read each dialect's catalog so the schema
// assertions stay exact: SQLite answers through sqlite_master and
// pragma_table_info, PostgreSQL through pg_tables and information_schema.
func tableExists(t *testing.T, m *Manager, table string) bool {
	t.Helper()
	var n int
	var err error
	if m.dialect == db.DialectSQLite {
		err = m.queryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = $1`, table).Scan(&n)
	} else {
		err = m.queryRow(`SELECT COUNT(*) FROM pg_tables WHERE schemaname = current_schema() AND tablename = $1`, table).Scan(&n)
	}
	if err != nil {
		t.Fatalf("look up table %s: %v", table, err)
	}
	return n > 0
}

func columnExists(t *testing.T, m *Manager, table, column string) bool {
	t.Helper()
	var n int
	var err error
	if m.dialect == db.DialectSQLite {
		err = m.queryRow(`SELECT COUNT(*) FROM pragma_table_info($1) WHERE name = $2`, table, column).Scan(&n)
	} else {
		err = m.queryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`, table, column).Scan(&n)
	}
	if err != nil {
		t.Fatalf("look up column %s.%s: %v", table, column, err)
	}
	return n > 0
}

var moderationNameCounter int

// TestModerationSchema is the delta-1 proof: it asserts on real inserts
// producing non-zero ids, never on CREATE TABLE succeeding. SQLite accepts
// SERIAL PRIMARY KEY silently and never autoincrements, so a test that only
// checked the DDL would ship tables whose ids are all null.
func TestModerationSchema(t *testing.T) {
	for _, be := range moderationBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			m := newModerationManager(t, be.dsn)

			for _, table := range moderationTables {
				if !tableExists(t, m, table) {
					t.Errorf("missing table %s", table)
				}
			}

			// Second pass over an existing schema: the CREATE TABLEs, the two
			// guarded ALTERs and the loads all have to be clean no-ops.
			if err := m.createTables(); err != nil {
				t.Fatalf("createTables over existing schema: %v", err)
			}

			// abuse_reports: the package inserts it but never reads the id
			// back, so MAX(id) is the observable form of "it incremented".
			name := uniqueModerationName("schema")
			if err := m.AddReport(AbuseReport{
				Reporter:    name,
				Target:      "target",
				ReportType:  ReportTypeSpam,
				Description: "schema proof",
				Status:      ReportStatusPending,
				Timestamp:   time.Now(),
			}); err != nil {
				t.Fatalf("AddReport: %v", err)
			}
			maxID, err := m.MaxReportID()
			if err != nil {
				t.Fatalf("MaxReportID: %v", err)
			}
			if maxID == 0 {
				t.Error("abuse_reports id did not autoincrement; SERIAL was not translated")
			}

			// word_filters: AddWordFilter scans RETURNING id, so a null column
			// surfaces as id 0 in the cache.
			pattern := uniqueModerationName("schemaproof")
			if err := m.AddWordFilter(pattern, false, "censor", "admin"); err != nil {
				t.Fatalf("AddWordFilter: %v", err)
			}
			filterID := 0
			for _, f := range m.GetWordFilters() {
				if f.Pattern == pattern {
					filterID = f.ID
				}
			}
			if filterID == 0 {
				t.Error("word_filters id did not autoincrement; SERIAL was not translated")
			}

			// admin_log has no insert path in this package (the audit table is
			// created but nothing writes to it), so its id is proven directly.
			var logID int
			if err := m.queryRow(
				`INSERT INTO admin_log (admin, action, target, reason) VALUES ($1, $2, $3, $4) RETURNING id`,
				"admin", string(ActionWarn), "target", "schema proof",
			).Scan(&logID); err != nil {
				t.Fatalf("insert admin_log: %v", err)
			}
			if logID == 0 {
				t.Error("admin_log id did not autoincrement; SERIAL was not translated")
			}

			// player_penalties keys on (player_name, penalty_type, issued_at)
			// and has no id at all, so the claim to check is the row itself.
			penaltyPlayer := uniqueModerationName("schemapen")
			if err := m.AddPenalty(PlayerPenalty{
				PlayerName:  penaltyPlayer,
				PenaltyType: ActionMute,
				IssuedAt:    time.Now(),
				Reason:      "schema proof",
				IssuedBy:    "admin",
			}); err != nil {
				t.Fatalf("AddPenalty: %v", err)
			}
			var rows int
			if err := m.queryRow(`SELECT COUNT(*) FROM player_penalties WHERE player_name = $1`, penaltyPlayer).Scan(&rows); err != nil {
				t.Fatalf("count player_penalties: %v", err)
			}
			if rows != 1 {
				t.Errorf("player_penalties rows = %d, want 1", rows)
			}

			// The two migration columns must be present: they are why the
			// schema pass used to stop at the second statement on SQLite.
			for _, col := range []string{"expired_at", "status"} {
				if !columnExists(t, m, "player_penalties", col) {
					t.Errorf("missing column player_penalties.%s", col)
				}
			}
		})
	}
}

// uniqueModerationName returns a name unique to this test process so PostgreSQL
// runs (which share one disposable database) do not collide across tests, and
// short enough for the VARCHAR(32) columns.
// TestModerationSchemaRestoresMigratedColumns covers the other half of delta 3:
// ADD COLUMN IF NOT EXISTS has no SQLite spelling, so the guard looks the column
// up first. Dropping the columns on an existing install and re-running the
// schema has to put them back on both dialects.
func TestModerationSchemaRestoresMigratedColumns(t *testing.T) {
	for _, be := range moderationBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			m := newModerationManager(t, be.dsn)

			// SQLite's ALTER TABLE takes one column per statement.
			drop := []string{
				`ALTER TABLE player_penalties DROP COLUMN expired_at`,
				`ALTER TABLE player_penalties DROP COLUMN status`,
			}
			if m.dialect == db.DialectPostgres {
				drop = []string{`ALTER TABLE player_penalties DROP COLUMN expired_at, DROP COLUMN status`}
			}
			for _, stmt := range drop {
				if _, err := m.db.Exec(stmt); err != nil {
					t.Fatalf("drop migration columns: %v", err)
				}
			}
			if columnExists(t, m, "player_penalties", "expired_at") || columnExists(t, m, "player_penalties", "status") {
				t.Fatal("columns still present after drop; the migration path is not under test")
			}

			if err := m.createTables(); err != nil {
				t.Fatalf("createTables after dropping migration columns: %v", err)
			}
			for _, col := range []string{"expired_at", "status"} {
				if !columnExists(t, m, "player_penalties", col) {
					t.Errorf("column player_penalties.%s was not restored", col)
				}
			}
		})
	}
}

// TestModerationPenaltyRoundTrip covers the NOW() deltas in query bodies: the
// deadline is computed in Go and bound, so a live penalty survives a restart and
// the cleanup pass can flip the stored row to expired.
func TestModerationPenaltyRoundTrip(t *testing.T) {
	for _, be := range moderationBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			m := newModerationManager(t, be.dsn)
			player := uniqueModerationName("penalty")
			now := time.Now()
			future := now.Add(time.Hour)
			past := now.Add(-time.Hour)

			if err := m.AddPenalty(PlayerPenalty{
				PlayerName:  player,
				PenaltyType: ActionMute,
				IssuedAt:    now,
				ExpiresAt:   &future,
				Reason:      "loud in ooc",
				IssuedBy:    "admin",
			}); err != nil {
				t.Fatalf("AddPenalty (live): %v", err)
			}
			if err := m.AddPenalty(PlayerPenalty{
				PlayerName:  player,
				PenaltyType: ActionBan,
				IssuedAt:    now.Add(-time.Second),
				ExpiresAt:   &past,
				Reason:      "already served",
				IssuedBy:    "admin",
			}); err != nil {
				t.Fatalf("AddPenalty (expired): %v", err)
			}

			// A second manager over the same store is the restart:
			// loadActivePenalties has to find the live penalty and skip the
			// expired one, which is the query that carried NOW(). The raw
			// cache is asserted rather than GetPlayerPenalties because the
			// cache holds what the SQL selected, before the in-memory expiry
			// filter re-reads the instant.
			reopened := newModerationManager(t, be.dsn)
			reopened.mu.RLock()
			loaded := append([]PlayerPenalty(nil), reopened.activePenalties[player]...)
			reopened.mu.RUnlock()
			if len(loaded) != 1 {
				t.Fatalf("penalties loaded after reload = %d, want 1 (the live mute): %+v", len(loaded), loaded)
			}
			if loaded[0].PenaltyType != ActionMute || loaded[0].Reason != "loud in ooc" {
				t.Errorf("reloaded penalty = %+v, want the live mute", loaded[0])
			}
			if loaded[0].ExpiresAt == nil {
				t.Fatal("reloaded expires_at is null; the column was not scanned into a time")
			}
			const wall = "2006-01-02 15:04:05"
			if got, want := loaded[0].ExpiresAt.Format(wall), future.Format(wall); got != want {
				t.Errorf("reloaded expires_at wall clock = %s, want %s", got, want)
			}

			// The instant only survives where the backend keeps the offset,
			// which is what the in-memory expiry filter compares against.
			if m.dialect == db.DialectSQLite {
				if !reopened.IsMuted(player) {
					t.Error("IsMuted = false after reload, want true")
				}
				if reopened.IsBanned(player) {
					t.Error("IsBanned = true after reload, want false: the expired ban was treated as active")
				}
			}

			// The cleanup pass marks instead of deleting, so the audit trail
			// survives. This is the UPDATE that carried two NOW() calls.
			reopened.cleanupExpiredPenalties()
			var status string
			var expiredAt sql.NullTime
			if err := reopened.queryRow(
				`SELECT status, expired_at FROM player_penalties WHERE player_name = $1 AND penalty_type = $2`,
				player, string(ActionBan),
			).Scan(&status, &expiredAt); err != nil {
				t.Fatalf("read expired penalty: %v", err)
			}
			if status != "expired" {
				t.Errorf("expired penalty status = %q, want expired", status)
			}
			if !expiredAt.Valid {
				t.Error("expired_at is null; the UPDATE did not run on this dialect")
			}
		})
	}
}

func uniqueModerationName(prefix string) string {
	moderationNameCounter++
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()+int64(moderationNameCounter))
}

// TestModerationWordFilterRoundTrip covers the word_filters statements end to
// end: INSERT ... RETURNING id, the created_at timestamp scan, and the DELETE by
// id. All three carry $N placeholders and a timestamp, so all three are
// dialect-sensitive.
func TestModerationWordFilterRoundTrip(t *testing.T) {
	for _, be := range moderationBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			m := newModerationManager(t, be.dsn)
			pattern := uniqueModerationName("rudeword")

			if err := m.AddWordFilter(pattern, false, "censor", "admin"); err != nil {
				t.Fatalf("AddWordFilter: %v", err)
			}
			id := 0
			for _, f := range m.GetWordFilters() {
				if f.Pattern == pattern {
					id = f.ID
				}
			}
			if id == 0 {
				t.Fatal("AddWordFilter did not report a real id for the inserted row")
			}

			// Reload: the id, the created_at timestamp and the action have to
			// come back intact.
			reopened := newModerationManager(t, be.dsn)
			var found *WordFilterEntry
			for _, f := range reopened.GetWordFilters() {
				if f.Pattern == pattern {
					entry := f
					found = &entry
					break
				}
			}
			if found == nil {
				t.Fatalf("filter %q was not reloaded", pattern)
			}
			if found.ID != id {
				t.Errorf("reloaded id = %d, want %d", found.ID, id)
			}
			if found.Action != FilterActionCensor || found.IsRegex {
				t.Errorf("reloaded filter = %+v, want a censoring exact match", *found)
			}
			if found.CreatedAt.IsZero() {
				t.Error("created_at did not round-trip; TIMESTAMP was not scanned as a time")
			}

			// The reloaded entry is live, not just present: the censor path
			// uses it on the next message.
			msg, action, blocked := reopened.CheckMessage("speaker", "you "+pattern+" now")
			if blocked || action != FilterActionCensor {
				t.Fatalf("CheckMessage = (%q, %v, %v), want a censor", msg, action, blocked)
			}
			if strings.Contains(msg, pattern) {
				t.Errorf("CheckMessage did not censor: %q", msg)
			}

			// Delete by id must reach the same row the reload found.
			reopened.RemoveWordFilter(found.ID)
			var remaining int
			if err := reopened.queryRow(`SELECT COUNT(*) FROM word_filters WHERE id = $1`, found.ID).Scan(&remaining); err != nil {
				t.Fatalf("count word_filters: %v", err)
			}
			if remaining != 0 {
				t.Errorf("word_filters rows for id %d = %d, want 0", found.ID, remaining)
			}
		})
	}
}

// TestModerationReportRoundTrip covers the abuse_reports insert and read-back,
// including the timestamp column and its nullable neighbours.
func TestModerationReportRoundTrip(t *testing.T) {
	for _, be := range moderationBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			m := newModerationManager(t, be.dsn)
			reporter := uniqueModerationName("reporter")
			submitted := time.Now()

			if err := m.AddReport(AbuseReport{
				Reporter:    reporter,
				Target:      "target",
				ReportType:  ReportTypeHarassment,
				Description: "would not stop following me",
				RoomVNum:    3001,
				Timestamp:   submitted,
				Status:      ReportStatusPending,
			}); err != nil {
				t.Fatalf("AddReport: %v", err)
			}

			reports, err := m.ListReports()
			if err != nil {
				t.Fatalf("ListReports: %v", err)
			}
			var found *AbuseReport
			for i := range reports {
				if reports[i].Reporter == reporter {
					found = &reports[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("report from %q was not listed", reporter)
			}
			if found.Target != "target" || found.RoomVNum != 3001 ||
				found.ReportType != ReportTypeHarassment || found.Status != ReportStatusPending {
				t.Errorf("reloaded report = %+v", *found)
			}
			// A timezone-naive column stores the wall clock, so that is the
			// claim: the instant is not preserved on PostgreSQL (see the
			// timestamps note at the top of this file).
			const wall = "2006-01-02 15:04:05"
			if got, want := found.Timestamp.Format(wall), submitted.Format(wall); got != want {
				t.Errorf("reloaded timestamp = %s, want %s", got, want)
			}
		})
	}
}
