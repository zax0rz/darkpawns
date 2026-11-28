package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// corpseLongDesc returns the long description of the single corpse in room.
func corpseLongDesc(t *testing.T, w *World, roomVNum int) string {
	t.Helper()
	var found []string
	for _, item := range w.GetItemsInRoom(roomVNum) {
		if item.IsCorpse {
			found = append(found, item.Runtime.LongDesc)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d corpses in room %d, want 1: %q", len(found), roomVNum, found)
	}
	return found[0]
}

// setWieldedAttackType rewrites the equipped test sword's val3, which is C's
// weapon attack-type offset (fight.c:1795 w_type = GET_OBJ_VAL(wielded, 3) +
// TYPE_HIT). 3=slash, 5=bludgeon, 6=crush, 11=pierce.
func setWieldedAttackType(t *testing.T, p *Player, offset int) {
	t.Helper()
	weapon, ok := p.Equipment.GetItemInSlot(SlotWield)
	if !ok || weapon == nil || weapon.Prototype == nil {
		t.Fatal("test sword is not wielded")
	}
	weapon.Prototype.Values[3] = offset
}

// TestWeaponKillCorpseWordingOnLivePath is the R1 proof for every weapon group
// make_corpse names (src/fight.c:283-370). The kill goes through the real path:
// do_hit's deferred initial swing while the attacker is already fighting
// (the shape play-lab s00000 reaches after kicking the recruit), so the value
// must travel with the damage call rather than ride on the combat pair.
func TestWeaponKillCorpseWordingOnLivePath(t *testing.T) {
	cases := []struct {
		name   string
		offset int
		want   string
	}{
		{"slash", 3, "The hacked up, bloody corpse of fish is lying here."},
		{"pierce", 11, "The well-ventilated corpse of fish is lying here."},
		{"bludgeon", 5, "The bruised, battered corpse of fish is lying here."},
		{"crush", 6, "The crushed, barely recognizable corpse of fish is lying here."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origRoller := combat.GetRoller()
			t.Cleanup(func() { combat.SetRoller(origRoller) })
			combat.SetRoller(combat.NewSeededRoller(3))
			w, god, mob, _, _ := secondDeathWorld(t)
			setWieldedAttackType(t, god, tc.offset)

			ce := w.combatEngine.(*combat.CombatEngine)
			// Already engaged: the play session kicks before it kills.
			if err := ce.StartCombat(god, mob); err != nil {
				t.Fatalf("StartCombat: %v", err)
			}
			if err := ce.PerformDeferredInitialAttack(god, mob); err != nil {
				t.Fatalf("PerformDeferredInitialAttack: %v", err)
			}
			if mob.IsAlive() {
				t.Fatalf("swing left the mob alive at %d HP; the seed must land the kill", mob.GetHP())
			}
			if got := corpseLongDesc(t, w, 1001); got != tc.want {
				t.Errorf("corpse long desc = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSkillKillCorpseWordingOnLivePath kills through the skill damage seam
// World.ApplySkillDamage, which is C damage(ch, vict, dam, SKILL_KICK)
// (src/act.offensive.c:627-630): the skill number, not the wielded weapon, must
// reach make_corpse.
func TestSkillKillCorpseWordingOnLivePath(t *testing.T) {
	origRoller := combat.GetRoller()
	t.Cleanup(func() { combat.SetRoller(origRoller) })
	combat.SetRoller(combat.NewSeededRoller(3))
	w, god, mob, _, _ := secondDeathWorld(t)

	if !w.ApplySkillDamage(god, mob, 60, SkillKickNum) {
		t.Fatal("ApplySkillDamage refused a lethal kick")
	}
	if mob.IsAlive() {
		t.Fatalf("kick left the mob alive at %d HP", mob.GetHP())
	}
	want := "The bruised, battered corpse of fish is lying here."
	if got := corpseLongDesc(t, w, 1001); got != want {
		t.Errorf("corpse long desc = %q, want %q", got, want)
	}
}

// TestSpellKillCorpseWordingOnLivePath kills through World.DoSpellDamage, the
// area-spell seam (hellfire, meteor swarm): C mag_damage passes the spell
// number to damage(), and make_corpse reads it (SPELL_HELLFIRE -> the charred
// wording, src/fight.c:294-299).
func TestSpellKillCorpseWordingOnLivePath(t *testing.T) {
	origRoller := combat.GetRoller()
	t.Cleanup(func() { combat.SetRoller(origRoller) })
	combat.SetRoller(combat.NewSeededRoller(3))
	w, god, mob, _, _ := secondDeathWorld(t)

	if !w.DoSpellDamage(god, mob, 60, "hellfire") {
		t.Fatal("DoSpellDamage refused a lethal hellfire")
	}
	if mob.IsAlive() {
		t.Fatalf("hellfire left the mob alive at %d HP", mob.GetHP())
	}
	want := "The charred corpse of fish is lying here, still smoking."
	if got := corpseLongDesc(t, w, 1001); got != want {
		t.Errorf("corpse long desc = %q, want %q", got, want)
	}
}
