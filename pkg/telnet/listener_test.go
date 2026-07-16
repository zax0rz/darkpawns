package telnet

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

func newTestManager(t *testing.T) (*session.Manager, *game.World) {
	t.Helper()
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)
	return manager, world
}

// drain reads and discards everything from c until it is closed.
func drain(c net.Conn) {
	buf := make([]byte, 4096)
	for {
		_, err := c.Read(buf)
		if err != nil {
			return
		}
	}
}

// TestHandleConnDisconnectDuringPasswordPrompt verifies that handleConn returns
// when the client disconnects instead of proceeding with an empty password.
func TestHandleConnDisconnectDuringPasswordPrompt(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	go drain(client)

	// Provide a name, then close the connection while the server is waiting
	// for a password/confirmation response.
	_, _ = client.Write([]byte("testplayer\r\n"))
	time.Sleep(50 * time.Millisecond)
	_ = client.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after disconnect")
	}
}

// TestHandleConnRejectsEmptyPassword verifies that empty new-character passwords
// are rejected and the connection handler returns instead of continuing.
func TestHandleConnRejectsEmptyPassword(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	go drain(client)

	// No DB path: name -> yes -> empty password -> empty confirmation.
	_, _ = client.Write([]byte("newplayer\r\ny\r\n\r\n\r\n"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after empty password")
	}
}

func TestEffectiveBanLevel_IPAndHostname(t *testing.T) {
	bm := game.NewBanManager()
	bm.AddBan("192.0.2.10", game.BanNew, "test")
	bm.AddBan("evil.example.com", game.BanAll, "test")

	origLookup := lookupAddr
	defer func() { lookupAddr = origLookup }()
	lookupAddr = func(addr string) ([]string, error) {
		if addr == "192.0.2.10" {
			return []string{"evil.example.com."}, nil
		}
		return nil, nil
	}

	// IP-only check would return BanNew; hostname check elevates to BanAll.
	if got := effectiveBanLevel("192.0.2.10", bm); got != game.BanAll {
		t.Fatalf("effectiveBanLevel = %d, want BanAll (%d)", got, game.BanAll)
	}
}

func TestEffectiveBanLevel_TimeoutFallsBackToIP(t *testing.T) {
	bm := game.NewBanManager()
	bm.AddBan("198.51.100.5", game.BanAll, "test")

	origLookup := lookupAddr
	origTimeout := dnsLookupTimeout
	defer func() {
		lookupAddr = origLookup
		dnsLookupTimeout = origTimeout
	}()

	lookupAddr = func(addr string) ([]string, error) {
		time.Sleep(100 * time.Millisecond)
		return []string{"slow.example.com."}, nil
	}
	dnsLookupTimeout = 10 * time.Millisecond

	// Even though DNS times out, the raw IP is still banned.
	if got := effectiveBanLevel("198.51.100.5", bm); got != game.BanAll {
		t.Fatalf("effectiveBanLevel = %d, want BanAll (%d)", got, game.BanAll)
	}
}

func TestEffectiveBanLevel_NoMatch(t *testing.T) {
	bm := game.NewBanManager()

	origLookup := lookupAddr
	defer func() { lookupAddr = origLookup }()
	lookupAddr = func(addr string) ([]string, error) {
		return []string{"clean.example.com."}, nil
	}

	if got := effectiveBanLevel("203.0.113.7", bm); got != game.BanNot {
		t.Fatalf("effectiveBanLevel = %d, want BanNot (%d)", got, game.BanNot)
	}
}

// TestReadLineCapsBufferWithoutNewline verifies that a client sending more
// than maxInputLen bytes without a newline cannot grow the input buffer
// unboundedly (DP-622).
func TestReadLineCapsBufferWithoutNewline(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	data := make([]byte, maxInputLen+10)
	for i := range data {
		data[i] = 'a'
	}
	data = append(data, '\n')
	go func() {
		_, _ = client.Write(data)
		go drain(client)
	}()

	line, ok := tc.readLine()
	if !ok {
		t.Fatal("readLine returned false, want true")
	}
	if len(line) != maxInputLen {
		t.Errorf("readLine length = %d, want %d", len(line), maxInputLen)
	}
}

// TestReadLinePreAuthIdleTimeout confirms that readLinePreAuth drops an idle
// pre-auth connection (no input) after loginIdleTimeout and writes the goodbye
// line (DP-912).
func TestReadLinePreAuthIdleTimeout(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	// Shorten the idle timeout for a deterministic, fast test.
	prev := loginIdleTimeout
	loginIdleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { loginIdleTimeout = prev })

	// net.Pipe writes block until read; drain the client side in a goroutine so
	// readLinePreAuth's goodbye write doesn't deadlock.
	got := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, client)
		got <- buf.Bytes()
	}()

	start := time.Now()
	_, ok := tc.readLinePreAuth()
	elapsed := time.Since(start)

	if ok {
		t.Fatal("readLinePreAuth returned ok=true on an idle connection, want timeout")
	}
	if elapsed > 2*time.Second {
		t.Errorf("readLinePreAuth took %v, want it to time out within ~%v", elapsed, loginIdleTimeout)
	}

	// Closing the server side (via defer) lets the drain goroutine finish.
	server.Close()
	client.Close()
	select {
	case b := <-got:
		if !bytes.Contains(b, []byte("Idle timeout")) {
			t.Errorf("expected goodbye containing 'Idle timeout', got %q", b)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for goodbye bytes on client side")
	}
}

