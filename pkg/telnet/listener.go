// Package telnet provides a raw TCP telnet listener for the Dark Pawns MUD.
package telnet

import (
	"bufio"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/validation"
)

// Telnet protocol bytes
const (
	IAC  byte = 255
	WILL byte = 251
	WONT byte = 252
	DO   byte = 253
	DONT byte = 254
	SB   byte = 250
	SE   byte = 240

	OPT_ECHO      byte = 1
	OPT_SGA       byte = 3
	OPT_MSSP      byte = 70
	OPT_GMCP      byte = 201
	OPT_COMPRESS2 byte = 86

	MSSP_VAR byte = 1
	MSSP_VAL byte = 2
)

var (
	maxConnsPerIP    = 3
	maxTotalConns    = 200
	loginIdleTimeout = 120 * time.Second // DP-912: drop parked pre-auth connections
)

const maxControlPumpPulses = 100_000

func init() {
	if v := os.Getenv("TELNET_MAX_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			slog.Warn("TELNET_MAX_CONNS invalid, using default", "value", v, "default", maxTotalConns)
		} else {
			maxTotalConns = n
		}
	}
	if v := os.Getenv("TELNET_MAX_CONNS_PER_IP"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			slog.Warn("TELNET_MAX_CONNS_PER_IP invalid, using default", "value", v, "default", maxConnsPerIP)
		} else {
			maxConnsPerIP = n
		}
	}
	// DP-912: idle timeout for pre-auth (banner/login/char-create) reads.
	// A connection parked at the banner used to idle indefinitely. Tunable via
	// LOGIN_IDLE_TIMEOUT (seconds). Invalid values keep the default.
	if v := os.Getenv("LOGIN_IDLE_TIMEOUT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			slog.Warn("LOGIN_IDLE_TIMEOUT invalid, using default", "value", v, "default", loginIdleTimeout)
		} else {
			loginIdleTimeout = time.Duration(n) * time.Second
		}
	}
}

// dnsLookupTimeout caps reverse-DNS resolution during ban checks. It is a
// variable (not a const) so tests can shorten it for deterministic timeouts.
var dnsLookupTimeout = 3 * time.Second

// lookupAddr is a package-level shim around net.LookupAddr so tests can
// inject deterministic reverse-DNS results without touching the network.
var lookupAddr = net.LookupAddr

var listenTCP = net.Listen

var startTime = time.Now()

const greetingsLogo = "\r\n\r\n" +
	"         (_____)           (_)    (_____)\r\n" +
	"   _     /  __ \\           | |    |  __ \\                            _\r\n" +
	"  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\r\n" +
	"   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\r\n" +
	" .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\r\n" +
	" ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\r\n" +
	"                                  |||\r\n" +
	"                                  |||\r\n" +
	"                                  `.'\r\n\r\n" +
	"             Based on CircleMUD 3.0 created by J. Elson and\r\n" +
	"            DikuMUD Gamma 0.0 created by K. Nyboe, T. Madsen,\r\n" +
	"                H. Staerfeldt, M. Seifert, and S. Hammer\r\n\r\n" +
	"   As of 10-17-2008 there has been a pwipe.  Enjoy your new adventures!\r\n" +
	"\r\n\r\n"

var (
	connMu   sync.Mutex
	listener net.Listener
)

var (
	connCount int
	connPerIP = map[string]int{}
)

