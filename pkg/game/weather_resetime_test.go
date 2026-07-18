package game

import (
	"testing"
)

// TestMudTimePassedMatchesC locks in the faithful port of C's mud_time_passed
// (utils.c:306-327). Each vector was produced by compiling the C function
// unchanged (cc /tmp/mtp_check.c) and reading its output, so these are the
// authoritative {hours,day,month,year} for (beginningOfTime + N). The same
// sequential-subtraction arithmetic must yield identical results in Go.
//
// Calendar constants (utils.h:135-138): SECS_PER_MUD_HOUR=63, DAY=1512,
// MONTH=52920, YEAR=899640.
func TestMudTimePassedMatchesC(t *testing.T) {
	const bot = int64(650336715) // beginning_of_time, db.c:417
	tests := []struct {
		name       string
		deltaSecs  int64
		hoursDayMo [3]int
		year       int
	}{
		{"epoch zero", 0, [3]int{0, 0, 0}, 0},
		{"sub-hour truncation", 1, [3]int{0, 0, 0}, 0},
		{"just under one hour", 62, [3]int{0, 0, 0}, 0},
		{"exactly one hour", secsPerMUDHour, [3]int{1, 0, 0}, 0},
		{"hour carries floor", 64, [3]int{1, 0, 0}, 0},
		{"last hour of day zero", secsPerMUDDay - 1, [3]int{23, 0, 0}, 0},
		{"exactly one day", secsPerMUDDay, [3]int{0, 1, 0}, 0},
		{"last day of month zero", secsPerMUDMonth - 1, [3]int{23, 34, 0}, 0},
		{"exactly one month", secsPerMUDMonth, [3]int{0, 0, 1}, 0},
		{"last day of last month of year zero", secsPerMUDYear - 1, [3]int{23, 34, 16}, 0},
		{"exactly one year", secsPerMUDYear, [3]int{0, 0, 0}, 1},
		{"year plus one second", secsPerMUDYear + 1, [3]int{0, 0, 0}, 1},
		{"large elapsed (86y 11m 32d 7h)", 78000000, [3]int{7, 32, 11}, 86},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mudTimePassed(bot+tc.deltaSecs, bot)
			want := TimeInfoData{
				Hours: tc.hoursDayMo[0],
				Day:   tc.hoursDayMo[1],
				Month: tc.hoursDayMo[2],
				Year:  tc.year,
				Moon:  0, // mud_time_passed leaves moon 0; ResetTime derives it.
			}
			if got != want {
				t.Errorf("mudTimePassed(+%d) = %+v, want %+v", tc.deltaSecs, got, want)
			}
		})
	}
}

// TestMudTimePassedSubHourDoesNotCarry guards the C truncation contract: any
// sub-hour remainder is dropped (integer division), so 62 seconds is still
// hour 0. A regression that used rounding or ceiling would fail here.
func TestMudTimePassedSubHourDoesNotCarry(t *testing.T) {
	got := mudTimePassed(beginningOfTime+62, beginningOfTime)
	if got.Hours != 0 || got.Day != 0 || got.Month != 0 || got.Year != 0 {
		t.Errorf("62s should be all-zero, got %+v", got)
	}
}

