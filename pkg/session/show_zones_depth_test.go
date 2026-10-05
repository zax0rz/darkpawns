package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Reports execute synchronously; an absent block is an assertion, not a timeout.
func drainShowReport(t *testing.T, s *Session) string {
	t.Helper()
	var b strings.Builder
	for {
		select {
		case msg := <-s.send:
			var envelope struct {
				Data EventData `json:"data"`
			}
			if err := json.Unmarshal(msg, &envelope); err != nil {
				t.Fatal(err)
			}
			b.WriteString(envelope.Data.Text)
		default:
			return b.String()
		}
	}
}

func TestShowZonesRuntimeAndSelection(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Zone: 7}}, Zones: []parser.Zone{
		{Number: 9, Name: "Nine", TopRoom: 9999, Lifespan: 12, ResetMode: 0},
		{Number: 7, Name: "Seven", TopRoom: 7999, Lifespan: 10, ResetMode: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.StartZoneResets(); err != nil {
		t.Fatal(err)
	}
	for range 6 {
		w.ZoneUpdate()
	}
	s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
	const seven = "  7 Seven                          Age:   1; Reset:  10 (1); Top:  7999\r\n"
	const nine = "  9 Nine                           Age:   0; Reset:  12 (0); Top:  9999\r\n"
	for _, tc := range []struct{ value, want string }{{".", seven}, {"007", seven}, {"9", nine}, {"", seven + nine}, {"nonnumeric", seven + nine}, {"-7", seven + nine}, {"999", "That is not a valid zone.\r\n"}} {
		args := []string{"zones"}
		if tc.value != "" {
			args = append(args, tc.value)
		}
		if err := cmdShow(s, args); err != nil {
			t.Fatal(err)
		}
		if got := drainShowReport(t, s); got != tc.want {
			t.Errorf("%q: %q want %q", tc.value, got, tc.want)
		}
	}
}
