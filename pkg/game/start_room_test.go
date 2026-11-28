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
		wantOutput  string
	}{
		{
			name:        "mortal",
			level:       1,
			wantHandled: true,
			wantRoom:    8162,
			wantOutput: "   Suddenly the hairs on the back of your neck stand up as if lightning had\n\r" +
				"struck nearby. A keen wailing fills the air, and an ethereal image appears\n" +
				"before you.\n\r" +
				"   'Startroom, now is not your time to die,' speaks the figure.\r\n" +
				"   'Prove your worth and I may well grant you eternal life.'\r\n" +
				"   'Trust no one, for all here are but dark pawns above which you must \r\n" +
				"struggle to prove yourself.  All here strive to be a king... at any cost.'\r\n" +
				"   The figure glows a moment, then disappears, but his voice remains.\r\n" +
				"   'Your life begins now...' it says, then fades -- just as the world around\r\n" +
				"you does the same.\r\n\r\n" +
				"Temple Infirmary\r\n   The infirmary.\r\n[ Exits: None! ]\r\n",
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
					{VNum: 8162, Name: "Temple Infirmary", Description: "   The infirmary.\n"},
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
			if tt.wantOutput != "" {
				if got := output.String(); got != tt.wantOutput {
					t.Errorf("output = %q, want exact C transcript %q", got, tt.wantOutput)
				}
			} else if got := output.String(); got != "" {
				t.Errorf("output = %q, want empty output", got)
			}
		})
	}
}