// Listen starts a TCP telnet server on the given port. Returns immediately.
func Listen(port int, manager *session.Manager) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := listenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("telnet listen: %w", err)
	}
	slog.Info("Telnet listening", "address", addr)

	connMu.Lock()
	listener = ln
	connMu.Unlock()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				var netErr net.Error
				//nolint:staticcheck // Temporary is deprecated but required by brief specifications
				if errors.As(err, &netErr) && netErr.Temporary() {
					slog.Warn("Telnet: temporary accept error, retrying", "error", err)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				slog.Error("Telnet accept error, listener stopped", "error", err)
				return
			}
			remoteIP := ipFromAddr(conn.RemoteAddr().String())

			connMu.Lock()
			if connCount >= maxTotalConns {
				connMu.Unlock()
				_ = conn.Close() //nolint:errcheck // best-effort cleanup
				slog.Warn("Telnet: max total connections reached, rejecting", "remote_addr", conn.RemoteAddr())
				continue
			}
			if connPerIP[remoteIP] >= maxConnsPerIP {
				connMu.Unlock()
				_ = conn.Close() //nolint:errcheck // best-effort cleanup
				slog.Warn("Telnet: max per-IP connections reached, rejecting", "remote_addr", conn.RemoteAddr())
				continue
			}
			connCount++
			connPerIP[remoteIP]++
			connMu.Unlock()

			// Check site bans (DP-419 / DP-557): BanAll disconnects immediately;
			// BanNew/BanSelect allow connection but restrict at login.
			// The accept loop only runs the fast in-memory IP check. Hostname
			// reverse-DNS resolution (which can block up to dnsLookupTimeout)
			// happens inside the per-connection goroutine so a slow or
			// unresponsive resolver cannot stall the accept queue.
			banManager := manager.GetBanManager()
			if banManager.IsBanned(remoteIP) == game.BanAll {
				_ = conn.Close() //nolint:errcheck // best-effort cleanup
				slog.Warn("Telnet: BanAll connection rejected", "remote_addr", conn.RemoteAddr())
				connMu.Lock()
				connCount--
				connPerIP[remoteIP]--
				if connPerIP[remoteIP] <= 0 {
					delete(connPerIP, remoteIP)
				}
				connMu.Unlock()
				continue
			}

			go func(ip string) {
				banLevel := effectiveBanLevel(ip, banManager)
				if banLevel == game.BanAll {
					_ = conn.Close() //nolint:errcheck // best-effort cleanup
					slog.Warn("Telnet: BanAll connection rejected", "remote_addr", conn.RemoteAddr())
					connMu.Lock()
					connCount--
					connPerIP[ip]--
					if connPerIP[ip] <= 0 {
						delete(connPerIP, ip)
					}
					connMu.Unlock()
					return
				}
				handleConn(conn, manager, banLevel)
				connMu.Lock()
				connCount--
				connPerIP[ip]--
				if connPerIP[ip] <= 0 {
					delete(connPerIP, ip)
				}
				connMu.Unlock()
			}(remoteIP)
		}
	}()
	return nil
}

func ipFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// effectiveBanLevel returns the most restrictive ban level for remoteIP,
// checking both the raw IP and any hostnames returned by reverse-DNS lookup.
//
// C's gethostbyaddr() was synchronous and unbounded; this version caps DNS
// resolution at dnsLookupTimeout so a slow or unresponsive resolver cannot
// block the connection accept loop.
func effectiveBanLevel(remoteIP string, banManager *game.BanManager) int {
	level := banManager.IsBanned(remoteIP)
	// BanAll is the most restrictive level. If the in-memory IP check already
	// matches, no hostname ban can be stricter, so skip the reverse-DNS wait
	// entirely (banned IPs must be dropped without paying for a slow PTR).
	if level == game.BanAll {
		return level
	}

	type result struct {
		hostnames []string
		err       error
	}
	done := make(chan result, 1)
	lookup := lookupAddr // capture for goroutine so tests can restore package var safely
	go func() {
		names, err := lookup(remoteIP)
		done <- result{hostnames: names, err: err}
	}()

	var hostnames []string
	select {
	case res := <-done:
		if res.err != nil {
			slog.Debug("DNS lookup failed for ban check", "remote_ip", remoteIP, "error", res.err)
		} else {
			hostnames = res.hostnames
		}
	case <-time.After(dnsLookupTimeout):
		slog.Warn("DNS lookup timed out for ban check", "remote_ip", remoteIP)
	}

	for _, hostname := range hostnames {
		// PTR records commonly end with a trailing dot.
		hostname = strings.TrimSuffix(hostname, ".")
		hostLevel := banManager.IsBanned(hostname)
		if hostLevel > level {
			level = hostLevel
		}
	}
	return level
}

type telnetConn struct {
	net.Conn
	br             *bufio.Reader
	wmu            chan struct{} // buffered(1) acts as a write mutex
	manager        *session.Manager
	hasGMCP        atomic.Bool
	sess           *session.Session
	compressWriter *zlib.Writer
}

