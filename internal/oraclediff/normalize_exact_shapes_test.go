package oraclediff

import (
	"strings"
	"testing"
)

func TestExactShapeWallClock(t *testing.T) {
	// C asctime/ctime (src/act.wizard.c:1814-1826, src/mail.c:429):
	// "%.3s %.3s%3d %.2d:%.2d:%.2d %d" — day 1-9 has two spaces after month,
	// day 10-31 has one space.
	valid := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single-digit day two spaces",
			in:   "Current machine time: Thu Nov  5 09:07:03 2026\r\n",
			want: "Current machine time: <WALL_CLOCK>\n",
		},
		{
			name: "two-digit day one space",
			in:   "Up since Sat Nov 21 14:35:09 2026: 0 days, 0:00\r\n",
			want: "Up since <WALL_CLOCK>: 0 days, 0:00\n",
		},
		{
			name: "mail date line single-digit day",
			in:   "Date: Sun Sep  6 11:44:17 2026\r\n",
			want: "Date: <WALL_CLOCK>\n",
		},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	invalid := []struct {
		name string
		in   string
	}{
		{"single-digit day one space (PF-151)", "Current machine time: Thu Nov 5 09:07:03 2026"},
		{"mail date single-digit day one space (PF-151)", "Date: Thu Nov 5 09:07:03 2026"},
		{"single-digit day zero-padded", "Current machine time: Thu Nov 05 09:07:03 2026"},
		{"single-digit day three spaces", "Current machine time: Thu Nov   5 09:07:03 2026"},
		{"two-digit day two spaces", "Up since Sat Nov  21 14:35:09 2026: 0 days, 0:00"},
		{"invalid day 32", "Date: Sat Nov 32 14:35:09 2026"},
		{"invalid hour 25", "Date: Sat Nov 21 25:35:09 2026"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			got := Normalize(tc.in + "\r\n")
			if strings.Contains(got, "<WALL_CLOCK>") || got != tc.in+"\n" {
				t.Fatalf("wrong shape was masked or dropped: got %q, want %q", got, tc.in+"\n")
			}
		})
	}
}

func TestExactShapeLastDeathStamp(t *testing.T) {
	// C src/act.wizard.c:781-784: first 16 chars of asctime.
	valid := []struct {
		name string
		in   string
	}{
		{"single-digit day", "Last Death: [Thu Nov  5 09:07]"},
		{"two-digit day", "Last Death: [Mon Oct 26 18:07]"},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			want := "Last Death: [<WALL_CLOCK>]\n"
			if got := Normalize(tc.in + "\r\n"); got != want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, want)
			}
		})
	}

	invalid := []struct {
		name string
		in   string
	}{
		{"single-digit day one space", "Last Death: [Thu Nov 5 09:07]"},
		{"single-digit day zero-padded", "Last Death: [Thu Nov 05 09:07]"},
		{"two-digit day two spaces", "Last Death: [Mon Oct  26 18:07]"},
		{"invalid hour 29", "Last Death: [Mon Oct 26 29:07]"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			got := Normalize(tc.in + "\r\n")
			if got != tc.in+"\n" {
				t.Fatalf("wrong Last Death shape was masked: got %q, want %q", got, tc.in+"\n")
			}
		})
	}
}

