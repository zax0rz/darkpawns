package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShowPlayerDurableReport(t *testing.T) {
	e := newRecoveryEnv(t)
	p := e.s.player
	p.Sex = game.SexFemale
	p.Class = 3
	p.Level = 12
	p.Gold = 123
	p.BankGold = 456
	p.Exp = 789
	p.Alignment = -12
	p.Practices = 7
	p.Birth = 946684800
	p.PlayedDuration = 3*3600 + 47*60
	r, err := db.PlayerToRecord(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.LastLogon = 978307200
	// Even invalid object payloads cannot interfere with a scalar report.
	r.Inventory = []byte(`[{"vnum":7101}]`)
	r.Equipment = []byte(`[]`)
	if err := e.store.SavePlayer(r); err != nil {
		t.Fatal(err)
	}
	p.Gold = 9999
	p.Level = 40
	actor := makeCommandTestSession(t, e.m, "Showgod", 40, 1001)
	objectsBefore := len(e.world.GetAllObjects())
	for range 2 {
		if err := cmdShow(actor, []string{"player", "recoverer"}); err != nil {
			t.Fatal(err)
		}
		birth := time.Unix(946684800, 0).Local().Format("Mon Jan _2 15:04")
		last := time.Unix(978307200, 0).Local().Format("Mon Jan _2 15:04")
		got := renderedOutput(actor)
		expected := fmt.Sprintf("Player: %-12s (Female) [12 %s]\r\nAu: 123       Bal: 456       Exp: 789       Align: -12    Lessons: 7  \r\nStarted: %-20s  Last: %-20s  Played:   3h 47m\r\n", "Recoverer", game.ClassAbbrevs[3], birth, last)
		if got != expected {
			t.Fatalf("durable report=%q want=%q", got, expected)
		}
	}
	if len(e.world.GetAllObjects()) != objectsBefore {
		t.Fatal("report allocated objects")
	}
}

func TestPasswordChangeUpdatesCharacterSaveMetadata(t *testing.T) {
	e := newRecoveryEnv(t)
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	r.LastLogon = 123
	if err := e.store.SavePlayer(r); err != nil {
		t.Fatal(err)
	}
	e.s.player.Gold = 333
	e.s.player.PlayedDuration = 7200
	e.s.menuNewPasswordHash = "new-hash"
	if err := e.s.persistChangedPassword(); err != nil {
		t.Fatal(err)
	}
	saved, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	scalar := *saved
	scalar.Inventory, scalar.Equipment = nil, nil
	p, err := db.RecordToPlayer(&scalar, nil)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Password != "new-hash" || saved.LastLogon == 123 || saved.LastLogon != e.s.player.GetLastLogon() || p.Gold != 333 || p.PlayedDuration < 7200 {
		t.Fatalf("password confirmation omitted character save: %+v gold=%d played=%d", saved, p.Gold, p.PlayedDuration)
	}
}

func TestMenuDeleteUpdatesCharacterSaveMetadata(t *testing.T) {
	t.Chdir(t.TempDir())
	e := newRecoveryEnv(t)
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	r.LastLogon = 123
	if err := e.store.SavePlayer(r); err != nil {
		t.Fatal(err)
	}
	if err := e.s.confirmDelete("yes"); err != nil {
		t.Fatal(err)
	}
	saved, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	if saved.LastLogon <= 123 {
		t.Fatal("delete omitted C character save timestamp")
	}
}