// TestReadLinePreAuthSuccess confirms a prompt read still returns the line when
// input arrives before the idle timeout.
func TestReadLinePreAuthSuccess(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}
	loginIdleTimeout = 5 * time.Second

	go func() {
		_, _ = client.Write([]byte("bob\r\n"))
		go drain(client)
	}()

	line, ok := tc.readLinePreAuth()
	if !ok {
		t.Fatal("readLinePreAuth returned false, want true")
	}
	if line != "bob" {
		t.Errorf("readLinePreAuth line = %q, want %q", line, "bob")
	}
}

// TestReadLineCapsIACEscapedBytes verifies that a stream of IAC IAC escape
// sequences (each appending a literal 0xFF byte) is also capped at
// maxInputLen (DP-622).
func TestReadLineCapsIACEscapedBytes(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	data := make([]byte, 0, (maxInputLen+10)*2+1)
	for i := 0; i < maxInputLen+10; i++ {
		data = append(data, IAC, IAC)
	}
	data = append(data, '\n')

	go func() {
		_, _ = client.Write(data)
		go drain(client)
	}()

	line, ok := tc.readLine()
	if !ok {
		t.Fatal("readLine returned false, want true")
	}
	if len(line) != maxInputLen {
		t.Errorf("readLine length = %d, want %d", len(line), maxInputLen)
	}
}

// readAllAvailable reads from c until the deadline expires without new data.
func readAllAvailable(t *testing.T, c net.Conn, timeout time.Duration) []byte {
	t.Helper()
	var out bytes.Buffer
	buf := make([]byte, 256)
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return out.Bytes()
		default:
		}
		_ = c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		n, err := c.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return out.Bytes()
			}
			return out.Bytes()
		}
		out.Write(buf[:n])
	}
}

// TestInitialNegotiationOffersMCCP2 verifies that the server offers MCCP2
// compression during initial telnet negotiation (DP-598).
func TestInitialNegotiationOffersMCCP2(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	data := readAllAvailable(t, client, 200*time.Millisecond)
	found := bytes.Contains(data, []byte{IAC, WILL, OPT_COMPRESS2})
	_ = client.Close()
	<-done

	if !found {
		t.Errorf("initial negotiation did not include IAC WILL COMPRESS2, got %v", data)
	}
}

