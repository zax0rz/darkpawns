package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// readReportLine drains a session's outgoing messages until one starts with
// prefix and returns it. It fails when the report ends first.
func readReportLine(t *testing.T, s *Session, prefix string) string {
	t.Helper()
	for i := 0; i < 40; i++ {
		line := readSendText(t, s)
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("report has no line starting %q", prefix)
	return ""
}

// TestShowOptionsListFiltersByLevel pins C's no-argument `show` listing
// (src/act.wizard.c:2271-2277): only the rows whose level the character meets,
// laid out five per line, then one trailing CRLF. The level-40 bytes are
// exactly what this code hardcoded before the review fix, so reverting to that
// literal string fails the level-31 and level-34 cases.
func TestShowOptionsListFiltersByLevel(t *testing.T) {
	cases := []struct {
		level int
		want  string
	}{
		{31, "Show options:\r\n" +
			"zones          stats          shops          tattoos        reagents       \r\n" +
			"hooks          neutral        \r\n"},
		{34, "Show options:\r\n" +
			"zones          player         rent           stats          death          \r\n" +
			"godrooms       shops          houses         tattoos        reagents       \r\n" +
			"hooks          neutral        \r\n"},
		{40, "Show options:\r\n" +
			"zones          player         rent           stats          errors         \r\n" +
			"death          godrooms       shops          houses         tattoos        \r\n" +
			"aggr           reagents       hooks          neutral        \r\n"},
	}
	for _, tc := range cases {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "Showgod", tc.level, 1001)
		if err := cmdShow(s, nil); err != nil {
			t.Fatalf("level %d: cmdShow: %v", tc.level, err)
		}
		if got := readSendText(t, s); got != tc.want {
			t.Errorf("level %d listing =\n%q\nwant\n%q", tc.level, got, tc.want)
		}
	}
}

// TestStatHometownLineAddsOlcZoneForImmortalsOnly pins C's conditional
// ", OLC[zone]" suffix on the stat report's Hometown line
// (src/act.wizard.c:767-776): immortals get it, mortals do not.
func TestStatHometownLineAddsOlcZoneForImmortalsOnly(t *testing.T) {
	cases := []struct {
		targetLevel int
		wantOLC     bool
	}{
		{31, true},
		{30, false},
	}
	for _, tc := range cases {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Statgod", 34, 1001)
		target := makeCommandTestSession(t, m, "Stattarget", tc.targetLevel, 1001)
		target.olcZone = 77
		m.mu.Lock()
		m.sessions["target"] = target
		m.mu.Unlock()

		if err := cmdStat(actor, []string{"player", "Stattarget"}); err != nil {
			t.Fatalf("level %d: cmdStat: %v", tc.targetLevel, err)
		}
		line := readReportLine(t, actor, "Hometown: [")
		if tc.wantOLC && !strings.Contains(line, ", OLC[77]") {
			t.Errorf("level %d target Hometown line = %q, want the OLC zone", tc.targetLevel, line)
		}
		if !tc.wantOLC && strings.Contains(line, "OLC[") {
			t.Errorf("level %d target Hometown line = %q, want no OLC zone", tc.targetLevel, line)
		}
	}
}

// TestShowStatsCountsVisibleConnectedAndRegistered pins C's `show stats`
// counters (src/act.wizard.c:2356-2378): the visible non-NPC characters in the
// world, how many of those hold a descriptor (a linkdead player counts only in
// the first), and the player-file index size, which the port reads from the
// player store. The pre-fix code printed the online count three times.
func TestShowStatsCountsVisibleConnectedAndRegistered(t *testing.T) {
	store, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "reports.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	for i := 0; i < 3; i++ {
		if err := store.CreatePlayer(&db.PlayerRecord{
			Name: fmt.Sprintf("Saved%d", i), Password: "x", Level: 1, RoomVNum: 1001,
		}); err != nil {
			t.Fatalf("seed record %d: %v", i, err)
		}
	}

	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1},
			{VNum: 1002, Name: "Room B", Zone: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	m := newTestManager(t, w, store)
	actor := makeCommandTestSession(t, m, "Statgod", 34, 1001)

	// One visible player on a live connection.
	online := makeCommandTestSession(t, m, "Onlineone", 30, 1001)
	if err := m.world.AddPlayer(online.player); err != nil {
		t.Fatalf("add online player: %v", err)
	}
	m.mu.Lock()
	m.sessions["online"] = online
	m.mu.Unlock()

	// One linkdead player: still in the world, with no session.
	if err := m.world.AddPlayer(game.NewPlayer(9, "Linkdeadone", 1001)); err != nil {
		t.Fatalf("add linkdead player: %v", err)
	}

	if err := cmdShow(actor, []string{"stats"}); err != nil {
		t.Fatalf("cmdShow stats: %v", err)
	}
	block := readSendText(t, actor)
	if !strings.Contains(block, "  "+"    2"+" players in game  "+"    1"+" connected\r\n") {
		t.Errorf("players-in-game/connected line wrong:\n%q", block)
	}
	if !strings.Contains(block, "  "+"    3"+" registered\r\n") {
		t.Errorf("registered line wrong (want the store count):\n%q", block)
	}
}

// TestShowLookupResolvesNothingRowFirst pins C's lookup order
// (src/act.wizard.c:2285-2287): the prefix scan starts at row 0, "nothing",
// which has no switch case, so "n", "no" and "nothing" get C's default reply
// instead of the neutral-room report; "ne" still reaches "neutral".
func TestShowLookupResolvesNothingRowFirst(t *testing.T) {
	const sorry = "Sorry, I don't understand that."
	for _, field := range []string{"n", "no", "nothing"} {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "Showgod", 40, 1001)
		if err := cmdShow(s, []string{field}); err != nil {
			t.Fatalf("show %s: %v", field, err)
		}
		if got := readSendText(t, s); got != sorry {
			t.Errorf("show %s = %q, want %q", field, got, sorry)
		}
	}
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Showgod", 40, 1001)
	if err := cmdShow(s, []string{"ne"}); err != nil {
		t.Fatalf("show ne: %v", err)
	}
	if got := readSendText(t, s); !strings.HasPrefix(got, "Neutral Rooms\r\n") {
		t.Errorf("show ne = %q, want the neutral-room report", got)
	}
}
