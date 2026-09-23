package db

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// persistedCharacter is a character with a non-default value in every field
// C's char_file_u keeps that the players table has no column for.
func persistedCharacter() *game.Player {
	p := game.RestoreCharacterWithStats(7, "Tester", game.ClassThief, game.RaceKender,
		game.CharStats{Str: 13, Int: 14, Wis: 12, Dex: 17, Con: 11, Cha: 10})
	p.Level = 5
	p.SetGold(1234)
	p.BankGold = 500
	p.SetSex(game.SexFemale)
	p.SetAlignment(-350)
	p.SetSkill("kick", 55)
	p.Practices = 4
	p.SetPlrFlag(game.PrfBrief, true)
	p.SetPlrFlag(game.PrfCompact, true)
	p.SetPlrFlag(game.PrfColor1, true)
	p.SetPlrFlag(game.PlrNoshout, true)
	p.AC, p.Hitroll, p.Damroll = 90, 2, 3
	p.Height, p.Weight = 150, 60
	p.Birth, p.PlayedDuration = 1700000000, 3600
	p.WimpLevel, p.InvisLevel, p.FreezeLevel = 10, 0, 0
	p.Kills, p.PKs, p.Deaths = 12, 1, 2
	p.OrigCon = 12
	p.SavingThrows = [5]int{1, 0, -1, 0, 2}
	p.Tattoo, p.TatTimer = 3, 7
	p.MountVNum, p.MountCostDay, p.MountRentTime = 8050, 20, 1700000100
	p.LastDeath = 1700000200
	p.ClanID, p.ClanRank = 2, 3
	p.PoofIn, p.PoofOut = "", ""
	p.AutoGold, p.AutoSplit = true, true
	return p
}

type persistedField struct {
	name string
	get  func(*game.Player) any
}

var persistedFields = []persistedField{
	{"sex", func(p *game.Player) any { return p.GetSex() }},
	{"gold", func(p *game.Player) any { return p.GetGold() }},
	{"bank gold", func(p *game.Player) any { return p.BankGold }},
	{"alignment", func(p *game.Player) any { return p.GetAlignment() }},
	{"skill kick", func(p *game.Player) any { return p.GetSkill("kick") }},
	{"practices", func(p *game.Player) any { return p.Practices }},
	{"flags", func(p *game.Player) any { return p.GetFlags() }},
	{"base AC/hitroll/damroll", func(p *game.Player) any { return [3]int{p.AC, p.Hitroll, p.Damroll} }},
	{"height/weight", func(p *game.Player) any { return [2]int{p.Height, p.Weight} }},
	{"birth/played", func(p *game.Player) any { return [2]int64{p.Birth, p.PlayedDuration} }},
	{"wimp level", func(p *game.Player) any { return p.WimpLevel }},
	{"kills/pks/deaths", func(p *game.Player) any { return [3]int{p.Kills, p.PKs, p.Deaths} }},
	{"orig con", func(p *game.Player) any { return p.OrigCon }},
	{"saving throws", func(p *game.Player) any { return p.SavingThrows }},
	{"tattoo", func(p *game.Player) any { return [2]int{p.Tattoo, p.TatTimer} }},
	{"mount", func(p *game.Player) any { return [3]int64{int64(p.MountVNum), int64(p.MountCostDay), p.MountRentTime} }},
	{"last death", func(p *game.Player) any { return p.LastDeath }},
	{"clan", func(p *game.Player) any { return [2]int{p.ClanID, p.ClanRank} }},
	{"auto gold/split", func(p *game.Player) any { return [2]bool{p.AutoGold, p.AutoSplit} }},
}

func checkPersisted(t *testing.T, want, got *game.Player) {
	t.Helper()
	for _, f := range persistedFields {
		if w, g := f.get(want), f.get(got); w != g {
			t.Errorf("%s: saved %v, loaded %v", f.name, w, g)
		}
	}
}

func testWorld(t *testing.T) *game.World {
	t.Helper()
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 8004, Name: "Altar", Flags: []string{"0", "0", "0", "0"}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// TestCharacterDataRoundTrip: everything C saves survives the record the
// game store writes. Before DP-1314 gold, bank gold, sex, alignment, skills,
// practices and every flag came back at their defaults.
func TestCharacterDataRoundTrip(t *testing.T) {
	want := persistedCharacter()
	rec, err := PlayerToRecord(want, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecordToPlayer(rec, testWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	checkPersisted(t, want, got)
}

// TestCharacterDataThroughTheStore: the same round trip through a real
// store, create then save then load, on every backend the suite runs.
func TestCharacterDataThroughTheStore(t *testing.T) {
	for _, be := range gameStoreBackends(t) {
		t.Run(be.name, func(t *testing.T) {
			store := openGameStore(t, be.dsn)
			want := persistedCharacter()
			rec, err := PlayerToRecord(want, nil)
			if err != nil {
				t.Fatal(err)
			}
			rec.Name, rec.Password = "Persisttester", "hash"
			if err := store.CreatePlayer(rec); err != nil {
				t.Fatal(err)
			}
			want.SetGold(4321) // a later save must overwrite, not only the first
			rec2, err := PlayerToRecord(want, nil)
			if err != nil {
				t.Fatal(err)
			}
			rec2.ID = rec.ID
			if err := store.SavePlayer(rec2); err != nil {
				t.Fatal(err)
			}
			loaded, err := store.GetPlayer("persisttester")
			if err != nil || loaded == nil {
				t.Fatalf("GetPlayer: %v %v", loaded, err)
			}
			got, err := RecordToPlayer(loaded, testWorld(t))
			if err != nil {
				t.Fatal(err)
			}
			checkPersisted(t, want, got)
		})
	}
}

// TestCharacterDataAbsentKeepsColumns: a record saved before the column
// existed ('{}') loads as the columns alone, without error.
func TestCharacterDataAbsentKeepsColumns(t *testing.T) {
	rec, err := PlayerToRecord(persistedCharacter(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rec.CharacterData = []byte("{}")
	got, err := RecordToPlayer(rec, testWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != 5 || got.GetGold() != 0 {
		t.Fatalf("level %d gold %d, want the columns' level 5 and no gold", got.Level, got.GetGold())
	}
}
