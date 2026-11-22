package game

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests port-proof DP-1603: write_mud_date_to_file (src/comm.c:2624-2637)
// and read_mud_date_from_file (src/db.c:3197-3218). C's format is sprintf's
// "%d %d %d" of year month day followed by fwrite at strlen — no trailing
// newline; the reader is fscanf's "%d %d %d", with missing and corrupt files
// both keeping the epoch-derived calendar (src/db.c:3203-3218).
//
// Every test below fails if the read or write is reverted (R5h).

// setMudDateTestHooks pins the clock and redirects the record path into dir,
// restoring all touched globals on cleanup.
func setMudDateTestHooks(t *testing.T, dir string, nowDelta int64) {
	t.Helper()
	origNow, origPinned, origTime, origWeather, origPath := nowFunc, fixedTimePinned, timeInfo, weatherInfo, mudDateRecordPath
	t.Cleanup(func() {
		nowFunc = origNow
		fixedTimePinned = origPinned
		timeInfo = origTime
		weatherInfo = origWeather
		mudDateRecordPath = origPath
	})
	// beginningOfTime + secsPerMUDYear + 8*secsPerMUDMonth (month 8, day 0,
	// hour 0, year 1) — the same instant TestInitializeWeatherConsumesCPressureRoll
	// pins; month 8 also exercises the 7-12 pressure band.
	nowFunc = func() int64 { return beginningOfTime + nowDelta }
	fixedTimePinned = false
	mudDateRecordPath = filepath.Join(dir, "etc", "date_record")
}

func TestWriteMudDateFormatMatchesC(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	ResetTime()
	if err := WriteMudDateToFile(); err != nil {
		t.Fatalf("WriteMudDateToFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "etc", "date_record"))
	if err != nil {
		t.Fatalf("read back date record: %v", err)
	}
	// C: sprintf("%d %d %d", year, month, day) + fwrite at strlen — exactly
	// three space-separated integers and no trailing newline
	// (src/comm.c:2631-2633).
	want := "1 8 0"
	if string(data) != want {
		t.Fatalf("date record = %q, want C-format %q (no trailing newline)", string(data), want)
	}
}

func TestReadMudDateRecordOverridesCalendar(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	// The shipped production record is the C fixture: lib/etc/date_record
	// holds "1251 14 22" — exactly this field order (year month day).
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "date_record"), []byte("1251 14 22"), 0o600); err != nil {
		t.Fatal(err)
	}

	ResetTime()

	// src/db.c:418-421 computes the wall-clock calendar and then overlays only
	// {year,month,day} from the record; the calculated hour remains intact.
	if timeInfo.Year != 1251 || timeInfo.Month != 14 || timeInfo.Day != 22 {
		t.Fatalf("calendar after read = y%d m%d d%d, want y1251 m14 d22 from the record", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}
	if timeInfo.Hours != 0 {
		t.Fatalf("hours after read = %d, want 0 (epoch-derived, untouched by the record)", timeInfo.Hours)
	}
	if weatherInfo.Sunlight != SunDark {
		t.Fatalf("sunlight from overridden calendar's hour = %d, want %d", weatherInfo.Sunlight, SunDark)
	}
}

func TestReadMudDateMissingRecordKeepsComputedCalendar(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	ResetTime() // no record written: C's fopen fails and the SYSERR path stands

	if timeInfo.Year != 1 || timeInfo.Month != 8 || timeInfo.Day != 0 {
		t.Fatalf("calendar with no record = y%d m%d d%d, want computed y1 m8 d0 (src/db.c:3203-3208)", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}
}

func TestReadMudDateCorruptRecordKeepsComputedCalendar(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o750); err != nil {
		t.Fatal(err)
	}
	// fscanf returning fewer than 3 conversions is C's corrupt path
	// (src/db.c:3210-3211): too few fields...
	if err := os.WriteFile(filepath.Join(dir, "etc", "date_record"), []byte("1251 14"), 0o600); err != nil {
		t.Fatal(err)
	}
	ResetTime()
	if timeInfo.Year != 1 || timeInfo.Month != 8 || timeInfo.Day != 0 {
		t.Fatalf("calendar after short record = y%d m%d d%d, want computed y1 m8 d0", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}

	// ...and unparseable content.
	if err := os.WriteFile(filepath.Join(dir, "etc", "date_record"), []byte("not a date"), 0o600); err != nil {
		t.Fatal(err)
	}
	ResetTime()
	if timeInfo.Year != 1 || timeInfo.Month != 8 || timeInfo.Day != 0 {
		t.Fatalf("calendar after garbage record = y%d m%d d%d, want computed y1 m8 d0", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}
}

func TestReadMudDateSkippedWhenFixedTimePinned(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "etc", "date_record"), []byte("9999 1 1"), 0o600); err != nil {
		t.Fatal(err)
	}

	// The Go fixed-time fixture pins the epoch before ResetTime so this test can
	// isolate the startup date-record skip; the fixture seam is Go-specific.
	fixedTimePinned = true
	ResetTime()

	if timeInfo.Year != 1 || timeInfo.Month != 8 || timeInfo.Day != 0 {
		t.Fatalf("pinned calendar with a record on disk = y%d m%d d%d, want the pinned y1 m8 d0 (record ignored)", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}
}

func TestMudDateRoundTripAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	setMudDateTestHooks(t, dir, 1323000)

	// Boot once and persist, as the shutdown path does (src/comm.c:291).
	ResetTime()
	if err := WriteMudDateToFile(); err != nil {
		t.Fatalf("WriteMudDateToFile: %v", err)
	}

	// Restart with the wall clock hours changed but the record still on
	// disk: the calendar survives, the hour does not.
	nowFunc = func() int64 { return beginningOfTime + 1323000 + 5*secsPerMUDHour }
	ResetTime()

	if timeInfo.Year != 1 || timeInfo.Month != 8 || timeInfo.Day != 0 {
		t.Fatalf("calendar after restart = y%d m%d d%d, want persisted y1 m8 d0", timeInfo.Year, timeInfo.Month, timeInfo.Day)
	}
	if timeInfo.Hours != 5 {
		t.Fatalf("hours after restart = %d, want 5 (recomputed from the epoch)", timeInfo.Hours)
	}
}
