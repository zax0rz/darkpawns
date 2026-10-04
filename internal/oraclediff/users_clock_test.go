package oraclediff

import (
	"strings"
	"testing"
)

const usersClockHeader = "Num Class   Name         State          Idl Login@   Site\r\n--- ------- ------------ -------------- --- -------- ------------------------\r\n"

func TestNormalizeUsersLoginAtOnlyClock(t *testing.T) {
	rows := []string{
		"  1 [40 Wa] InformativeResidual Playing          0 19:53:49 [127.000.000.001]\r\n",
		" 12 [ 1 Mu] Short        Get name            19:53:49 [Hostname unknown]\r\n",
		"  2 [34 Th] Switchedname Switched        125 19:53:49 [clock-12:34:56.example]\r\n",
		"  3    -    UNDEFINED    Get password        19:53:49 [127.000.000.001]\r\n",
	}
	for _, row := range rows {
		c := usersClockHeader + row + "\r\n1 visible sockets connected.\r\n> "
		g := strings.Replace(c, "19:53:49", "19:53:50", 1)
		for name, normalize := range map[string]func(string) string{
			"ordinary": Normalize, "ansi": NormalizeKeepANSI, "prompts": NormalizeKeepPrompts,
		} {
			t.Run(name+row, func(t *testing.T) {
				if normalize(c) != normalize(g) {
					t.Fatalf("Login@ wall-clock difference survived: C=%q Go=%q", normalize(c), normalize(g))
				}
			})
		}
		want := strings.Replace(c, "19:53:49", "<LOGIN@>", 1)
		if got := NormalizeKeepPrompts(c); got != want {
			t.Fatalf("changed non-clock bytes: got %q want %q", got, want)
		}
	}
}

func TestNormalizeUsersLoginAtPreservesOtherFields(t *testing.T) {
	row := "  1 [40 Wa] InformativeResidual Playing          0 19:53:49 [127.000.000.001]\r\n"
	raw := usersClockHeader + row + "\r\n1 visible sockets connected.\r\n"
	for _, replacement := range [][2]string{
		{"  1 [", "  2 ["},
		{"[40 Wa]", "[39 Wa]"},
		{"[40 Wa]", "[40 Mu]"},
		{"InformativeResidual", "Differentname"},
		{"Playing", "Switched"},
		{"          0 ", "          1 "},
		{"127.000.000.001", "127.000.000.002"},
		{"1 visible sockets", "2 visible sockets"},
		{"19:53:49", "99:53:49"},
	} {
		changed := strings.Replace(raw, replacement[0], replacement[1], 1)
		if changed == raw {
			t.Fatal("mutation missed its input")
		}
		for name, normalize := range map[string]func(string) string{"ordinary": Normalize, "ansi": NormalizeKeepANSI, "prompts": NormalizeKeepPrompts} {
			if normalize(raw) == normalize(changed) {
				t.Errorf("%s hid other users bytes: %v", name, replacement)
			}
		}
	}
}

func TestNormalizeUsersLoginAtRequiresTableContext(t *testing.T) {
	row := "  1 [40 Wa] InformativeResidual Playing          0 19:53:49 [127.000.000.001]\r\n"
	for _, raw := range []string{
		row, "The clock reads 19:53:49.\r\n", "Someone says '19:53:49'.\r\n",
		strings.Replace(usersClockHeader, "Login@", "Wrong@", 1) + row,
		strings.Replace(usersClockHeader, "--- -------", "-- --------", 1) + row,
		usersClockHeader + "\r\n" + row,
	} {
		if got := NormalizeKeepPrompts(raw); got != raw {
			t.Fatalf("masked clock outside valid users row: %q -> %q", raw, got)
		}
	}
}

func TestNormalizeUsersLoginAtPreservesANSIAndPromptFraming(t *testing.T) {
	raw := "\x1b[32m" + usersClockHeader + "\x1b[0m  1 [40 Wa] Longname Playing 0 19:53:49 [127.000.000.001]\x1b[0m\r\n\r\n> "
	want := strings.Replace(raw, "19:53:49", "<LOGIN@>", 1)
	if got := NormalizeKeepPrompts(raw); got != want {
		t.Fatalf("raw framing/colors changed: got %q want %q", got, want)
	}
	if got := NormalizeKeepANSI(raw); !strings.Contains(got, "\x1b[32m") || !strings.Contains(got, "<LOGIN@>") {
		t.Fatalf("ANSI clock normalization=%q", got)
	}
}
