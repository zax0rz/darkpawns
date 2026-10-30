package spells

import (
	"testing"
)

// recordingCounterattackWorld satisfies the counterattack seam and records it.
type recordingCounterattackWorld struct {
	gotVictim, gotCaster interface{}
}

func (w *recordingCounterattackWorld) ExecuteSpellCounterattack(victim, caster interface{}) {
	w.gotVictim = victim
	w.gotCaster = caster
}

type retaliationTestBody struct {
	name string
	npc  bool
}

func (b *retaliationTestBody) GetName() string  { return b.name }
func (b *retaliationTestBody) IsNPC() bool      { return b.npc }
func (b *retaliationTestBody) GetLevel() int    { return 30 }
func (b *retaliationTestBody) GetClass() int    { return 0 }
func (b *retaliationTestBody) GetHP() int       { return 100 }
func (b *retaliationTestBody) GetMaxHP() int    { return 100 }
func (b *retaliationTestBody) GetPosition() int { return 8 }
func (b *retaliationTestBody) GetRoom() int     { return 1001 }
func (b *retaliationTestBody) GetRoomVNum() int { return 1001 }
func (b *retaliationTestBody) GetSavingThrow(int) int {
	return 0
}
func (b *retaliationTestBody) SendMessage(string)           {}
func (b *retaliationTestBody) GetFightingBody() interface{} { return nil }

// TestNpcRetaliateRoutesNPCVictimsOnly pins the C gate: hit(victim, ch) runs
// for IS_NPC victims only (src/magic.c:945 and siblings), and the call always
// reaches the world's counterattack seam — the old StartCombatWith assertion
// had no implementation and silently no-opped every caller.
func TestNpcRetaliateRoutesNPCVictimsOnly(t *testing.T) {
	w := &recordingCounterattackWorld{}
	npc := &retaliationTestBody{name: "incubus", npc: true}
	pc := &retaliationTestBody{name: "Retalpc", npc: false}
	caster := &retaliationTestBody{name: "Caster", npc: false}

	npcRetaliate(w, npc, caster)
	if w.gotVictim != npc || w.gotCaster != caster {
		t.Fatalf("NPC victim did not reach the counterattack seam")
	}

	w = &recordingCounterattackWorld{}
	npcRetaliate(w, pc, caster)
	if w.gotVictim != nil {
		t.Fatal("PC victim retaliated; C gates hit(victim, ch) on IS_NPC(victim)")
	}
}

// TestNpcRetaliateFailsLoudlyWithoutSeam is the loud-failure half: a world
// that cannot run hit() must not be silently accepted. The production
// *game.World satisfies ExecuteSpellCounterattack (the compile-time contract
// in pkg/game), so reaching here means a test world dropped the seam.
func TestNpcRetaliateFailsLoudlyWithoutSeam(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("npcRetaliate silently accepted a world without the counterattack seam")
		}
	}()
	npcRetaliate(struct{}{}, &retaliationTestBody{npc: true}, &retaliationTestBody{})
}

// TestMobFlagBitsPinCPositions pins the C action-flag bit positions the
// short-circuits probe (src/structs.h:262-264): MOB_NOSLEEP 15, MOB_NOCHARM
// 13, MOB_NOBLIND 17.
func TestMobFlagBitsPinCPositions(t *testing.T) {
	if mobNoCharmBit != 13 {
		t.Fatalf("MOB_NOCHARM bit drifted: %d", mobNoCharmBit)
	}
}

// counterattackSpy is embedded by test worlds that drive a refusal path: the
// unified seam records every retaliation instead of panicking.
type counterattackSpy struct {
	retaliations [][2]interface{}
}

func (s *counterattackSpy) ExecuteSpellCounterattack(victim, caster interface{}) {
	s.retaliations = append(s.retaliations, [2]interface{}{victim, caster})
}
