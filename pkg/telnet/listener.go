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

	"github.com/zax0rz/darkpawns/pkg/db"
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
	"                H. Staerfeldt, M. Seifert, and S. Hammer\r\n\r\n"

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
			// Hostnames resolved from the IP are also checked, with a timeout to
			// avoid blocking the accept loop on slow DNS.
			banLevel := effectiveBanLevel(remoteIP, manager.GetBanManager())
			if banLevel == game.BanAll {
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

			go func(ip string, bl int) {
				handleConn(conn, manager, bl)
				connMu.Lock()
				connCount--
				connPerIP[ip]--
				if connPerIP[ip] <= 0 {
					delete(connPerIP, ip)
				}
				connMu.Unlock()
			}(remoteIP, banLevel)
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

	// Send initial negotiation: WONT echo (so client local echo is ON by default)
	tc.write([]byte{IAC, WONT, OPT_ECHO})
	tc.write([]byte{IAC, WILL, OPT_SGA})
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
	tc.writeLine("By what name do you wish to be known? ")

	// Read name with the pre-auth idle timeout (DP-912).
	name, ok := tc.readLinePreAuth()
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		tc.writeLine("\r\nGoodbye.\r\n")
		return
	}

	// Validate player name (same rules as WebSocket path)
	if !validation.IsValidPlayerName(name) {
		tc.writeLine("\r\nInvalid name. Use 2-32 characters: letters, numbers, spaces, dots, dashes, underscores.\r\n")
		return
	}

	var password string
	var newChar bool

	if strings.HasPrefix(strings.ToLower(name), "guest") {
		// Ephemeral guest bypasses password prompting!
		newChar = false
	} else if manager.HasDatabase() {
		database := manager.GetDatabase()
		var rec *db.PlayerRecord
		var err error
		rec, err = database.GetPlayer(name)
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
			// New character - ask to confirm creation
			tc.writeLine("Character does not exist. Do you want to create a new character? (Y/N): ")
			choice, ok := tc.readLinePreAuth()
			if !ok {
				return
			}
			choice = strings.TrimSpace(strings.ToLower(choice))
			if choice != "y" && choice != "yes" {
				tc.writeLine("\r\nGoodbye.\r\n")
				return
			}

			// Prompt to create and confirm a password (ECHO OFF)
			tc.write([]byte{IAC, WILL, OPT_ECHO})
			tc.writeLine("Choose a password: ")
			p1, ok := tc.readLinePreAuth()
			if !ok {
				tc.write([]byte{IAC, WONT, OPT_ECHO})
				tc.writeLine("\r\n")
				return
			}
			tc.writeLine("\r\nConfirm password: ")
			p2, ok := tc.readLinePreAuth()
			if !ok {
				tc.write([]byte{IAC, WONT, OPT_ECHO})
				tc.writeLine("\r\n")
				return
			}
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")

			if strings.TrimSpace(p1) == "" || strings.TrimSpace(p2) == "" {
				tc.writeLine("Password cannot be empty. Disconnecting.\r\n")
				return
			}
			if p1 != p2 {
				tc.writeLine("Passwords do not match. Disconnecting.\r\n")
				return
			}
			password = p1
			newChar = true
		}
	} else {
		// No DB - ask to confirm creation and create password
		tc.writeLine("No database connection. Create new character? (Y/N): ")
		choice, ok := tc.readLinePreAuth()
		if !ok {
			return
		}
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice != "y" && choice != "yes" {
			tc.writeLine("\r\nGoodbye.\r\n")
			return
		}

		tc.write([]byte{IAC, WILL, OPT_ECHO})
		tc.writeLine("Choose a password: ")
		p1, ok := tc.readLinePreAuth()
		if !ok {
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")
			return
		}
		tc.writeLine("\r\nConfirm password: ")
		p2, ok := tc.readLinePreAuth()
		if !ok {
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")
			return
		}
		tc.write([]byte{IAC, WONT, OPT_ECHO})
		tc.writeLine("\r\n")

		if strings.TrimSpace(p1) == "" || strings.TrimSpace(p2) == "" {
			tc.writeLine("Password cannot be empty. Disconnecting.\r\n")
			return
		}
		if p1 != p2 {
			tc.writeLine("Passwords do not match. Disconnecting.\r\n")
			return
		}
		password = p1
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

		// DP-928: any inbound traffic proves the TCP socket is alive. Update the
		// shared lastActive timestamp so the linkdead reaper also covers telnet.
		s.OnInboundActivity()

		line = strings.TrimSpace(line)

		_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		if s.IsCharCreating() {
			// A blank line is meaningful during character creation (e.g. the
			// "PRESS RETURN" step), so forward it as char_input rather than
			// swallowing it. Forwarding "" disconnected new players otherwise.
			if err := sendCharInput(s, line); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
		} else if line == "" {
			// Pressing Enter with no command just refreshes the prompt.
			tc.writeLine("> ")
		} else {
			parts := strings.Fields(line)
			if err := sendCommand(s, parts[0], parts[1:]); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
			tc.writeLine("> ")
		}
	}

	// Cleanup
	s.Manager().Unregister(s.PlayerName())
	s.CloseSend()
	slog.Info("Telnet disconnect", "remote_addr", remoteAddr, "player", s.PlayerName())
}

