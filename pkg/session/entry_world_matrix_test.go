package session

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func entryWorldManager(t *testing.T, database db.GameStore) *Manager {
	t.Helper()
	t.Setenv("JWT_SECRET", "entry-world-test-jwt-secret-at-least-32")
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Saved"}, {VNum: game.MortalStartRoom, Name: "Mortal"}, {VNum: game.ImmortStartRoom, Name: "Immortal"}, {VNum: game.FrozenStartRoom, Name: "Frozen"}, {VNum: game.NewbieStartRoom, Name: "Newbie"}}}
	parsed.Objs = []parser.Obj{{VNum: 8023, Keywords: "club", WearFlags: [4]int{1}}, {VNum: 8019, Keywords: "tunic", WearFlags: [4]int{9}, Affects: []parser.ObjAffect{{Location: game.ApplyCon, Modifier: 2}}}}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	return newTestManager(t, w, database)
}

// src/interpreter.c:2174-2243: selected room, unhealthy guard, reset_char,
// INVSTART, one registration, and act's visible room audience.
func TestEntryWorldMatrix(t *testing.T) {
	cases := []struct {
		name              string
		level, load, want int
		frozen, inv       bool
	}{
		{"saved mortal", 1, 1001, 1001, false, false},
		{"saved immortal", 40, 1001, 1001, false, false},
		{"mortal fallback", 1, -1, game.MortalStartRoom, false, false},
		{"invalid fallback", 1, 999999, game.MortalStartRoom, false, false},
		{"immortal fallback", game.LVL_IMMORT, -1, game.ImmortStartRoom, false, false},
		{"frozen", 1, 1001, game.FrozenStartRoom, true, false},
		{"invstart", 40, 1001, 1001, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := entryWorldManager(t, nil)
			observer := makeTestSession(t, m, "Watcher", tc.want, true)
			observer.player.SetLevel(1)
			registerTestSession(t, m, observer, "Watcher")
			renderedOutput(observer)
			remote := makeTestSession(t, m, "Remote", game.NewbieStartRoom, true)
			registerTestSession(t, m, remote, "Remote")
			renderedOutput(remote)
			s := makeCharSession(t, m)
			s.player = game.NewPlayer(7, "Entrant", 1001)
			s.player.Stats.Con = 10
			s.player.CopyBaseAttributes()
			s.player.SetLevel(tc.level)
			s.player.SetLoadRoom(tc.load)
			s.player.SetPlrFlag(game.PlrFrozen, tc.frozen)
			s.player.SetPlrFlag(game.PlrInvstart, tc.inv)
			s.player.SetHP(0)
			s.player.SetMana(0)
			s.player.SetMove(0)
			s.player.SetPosition(game.PosSleeping)
			s.authenticated = true
			s.menuActive = true
			s.menuStage = "menu"
			s.playerName = "Entrant"
			if err := s.enterReturningPlayer(); err != nil {
				t.Fatal(err)
			}
			if s.player.GetRoom() != tc.want || s.menuActive || s.player.GetPosition() != game.PosStanding || s.player.GetHP() != 1 || s.player.GetMana() != 1 || s.player.GetMove() != 1 {
				t.Fatalf("entry state: room=%d menu=%v pos=%d pools=%d/%d/%d", s.player.GetRoom(), s.menuActive, s.player.GetPosition(), s.player.GetHP(), s.player.GetMana(), s.player.GetMove())
			}
			if got, _ := m.GetSession("Entrant"); got != s {
				t.Fatal("registration")
			}
			if m.world.GetPlayerCount() != 3 {
				t.Fatal("world registration count")
			}
			room := renderedOutput(observer)
			if tc.inv {
				if room != "" {
					t.Fatalf("invisible audience: %q", room)
				}
			} else if room != "Entrant has entered the game.\r\n" {
				t.Fatalf("room: %q", room)
			}
			if got := renderedOutput(remote); got != "" {
				t.Fatalf("other room: %q", got)
			}
			own := renderedOutput(s)
			if !strings.Contains(own, "Welcome to Dark Pawns! May your visit here be... Interesting.") || strings.Contains(own, "Entrant has entered the game.") {
				t.Fatalf("own output: %q", own)
			}
		})
	}
}

