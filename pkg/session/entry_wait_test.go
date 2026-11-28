package session

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

// The nanny's name step loads an existing character with wait = 1
// (src/interpreter.c:1766). C's loop decrements every descriptor's wait on
// every pass (src/comm.c:597), so in live play the password is read as it
// arrives. Under the oracle's frozen clock only a pumped pass decrements it:
// the password and the lines behind it wait on d->input until a pump, then run
// back to back before one output flush (src/comm.c:594-642). Lockstep play
// relogged a linkdead body with the plain login script and found the port
// reading each line at once (triage s00076).

const entryWaitPassword = "hunter2"

// newEntryWaitManager stores "Returner" and installs a pulse pump over the
// per-pulse drain, as the server's game loop does.
func newEntryWaitManager(t *testing.T) *Manager {
	t.Helper()
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)
	hash, err := bcrypt.GenerateFromPassword([]byte(entryWaitPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{
		Name: "Returner", Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 1, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	m.SetPulsePump(func(n int) error {
		m.BeginHeartbeatOutput()
		defer m.EndHeartbeatOutput()
		for range n {
			m.DrainInputQueues()
		}
		return nil
	})
	return m
}

// newEntryConnection is a connection that has not logged in. C's loop walks
// descriptor_list, so the port's drain has to find it before it has a player:
// the port keeps such a connection in m.descriptors, not m.sessions.
func newEntryConnection(t *testing.T, m *Manager) *Session {
	t.Helper()
	s := makeCharSession(t, m)
	m.trackDescriptor(s)
	return s
}

// entryLine sends one terminal line and fails if the connection closes.
func entryLine(t *testing.T, s *Session, line string) {
	t.Helper()
	if open := s.TerminalLine(line); !open {
		t.Fatalf("TerminalLine(%q) closed the connection", line)
	}
}

// outputLines is the text a session received, split on line endings and
// trimmed, so a prompt reads ">".
func outputLines(s *Session) []string {
	rendered := renderAllOutput(s)
	var lines []string
	for _, line := range strings.FieldsFunc(rendered, func(r rune) bool { return r == '\r' || r == '\n' }) {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func TestEntryWaitHoldsThePasswordUntilAPulsedPass(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	m := newEntryWaitManager(t)
	s := newEntryConnection(t, m)

	entryLine(t, s, "Returner")
	if got := s.entryWait.Load(); got != 1 {
		t.Fatalf("wait after the name = %d, want 1 (src/interpreter.c:1766)", got)
	}
	entryLine(t, s, entryWaitPassword)
	if s.player != nil || s.queueLen() != 1 {
		t.Fatalf("the password was read without a pulsing pass: player=%v queue=%d", s.player != nil, s.queueLen())
	}

	// An idle pass leaves the wait and the line alone, as the oracle's does.
	m.drainInputPass(false)
	if s.entryWait.Load() != 1 || s.queueLen() != 1 {
		t.Fatalf("an idle pass moved the hold: wait=%d queue=%d", s.entryWait.Load(), s.queueLen())
	}

	m.DrainInputQueues()
	if s.entryWait.Load() != 0 || s.queueLen() != 0 {
		t.Fatalf("a pulsing pass did not release the hold: wait=%d queue=%d", s.entryWait.Load(), s.queueLen())
	}
	if !s.menuActive {
		t.Fatal("the held password was not read by the pulsing pass")
	}
}

// TestEntryWaitReconnectRunsHeldLinesBeforeOneFlush is the lockstep finding:
// a linkdead body is relogged with the creation script's lines (password,
// RETURN, the menu choice 1, RETURN) and no pulse between them.
func TestEntryWaitReconnectRunsHeldLinesBeforeOneFlush(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	m := newEntryWaitManager(t)

	// Returner is in the world; the connection dropped without a quit.
	old := makeTestSession(t, m, "Returner", game.MortalStartRoom, true)
	old.transportDone = make(chan struct{})
	registerTestSession(t, m, old, "Returner")
	old.player.SetLinkless(true)
	old.DetachTransport()

	s := newEntryConnection(t, m)
	entryLine(t, s, "Returner")
	entryLine(t, s, entryWaitPassword)
	entryLine(t, s, "")
	entryLine(t, s, "1")
	entryLine(t, s, "")
	if got := s.queueLen(); got != 4 {
		t.Fatalf("queued lines = %d, want the password and the three behind it", got)
	}
	if got := outputLines(s); len(got) != 0 {
		t.Fatalf("output before the pump = %q, want none", got)
	}

	// The harness settles a login with 40 pulses after its menu choice.
	if err := m.PumpPulsesFrom(s, 40); err != nil {
		t.Fatalf("PumpPulsesFrom: %v", err)
	}
	if s.player != old.player {
		t.Fatal("the held password did not reconnect the linkdead body")
	}
	if got := s.queueLen(); got != 0 {
		t.Fatalf("queued lines after the pump = %d, want 0", got)
	}
	got := outputLines(s)
	want := []string{"Reconnecting.", "Huh?!?", ">"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("reconnect output = %q, want C's one flush %q", got, want)
	}
}

func TestEntryNameSetsNoWaitOutsideTheFrozenClock(t *testing.T) {
	// Frozen() is the presence of DP_CLOCK, so an empty value is still frozen.
	if err := os.Unsetenv("DP_CLOCK"); err != nil {
		t.Fatal(err)
	}
	m := newEntryWaitManager(t)
	s := newEntryConnection(t, m)

	entryLine(t, s, "Returner")
	if got := s.entryWait.Load(); got != 0 {
		t.Fatalf("wait after the name = %d, want none in live play", got)
	}
	entryLine(t, s, entryWaitPassword)
	if !s.menuActive || s.queueLen() != 0 {
		t.Fatalf("live password was not read as it arrived: menu=%v queue=%d", s.menuActive, s.queueLen())
	}
}

func TestEntryNameSetsNoWaitForANewCharacter(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	m := newEntryWaitManager(t)
	s := newEntryConnection(t, m)

	entryLine(t, s, "Freshface")
	if got := s.entryWait.Load(); got != 0 {
		t.Fatalf("wait after a new name = %d, want none (src/interpreter.c:1762-1766 loads only a stored character)", got)
	}
	// The confirmation's answer is read at once too: the flow moves on to the
	// password prompt and nothing is left on the queue.
	entryLine(t, s, "Y")
	if s.charStage != "create_password" || s.queueLen() != 0 {
		t.Fatalf("a new name's confirmation was held: stage=%q queue=%d", s.charStage, s.queueLen())
	}
}

// TestEntryWaitLinesBehindThePasswordReachTheGame is the other arm of the
// password: no live body, so it leads to the MOTD and the menu. Between the
// two the session has a player but is not in m.sessions yet, and the lines
// behind the password still have to drain.
func TestEntryWaitLinesBehindThePasswordReachTheGame(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	m := newEntryWaitManager(t)
	s := newEntryConnection(t, m)

	entryLine(t, s, "Returner")
	entryLine(t, s, entryWaitPassword)
	entryLine(t, s, "")
	entryLine(t, s, "1")
	if got := s.queueLen(); got != 3 {
		t.Fatalf("queued lines = %d, want the password and the two behind it", got)
	}

	if err := m.PumpPulsesFrom(s, 40); err != nil {
		t.Fatalf("PumpPulsesFrom: %v", err)
	}
	if got := s.queueLen(); got != 0 {
		t.Fatalf("queued lines after the pump = %d, want 0", got)
	}
	if s.menuActive {
		t.Fatal("the menu choice behind the password was not read")
	}
	if _, ok := m.GetSession("Returner"); !ok {
		t.Fatal("the returning player was not registered")
	}
}