func handleConn(rawConn net.Conn, manager *session.Manager, banLevel int) {
	tc := &telnetConn{
		Conn:    rawConn,
		br:      bufio.NewReader(rawConn),
		wmu:     make(chan struct{}, 1),
		manager: manager,
	}
	defer func() {
		if tc.compressWriter != nil {
			_ = tc.compressWriter.Close()
		}
		_ = rawConn.Close()
	}()

	remoteAddr := rawConn.RemoteAddr().String()
	slog.Info("Telnet connect", "remote_addr", remoteAddr)

	// Send initial negotiation: WONT echo (so client local echo is ON by default).
	// Do NOT offer WILL SGA: the original C DarkPawns never negotiates
	// Suppress-Go-Ahead, and offering it here — combined with the transient
	// WILL ECHO around the password prompt — is the classic character-at-a-time
	// signature that makes line-mode clients (e.g. Mudlet) warn and mishandle
	// input. Password echo-off works without it.
	tc.write([]byte{IAC, WONT, OPT_ECHO})
	tc.write([]byte{IAC, WILL, OPT_MSSP})
	tc.write([]byte{IAC, WILL, OPT_GMCP})
	tc.write([]byte{IAC, WILL, OPT_COMPRESS2})

	s := manager.NewSession()
	tc.sess = s
	s.SetCloseFunc(func() { _ = rawConn.Close() })
	remoteIP := ipFromAddr(remoteAddr)
	s.SetRemoteIP(remoteIP)
	if banLevel != game.BanNot {
		s.SetBanLevel(banLevel)
	}

	// Welcome + prompt
	tc.writeLine(greetingsLogo)
	// C emits one visible line break at the ident-to-name boundary. Use a
	// well-formed CRLF rather than carrying its legacy LFCR framing forward.
	tc.writeLine("\r\nBy what name do you wish to be known? ")

	// C's CON_GET_NAME keeps the connection open until it receives a valid
	// fantasy name. Transport only owns this first read; new-character dialogue
	// after it belongs entirely to the shared session nanny.
	var name string
	for {
		var ok bool
		name, ok = tc.readLinePreAuth()
		if !ok {
			return
		}
		name = strings.TrimSpace(name)
		if name == "" {
			tc.writeLine("\r\nGoodbye.\r\n")
			return
		}
		if strings.HasPrefix(strings.ToLower(name), "guest") ||
			(validation.IsValidPlayerName(name) && game.ValidNameNoActive(name)) {
			break
		}
		tc.writeLine("Invalid name, please try another.\r\nName: ")
	}

	var password string
	var newChar bool

	if strings.HasPrefix(strings.ToLower(name), "guest") {
		// Ephemeral guest bypasses password prompting!
		newChar = false
	} else if manager.HasDatabase() {
		database := manager.GetDatabase()
		rec, err := database.GetPlayer(name)
		if err != nil {
			slog.Error("Telnet DB lookup error", "player", name, "error", err)
		}

		if rec != nil {
			// Returning player - prompt for password statefully (ECHO OFF)
			tc.write([]byte{IAC, WILL, OPT_ECHO})
			tc.writeLine("Password: ")
			var ok bool
			password, ok = tc.readLinePreAuth()
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")
			if !ok {
				return
			}
			if strings.TrimSpace(password) == "" {
				tc.writeLine("Password cannot be empty. Disconnecting.\r\n")
				return
			}
			newChar = false
		} else {
			newChar = true
		}
	} else {
		// In no-DB/dev mode every non-guest name follows the same C creation
		// path as an unknown player-file name.
		newChar = true
	}

	// Start the output writer before login so everything login produces —
	// prompts, rejection messages, and the success welcome/look — reaches the
	// client as it is generated. (DP-591)
	done := make(chan struct{})
	go func() {
		defer close(done)
		writeLoop(tc, s)
	}()

	// Send login with password
	if err := sendLoginWithPassword(s, name, password, newChar); err != nil {
		tc.writeLine(fmt.Sprintf("\r\nLogin failed: %v\r\n", err))
		// Set a write deadline so a client that stops reading cannot block
		// writeLoop forever and leak this goroutine/file descriptor.
		_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		s.CloseSend()
		<-done
		return
	}

	// handleLogin rejects bad credentials (wrong password, invalid name, banned)
	// without returning an error — it has already queued the reason on the
	// session output channel and called CloseSend. Flush writeLoop so the
	// error message reaches the client before the raw connection closes. (DP-591)
	if s.SendClosed() {
		// Set a write deadline so a client that stops reading cannot block
		// writeLoop forever and leak this goroutine/file descriptor.
		_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		s.CloseSend()
		<-done
		return
	}

	_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	// Input loop
	for {
		line, ok := tc.readLine()
		if !ok {
			// EOF or connection error — the client hung up.
			break
		}

		line = strings.TrimSpace(line)

		_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// The oracle harness control is intercepted before player/session
		// command handling so the trigger itself consumes no command RNG, wait
		// state, or activity state. Only the pumped heartbeats may draw.
		if handlePulseControl(manager, line) {
			continue
		}

		// DP-928: any inbound traffic proves the TCP socket is alive. Update the
		// shared lastActive timestamp so the linkdead reaper also covers telnet.
		s.OnInboundActivity()

		if s.IsCharCreating() || s.IsMenuActive() {
			// A blank line is meaningful during character creation (e.g. the
			// "PRESS RETURN" step), so forward it as char_input rather than
			// swallowing it. Forwarding "" disconnected new players otherwise.
			if err := sendCharInput(s, line); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
		} else if s.IsPaging() {
			// Output pager (DP-1195): while paging, every input line — including
			// a bare RETURN (next page) — routes to the pager navigator, never
			// to ExecuteCommand (C: comm.c:617 showstr_count routing). This
			// branch sits above the `line == ""` refresh so RETURN reaches the
			// pager. No "> " prompt: the pager prints its own prompt.
			if err := sendPagerInput(s, line); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
		} else if line == "" {
			// Pressing Enter with no command just refreshes the prompt. Route it
			// through the session's send channel so writeLoop renders it in FIFO
			// order after any still-pending output (C: comm.c:643-648).
			s.SendPrompt()
		} else {
			// C-faithful tokenization (interpreter.c:883-907): a non-letter
			// first char is a one-char command, no separating space needed
			// ("'hello"). Plain whitespace splitting broke those forms.
			cmdWord, cmdArgs := session.SplitCommandInput(line)
			if err := sendCommand(s, cmdWord, cmdArgs); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
			// The prompt is enqueued after the command so writeLoop drains the
			// command's output first, then prints "> " — matching C's flush-then-
			// prompt order instead of racing the output writer goroutine.
			if !s.SendClosed() {
				s.SendPrompt()
			}
		}
		if s.SendClosed() {
			break
		}
	}

	// Cleanup
	if !s.Manager().HandleTelnetDisconnect(s) {
		s.Manager().Unregister(s.PlayerName())
		s.CloseSend()
	}
	// A successful quit queues its goodbye immediately before Unregister closes
	// the send channel. For an unexpected EOF, DetachTransport ends writeLoop
	// while retaining the open send channel for the linkdead session.
	_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	<-done
	slog.Info("Telnet disconnect", "remote_addr", remoteAddr, "player", s.PlayerName())
}

