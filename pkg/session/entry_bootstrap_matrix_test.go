package session

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func bootstrapWorld(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{{VNum: game.ImmortStartRoom, Name: "Immortal room"}, {VNum: game.NewbieStartRoom, Name: "Newbie room"}, {VNum: game.MortalStartRoom, Name: "Mortal room"}}}
	for _, vnum := range []int{8038, 8027, 8036, 1239, 8037, 8023, 8019, 8010, 8063} {
		kind := 0
		if vnum == 8038 {
			kind = 15
		}
		parsed.Objs = append(parsed.Objs, parser.Obj{VNum: vnum, Keywords: fmt.Sprint(vnum), ShortDesc: fmt.Sprint(vnum), TypeFlag: kind, Values: [4]int{100, 0, 0, 0}})
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// Complete init_char/do_start boundary: src/db.c:2976-2989,3006-3078;
// src/class.c:501-591,599-712; src/handler.c:559-571,939-951.
func TestEntryBootstrapMatrix(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		for _, god := range []bool{false, true} {
			for _, class := range []int{game.ClassMageUser, game.ClassCleric, game.ClassThief, game.ClassWarrior, game.ClassNinja, game.ClassPsionic} {
				t.Run(fmt.Sprintf("%d/god=%v/class=%d", seed, god, class), func(t *testing.T) {
					t.Setenv("JWT_SECRET", "entry-bootstrap-test-jwt-secret-at-least-32")
					t.Setenv("DP_CLOCK", "1")
					t.Setenv("DP_FRESH_MUD", "")
					if god {
						t.Setenv("DP_FRESH_MUD", "1")
					}
					s := makeCharSession(t, newTestManager(t, bootstrapWorld(t), nil))
					s.charName = "Boothero"
					s.charClass = class
					s.charRace = game.RaceHuman
					s.charHometown = 1
					s.charStats = game.CharStats{Str: 14, Dex: 14, Con: 14, Wis: 14, Int: 14, Cha: 14}
					dprng.ResetStream(seed)
					ref := dprng.New(seed)
					weight, height := ref.Number(120, 180), ref.Number(160, 200)
					if err := s.persistAcceptedCharacter(); err != nil {
						t.Fatal(err)
					}
					if s.player.Weight != weight || s.player.Height != height || s.player.Stats != s.charStats {
						t.Fatal("init_char values changed")
					}
					if err := s.completeCharCreation(); err != nil {
						t.Fatal(err)
					}
					p := s.player
					if god {
						if p.Level != game.LVL_IMPL || p.Exp != 7000000 || p.MaxHealth != 500 || p.Health != 500 || p.MaxMana != 100 || p.Mana != 100 || p.MaxMove != 82 || p.Move != 82 || p.Practices != 0 || p.GetAutoExit() || p.WimpLevel != 0 || len(p.GetInventory()) != 0 {
							t.Fatalf("God inherited do_start: level=%d practices=%d kit=%d pools=%d/%d/%d", p.Level, p.Practices, len(p.GetInventory()), p.Health, p.Mana, p.Move)
						}
						for _, skill := range []string{"sneak", "hide", "steal", "backstab", "pick_lock"} {
							if p.GetSkill(skill) != 100 {
								t.Fatalf("God skill %s=%d", skill, p.GetSkill(skill))
							}
						}
						for _, cond := range []int{game.CondFull, game.CondThirst, game.CondDrunk} {
							if p.GetCondition(cond) != -1 {
								t.Fatal("God conditions")
							}
						}
					} else {
						hpMin, hpMax, moveMax := 11, 14, 4
						manaDraw := false
						practice := 4
						switch class {
						case game.ClassMageUser:
							hpMin, hpMax, moveMax, manaDraw, practice = 4, 8, 3, true, 5
						case game.ClassCleric:
							hpMin, hpMax, manaDraw, practice = 5, 9, true, 5
							moveMax = 3
						case game.ClassThief:
							hpMin, hpMax = 7, 13
						case game.ClassNinja:
							hpMin, hpMax, manaDraw = 8, 13, true
						case game.ClassPsionic:
							hpMin, hpMax, manaDraw, practice = 4, 8, true, 5
						}
						hp := 10 + ref.Number(hpMin, hpMax)
						if manaDraw {
							upper := 2
							if class == game.ClassMageUser || class == game.ClassCleric {
								upper = 3
							}
							ref.Number(1, upper)
						}
						move := 82 + ref.Number(1, moveMax)
						if p.Level != 1 || p.Exp != 1 || p.MaxHealth != hp || p.Health != hp || p.Mana != 100 || p.MaxMana != 100 || p.MaxMove != move || p.Move != move || p.Practices != practice || !p.GetAutoExit() || p.WimpLevel != 5 || p.OrigCon != 14 {
							t.Fatalf("mortal bootstrap: class=%d hp=%d/%d want=%d move=%d want=%d practices=%d want=%d", class, p.Health, p.MaxHealth, hp, p.Move, move, p.Practices, practice)
						}
						if p.GetCondition(game.CondFull) != 36 || p.GetCondition(game.CondThirst) != 36 || p.GetCondition(game.CondDrunk) != 0 {
							t.Fatal("mortal conditions")
						}
						want := []int{8038, 8019, 8037}
						switch class {
						case game.ClassMageUser:
							want = []int{8038, 8019, 1239, 1239, 8036}
						case game.ClassCleric:
							want = []int{8038, 8019, 8023}
						case game.ClassThief, game.ClassNinja:
							want = []int{8038, 8019, 8036}
						}
						var got []int
						for _, obj := range p.GetInventory() {
							got = append(got, obj.Prototype.VNum)
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("kit order %v want %v", got, want)
						}
						pack := p.GetInventory()[0]
						for _, obj := range p.GetInventory()[1:] {
							if obj.ID < pack.ID {
								t.Fatal("pack was allocated after class objects")
							}
						}
						var contents []int
						for _, obj := range pack.Contains {
							contents = append(contents, obj.Prototype.VNum)
						}
						wantContents := []int{8063, 8010}
						if class == game.ClassThief {
							wantContents = append(wantContents, 8027)
						}
						if !reflect.DeepEqual(contents, wantContents) {
							t.Fatalf("pack order %v want %v", contents, wantContents)
						}
						if class == game.ClassThief && p.GetSkill("backstab") != 10 {
							t.Fatal("thief skills")
						}
					}
					if got := dprng.Next(); got != ref.Next() {
						t.Fatalf("bootstrap draw stream changed: %d", got)
					}
				})
			}
		}
	}
}
