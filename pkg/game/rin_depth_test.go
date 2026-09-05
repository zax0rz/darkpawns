package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestDoKujiKiri_RinSuccessMessages(t *testing.T) {
	p := NewPlayer(1, "Rinactor", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkRin, 100)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkRin, nil)
	if !result.Success {
		t.Fatalf("Rin should succeed at skill 100 with seed 1: %q", result.MessageToCh)
	}
	if result.MessageToCh != "Interlacing your fingers, you harden your mind and body." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Rinactor interlaces his fingers, and his skin becomes metal!" {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
}

func TestDoKujiKiri_RinFailureMessages(t *testing.T) {
	p := NewPlayer(1, "Rinactor", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkRin, 1)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkRin, nil)
	if result.Success {
		t.Fatal("Rin should fail at skill 1 with seed 1")
	}
	if result.MessageToCh != "You try the art of kuji-kiri, but can't concentrate!" {
		t.Errorf("failure actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Rinactor interlaces his fingers, and his skin becomes metal!" {
		t.Errorf("failure room message = %q", result.MessageToRoom)
	}
}