func handlePulseControl(manager *session.Manager, line string) bool {
	if !dpclock.Frozen() {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != "~dpclock" || fields[1] != "pulse" {
		return false
	}
	n, err := strconv.Atoi(fields[2])
	if err != nil || n <= 0 || n > maxControlPumpPulses {
		return false
	}
	if err := manager.PumpPulses(n); err != nil {
		slog.Error("DP_CLOCK pulse pump failed", "pulses", n, "error", err)
	}
	return true
}

// writeLoop reads from the session's send channel and writes formatted output to the telnet conn.
func writeLoop(tc *telnetConn, s *session.Session) {
	ch := s.SendChannel()
	for {
		var msg []byte
		var ok bool
		select {
		case msg, ok = <-ch:
			if !ok {
				return
			}
		case <-s.TransportDone():
			return
		}
		var sm session.ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			continue
		}
		switch sm.Type {
		case "state":
			// State is structured client data. Room text is emitted from the same
			// game ObservationResult as act() messages, so telnet must not maintain
			// a third room renderer here.
			if stateText := formatState(sm); stateText != "" {
				tc.writeLine(stateText)
			}
		case "event":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				if text, ok := ed["text"].(string); ok {
					if strings.HasSuffix(text, "\n") {
						tc.writeLine(text)
					} else {
						tc.writeLine(text + "\r\n")
					}
				}
			}
		case "error":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				if msg, ok := ed["message"].(string); ok {
					tc.writeLine(fmt.Sprintf("\r\n!! %s\r\n", msg))
				}
			}
		case "text":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				if text, ok := ed["text"].(string); ok {
					tc.writeLine(fmt.Sprintf("%s\r\n", text))
				}
			}
		case "prompt":
			// The command prompt travels through the session's send channel so
			// it is written only after the command's queued output has been
			// drained (C: comm.c:643-648 flush output, then prompt).
			prompt := "> "
			if data, ok := sm.Data.(map[string]interface{}); ok {
				if text, ok := data["text"].(string); ok && text != "" {
					prompt = text
				}
			}
			tc.writeLine(prompt)
		case "char_create":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				secret, _ := ed["secret"].(bool)
				if secret {
					tc.write([]byte{IAC, WILL, OPT_ECHO})
				} else {
					tc.write([]byte{IAC, WONT, OPT_ECHO})
				}
				prompt, _ := ed["prompt"].(string)
				if prompt != "" {
					// Nanny prompts are already byte-exact C strings. In particular,
					// MENU intentionally contains mixed LF/CR ordering, so bypass the
					// general telnet newline normalizer here.
					tc.write([]byte(prompt))
				}
			}
		case "vars":
			if tc.hasGMCP.Load() {
				if ed, ok := sm.Data.(map[string]interface{}); ok {
					// Build all GMCP frames and send in a single write to minimize syscalls.
					var buf []byte

					vitals := make(map[string]interface{})
					if hp, ok := ed["HEALTH"]; ok {
						vitals["hp"] = hp
					}
					if maxhp, ok := ed["MAX_HEALTH"]; ok {
						vitals["maxhp"] = maxhp
					}
					if mp, ok := ed["MANA"]; ok {
						vitals["mp"] = mp
					}
					if maxmp, ok := ed["MAX_MANA"]; ok {
						vitals["maxmp"] = maxmp
					}
					if mv, ok := ed["MOVE"]; ok {
						vitals["mv"] = mv
					}
					if maxmv, ok := ed["MAX_MOVE"]; ok {
						vitals["maxmv"] = maxmv
					}
					if len(vitals) > 0 {
						buf = append(buf, buildGMCPFrame("Char.Vitals", vitals)...)
					}

					status := make(map[string]interface{})
					if lvl, ok := ed["LEVEL"]; ok {
						status["level"] = lvl
					}
					if gold, ok := ed["GOLD"]; ok {
						status["gold"] = gold
					}
					if exp, ok := ed["EXP"]; ok {
						status["exp"] = exp
					}
					if len(status) > 0 {
						buf = append(buf, buildGMCPFrame("Char.Status", status)...)
					}

					room := make(map[string]interface{})
					if num, ok := ed["ROOM_VNUM"]; ok {
						room["num"] = num
					}
					if name, ok := ed["ROOM_NAME"]; ok {
						room["name"] = name
					}
					if exits, ok := ed["ROOM_EXITS"]; ok {
						room["exits"] = exits
					}
					if len(room) > 0 {
						buf = append(buf, buildGMCPFrame("Room.Info", room)...)
					}

					if inv, ok := ed["INVENTORY"]; ok {
						buf = append(buf, buildGMCPFrame("Char.Items", map[string]interface{}{"location": "inventory", "items": inv})...)
					}
					if equip, ok := ed["EQUIPMENT"]; ok {
						buf = append(buf, buildGMCPFrame("Char.Items", map[string]interface{}{"location": "equipped", "items": equip})...)
					}

					if len(buf) > 0 {
						tc.write(buf)
					}
				}
			}
		default:
			tc.writeLine(fmt.Sprintf("[%s]\r\n", string(msg)))
		}
	}
}

