package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type creationCapture struct {
	db.GameStore
	saved []*db.PlayerRecord
}

func (c *creationCapture) SavePlayer(r *db.PlayerRecord) error {
	if err := c.GameStore.SavePlayer(r); err != nil {
		return err
	}
	back, err := c.GetPlayer(r.Name)
	if err == nil {
		c.saved = append(c.saved, back)
	}
	return err
}

func creationProof(t *testing.T) (*Session, *creationCapture) {
	t.Helper()
	t.Setenv("DP_CLOCK", "1")
	t.Setenv("DP_FRESH_MUD", "")
	store := entryDatabase(t)
	entrySeed(t, store, "Founder")
	c := &creationCapture{GameStore: store}
	s := entrySession(t, c)
	s.charName, s.charClass, s.charRace = "Creationproof", game.ClassWarrior, game.RaceHuman
	s.charHometown = 1
	s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 12, Cha: 9}
	dprng.ResetStream(123)
	if err := s.persistAcceptedCharacter(); err != nil {
		t.Fatal(err)
	}
	return s, c
}

func creationRead(t *testing.T, c *creationCapture, w *game.World) *game.Player {
	t.Helper()
	r, err := c.GetPlayer("Creationproof")
	if err != nil || r == nil {
		t.Fatalf("record: %v %v", r, err)
	}
	p, err := db.RecordToPlayer(r, w)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func assertCreationPhase(t *testing.T, p *game.Player, level, hp, maxHP, practices int) {
	t.Helper()
	if p.Level != level || p.Health != hp || p.MaxHealth != maxHP || p.Practices != practices || p.OrigCon != 0 || p.Hometown != 1 {
		t.Fatalf("durable phase: level=%d hp=%d/%d practices=%d orig_con=%d hometown=%d; want %d %d/%d %d 0 1", p.Level, p.Health, p.MaxHealth, p.Practices, p.OrigCon, p.Hometown, level, hp, maxHP, practices)
	}
	if p.NeedsCrashSave() != (level == 1) {
		t.Fatal("creation record crash flag disagrees with C kit/save order")
	}
	if p.GetAutoExit() || p.WimpLevel != 0 || p.GetFlags()&((1<<uint(game.PrfDisphp))|(1<<uint(game.PrfDispmmana))|(1<<uint(game.PrfDispmove))) != 0 {
		t.Fatal("do_start's final preferences leaked into durable phase")
	}
	for _, cond := range []int{game.CondFull, game.CondThirst, game.CondDrunk} {
		if p.GetCondition(cond) != 24 {
			t.Fatalf("condition %d=%d", cond, p.GetCondition(cond))
		}
	}
}

// C init_char/save_char at CON_ROLLABL2 (interpreter.c:2128-2129).
func TestCreationAcceptedSQLiteRecord(t *testing.T) {
	s, c := creationProof(t)
	assertCreationPhase(t, creationRead(t, c, s.manager.world), 0, 0, 0, 0)
}

// C entry save (interpreter.c:2186) followed by advance_level save (class.c:712).
func TestCreationEntryAndAdvanceSQLiteRecords(t *testing.T) {
	s, c := creationProof(t)
	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}
	if len(c.saved) != 2 {
		t.Fatalf("save count=%d, want two C boundaries", len(c.saved))
	}
	for i, r := range c.saved {
		p, err := db.RecordToPlayer(r, s.manager.world)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			assertCreationPhase(t, p, 0, 1, 0, 0)
		} else {
			assertCreationPhase(t, p, 1, 1, s.player.MaxHealth, 2)
		}
	}
	assertCreationPhase(t, creationRead(t, c, s.manager.world), 1, 1, s.player.MaxHealth, 2)
}

// Initial live do_start output and its draw count must survive the save repair.
func TestCreationLiveStateAndDrawsUnchanged(t *testing.T) {
	s, c := creationProof(t)
	// Independently reproduce the old live constructor/do_start path.
	dprng.ResetStream(123)
	want := game.NewCharacterWithStats(0, s.charName, s.charClass, s.charRace, s.charSex, s.charStats)
	want.AdvanceLevel()
	want.Health, want.Mana, want.Move = want.MaxHealth, want.MaxMana, want.MaxMove
	game.GiveStartingSkills(want)
	next := dprng.Next()
	dprng.ResetStream(123)
	// Reconstruct the accepted live candidate so both runs begin at the same draw.
	s.player = game.NewCharacterWithStats(s.player.ID, s.charName, s.charClass, s.charRace, s.charSex, s.charStats)
	s.player.Level, s.player.Exp, s.player.Hometown = 0, 0, 1
	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}
	got := s.player
	if dprng.Next() != next {
		t.Fatal("creation persistence consumed RNG draws")
	}
	if got.Health != want.Health || got.MaxHealth != want.MaxHealth || got.Mana != want.Mana || got.Move != want.Move || got.Practices != want.Practices || got.OrigCon != want.OrigCon || got.Weight != want.Weight || got.Height != want.Height || !got.GetAutoExit() || got.WimpLevel != 5 || got.GetCondition(game.CondFull) != 36 || got.GetCondition(game.CondThirst) != 36 {
		t.Fatalf("live state changed: got hp=%d/%d practices=%d orig_con=%d; want %d/%d %d %d", got.Health, got.MaxHealth, got.Practices, got.OrigCon, want.Health, want.MaxHealth, want.Practices, want.OrigCon)
	}
	// Also ensure the live state was not preserved by leaving the old record.
	assertCreationPhase(t, creationRead(t, c, s.manager.world), 1, 1, got.MaxHealth, 2)
}

// A fresh login, without cleanup/autosave, restores C's intermediate record.
func TestCreationReconnectDurableValues(t *testing.T) {
	s, c := creationProof(t)
	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}
	fresh := entrySession(t, c)
	fresh.player = s.player
	fresh.authenticated = true
	if err := fresh.reloadCharacterFromStore(); err != nil {
		t.Fatal(err)
	}
	assertCreationPhase(t, fresh.player, 1, 1, s.player.MaxHealth, 2)
	if err := fresh.enterReturningPlayer(); err != nil {
		t.Fatal(err)
	}
	if fresh.player.GetOrigCon() != 14 {
		t.Fatal("C entry sanity assignment missing")
	}
	// The entry save precedes the sanity assignment; the next save has not run.
	assertCreationPhase(t, creationRead(t, c, fresh.manager.world), 1, 1, s.player.MaxHealth, 2)
	if s.player.Health != s.player.MaxHealth || s.player.Practices != 4 || s.player.OrigCon != 14 {
		t.Fatal("fresh restore mutated original live character")
	}
}
