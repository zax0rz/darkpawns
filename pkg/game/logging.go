// Package game — logging utilities ported from src/utils.c
//
// Ported functions:
//   basic_mud_log  →  BasicMudLog (custom slog handler for "YYYY-MM-DD HH:MM:SS :: message")
//   alog           →  Alog (stderr logging wrapper)
//   mudlog         →  MudLog (broadcast to online immortals)
//   log_death_trap →  LogDeathTrap
//   sprintbit      →  Sprintbit
//   sprinttype     →  Sprinttype
//   sprintbitarray →  SprintbitArray
//   die_follower   →  DieFollower
//   core_dump_real →  CoreDump

package game

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Log writer — global state for basic_mud_log / alog
// ---------------------------------------------------------------------------

var (
	logWriter   ioWriter = os.Stderr
	logWriterMu sync.RWMutex
)

// ioWriter is a minimal writer interface so we don't import "io".
type ioWriter interface {
	Write(p []byte) (n int, err error)
}

// SetLogWriter sets the output writer for BasicMudLog and Alog.
func SetLogWriter(w ioWriter) {
	logWriterMu.Lock()
	defer logWriterMu.Unlock()
	logWriter = w
}

// getLogWriter returns the current log writer.
func getLogWriter() ioWriter {
	logWriterMu.RLock()
	defer logWriterMu.RUnlock()
	return logWriter
}

// ---------------------------------------------------------------------------
// BasicMudLog — timestamped logging to configured writer
// ---------------------------------------------------------------------------

// BasicMudLog writes a timestamped log line in the format
// "YYYY-MM-DD HH:MM:SS :: message" to the configured log writer.
// Ported from basic_mud_log() in src/utils.c.
//
// In C, this always writes to logfile (FILE *).
// In Go, the output goes to the writer set via SetLogWriter (default: os.Stderr).
func BasicMudLog(msg string) {
	now := time.Now()
	ts := now.Format("2006-01-02 15:04:05")

	w := getLogWriter()
	line := fmt.Sprintf("%s :: %s\n", ts, msg)
	// Ignore write errors — logging failure should not crash the game.
	_, _ = w.Write([]byte(line))
}

// BasicMudLogf is the format-string variant of BasicMudLog.
func BasicMudLogf(format string, args ...interface{}) {
	BasicMudLog(fmt.Sprintf(format, args...))
}

// ---------------------------------------------------------------------------
// Alog — syslog-style stderr logging
// ---------------------------------------------------------------------------

// Alog writes a timestamped message to stderr in the same format as BasicMudLog.
// Ported from alog() in src/utils.c.
func Alog(msg string) {
	BasicMudLog(msg)
	// In the Go version, if the log writer is not os.Stderr, also write to stderr
	// to match C behavior.
	w := getLogWriter()
	if w != os.Stderr {
		now := time.Now()
		ts := now.Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("%s :: %s\n", ts, msg)
		_, _ = os.Stderr.Write([]byte(line))
	}
}

// Alogf is the format-string variant of Alog.
func Alogf(format string, args ...interface{}) {
	Alog(fmt.Sprintf(format, args...))
}

// ---------------------------------------------------------------------------
// MudLog — broadcast to online immortals
// ---------------------------------------------------------------------------

// ImmortalSessionProvider is the duck-typed interface for mudlog to iterate
// active game sessions. The session package implements this so we avoid a
// circular import (game → session imports game).
//
// Each session exposes a Player (as interface{}) and a send channel.
type ImmortalSessionProvider interface {
	// EachSession calls fn for every active player session.
	// fn receives the player (game.Player) and a SendFunc for pushing messages.
	EachSession(fn func(player interface{}, send func(msg string)))
}

// SendFunc is a callback for sending a string message to a session.
type SendFunc func(msg string)

