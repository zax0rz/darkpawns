package spells

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestMagDamageBackfireAndPsiblastRecoilAssertCMessages selects each C
// random branch from deterministic stream seeds and pins its exact caster line.
func TestMagDamageBackfireAndPsiblastRecoilAssertCMessages(t *testing.T) {
	cases := []struct {
		name  string
		spell int
		want  string
	}{
		{name: "disintegrate", spell: SpellDisintegrate, want: "Your magick backfires!\r\n"},
		{name: "disrupt", spell: SpellDisrupt, want: "Your magick backfires!\r\n"},
		{name: "psiblast", spell: SpellPsiblast, want: "Suddenly, your psionic power recoils!\r\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			for seed := uint32(1); seed <= 2048; seed++ {
				dprng.ResetStream(seed)
				caster := &mockSpellsChar{
					name: "Mage", level: 20, class: 0, hp: 100_000, maxHP: 100_000, inventory: &mockInventory{},
				}
				victim := &mockSpellsChar{
					name: "Target", level: 20, hp: 100_000, maxHP: 100_000,
				}

				MagDamage(20, caster, victim, tc.spell, int(SaveSpell), nil)

				if reflect.DeepEqual(caster.messages, []string{tc.want}) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("did not select C %s branch with seeds 1..2048; want caster output %q", tc.name, tc.want)
			}
		})
	}
}

func TestMagDamageDispelProtectionNamesTarget(t *testing.T) {
	cases := []struct {
		name      string
		spellNum  int
		alignment int
	}{
		{"dispel evil protects a good target", SpellDispelEvil, 1000},
		{"dispel good protects an evil target", SpellDispelGood, -1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dprng.ResetStream(1)
			caster := newAffectMale("Castero", 8162)
			victim := newAffectMale("Barnaby", 8162)
			victim.alignment = tc.alignment

			MagDamage(20, caster, victim, tc.spellNum, int(SaveSpell), nil)

			if !reflect.DeepEqual(caster.messages, []string{"The gods protect Barnaby.\r\n"}) {
				t.Errorf("caster messages = %q, want target-substituted act output", caster.messages)
			}
			if len(victim.messages) != 0 {
				t.Errorf("victim messages = %q, want none", victim.messages)
			}
		})
	}
}

func TestMagDamageReagentRoomMessagesMatchC(t *testing.T) {
	cases := []struct {
		name       string
		spell      int
		reagent    string
		casterLine string
		roomLine   string
	}{
		{
			name:       "color spray",
			spell:      SpellColorSpray,
			reagent:    "prism",
			casterLine: "Pulling a prism from a pocket, you crush it under your heel...\r\n",
			roomLine:   "Mage pulls something out of his pocket and crushes it beneath his heel...\r\n",
		},
		{
			name:       "fireball",
			spell:      SpellFireball,
			reagent:    "pinch of ash",
			casterLine: "Pulling a pinch of ash from a pocket, you cast it about the room...\r\n",
			roomLine:   "Mage pulls a pinch of ash out of a pocket and casts it about the room.\r\n",
		},
		{
			name:       "disintegrate",
			spell:      SpellDisintegrate,
			reagent:    "eye of a beholder",
			casterLine: "Pulling the eye of a beholder from a pocket, you throw it to the ground...\r\n",
			roomLine:   "Mage pulls a small orb out of a pocket and dashes it to the ground.\r\n",
		},
		{
			name:       "energy drain",
			spell:      SpellEnergyDrain,
			reagent:    "vampire dust",
			casterLine: "Pulling the vampire dust from a pocket, you throw it into the air...\r\n",
			roomLine:   "Mage throws some dust into the air...\r\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dprng.ResetStream(1)
			caster := &mockSpellsChar{
				name:      "Mage",
				level:     30,
				class:     0, // CLASS_MAGIC_USER
				roomVNum:  8162,
				inventory: &mockInventory{items: []*mockItem{{shortDesc: tc.reagent}}},
			}
			victim := &mockSpellsChar{name: "Target", level: 20, roomVNum: 8162}
			witness := newAffectMale("Witness", 8162)
			world := newAffectMsgWorld(witness)

			MagDamage(30, caster, victim, tc.spell, int(SaveSpell), world)

			foundCasterLine := false
			for _, msg := range caster.messages {
				if msg == tc.casterLine {
					foundCasterLine = true
					break
				}
			}
			if !foundCasterLine {
				t.Errorf("caster output = %q, want C line %q", caster.messages, tc.casterLine)
			}
			if !reflect.DeepEqual(witness.messages, []string{tc.roomLine}) {
				t.Errorf("room output = %q, want C act output %q", witness.messages, tc.roomLine)
			}
			if len(caster.inventory.items) != 0 {
				t.Errorf("reagent remains in caster inventory: %q", caster.inventory.items[0].shortDesc)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DP-938: MagDamage must not silently fabricate damage for spells that have
// no case in C's mag_damage() switch (src/magic.c:615-819). Hellfire,
// MeteorSwarm, and Calliope are real MANUAL_SPELL routines ported separately
// (castHellfire/castMeteorSwarm/castCalliope — see area_spells_test.go).
// MentalLapse and Smokescreen are non-damage spells (mob-aggro reset and a
// blindness affect, respectively) handled elsewhere. Petrify is a
// look-triggered instant-kill in the Medusa special mob procedure, Drowning
// is a flat environmental HP tick, and the five breath spells plus the
// fabricated "DragonBreath" have no C mag_damage formula. The breath spells
// still pass through the generic mag_areas → mag_damage(0) path, so they deal
// zero damage while retaining C's save/state/message side effects. None of the
// 13 should route through MagDamage's dice-table fallback.
// ---------------------------------------------------------------------------

func TestMagDamage_DoesNotFabricateRemovedSpells(t *testing.T) {
	fabricated := []struct {
		name     string
		spellNum int
	}{
		{"Hellfire", SpellHellfire},
		{"MeteorSwarm", SpellMeteorSwarm},
		{"Calliope", SpellCalliope},
		{"Smokescreen", SpellSmokescreen},
		{"MentalLapse", SpellMentalLapse},
		{"FireBreath", SpellFireBreath},
		{"GasBreath", SpellGasBreath},
		{"FrostBreath", SpellFrostBreath},
		{"AcidBreath", SpellAcidBreath},
		{"LightningBreath", SpellLightningBreath},
		{"DragonBreath", SpellDragonBreath},
		{"Drowning", SpellDrowning},
		{"Petrify", SpellPetrify},
	}

	for _, tc := range fabricated {
		t.Run(tc.name, func(t *testing.T) {
			ch := &mockSpellsChar{name: "Caster", level: 40, class: 99}
			victim := &mockSpellsChar{name: "Victim", level: 20, maxHP: 100, hp: 100}

			MagDamage(40, ch, victim, tc.spellNum, int(SaveSpell), nil)

			if victim.hp != 100 {
				t.Errorf("DP-938: MagDamage dealt %d fabricated damage for %s (spell %d); expected 0 (not a real mag_damage case in C)",
					100-victim.hp, tc.name, tc.spellNum)
			}
		})
	}
}
