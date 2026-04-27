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
// #nosec G104
				conn.Close()
				slog.Warn("Telnet: max total connections reached, rejecting", "remote_addr", conn.RemoteAddr())
				continue
			}
			if connPerIP[remoteIP] >= maxConnsPerIP {
				connMu.Unlock()
// #nosec G104
				conn.Close()
				slog.Warn("Telnet: max per-IP connections reached, rejecting", "remote_addr", conn.RemoteAddr())
				continue
			}
			connCount++
			connPerIP[remoteIP]++
			connMu.Unlock()

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
	defer rawConn.Close()

	remoteAddr := rawConn.RemoteAddr().String()
	slog.Info("Telnet connect", "remote_addr", remoteAddr)

	// Send initial negotiation
	tc.write([]byte{IAC, WILL, OPT_ECHO})
	tc.write([]byte{IAC, WILL, OPT_SGA})

	s := manager.NewSession()

	// Welcome + prompt
	tc.writeLine("\r\n  Dark Pawns\r\n\r\nEnter your name: ")

	// Read name with timeout
// #nosec G104
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

	// Send login
	if err := sendLogin(s, name); err != nil {
		tc.writeLine(fmt.Sprintf("\r\nLogin failed: %v\r\n", err))
		return
	}

// #nosec G104
	rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

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
			tc.writeLine("> ")
			continue
		}

// #nosec G104
		rawConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		parts := strings.Fields(line)
		if err := sendCommand(s, parts[0], parts[1:]); err != nil {
			tc.writeLine(fmt.Sprintf("Error: %v\r\n", err))
		}
		tc.writeLine("> ")
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
					tc.writeLine(fmt.Sprintf("%s> ", prompt))
				}
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
		b.WriteString(fmt.Sprintf("  %s", player["name"]))
		if cls, ok := player["class"].(string); ok && cls != "" {
			b.WriteString(fmt.Sprintf(" the %s", cls))
		}
		if race, ok := player["race"].(string); ok && race != "" {
			b.WriteString(fmt.Sprintf(" (%s)", race))
		}
		b.WriteString(fmt.Sprintf("  Lvl %v  HP: %v/%v\r\n",
			player["level"], player["health"], player["max_health"]))
	}

	if room, ok := d["room"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("\r\n  %s [%v]\r\n", room["name"], room["vnum"]))
		if desc, ok := room["description"].(string); ok {
			b.WriteString(fmt.Sprintf("  %s\r\n", desc))
		}
		if exits, ok := room["exits"].([]interface{}); ok && len(exits) > 0 {
			names := make([]string, len(exits))
			for i, e := range exits {
				names[i] = fmt.Sprintf("%v", e)
			}
			b.WriteString(fmt.Sprintf("  Exits: %s\r\n", strings.Join(names, ", ")))
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
// #nosec G104
				tc.br.ReadByte()
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
// #nosec G104
	tc.Conn.Write(data)
	<-tc.wmu
}

func (tc *telnetConn) writeLine(s string) {
	tc.write([]byte(s))
}

func sendLogin(s *session.Session, name string) error {
	loginData, err := json.Marshal(map[string]interface{}{
		"player_name": name,
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
