package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func seedForCastRoll(t *testing.T, accept func(int) bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if accept(dprng.New(seed).Number(0, 101)) {
			return seed
		}
	}
	t.Fatal("could not find cast-roll seed")
	return 0
}

func prepareCaster(s *Session, class int, spellName string, proficiency int) {
	s.player.Class = class
	s.player.SetLevel(1)
	s.player.SetMana(100)
	s.player.SpellMap[spellName] = 1
	s.player.SetSkill(spellName, proficiency)
}

func TestManaCostUsesClassMinimumLevel(t *testing.T) {
	if got := manaCost(spellDB[spells.SpellMagicMissile], 1, game.ClassMageUser); got != 30 {
		t.Errorf("L1 flame arrow mana = %d, want 30", got)
	}
	if got := manaCost(spellDB[spells.SpellCureLight], 1, game.ClassCleric); got != 30 {
		t.Errorf("L1 cure light mana = %d, want 30", got)
	}
	if got := manaCost(spellDB[spells.SpellInfravision], 1, game.ClassMageUser); got != 25 {
		t.Errorf("L1 infravision mana = %d, want 25", got)
	}
}

func TestCastCureLightDrawOrderManaWaitAndFullHPCap(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Cleric", 1001, true)
	registerInWorld(t, s)
	prepareCaster(s, game.ClassCleric, "cure light", 95)
	s.player.SetHP(s.player.GetMaxHP())

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101) // concentration
	wantStream.Number(1, 8)   // cure-light die 1
	wantStream.Number(1, 8)   // cure-light die 2
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast cure light: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("first output = %q, want Okay", got)
	}
	if got := readSessionText(t, s); got != "You feel better.\r\n" {
		t.Errorf("effect output = %q", got)
	}
	if got := s.player.GetHP(); got != s.player.GetMaxHP() {
		t.Errorf("full-HP cure changed HP to %d/%d", got, s.player.GetMaxHP())
	}
	if got := s.player.GetMana(); got != 70 {
		t.Errorf("mana = %d, want 70", got)
	}
	if got := s.player.GetWaitState(); got != 1 {
		t.Errorf("wait = %d, want 1", got)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration + 2d8", got, wantNext)
	}
}

func TestCastInfravisionConsumesOnlyConcentrationAndAccumulates(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Mage", 1001, true)
	registerInWorld(t, s)
	prepareCaster(s, game.ClassMageUser, "infravision", 95)

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101)
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"infravision"}); err != nil {
		t.Fatalf("cmdCast infravision: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("first output = %q", got)
	}
	if got := readSessionText(t, s); got != "Your eyes glow red.\r\n" {
		t.Errorf("effect output = %q", got)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration only", got, wantNext)
	}
	if len(s.player.ActiveAffects) != 1 || s.player.ActiveAffects[0].Duration != 13 {
		t.Fatalf("first infravision affect = %+v, want one duration-13 affect", s.player.ActiveAffects)
	}

	// Direct command invocation bypasses the dispatcher wait gate so the recast
	// can exercise C affect_join(accum_duration=true).
	dprng.ResetStream(seed)
	if err := cmdCast(s, []string{"infravision"}); err != nil {
		t.Fatalf("recast infravision: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("recast first output = %q", got)
	}
	if got := readSessionText(t, s); got != "Your eyes glow red.\r\n" {
		t.Errorf("recast effect output = %q", got)
	}
	if len(s.player.ActiveAffects) != 1 || s.player.ActiveAffects[0].Duration != 26 {
		t.Fatalf("recast infravision affect = %+v, want one duration-26 affect", s.player.ActiveAffects)
	}
	if got := s.player.GetMana(); got != 50 {
		t.Errorf("mana after two casts = %d, want 50", got)
	}
}

func TestCastFlameArrowDrawOrderIncantationAndComponentMessage(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	caster := makeGateSession(t, m, 1, "Caster", 1)
	observer := makeGateSession(t, m, 2, "Observer", 1)
	prepareCaster(caster, game.ClassMageUser, "flame arrow", 95)

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101) // concentration
	for range 4 {
		wantStream.Number(1, 3)
	}
	wantStream.Number(0, 99) // target saving throw
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	beforeHP := mob.GetHP()
	if err := cmdCast(caster, []string{"'flame", "arrow'", "target"}); err != nil {
		t.Fatalf("cmdCast flame arrow: %v", err)
	}
	if got := readSessionText(t, caster); got != "Okay.\r\n" {
		t.Errorf("first output = %q", got)
	}
	if got := readSessionText(t, caster); got != "You attempt the spell without the components..\r\n" {
		t.Errorf("component output = %q", got)
	}
	incantation := readSessionText(t, observer)
	if !strings.Contains(incantation, "Caster stares at a test target") || !strings.Contains(incantation, "'flame arrow'") {
		t.Errorf("observer incantation = %q", incantation)
	}
	if mob.GetHP() >= beforeHP {
		t.Errorf("flame arrow did not damage mob: %d -> %d", beforeHP, mob.GetHP())
	}
	if got := caster.player.GetMana(); got != 70 {
		t.Errorf("mana = %d, want 70", got)
	}
	if got := caster.player.GetWaitState(); got != 1 {
		t.Errorf("wait = %d, want 1", got)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration + 4d3 + save", got, wantNext)
	}
}

func TestCastFailureConsumesOneDrawHalfManaAndNoIncantation(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Cleric", 1001, true)
	observer := makeTestSession(t, m, "Observer", 1001, true)
	registerInWorld(t, s)
	registerInWorld(t, observer)
	prepareCaster(s, game.ClassCleric, "cure light", 1)

	seed := seedForCastRoll(t, func(roll int) bool { return roll > 1 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101)
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast failure: %v", err)
	}
	if got := readSessionText(t, s); got != "You lost your concentration!\r\n" {
		t.Errorf("failure output = %q", got)
	}
	if got := s.player.GetMana(); got != 85 {
		t.Errorf("failure mana = %d, want 85", got)
	}
	if got := s.player.GetWaitState(); got != 1 {
		t.Errorf("failure wait = %d, want 1", got)
	}
	select {
	case msg := <-observer.send:
		t.Fatalf("observer received incantation on failed cast: %s", msg)
	default:
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after failure draw", got, wantNext)
	}
}
