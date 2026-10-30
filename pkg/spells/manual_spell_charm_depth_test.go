package spells

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type charmDispatchWorld struct {
	followers  int
	circle     bool
	shopkeeper bool
	acts       []string
	firstHits  [][2]combat.Combatant
}

func (w *charmDispatchWorld) AddFollowerQuiet(interface{}, interface{}) {}
func (w *charmDispatchWorld) StopFollowerByName(string)                 {}
func (w *charmDispatchWorld) CircleFollowByName(string, string) bool    { return w.circle }
func (w *charmDispatchWorld) NumFollowers(string) int                   { return w.followers }
func (w *charmDispatchWorld) IsSpellCharmShopkeeper(interface{}) bool {
	return w.shopkeeper
}

func (w *charmDispatchWorld) SpellActToVictim(caster, victim interface{}, format string) bool {
	name := caster.(*mockSpellsChar).GetName()
	line := capitalizeActMessage(strings.ReplaceAll(format, "$n", name)) + "\r\n"
	sendToVictim(victim, line)
	w.acts = append(w.acts, format)
	return true
}

func (w *charmDispatchWorld) MobFirstHit(attacker, defender combat.Combatant) {
	w.firstHits = append(w.firstHits, [2]combat.Combatant{attacker, defender})
}

// ExecuteSpellCounterattack is the unified retaliation seam (npcRetaliate);
// MobFirstHit keeps its recording shape for the assertions below.
func (w *charmDispatchWorld) ExecuteSpellCounterattack(victim, caster interface{}) {
	a, _ := victim.(combat.Combatant)
	d, _ := caster.(combat.Combatant)
	w.firstHits = append(w.firstHits, [2]combat.Combatant{a, d})
}

func TestCharmManualDispatchMatchesCOutputGates(t *testing.T) {
	tests := []struct {
		name           string
		caster         *mockSpellsChar
		victim         *mockSpellsChar
		world          *charmDispatchWorld
		save           bool
		wantCaster     []string
		wantVictim     []string
		wantSuccessAct bool
	}{
		{
			name:       "self",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"You like yourself even better!\r\n"},
			wantVictim: []string{"You like yourself even better!\r\n"},
		},
		{
			name:       "sanctuary",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true, aff: 1 << affSanctuaryBit},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"Your victim is protected by sanctuary!\r\n"},
		},
		{
			name:       "no-charm mob bit",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true, mobFlags: 1 << mobNoCharmBit},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"Your victim resists!\r\n"},
		},
		{
			name:       "charmed caster",
			caster:     &mockSpellsChar{name: "Mage", level: 30, aff: 1 << affCharmBit},
			victim:     &mockSpellsChar{name: "Target", npc: true},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"You can't have any followers of your own!\r\n"},
		},
		{
			name:       "charmed victim",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true, aff: 1 << affCharmBit},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"You fail.\r\n"},
		},
		{
			name:       "level refusal",
			caster:     &mockSpellsChar{name: "Mage", level: 10},
			victim:     &mockSpellsChar{name: "Target", npc: true, level: 31},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"You fail.\r\n"},
		},
		{
			name:       "shopkeeper",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true},
			world:      &charmDispatchWorld{shopkeeper: true},
			wantCaster: []string{"Ha ha... Don't think so.\r\n"},
		},
		{
			name:       "following circle",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true},
			world:      &charmDispatchWorld{circle: true},
			wantCaster: []string{"Sorry, following in circles can not be allowed.\r\n"},
		},
		{
			name:       "saving throw",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true},
			world:      &charmDispatchWorld{},
			save:       true,
			wantCaster: []string{"Your victim resists!\r\n"},
		},
		{
			name:       "follower limit",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", npc: true},
			world:      &charmDispatchWorld{followers: 9},
			wantCaster: []string{"You can't have any more followers!\r\n"},
		},
		{
			name:           "follower limit uses charisma",
			caster:         &mockSpellsChar{name: "Mage", level: 30},
			victim:         &mockSpellsChar{name: "Target", npc: true},
			world:          &charmDispatchWorld{followers: 8},
			wantVictim:     []string{"Isn't Mage just such a nice fellow?\r\n"},
			wantSuccessAct: true,
		},
		{
			name:       "non-outlaw player",
			caster:     &mockSpellsChar{name: "Mage", level: 30},
			victim:     &mockSpellsChar{name: "Target", level: 5},
			world:      &charmDispatchWorld{},
			wantCaster: []string{"Your power fails to effect them because you are not an Outlaw!\r\n"},
			wantVictim: []string{"Mage tried to control you but failed because Mage in not an Outlaw!\r\n"},
		},
		{
			name:           "successful act",
			caster:         &mockSpellsChar{name: "Mage", level: 30},
			victim:         &mockSpellsChar{name: "Target", npc: true},
			world:          &charmDispatchWorld{},
			wantVictim:     []string{"Isn't Mage just such a nice fellow?\r\n"},
			wantSuccessAct: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSave := magCharmSaveRoll
			magCharmSaveRoll = func(interface{}, int) bool { return tt.save }
			t.Cleanup(func() { magCharmSaveRoll = oldSave })

			victim := tt.victim
			if victim == nil {
				victim = tt.caster
			}
			ExecuteManualSpell(SpellCharm, 30, tt.caster, victim, nil, "", tt.world)

			if !reflect.DeepEqual(tt.caster.messages, tt.wantCaster) {
				t.Fatalf("caster output = %#v, want %#v", tt.caster.messages, tt.wantCaster)
			}
			if !reflect.DeepEqual(victim.messages, tt.wantVictim) {
				t.Fatalf("victim output = %#v, want %#v", victim.messages, tt.wantVictim)
			}
			if got := len(tt.world.acts) != 0; got != tt.wantSuccessAct {
				t.Fatalf("success act dispatched = %t, want %t", got, tt.wantSuccessAct)
			}
		})
	}
}

