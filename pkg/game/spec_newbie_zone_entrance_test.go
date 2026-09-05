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
