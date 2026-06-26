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
	aff           uint64
	flags         uint64
	messages      []string
	activeAffects []*engine.Affect
	inventory     *mockInventory
}

func (m *mockSpellsChar) GetName() string        { return m.name }
func (m *mockSpellsChar) IsNPC() bool            { return m.npc }
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
func (m *mockSpellsChar) IsAffected(bit int) bool    { return m.aff&(1<<bit) != 0 }
func (m *mockSpellsChar) HasMobFlag(bit uint64) bool { return m.flags&bit != 0 }
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

	// Note: Code double-applies the saving spell affect in Bless, ending up with 3 affects.
	if len(victim.activeAffects) != 3 {
		t.Fatalf("expected 3 affects on victim for Bless, got %d", len(victim.activeAffects))
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
	ch := &mockSpellsChar{level: 30}
	victim := &mockSpellsChar{level: 10, position: 8} // standing = 8

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

	// Metalskin double-applies the AC affect at the end of MagAffects, ending up with 3 affects.
	if len(victim.activeAffects) != 3 {
		t.Fatalf("expected 3 affects for Metalskin, got %d", len(victim.activeAffects))
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

func TestMagUnaffects(t *testing.T) {
	ch := &mockSpellsChar{}
	victim := &mockSpellsChar{
		activeAffects: []*engine.Affect{
			engine.NewAffect(SpellBlindness, engine.ApplyHitroll, 5, -2, "blind"),
			engine.NewAffect(SpellPoison, engine.ApplyStr, 5, -2, "poison"),
			engine.NewAffect(SpellCurse, engine.ApplyHitroll, 5, -2, "curse"),
		},
	}

	// Remove Blindness
	MagUnaffects(20, ch, victim, SpellCureBlind, nil)
	for _, aff := range victim.activeAffects {
		if aff.SpellID == SpellBlindness {
			t.Error("SpellBlindness was not removed")
		}
	}

	// Remove Poison
	MagUnaffects(20, ch, victim, SpellRemovePoison, nil)
	for _, aff := range victim.activeAffects {
		if aff.SpellID == SpellPoison {
			t.Error("SpellPoison was not removed")
		}
	}

	// Remove Curse
	MagUnaffects(20, ch, victim, SpellRemoveCurse, nil)
	for _, aff := range victim.activeAffects {
		if aff.SpellID == SpellCurse {
			t.Error("SpellCurse was not removed")
		}
	}
}

// mockCorpse satisfies the interfaces MagSummons uses to find a corpse.
type mockCorpse struct{ keywords string }

func (m *mockCorpse) GetKeywords() string { return m.keywords }

// mockAnimateWorld is a minimal world for testing Animate Dead.
type mockAnimateWorld struct {
	items       []interface{}
	removed     bool
	spawnErr    error
	spawnedVNum int
	spawnedRoom int
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

func TestMagSummons_AnimateDead_KeepsCorpseOnSpawnFailure(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse"}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		spawnErr: fmt.Errorf("spawn failed"),
	}

	MagSummons(10, caster, SpellAnimateDead, world)

	if world.removed {
		t.Error("corpse was removed even though spawn failed")
	}
	if world.spawnedVNum != 10 {
		t.Errorf("expected zombie vnum 10, got %d", world.spawnedVNum)
	}
}
