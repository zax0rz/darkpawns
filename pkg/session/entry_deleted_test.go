package session

import (
	"bytes"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/interpreter.c:1768-1787: a deleted saved record starts fresh creation.
func TestEntryDeletedRecordStartsFresh(t *testing.T) {
	for _, route := range []string{"json", "terminal", "retry"} {
		t.Run(route, func(t *testing.T) {
			database := entryDatabase(t)
			old := entrySeed(t, database, "Reclaim")
			p := game.NewCharacter(old.ID, old.Name, game.ClassThief, game.RaceKender)
			p.SetPlrFlag(game.PlrDeleted, true)
			raw, err := game.EncodeCharacterData(p)
			if err != nil {
				t.Fatal(err)
			}
			old.CharacterData = raw
			if err := database.SavePlayer(old); err != nil {
				t.Fatal(err)
			}
			s := entrySession(t, database)
			switch route {
			case "json":
				err = s.handleLogin(loginMsg("RECLAIM", "oldpassword"))
			case "terminal":
				s.TerminalLine("RECLAIM")
			case "retry":
				s.charCreating = true
				s.charStage = "get_name"
				err = entryInput(s, "RECLAIM")
			}
			if err != nil {
				t.Fatal(err)
			}
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			want := "Please remember to choose an appropriate fantasy-oriented name.\r\nDid I get that right, Reclaim (Y/N)? "
			if s.charStage != "confirm_name" || s.charName != "Reclaim" || s.player != nil || s.authenticated || prompt.Secret || prompt.Prompt != want {
				t.Fatalf("deleted identity did not start fresh: stage=%q name=%q prompt=%+v", s.charStage, s.charName, prompt)
			}
			unchanged, err := database.GetPlayer("reclaim")
			if err != nil || unchanged == nil || unchanged.ID != old.ID || !bytes.Equal(unchanged.CharacterData, raw) {
				t.Fatal("name lookup changed deleted durable record")
			}
		})
	}
}

// src/interpreter.c:2144-2147 saves only after stats acceptance; db.c:3057
// allocates a new identity. SQLite replaces the tombstone in one transaction.
func TestEntryDeletedReplacementLifecycle(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Keeper")
	old := entrySeed(t, database, "Reclaim")
	p := game.NewCharacter(old.ID, old.Name, game.ClassThief, game.RaceKender)
	p.SetPlrFlag(game.PlrDeleted, true)
	raw, err := game.EncodeCharacterData(p)
	if err != nil {
		t.Fatal(err)
	}
	old.CharacterData = raw
	old.Description = "old description"
	old.Level = 45
	if err := database.SavePlayer(old); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("RECLAIM", "")); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"Y", "freshpass", "freshpass", "N", "F", "H", "W", "K"} {
		if err := entryInput(s, input); err != nil {
			t.Fatal(err)
		}
	}
	before, err := database.GetPlayer("Reclaim")
	if err != nil || before == nil || before.ID != old.ID || before.Password != old.Password {
		t.Fatal("deleted row changed before accepted stats")
	}
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	fresh, err := database.GetPlayer("reclaim")
	if err != nil || fresh == nil {
		t.Fatalf("replacement missing: %v", err)
	}
	if fresh.ID <= old.ID || fresh.Name != "Reclaim" || fresh.Level != 0 || fresh.Description != "" || fresh.Password == old.Password || game.CharacterDataDeleted(fresh.CharacterData) {
		t.Fatalf("old identity leaked into replacement: %+v", fresh)
	}
	if n, err := database.CountPlayers(); err != nil || n != 2 {
		t.Fatalf("replacement row count=%d err=%v", n, err)
	}
	if !s.authenticated || !s.creationSaved || s.player.ID != fresh.ID {
		t.Fatal("replacement was not adopted at accepted stats")
	}
	s.CloseSend()
	next := entrySession(t, database)
	if err := next.handleLogin(loginMsg("RECLAIM", "")); err != nil {
		t.Fatal(err)
	}
	if next.charStage != "login_password" {
		t.Fatal("fresh replacement is still treated as deleted")
	}
}

func TestEntryDeletedAbandonKeepsRecord(t *testing.T) {
	database := entryDatabase(t)
	old := entrySeed(t, database, "Reclaim")
	p := game.NewCharacter(old.ID, old.Name, game.ClassThief, game.RaceKender)
	p.SetPlrFlag(game.PlrDeleted, true)
	raw, err := game.EncodeCharacterData(p)
	if err != nil {
		t.Fatal(err)
	}
	old.CharacterData = raw
	if err := database.SavePlayer(old); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Reclaim", "")); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(s, "N"); err != nil {
		t.Fatal(err)
	}
	if s.creationReplacement != nil {
		t.Fatal("N retained deleted replacement ownership")
	}
	if err := entryInput(s, "Othername"); err != nil {
		t.Fatal(err)
	}
	if s.creationReplacement != nil {
		t.Fatal("N retained deleted replacement ownership")
	}
	s.CloseSend()
	unchanged, err := database.GetPlayer("Reclaim")
	if err != nil || unchanged == nil || unchanged.ID != old.ID || unchanged.Password != old.Password || !bytes.Equal(unchanged.CharacterData, raw) {
		t.Fatal("abandoned replacement destroyed deleted record")
	}
}

