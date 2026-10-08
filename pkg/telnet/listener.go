// Package telnet provides a raw TCP telnet listener for the Dark Pawns MUD.
package telnet

import (
	"bufio"
	"bytes"
	"compress/zlib"
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

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/grapevine"
	"github.com/zax0rz/darkpawns/pkg/metrics"
	"github.com/zax0rz/darkpawns/pkg/session"
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
	EOR  byte = 239

	OPT_ECHO      byte = 1
	OPT_SGA       byte = 3
	OPT_EOR       byte = 25
	OPT_MSSP      byte = 70
	OPT_GMCP      byte = 201
	OPT_COMPRESS2 byte = 86

	MSSP_VAR byte = 1
	MSSP_VAL byte = 2
)

// maxConnsPerIP caps simultaneous telnet connections from one address
// (TELNET_MAX_CONNS_PER_IP). The C server had no such cap; it is flood
// protection, sized like webSocketMaxConnsPerIP for two players sharing a
// home connection at the three-character multiplay allowance, plus room to
// reconnect. At 3 it equalled one player's allowance, so a second player in
// the house (or on a carrier that puts many phones behind one address) was
// refused.
var (
	maxConnsPerIP    = 8
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

var (
	connMu   sync.Mutex
	listener net.Listener
	// tlsListener is the optional TLS telnet listener (ListenTLS).
	tlsListener net.Listener
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

	serve(ln, manager)
	return nil
}

// serve runs the accept loop for ln in the background. Plain and TLS
// listeners share it, and with it the connection limits and ban checks: a
// TLS connection is the same telnet session once its handshake is done.
func serve(ln net.Listener, manager *session.Manager) {
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
			//
			// C's new_descriptor computed d->host and ran all three isbanned()
			// checks synchronously in the accept path (src/comm.c:1518-1569).
			// With nameserver_is_slow — C's shipped default (src/config.c:206) —
			// no lookup happens at all, so the whole check can run here;
			// otherwise only the reverse-DNS part moves into the
			// per-connection goroutine, so a slow resolver cannot stall the
			// accept queue.
			banManager := manager.GetBanManager()
			identity := session.ConnectionIdentity{}
			if game.NameserverIsSlow() {
				identity = identifyConnection(remoteIP, banManager)
				if identity.Level == game.BanAll {
					releaseConnectionSlot(remoteIP)
					_ = conn.Close() //nolint:errcheck // best-effort cleanup
					continue
				}
			}

			go func(ip string, precomputed session.ConnectionIdentity, prechecked bool) {
				if !prechecked {
					precomputed = identifyConnection(ip, banManager)
				}
				reject := precomputed.Level == game.BanAll
				if !reject && !completeTLSHandshake(conn) {
					// A banned address is dropped before any TLS work; everyone
					// else must finish the handshake before the banner is sent.
					reject = true
				}
				if reject {
					_ = conn.Close() //nolint:errcheck // best-effort cleanup
					releaseConnectionSlot(ip)
					return
				}
				handleConn(conn, manager, precomputed)
				releaseConnectionSlot(ip)
			}(remoteIP, identity, game.NameserverIsSlow())
		}
	}()
}

// releaseConnectionSlot drops one connection from the accept-loop tally. A
// refused connection never reaches handleConn, so it must release its slot
// itself.
func releaseConnectionSlot(ip string) {
	connMu.Lock()
	connCount--
	connPerIP[ip]--
	if connPerIP[ip] <= 0 {
		delete(connPerIP, ip)
	}
	connMu.Unlock()
}

func ipFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// identifyConnection computes C's new_descriptor view of an accepted
// connection: the string it stores in d->host and the ban level its three
// isbanned() calls produce (src/comm.c:1518-1569).
//
// With nameserver_is_slow — C's shipped default (src/config.c:206) — C never
// resolves (src/comm.c:1527-1529), so no lookup runs and d->host is the padded
// quad. Otherwise it resolves, and a failed lookup logs C's producer
// (src/comm.c:1552-1555) before the ban check: "DNS lookup failed on %s." at
// CMP / LVL_GOD / file TRUE.
func identifyConnection(remoteIP string, banManager *game.BanManager) session.ConnectionIdentity {
	padded := session.CConnectionHost(remoteIP)
	if game.NameserverIsSlow() {
		// C's wildhost and double_wild are both defined on this branch
		// (src/comm.c:1536-1548).
		return session.ConnectionIdentity{
			Host:  padded,
			Level: session.AcceptBanLevel(banManager, padded, false),
		}
	}

	if name, err := reverseLookup(remoteIP); err == nil {
		// d->host is the resolved name (src/comm.c:1558-1560). C's wildhost
		// and double_wild are uninitialised stack here, so C compares only the
		// name; the port does the same (R1a; see session.AcceptBanLevel).
		return session.ConnectionIdentity{
			Host:         name,
			HostResolved: true,
			Level:        session.AcceptBanLevel(banManager, name, true),
		}
	}

	game.MudLog(fmt.Sprintf("DNS lookup failed on %s.", padded), game.MudlogComplete, game.LVL_GOD, true) // comm.c:1554
	return session.ConnectionIdentity{
		Host:  padded,
		Level: session.AcceptBanLevel(banManager, padded, false),
	}
}

// reverseLookup resolves a connection's PTR record within dnsLookupTimeout and
// returns the canonical first name with its trailing dot removed. A timeout
// counts as a failure, exactly as a NULL gethostbyaddr result does in C.
func reverseLookup(remoteIP string) (string, error) {
	lookup := lookupAddr // capture so tests can restore package var safely
	type result struct {
		names []string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		names, err := lookup(remoteIP)
		done <- result{names: names, err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			return "", res.err
		}
		if len(res.names) == 0 {
			return "", fmt.Errorf("no PTR record for %s", remoteIP)
		}
		// C stores from->h_name, the canonical name; Go returns the PTR
		// targets in order with a trailing dot.
		return strings.TrimSuffix(res.names[0], "."), nil
	case <-time.After(dnsLookupTimeout):
		return "", fmt.Errorf("reverse lookup for %s timed out", remoteIP)
	}
}

type telnetConn struct {
	net.Conn
	br      *bufio.Reader
	wmu     chan struct{} // buffered(1) acts as a write mutex
	manager *session.Manager
	hasGMCP atomic.Bool
	// hasEOR is set once the client agrees (DO EOR) to have prompts marked
	// with IAC EOR, which is how Mudlet and similar clients tell a prompt
	// from a partial line. Clients that never agree see no EOR bytes.
	hasEOR         atomic.Bool
	sess           *session.Session
	compressWriter *zlib.Writer
}