func formatState(sm session.ServerMessage) string {
	_ = sm
	return ""
}

// readLine reads a line, handling IAC negotiation and responding appropriately.
// The bool return is false on EOF/error and true otherwise. A blank line
// (the user simply pressing Return) returns ("", true) — distinct from EOF,
// which returns ("", false). Callers must use the bool to decide whether to
// disconnect; treating "" alone as EOF drops players who press Enter and
// breaks the "PRESS RETURN" step of character creation.
// Input exceeding maxInputLen bytes is truncated and logged.
const maxInputLen = 1024

func (tc *telnetConn) readLine() (string, bool) {
	var line []byte
	for {
		b, err := tc.br.ReadByte()
		if err != nil {
			return "", false
		}

		if b == IAC {
			cmd, err := tc.br.ReadByte()
			if err != nil {
				return "", false
			}
			switch cmd {
			case IAC:
				if len(line) >= maxInputLen {
					slog.Warn("telnet: input exceeds max length, discarding remainder", "max", maxInputLen)
					for {
						b2, err := tc.br.ReadByte()
						if err != nil {
							return "", false
						}
						if b2 == '\r' {
							if next, _ := tc.br.Peek(1); len(next) > 0 && next[0] == '\n' {
								_, _ = tc.br.ReadByte()
							}
							return string(line[:maxInputLen]), true
						}
						if b2 == '\n' {
							return string(line[:maxInputLen]), true
						}
					}
				}
				line = append(line, 0xFF)
			case WILL:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return "", false
				}
				// Respond: DO for ECHO/SGA/GMCP, DONT for everything else
				switch opt {
				case OPT_ECHO, OPT_SGA:
					tc.write([]byte{IAC, DO, opt})
				case OPT_GMCP:
					tc.write([]byte{IAC, DO, OPT_GMCP})
					tc.hasGMCP.Store(true)
					if tc.sess != nil {
						tc.sess.SetWantsStructuredData(true)
					}
				default:
					tc.write([]byte{IAC, DONT, opt})
				}
			case WONT:
				opt, _ := tc.br.ReadByte()
				tc.write([]byte{IAC, DONT, opt})
			case DO:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return "", false
				}
				switch opt {
				case OPT_ECHO:
					// Server manages ECHO explicitly (password masking).
					// Never assert WILL SGA (see connect negotiation) — a
					// client's DO SGA gets WONT via default, so the server
					// never enters the character-at-a-time signature.
					tc.write([]byte{IAC, WILL, opt})
				case OPT_MSSP:
					tc.write([]byte{IAC, WILL, OPT_MSSP})
					tc.sendMSSP()
				case OPT_GMCP:
					tc.write([]byte{IAC, WILL, OPT_GMCP})
					tc.hasGMCP.Store(true)
					if tc.sess != nil {
						tc.sess.SetWantsStructuredData(true)
					}
				case OPT_COMPRESS2:
					tc.enableCompression()
				default:
					tc.write([]byte{IAC, WONT, opt})
				}
			case DONT:
				opt, _ := tc.br.ReadByte()
				if opt == OPT_COMPRESS2 {
					break
				}
				tc.write([]byte{IAC, WONT, opt})
			case SB:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return "", false
				}
				const maxSubnegLen = 4096
				if opt == OPT_GMCP {
					var subPayload []byte
					for {
						b2, err := tc.br.ReadByte()
						if err != nil {
							return "", false
						}
						if b2 == IAC {
							b3, err := tc.br.ReadByte()
							if err != nil {
								return "", false
							}
							if b3 == SE {
								break
							}
							if b3 == IAC {
								if len(subPayload) >= maxSubnegLen {
									slog.Warn("telnet: GMCP subnegotiation payload exceeded limit, closing connection")
									return "", false
								}
								subPayload = append(subPayload, IAC)
								continue
							}
						}
						if len(subPayload) >= maxSubnegLen {
							slog.Warn("telnet: GMCP subnegotiation payload exceeded limit, closing connection")
							return "", false
						}
						subPayload = append(subPayload, b2)
					}

					tc.handleIncomingGMCP(subPayload)
				} else {
					// Skip subnegotiation until SE with length cap to prevent DoS
					skipCount := 0
					for {
						b2, err := tc.br.ReadByte()
						if err != nil {
							return "", false
						}
						skipCount++
						if skipCount > maxSubnegLen {
							slog.Warn("telnet: subnegotiation skip exceeded limit, closing connection")
							return "", false
						}
						if b2 == IAC {
							b3, err := tc.br.ReadByte()
							if err != nil {
								return "", false
							}
							if b3 == SE {
								break
							}
						}
					}
				}

			}
			continue
		}

		if b == '\r' {
			if next, _ := tc.br.Peek(1); len(next) > 0 && next[0] == '\n' {
				_, _ = tc.br.ReadByte()
			}
			if len(line) > maxInputLen {
				slog.Warn("telnet: input truncated", "length", len(line), "max", maxInputLen)
				line = line[:maxInputLen]
			}
			return string(line), true
		}
		if b == '\n' {
			if len(line) > maxInputLen {
				slog.Warn("telnet: input truncated", "length", len(line), "max", maxInputLen)
				line = line[:maxInputLen]
			}
			return string(line), true
		}
		if len(line) >= maxInputLen {
			slog.Warn("telnet: input exceeds max length, discarding remainder", "max", maxInputLen)
			for {
				b2, err := tc.br.ReadByte()
				if err != nil {
					return "", false
				}
				if b2 == '\r' {
					if next, _ := tc.br.Peek(1); len(next) > 0 && next[0] == '\n' {
						_, _ = tc.br.ReadByte()
					}
					return string(line[:maxInputLen]), true
				}
				if b2 == '\n' {
					return string(line[:maxInputLen]), true
				}
			}
		}
		line = append(line, b)
	}
}

