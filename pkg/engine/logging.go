// Package engine provides core game engine functionality: game loop, logging,
// skill and affect management.
//
// logging.go — ported from utils.c (basic_mud_log, alog, mudlog, sprintbit, sprinttype)
// and comm.c (record_usage).

package engine

// ---------------------------------------------------------------------------
// Logging — ported from utils.c
// ---------------------------------------------------------------------------

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
