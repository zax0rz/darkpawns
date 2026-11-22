package game

import (
	"strings"
	"testing"
)

// TestWhodGreetingMatchesConfigBytes pins SHORT_GREETINGS
// (src/config.c:221-233): 509 bytes, all lines checked
// literally so a transcription slip cannot pass.
func TestWhodGreetingMatchesConfigBytes(t *testing.T) {
	want := "\n\r\n\r" +
		"         (_____)           (_)    (_____)\n\r" +
		"   _     /  __ \\           | |    |  __ \\                            _\n\r" +
		"  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\n\r" +
		"   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\n\r" +
		" .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\n\r" +
		" ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\n\r" +
		"        /                         |||\n\r" +
		"                             |||\n\r" +
		"                 `.'\n\r" +
		"\n\r"
	if got := WhodGreeting(); got != want {
		t.Fatalf("greeting bytes = %q, want C SHORT_GREETINGS %q (src/config.c:221-233)", got, want)
	}
}

func TestWhodHeaderPinsMudName(t *testing.T) {
	if got, want := WhodHeader(), "\n\rA list of active players on Dark Pawns:\n\r\n\r"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
}

// TestWhodBuildWhoListDefaultMode pins the default-mode bytes: header, one
// mortal line, footer (src/whod.c:341-425).
func TestWhodBuildWhoListDefaultMode(t *testing.T) {
	w := NewWhod()
	got := w.BuildWhoList([]WhodEntry{{Name: "Albe", Class: ClassWarrior, Level: 3, Title: "the Test"}})
	want := WhodHeader() + "[ 3 Wa] Albe the Test\r\n" + "\n\rPlayers : 1     Gods : 0\n\r\n\r"
	if got != want {
		t.Fatalf("default-mode bytes =\n%q\nwant\n%q", got, want)
	}
}

// TestWhodBuildWhoListGodTokenLadder pins the wizard ladder and the _GOD_
// fallback when only one of SHOW_LEVEL/SHOW_CLASS is set (src/whod.c:350-371).
func TestWhodBuildWhoListGodTokenLadder(t *testing.T) {
	mode := WhodShowName | WhodShowLevel | WhodShowClass
	for _, tc := range []struct {
		level int
		want  string
	}{
		{40, "*IMP*"},
		{38, "GRGOD"},
		{36, "HIGOD"},
		{35, "_LEG_"},
		{34, " GOD "},
		{32, "TITAN"},
		{31, " IMM "},
	} {
		w := NewWhod()
		w.SetMode(mode)
		got := w.BuildWhoList([]WhodEntry{{Name: "God", Class: ClassWarrior, Level: tc.level}})
		if !strings.Contains(got, "["+tc.want+"] God\r\n") {
			t.Fatalf("level %d token = %q, want %q", tc.level, got, tc.want)
		}
		if !strings.Contains(got, "Gods : 1") {
			t.Fatalf("level %d must count as a god: %q", tc.level, got)
		}
	}
	// Only one of the pair set falls back to _GOD_ (src/whod.c:372).
	w := NewWhod()
	w.SetMode(WhodShowName | WhodShowLevel)
	if got := w.BuildWhoList([]WhodEntry{{Name: "God", Class: ClassWarrior, Level: 40}}); !strings.Contains(got, "[_GOD_] God\r\n") {
		t.Fatalf("single-flag wizard = %q, want the _GOD_ fallback", got)
	}
	// SHOW_WIZLEVEL suppresses the ladder and prints level+class.
	w = NewWhod()
	w.SetMode(WhodShowName | WhodShowLevel | WhodShowClass | WhodShowWizLevel)
	if got := w.BuildWhoList([]WhodEntry{{Name: "God", Class: ClassWarrior, Level: 40}}); !strings.Contains(got, "[40 Wa] God\r\n") {
		t.Fatalf("SHOW_WIZLEVEL = %q, want level and class", got)
	}
}

// TestWhodBuildWhoListInvisibleAndSite pins the IS_INVIS gate (src/whod.c:345)
// and the site column's two forms (src/whod.c:404-411).
func TestWhodBuildWhoListInvisibleAndSite(t *testing.T) {
	w := NewWhod()
	w.SetMode(WhodShowName | WhodShowSite)
	entries := []WhodEntry{
		{Name: "Hidden", Level: 3, IsInvisible: true, Host: "010.000.000.001"},
		{Name: "Seen", Level: 3, Host: ""},
	}
	got := w.BuildWhoList(entries)
	if strings.Contains(got, "Hidden") {
		t.Fatalf("an invisible player leaked without SHOW_INVIS: %q", got)
	}
	if !strings.Contains(got, "Seen [ ** Unknown ** ]\r\n") {
		t.Fatalf("unknown host form = %q", got)
	}

	w.SetMode(WhodShowName | WhodShowSite | WhodShowInvis)
	got = w.BuildWhoList(entries)
	if !strings.Contains(got, "Hidden [010.000.000.001]\r\n") || !strings.Contains(got, "Players : 2     Gods : 0") {
		t.Fatalf("SHOW_INVIS bytes = %q", got)
	}
}

// TestWhodBuildWhoListAFK marker and the name/title/level modes.
func TestWhodBuildWhoListAfkMarker(t *testing.T) {
	w := NewWhod()
	w.SetMode(WhodShowName | WhodShowTitle)
	if got := w.BuildWhoList([]WhodEntry{{Name: "Albe", Level: 3, Title: "the Test", AFK: true}}); !strings.Contains(got, "Albe the Test (AFK)\r\n") {
		t.Fatalf("AFK bytes = %q", got)
	}
	w.SetMode(WhodShowName)
	if got := w.BuildWhoList([]WhodEntry{{Name: "Albe", Level: 3, Title: "the Test", AFK: true}}); !strings.Contains(got, "Albe\r\n") {
		t.Fatalf("name-only bytes = %q", got)
	}
}