func TestEntryDeletedMenuReuse(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Keeper")
	old := entrySeed(t, database, "Reclaim")
	old.Inventory, old.Equipment = []byte("[14425]"), []byte(`{"body":14425}`)
	if err := database.SavePlayer(old); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Reclaim", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"", "5", "oraclepass", "yes"} {
		if err := entryInput(s, input); err != nil {
			t.Fatal(err)
		}
	}
	if !s.SendClosed() {
		t.Fatal("menu deletion did not close")
	}
	if rec, err := database.GetPlayer("Reclaim"); err != nil || (rec == nil || !game.CharacterDataDeleted(rec.CharacterData)) {
		t.Fatal("menu deletion did not retain C deleted marker")
	} else if string(rec.Inventory) != "[]" || string(rec.Equipment) != "{}" {
		t.Fatal("menu deletion retained crash objects")
	}
	next := entrySession(t, database)
	if err := next.handleLogin(loginMsg("RECLAIM", "")); err != nil {
		t.Fatal(err)
	}
	if next.charStage != "confirm_name" || next.charName != "Reclaim" {
		t.Fatal("deleted menu name did not become fresh creation")
	}
	for _, input := range []string{"Y", "freshpass", "freshpass", "N", "F", "H", "W", "K", "Y"} {
		if err := entryInput(next, input); err != nil {
			t.Fatal(err)
		}
	}
	fresh, err := database.GetPlayer("reclaim")
	if err != nil || fresh == nil || fresh.ID <= old.ID || fresh.Password == old.Password {
		t.Fatal("menu reuse kept old durable identity")
	}
}

func TestEntryDeletedReplacementFailureCloses(t *testing.T) {
	database := entryDatabase(t)
	old := entrySeed(t, database, "Reclaim")
	p := game.NewCharacter(old.ID, old.Name, game.ClassThief, game.RaceKender)
	p.SetPlrFlag(game.PlrDeleted, true)
	raw, err := game.EncodeCharacterData(p)
	if err != nil {
		t.Fatal(err)
	}
	old.CharacterData = raw
	if err := database.SavePlayer(old); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("CREATE TRIGGER reject_replacement BEFORE INSERT ON players BEGIN SELECT RAISE(ABORT,'replacement refused'); END"); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Reclaim", "")); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"Y", "freshpass", "freshpass", "N", "F", "H", "W", "K", "Y"} {
		if err := entryInput(s, input); err != nil {
			t.Fatal(err)
		}
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.creationSaved {
		t.Fatal("failed replacement left an enterable candidate")
	}
	got, err := database.GetPlayer("Reclaim")
	if err != nil || got == nil || got.ID != old.ID || got.Password != old.Password {
		t.Fatal("failed creation destroyed deleted record")
	}
}

func TestEntryDeletedMenuLevelGate(t *testing.T) {
	for _, level := range []int{game.LVL_GRGOD - 1, game.LVL_GRGOD, game.LVL_GRGOD + 1} {
		database := entryDatabase(t)
		old := entrySeed(t, database, "Reclaim")
		old.Level = level
		if err := database.SavePlayer(old); err != nil {
			t.Fatal(err)
		}
		s := entrySession(t, database)
		if err := s.handleLogin(loginMsg("Reclaim", "oraclepass")); err != nil {
			t.Fatal(err)
		}
		for _, input := range []string{"", "5", "oraclepass", "yes"} {
			if err := entryInput(s, input); err != nil {
				t.Fatal(err)
			}
		}
		got, err := database.GetPlayer("Reclaim")
		if err != nil || got == nil {
			t.Fatal("menu deletion removed C playerfile row")
		}
		if game.CharacterDataDeleted(got.CharacterData) != (level < game.LVL_GRGOD) {
			t.Fatalf("level %d deleted flag wrong", level)
		}
		next := entrySession(t, database)
		if err := next.handleLogin(loginMsg("RECLAIM", "")); err != nil {
			t.Fatal(err)
		}
		want := "login_password"
		if level < game.LVL_GRGOD {
			want = "confirm_name"
		}
		if next.charStage != want {
			t.Fatalf("level %d reused name stage=%s want=%s", level, next.charStage, want)
		}
	}
}