// readLinePreAuth reads a line with the pre-auth idle deadline applied (DP-912).
// It refreshes the read deadline to now+loginIdleTimeout before reading so a
// connection parked at the banner or a password prompt is dropped instead of
// idling forever. On any failed read it sends a best-effort goodbye line; the
// caller treats the (false) return as "disconnect now".
func (tc *telnetConn) readLinePreAuth() (string, bool) {
	//nolint:errcheck // best-effort deadline; a failure here means the conn is dead anyway
	tc.SetReadDeadline(time.Now().Add(loginIdleTimeout))
	line, ok := tc.readLine()
	if !ok {
		// Best-effort goodbye. Write errors on a closed/timed-out conn are
		// ignored by writeLine; the caller still tears the connection down.
		tc.writeLine("\r\nIdle timeout reached — disconnecting.\r\n")
		slog.Info("Telnet pre-auth idle timeout / read failure, disconnecting",
			"remote_addr", addrOrNull(tc))
	}
	return line, ok
}

// addrOrNull returns the remote address of a telnetConn, or "" if unset/errored.
func addrOrNull(tc *telnetConn) string {
	if tc == nil {
		return ""
	}
	if a := tc.RemoteAddr(); a != nil {
		return a.String()
	}
	return ""
}

// write sends bytes with a simple mutex to avoid interleaving.
func (tc *telnetConn) write(data []byte) {
	tc.wmu <- struct{}{}
	tc.writeLocked(data)
	<-tc.wmu
}