func handleConn(rawConn net.Conn, manager *session.Manager, identity session.ConnectionIdentity) {
	tc := &telnetConn{
		Conn:    rawConn,
		br:      bufio.NewReader(rawConn),
		wmu:     make(chan struct{}, 1),
		manager: manager,
	}
	// Paired here rather than at Accept: the reject paths above close the
	// connection and return without ever reaching this function, so counting at
	// Accept would leak an increment into connections_active every time a ban
	// fired. A gauge that drifts upward forever is worse than one that does not
	// count refused connections.
	metrics.ConnectionOpened()
	defer func() {
		metrics.ConnectionClosed()
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
	tc.write([]byte{IAC, WILL, OPT_EOR})

	s := manager.NewSession()
	tc.sess = s
	s.SetCloseFunc(func() { _ = rawConn.Close() })
	remoteIP := ipFromAddr(remoteAddr)
	s.SetRemoteIP(remoteIP)
	// C's d->host for this descriptor (src/comm.c:1534-1563). Every admission
	// producer and the nanny's own ban checks print it.
	s.SetMudHost(identity.Host)
	// C's nanny tests isbanned(d->host) at each boundary (src/interpreter.c:1822,
	// :1896) — the d->host string only, not the raw IP.
	if identity.Host != "" {
		s.SetBanHosts([]string{identity.Host})
	}
	if identity.Level != game.BanNot {
		s.SetBanLevel(identity.Level)
	}

	// Welcome + name prompt
	tc.write(escapeIAC([]byte(session.TerminalGreeting())))

	// Start the output writer before the name prompt is answered. Login output
	// must reach the client as it is generated (DP-591), and so must what a
	// session queues before login: GMCP negotiation happens while the player
	// is still at the name prompt, and Mudlet installs the offered package
	// (Client.GUI) from that moment. Until the name is in, nothing but GMCP is
	// queued, so the name dialogue's own bytes are unaffected. Every exit
	// before login stops the writer.
	done := make(chan struct{})
	go func() {
		defer close(done)
		writeLoop(tc, s)
	}()
	stopWriter := func() {
		// A client that stops reading cannot block writeLoop forever and leak
		// this goroutine/file descriptor.
		_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		s.CloseSend()
		<-done
	}

	// The shared terminal owns the name dialogue and every line after it
	// (session.TerminalLine). Telnet reads lines with the login reader, which
	// applies C's check_idle_passwords clock (comm.c:2170-2187), until the
	// character is in the world; then with the playing reader, which applies
	// no transport deadline at all. C has none for a playing descriptor —
	// idling is check_idling's business (limits.c:419-454) — so the
	// five-minute deadline this loop used to arm once a name was accepted
	// dropped idle players and immortals alike (DP-1385, R4).
	for {
		var line string
		var ok bool
		if s.IsPlaying() {
			line, ok = tc.readLine()
		} else {
			line, ok = tc.readLinePreAuth()
		}
		if !ok {
			// EOF or connection error: the client hung up.
			if !s.TerminalNamed() {
				stopWriter()
				return
			}
			break
		}
		if !s.TerminalLine(line) {
			if !s.TerminalNamed() || !s.IsAuthenticated() {
				stopWriter()
				return
			}
			break
		}
	}

	// Cleanup
	if !s.Manager().HandleTransportDisconnect(s) {
		s.Manager().UnregisterSession(s)
		s.CloseSend()
	}
	// A successful quit queues its goodbye immediately before Unregister closes
	// the send channel. For an unexpected EOF, DetachTransport ends writeLoop
	// while retaining the open send channel for the linkdead session.
	_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	<-done
	slog.Info("Telnet disconnect", "remote_addr", remoteAddr, "player", s.PlayerName())
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
		f, ok := session.RenderTerminalFrame(msg)
		if !ok {
			continue
		}
		f = s.TrackPrompt(f)
		switch f.Kind {
		case session.FrameText:
			tc.write(escapeIAC([]byte(f.Text)))
		case session.FramePrompt:
			tc.writePrompt(f.Text)
		case session.FrameEntryPrompt:
			if f.Secret {
				tc.write([]byte{IAC, WILL, OPT_ECHO})
			} else {
				tc.write([]byte{IAC, WONT, OPT_ECHO})
			}
			if f.Text != "" {
				tc.write(tc.markPrompt(escapeIAC([]byte(f.Text))))
			}
		case session.FrameGMCP:
			if tc.hasGMCP.Load() {
				tc.write(buildGMCPFrameRaw(f.GMCPPackage, f.GMCPPayload))
			}
		}
	}
}

// maxInputLen caps a single input line; longer input is truncated and logged.
const maxInputLen = 1024

// readLine reads a line for a playing descriptor. C attaches no transport read
// deadline to one — a silent player is check_idling's business
// (src/limits.c:419-454) — so any deadline the login reader armed is cleared
// before the read. Dropping an idle player on a five-minute transport clock
// was invented behaviour (DP-1385, R4).
func (tc *telnetConn) readLine() (string, bool) {
	_ = tc.SetReadDeadline(time.Time{})
	return tc.readLineBody()
}