// writeLoop reads from the session's send channel and writes formatted output to the telnet conn.
// promptContainsMenu reports whether the char-create prompt text already
// embeds a formatted option menu. The static menu texts (RaceMenuText,
// ClassMenuText, HometownMenuText, …) all render options as "[X] label" lines,
// so a '[' in the prompt means the menu is already shown and the separate
// options list should NOT be printed again (DP-909: menus were doubled).
func promptContainsMenu(prompt string) bool {
	return strings.Contains(prompt, "[")
}

// renderCharCreateOptions renders the char-create options list for telnet.
// Options arrive as a JSON array of {"key":..,"label":..} objects (formerly a
// map, which randomized order). Each is printed as "  [key] label". If the
// payload is absent or in the legacy map shape, the map path is used as a
// fallback (DP-909).
func renderCharCreateOptions(tc *telnetConn, options interface{}) {
	if arr, ok := options.([]interface{}); ok {
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			k, _ := m["key"].(string)
			v, _ := m["label"].(string)
			tc.writeLine(fmt.Sprintf("  [%s] %s\r\n", k, v))
		}
		return
	}
	// Legacy map shape (pre-DP-909). Kept for mixed-version clients.
	if m, ok := options.(map[string]interface{}); ok {
		for k, v := range m {
			tc.writeLine(fmt.Sprintf("  [%s] %v\r\n", k, v))
		}
	}
}

func writeLoop(tc *telnetConn, s *session.Session) {
	ch := s.SendChannel()
	for msg := range ch {
		var sm session.ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			continue
		}
		switch sm.Type {
		case "state":
			// Format game state as readable text
			tc.writeLine(formatState(sm))
		case "event":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				if text, ok := ed["text"].(string); ok {
					tc.writeLine(fmt.Sprintf("\r\n%s\r\n", text))
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
		case "char_create":
			if ed, ok := sm.Data.(map[string]interface{}); ok {
				prompt, _ := ed["prompt"].(string)
				if prompt != "" {
					tc.writeLine(fmt.Sprintf("\r\n%s\r\n", prompt))
				}
				// DP-909: the prompt text already carries the full formatted
				// menu (e.g. RaceMenuText). Options are now a stable-order
				// array of {key,label} for structured clients (web/agent); the
				// telnet path used to ALSO print a randomized duplicate
				// [key] list, doubling every menu. Only render the option list
				// when the prompt did not already include the menu — detected
				// by the absence of a "[" bracket line, which the static menu
				// texts always contain.
				if !promptContainsMenu(prompt) {
					renderCharCreateOptions(tc, ed["options"])
				}
				tc.writeLine("> ")
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
	d, ok := sm.Data.(map[string]interface{})
	if !ok {
		return string(sm.Data.([]byte))
	}

	var b strings.Builder
	b.WriteString("\r\n---\r\n")

	if player, ok := d["player"].(map[string]interface{}); ok {
		_, _ = fmt.Fprintf(&b, "  %s", player["name"])
		if cls, ok := player["class"].(string); ok && cls != "" {
			_, _ = fmt.Fprintf(&b, " the %s", cls)
		}
		if race, ok := player["race"].(string); ok && race != "" {
			_, _ = fmt.Fprintf(&b, " (%s)", race)
		}
		_, _ = fmt.Fprintf(&b, "  Lvl %v  HP: %v/%v\r\n",
			player["level"], player["health"], player["max_health"])
	}

	if room, ok := d["room"].(map[string]interface{}); ok {
		fmt.Fprintf(&b, "\r\n  %s [%v]\r\n", room["name"], room["vnum"])
		if desc, ok := room["description"].(string); ok {
			fmt.Fprintf(&b, "  %s\r\n", desc)
		}
		// Items on the ground, NPCs, and other players present. Without these
		// a telnet player is blind to everything they can interact with —
		// loot, the mob they want to fight, who else is in the room. (DP-592)
		for _, line := range jsonStrings(room["items"]) {
			fmt.Fprintf(&b, "  %s\r\n", line)
		}
		for _, line := range jsonStrings(room["mobs"]) {
			fmt.Fprintf(&b, "  %s\r\n", line)
		}
		for _, name := range jsonStrings(room["players"]) {
			fmt.Fprintf(&b, "  %s is here.\r\n", name)
		}
		if exits, ok := room["exits"].([]interface{}); ok && len(exits) > 0 {
			names := make([]string, len(exits))
			for i, e := range exits {
				names[i] = fmt.Sprintf("%v", e)
			}
			fmt.Fprintf(&b, "  Exits: %s\r\n", strings.Join(names, ", "))
		}
	}

	b.WriteString("---\r\n")
	return b.String()
}

// jsonStrings coerces a JSON-decoded array (interface{}) into a []string,
// skipping any non-string elements.
func jsonStrings(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
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
				case OPT_ECHO, OPT_SGA:
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

func (tc *telnetConn) writeLine(s string) {
	tc.write([]byte(s))
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
