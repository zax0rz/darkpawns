package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestOrgasmRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["orgasm"]
	if !ok {
		t.Fatal("orgasm command has no C gate")
	}
	if gate.MinLevel != game.LVL_IMMORT || gate.MinPosition != combat.PosResting {
		t.Fatalf("orgasm gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, game.LVL_IMMORT, combat.PosResting)
	}

	entry, ok := cmdRegistry.Lookup("orgasm")
	if !ok {
		t.Fatal("orgasm command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("orgasm registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestOrgasmHandlerKeepsCImmortalGuard(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Otouchactor", 1001, true)
	registerInWorld(t, actor)
	actor.player.SetLevel(1)

	if err := cmdOrgasm(actor, nil); err != nil {
		t.Fatalf("cmdOrgasm: %v", err)
	}
	if got := readSessionText(t, actor); got != "Come again?\n\r" {
		t.Fatalf("low-level handler output = %q, want C guard", got)
	}
}

func TestOrgasmMaleVictimUpdatesCState(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Otouchactor", 1001, true)
	victim := makeTestSession(t, m, "Otouchvictim", 1001, true)
	observer := makeTestSession(t, m, "Otouchobserver", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, victim)
	registerInWorld(t, observer)
	actor.player.SetLevel(game.LVL_IMMORT)
	victim.player.SetSex(game.SexMale)
	victim.player.SetHealth(10)
	victim.player.SetCondition(game.CondFull, 20)

	if err := cmdOrgasm(actor, []string{"Otouchvictim"}); err != nil {
		t.Fatalf("cmdOrgasm: %v", err)
	}
	if got := victim.player.GetHealth(); got != 12 {
		t.Fatalf("male victim health = %d, want C GET_HIT += 2", got)
	}
	if got := victim.player.GetCondition(game.CondFull); got != 0 {
		t.Fatalf("male victim full condition = %d, want C reset to zero", got)
	}
	for _, session := range []*Session{actor, victim, observer} {
		_ = drainSendChannel(t, session)
	}
}

func TestOrgasmFemaleActorDoesNotTriggerMaleFollowup(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Otouchactor", 1001, true)
	victim := makeTestSession(t, m, "Otouchvictim", 1001, true)
	observer := makeTestSession(t, m, "Otouchobserver", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, victim)
	registerInWorld(t, observer)
	actor.player.SetLevel(game.LVL_IMMORT)
	actor.player.SetSex(game.SexFemale)
	victim.player.SetSex(game.SexFemale)
	victim.player.SetHealth(10)
	victim.player.SetCondition(game.CondFull, 20)

	if err := cmdOrgasm(actor, []string{"the", "Otouchvictim", "ignored"}); err != nil {
		t.Fatalf("cmdOrgasm: %v", err)
	}
	if got := victim.player.GetHealth(); got != 12 {
		t.Fatalf("female victim health = %d, want C GET_HIT += 2", got)
	}
	if got := victim.player.GetCondition(game.CondFull); got != 20 {
		t.Fatalf("female victim full condition = %d, want unchanged", got)
	}
	victimText := readSessionText(t, victim) + readSessionText(t, victim)
	if strings.Contains(victimText, "Once is never enough") {
		t.Fatalf("female actor triggered male-only follow-up: %q", victimText)
	}
	for _, session := range []*Session{actor, observer} {
		_ = drainSendChannel(t, session)
	}
}
