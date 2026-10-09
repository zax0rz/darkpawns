// Package engine provides core game engine functionality: game loop, logging,
// skill and affect management.
//
// logging.go — ported from utils.c (basic_mud_log, alog, mudlog, sprintbit, sprinttype)
// and comm.c (record_usage).

package engine

import (
	"fmt"
	"log/slog"
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

// ---------------------------------------------------------------------------
// Usage recording — ported from comm.c:record_usage()
// ---------------------------------------------------------------------------

// UsageCounter is an interface for counting active game sessions.
type UsageCounter interface {
	CountSessions() (connected int, playing int)
}