func TestExactShapePlayerSavedDates(t *testing.T) {
	// C src/act.wizard.c:2342-2346:
	// "%sStarted: %-20.16s  Last: %-20.16s  Played: %3dh %2dm\r\n"
	// 16 ctime chars padded to 20 (+4 spaces) + 2 literal spaces = 6 spaces.
	valid := "Started: Sun Oct  4 22:59      Last: Mon Oct 12 00:01      Played:   3h 47m\r\n"
	want := "Started: <WALL_CLOCK>      Last: <WALL_CLOCK>      Played:   3h 47m\n"
	if got := Normalize(valid); got != want {
		t.Fatalf("Normalize(valid) = %q, want %q", got, want)
	}
	wantKP := "Started: <WALL_CLOCK>      Last: <WALL_CLOCK>      Played:   3h 47m\r\n"
	if got := NormalizeKeepPrompts(valid); got != wantKP {
		t.Fatalf("NormalizeKeepPrompts(valid) = %q, want %q", got, wantKP)
	}

	invalid := []struct {
		name string
		in   string
	}{
		{"one-space single-digit day in Started", "Started: Sun Oct 4 22:59      Last: Mon Oct 12 00:01      Played:   3h 47m"},
		{"one-space single-digit day in Last", "Started: Sun Oct  4 22:59      Last: Mon Oct 5 00:01      Played:   3h 47m"},
		{"five spaces before Last", "Started: Sun Oct  4 22:59     Last: Mon Oct 12 00:01      Played:   3h 47m"},
		{"seven spaces before Last", "Started: Sun Oct  4 22:59       Last: Mon Oct 12 00:01      Played:   3h 47m"},
		{"five spaces before Played", "Started: Sun Oct  4 22:59      Last: Mon Oct 12 00:01     Played:   3h 47m"},
		{"seven spaces before Played", "Started: Sun Oct  4 22:59      Last: Mon Oct 12 00:01       Played:   3h 47m"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			if got := Normalize(tc.in + "\r\n"); got != tc.in+"\n" {
				t.Fatalf("Normalize hid wrong shape: got %q, want %q", got, tc.in+"\n")
			}
			if got := NormalizeKeepPrompts(tc.in + "\r\n"); got != tc.in+"\r\n" {
				t.Fatalf("NormalizeKeepPrompts hid wrong shape: got %q, want %q", got, tc.in+"\r\n")
			}
		})
	}
}

func TestExactShapeCompileTimeStamp(t *testing.T) {
	// C src/version.c:57-65: "%s %d, %d   %02d:%02d" — unpadded day 1..31.
	valid := "Compile Time: February 5, 2026   09:07\r\n"
	if got := Normalize(valid); got != "Compile Time: <COMPILE_TIME>\n" {
		t.Fatalf("Normalize(valid) = %q", got)
	}

	invalid := []struct {
		name string
		in   string
	}{
		{"zero-padded day", "Compile Time: October 01, 2026   09:27"},
		{"space-padded day", "Compile Time: October  1, 2026   09:27"},
		{"invalid day 32", "Compile Time: October 32, 2026   09:27"},
		{"invalid hour 24", "Compile Time: October 1, 2026   24:00"},
		{"invalid minute 60", "Compile Time: October 1, 2026   09:60"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			if got := Normalize(tc.in + "\r\n"); got != tc.in+"\n" {
				t.Fatalf("wrong Compile Time shape was masked: got %q, want %q", got, tc.in+"\n")
			}
		})
	}
}

func TestExactShapeNoVolatileLineDrops(t *testing.T) {
	// None of the old volatileLine alternatives may drop a line.
	// Current machine time stays in the comparison with its timestamp masked.
	if got, want := Normalize("Current machine time: Sat Nov 21 09:10:11 2026\r\n"), "Current machine time: <WALL_CLOCK>\n"; got != want {
		t.Fatalf("Current machine time dropped or mis-masked: got %q, want %q", got, want)
	}
	// Static C lines and invented Go lines that volatileLine used to drop must survive verbatim.
	kept := []struct {
		name string
		line string
	}{
		{"invented server time", "Server time: 12:00:00"},
		{"invented uptime colon", "Uptime: 3 hours"},
		{"invented players online", "Players online: 7"},
		{"static greeting pwipe line (src/config.c:253)", "   As of 10-17-2008 there has been a pwipe.  Enjoy your new adventures!"},
		{"static motd beta line (lib/text/motd:7)", "**** Dark Pawns 3.0 beta is live at darkpawns.com 4355"},
		{"invented motd updated", "MOTD updated: 2026-11-21"},
		{"invented version colon", "Version: 1.0.0"},
		{"invented vitals slash", "Player HP: 22/22"},
		{"leading space before auto-exits", " [ Exits: north ]"},
	}
	for _, tc := range kept {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.line + "\r\n"); got != tc.line+"\n" {
				t.Fatalf("line was dropped or altered: got %q, want %q", got, tc.line+"\n")
			}
			if got := NormalizeKeepANSI(tc.line + "\r\n"); got != tc.line+"\n" {
				t.Fatalf("KeepANSI dropped or altered line: got %q, want %q", got, tc.line+"\n")
			}
		})
	}
}

