package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestNewbieHometownRoom(t *testing.T) {
	tests := []struct {
		name     string
		hometown int
		want     int
	}{
		{name: "Kir Drax'in", hometown: 1, want: 8162},
		{name: "Kir-Oshi", hometown: 2, want: 18201},
		{name: "Alaozar", hometown: 3, want: 21202},
		{name: "unset", hometown: 0, want: MortalStartRoom},
		{name: "unknown", hometown: 99, want: MortalStartRoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewbieHometownRoom(tt.hometown); got != tt.want {
				t.Errorf("NewbieHometownRoom(%d) = %d, want %d", tt.hometown, got, tt.want)
			}
		})
	}
}

func TestSpecStartRoom_BirthTransitionAndImmortalGate(t *testing.T) {
	tests := []struct {
		name        string
		level       int
		wantHandled bool
		wantRoom    int
		wantPrefix  string
	}{
		{
			name:        "mortal",
			level:       1,
			wantHandled: true,
			wantRoom:    8162,
			wantPrefix:  "\r\n   'Startroom, now is not your time to die,' speaks the figure.",
		},
		{
			name:        "immortal",
			level:       lvlImmort,
			wantHandled: false,
			wantRoom:    NewbieStartRoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &parser.World{
				Rooms: []parser.Room{
					{VNum: NewbieStartRoom, Name: "Burning Hut", Description: "intro"},
					{VNum: 8162, Name: "Temple Infirmary", Description: "infirmary"},
				},
			}
			w, err := NewWorld(parsed)
			if err != nil {
				t.Fatalf("NewWorld failed: %v", err)
			}
			t.Cleanup(func() { w.StopAITicker() })

			var output strings.Builder
			w.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
			player := NewPlayer(1, "Startroom", NewbieStartRoom)
			player.SetLevel(tt.level)
			player.Hometown = 1
			if err := w.AddPlayer(player); err != nil {
				t.Fatalf("AddPlayer failed: %v", err)
			}

			if got := specStartRoom(w, player, nil, "look", ""); got != tt.wantHandled {
				t.Errorf("handled = %v, want %v", got, tt.wantHandled)
			}
			if got := player.GetRoomVNum(); got != tt.wantRoom {
				t.Errorf("room = %d, want %d", got, tt.wantRoom)
			}
			if tt.wantPrefix != "" {
				if got := output.String(); !strings.HasPrefix(got, tt.wantPrefix) {
					t.Errorf("output = %q, want prefix %q", got, tt.wantPrefix)
				}
			} else if got := output.String(); got != "" {
				t.Errorf("output = %q, want empty output", got)
			}
		})
	}
}
