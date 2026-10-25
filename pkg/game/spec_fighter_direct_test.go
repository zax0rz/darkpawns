package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestSpecFighterHeadbuttRefusesImmortalDirectTarget(t *testing.T) {
	w, player, transcript := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	player.SetLevel(LVL_IMMORT)
	player.SetPosition(combat.PosFighting)
	mob.SetPosition(combat.PosFighting)
	mob.SetFightingBody(player)

	var commandName, commandArgument string
	w.CharacterCommandDirectExecFunc = func(_ combat.Combatant, name, argument string) CharacterCommandResult {
		commandName, commandArgument = name, argument
		return CharacterCommandUnsupported
	}

	var seed uint32
	for candidate := uint32(1); candidate < 10000; candidate++ {
		dprng.ResetStream(candidate)
		if dprng.Number(0, 10) == 1 {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("could not find a seed selecting fighter's headbutt case")
	}
	dprng.ResetStream(seed)
	dprng.Number(0, 10) // specFighter's sole draw; C refuses before headbutt draws.
	wantNext := dprng.Number(0, 100)
	dprng.ResetStream(seed)
	transcript() // discard setup messages

	if !specFighter(w, nil, mob, "", "") {
		t.Fatal("fighter did not handle its headbutt case")
	}
	if commandName != "headbutt" || commandArgument != player.GetName() {
		t.Fatalf("direct command = %q %q, want headbutt %q", commandName, commandArgument, player.GetName())
	}
	if got := transcript(); got != "" {
		// Revert control: restoring the Go victim line must fail this assertion.
		t.Fatalf("immortal headbutt bytes = %q, want no victim output", got)
	}
	if got := mob.GetPosition(); got != combat.PosSitting {
		t.Fatalf("mobile position after refusal = %d, want sitting (%d)", got, combat.PosSitting)
	}
	if got := dprng.Number(0, 100); got != wantNext {
		t.Fatalf("immortal refusal consumed a headbutt draw: next=%d want=%d", got, wantNext)
	}
}
