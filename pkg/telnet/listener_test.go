package telnet

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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

func cGreetingsFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/c_greetings.txt")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\n", "\r\n")
}

func TestGreetingsLogoMatchesCFixture(t *testing.T) {
	if want := cGreetingsFixture(t); greetingsLogo != want {
		t.Fatalf("greetingsLogo differs from C fixture\ngot:  %q\nwant: %q", greetingsLogo, want)
	}
}

func TestHandlePulseControlIsDPClockOnlyAndDrawNeutral(t *testing.T) {
	manager, _ := newTestManager(t)
	var pumped int
	manager.SetPulsePump(func(n int) error {
		pumped += n
		return nil
	})

	if handlePulseControl(nil, manager, "~dpclock pulse 40") {
		t.Fatal("control intercepted with DP_CLOCK unset")
	}
	if pumped != 0 {
		t.Fatalf("pumped %d pulses with DP_CLOCK unset", pumped)
	}

	t.Setenv("DP_CLOCK", "1")
	if !handlePulseControl(nil, manager, "~dpclock pulse 40") {
		t.Fatal("valid control was not intercepted")
	}
	if pumped != 40 {
		t.Fatalf("pumped %d pulses, want 40", pumped)
	}
	if handlePulseControl(nil, manager, "~dpclock pulse 0") {
		t.Fatal("invalid pulse count was intercepted")
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

// TestHandleConnRepromptsEmptyPassword verifies C's illegal-password branch
// keeps the connection open instead of disconnecting the new character.
func TestHandleConnRepromptsEmptyPassword(t *testing.T) {
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

	// No DB path: name -> confirm yes -> empty password.
	_, _ = client.Write([]byte("newplayer\r\ny\r\n\r\n"))
	time.Sleep(100 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("handleConn disconnected after an illegal password")
	default:
	}

	_ = client.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after client disconnect")
	}
}

func TestHandleConnQuitFlushesGoodbyeBeforeDisconnect(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum: game.MortalStartRoom,
		Name: "The Temple",
		Zone: 80,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)

	client, server := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	transcript := make(chan []byte, 1)
	go func() {
		output, _ := io.ReadAll(client)
		transcript <- output
	}()

	const playerName = "guest_quit_flush"
	if _, err := client.Write([]byte(playerName + "\r\n")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := manager.GetSession(playerName); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("guest session was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := client.Write([]byte("quit\r\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after quit")
	}

	visible := string(stripTelnetCommands(<-transcript))
	if !strings.Contains(visible, "Goodbye, friend.. Come back soon!\r\n") {
		t.Fatalf("quit transcript dropped goodbye: %q", visible)
	}
}

func TestNewCharacterTelnetTranscriptMatchesC(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum:        game.NewbieStartRoom,
		Name:        "A Burning Hut",
		Description: "Flames dance along the walls of the ruined hut.",
		Zone:        80,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	world.WorldPath = "../../lib/world"
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)

	client, server := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()
	t.Cleanup(func() { _ = client.Close() })

	var transcript []byte
	readUntil := func(want string) {
		t.Helper()
		_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 4096)
		for !bytes.Contains(transcript, []byte(want)) {
			n, readErr := client.Read(buf)
			if readErr != nil {
				t.Fatalf("read waiting for %q: %v\ntranscript: %q", want, readErr, stripTelnetCommands(transcript))
			}
			transcript = append(transcript, buf[:n]...)
		}
	}
	answer := func(prompt, input string) {
		t.Helper()
		readUntil(prompt)
		if _, writeErr := client.Write([]byte(input + "\r\n")); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	answer("By what name do you wish to be known? ", "!")
	answer("Invalid name, please try another.\r\nName: ", "Transcript")
	answer("Did I get that right, Transcript (Y/N)? ", "Y")
	answer("Give me a password for Transcript: ", "hunter2")
	answer("Please retype password: ", "hunter2")
	answer("Do you want ANSI color (Y/N)? ", "N")
	answer("What is your sex (M/F)? ", "M")
	answer("Race: ", "H")
	answer("Class: ", "W")
	answer("Select: ", "K")
	answer("reroll:", "Y")
	answer("*** PRESS RETURN: ", "")
	answer("Make your choice: ", "1")
	readUntil("Flames dance along the walls of the ruined hut.")
	if bytes.Contains(transcript, []byte{'\f'}) {
		t.Fatal("telnet transcript leaked C's malformed echo-on form-feed byte")
	}

	visible := string(stripTelnetCommands(transcript))
	statsPattern := regexp.MustCompile(`\r\nYour ability scores:\r\n  Str: .+ Dex: .+ Int: .+\r\n  Wis: .+ Con: .+ Cha: .+\r\n`)
	visible = statsPattern.ReplaceAllString(visible, "<ROLLED_STATS>")
	motd, err := os.ReadFile("../../lib/world/text/motd")
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := cGreetingsFixture(t) +
		"\r\nBy what name do you wish to be known? " +
		"Invalid name, please try another.\r\nName: " +
		"Please remember to choose an appropriate fantasy-oriented name.\r\n" +
		"Did I get that right, Transcript (Y/N)? " +
		"New character.\r\nGive me a password for Transcript: " +
		"\r\nPlease retype password: " +
		"\r\nDo you want ANSI color (Y/N)? " +
		"What is your sex (M/F)? " +
		session.RaceMenuText + "\r\nRace: " +
		session.HumanClassMenuText + "\r\nClass: " +
		session.HometownMenuText + "\r\nSelect: " +
		"<ROLLED_STATS>\r\nPress 'Y' to keep these stats, and 'N' to reroll:" +
		string(motd) + "\r\n\n*** PRESS RETURN: " +
		"\n\rWelcome to Dark Pawns!\n\r0) Exit from Dark Pawns.\n\r1) Enter the game.\r\n" +
		"2) Enter description.\r\n3) Read the background story.\r\n4) Change password.\r\n" +
		"5) Delete this character.\r\n\r\n   Make your choice: " +
		"\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n\r\n"
	if !strings.HasPrefix(visible, wantPrefix) {
		t.Fatalf("creation transcript prefix differs from C\ngot:  %q\nwant: %q", visible, wantPrefix)
	}
	if got := strings.Count(visible, "Welcome to Dark Pawns: darkpawns.com 4300"); got != 1 {
		t.Errorf("MOTD count = %d, want 1", got)
	}
	if got := strings.Count(visible, "A Burning Hut"); got != 1 {
		t.Errorf("start-room count = %d, want 1", got)
	}

	_ = client.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not stop after transcript client closed")
	}
}

