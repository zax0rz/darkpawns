package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestReturnRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("return")
	if !ok {
		t.Fatal("return command is not registered")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("return gate = (level %d, position %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}

func TestCmdReturnMatchesCSwitchedBodyMessage(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Switched", 1001, true)
	original := makeTestSession(t, m, "Wizard", 1001, true)
	s.isSwitched = true
	s.switchedOriginal = original.player
	s.switchedOriginalLevel = LVL_IMPL

	if err := cmdReturn(s, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "You return to your original body.\r\n" {
		t.Fatalf("return output = %q", got)
	}
}

func TestCmdReturnMatchesCSwitchedBodyState(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Switched", 1001, true)
	original := game.NewPlayer(2, "Wizard", 1001)
	s.isSwitched = true
	s.switchedOriginal = original
	s.switchedOriginalLevel = LVL_IMPL

	if err := cmdReturn(s, nil); err != nil {
		t.Fatal(err)
	}
	if s.player != original {
		t.Fatal("return did not restore the original player")
	}
	if s.isSwitched || s.switchedOriginal != nil || s.switchedOriginalLevel != 0 {
		t.Fatalf("switched state not cleared: switched=%v original=%p level=%d", s.isSwitched, s.switchedOriginal, s.switchedOriginalLevel)
	}
}

func TestCmdReturnWithoutOriginalBodyIsSilent(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Mortal", 1001, true)

	if err := cmdReturn(s, []string{"ignored"}); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-s.send:
		t.Fatalf("return without switched body emitted %q", string(msg))
	default:
	}
}