// TestMCCP2CompressionStartsOnDO verifies that replying IAC DO COMPRESS2
// switches the stream to zlib-compressed output (DP-598).
func TestMCCP2CompressionStartsOnDO(t *testing.T) {
	// Use a buffered connection to avoid pipe write blocking.
	client, server := net.Pipe()
	defer client.Close()

	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	// Continuous reader drains everything the server sends.
	readCh := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		b := make([]byte, 1024)
		for {
			n, err := client.Read(b)
			if err != nil {
				break
			}
			buf.Write(b[:n])
		}
		readCh <- buf.Bytes()
	}()

	// Run readLine in a goroutine; feed it DO COMPRESS2 then a newline so it
	// returns after enabling compression.
	readLineDone := make(chan struct{})
	go func() {
		defer close(readLineDone)
		_, _ = tc.readLine()
	}()

	// Send DO COMPRESS2 from the client side.
	_, _ = client.Write([]byte{IAC, DO, OPT_COMPRESS2, '\n'})
	<-readLineDone

	// Send some additional data through the compressor.
	tc.writeLine("hello compressed world")
	_ = tc.compressWriter.Close()
	_ = server.Close()

	readBuf := <-readCh

	sentinel := []byte{IAC, SB, OPT_COMPRESS2, IAC, SE}
	if !bytes.Contains(readBuf, sentinel) {
		t.Fatalf("response did not contain COMPRESS_START sentinel: %v", readBuf)
	}
	idx := bytes.Index(readBuf, sentinel)
	compressed := readBuf[idx+len(sentinel):]

	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("zlib.NewReader: %v", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("zlib decode: %v", err)
	}

	if !bytes.Contains(decompressed, []byte("hello compressed world")) {
		t.Errorf("decompressed stream did not contain expected text, got %q", decompressed)
	}
}

// TestMCCP2DONTKeepsPlaintext verifies that IAC DONT COMPRESS2 leaves the
// stream uncompressed (DP-598).
func TestMCCP2DONTKeepsPlaintext(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	// Continuously drain server output so its writes do not block.
	readCh := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		b := make([]byte, 1024)
		for {
			n, err := client.Read(b)
			if err != nil {
				break
			}
			buf.Write(b[:n])
		}
		readCh <- buf.Bytes()
	}()

	// Decline compression.
	_, _ = client.Write([]byte{IAC, DONT, OPT_COMPRESS2})
	// Provide a name so the server sends the greeting prompt.
	_, _ = client.Write([]byte("testplayer\r\n"))

	// Give the server time to respond, then close and collect output.
	time.Sleep(200 * time.Millisecond)
	_ = client.Close()
	readBuf := <-readCh
	<-done

	if bytes.Contains(readBuf, []byte{IAC, SB, OPT_COMPRESS2, IAC, SE}) {
		t.Error("server sent COMPRESS_START after DONT COMPRESS2")
	}
	// Sanity check: the plaintext greeting prompt should be present.
	if !bytes.Contains(readBuf, []byte("By what name")) {
		t.Errorf("plaintext greeting missing from response: %q", readBuf)
	}
}

// TestConfigurableConnectionLimits verifies that TELNET_MAX_CONNS and
// TELNET_MAX_CONNS_PER_IP are read from the environment (DP-556).
func TestConfigurableConnectionLimits(t *testing.T) {
	origMaxTotal := maxTotalConns
	origMaxPerIP := maxConnsPerIP
	defer func() {
		maxTotalConns = origMaxTotal
		maxConnsPerIP = origMaxPerIP
	}()

	maxTotalConns = 5
	maxConnsPerIP = 2

	manager, world := newTestManager(t)
	defer world.StopAITicker()

	origListen := listenTCP
	defer func() { listenTCP = origListen }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	listenTCP = func(network, address string) (net.Listener, error) {
		return ln, nil
	}

	if err := Listen(ln.Addr().(*net.TCPAddr).Port, manager); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer Stop()

	time.Sleep(50 * time.Millisecond)

	addr := ln.Addr().String()
	conns := make([]net.Conn, 0, 5)
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	// First two connections from the same IP should be accepted.
	for i := 0; i < 2; i++ {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		conns = append(conns, c)
	}

	// Third connection from the same IP should be rejected.
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial rejected: %v", err)
	}
	defer c.Close()

	// The server closes the rejected connection; reading should return EOF.
	buf := make([]byte, 1)
	_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = c.Read(buf)
	if err == nil {
		t.Error("expected third same-IP connection to be rejected/closed")
	}
}

