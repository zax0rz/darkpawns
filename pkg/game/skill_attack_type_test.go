package game

import (
	"strings"
	"testing"
)

// C passes ONE value to damage() — the skill's own constant — and derives both
// the fight-message set and the corpse's long description from it
// (src/fight.c:1314-1545 damage(), src/fight.c:283-370 make_corpse). make_corpse
// groups SKILL_* and TYPE_* values separately, and several skills belong to no
// group at all and take the plain default line, so a substituted TYPE_* value
// silently changes the corpse bytes.
//
// The port used to map bash/kick/punch/tiger_punch/headbutt/smackheads/slug/
// serpent_kick to TYPE_BLUDGEON and every unknown name to TYPE_SLASH. That was
// invisible for the bludgeon group (same description) but wrong for the rest:
// trip, bearhug and strike reached the "hacked up, bloody" group where C prints
// the plain line.
func TestSkillDamageAttackTypeMatchesC(t *testing.T) {
	const (
		bruised = "The bruised, battered corpse of X is lying here."
		plain   = "The corpse of X is lying here."
		hacked  = "The hacked up, bloody corpse of X is lying here."
	)
	cases := []struct {
		skill      string
		wantType   int
		wantCorpse string
	}{
		{SkillBackstab, SkillBackstabNum, hacked}, // SKILL_BACKSTAB joins the slash group
		{SkillBash, SkillBashNum, bruised},        // SKILL_BASH is its own group
		{SkillKick, SkillKickNum, bruised},
		{SkillHeadbutt, SkillHeadbuttNum, bruised},
		{SkillDragonKick, SkillDragonKickNum, bruised},
		{SkillTigerPunch, SkillTigerPunchNum, bruised},
		{SkillSmackheads, SkillSmackheadsNum, bruised},
		{SkillSlug, SkillSlugNum, bruised},
		{SkillSerpentKick, SkillSerpentKickNum, bruised},
		{SkillBite, SkillBiteNum, hacked},
		{SkillCircle, SkillCircleNum, hacked},
		{SkillDisembowel, SkillDisembowelNum, "The corpse of X is lying here, guts spilled everywhere."},
		{SkillNeckbreak, SkillNeckbreakNum, "The corpse of X is lying here, his neck snapped in two."},
		{SkillTrip, SkillTripNum, plain}, // no group: C's default arm
		{SkillBearhug, SkillBearhugNum, plain},
		{SkillStrike, SkillStrikeNum, plain},
		{SkillAmbush, SkillAmbushNum, plain},
		{SkillCutthroat, SkillCutthroatNum, plain},
		{SkillGroinrip, SkillGroinripNum, plain},
		{SkillCharge, SkillChargeNum, plain},
		{"punch", SkillPunchNum, bruised},
	}
	for _, tc := range cases {
		gotType := skillToAttackType(tc.skill)
		if gotType != tc.wantType {
			t.Errorf("skillToAttackType(%q) = %d, want C's %d", tc.skill, gotType, tc.wantType)
			continue
		}
		got := corpseAttackLongDesc("X", attackTypeToCorpseAttack(gotType), "his")
		if got != tc.wantCorpse {
			t.Errorf("corpse for %q = %q, want %q", tc.skill, got, tc.wantCorpse)
		}
	}

	// An unknown name is C's "no specific attack type": make_corpse's
	// TYPE_UNDEFINED case and its default case print the same line.
	if got := skillToAttackType("not-a-skill"); got != TypeUndefined {
		t.Errorf("skillToAttackType(unknown) = %d, want TYPE_UNDEFINED (%d)", got, TypeUndefined)
	}
}

// A result that emits a skill_message set must hand that same key to the damage
// pipeline, because C derived both from one damage() argument. This is the
// guard for the results that never set DamageSkill: bash, kick, headbutt, trip,
// bearhug and backstab had correct messages and wrong corpses.
func TestSkillResultDamageAttackTypePrefersCSkillMessageKey(t *testing.T) {
	cases := []struct {
		name string
		r    SkillResult
		want int
	}{
		{"bash", SkillResult{SkillMsgType: SkillBashNum}, SkillBashNum},
		{"kick", SkillResult{SkillMsgType: SkillKickNum}, SkillKickNum},
		{"headbutt", SkillResult{SkillMsgType: SkillHeadbuttNum}, SkillHeadbuttNum},

		{"trip", SkillResult{SkillMsgType: SkillTripNum}, SkillTripNum},
		{"bearhug", SkillResult{SkillMsgType: SkillBearhugNum}, SkillBearhugNum},
		{"strike with a name too", SkillResult{SkillMsgType: SkillStrikeNum, DamageSkill: SkillStrike}, SkillStrikeNum},
		{"no skill_message uses the name", SkillResult{DamageSkill: SkillNeckbreak}, SkillNeckbreakNum},
		{"unknown name is TYPE_UNDEFINED", SkillResult{DamageSkill: "not-a-skill"}, TypeUndefined},
		{"empty result is TYPE_UNDEFINED", SkillResult{}, TypeUndefined},
	}
	for _, tc := range cases {
		if got := tc.r.DamageAttackType(); got != tc.want {
			t.Errorf("%s: DamageAttackType() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestSkillKillCorpseUsesCSkillAttackType drives the real skill-kill pipeline —
// ApplySkillDamageWithMessage, the function sendSkillResult calls — with the
// attack type a SkillResult hands it, and asserts the corpse's long description
// is the one C's make_corpse switch prints for that attack type
// (src/fight.c:283-370).
func TestSkillKillCorpseUsesCSkillAttackType(t *testing.T) {
	cases := []struct {
		name         string
		skillMsgType int
		wantFragment string
		wantAbsent   string
	}{
		// SKILL_HEADBUTT (141) joins the bruised/battered group.
		{"headbutt", SkillHeadbuttNum, "bruised, battered corpse of", "hacked up"},
		// SKILL_TRIP (144) belongs to no group: C's default arm prints the
		// plain line, which the pre-audit port replaced with TYPE_SLASH's.
		{"trip", SkillTripNum, "The corpse of", "hacked up"},
		// SKILL_BACKSTAB (131) really is in the slash group.
		{"backstab", SkillBackstabNum, "hacked up, bloody corpse of", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newCombatTestWorld(t)
			mob := spawnTargetMob(t, w)
			result := SkillResult{SkillMsgType: tc.skillMsgType}
			dam := mob.GetHP() + 50 // lethal whatever the prototype's HP
			w.ApplySkillDamageWithMessage(ch, mob, dam, result.DamageAttackType(), nil)

			for _, obj := range w.GetItemsInRoom(ch.GetRoom()) {
				if !obj.IsCorpse {
					continue
				}
				desc := obj.GetLongDesc()
				if !strings.Contains(desc, tc.wantFragment) {
					t.Fatalf("corpse long desc = %q, want it to contain %q", desc, tc.wantFragment)
				}
				if tc.wantAbsent != "" && strings.Contains(desc, tc.wantAbsent) {
					t.Fatalf("corpse long desc = %q, must not contain %q", desc, tc.wantAbsent)
				}
				return
			}
			t.Fatal("skill kill created no corpse")
		})
	}
}
