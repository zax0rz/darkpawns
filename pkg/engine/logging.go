// Package engine provides core game engine functionality: game loop, logging,
// skill and affect management.
//
// logging.go — ported from utils.c (basic_mud_log, alog, mudlog, sprintbit, sprinttype)
// and comm.c (record_usage).

package engine

import (
	"fmt"
	"log/slog"
	"strings"
)

// ---------------------------------------------------------------------------
// Logging — ported from utils.c
// ---------------------------------------------------------------------------

// BasicMudLog implements basic_mud_log() from utils.c.
// Writes a formatted log at the given level using slog.
// level: 0=debug, 1=info, 2=warn, 3=error (matching C log levels).
func BasicMudLog(level int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	switch {
	case level <= 0:
		slog.Debug(msg)
	case level == 1:
		slog.Info(msg)
	case level == 2:
		slog.Warn(msg)
	default:
		slog.Error(msg)
	}
}

// Alog implements alog() from utils.c — logs to a file and syslog.
// In the Go version: writes a slog.Warn with syslog prefix.
func Alog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	slog.Warn("[ALOG] " + msg)
}

// MudLog implements mudlog() from utils.c — conditional log based on level.
func MudLog(level int, logLevel int, logAll bool, format string, args ...interface{}) {
	if !logAll && level < logLevel {
		return
	}
	BasicMudLog(level, format, args...)
}

// ---------------------------------------------------------------------------
// String utilities — ported from utils.c
// ---------------------------------------------------------------------------

// Sprintbit returns a string representation of the bits set in a vector bitmask.
// Ported from sprintbit() in utils.c.
// bitvector: the bitmask to decode.
// names: slice of string names indexed by bit position (0-based).
// Returns a space-separated string of set bit names, or "NOBITS" if none.
func Sprintbit(bitvector uint64, names []string) string {
	var result []string
	for i, name := range names {
		if name != "" && (bitvector&(uint64(1)<<uint(i))) != 0 {
			result = append(result, name)
		}
	}
	if len(result) == 0 {
		return "NOBITS"
	}
	return strings.Join(result, " ")
}

// Sprinttype decodes an integer into a named type string.
// Ported from sprinttype() in utils.c.
// typeNum: the integer value to decode.
// names: slice of string names indexed by position.
// Returns the name at index typeNum, or "UNDEFINED" if out of range.
func Sprinttype(typeNum int, names []string) string {
	if typeNum >= 0 && typeNum < len(names) && names[typeNum] != "" {
		return names[typeNum]
	}
	return "UNDEFINED"
}

// Sprintbit64 is like Sprintbit but for a full 64-bit bitmask.
func Sprintbit64(bitvector uint64, names []string) string {
	return Sprintbit(bitvector, names)
}

// ---------------------------------------------------------------------------
// Usage recording — ported from comm.c:record_usage()
// ---------------------------------------------------------------------------

// UsageCounter is an interface for counting active game sessions.
type UsageCounter interface {
	CountSessions() (connected int, playing int)
}

// RecordUsage logs the current number of connected and playing sessions.
// Ported from comm.c:record_usage().
func RecordUsage(counter UsageCounter) {
	connected, playing := counter.CountSessions()
	slog.Info("usage record",
		"connected", connected,
		"playing", playing,
	)
}
