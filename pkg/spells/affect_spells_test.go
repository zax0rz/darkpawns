package spells

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// mockSpellsChar satisfies all local interfaces in affect_spells.go
type mockSpellsChar struct {
	name          string
	npc           bool
	level         int
	class         int
	sex           int
	roomVNum      int
	position      int
	hp            int
	maxHP         int
	move          int
	maxMove       int
	dex           int
	aff           uint64
	flags         uint64
	inGroup       bool
	following     string
	messages      []string
	activeAffects []*engine.Affect
	inventory     *mockInventory
}

func (m *mockSpellsChar) GetName() string  { return m.name }
func (m *mockSpellsChar) IsNPC() bool      { return m.npc }
func (m *mockSpellsChar) GetFlags() uint64 { return m.flags }
func (m *mockSpellsChar) HasSpellAffect(n int) bool {
	for _, aff := range m.activeAffects {
		if aff.SpellID == n {
			return true
		}
	}
	return false
}
func (m *mockSpellsChar) GetLevel() int          { return m.level }
func (m *mockSpellsChar) GetClass() int          { return m.class }
func (m *mockSpellsChar) GetSex() int            { return m.sex }
func (m *mockSpellsChar) GetRoomVNum() int       { return m.roomVNum }
func (m *mockSpellsChar) GetPosition() int       { return m.position }
func (m *mockSpellsChar) SetPosition(pos int)    { m.position = pos }
func (m *mockSpellsChar) GetHP() int             { return m.hp }
func (m *mockSpellsChar) GetMaxHP() int          { return m.maxHP }
func (m *mockSpellsChar) SetHP(hp int)           { m.hp = hp }
func (m *mockSpellsChar) SetHealth(hp int)       { m.hp = hp }
func (m *mockSpellsChar) GetMove() int           { return m.move }
func (m *mockSpellsChar) GetMaxMove() int        { return m.maxMove }
func (m *mockSpellsChar) SetMove(mv int)         { m.move = mv }
func (m *mockSpellsChar) SendMessage(msg string) { m.messages = append(m.messages, msg) }
func (m *mockSpellsChar) AddAffect(aff *engine.Affect) {
	m.activeAffects = append(m.activeAffects, aff)
}

func (m *mockSpellsChar) RemoveAffectBySpell(spellNum int) {
	filtered := m.activeAffects[:0]
	for _, aff := range m.activeAffects {
		if aff.SpellID != spellNum {
			filtered = append(filtered, aff)
		}
	}
	m.activeAffects = filtered
}
func (m *mockSpellsChar) GetDex() int                { return m.dex }
func (m *mockSpellsChar) IsInGroup() bool            { return m.inGroup }
func (m *mockSpellsChar) GetFollowing() string       { return m.following }
func (m *mockSpellsChar) IsAffected(bit int) bool    { return m.aff&(1<<bit) != 0 }
func (m *mockSpellsChar) HasMobFlag(bit uint64) bool { return m.flags&bit != 0 }
func (m *mockSpellsChar) GetCha() int                { return 18 }
func (m *mockSpellsChar) GetInventory() ReagentInventory {
	if m.inventory == nil {
		return nil
	}
	return m.inventory
}