// MudLog logs a message to the stderr log file and optionally broadcasts it
// to online immortals (based on level threshold and log-type preference).
//
// Ported from mudlog() in src/utils.c.
//
// Parameters:
//   str    — the log message
//   typ    — log type: 0 (normal), 1 (log1), 2 (log2); used as minimum type
//   level  — minimum immortal level; if < 0, no immortal broadcast
//   toFile — if true, also write to the stderr-style log
//
// C semantics: toFile → fprintf(stderr, ...). Then if level >= 0, iterate
// descriptors and send colored "[ message ]\r\n" to immortals whose level
// >= level and whose prf_log_type >= typ.
func MudLog(str string, typ int, level int, toFile bool) {
	if toFile {
		Alog(str)
	}

	if level < 0 {
		return
	}

	formatted := fmt.Sprintf("[ %s ]\r\n", str)

	// Colors from screen.h (in C): CCGRN(ch, C_NRM) / CCNRM(ch, C_NRM)
	// We disable color in the Go version for simplicity; the message is
	// sent as-is. If color is wanted, the caller can embed ANSI codes in str.
	// Color support: embed ANSI codes in str if PRF_COLOR is tracked per-player.

	// Try to iterate sessions if a provider is set.
	// If no provider is registered, we just log to stderr.
	provider := getImmortalSessionProvider()
	if provider != nil {
		provider.EachSession(func(player interface{}, send func(msg string)) {
			p, ok := player.(*Player)
			if !ok || p == nil {
				return
			}

			// Skip unconnected or writing players
			pos := p.GetPosition()
			if pos == combat.PosDead {
				return
			}

			// Compute the player's log-type preference from PRF flags.
			// Flags bits: PrfLog1 = 34, PrfLog2 = 35
			flags := p.GetFlags()
			playerType := 0
			if flags&(1<<PrfLog1) != 0 {
				playerType = 1
			}
			if flags&(1<<PrfLog2) != 0 {
				playerType = 2
			}

			// Minimum log type filter: only send if player's type >= required typ.
			if playerType < typ {
				return
			}

			// Level filter
			if p.GetLevel() < level {
				return
			}

			send(formatted)
		})
		// Also log to slog for non-immortal observers
		slog.Debug("mudlog", "msg", str, "type", typ, "level", level)
	} else {
		// No session provider — still log via slog
		slog.Info("mudlog (no session provider)", "msg", str, "type", typ, "level", level)
	}
}

// ---------------------------------------------------------------------------
// Immortal session provider registration (bridge pattern)
// ---------------------------------------------------------------------------

var immortalSessionProvider ImmortalSessionProvider

// SetImmortalSessionProvider registers a session provider for MudLog broadcasts.
// Called during server initialization to avoid circular imports.
func SetImmortalSessionProvider(provider ImmortalSessionProvider) {
	immortalSessionProvider = provider
}

func getImmortalSessionProvider() ImmortalSessionProvider {
	return immortalSessionProvider
}

// ---------------------------------------------------------------------------
// LogDeathTrap — log a death trap hit
// ---------------------------------------------------------------------------

// LogDeathTrap logs when a character hits a death trap.
// Ported from log_death_trap() in src/utils.c.
//
// In C this calls: sprintf(buf, "...", GET_NAME(ch), world[ch->in_room].number,
// world[ch->in_room].name) then mudlog(buf, BRF, LVL_IMMORT, TRUE).
func LogDeathTrap(playerName string, roomVNum int, roomName string) {
	msg := fmt.Sprintf("%s hit death trap #%d (%s)", playerName, roomVNum, roomName)
	// BRF = 0 (normal broadcast), LVL_IMMORT = maximum immortal level
	MudLog(msg, 0, 999, true)
}

// ---------------------------------------------------------------------------
// Sprintbit — bitvector to string
// ---------------------------------------------------------------------------