// TestTelnetLinkdeadReaperCleanup verifies that inbound telnet traffic updates
// the session's shared lastActive timestamp and that the linkdead reaper can
// extract an idle telnet session (DP-928).
func TestTelnetLinkdeadReaperCleanup(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1, Name: "Limbo", Zone: 0},
			{VNum: 3, Name: "A Totally Empty Room", Zone: 0},
			{VNum: game.MortalStartRoom, Name: "The Adventurers Guild", Zone: 80},
		},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)

	client, server := net.Pipe()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	go drain(client)

	// Wait for the banner and "By what name" prompt.
	time.Sleep(100 * time.Millisecond)

	// Use a non-generic guest name so we can look up the session deterministically.
	playerName := "guest_telnet_reaper"
	_, _ = client.Write([]byte(playerName + "\r\n"))

	// Wait for guest login to complete and enter the input loop.
	time.Sleep(300 * time.Millisecond)

	s, ok := manager.GetSession(playerName)
	if !ok {
		t.Fatal("telnet guest session not registered")
	}
	if !s.IsAuthenticated() {
		t.Fatal("telnet guest session not authenticated")
	}

	// Simulate idle linkdead: set last active to 10 minutes ago.
	s.SetLastActiveForTest(time.Now().Add(-10 * time.Minute).UnixNano())

	manager.ReapLinkdeadSessions()

	if _, ok := manager.GetSession(playerName); ok {
		t.Error("idle telnet session should have been reaped")
	}

	// Closing the client lets the handleConn goroutine exit cleanly.
	_ = client.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after client close")
	}
}

func TestObservationTelnetRenderingHasNoStateBoxAndGatesVNum(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum:        1234,
		Name:        "Viewer-Aware Hall",
		Description: "No transport-specific decoration belongs here.",
		Flags:       []string{"0", "0", "0", "0"},
		Sector:      0,
	}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	state := session.ServerMessage{Type: session.MsgState, Data: map[string]interface{}{
		"player": map[string]interface{}{"name": "Viewer", "level": 1, "health": 10, "max_health": 10},
		"room":   map[string]interface{}{"vnum": 1234, "name": "Viewer-Aware Hall"},
	}}
	if got := formatState(state); got != "" {
		t.Fatalf("retired state renderer produced telnet text %q", got)
	}

	mortal := game.NewPlayer(1, "Mortal", 1234)
	mortalResult := world.DoLookRoom(mortal, true)
	mortalText := observationFormats(mortalResult)
	if strings.Contains(mortalText, "---") || strings.Contains(mortalText, "Lvl ") || strings.Contains(mortalText, "[ 1234]") {
		t.Fatalf("mortal room text leaked state decoration or vnum: %q", mortalText)
	}

	roomflagsViewer := game.NewPlayer(2, "Builder", 1234)
	roomflagsViewer.SetLevel(game.LVL_IMMORT)
	roomflagsViewer.SetRoomFlags(true)
	roomflagsText := observationFormats(world.DoLookRoom(roomflagsViewer, true))
	if !strings.Contains(roomflagsText, "[ 1234] Viewer-Aware Hall") {
		t.Fatalf("roomflags viewer did not receive vnum: %q", roomflagsText)
	}
}

func observationFormats(result game.ObservationResult) string {
	var out strings.Builder
	for _, message := range result.Messages {
		out.WriteString(message.Format)
		out.WriteByte('\n')
	}
	return out.String()
}