func TestMagAffects_ChillTouch(t *testing.T) {
	ch := &mockSpellsChar{level: 10, class: 0}
	victim := &mockSpellsChar{level: 5}

	MagAffects(10, ch, victim, SpellChillTouch, int(SaveSpell), nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellChillTouch {
		t.Errorf("SpellID = %d, want %d", aff.SpellID, SpellChillTouch)
	}
	if aff.Location != engine.ApplyStr {
		t.Errorf("Location = %d, want %d (ApplyStr)", aff.Location, engine.ApplyStr)
	}
	if aff.Magnitude != -1 {
		t.Errorf("Magnitude = %d, want -1", aff.Magnitude)
	}
}

func TestMagAffects_Bless(t *testing.T) {
	ch := &mockSpellsChar{level: 20}
	victim := &mockSpellsChar{level: 10}

	MagAffects(20, ch, victim, SpellBless, int(SaveSpell), nil)

	// Bless applies two affects: hitroll +2 and saving spell -2.
	if len(victim.activeAffects) != 2 {
		t.Fatalf("expected 2 affects on victim for Bless, got %d", len(victim.activeAffects))
	}
	aff1 := victim.activeAffects[0]
	if aff1.Location != engine.ApplyHitroll || aff1.Magnitude != 2 {
		t.Errorf("unexpected bless hitroll affect: %+v", aff1)
	}
	aff2 := victim.activeAffects[1]
	if aff2.Location != engine.ApplySavingSpell || aff2.Magnitude != -2 {
		t.Errorf("unexpected bless save affect: %+v", aff2)
	}
}

func TestMagAffects_Armor(t *testing.T) {
	ch := &mockSpellsChar{level: 10}
	victim := &mockSpellsChar{level: 10}

	MagAffects(10, ch, victim, SpellArmor, int(SaveSpell), nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellArmor || aff.Location != engine.ApplyAC || aff.Magnitude != -15 {
		t.Errorf("unexpected armor affect: %+v", aff)
	}
}

func TestMagAffects_Sleep(t *testing.T) {
	ch := &mockSpellsChar{level: 30, flags: 1}        // PLR_OUTLAW
	victim := &mockSpellsChar{level: 30, position: 8} // standing = 8

	// Sleep has a saving throw. Loop up to 50 times until the save fails and spell succeeds.
	succeeded := false
	for i := 0; i < 50; i++ {
		victim.activeAffects = nil
		victim.position = 8
		MagAffects(30, ch, victim, SpellSleep, int(SaveSpell), nil)
		if len(victim.activeAffects) > 0 {
			succeeded = true
			break
		}
	}

	if !succeeded {
		t.Fatal("failed to land Sleep spell after 50 retries")
	}

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellSleep {
		t.Errorf("SpellID = %d, want %d", aff.SpellID, SpellSleep)
	}
	if victim.position != 4 { // PosSleeping = 4
		t.Errorf("victim position = %d, want 4 (PosSleeping)", victim.position)
	}
}

func TestMagAffects_Poison(t *testing.T) {
	ch := &mockSpellsChar{level: 20}
	victim := &mockSpellsChar{level: 10}

	// Poison has a saving throw. Loop up to 50 times until the save fails and spell succeeds.
	succeeded := false
	for i := 0; i < 50; i++ {
		victim.activeAffects = nil
		MagAffects(20, ch, victim, SpellPoison, int(SaveSpell), nil)
		if len(victim.activeAffects) > 0 {
			succeeded = true
			break
		}
	}

	if !succeeded {
		t.Fatal("failed to land Poison spell after 50 retries")
	}

	if len(victim.activeAffects) != 3 {
		t.Fatalf("expected 3 affects on victim for Poison, got %d", len(victim.activeAffects))
	}
	// Direct, Str, and Hitroll affects
	var foundStr, foundHitroll, foundDirect bool
	for _, aff := range victim.activeAffects {
		if aff.SpellID == SpellPoison {
			switch aff.Location {
			case engine.ApplyStr:
				foundStr = true
			case engine.ApplyHitroll:
				foundHitroll = true
			case engine.ApplyNone:
				foundDirect = true
			}
		}
	}

	if !foundStr || !foundHitroll || !foundDirect {
		t.Errorf("missing expected poison affects: str=%t, hitroll=%t, direct=%t", foundStr, foundHitroll, foundDirect)
	}
}

func TestMagAffects_Metalskin(t *testing.T) {
	ch := &mockSpellsChar{
		level: 30,
		inventory: &mockInventory{
			items: []*mockItem{
				{shortDesc: "chunk of iron"},
			},
		},
	}
	victim := &mockSpellsChar{level: 10}

	MagAffects(30, ch, victim, SpellMetalskin, int(SaveSpell), nil)

	// Metalskin applies two affects: AFF_METALSKIN flag and AC modifier.
	if len(victim.activeAffects) != 2 {
		t.Fatalf("expected 2 affects for Metalskin, got %d", len(victim.activeAffects))
	}
	aff1 := victim.activeAffects[0]
	if aff1.SpellID != SpellMetalskin || aff1.Flags != engine.AFFMetalskin {
		t.Errorf("unexpected metalskin flag affect: %+v", aff1)
	}
	aff2 := victim.activeAffects[1]
	if aff2.SpellID != SpellMetalskin || aff2.Location != engine.ApplyAC {
		t.Errorf("unexpected metalskin AC affect: %+v", aff2)
	}
}

func TestMagPoints(t *testing.T) {
	tests := []struct {
		spell    int
		initial  int
		max      int
		wantHP   int
		expected string
	}{
		{SpellCureLight, 50, 100, 50, "better"},
		{SpellCureCritic, 50, 100, 50, "lot better"},
		{SpellHeal, 50, 200, 150, "warm feeling"},
		{SpellVitality, 50, 150, 50, "vitalized"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ch := &mockSpellsChar{level: 20}
			victim := &mockSpellsChar{hp: tt.initial, maxHP: tt.max}

			MagPoints(20, ch, victim, tt.spell, int(SaveSpell), nil)

			if victim.hp <= tt.initial {
				t.Errorf("Heal failed: HP remained at %d", victim.hp)
			}
			if victim.hp > tt.max {
				t.Errorf("HP overflowed: %d, max is %d", victim.hp, tt.max)
			}

			// Verify text message
			foundMsg := false
			for _, m := range victim.messages {
				if strings.Contains(m, tt.expected) {
					foundMsg = true
					break
				}
			}
			if !foundMsg {
				t.Errorf("expected msg containing %q, got messages: %v", tt.expected, victim.messages)
			}
		})
	}
}

// TestMagUnaffects pins mag_unaffects (magic.c:1828-1876): the cured pair for
// the vision arm is BLINDNESS and SMOKESCREEN, unaffected victims answer
// NOEFFECT to the caster (SILENT for heal/mass-heal, which ride along other
// routines), and the removed affect yields "Your vision returns!".
func victimText(m *mockSpellsChar) string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[0]
}

