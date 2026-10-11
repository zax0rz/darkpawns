package session

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// whodWorld builds a world with one room and a manager, without starting the
// daemon, so a test can capture the log file or register observers first.
func whodWorld(t *testing.T) (*Manager, *game.World) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Whod Room"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return newTestManager(t, w, nil), w
}

// startWhod starts the daemon on an ephemeral port with no post-write delay
// and returns its address.
func startWhod(t *testing.T, m *Manager) string {
	t.Helper()
	if err := m.startWhodListener(0, 0); err != nil {
		t.Fatalf("startWhodListener: %v", err)
	}
	t.Cleanup(m.StopWhodListener)
	addr := m.WhodListenerAddr()
	if addr == "" {
		t.Fatal("listener has no bound address")
	}
	return addr
}

// whodLogCapture is captureMudlogFile with a lock: the listener's serve and
// supervisor goroutines log while these tests read, so a bare bytes.Buffer
// races.
type whodLogCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *whodLogCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *whodLogCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func captureWhodLog(t *testing.T) *whodLogCapture {
	t.Helper()
	c := &whodLogCapture{}
	game.SetLogWriter(c)
	t.Cleanup(func() { game.SetLogWriter(os.Stderr) })
	return c
}

// whodWhoRequest opens one connection and returns C's response bytes.
func whodWhoRequest(t *testing.T, addr string) string {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	data, err := io.ReadAll(conn)
	if err != nil && len(data) == 0 {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

// TestWhodListenerServesExactCBytes pins the full response for the default
// mode: greeting, header, one mortal line, footer.
func TestWhodListenerServesExactCBytes(t *testing.T) {
	m, _ := whodWorld(t)
	addr := startWhod(t, m)

	s := makeTestSession(t, m, "Albe", 1001, true)
	s.player.Class = game.ClassWarrior
	s.player.Level = 3
	s.player.Title = "the Test"
	registerTestSession(t, m, s, "Albe")

	got := whodWhoRequest(t, addr)
	want := game.WhodGreeting() + game.WhodHeader() +
		"[ 3 Wa] Albe the Test\r\n" + "\n\rPlayers : 1     Gods : 0\n\r\n\r"
	if got != want {
		t.Fatalf("WHOD bytes =\n%q\nwant\n%q", got, want)
	}
}

// TestWhodListenerHidesInvisibleAndLinkless pins the two filters C applies per
// character: IS_INVIS without SHOW_INVIS (src/whod.c:345) and ch->desc with
// DISPLAY_LINKDEAD 0 (src/whod.c:45,344).
func TestWhodListenerHidesInvisibleAndLinkless(t *testing.T) {
	m, _ := whodWorld(t)
	addr := startWhod(t, m)

	hidden := makeTestSession(t, m, "Hidden", 1001, true)
	hidden.player.Level = 3
	hidden.player.SetInvisLevel(1)
	registerTestSession(t, m, hidden, "Hidden")

	gone := makeTestSession(t, m, "Gone", 1001, true)
	gone.player.Level = 3
	gone.player.SetLinkless(true)
	registerTestSession(t, m, gone, "Gone")

	seen := makeTestSession(t, m, "Seen", 1001, true)
	seen.player.Level = 3
	registerTestSession(t, m, seen, "Seen")

	got := whodWhoRequest(t, addr)
	if strings.Contains(got, "Hidden") {
		t.Fatalf("wizinvis player leaked without SHOW_INVIS: %q", got)
	}
	if strings.Contains(got, "Gone") {
		t.Fatalf("linkless player leaked (DISPLAY_LINKDEAD is 0): %q", got)
	}
	if !strings.Contains(got, "Seen") || !strings.Contains(got, "Players : 1     Gods : 0") {
		t.Fatalf("visible player missing or miscounted: %q", got)
	}

	// SHOW_INVIS reveals the wizinvis player.
	m.world.WhodDisplay.SetMode(game.WhodDefaultMode | game.WhodShowInvis)
	got = whodWhoRequest(t, addr)
	if !strings.Contains(got, "Hidden") || !strings.Contains(got, "Players : 2     Gods : 0") {
		t.Fatalf("SHOW_INVIS bytes = %q", got)
	}
}

// TestWhodListenerProducers pins the three socket-lifecycle producers at
// BRF / LVL_GOD / file TRUE (src/whod.c:250,267,296,335).
func TestWhodListenerProducers(t *testing.T) {
	m, _ := whodWorld(t)
	watch := makeTestSession(t, m, "Whodwatch", 1001, true)
	watch.player.SetLevel(game.LVL_GOD)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	registerTestSession(t, m, watch, "Whodwatch")
	below := makeTestSession(t, m, "Whodbelow", 1001, true)
	below.player.SetLevel(game.LVL_GOD - 1)
	below.player.SetPlrFlag(game.PrfLog1, true)
	registerTestSession(t, m, below, "Whodbelow")
	drainSessionText(t, watch)
	drainSessionText(t, below)
	file := captureWhodLog(t)

	addr := startWhod(t, m)
	whodWhoRequest(t, addr)
	m.StopWhodListener()

	log := file.String()
	for _, payload := range []string{"WHOD port opened.", "WHO request from ", " served.", "WHOD port closed."} {
		if !strings.Contains(log, payload) {
			t.Fatalf("missing file payload %q in %q", payload, log)
		}
	}
	seen := strings.Join(drainSessionText(t, watch), "")
	for _, payload := range []string{"WHOD port opened.", "WHOD port closed."} {
		if !strings.Contains(seen, "[ "+payload+" ]\r\n") {
			t.Fatalf("observer missing %q: %q", payload, seen)
		}
	}
	if !strings.Contains(seen, "[ WHO request from ") || !strings.Contains(seen, " served. ]") {
		t.Fatalf("observer missing the served producer: %q", seen)
	}
	if got := strings.Join(drainSessionText(t, below), ""); got != "" {
		t.Fatalf("below-threshold observer saw %q", got)
	}
}

// TestWhodListenerOffClosesAndOnReopens pins C's on/off lifecycle: `whod off`
// closes the port with one producer, `whod on` reopens it through the OPENING
// arm (src/whod.c:432-433,294-298,459-462).
func TestWhodListenerOffClosesAndOnReopens(t *testing.T) {
	m, w := whodWorld(t)
	file := captureWhodLog(t)
	addr := startWhod(t, m)
	if !strings.Contains(file.String(), "WHOD port opened.") {
		t.Fatalf("start did not log the open: %q", file.String())
	}

	w.WhodDisplay.SetMode(game.WhodShowOff)
	waitFor(t, func() bool { return m.WhodListenerAddr() == "" })
	if got := strings.Count(file.String(), "WHOD port closed."); got != 1 {
		t.Fatalf("closed producer count = %d, want 1: %q", got, file.String())
	}
	if got := whodWhoRequestAddr(addr); got != "" {
		t.Fatalf("the closed port still answered: %q", got)
	}

	w.WhodDisplay.SetMode(game.WhodShowOn)
	waitFor(t, func() bool { return m.WhodListenerAddr() != "" })
	if got := strings.Count(file.String(), "WHOD port opened."); got != 2 {
		t.Fatalf("opened producer count = %d, want 2: %q", got, file.String())
	}
	if got := whodWhoRequest(t, m.WhodListenerAddr()); !strings.Contains(got, "A list of active players on Dark Pawns:") {
		t.Fatalf("the reopened port did not serve: %q", got)
	}
}

// waitFor polls cond for up to 3 seconds: the supervisor reconciles every
// whodPollInterval, so a state change is not instantaneous.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not reached within 3s")
}

// whodWhoRequestAddr is whodWhoRequest for a port expected to refuse.
func whodWhoRequestAddr(addr string) string {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return ""
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	data, _ := io.ReadAll(conn)
	return string(data)
}

// TestWhodRequestHostBranches pins whod_loop's two host forms (src/whod.c:322-333).
func TestWhodRequestHostBranches(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("10.1.2.3"), Port: 5}
	resolve := func(string) ([]string, error) { return []string{"whod.example"}, nil }
	if got := whodRequestHost(addr, resolve); got != "whod.example" {
		t.Fatalf("resolved host = %q", got)
	}
	fail := func(string) ([]string, error) { return nil, errors.New("no ptr") }
	if got := whodRequestHost(addr, fail); got != "10.1.2.3" {
		t.Fatalf("fallback quad = %q, want C's unpadded form", got)
	}
	if got := whodRequestHost(nil, fail); got != "0.0.0.0" {
		t.Fatalf("nil addr = %q", got)
	}
}
