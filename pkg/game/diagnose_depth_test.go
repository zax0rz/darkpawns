package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDiagConditionMatchesCThresholds(t *testing.T) {
	tests := []struct {
		name  string
		hp    int
		maxHP int
		want  string
	}{
		{name: "excellent at full", hp: 100, maxHP: 100, want: "is in excellent condition."},
		{name: "scratches at 99", hp: 99, maxHP: 100, want: "has a few scratches."},
		{name: "scratches at 90", hp: 90, maxHP: 100, want: "has a few scratches."},
		{name: "small wounds at 89", hp: 89, maxHP: 100, want: "has some small wounds and bruises."},
		{name: "small wounds at 75", hp: 75, maxHP: 100, want: "has some small wounds and bruises."},
		{name: "few wounds at 74", hp: 74, maxHP: 100, want: "has quite a few wounds."},
		{name: "few wounds at 50", hp: 50, maxHP: 100, want: "has quite a few wounds."},
		{name: "big wounds at 49", hp: 49, maxHP: 100, want: "has some big nasty wounds and scratches."},
		{name: "big wounds at 30", hp: 30, maxHP: 100, want: "has some big nasty wounds and scratches."},
		{name: "hurt at 29", hp: 29, maxHP: 100, want: "looks pretty hurt."},
		{name: "hurt at 15", hp: 15, maxHP: 100, want: "looks pretty hurt."},
		{name: "awful at 14", hp: 14, maxHP: 100, want: "is in awful condition."},
		{name: "awful at zero", hp: 0, maxHP: 100, want: "is in awful condition."},
		{name: "bleeding below zero", hp: -1, maxHP: 100, want: "is bleeding awfully from big wounds."},
		{name: "nonpositive maximum", hp: 1, maxHP: 0, want: "is bleeding awfully from big wounds."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := diagCondition(test.hp, test.maxHP); got != test.want {
				t.Fatalf("diagCondition(%d, %d) = %q, want %q", test.hp, test.maxHP, got, test.want)
			}
		})
	}
}

func TestDoDiagnoseUsesFightingFallbackAndPrivateOutput(t *testing.T) {
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Diagnose Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:       2001,
			Keywords:   "target",
			ShortDesc:  "a target",
			HP:         parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
			Damage:     parser.DiceRoll{Num: 1, Sides: 1},
			Position:   8,
			DefaultPos: 8,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	actor := NewPlayer(1, "Actor", 1001)
	observer := NewPlayer(2, "Observer", 1001)
	if err := world.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer(actor): %v", err)
	}
	if err := world.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer(observer): %v", err)
	}
	mob, err := world.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	mob.SetHealth(mob.GetMaxHP())
	actor.SetFighting(mob.GetName())

	result := world.DoDiagnose(actor, "")
	if len(result.Messages) != 1 {
		t.Fatalf("fighting fallback messages = %+v, want one private message", result.Messages)
	}
	if got := result.Messages[0].Format; got != "$N is in excellent condition." {
		t.Fatalf("fighting fallback format = %q, want C format", got)
	}

	messages := make(map[string]string)
	world.MessageSink = func(playerName string, message []byte) {
		messages[playerName] += string(message)
	}
	world.RenderObservationMessages(result)
	if got := messages[actor.Name]; got != "A target is in excellent condition.\r\n" {
		t.Fatalf("actor diagnosis = %q, want exact C bytes", got)
	}
	if got := messages[observer.Name]; got != "" {
		t.Fatalf("observer received private diagnosis = %q", got)
	}
}