// readLineBody performs the actual line read under whatever read deadline the
// caller has set. It handles IAC negotiation and responds appropriately. The
// bool return is false on EOF/error and true otherwise. A blank line (the user
// simply pressing Return) returns ("", true) — distinct from EOF, which returns
// ("", false). Callers must use the bool to decide whether to disconnect;
// treating "" alone as EOF drops players who press Enter and breaks the "PRESS
// RETURN" step of character creation. Input exceeding maxInputLen bytes is
// truncated and logged.
func (tc *telnetConn) readLineBody() (string, bool) {
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
					tc.enableGMCP()
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
					// Answer only a DO the server has not already agreed to: the
					// WILL sent at connect is the offer, and re-asserting it on the
					// client's DO would start a negotiation loop.
					if !tc.hasGMCP.Load() {
						tc.enableGMCP()
					}
				case OPT_EOR:
					tc.hasEOR.Store(true)
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
				if opt == OPT_EOR {
					tc.hasEOR.Store(false)
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
	line, ok := tc.readLineBody()
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
	tc.write(escapeIAC([]byte(session.NormalizeCRLF(s))))
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
	// PLAYERS counts playing characters, not raw sessions: SessionCount
	// is len(manager.sessions), which includes pre-auth connections (a
	// crawler would count its own login screen). The count uses the
	// same mortal-visibility rule as the Grapevine heartbeat
	// (World.VisiblePlayersForMortal), so a wizinvis immortal is not
	// revealed to crawlers either.
	playerCount := tc.manager.SessionCount()
	if world := tc.manager.World(); world != nil {
		playerCount = len(world.VisiblePlayersForMortal())
	}
	writeField("PLAYERS", fmt.Sprintf("%d", playerCount))
	writeField("UPTIME", fmt.Sprintf("%d", startTime.Unix()))
	writeField("CODEBASE", "CircleMUD 3.0 (Go port)")
	writeField("FAMILY", "DikuMUD")
	// CREATED: website/data/history.toml is the source of truth
	// (founded_year = 1994).
	writeField("CREATED", "1994")
	writeField("WEBSITE", "darkpawns.org")
	writeField("PORT", "7777")
	// TLS (with the certificate's HOSTNAME) is what Mudlet reads to offer a
	// plaintext player the encrypted port; nothing is printed to the player.
	if advert := tlsAdvert.Load(); advert != nil {
		writeField("TLS", strconv.Itoa(advert.port))
		if advert.hostname != "" {
			writeField("HOSTNAME", advert.hostname)
		}
	}
	writeField("ANSI", "1")
	writeField("GMCP", "1")
	writeField("MCCP", "1")
	writeField("LANGUAGE", "English")
	writeField("LOCATION", "US")
	writeField("GENRE", "Fantasy")
	writeField("GAMEPLAY", "Hack and Slash")
	writeField("STATUS", "Live")
	// INTERMUD is advertised only when the Grapevine integration can
	// actually come up (credentials present and mode not off).
	if grapevine.ModeFromEnv() != grapevine.ModeOff {
		writeField("INTERMUD", "Grapevine")
	}
	// World-shape counts are computed from the loaded world, never
	// hard-coded: zones/rooms/prototypes from the parser output held by
	// game.World, help entries from World.HelpTable.
	if world := tc.manager.World(); world != nil {
		writeField("AREAS", strconv.Itoa(world.GetZoneCount()))
		writeField("ROOMS", strconv.Itoa(world.GetRoomCount()))
		writeField("MOBILES", strconv.Itoa(world.GetMobPrototypeCount()))
		writeField("OBJECTS", strconv.Itoa(world.GetObjPrototypeCount()))
		writeField("HELPFILES", strconv.Itoa(world.GetHelpFileCount()))
	}
	// LEVELS is the implementor level cap (game.LVL_IMPL); CLASSES and
	// RACES count the definition tables in pkg/game/character.go.
	writeField("LEVELS", strconv.Itoa(game.LVL_IMPL))
	writeField("CLASSES", strconv.Itoa(len(game.ClassNames)))
	writeField("RACES", strconv.Itoa(len(game.RaceNames)))
	// UTF-8 is "0": local input is ASCII-filtered (#1828), faithful to C.
	writeField("UTF-8", "0")
	writeField("MSDP", "0")
	writeField("MXP", "0")
	writeField("MSP", "0")
	writeField("PAY TO PLAY", "0")
	writeField("PAY FOR PERKS", "0")

	payload = append(payload, IAC, SE)

	tc.writeLocked(payload)
}

// buildGMCPFrameRaw frames an already-encoded GMCP message: IAC SB GMCP
// "<package> <json>" IAC SE, with any 0xFF data byte doubled as telnet
// requires. An empty payload sends the package name alone.
func buildGMCPFrameRaw(pkg, payload string) []byte {
	frame := make([]byte, 0, 3+len(pkg)+1+len(payload)+2)
	frame = append(frame, IAC, SB, OPT_GMCP)
	frame = appendIACEscaped(frame, pkg)
	if payload != "" {
		frame = append(frame, ' ')
		frame = appendIACEscaped(frame, payload)
	}
	frame = append(frame, IAC, SE)
	return frame
}

func appendIACEscaped(dst []byte, text string) []byte {
	for i := 0; i < len(text); i++ {
		if text[i] == IAC {
			dst = append(dst, IAC)
		}
		dst = append(dst, text[i])
	}
	return dst
}

func (tc *telnetConn) handleIncomingGMCP(payload []byte) {
	if len(payload) == 0 {
		return
	}
	msgName, jsonStr, _ := strings.Cut(string(payload), " ")
	slog.Debug("telnet: received GMCP", "message", msgName, "payload", jsonStr)
	if tc.sess != nil {
		tc.sess.HandleGMCP(msgName, strings.TrimSpace(jsonStr))
	}
}

// enableGMCP records that the client agreed to GMCP and turns it on for the
// session. GMCP is framing only: it never changes what text the session is
// sent (see session.EnableGMCP).
func (tc *telnetConn) enableGMCP() {
	tc.hasGMCP.Store(true)
	if tc.sess != nil {
		tc.sess.EnableGMCP()
	}
}

// writePrompt writes the command prompt, marked with IAC EOR for clients that
// asked for prompt marking.
func (tc *telnetConn) writePrompt(prompt string) {
	// RenderTerminalFrame has already normalized ordinary prompts. A pager
	// prompt intentionally carries C's lone leading carriage return.
	tc.write(tc.markPrompt(escapeIAC([]byte(prompt))))
}

// escapeIAC doubles every 0xFF byte in game text, as telnet requires for a
// data byte equal to IAC (RFC 854). Unescaped, a stored 0xFF made the client
// parse the following game bytes as a telnet command (REF-1). Only text is
// escaped: negotiation sequences and GMCP frames (already escaped by
// buildGMCPFrameRaw) are written as-is. Shipped world and help text contain
// no 0xFF, and since #1828 player input cannot introduce one.
func escapeIAC(data []byte) []byte {
	n := bytes.Count(data, []byte{IAC})
	if n == 0 {
		return data
	}
	out := make([]byte, 0, len(data)+n)
	for _, b := range data {
		out = append(out, b)
		if b == IAC {
			out = append(out, IAC)
		}
	}
	return out
}

// markPrompt appends IAC EOR to prompt bytes when the client negotiated EOR.
// The marker is a telnet command, so it never reaches the player's screen.
func (tc *telnetConn) markPrompt(prompt []byte) []byte {
	if !tc.hasEOR.Load() || len(prompt) == 0 {
		return prompt
	}
	return append(prompt, IAC, EOR)
}

// Stop closes the TCP telnet listener and the TLS listener, if any.
func Stop() {
	connMu.Lock()
	defer connMu.Unlock()
	if listener != nil {
		_ = listener.Close()
		listener = nil
	}
	if tlsListener != nil {
		_ = tlsListener.Close()
		tlsListener = nil
	}
	tlsAdvert.Store(nil)
}
