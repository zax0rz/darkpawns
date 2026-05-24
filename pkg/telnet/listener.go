// Package telnet provides a raw TCP telnet listener for the Dark Pawns MUD.
package telnet

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
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

	OPT_ECHO byte = 1
	OPT_SGA  byte = 3

	maxConnsPerIP = 3
	maxTotalConns = 200
)

var (
	connMu    sync.Mutex
	connCount int
	connPerIP = map[string]int{}
)

// Listen starts a TCP telnet server on the given port. Returns immediately.
func Listen(port int, manager *session.Manager) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("telnet listen: %w", err)
	}
	slog.Info("Telnet listening", "address", addr)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				slog.Error("Telnet accept error", "error", err)
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

			// Check site bans before allowing login (DP-296)
			if banLevel := manager.GetBanManager().IsBanned(remoteIP); banLevel > 0 {
				_ = conn.Close() //nolint:errcheck // best-effort cleanup
				slog.Warn("Telnet: banned connection rejected", "remote_addr", conn.RemoteAddr(), "ban_level", banLevel)
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
				handleConn(conn, manager)
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

type telnetConn struct {
	net.Conn
	br    *bufio.Reader
	wmu   chan struct{} // buffered(1) acts as a write mutex
}

func handleConn(rawConn net.Conn, manager *session.Manager) {
	tc := &telnetConn{
		Conn: rawConn,
		br:   bufio.NewReader(rawConn),
		wmu:  make(chan struct{}, 1),
	}
	defer func() { _ = rawConn.Close() }()

	remoteAddr := rawConn.RemoteAddr().String()
	slog.Info("Telnet connect", "remote_addr", remoteAddr)

	// Send initial negotiation: WONT echo (so client local echo is ON by default)
	tc.write([]byte{IAC, WONT, OPT_ECHO})
	tc.write([]byte{IAC, WILL, OPT_SGA})

	s := manager.NewSession()

	// Welcome + prompt
	tc.writeLine("\r\n  Dark Pawns\r\n\r\nBy what name do you wish to be known? ")

	// Read name with timeout
	//nolint:errcheck // best-effort cleanup
	rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	name := tc.readLine()
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

	// Stateful Authentication Check
	if manager.HasDatabase() {
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
			password = tc.readLine()
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")
			newChar = false
		} else {
			// New character - ask to confirm creation
			tc.writeLine("Character does not exist. Do you want to create a new character? (Y/N): ")
			choice := tc.readLine()
			choice = strings.TrimSpace(strings.ToLower(choice))
			if choice != "y" && choice != "yes" {
				tc.writeLine("\r\nGoodbye.\r\n")
				return
			}

			// Prompt to create and confirm a password (ECHO OFF)
			tc.write([]byte{IAC, WILL, OPT_ECHO})
			tc.writeLine("Choose a password: ")
			p1 := tc.readLine()
			tc.writeLine("\r\nConfirm password: ")
			p2 := tc.readLine()
			tc.write([]byte{IAC, WONT, OPT_ECHO})
			tc.writeLine("\r\n")

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
		choice := tc.readLine()
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice != "y" && choice != "yes" {
			tc.writeLine("\r\nGoodbye.\r\n")
			return
		}

		tc.write([]byte{IAC, WILL, OPT_ECHO})
		tc.writeLine("Choose a password: ")
		p1 := tc.readLine()
		tc.writeLine("\r\nConfirm password: ")
		p2 := tc.readLine()
		tc.write([]byte{IAC, WONT, OPT_ECHO})
		tc.writeLine("\r\n")

		if p1 != p2 {
			tc.writeLine("Passwords do not match. Disconnecting.\r\n")
			return
		}
		password = p1
		newChar = true
	}

	// Send login with password
	if err := sendLoginWithPassword(s, name, password, newChar); err != nil {
		tc.writeLine(fmt.Sprintf("\r\nLogin failed: %v\r\n", err))
		return
	}

	_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	// Start output writer goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		writeLoop(tc, s)
	}()

	// Input loop
	for {
		line := tc.readLine()
		if line == "" {
			// EOF or error
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if !s.IsCharCreating() {
				tc.writeLine("> ")
			}
			continue
		}

		_ = rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		if s.IsCharCreating() {
			// Route selection choice as char_input
			if err := sendCharInput(s, line); err != nil {
				tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
			}
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
				if prompt, ok := ed["prompt"].(string); ok {
					tc.writeLine(fmt.Sprintf("\r\n%s\r\n", prompt))
				}
				if options, ok := ed["options"].(map[string]interface{}); ok {
					for k, v := range options {
						tc.writeLine(fmt.Sprintf("  [%s] %s\r\n", k, v))
					}
				}
				tc.writeLine("> ")
			}
		case "vars":
			// Agent vars — skip for telnet
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

// readLine reads a line, handling IAC negotiation and responding appropriately.
// Returns empty string on EOF/error.
// Input exceeding maxInputLen bytes is truncated and logged.
const maxInputLen = 1024

func (tc *telnetConn) readLine() string {
	var line []byte
	for {
		b, err := tc.br.ReadByte()
		if err != nil {
			return ""
		}

		if b == IAC {
			cmd, err := tc.br.ReadByte()
			if err != nil {
				return ""
			}
			switch cmd {
			case IAC:
				line = append(line, 0xFF)
			case WILL:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return ""
				}
				// Respond: DO for ECHO/SGA, DONT for everything else
				if opt == OPT_ECHO || opt == OPT_SGA {
					tc.write([]byte{IAC, DO, opt})
				} else {
					tc.write([]byte{IAC, DONT, opt})
				}
			case WONT:
				opt, _ := tc.br.ReadByte()
				tc.write([]byte{IAC, DONT, opt})
			case DO:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return ""
				}
				if opt == OPT_ECHO || opt == OPT_SGA {
					tc.write([]byte{IAC, WILL, opt})
				} else {
					tc.write([]byte{IAC, WONT, opt})
				}
			case DONT:
				opt, _ := tc.br.ReadByte()
				tc.write([]byte{IAC, WONT, opt})
			case SB:
				// Skip subnegotiation until SE
				for {
					b2, err := tc.br.ReadByte()
					if err != nil {
						return ""
					}
					if b2 == IAC {
						b3, err := tc.br.ReadByte()
						if err != nil {
							return ""
						}
						if b3 == SE {
							break
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
			return string(line)
		}
		if b == '\n' {
			if len(line) > maxInputLen {
				slog.Warn("telnet: input truncated", "length", len(line), "max", maxInputLen)
				line = line[:maxInputLen]
			}
			return string(line)
		}
		line = append(line, b)
	}
}

// write sends bytes with a simple mutex to avoid interleaving.
func (tc *telnetConn) write(data []byte) {
	tc.wmu <- struct{}{}
	_, _ = tc.Write(data)
	<-tc.wmu
}

func (tc *telnetConn) writeLine(s string) {
	tc.write([]byte(s))
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
	loginMsg, err := json.Marshal(map[string]string{
		"type": "login",
		"data": string(loginData),
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
	choiceMsg, err := json.Marshal(map[string]string{
		"type": "char_input",
		"data": string(choiceData),
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
	cmdMsg, err := json.Marshal(map[string]string{
		"type": "command",
		"data": string(cmdData),
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return s.HandleMessage(cmdMsg)
}