func TestEntryWorldUnhealthy(t *testing.T) {
	for _, level := range []int{0, 1, 40} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			database := entryDatabase(t)
			rec := entrySeed(t, database, "Unhealthy")
			rec.Level = level
			rec.StatCon = 0
			rec.Inventory = []byte(`[{"vnum":8023}]`)
			rec.Equipment = []byte(`[{"vnum":8019,"locate":6}]`)
			if err := database.SavePlayer(rec); err != nil {
				t.Fatal(err)
			}
			before, err := database.GetPlayer("Unhealthy")
			if err != nil {
				t.Fatal(err)
			}
			m := entryWorldManager(t, database)
			s := makeCharSession(t, m)
			s.player, err = db.RecordToPlayer(before, m.world)
			if err != nil {
				t.Fatal(err)
			}
			if len(m.world.GetAllObjects()) != 2 || s.player.GetCon() != 2 || s.player.LoginConstitution() != 0 {
				t.Fatal("restored equipment must be a real positive-CON control")
			}
			s.authenticated = true
			s.menuActive = true
			s.menuStage = "menu"
			s.playerName = "Unhealthy"
			if err := m.Register("Unhealthy", s); err != nil {
				t.Fatal(err)
			}
			sendMenuInput(t, s, "1")
			if got := renderedOutput(s); got != "You are too unhealthy to play!\r\n" {
				t.Fatalf("unhealthy output: %q", got)
			}
			if !s.SendClosed() || s.authenticated || s.player != nil || m.world.GetPlayerCount() != 0 || len(m.world.GetAllObjects()) != 0 {
				t.Fatal("unhealthy retained candidate/world")
			}
			if _, ok := m.GetSession("Unhealthy"); ok {
				t.Fatal("unhealthy retained session")
			}
			after, err := database.GetPlayer("Unhealthy")
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("unhealthy rewrote durable record: %v", err)
			}
		})
	}
}

func TestEntryWorldNewMortalAudience(t *testing.T) {
	t.Setenv("DP_FRESH_MUD", "")
	m := newTestManager(t, bootstrapWorld(t), nil)
	oldRoom := makeTestSession(t, m, "Watcher", game.MortalStartRoom, true)
	registerTestSession(t, m, oldRoom, "Watcher")
	newbie := makeTestSession(t, m, "Newbie", game.NewbieStartRoom, true)
	registerTestSession(t, m, newbie, "Newbie")
	s := makeCharSession(t, m)
	s.charName = "Entrant"
	s.charStats = game.CharStats{Str: 14, Int: 14, Wis: 14, Dex: 14, Con: 14, Cha: 14}
	s.charClass = game.ClassWarrior
	s.charHometown = 1
	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}
	if s.player.GetRoom() != game.NewbieStartRoom {
		t.Fatalf("new room=%d", s.player.GetRoom())
	}
	if got := renderedOutput(oldRoom); got != "Entrant has entered the game.\r\n" {
		t.Fatalf("initial load room audience: %q", got)
	}
	if got := renderedOutput(newbie); got != "" {
		t.Fatalf("newbie room audience: %q", got)
	}
}

func TestEntryWorldAudience(t *testing.T) {
	for _, tc := range []struct {
		name         string
		position     int
		writing      bool
		level, invis int
		want         bool
	}{
		{"awake", game.PosStanding, false, 1, 0, true}, {"sleeping", game.PosSleeping, false, 1, 0, false}, {"writing", game.PosStanding, true, 1, 0, false}, {"invisible hidden", game.PosStanding, false, 1, 40, false}, {"invisible visible", game.PosStanding, false, 40, 40, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := entryWorldManager(t, nil)
			observer := makeTestSession(t, m, "Watcher", 1001, true)
			observer.player.SetLevel(tc.level)
			observer.player.SetPosition(tc.position)
			observer.player.SetPlrFlag(game.PlrWriting, tc.writing)
			registerTestSession(t, m, observer, "Watcher")
			renderedOutput(observer)
			s := makeCharSession(t, m)
			s.player = game.NewPlayer(7, "Entrant", 1001)
			s.player.Stats.Con = 10
			s.player.CopyBaseAttributes()
			s.player.SetLoadRoom(1001)
			s.player.SetInvisLevel(tc.invis)
			s.authenticated = true
			s.menuActive = true
			if err := s.enterReturningPlayer(); err != nil {
				t.Fatal(err)
			}
			want := ""
			if tc.want {
				want = "Entrant has entered the game.\r\n"
			}
			if got := renderedOutput(observer); got != want {
				t.Fatalf("audience=%q, want %q", got, want)
			}
		})
	}
}
