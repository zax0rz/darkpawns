package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type carrionTestCombatEngine struct {
	starts         [][2]string
	initialAttacks [][2]string
}

func (e *carrionTestCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *carrionTestCombatEngine) StartCombatFromMob(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *carrionTestCombatEngine) PerformInitialAttack(attacker, defender combat.Combatant) error {
	e.initialAttacks = append(e.initialAttacks, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *carrionTestCombatEngine) IsFighting(string) bool { return false }

func (e *carrionTestCombatEngine) GetCombatTarget(string) (combat.Combatant, bool) {
	return nil, false
}

func newCarrionTestWorld(t *testing.T) (*World, *Player, *Player, *carrionTestCombatEngine, map[string]string) {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 14305, Name: "Food Storage"}},
		Mobs: []parser.Mob{{
			VNum:      14308,
			Keywords:  "carrion stalker",
			ShortDesc: "a carrion stalker",
			LongDesc:  "A carrion stalker jumps towards you, tentacles outstretched!",
			Level:     12,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 20},
			Damage:    parser.DiceRoll{Num: 8, Sides: 4, Plus: 8},
			Position:  8,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	actor := NewPlayer(1, "Carrionactor", 14305)
	actor.SetLevel(17)
	observer := NewPlayer(2, "Carrionwitness", 14305)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }

	engine := &carrionTestCombatEngine{}
	w.SetCombatEngine(engine)
	return w, actor, observer, engine, transcript
}

func TestSpecCarrion_EntryGates(t *testing.T) {
	w, actor, _, _, _ := newCarrionTestWorld(t)

	if specCarrion(w, actor, nil, "", "corpse") {
		t.Fatal("commandless room dispatch should fall through")
	}
	if specCarrion(w, actor, nil, "say", "") {
		t.Fatal("blank argument should fall through")
	}
	for _, arg := range []string{"stone", "corps"} {
		if specCarrion(w, actor, nil, "say", arg) {
			t.Fatalf("argument %q should fall through", arg)
		}
	}
	if specCarrion(w, actor, newSpecProcTestMob(t, w, 14305, 10), "say", "corpse") {
		t.Fatal("room special should not accept a mobile actor")
	}
	if specCarrion(w, nil, nil, "say", "corpse") {
		t.Fatal("missing player should fall through")
	}

	actor.SetPosition(combat.PosSleeping)
	if specCarrion(w, actor, nil, "say", "corpse") {
		t.Fatal("sleeping player with positive HP should fall through")
	}
	actor.SetHP(0)
	if !specCarrion(w, actor, nil, "say", "corpse") {
		t.Fatal("sleeping player with non-positive HP should reach C's gate")
	}
}

func TestSpecCarrion_SuccessfulSpawnStateAndAudience(t *testing.T) {
	w, actor, observer, engine, transcript := newCarrionTestWorld(t)

	if !specCarrion(w, actor, nil, "say", "\tcorpse fragments") {
		t.Fatal("corpse disturbance should be handled")
	}

	mobs := w.GetMobsInRoom(actor.GetRoomVNum())
	if len(mobs) != 1 {
		t.Fatalf("spawned mobs = %d, want 1", len(mobs))
	}
	stalker := mobs[0]
	if stalker.GetVNum() != 14308 {
		t.Fatalf("spawned vnum = %d, want 14308", stalker.GetVNum())
	}
	if stalker.GetLevel() != actor.GetLevel() {
		t.Fatalf("spawned level = %d, want actor level %d", stalker.GetLevel(), actor.GetLevel())
	}
	if stalker.GetDamroll() != actor.GetLevel() {
		t.Fatalf("spawned damroll = %d, want actor level %d", stalker.GetDamroll(), actor.GetLevel())
	}
	if got := stalker.GetDamageRoll().Plus; got != 0 {
		t.Fatalf("overridden damage-roll plus = %d, want 0 (no prototype double count)", got)
	}

	want := "Suddenly a carrion stalker skitters from out of a corpse!\r\n"
	if transcript[actor.Name] != want || transcript[observer.Name] != want {
		t.Fatalf("room audience = actor %q observer %q, want %q", transcript[actor.Name], transcript[observer.Name], want)
	}
	if len(engine.starts) != 1 || engine.starts[0] != [2]string{stalker.GetName(), actor.GetName()} {
		t.Fatalf("mob combat starts = %#v, want stalker-to-actor", engine.starts)
	}
	if len(engine.initialAttacks) != 1 || engine.initialAttacks[0] != [2]string{stalker.GetName(), actor.GetName()} {
		t.Fatalf("initial attacks = %#v, want stalker-to-actor", engine.initialAttacks)
	}
}

func TestSpecCarrion_KeywordAliases(t *testing.T) {
	for _, keyword := range []string{"corpse", "corpses", "pile"} {
		t.Run(keyword, func(t *testing.T) {
			w, actor, _, _, transcript := newCarrionTestWorld(t)
			if !specCarrion(w, actor, nil, "say", keyword) {
				t.Fatalf("keyword %q should handle", keyword)
			}
			if !strings.Contains(transcript[actor.Name], "skitters from out of a corpse!") {
				t.Fatalf("keyword %q output = %q", keyword, transcript[actor.Name])
			}
		})
	}
}