func stripTelnetCommands(data []byte) []byte {
	visible := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == IAC && i+2 < len(data) {
			i += 2
			continue
		}
		visible = append(visible, data[i])
	}
	return visible
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
	// The server must NOT offer WILL SGA: combined with the password WILL ECHO
	// it is the character-at-a-time signature that breaks line-mode clients
	// (Mudlet). Original C DarkPawns never negotiates SGA.
	offeredSGA := bytes.Contains(data, []byte{IAC, WILL, OPT_SGA})
	_ = client.Close()
	<-done

	if !found {
		t.Errorf("initial negotiation did not include IAC WILL COMPRESS2, got %v", data)
	}
	if offeredSGA {
		t.Errorf("initial negotiation must not offer IAC WILL SGA (char-at-a-time), got %v", data)
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

// TestPromptAfterCommandOutput verifies the "> " prompt is written after the
// command's response, never before it. The prompt now travels through the
// session's send channel, so writeLoop drains the command output first and
// prints the prompt after (C: comm.c:643-648 flush output, then prompt).
func TestPromptAfterCommandOutput(t *testing.T) {
	world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum: game.MortalStartRoom,
		Name: "The Temple",
		Zone: 80,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)

	client, server := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, game.BanNot)
	}()

	transcript := make(chan []byte, 1)
	go func() {
		output, _ := io.ReadAll(client)
		transcript <- output
	}()

	const playerName = "guest_prompt_order"
	if _, err := client.Write([]byte(playerName + "\r\n")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := manager.GetSession(playerName); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("guest session was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := client.Write([]byte("say hello\r\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write([]byte("quit\r\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleConn did not return after quit")
	}

	visible := string(stripTelnetCommands(<-transcript))
	const response = "You say 'hello'\r\n"
	respIdx := strings.LastIndex(visible, response)
	if respIdx < 0 {
		t.Fatalf("transcript missing command response: %q", visible)
	}
	// C's process_output flush frame is output + "\r\n" + make_prompt
	// (comm.c:1624-1640), so the prompt follows the response after a line
	// break (plus any vitals fields), never before it.
	if !strings.Contains(visible[respIdx+len(response):], "> ") {
		t.Fatalf("prompt did not follow command response (prompt/response race): %q", visible)
	}
}

// TestEffectiveBanLevelShortCircuitsBanAllBeforeDNS verifies that an IP already
// banned at BanAll level is rejected from the in-memory check alone, without
// ever spawning the reverse-DNS lookup (banned IPs must not pay for a slow PTR).
func TestEffectiveBanLevelShortCircuitsBanAllBeforeDNS(t *testing.T) {
	bm := game.NewBanManager()
	if err := bm.AddBan("203.0.113.99", game.BanAll, "test"); err != nil {
		t.Fatal(err)
	}

	origLookup := lookupAddr
	defer func() { lookupAddr = origLookup }()
	var lookupCalled atomic.Bool
	lookupAddr = func(addr string) ([]string, error) {
		lookupCalled.Store(true)
		return nil, nil
	}

	if got := effectiveBanLevel("203.0.113.99", bm); got != game.BanAll {
		t.Fatalf("effectiveBanLevel = %d, want BanAll (%d)", got, game.BanAll)
	}
	if lookupCalled.Load() {
		t.Fatal("lookupAddr was invoked for an IP already banned at BanAll level")
	}
}

// TestListenAcceptNotBlockedBySlowReverseDNS verifies the accept loop only
// performs the fast in-memory IP ban check; hostname reverse-DNS resolution runs
// in the per-connection goroutine, so a connection whose PTR lookup blocks does
// not stall subsequent accepts.
func TestListenAcceptNotBlockedBySlowReverseDNS(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	origLookup := lookupAddr
	origTimeout := dnsLookupTimeout

	var mu sync.Mutex
	callCount := 0
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }

	var c1, c2 net.Conn
	// Tear down in an order that guarantees no per-connection goroutine is
	// still reading the lookupAddr / dnsLookupTimeout package vars when we
	// restore them (those vars are read once per connection at accept time,
	// inside effectiveBanLevel). Stop accepting, unblock the stalled lookup,
	// close the client conns, then wait for the handler goroutines to drain
	// before restoring the vars — otherwise the restore races the readers.
	defer func() {
		Stop()
		unblock()
		if c1 != nil {
			_ = c1.Close()
		}
		if c2 != nil {
			_ = c2.Close()
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			connMu.Lock()
			n := connCount
			connMu.Unlock()
			if n == 0 || time.Now().After(deadline) {
				break
			}
			time.Sleep(time.Millisecond)
		}
		lookupAddr = origLookup
		dnsLookupTimeout = origTimeout
	}()

	// Keep the DNS timeout longer than the test deadline so the blocked
	// connection stays pending on its resolver for the whole test.
	dnsLookupTimeout = 5 * time.Second

	lookupAddr = func(addr string) ([]string, error) {
		mu.Lock()
		callCount++
		block := callCount == 1
		mu.Unlock()
		if block {
			<-release
		}
		return nil, nil
	}

	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}

	connMu.Lock()
	addr := listener.Addr().String()
	connMu.Unlock()

	dial := func() net.Conn {
		t.Helper()
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("dial %s: %v", addr, err)
		}
		return c
	}
	c1 = dial()
	c2 = dial()

	// The first reverse-DNS lookup blocks; whichever connection it belongs to,
	// the peer connection must still be served promptly (proving the accept loop
	// did not stall on the PTR wait).
	served := make(chan string, 1)
	for name, c := range map[string]net.Conn{"first": c1, "second": c2} {
		name, c := name, c
		go func() {
			buf := make([]byte, 4096)
			for {
				_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
				var chunk [512]byte
				n, err := c.Read(chunk[:])
				if err != nil {
					return
				}
				buf = append(buf, chunk[:n]...)
				if bytes.Contains(buf, []byte("By what name do you wish to be known?")) {
					served <- name
					return
				}
			}
		}()
	}

	select {
	case name := <-served:
		t.Logf("connection %q served while its peer blocked on reverse-DNS", name)
	case <-time.After(2 * time.Second):
		t.Fatal("no connection served within deadline: accept loop stalled on reverse-DNS")
	}
}
