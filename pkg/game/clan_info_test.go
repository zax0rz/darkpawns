package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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
		ApplLevel: 8,
		Plan:      "Mortal member plan.",
	}
	c.RankName[0] = "Member"
	c.RankName[1] = "Leader"
	c.Spells[0] = 17
	for i := 0; i < NumCP; i++ {
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

	got := output.String()
	for _, want := range []string{
		"Ranks      : 2\r\nTitles     : Member Leader ",
		"Members    : 3\r\nPower      : 4\t\nTreasure   : 5",
		"Spells     : 17 ",
		"Clan privileges:\r\n   setplan   : 2",
		"Description:\r\nMortal member plan.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("clan info missing %q:\n%q", want, got)
		}
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
