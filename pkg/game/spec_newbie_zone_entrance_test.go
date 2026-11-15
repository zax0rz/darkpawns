package game

import "testing"

func TestSpecNewbieZoneEntrance_Gates(t *testing.T) {
	tests := []struct {
		name       string
		cmd        string
		level      int
		wantHandle bool
		wantOutput string
	}{
		{name: "non-south", cmd: "look", level: 11},
		{name: "below-newbie-level", cmd: "south", level: 10},
		{
			name:       "newbie-level-block",
			cmd:        "south",
			level:      11,
			wantHandle: true,
			wantOutput: "Nah, you're too much of a badass to go in there!\r\n",
		},
		{
			name:  "immortal-fallthrough",
			cmd:   "south",
			level: lvlImmort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			_ = lastMsg() // discard setup output
			player.SetLevel(tt.level)

			if got := specNewbieZoneEntrance(w, player, nil, tt.cmd, ""); got != tt.wantHandle {
				t.Errorf("handled = %v, want %v", got, tt.wantHandle)
			}
			if got := lastMsg(); got != tt.wantOutput {
				t.Errorf("output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestStartRoomBirthMessageExactBytes(t *testing.T) {
	got := startRoomBirthMessage("Alice")
	want := "   Suddenly the hairs on the back of your neck stand up as if lightning had\n\r" +
		"struck nearby. A keen wailing fills the air, and an ethereal image appears\n" +
		"before you.\n\r" +
		"   'Alice, now is not your time to die,' speaks the figure.\r\n" +
		"   'Prove your worth and I may well grant you eternal life.'\r\n" +
		"   'Trust no one, for all here are but dark pawns above which you must \r\n" +
		"struggle to prove yourself.  All here strive to be a king... at any cost.'\r\n" +
		"   The figure glows a moment, then disappears, but his voice remains.\r\n" +
		"   'Your life begins now...' it says, then fades -- just as the world around\r\n" +
		"you does the same.\r\n\r\n"
	if got != want {
		t.Fatalf("startRoomBirthMessage mismatch:\n got %q\nwant %q", got, want)
	}
}