func TestMagUnaffects(t *testing.T) {
	newVictim := func() *mockSpellsChar {
		return &mockSpellsChar{
			activeAffects: []*engine.Affect{
				engine.NewAffect(SpellBlindness, engine.ApplyHitroll, 5, -2, "blind"),
				engine.NewAffect(SpellPoison, engine.ApplyStr, 5, -2, "poison"),
				engine.NewAffect(SpellCurse, engine.ApplyHitroll, 5, -2, "curse"),
			},
		}
	}

	t.Run("cures", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := newVictim()

		MagUnaffects(20, ch, victim, SpellCureBlind, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellBlindness {
				t.Error("SpellBlindness was not removed")
			}
		}
		if got := victimText(victim); got != "Your vision returns!\r\n" {
			t.Errorf("vision to_vict = %q, want C's 'Your vision returns!'", got)
		}

		MagUnaffects(20, ch, victim, SpellRemovePoison, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellPoison {
				t.Error("SpellPoison was not removed")
			}
		}

		MagUnaffects(20, ch, victim, SpellRemoveCurse, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellCurse {
				t.Error("SpellCurse was not removed")
			}
		}
	})

	t.Run("mass heal on unblinded drinker is silent", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := &mockSpellsChar{} // no affects
		MagUnaffects(20, ch, victim, SpellMassHeal, nil)
		if got := victimText(victim); got != "" {
			t.Errorf("victim output = %q, want silence (guard returns before bytes)", got)
		}
		if got := victimText(ch); got != "" {
			t.Errorf("caster output = %q, want silence (heal/mass-heal never NOEFFECT)", got)
		}
	})

	t.Run("cure blind on unblinded victim answers caster NOEFFECT", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := &mockSpellsChar{}
		MagUnaffects(20, ch, victim, SpellCureBlind, nil)
		if got := victimText(ch); got != "Nothing seems to happen.\r\n" {
			t.Errorf("caster output = %q, want NOEFFECT", got)
		}
		if got := victimText(victim); got != "" {
			t.Errorf("victim output = %q, want none", got)
		}
	})
}

// mockCorpse satisfies the interfaces MagSummons uses to find a corpse.
type mockCorpse struct{ keywords string }

func (m *mockCorpse) GetKeywords() string { return m.keywords }

// mockAnimateWorld is a minimal world for testing Animate Dead.
type mockAnimateWorld struct {
	items             []interface{}
	removed           bool
	spawnErr          error
	spawnedVNum       int
	spawnedRoom       int
	canRaise          bool
	canRaiseMsg       string
	canRaiseCalled    bool
	charmFollowCalled bool
	charmed           bool
	following         string
}

func (w *mockAnimateWorld) GetItemsInRoomI(roomVNum int) []interface{} { return w.items }
func (w *mockAnimateWorld) RemoveItemFromRoomI(item interface{}, roomVNum int) {
	w.removed = true
}

func (w *mockAnimateWorld) SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error) {
	w.spawnedVNum = vnum
	w.spawnedRoom = roomVNum
	if w.spawnErr != nil {
		return nil, w.spawnErr
	}
	return &mockSpellsChar{name: "zombie"}, nil
}

