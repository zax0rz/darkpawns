package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdUnaffectMatchesCFlagsAffectsAndBytes(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Unaffectgod", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Unaffectvictim", 1, 1001)
	observer := makeCommandTestSession(t, m, "Unaffectobserver", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)
	registerInWorld(t, observer)

	actor.player.SetMana(actor.player.GetMaxMana() + 25)
	affPoison := game.EngineFlagToAffBit[engine.AFFPoison]
	target.player.SetPlrFlag(game.PlrIt, true)
	target.player.SetPlrFlag(game.PlrVampire, true)
	target.player.SetPlrFlag(game.PlrWerewolf, true)
	target.player.SetAffect(game.AffVampire, true)
	target.player.SetAffect(game.AffWerewolf, true)
	target.player.SetAffect(affPoison, true)
	target.player.Stats.Dex = 18
	target.player.CopyBaseAttributes()
	target.player.AddAffect(engine.NewAffectDirect(1, game.ApplyDex, 2, -30, engine.AFFPoison, "test"))
	if target.player.GetDex() != 0 {
		t.Fatal("fixture did not apply the active dexterity affect")
	}

	if err := ExecuteCommand(actor, "unaffect", []string{"Unaffectvictim"}); err != nil {
		t.Fatalf("ExecuteCommand unaffect: %v", err)
	}

	if got, want := readWizardUnaffectMessages(t, actor, 4), "Un-tagged!\r\nUn-vamped!\r\nUn-wered!\r\nAll spells removed.\r\n"; got != want {
		t.Fatalf("actor bytes = %q, want %q", got, want)
	}
	if got, want := readWizardUnaffectMessages(t, target, 4), "You're no longer it!\r\nYou're no longer a creature of the night!\r\nYou're no longer a creature of the night!\r\nThere is a brief flash of light!\r\nYou feel slightly different.\r\n"; got != want {
		t.Fatalf("victim bytes = %q, want %q", got, want)
	}
	assertWizardUnaffectNoMessage(t, observer)

	for _, flag := range []int{game.PlrIt, game.PlrVampire, game.PlrWerewolf} {
		if target.player.HasPLRFlag(flag) {
			t.Errorf("target retained player flag %d", flag)
		}
	}
	for _, bit := range []int{game.AffVampire, game.AffWerewolf, affPoison} {
		if target.player.IsAffected(bit) {
			t.Errorf("target retained affect bit %d", bit)
		}
	}
	if len(target.player.ActiveAffects) != 0 || target.player.GetDex() != 18 {
		t.Errorf("target active affects/dexterity = %v/%d, want none/18", target.player.ActiveAffects, target.player.GetDex())
	}
	if got, want := actor.player.GetMana(), actor.player.GetMaxMana(); got != want {
		t.Errorf("actor mana = %d, want C's vampire-branch clamp to %d", got, want)
	}
}

func TestCmdUnaffectMatchesCNoAffectBranch(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Unaffectgod", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Unaffectvictim", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	// C checks the linked-list head, so an empty but non-nil Go slice is still
	// the no-affects branch.
	target.player.ActiveAffects = make([]*engine.Affect, 0)
	target.player.SetPlrFlag(game.PlrIt, true)
	target.player.SetPlrFlag(game.PlrVampire, true)
	target.player.SetPlrFlag(game.PlrWerewolf, true)
	target.player.SetAffect(game.AffVampire, true)
	target.player.SetAffect(game.AffWerewolf, true)

	if err := ExecuteCommand(actor, "unaffect", []string{"Unaffectvictim"}); err != nil {
		t.Fatalf("ExecuteCommand unaffect: %v", err)
	}
	if got, want := readWizardUnaffectMessages(t, actor, 4), "Un-tagged!\r\nUn-vamped!\r\nUn-wered!\r\nYour victim does not have any affections!\r\n"; got != want {
		t.Fatalf("actor no-affect bytes = %q, want %q", got, want)
	}
	if got, want := readWizardUnaffectMessages(t, target, 3), "You're no longer it!\r\nYou're no longer a creature of the night!\r\nYou're no longer a creature of the night!\r\n"; got != want {
		t.Fatalf("victim no-affect bytes = %q, want %q", got, want)
	}
	for _, flag := range []int{game.PlrIt, game.PlrVampire, game.PlrWerewolf} {
		if target.player.HasPLRFlag(flag) {
			t.Errorf("target retained player flag %d", flag)
		}
	}
	for _, bit := range []int{game.AffVampire, game.AffWerewolf} {
		if target.player.IsAffected(bit) {
			t.Errorf("target retained affect bit %d", bit)
		}
	}
}

func readWizardUnaffectMessages(t *testing.T, s *Session, count int) string {
	t.Helper()
	var messages string
	for range count {
		messages += readMsgText(t, s)
	}
	return messages
}

func assertWizardUnaffectNoMessage(t *testing.T, s *Session) {
	t.Helper()
	select {
	case msg := <-s.send:
		t.Fatalf("unexpected unaffect observer message: %s", msg)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCmdUnaffectDispatchRemainsRegisteredAtGod(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("unaffect")
	if !ok {
		t.Fatal("unaffect command is not registered")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("unaffect gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosDead)
	}
}
