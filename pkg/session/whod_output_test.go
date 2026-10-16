package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestWhodOutputBranches exercises the registered command and pins the
// send_to_char bytes in C's no-argument, invalid-mode, and already-on/off arms.
func TestWhodOutputBranches(t *testing.T) {
	tests := []struct {
		name string
		mode int
		args []string
		want string
	}{
		{
			name: "current mode",
			mode: game.WhodDefaultMode,
			want: "Current WHOD mode:\n\r------------------\n\rname class level title on \n\r",
		},
		{
			name: "empty mode bitvector",
			mode: 0,
			want: "Current WHOD mode:\n\r------------------\n\rNOBITS \n\r",
		},
		{
			name: "invalid mode",
			mode: game.WhodDefaultMode,
			args: []string{"bogus"},
			want: "That mode does not exist.\n\rAvailable modes are:\n\rname class level title wizinvis site wizlevel on off \n\r",
		},
		{
			name: "already on",
			mode: game.WhodDefaultMode,
			args: []string{"on"},
			want: "WHOD already turned on.\n\r",
		},
		{
			name: "already off",
			mode: game.WhodShowOff,
			args: []string{"off"},
			want: "WHOD already turned off.\n\r",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, actor, _, _ := wizardLogFixture(t)
			m.world.WhodDisplay.SetMode(tc.mode)
			if err := executeCommand(actor, "whod", tc.args, false); err != nil {
				t.Fatalf("executeCommand whod: %v", err)
			}
			if got := strings.Join(drainSessionText(t, actor), ""); got != tc.want {
				t.Fatalf("whod output = %q, want %q", got, tc.want)
			}
		})
	}
}