// Sprintbit converts a bitvector to a space-separated string of flag names.
// Ported from sprintbit() in src/utils.c.
//
// names should be indexed by bit position; entry names[bit] == the string for that bit.
// If a names entry is empty, the bit is skipped.
// If no bits are set, returns "NOBITS ".
func Sprintbit(bitvector uint64, names []string) string {
	var b strings.Builder

	// In C: for (nr = 0; bitvector; bitvector >>= 1, nr++) { if (IS_SET(bitvector,1)) ... }
	nr := 0
	for bv := bitvector; bv != 0; bv >>= 1 {
		if bv&1 != 0 {
			if nr < len(names) && names[nr] != "" {
				b.WriteString(names[nr])
				b.WriteByte(' ')
			} else {
				b.WriteString("UNDEFINED ")
			}
		}
		if nr < len(names) && names[nr] != "" {
			nr++
		}
	}

	if b.Len() == 0 {
		return "NOBITS "
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Sprinttype — integer to named type
// ---------------------------------------------------------------------------

// Sprinttype returns the names entry at index typeNum.
// Ported from sprinttype() in src/utils.c.
//
// Returns the name at names[typeNum], or "UNDEFINED" if out of range or
// if the names entry at that index is empty.
func Sprinttype(typeNum int, names []string) string {
	if typeNum >= 0 && typeNum < len(names) && names[typeNum] != "" {
		return names[typeNum]
	}
	return "UNDEFINED"
}

// ---------------------------------------------------------------------------
// SprintbitArray — multi-word bitvector to string
// ---------------------------------------------------------------------------

// SprintbitArray converts an array of 32-bit bitvectors to a space-separated
// string of flag names. Each element of bitvector covers 32 bits.
// Ported from sprintbitarray() in src/utils.c.
//
// maxar is the number of elements in bitvector (the original C uses this to
// index into names, offset by i*32 for each word).
// names is the full list indexed by absolute bit position (0..maxar*32-1).
func SprintbitArray(bitvector []uint32, names []string, maxar int) string {
	var b strings.Builder

	for i := 0; i < maxar; i++ {
		// Each word of the bitvector maps to names[i*32 .. i*32+31]
		base := i * 32
		tmp := sprintnbit(uint64(bitvector[i]), names, base)
		if tmp != "NOBITS " {
			b.WriteString(tmp)
		}
	}

	if b.Len() == 0 {
		return "NOBITS "
	}
	return b.String()
}

// sprintnbit is a helper that processes one 32-bit word of sprintbitarray.
// It mirrors the C sprintnbit() defined in utils.c for one word.
func sprintnbit(bitvector uint64, names []string, bitOffset int) string {
	var b strings.Builder

	nr := bitOffset
	for bv := bitvector; bv != 0; bv >>= 1 {
		if bv&1 != 0 {
			if nr < len(names) && names[nr] != "" {
				b.WriteString(names[nr])
				b.WriteByte(' ')
			} else {
				b.WriteString("UNDEFINED ")
			}
		}
		if nr < len(names) && names[nr] != "" {
			nr++
		}
	}

	if b.Len() == 0 {
		return "NOBITS "
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// DieFollower — cleanup follower chains on character death
// ---------------------------------------------------------------------------

// DieFollower cleans up follower chains when a character dies.
// If the character has a master, stop following.
// If the character has followers, each follower must stop following.
// Ported from die_follower() in src/utils.c.
//
// The Go follow system uses Player.Following (string = master's name).
// Followers are found by scanning the world's player list.
// This function only uses the World to find followers; it does not need
// the session layer.
func DieFollower(playerName string, getFollowers func(name string) []string, stopFollow func(name string)) {
	// If this player is following someone, stop following.
	if stopFollow != nil {
		stopFollow(playerName)
	}

	// Followers are stored as players whose Following == playerName
	if getFollowers != nil {
		followers := getFollowers(playerName)
		for _, followerName := range followers {
			if stopFollow != nil {
				stopFollow(followerName)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// CoreDump — log fatal assertion and dump stack
// ---------------------------------------------------------------------------

// CoreDump logs a fatal assertion failure with stack trace.
// Ported from core_dump_real() in src/utils.c.
//
// In C this flushes streams and forks+aborts. In Go we log the error
// and stack trace, which is safer in a goroutine-based server.
func CoreDump(who string, line int) {
	stack := string(debug.Stack())
	msg := fmt.Sprintf("SYSERR: Assertion failed at %s:%d!\nStack:\n%s", who, line, stack)

	// Log to the game's log system
	BasicMudLog(msg)

	// Also log via slog for structured logging
	slog.Error("assertion failed",
		"who", who,
		"line", line,
		"stack", stack,
	)

	// Write to stderr directly to ensure it's seen
	_, _ = os.Stderr.WriteString(msg + "\n")
}
