// Package bootmarker names the one log line that means the Go server has
// finished booting.
//
// The oracle differential harness holds every scenario at a readiness gate
// until the server has rebuilt the world from the area files and the zone reset
// tables, and it holds a restarted engine at that same gate. The marker lives
// here, imported by both cmd/server and cmd/dp-oracle-diff, so the producer and
// the consumer cannot drift: rewording the server's line without rewording the
// gate is a compile error instead of a corpus-wide readiness timeout.
//
// It is operator log text. No player ever reads it, so RULEBOOK R1 is untouched.
package bootmarker

// Ready is the slog message cmd/server emits once boot is complete: the world
// has been rebuilt from the area files and the zone reset tables and the
// listeners are up.
//
// It must stay a single line with no committed formatting directives and no
// leading or trailing whitespace, because the harness matches it as a substring
// of the raw log stream.
const Ready = "world boot complete"