func TestCharmSelfGateUsesBodyIdentity(t *testing.T) {
	caster := &mockSpellsChar{name: "Twin", level: 30}
	victim := &mockSpellsChar{name: "Twin", level: 30}
	oldSave := magCharmSaveRoll
	magCharmSaveRoll = func(interface{}, int) bool { return true }
	t.Cleanup(func() { magCharmSaveRoll = oldSave })

	ExecuteManualSpell(SpellCharm, 30, caster, victim, nil, "", &charmDispatchWorld{})

	if got, want := caster.messages, []string{"Your victim resists!\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("same-name distinct victim output = %#v, want %#v", got, want)
	}
}

func TestCharmSavingNPCUsesCanonicalOpeningHit(t *testing.T) {
	caster := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Mage", level: 30, class: 0, roomVNum: 8105,
		position: int(PosStanding), hp: 100, maxHP: 100,
	}}
	victim := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Spirit", npc: true, level: 30, class: 0, roomVNum: 8105,
		position: int(PosStanding), hp: 100, maxHP: 100,
	}}
	world := &charmDispatchWorld{}
	oldSave := magCharmSaveRoll
	magCharmSaveRoll = func(interface{}, int) bool { return true }
	t.Cleanup(func() { magCharmSaveRoll = oldSave })

	ExecuteManualSpell(SpellCharm, 30, caster, victim, nil, "", world)

	if len(world.firstHits) != 1 {
		t.Fatalf("opening hits = %d, want one", len(world.firstHits))
	}
	if world.firstHits[0][0] != victim || world.firstHits[0][1] != caster {
		t.Fatalf("opening hit = %v -> %v, want Spirit -> Mage", world.firstHits[0][0].GetName(), world.firstHits[0][1].GetName())
	}
	if got, want := caster.messages, []string{"Your victim resists!\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("caster output = %#v, want %#v", got, want)
	}
}