// writeLocked writes data assuming the caller already holds tc.wmu.
func (tc *telnetConn) writeLocked(data []byte) {
	if tc.compressWriter != nil {
		_, _ = tc.compressWriter.Write(data)
		_ = tc.compressWriter.Flush()
	} else {
		_, _ = tc.Write(data)
	}
}

// writeLine writes a text string to the client, normalizing line endings to
// CRLF first. Much of the game's text (MOTD, room/help files under lib/world)
// is stored with bare "\n" line endings; a raw telnet terminal treats a lone
// LF as line-feed-only (cursor drops a row but does not return to column 0),
// producing the "staircase" indentation where each line starts further right
// than the last. Canonicalizing to "\r\n" here fixes every text source at the
// transport boundary. Idempotent: existing "\r\n" is preserved, not doubled.
func (tc *telnetConn) writeLine(s string) {
	tc.write([]byte(normalizeCRLF(s)))
}

// normalizeCRLF converts any mix of "\r\n", lone "\r", and lone "\n" line
// endings into canonical "\r\n". Applied to all text written to telnet clients.
func normalizeCRLF(s string) string {
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// enableCompression starts MCCP2 compression. It sends the COMPRESS_START
// sequence uncompressed and then wraps subsequent writes in a zlib deflater.
func (tc *telnetConn) enableCompression() {
	tc.wmu <- struct{}{}
	defer func() { <-tc.wmu }()

	if tc.compressWriter != nil {
		return
	}

	// C's COMPRESS_START: IAC SB COMPRESS2 IAC SE tells the client to start
	// decompressing everything after this subnegotiation. This sentinel must be
	// sent uncompressed, so write it before installing the compressor.
	_, _ = tc.Write([]byte{IAC, SB, OPT_COMPRESS2, IAC, SE})

	zw, err := zlib.NewWriterLevel(tc, zlib.DefaultCompression)
	if err != nil {
		slog.Error("telnet: failed to create MCCP2 compressor", "error", err)
		return
	}
	tc.compressWriter = zw
}

func (tc *telnetConn) sendMSSP() {
	tc.wmu <- struct{}{}
	defer func() { <-tc.wmu }()

	var payload []byte
	payload = append(payload, IAC, SB, OPT_MSSP)

	writeField := func(name, value string) {
		payload = append(payload, MSSP_VAR)
		payload = append(payload, []byte(name)...)
		payload = append(payload, MSSP_VAL)
		payload = append(payload, []byte(value)...)
	}

	writeField("NAME", "Dark Pawns")
	writeField("PLAYERS", fmt.Sprintf("%d", tc.manager.SessionCount()))
	writeField("UPTIME", fmt.Sprintf("%d", startTime.Unix()))
	writeField("CODEBASE", "CircleMUD 3.0 (Go port)")
	writeField("FAMILY", "DikuMUD")
	writeField("CREATED", "1997")
	writeField("WEBSITE", "darkpawns.labz0rz.com")
	writeField("PORT", "7777")
	writeField("LANGUAGE", "English")
	writeField("LOCATION", "US")

	payload = append(payload, IAC, SE)

	tc.writeLocked(payload)
}

// buildGMCPFrame builds the raw bytes for a single GMCP package without sending.
// Returns nil on marshal error.
func buildGMCPFrame(pkg string, data interface{}) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("buildGMCPFrame json marshal failed", "pkg", pkg, "error", err)
		return nil
	}
	frame := make([]byte, 0, 4+len(pkg)+1+len(jsonData)+2)
	frame = append(frame, IAC, SB, OPT_GMCP)
	frame = append(frame, []byte(pkg)...)
	if len(jsonData) > 0 {
		frame = append(frame, ' ')
		frame = append(frame, jsonData...)
	}
	frame = append(frame, IAC, SE)
	return frame
}