func (w *mockAnimateWorld) CanRaiseUndeadI(ch interface{}) (bool, string) {
	w.canRaiseCalled = true
	return w.canRaise, w.canRaiseMsg
}

func (w *mockAnimateWorld) CharmAndFollowI(mob, leader interface{}) {
	w.charmFollowCalled = true
	if m, ok := mob.(*mockSpellsChar); ok {
		m.aff |= 1 << 21 // affCharm bit
		w.charmed = true
	}
	if l, ok := leader.(*mockSpellsChar); ok {
		w.following = l.name
	}
}

// forceAnimateDeadRoll makes the SPELL_ANIMATE_DEAD pfail roll deterministic for
// the duration of a test: val < 8 always fails the roll, val >= 8 always passes.
// Restores the real (rand-backed) roll via t.Cleanup.
func forceAnimateDeadRoll(t *testing.T, val int) {
	t.Helper()
	prev := animateDeadPfailRoll
	animateDeadPfailRoll = func() int { return val }
	t.Cleanup(func() { animateDeadPfailRoll = prev })
}

func TestMagSummons_AnimateDead_KeepsCorpseOnSpawnFailure(t *testing.T) {
	forceAnimateDeadRoll(t, 101) // pass the pfail so we reach the spawn attempt
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		spawnErr: fmt.Errorf("spawn failed"),
		canRaise: true,
	}

	MagSummons(10, caster, SpellAnimateDead, world)

	if world.removed {
		t.Error("corpse was removed even though spawn failed")
	}
	if world.spawnedVNum != 10 {
		t.Errorf("expected zombie vnum 10, got %d", world.spawnedVNum)
	}
}

func TestMagSummons_AnimateDead_PfailKeepsCorpseNoSpawn(t *testing.T) {
	forceAnimateDeadRoll(t, 0) // 0 < 8 -> the pfail roll fails
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		canRaise: true,
	}

	MagSummons(10, caster, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Errorf("pfail should abort before spawn, but spawned vnum %d", world.spawnedVNum)
	}
	if world.removed {
		t.Error("pfail should not remove the corpse")
	}
	foundFailMsg := false
	for _, m := range caster.messages {
		if strings.Contains(m, "You failed") {
			foundFailMsg = true
			break
		}
	}
	if !foundFailMsg {
		t.Errorf("expected pfail message, got %v", caster.messages)
	}
}

func TestMagSummons_AnimateDead_GiddyBlocked(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:       []interface{}{corpse},
		canRaise:    false,
		canRaiseMsg: "You are too giddy to have any followers!\r\n",
	}

	MagSummons(10, caster, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Error("expected no spawn when caster is charmed")
	}
	if world.removed {
		t.Error("corpse should not be removed when blocked")
	}
	if !strings.Contains(caster.messages[0], "too giddy") {
		t.Errorf("expected giddy message, got %q", caster.messages)
	}
}

func TestMagSummons_AnimateDead_FollowerCapBlocked(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:       []interface{}{corpse},
		canRaise:    false,
		canRaiseMsg: "You can't have any more followers!\r\n",
	}

	MagSummons(10, caster, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Error("expected no spawn at follower cap")
	}
	if world.removed {
		t.Error("corpse should not be removed when blocked")
	}
	if !strings.Contains(caster.messages[0], "can't have any more followers") {
		t.Errorf("expected cap message, got %q", caster.messages)
	}
}

func TestMagSummons_AnimateDead_Success(t *testing.T) {
	forceAnimateDeadRoll(t, 101) // pass the pfail so the spawn path runs
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		canRaise: true,
	}

	MagSummons(10, caster, SpellAnimateDead, world)
	if world.spawnedVNum != 10 {
		t.Fatalf("expected zombie vnum 10 spawned, got %d", world.spawnedVNum)
	}

	if !world.removed {
		t.Error("corpse should be removed on successful spawn")
	}
	if !world.charmFollowCalled {
		t.Error("CharmAndFollowI should be called on successful spawn")
	}
	if !world.charmed {
		t.Error("spawned mob should be charmed")
	}
	if world.following != caster.name {
		t.Errorf("expected mob to follow %q, got %q", caster.name, world.following)
	}
	foundSuccessMsg := false
	for _, m := range caster.messages {
		if strings.Contains(m, "stands with a life of its own") {
			foundSuccessMsg = true
			break
		}
	}
	if !foundSuccessMsg {
		t.Errorf("expected success message, got %v", caster.messages)
	}
}
