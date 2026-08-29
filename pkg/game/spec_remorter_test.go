package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRemorterClassMatchesCMapping(t *testing.T) {
	tests := []struct {
		class int
		race  int
		want  int
	}{
		{ClassWarrior, RaceHuman, ClassPaladin},
		{ClassPaladin, RaceElf, ClassPaladin},
		{ClassRanger, RaceDwarf, ClassPaladin},
		{ClassWarrior, RaceKender, ClassRanger},
		{ClassCleric, RaceHuman, ClassAvatar},
		{ClassThief, RaceHuman, ClassAssassin},
		{ClassMageUser, RaceHuman, ClassMagus},
		{ClassPsionic, RaceHuman, ClassMystic},
		{ClassNinja, RaceHuman, ClassNinja},
	}

	for _, test := range tests {
		if got := remorterClass(test.class, test.race); got != test.want {
			t.Errorf("remorterClass(%d, %d) = %d, want %d", test.class, test.race, got, test.want)
		}
	}
}

func TestSpecRemorterSuccessResetsStateAndSeedsClassSkills(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Remort Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 2001, Keywords: "remorter", ShortDesc: "the remorter"}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var output strings.Builder
	w.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
	ch := NewPlayer(1, "RemortTest", 1001)
	ch.Class = ClassMageUser
	ch.Race = RaceHuman
	ch.Level = LVL_IMMORT - 1
	ch.Gold = 60000
	ch.SetPlrFlag(PlrIt, true)
	ch.SetPlrFlag(PlrVampire, true)
	ch.SetAffect(affVampire, true)
	ch.AddAffect(engine.NewAffectDirect(999, engine.ApplyHitroll, 4, 3, engine.AFFNone, "test"))
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	me := NewMobInstance(&parsed.Mobs[0], 1001)
	previousLevelNumber := levelNumber
	levelNumber = func(_, _ int) int { return 1 }
	t.Cleanup(func() { levelNumber = previousLevelNumber })

	if !specRemorter(w, ch, me, "remort", "") {
		t.Fatal("successful remort should consume the command")
	}

	if got := ch.GetClass(); got != ClassMagus {
		t.Errorf("class after remort = %d, want %d", got, ClassMagus)
	}
	if got := ch.GetLevel(); got != 1 {
		t.Errorf("level after remort = %d, want 1", got)
	}
	if got := ch.GetExp(); got != 1 {
		t.Errorf("experience after remort = %d, want 1", got)
	}
	if got := ch.GetGold(); got != 0 {
		t.Errorf("gold after remort = %d, want 0", got)
	}
	if ch.GetFlags()&(1<<PlrRemort) == 0 {
		t.Error("remort flag was not set")
	}
	if ch.GetFlags()&(1<<PlrIt|1<<PlrVampire|1<<PlrWerewolf) != 0 {
		t.Errorf("old remort flags remain: %#x", ch.GetFlags())
	}
	if ch.IsAffected(affVampire) || len(ch.ActiveAffects) != 0 {
		t.Error("old vampire/active affects remain after remort")
	}
	if got := ch.GetSkill("flame arrow"); got != 40 {
		t.Errorf("magus magic-missile slot = %d, want 40", got)
	}
	if got := ch.GetSkill("acid blast"); got != 40 {
		t.Errorf("magus acid-blast slot = %d, want 40", got)
	}
	if got := ch.GetPractices(); got != 12 {
		t.Errorf("practices after level-one advance = %d, want 12", got)
	}
	if !strings.Contains(output.String(), "Enjoy your new life") || !strings.Contains(output.String(), "Colors spiral") {
		t.Errorf("success output = %q", output.String())
	}
}

func TestSpecRemorterRejectsPlayerCommandsAtTheCorrectGates(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.Prototype.ShortDesc = "the remorter"

	cases := []struct {
		name  string
		level int
		gold  int
		want  string
	}{
		{"low level", LVL_IMMORT - 2, 60000, "You can't remort until level 30"},
		{"immortal", LVL_IMMORT, 60000, "Immortals cannot remort"},
		{"poor", LVL_IMMORT - 1, 0, "It costs 60000 gold"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var output strings.Builder
			w.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
			player.SetLevel(test.level)
			player.SetGold(test.gold)
			if !specRemorter(w, player, mob, "remort", "") {
				t.Fatal("gate should consume remort command")
			}
			if !strings.Contains(output.String(), test.want) {
				t.Errorf("gate output = %q, want substring %q", output.String(), test.want)
			}
		})
	}

	if specRemorter(w, player, mob, "look", "") {
		t.Error("unrelated command should fall through")
	}
	if specRemorter(w, nil, mob, "remort", "") {
		t.Error("nil autonomous actor should fall through")
	}
}
