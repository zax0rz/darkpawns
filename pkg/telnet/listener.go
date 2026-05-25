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
	OPT_MSSP byte = 70
	OPT_GMCP byte = 201

	MSSP_VAR byte = 1
	MSSP_VAL byte = 2

	maxConnsPerIP = 3
	maxTotalConns = 200
)

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
	ln, err := net.Listen("tcp", addr)
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
	br      *bufio.Reader
	wmu     chan struct{} // buffered(1) acts as a write mutex
	manager *session.Manager
	hasGMCP bool
	sess    *session.Session
}

func handleConn(rawConn net.Conn, manager *session.Manager) {
	tc := &telnetConn{
		Conn:    rawConn,
		br:      bufio.NewReader(rawConn),
		wmu:     make(chan struct{}, 1),
		manager: manager,
		hasGMCP: false,
	}
	defer func() { _ = rawConn.Close() }()

	remoteAddr := rawConn.RemoteAddr().String()
	slog.Info("Telnet connect", "remote_addr", remoteAddr)

	// Send initial negotiation: WONT echo (so client local echo is ON by default)
	tc.write([]byte{IAC, WONT, OPT_ECHO})
	tc.write([]byte{IAC, WILL, OPT_SGA})
	tc.write([]byte{IAC, WILL, OPT_MSSP})
	tc.write([]byte{IAC, WILL, OPT_GMCP})

	s := manager.NewSession()
	tc.sess = s
	remoteIP := ipFromAddr(remoteAddr)
	s.SetRemoteIP(remoteIP)

	// Welcome + prompt
	tc.writeLine(greetingsLogo)
	tc.writeLine("By what name do you wish to be known? ")

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
			if tc.hasGMCP {
				if ed, ok := sm.Data.(map[string]interface{}); ok {
					// Build all GMCP frames and send in a single write to minimize syscalls.
					var buf []byte

					vitals := make(map[string]interface{})
					if hp, ok := ed["HEALTH"]; ok { vitals["hp"] = hp }
					if maxhp, ok := ed["MAX_HEALTH"]; ok { vitals["maxhp"] = maxhp }
					if mp, ok := ed["MANA"]; ok { vitals["mp"] = mp }
					if maxmp, ok := ed["MAX_MANA"]; ok { vitals["maxmp"] = maxmp }
					if mv, ok := ed["MOVE"]; ok { vitals["mv"] = mv }
					if maxmv, ok := ed["MAX_MOVE"]; ok { vitals["maxmv"] = maxmv }
					if len(vitals) > 0 {
						buf = append(buf, buildGMCPFrame("Char.Vitals", vitals)...)
					}

					status := make(map[string]interface{})
					if lvl, ok := ed["LEVEL"]; ok { status["level"] = lvl }
					if gold, ok := ed["GOLD"]; ok { status["gold"] = gold }
					if exp, ok := ed["EXP"]; ok { status["exp"] = exp }
					if len(status) > 0 {
						buf = append(buf, buildGMCPFrame("Char.Status", status)...)
					}

					room := make(map[string]interface{})
					if num, ok := ed["ROOM_VNUM"]; ok { room["num"] = num }
					if name, ok := ed["ROOM_NAME"]; ok { room["name"] = name }
					if exits, ok := ed["ROOM_EXITS"]; ok { room["exits"] = exits }
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
				// Respond: DO for ECHO/SGA/GMCP, DONT for everything else
				if opt == OPT_ECHO || opt == OPT_SGA {
					tc.write([]byte{IAC, DO, opt})
				} else if opt == OPT_GMCP {
					tc.write([]byte{IAC, DO, OPT_GMCP})
					tc.hasGMCP = true
					if tc.sess != nil {
						tc.sess.SetWantsStructuredData(true)
					}
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
				} else if opt == OPT_MSSP {
					tc.write([]byte{IAC, WILL, OPT_MSSP})
					tc.sendMSSP()
				} else if opt == OPT_GMCP {
					tc.write([]byte{IAC, WILL, OPT_GMCP})
					tc.hasGMCP = true
					if tc.sess != nil {
						tc.sess.SetWantsStructuredData(true)
					}
				} else {
					tc.write([]byte{IAC, WONT, opt})
				}
			case DONT:
				opt, _ := tc.br.ReadByte()
				tc.write([]byte{IAC, WONT, opt})
			case SB:
				opt, err := tc.br.ReadByte()
				if err != nil {
					return ""
				}
				const maxSubnegLen = 4096
				if opt == OPT_GMCP {
					var subPayload []byte
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
							if b3 == IAC {
								if len(subPayload) >= maxSubnegLen {
									slog.Warn("telnet: GMCP subnegotiation payload exceeded limit, closing connection")
									return ""
								}
								subPayload = append(subPayload, IAC)
								continue
							}
						}
						if len(subPayload) >= maxSubnegLen {
							slog.Warn("telnet: GMCP subnegotiation payload exceeded limit, closing connection")
							return ""
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
							return ""
						}
						skipCount++
						if skipCount > maxSubnegLen {
							slog.Warn("telnet: subnegotiation skip exceeded limit, closing connection")
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

	_, _ = tc.Write(payload)
}

func (tc *telnetConn) sendGMCP(pkg string, data interface{}) {
	frame := buildGMCPFrame(pkg, data)
	if frame != nil {
		tc.write(frame)
	}
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

// Stop closes the TCP telnet listener.
func Stop() {
	connMu.Lock()
	defer connMu.Unlock()
	if listener != nil {
		_ = listener.Close()
		listener = nil
	}
}
