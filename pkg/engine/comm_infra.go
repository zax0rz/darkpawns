// Package engine — comm_infra.go: infrastructure helpers ported from
// comm.c (nonblock, set_sendbuf, get_from_q, timediff, timeadd,
// perform_subst, perform_alias, make_prompt, setup_log, open_logfile).
//
// Many of these are thin wrappers around Go standard library equivalents
// since Go's runtime and stdlib handle most low-level socket concerns.
package engine

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// timediff / timeadd — ported from comm.c lines 861-899
// ---------------------------------------------------------------------------
// In C these operate on struct timeval (seconds + microseconds).
// In Go we use time.Duration which handles this natively.
// These are convenience wrappers for compatibility.

// TimeDiff returns the difference a - b as a time.Duration.
// Ported from comm.c:timediff(). Returns 0 if b > a (clamped to zero).
func TimeDiff(a, b time.Time) time.Duration {
	d := a.Sub(b)
	if d < 0 {
		return 0
	}
	return d
}

// TimeAdd returns a + b as a time.Duration.
// Ported from comm.c:timeadd().
func TimeAdd(a, b time.Duration) time.Duration {
	return a + b
}

// ---------------------------------------------------------------------------
// nonblock — ported from comm.c line 2203
// ---------------------------------------------------------------------------
// In Go, TCP connections from net.Listener are already non-blocking at the
// runtime level (goroutine-safe I/O). This is a no-op wrapper for
// compatibility.

// Nonblock makes a socket non-blocking. In Go, net.Conn is already
// goroutine-safe and operates with non-blocking I/O internally.
// This is a no-op wrapper for interface compatibility.
func Nonblock(conn net.Conn) {
	// NOTE: Go net.Conn uses runtime-integrated non-blocking I/O.
	// No explicit fcntl needed.
}

// ---------------------------------------------------------------------------
// set_sendbuf — ported from comm.c line 1264
// ---------------------------------------------------------------------------
// Sets the kernel's send buffer size for the connection.
// Uses Go's SetWriteBuffer if available (net.TCPConn).

