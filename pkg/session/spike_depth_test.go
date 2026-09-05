package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSpikeRegistrationUsesCEntryGate(t *testing.T) {
	for _, name := range []string{"spike", "stake"} {
		entry, ok := commandGates[name]
		if !ok {
			t.Fatalf("%s command has no C gate", name)
		}
		if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
			t.Fatalf("%s gate = level %d position %d, want level 0 position %d", name, entry.MinLevel, entry.MinPosition, combat.PosStanding)
		}
	}
}

func TestSpikeRegistrationUsesNativeHandler(t *testing.T) {
	for _, name := range []string{"spike", "stake"} {
		entry, ok := cmdRegistry.Lookup(name)
		if !ok {
			t.Fatalf("%s command is not registered", name)
		}
		if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
			t.Fatalf("%s registry gate = level %d position %d, want level 0 position %d", name, entry.MinLevel, entry.MinPosition, combat.PosStanding)
		}
	}
}

func TestSpikeEntryGateIsNotSkillKnowledgeGated(t *testing.T) {
	if _, ok := game.SkillUnknownMsg[game.SkillSpike]; ok {
		t.Fatal("spike must not use the generic skill-knowledge gate: do_spike has no GET_SKILL check")
	}
	if _, ok := game.SkillClassReq[game.SkillSpike]; ok {
		t.Fatal("spike must not have an invented class/level skill requirement")
	}
}