func TestExactShapeUsersLoginRowClock(t *testing.T) {
	// C src/act.informative.c:2061-2096: "%-8s " followed immediately by "[%s]\r\n".
	validRow := "  1 [40 Wa] InformativeResidual Playing          0 19:53:49 [127.000.000.001]\r\n"
	valid := usersClockHeader + validRow
	wantValid := usersClockHeader + strings.Replace(validRow, "19:53:49", "<LOGIN@>", 1)
	if got := NormalizeKeepPrompts(valid); got != wantValid {
		t.Fatalf("NormalizeKeepPrompts(valid) = %q, want %q", got, wantValid)
	}

	invalidRows := []struct {
		name string
		row  string
	}{
		{"two spaces before host bracket", "  1 [40 Wa] InformativeResidual Playing          0 19:53:49  [127.000.000.001]\r\n"},
		{"tab before host bracket", "  1 [40 Wa] InformativeResidual Playing          0 19:53:49\t[127.000.000.001]\r\n"},
		{"trailing space after host bracket", "  1 [40 Wa] InformativeResidual Playing          0 19:53:49 [127.000.000.001] \r\n"},
		{"single-digit hour", "  1 [40 Wa] InformativeResidual Playing          0  9:53:49 [127.000.000.001]\r\n"},
	}
	for _, tc := range invalidRows {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			raw := usersClockHeader + tc.row
			if got := NormalizeKeepPrompts(raw); got != raw {
				t.Fatalf("wrong users row shape was masked: got %q, want %q", got, raw)
			}
		})
	}
}

func TestExactShapeStatsLine(t *testing.T) {
	// C src/interpreter.c:2109-2110, :2137-2138 with abil_names[] (src/constants.c:61-90):
	// "  Str: %-13s Dex: %-13s Int: %-13s\r\n"
	// "  Wis: %-13s Con: %-13s Cha: %-13s\r\n"
	valid := []struct {
		name string
		in   string
	}{
		{
			name: "standard rolled block with exact 13-byte fields",
			in: "  Str: very good     Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "all remaining abil_names entries",
			in: "  Str: terrible      Dex: awful         Int: bad          \r\n" +
				"  Wis: below average Con: excellent     Cha: godlike      \r\n",
		},
		{
			name: "superior entry",
			in: "  Str: superior      Dex: below average Int: below average\r\n" +
				"  Wis: godlike       Con: terrible      Cha: awful        \r\n",
		},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); got != "<ROLLED_STATS>\n" {
				t.Fatalf("Normalize(valid) = %q, want %q", got, "<ROLLED_STATS>\n")
			}
		})
	}

	// Two consecutive valid blocks must produce two <ROLLED_STATS> tokens, not collapse to one.
	t.Run("two_consecutive_blocks", func(t *testing.T) {
		twoBlocks := valid[0].in + valid[1].in
		if got := Normalize(twoBlocks); got != "<ROLLED_STATS>\n<ROLLED_STATS>\n" {
			t.Fatalf("two consecutive blocks collapsed: got %q, want %q", got, "<ROLLED_STATS>\n<ROLLED_STATS>\n")
		}
	})

	invalid := []struct {
		name string
		in   string
	}{
		{
			name: "one leading space instead of two",
			in: " Str: very good     Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "three leading spaces instead of two",
			in: "   Str: very good     Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "wrong pad width on Str column",
			in: "  Str: good        Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "missing trailing pad spaces on Int column",
			in: "  Str: very good     Dex: good          Int: average\r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "missing trailing pad spaces on Cha column",
			in: "  Str: very good     Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor\r\n",
		},
		{
			name: "unknown adjective not in abil_names",
			in: "  Str: amazing       Dex: good          Int: average      \r\n" +
				"  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "numeric stats instead of adjectives",
			in: "  Str: 18/00         Dex: 14            Int: 12           \r\n" +
				"  Wis: 11            Con: 15            Cha: 9            \r\n",
		},
		{
			name: "lone Str line without Wis line",
			in:   "  Str: very good     Dex: good          Int: average      \r\n",
		},
		{
			name: "lone Wis line without Str line",
			in:   "  Wis: decent        Con: good          Cha: poor         \r\n",
		},
		{
			name: "swapped Wis before Str",
			in: "  Wis: decent        Con: good          Cha: poor         \r\n" +
				"  Str: very good     Dex: good          Int: average      \r\n",
		},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			want := strings.ReplaceAll(tc.in, "\r\n", "\n")
			if got := Normalize(tc.in); got != want {
				t.Fatalf("wrong statsLine shape was masked: got %q, want %q", got, want)
			}
		})
	}
}