// SetSendBuf sets the kernel send buffer size on a TCP connection.
// Ported from comm.c:set_sendbuf().
func SetSendBuf(conn net.Conn, size int) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("SetSendBuf: not a TCP connection")
	}
	if err := tcpConn.SetWriteBuffer(size); err != nil {
		slog.Warn("SetSendBuf: SetWriteBuffer failed", "error", err)
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// txt_q / get_from_q — ported from comm.c lines 1197-1250
// ---------------------------------------------------------------------------
// In C, txt_q is a linked list of txt_block structs with manual alloc/free.
// In Go, we use a simple slice-based queue.
//
// NOTE: The Go port uses in-memory session input handling differently from
// the C original (which used linked-list input queues for descriptor data).
// The queue type here is provided for compatibility if any subsystem needs
// character-by-character queuing.

// TxtBlock represents a single queued text entry with alias tracking.
type TxtBlock struct {
	Text    string
	Aliased bool
}

// TxtQ is a simple FIFO queue of text blocks.
// Ported from comm.c struct txt_q (head/tail linked list).
type TxtQ struct {
	items []TxtBlock
}

// NewTxtQ creates an empty text queue.
func NewTxtQ() *TxtQ {
	return &TxtQ{items: make([]TxtBlock, 0)}
}

// Put enqueues a text block (port of write_to_q).
func (q *TxtQ) Put(text string, aliased bool) {
	q.items = append(q.items, TxtBlock{Text: text, Aliased: aliased})
}

// Get dequeues a text block (port of get_from_q).
// Returns false if the queue is empty.
func (q *TxtQ) Get() (TxtBlock, bool) {
	if len(q.items) == 0 {
		return TxtBlock{}, false
	}
	block := q.items[0]
	q.items = q.items[1:]
	return block, true
}

// Len returns the number of items in the queue.
func (q *TxtQ) Len() int {
	return len(q.items)
}

// Flush clears all items from the queue.
func (q *TxtQ) Flush() {
	q.items = q.items[:0]
}

// ---------------------------------------------------------------------------
// perform_subst — ported from comm.c line 2037
// ---------------------------------------------------------------------------
// Performs substitution on input strings using '^' as delimiter.
// Format: ^old^new  — replaces first occurrence of 'old' with 'new' in 'orig'.

// PerformSubst performs history-style substitution on input.
// Ported from comm.c:perform_subst(). Uses '^' as the delimiter.
// orig: the original text to apply substitution to.
// subst: the substitution string (e.g., "^old^new").
// Returns (newText, ok).
func PerformSubst(orig, subst string) (string, bool) {
	// subst must start with '^'
	if len(subst) < 2 || subst[0] != '^' {
		return "", false
	}

	// Find the second '^' delimiter
	rest := subst[1:]
	sepIdx := strings.IndexByte(rest, '^')
	if sepIdx < 0 {
		return "", false
	}

	oldStr := rest[:sepIdx]
	newStr := rest[sepIdx+1:]

	// Find first occurrence of oldStr in orig
	pos := strings.Index(orig, oldStr)
	if pos < 0 {
		return "", false
	}

	// Build result: pre + new + post
	var result strings.Builder
	result.Grow(len(orig) - len(oldStr) + len(newStr))
	result.WriteString(orig[:pos])
	result.WriteString(newStr)
	result.WriteString(orig[pos+len(oldStr):])

	return result.String(), true
}

// ---------------------------------------------------------------------------
// perform_alias — ported from comm.c line ??? (aliases)
// ---------------------------------------------------------------------------
// In C, aliases are stored per-descriptor and the first word of input is
// checked against the alias list. If matched, the alias expansion replaces
// the input.

// AliasEntry represents a single alias mapping.
type AliasEntry struct {
	Alias     string
	Expansion string
}

// PerformAlias checks for an alias match and expands if found.
// Ported from comm.c:perform_alias().
// aliases: the list of active aliases.
// input: the raw input line.
// Returns (expanded, wasAliased).
func PerformAlias(aliases []AliasEntry, input string) (string, bool) {
	if len(input) == 0 || len(aliases) == 0 {
		return input, false
	}

	// Extract the first word (command)
	firstWord := input
	if idx := strings.IndexAny(input, " \t"); idx >= 0 {
		firstWord = input[:idx]
	}

	for _, a := range aliases {
		if strings.EqualFold(a.Alias, firstWord) {
			// Replace first word with alias expansion, preserving rest of line
			rest := ""
			if len(firstWord) < len(input) {
				rest = input[len(firstWord):]
			}
			return a.Expansion + rest, true
		}
	}

	return input, false
}

// ---------------------------------------------------------------------------
// make_prompt — ported from comm.c line 1028
// ---------------------------------------------------------------------------
// Builds the player prompt string showing HP/Mana/Move, combat targets,
// AFK/INACTIVE status, invis level, and pager state.
//
// NOTE: The C version uses static buffers and ANSI color macros (CCRED, etc.).
// The Go version constructs a string with ANSI escape sequences directly.

// PromptConfig holds settings that influence prompt rendering.
type PromptConfig struct {
	ShowHP     bool
	ShowMana   bool
	ShowMove   bool
	ShowTarget bool
	ShowTank   bool
	Color      bool // Enable ANSI colors
}

// DefaultPromptConfig returns the default prompt configuration.
func DefaultPromptConfig() PromptConfig {
	return PromptConfig{
		ShowHP:     true,
		ShowMana:   true,
		ShowMove:   true,
		ShowTarget: false,
		ShowTank:   false,
		Color:      true,
	}
}

// MakePrompt builds the player prompt string.
// Ported from comm.c:make_prompt().
// Parameters:
//   - connected: false if player is logged in (playing game)
//   - showstrCount: number of pages in a scrolling display (0 if none active)
//   - showstrPage: current page number
//   - strActive: true if awaiting string input (e.g., writing mail)
//   - playerName: character name
//   - hp, maxHP, mana, maxMana, move, maxMove: current/max stats
//   - afk: away-from-keyboard flag
//   - inactive: inactive flag
//   - invisLevel: invisibility level (0 = visible)
//   - fighting: name of target being fought (empty if not fighting)
//   - config: prompt display settings
func MakePrompt(
	connected bool,
	showstrCount int,
	showstrPage int,
	strActive bool,
	playerName string,
	hp, maxHP int,
	mana, maxMana int,
	move, maxMove int,
	afk bool,
	inactive bool,
	invisLevel int,
	fighting string,
	config PromptConfig,
) string {
	// If awaiting string input: "] "
	if strActive {
		return "] "
	}

	// If showing a scrollable display (pager): page prompt
	if !connected && showstrCount > 0 {
		if config.Color {
			return fmt.Sprintf("\r\n\033[36m[ \033[31mReturn\033[36m to continue, (\033[31mq\033[36m)uit, (\033[31mr\033[36m)efresh, (\033[31mb\033[36m)ack, or page number (\033[31m%d\033[36m/\033[31m%d\033[36m) ]\033[0m",
				showstrPage, showstrCount)
		}
		return fmt.Sprintf("\r\n[ Return to continue, (q)uit, (r)efresh, (b)ack, or page number (%d/%d) ]",
			showstrPage, showstrCount)
	}

	// If connected (not playing yet): empty prompt
	var b strings.Builder

	if !connected {
		// Invisibility level (for immorts)
		if invisLevel > 0 {
			fmt.Fprintf(&b, "i%d ", invisLevel)
		}

		// HP display with color based on percentage — ported from make_prompt
		// lines: CCGRN (>= 75%), CCYEL (>= 33%), CCRED (< 33%)
		if config.ShowHP && config.Color {
			percent := float64(hp) / math.Max(float64(maxHP), 1)
			switch {
			case percent >= 0.75:
				b.WriteString("\033[0;32m") // green
			case percent >= 0.33:
				b.WriteString("\033[0;33m") // yellow
			default:
				b.WriteString("\033[0;31m") // red
			}
			fmt.Fprintf(&b, "%d", hp)
			b.WriteString("\033[0mH ")
		} else if config.ShowHP {
			fmt.Fprintf(&b, "%dH ", hp)
		}

		// Mana display
		if config.ShowMana && config.Color {
			percent := float64(mana) / math.Max(float64(maxMana), 1)
			switch {
			case percent >= 0.75:
				b.WriteString("\033[0;32m")
			case percent >= 0.33:
				b.WriteString("\033[0;33m")
			default:
				b.WriteString("\033[0;31m")
			}
			fmt.Fprintf(&b, "%d", mana)
			b.WriteString("\033[0mM ")
		} else if config.ShowMana {
			fmt.Fprintf(&b, "%dM ", mana)
		}

		// Move display
		if config.ShowMove && config.Color {
			percent := float64(move) / math.Max(float64(maxMove), 1)
			switch {
			case percent >= 0.75:
				b.WriteString("\033[0;32m")
			case percent >= 0.33:
				b.WriteString("\033[0;33m")
			default:
				b.WriteString("\033[0;31m")
			}
			fmt.Fprintf(&b, "%d", move)
			b.WriteString("\033[0mV ")
		} else if config.ShowMove {
			fmt.Fprintf(&b, "%dV ", move)
		}

		// Target status
		if config.ShowTarget && fighting != "" {
			fmt.Fprintf(&b, "\033[0;31m%s\033[0m ", fighting)
		}

		// AFK status
		if afk {
			b.WriteString("\033[0;31mAFK\033[0m ")
		}

		// Inactive status
		if inactive {
			b.WriteString("\033[0;31mINACTIVE\033[0m ")
		}

		b.WriteString("> ")
	}

	// For unconnected descriptors or pager: return builder content or empty
	result := b.String()
	if result == "" && !connected {
		return "> "
	}
	return result
}

// ---------------------------------------------------------------------------
// setup_log / open_logfile — ported from comm.c lines 2561-2626
// ---------------------------------------------------------------------------
// In C, these manage a global log FILE* and attempt freopen/fopen fallback.
// In Go, structured logging via slog replaces this functionality entirely.
// These are informational wrappers for interface compatibility.

// SetupLog configures the logging system. This is a no-op in the Go port
// because slog handles log destination configuration separately.
// Ported from comm.c:setup_log().
func SetupLog(filename string) {
	slog.Info("log setup (Go slog handles log configuration; setup_log is a compat wrapper)",
		"requestedFilename", filename)
}

// OpenLogfile attempts to open a log file. In the Go port this is a no-op
// since slog handles file output via standard slog handler options.
// Ported from comm.c:open_logfile().
// Returns true if "successful" (always true in Go; actual log config is
// done through slog handlers).
func OpenLogfile(filename string) bool {
	slog.Info("log file (Go slog handles file output; open_logfile is a compat stub)",
		"filename", filename)
	return true
}
