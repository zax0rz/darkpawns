package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestAccuseRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["accuse"]
	if !ok {
		t.Fatal("accuse command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosSitting {
		t.Fatalf("accuse gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosSitting)
	}

	social, ok := game.Socials["accuse"]
	if !ok {
		t.Fatal("accuse social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("accuse social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position %d, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("accuse social has %d messages, want 8", len(social.Messages))
	}
}

// TestAccuseNoArgBytesPinCSource enforces the bytes the oracle census cannot
// verify: accuse's no-argument block is classified EXPECTED_UNSTABLE because
// the C oracle emits run-varying garbage where its own socials file specifies
// these bytes (accuse.tsv stability=run-varying; the # line is fread_action's
// NULL-field convention, so C's no-arg path sends only char_no_arg and act()
// no-ops the NULL others_no_arg). A Go-side regression here would stay
// census-green, so the source-literal expectation is enforced by unit test
// instead (R1/R4/R5e; see PR #1501's masking-tradeoff note).
func TestAccuseNoArgBytesPinCSource(t *testing.T) {
	social, ok := game.Socials["accuse"]
	if !ok {
		t.Fatal("accuse social is not registered")
	}
	// C social_messg field order: char_no_arg, others_no_arg, char_found, ...
	if social.Messages[0] != "Accuse who??" {
		t.Fatalf("char_no_arg = %q, want the socials-file literal", social.Messages[0])
	}
	if social.Messages[1] != "#" {
		t.Fatalf("others_no_arg = %q, want # (fread_action's NULL field)", social.Messages[1])
	}

	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Accuser", 1, 1001)
	// The no-arg path delivers through Act's world sink, which routes by
	// registered player name; an unregistered fixture player would swallow it.
	if err := m.Register("Accuser", s); err != nil {
		t.Fatalf("register Accuser: %v", err)
	}
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if err := ExecuteCommand(s, "accuse", nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Accuse who??\r\n" {
		t.Fatalf("no-arg actor bytes = %q, want %q", got, "Accuse who??\r\n")
	}
	// The room line is absent by source (others_no_arg is NULL): nothing may
	// follow the actor line.
	if extra := readMsgTextOrEmpty(t, s); extra != "" {
		t.Fatalf("unexpected output after the actor line: %q", extra)
	}
}
