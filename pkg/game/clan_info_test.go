package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// This test pins the complete detail transcript, including war state, money
// fields, application level, and C's CRLF+LF plan ending (src/clan.c:758-783).
func TestDoClanInfoIncludesDefinedCDetails(t *testing.T) {
	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.Clans = NewClanManager()

	c := &Clan{
		ID:        1,
		Name:      "Wolves",
		Ranks:     2,
		Members:   3,
		Power:     4,
		Treasure:  5,
		AppFee:    6,
		Dues:      7,
		ApplLevel: 8,
		Plan:      "Mortal member plan.",
	}
	c.RankName[0] = "Member"
	c.RankName[1] = "Leader"
	c.Spells[0] = 17
	for i := range NumCP {
		c.Privilege[i] = 2
	}
	w.Clans.AddClan(c)

	p := NewPlayer(1, "ClanTester", 1)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var output strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == p.Name {
			output.Write(msg)
		}
	}

	w.doClanInfo(p, "Wolves")

	want := "Info for the clan Wolves :\r\n" +
		"Ranks      : 2\r\nTitles     : Member Leader \r\n" +
		"Members    : 3\r\nPower      : 4\t\nTreasure   : 5\r\n" +
		"Spells     : 17 \r\n" +
		"Clan privileges:\r\n" +
		"   setplan   : 2\r\n   enroll    : 2\r\n   expel     : 2\r\n" +
		"   promote   : 2\r\n   demote    : 2\r\n   setfees   : 2\r\n" +
		"   withdraw  : 2\r\n   setapplev : 2\r\n\r\n" +
		"Description:\r\nMortal member plan.\r\n\n" +
		"This clan is at peace with all others.\r\n" +
		"Application fee  : 6 gold\r\nMonthly Dues     : 7 gold\r\n" +
		"Application level: 8\r\n"
	if got := output.String(); got != want {
		t.Fatalf("clan info output = %q, want %q", got, want)
	}

	c.AtWar[0] = 1
	output.Reset()
	w.doClanInfo(p, "Wolves")
	wantWar := strings.Replace(want, "This clan is at peace with all others.", "This clan is at war.", 1)
	if got := output.String(); got != wantWar {
		t.Fatalf("clan info at-war output = %q, want %q", got, wantWar)
	}
}

func TestDoClanInfoListIncludesDefinedCHeading(t *testing.T) {
	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.Clans = NewClanManager()
	w.Clans.AddClan(&Clan{ID: 1, Name: "Wolves", Members: 1, Power: 1, ApplLevel: 8})

	p := NewPlayer(1, "ClanTester", 1)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var output strings.Builder
	w.MessageSink = func(name string, msg []byte) {
		if name == p.Name {
			output.Write(msg)
		}
	}

	w.doClanInfo(p, "")

	if got := output.String(); !strings.HasPrefix(got, "\t\t\tooO Clans of Dark Pawns Ooo\r\n") {
		t.Fatalf("clan list heading missing the C bytes: %q", got)
	} else if got[0] != '\t' {
		// C writes "\r" and then overwrites that buffer with the heading
		// (clan.c:736-737), so the first byte a player sees is the first tab.
		t.Fatalf("clan list heading starts with %q, want a tab", got[0])
	}
}
