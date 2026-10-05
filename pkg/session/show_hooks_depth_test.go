package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShowHooksIndexedExitMatrix(t *testing.T) {
	w, err := game.NewWorld(&parser.World{Zones: []parser.Zone{{Number: 7, Name: "Seven"}, {Number: 9, Name: "Nine"}}, Rooms: []parser.Room{
		{VNum: 1002, Zone: 7, Exits: map[string]parser.Exit{"west": {ToRoom: 9001}, "north": {ToRoom: 9001}, "east": {ToRoom: 1001}}},
		{VNum: 1001, Zone: 7, Exits: map[string]parser.Exit{"down": {ToRoom: 9001}}},
		{VNum: 9001, Zone: 9, Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
	for _, tc := range []struct{ value, want string }{
		{"7", "Connections in zone 7.\r\n========================\r\n 1002 leads north to 9001  -- Nine\r\n 1002 leads west to 9001  -- Nine\r\n 1001 leads down to 9001  -- Nine\r\n"},
		{"9", "Connections in zone 9.\r\n========================\r\n 9001 leads south to 1001  -- Seven\r\n"},
		{"foo", ""},
		{"-7", ""},
		{"999", "That is not a valid zone.\r\n"},
	} {
		if err := cmdShow(s, []string{"hooks", tc.value}); err != nil {
			t.Fatal(err)
		}
		if got := drainShowReport(t, s); got != tc.want {
			t.Errorf("%q: %q want %q", tc.value, got, tc.want)
		}
	}
}
