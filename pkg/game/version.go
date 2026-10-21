package game

import "time"

// compileTimeLayout is the *intended* shape of C's compile_time_str
// (src/version.c:52-60):
//
//	sprintf(buf, "%s %d, %d   ", mon, now->tm_mday, now->tm_year+1900);
//	sprintf(buf, "%s%d:", buf, now->tm_hour);   /* "%s0%d:" under 10 */
//	sprintf(buf, "%s%d",  buf, now->tm_min);    /* "%s0%d"  under 10 */
//
// i.e. "February 11, 2026   19:34": long month name, the day without zero
// padding, a comma, the four-digit year, three spaces, then zero-padded 24-hour
// HH:MM. Those sprintf calls pass their own destination as a source, which is
// undefined behaviour in C; the layout below is the output they intend (R1a).
const compileTimeLayout = "January 2, 2006   15:04"

// compileTimeStr mirrors C's compile_time_str (src/version.c:9). C leaves it
// NULL until load_compile_time() runs at boot, and SCMD_VERSION prints nothing
// while it is NULL (src/act.informative.c:2172), so "" means "not booted yet".
var compileTimeStr string

// formatCompileTime renders t the way load_compile_time() intends.
func formatCompileTime(t time.Time) string {
	return t.Format(compileTimeLayout)
}

// compileTimeNow is load_compile_time()'s clock. It is deliberately NOT
// nowFunc: nowFunc honours DP_FIXED_TIME, while C's compile_time is time(0) —
// the real boot clock (src/version.c:16-19). Pinning the port's stamp to
// DP_FIXED_TIME would print a date the C oracle never prints, and the harness's
// scenario comparison would see a divergence that no player could. It is a
// variable only so tests can read it.
var compileTimeNow = time.Now

// loadCompileTime ports load_compile_time() (src/version.c:12-62), which
// boot_db calls right after reset_time() (src/db.c:313). Despite the name it
// stamps *boot* time: C assigns compile_time = time(0), not a build timestamp,
// and formats it with localtime(), which time.Now() also returns.
func loadCompileTime() {
	compileTimeStr = formatCompileTime(compileTimeNow())
}

// CompileTimeString returns the boot-time string SCMD_VERSION prints for
// immortals, or "" before boot (C's NULL compile_time_str).
func CompileTimeString() string {
	return compileTimeStr
}
