# BRIEF: Report System — reportSeq persistence + DB reconciliation

**Date:** 2026-07-05
**Issues:** DP-711, DP-709
**Priority:** Medium
**Files:** `pkg/command/admin_commands.go`, `pkg/moderation/manager.go`
**Cite:** No C equivalent — Go-only admin/reporting system. The C source (`src/`) has no abuse reporting system — this is a Go addition.

---

## Fix 1: DP-711 — reportSeq resets to 0 on restart (MEDIUM)

**File:** `pkg/command/admin_commands.go` — line 33

**Problem:**
```go
var (
    reports   []Report
    reportsMu sync.RWMutex
    reportSeq int // auto-increment ID
)
```

`reportSeq` is a package-level `int` initialized to Go's zero value (0). On every process restart, the first new report gets ID 1 again, creating ambiguous report IDs with historical data persisted in the DB.

**Fix:**
Initialize `reportSeq` from the DB on startup. After the `AdminCommands` struct is created (where `ac.mod` is set), load the max existing report ID:

In `NewAdminCommands()` or an `initReports()` method:

```go
func (ac *AdminCommands) initReports() {
    if ac.mod == nil {
        return
    }
    // Get the highest existing report ID from DB to seed reportSeq.
    maxID, err := ac.mod.MaxReportID()
    if err != nil {
        slog.Warn("failed to load max report ID, starting from 0", "error", err)
        return
    }
    reportsMu.Lock()
    reportSeq = maxID
    reportsMu.Unlock()
}
```

This requires adding a `MaxReportID() int64, error` method to the moderation manager that queries `SELECT COALESCE(MAX(id), 0) FROM abuse_reports` (or equivalent). If the moderation DB interface doesn't expose this, it needs to be added.

**Alternative (simpler):** If the `Report` struct stores a DB-assigned ID (auto-increment), use that instead of the in-memory `reportSeq`. But the current code uses the in-memory seq as the report ID, so this requires the init approach.

---

## Fix 2: DP-709 — In-memory report store disconnected from DB store (MEDIUM)

**File:** `pkg/command/admin_commands.go` — `cmdInvestigate()` (line 452), `cmdListReports()` (line 474)

**Problem:**
`cmdReport` writes to both in-memory `reports` and the DB via `ac.mod.AddReport()`. But `cmdInvestigate` (line 452-468) and `cmdListReports` (line 474-522) only read from the in-memory `reports` slice. After a restart, the in-memory slice is empty — DB-persisted reports are invisible.

**Fix:**
On startup, load reports from the DB into the in-memory slice. Add a `ListReports()` method to the moderation manager:

In the moderation manager, add:
```go
func (m *Manager) ListReports() ([]AbuseReport, error) {
    // SELECT id, reporter, target, report_type, description, timestamp, status
    // FROM abuse_reports ORDER BY id DESC
}
```

Then in `AdminCommands.initReports()` (same init function as DP-711):

```go
func (ac *AdminCommands) initReports() {
    if ac.mod == nil {
        return
    }

    // Seed reportSeq from DB
    maxID, err := ac.mod.MaxReportID()
    if err != nil {
        slog.Warn("failed to load max report ID", "error", err)
    } else {
        reportsMu.Lock()
        reportSeq = int(maxID)
        reportsMu.Unlock()
    }

    // Load existing reports from DB
    dbReports, err := ac.mod.ListReports()
    if err != nil {
        slog.Warn("failed to load reports from DB", "error", err)
        return
    }

    reportsMu.Lock()
    defer reportsMu.Unlock()
    for _, dr := range dbReports {
        reports = append(reports, Report{
            ID:          int(dr.ID),
            Reporter:    dr.Reporter,
            Target:      dr.Target,
            ReportType:  string(dr.ReportType),
            Description: dr.Description,
            Timestamp:   dr.Timestamp,
            Resolved:    dr.Status == moderation.ReportStatusResolved,
        })
    }
}
```

**IMPORTANT:** Check the actual `AbuseReport` struct in `pkg/moderation/` for the correct field names and types. The conversion from `AbuseReport` to the local `Report` struct in `admin_commands.go` needs to match.

Also check if the moderation DB schema already has a `ListReports` or `GetReports` method — it might already exist and just not be called.

**Regression Test:** `pkg/command/admin_commands_test.go`
- Add `TestInitReports_LoadsFromDB` — mock the moderation manager to return reports, call initReports, verify the in-memory `reports` slice is populated and `reportSeq` is set to the max ID.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Check `pkg/moderation/manager.go` for existing ListReports/MaxReportID methods
2. Add any missing methods to the moderation manager
3. Add `initReports()` to `AdminCommands` and call it from constructor
4. Add tests

## After All Fixes

- Run `go build ./... && go vet ./... && go test ./...`
- Create feature branch: `fix/dp-711-709-report-persistence`
- Commit: `fix: reportSeq persistence + DB reconciliation on startup (DP-711, DP-709)`
- Open PR against `main`
- Mark DP-711 and DP-709 as Done in Linear