// TestResetTimeDerivesClockFromEpoch verifies ResetTime ports reset_time()'s
// full derivation: time from the epoch, sunlight from the hour, moon from the
// day, and pressure from the (mocked) first PRNG draw using the derived month.
func TestResetTimeDerivesClockFromEpoch(t *testing.T) {
	originalNumber := weatherInitNumber
	originalNow := nowFunc
	t.Cleanup(func() {
		weatherInitNumber = originalNumber
		nowFunc = originalNow
	})

	// Pin a fixed clock: year 0, month 0, day 10, hours 5 (sunrise). Solve:
	//   secs = 10*secsPerMUDDay + 5*secsPerMUDHour = 15120 + 315 = 15435
	const fixedSecs = int64(10*secsPerMUDDay + 5*secsPerMUDHour)
	nowFunc = func() int64 { return beginningOfTime + fixedSecs }
	weatherInitNumber = func(_, _ int) int { return 10 }

	ResetTime()

	if timeInfo.Year != 0 || timeInfo.Month != 0 || timeInfo.Day != 10 || timeInfo.Hours != 5 {
		t.Errorf("ResetTime derived clock = %+v, want {h5 d10 m0 y0}", timeInfo)
	}
	// db.c:425-426 — hours==5 -> SUN_RISE.
	if weatherInfo.Sunlight != SunRise {
		t.Errorf("sunlight = %d, want SunRise for hour 5", weatherInfo.Sunlight)
	}
	// db.c:440 moon cascade: day 10 (< 11) -> MoonQuarterFull.
	if timeInfo.Moon != MoonQuarterFull {
		t.Errorf("moon = %d, want MoonQuarterFull for day 10", timeInfo.Moon)
	}
	// db.c:447-451 — month 0 (outside 7-12) -> range 80; 960 + mocked 10 = 970.
	if weatherInfo.Pressure != 970 {
		t.Errorf("pressure = %d, want 970 (960 + mocked 10, month 0 -> range 80)", weatherInfo.Pressure)
	}
	// db.c:455-462 — pressure 970 (<=980) -> SkyLightning.
	if weatherInfo.Sky != SkyLightning {
		t.Errorf("sky = %d, want SkyLightning for pressure 970", weatherInfo.Sky)
	}
}

// TestResetTimeSunlightBands covers each of C's sunlight branches (db.c:423-432)
// so the hour→sunlight port can't silently flip a band.
func TestResetTimeSunlightBands(t *testing.T) {
	originalNumber := weatherInitNumber
	originalNow := nowFunc
	t.Cleanup(func() {
		weatherInitNumber = originalNumber
		nowFunc = originalNow
	})
	weatherInitNumber = func(_, _ int) int { return 0 }

	cases := []struct {
		hour     int64
		sunlight int
	}{
		{0, SunDark}, // hours <= 4
		{4, SunDark},
		{5, SunRise},  // hours == 5
		{6, SunLight}, // hours <= 20
		{20, SunLight},
		{21, SunSet},  // hours == 21
		{22, SunDark}, // else
		{23, SunDark},
	}
	for _, c := range cases {
		nowFunc = func() int64 { return beginningOfTime + c.hour*secsPerMUDHour }
		ResetTime()
		if weatherInfo.Sunlight != c.sunlight {
			t.Errorf("hour %d: sunlight = %d, want %d", c.hour, weatherInfo.Sunlight, c.sunlight)
		}
	}
}

// TestResetTimeMoonCascade covers representative days across C's moon cascade
// (db.c:438-445), ensuring the day→moon mapping matches branch-for-branch.
func TestResetTimeMoonCascade(t *testing.T) {
	originalNumber := weatherInitNumber
	originalNow := nowFunc
	t.Cleanup(func() {
		weatherInitNumber = originalNumber
		nowFunc = originalNow
	})
	weatherInitNumber = func(_, _ int) int { return 0 }

	cases := []struct {
		day  int64
		moon int
	}{
		{0, MoonNew}, // day < 5
		{4, MoonNew},
		{5, MoonQuarterFull}, // day < 11
		{10, MoonQuarterFull},
		{11, MoonHalfFull},     // day < 16
		{16, MoonThreeFull},    // day < 21
		{21, MoonFull},         // day < 25
		{25, MoonQuarterEmpty}, // day < 29
		{29, MoonHalfEmpty},    // day < 33
		{33, MoonThreeEmpty},   // else (max day in a 35-day month is 34)
		{34, MoonThreeEmpty},
	}
	for _, c := range cases {
		nowFunc = func() int64 { return beginningOfTime + c.day*secsPerMUDDay }
		ResetTime()
		if timeInfo.Moon != c.moon {
			t.Errorf("day %d: moon = %d, want %d", c.day, timeInfo.Moon, c.moon)
		}
	}
}
