package game

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// This file ports C's mud-date persistence: write_mud_date_to_file
// (src/comm.c:2624-2638) and read_mud_date_from_file (src/db.c:3197-3220),
// which together keep the in-game calendar (year/month/day) across server
// restarts via etc/date_record (DP-1603, approved by Zach 2026-11-20).
//
// C's call points, both preserved here:
//   - heartbeat: every 60 minutes of pulses, after record_usage
//     (src/comm.c:843);
//   - shutdown: after the sockets close and the clan table saves
//     (src/comm.c:289-291).
//
// reset_time reads the record right after deriving the calendar from the
// epoch (src/db.c:420-421): the file's {year,month,day} REPLACE the computed
// ones, while hours (and the sunlight derived from them) always come from the
// wall clock. A missing or corrupt record keeps the computed calendar, exactly
// as C's two SYSERR paths do (src/db.c:3206-3211).

// mudDateRecordPath is C's "etc/date_record" (src/comm.c:2629, src/db.c:3203).
// It is relative because the server anchors its working directory to the game
// root at boot (cmd/server/main.go), the same anchor C's relative paths use.
var mudDateRecordPath = "etc/date_record"

// readMudDateFromFile ports read_mud_date_from_file (src/db.c:3197-3220).
// Callers must hold weatherMu; on any failure the computed calendar stands,
// matching C's fallbacks (missing file: db.c:3206-3208; corrupt file:
// db.c:3210-3211).
func readMudDateFromFile() {
	data, err := os.ReadFile(filepath.Clean(mudDateRecordPath))
	if err != nil {
		slog.Error("SYSERR: File etc/date_record not found, mud date will be reset to default!")
		return
	}
	var year, month, day int
	if n, _ := fmt.Sscanf(string(data), "%d %d %d", &year, &month, &day); n != 3 {
		slog.Error("SYSERR: File etc/date_record is corrupted, mud date will be reset to default!")
		return
	}
	// db.c:421 — the record overrides only the calendar fields; hours keep
	// the wall-clock value mud_time_passed just computed.
	timeInfo.Year = year
	timeInfo.Month = month
	timeInfo.Day = day
}

// WriteMudDateToFile ports write_mud_date_to_file (src/comm.c:2624-2638):
// "%d %d %d" of year month day with no trailing newline (C fwrite's the
// sprintf result at its strlen), then C's "Saving mud time." log line. C
// opens with fopen("w"), which creates the file but not the directory; the
// MkdirAll below covers data roots that ship without etc/ (the disposable
// roots the oracle harness builds), where C never needs it because its lib
// tree always carries the directory.
func WriteMudDateToFile() error {
	weatherMu.RLock()
	record := fmt.Sprintf("%d %d %d", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	weatherMu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(mudDateRecordPath), 0o750); err != nil {
		return fmt.Errorf("create etc directory for mud date record: %w", err)
	}
	// #nosec G306 — C's fopen("w") yields 0666&umask (0644 by default).
	if err := os.WriteFile(filepath.Clean(mudDateRecordPath), []byte(record), 0o644); err != nil {
		return fmt.Errorf("write mud date record: %w", err)
	}
	slog.Info("Saving mud time.")
	return nil
}
