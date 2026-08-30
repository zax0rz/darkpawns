package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type elementsGuardianCombatEngine struct {
	starts         [][2]string
	initialAttacks [][2]string
}

func (e *elementsGuardianCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *elementsGuardianCombatEngine) PerformInitialAttack(attacker, defender combat.Combatant) error {
	e.initialAttacks = append(e.initialAttacks, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *elementsGuardianCombatEngine) IsFighting(string) bool { return false }

func (e *elementsGuardianCombatEngine) GetCombatTarget(string) (combat.Combatant, bool) {
	return nil, false
}

func TestSpecElementsGuardian_CommandlessAndNilEntryGates(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 80)
	_ = lastMsg()

	if specElementsGuardian(w, nil, mob, "", "") {
		t.Fatal("commandless guardian should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("commandless guardian emitted %q", got)
	}
	if specElementsGuardian(w, nil, mob, "say", "hello") {
		t.Fatal("guardian with no command actor should fall through")
	}
}

func TestSpecElementsGuardian_PairUsesRoomOrderAudienceAndHit(t *testing.T) {
	w, actor, _ := newSpecProcTestWorld(t)
	actor.SetLevel(LVL_IMMORT + 1)
	witness := NewPlayer(2, "GuardianWitness", actor.GetRoomVNum())
	third := NewPlayer(3, "GuardianThird", actor.GetRoomVNum())
	if err := w.AddPlayer(witness); err != nil {
		t.Fatalf("AddPlayer witness: %v", err)
	}
	if err := w.AddPlayer(third); err != nil {
		t.Fatalf("AddPlayer third: %v", err)
	}
	mob := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 80)
	mob.Prototype.ShortDesc = "a guardian spirit"

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }
	engine := &elementsGuardianCombatEngine{}
	w.SetCombatEngine(engine)

	if specElementsGuardian(w, actor, mob, "say", "hello") {
		t.Fatal("guardian should leave the command available")
	}

	if got, want := len(engine.starts), 1; got != want {
		t.Fatalf("combat starts = %d, want %d", got, want)
	}
	if got, want := engine.starts[0], [2]string{"GuardianThird", "GuardianWitness"}; got != want {
		t.Fatalf("combat pair = %#v, want %#v", got, want)
	}
	if got, want := len(engine.initialAttacks), 1; got != want {
		t.Fatalf("initial attacks = %d, want %d", got, want)
	}
	if got, want := engine.initialAttacks[0], [2]string{"GuardianThird", "GuardianWitness"}; got != want {
		t.Fatalf("initial attack = %#v, want %#v", got, want)
	}
	if third.IsAffected(affCharm) || witness.IsAffected(affCharm) {
		t.Fatal("guardian should not add an affect absent from C")
	}

	if got, want := transcript[actor.Name], "A guardian spirit mumbles softly and GuardianThird screams loudly, attacking GuardianWitness!\r\n"; got != want {
		t.Fatalf("observer transcript = %q, want %q", got, want)
	}
	if got, want := transcript[witness.Name], "A guardian spirit mumbles softly and GuardianThird screams loudly, attacking you!\r\n"; got != want {
		t.Fatalf("victim transcript = %q, want %q", got, want)
	}
	if got, want := transcript[third.Name], "A guardian spirit mumbles softly and you scream loudly, attacking GuardianWitness!\r\n"; got != want {
		t.Fatalf("attacker transcript = %q, want %q", got, want)
	}
}

func TestSpecElementsGuardian_SoloUsesSelfDamageAndActPronouns(t *testing.T) {
	w, actor, _ := newSpecProcTestWorld(t)
	actor.SetLevel(LVL_IMMORT + 1)
	target := NewPlayer(2, "GuardianTarget", actor.GetRoomVNum())
	target.SetMaxHP(1000)
	target.SetHP(1000)
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}
	mob := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 80)
	mob.Prototype.ShortDesc = "a guardian spirit"

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }
	if specElementsGuardian(w, actor, mob, "say", "hello") {
		t.Fatal("guardian should leave the command available")
	}

	if got, want := transcript[actor.Name], "A guardian spirit mumbles softly and GuardianTarget begins screaming loudly, hitting himself.\r\n"; got != want {
		t.Fatalf("observer transcript = %q, want %q", got, want)
	}
	if got, want := transcript[target.Name], "A guardian spirit mumbles softly and you begin to scream, involuntarily hitting yourself.\r\n"; got != want {
		t.Fatalf("target transcript = %q, want %q", got, want)
	}
	if target.GetFighting() != "" {
		t.Fatalf("self-damage unexpectedly enrolled combat target %q", target.GetFighting())
	}
	if strings.Contains(transcript[actor.Name], "GuardianTargetself") {
		t.Fatal("solo Act output used a literal name instead of C's pronoun")
	}
}
