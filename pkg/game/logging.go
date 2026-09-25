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
// mudlog message types, C's BRF and NRM (utils.h:115-116; OFF is 0, CMP
// 3): a message reaches an immortal whose syslog level is at least its type.
const (
	MudlogBrief  = 1
	MudlogNormal = 2
)

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

// MudLog is mudlog (utils.c:242-272): the message goes to the log when
// toFile is set and, unless level is negative, to every player in the game
// who is not writing, whose level is at least level and whose syslog level
// (PRF_LOG1 counts 1, PRF_LOG2 counts 2) is at least typ, as green
// "[ message ]" at the normal color level.
//
// typ is C's OFF/BRF/NRM/CMP (0-3, utils.h:114-117).
func MudLog(str string, typ int, level int, toFile bool) {
	if toFile {
		Alog(str)
	}
	if level < 0 {
		return
	}
	provider := getImmortalSessionProvider()
	if provider == nil {
		slog.Info("mudlog (no session provider)", "msg", str, "type", typ, "level", level)
		return
	}
	provider.EachSession(func(player interface{}, send func(msg string)) {
		p, ok := player.(*Player)
		if !ok || p == nil {
			return
		}
		flags := p.GetFlags()
		if flags&(1<<uint(PlrWriting)) != 0 {
			return
		}
		logLevel := 0
		if flags&(1<<uint(PrfLog1)) != 0 {
			logLevel++
		}
		if flags&(1<<uint(PrfLog2)) != 0 {
			logLevel += 2
		}
		if p.GetLevel() < level || logLevel < typ {
			return
		}
		// send_to_char(CCGRN), buf, CCNRM. The session ends each message's
		// line, so the color reset goes before the line's CRLF, not after.
		green, normal := observationColors(p, "\x1b[32m"), observationColors(p, "\x1b[0m")
		send(green + "[ " + str + " ]" + normal + "\r\n")
	})
}

// ---------------------------------------------------------------------------
// Immortal session provider registration (bridge pattern)
// ---------------------------------------------------------------------------

var (
	immortalSessionProviderMu sync.RWMutex
	immortalSessionProvider   ImmortalSessionProvider
)

// SetImmortalSessionProvider registers the session list MudLog broadcasts to
// (the session manager, which cannot be imported here). The last call wins,
// so each manager a test creates receives its own broadcasts.
func SetImmortalSessionProvider(provider ImmortalSessionProvider) {
	immortalSessionProviderMu.Lock()
	immortalSessionProvider = provider
	immortalSessionProviderMu.Unlock()
}

// ClearImmortalSessionProvider unregisters provider if it is still the
// registered one; a newer registration is left alone. Providers are
// compared by identity, so they must be comparable (the session manager is
// a pointer).
func ClearImmortalSessionProvider(provider ImmortalSessionProvider) {
	immortalSessionProviderMu.Lock()
	defer immortalSessionProviderMu.Unlock()
	if immortalSessionProvider == provider {
		immortalSessionProvider = nil
	}
}

func getImmortalSessionProvider() ImmortalSessionProvider {
	immortalSessionProviderMu.RLock()
	defer immortalSessionProviderMu.RUnlock()
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
	MudLog(msg, MudlogBrief, lvlImmort, true)
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
	return sprintnbitWithOffset(bitvector, names, 0)
}

func sprintnbitWithOffset(bitvector uint64, names []string, bitOffset int) string {
	var b strings.Builder

	// In C: for (nr = 0; bitvector; bitvector >>= 1, nr++) { if (IS_SET(bitvector,1)) ... }
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
	return sprintnbitWithOffset(bitvector, names, bitOffset)
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
	slog.Error(
		"assertion failed",
		"who", who,
		"line", line,
		"stack", stack,
	)

	// Write to stderr directly to ensure it's seen
	_, _ = os.Stderr.WriteString(msg + "\n")
}