func (tc *telnetConn) handleIncomingGMCP(payload []byte) {
	if len(payload) == 0 {
		return
	}
	s := string(payload)
	parts := strings.SplitN(s, " ", 2)
	msgName := parts[0]
	var jsonStr string
	if len(parts) > 1 {
		jsonStr = parts[1]
	}

	slog.Debug("telnet: received GMCP", "message", msgName, "payload", jsonStr)

	// Support basic hello handshake from client (like Mudlet)
	if msgName == "Core.Hello" {
		var hello struct {
			Client  string `json:"client"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &hello); err == nil {
			slog.Info("telnet: GMCP client identified", "client", hello.Client, "version", hello.Version)
			if tc.sess != nil {
				if strings.Contains(strings.ToLower(hello.Client), "brenda") || strings.Contains(strings.ToLower(hello.Client), "goat") || strings.Contains(strings.ToLower(hello.Client), "mudlet") {
					tc.sess.SetWantsStructuredData(true)
				}
			}
		}
	}
}

func sendLoginWithPassword(s *session.Session, name string, password string, newChar bool) error {
	loginData, err := json.Marshal(map[string]interface{}{
		"player_name": name,
		"password":    password,
		"new_char":    newChar,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	loginMsg, err := json.Marshal(session.ClientMessage{
		Type: "login",
		Data: loginData,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return s.HandleMessage(loginMsg)
}

func sendCharInput(s *session.Session, choice string) error {
	choiceData, err := json.Marshal(map[string]interface{}{
		"choice": choice,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	choiceMsg, err := json.Marshal(session.ClientMessage{
		Type: "char_input",
		Data: choiceData,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return s.HandleMessage(choiceMsg)
}

// sendPagerInput forwards a pager navigation line (including "" for RETURN) to
// the session's pager navigator. Mirrors sendCharInput's envelope shape.
func sendPagerInput(s *session.Session, line string) error {
	lineData, err := json.Marshal(map[string]interface{}{
		"choice": line,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	lineMsg, err := json.Marshal(session.ClientMessage{
		Type: "pager_input",
		Data: lineData,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return s.HandleMessage(lineMsg)
}

func sendCommand(s *session.Session, cmd string, args []string) error {
	cmdData, err := json.Marshal(map[string]interface{}{
		"command": cmd,
		"args":    args,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	cmdMsg, err := json.Marshal(session.ClientMessage{
		Type: "command",
		Data: cmdData,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return s.HandleMessage(cmdMsg)
}

// Stop closes the TCP telnet listener.
func Stop() {
	connMu.Lock()
	defer connMu.Unlock()
	if listener != nil {
		_ = listener.Close()
		listener = nil
	}
}
