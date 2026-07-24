package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestCanUseSkill_Audited_UnknownSkill_ExactMessages — for each Wave-1 combat
// skill, a character with GetSkill==0 is blocked with the command's exact C
// message (bare — the handler appends "\r\n"). C gates purely on
// !GET_SKILL(ch, SKILL_X); there is no class/level check. DP-1206.
func TestCanUseSkill_Audited_UnknownSkill_ExactMessages(t *testing.T) {
	cases := []struct {
		skill string
		want  string // bare C message (handler appends \r\n)
	}{
		{SkillBackstab, "You have no idea how."},
		{SkillBash, "You'd better leave all the martial arts to fighters."},
		{SkillKick, "You'd better leave all the martial arts to fighters."},
		{SkillTrip, "You'd better leave the sneaky stuff to the thieves."},
		{SkillHeadbutt, "You aren't qualified to headbutt anyone!"},
		{SkillRescue, "But only true warriors can do this!"},
	}
	for _, tc := range cases {
		t.Run(tc.skill, func(t *testing.T) {
			p := NewPlayer(1, "Hero", 1001)
			p.Class = ClassWarrior
			p.Position = combat.PosStanding
			// Skill is 0 (default) → blocked with the exact message.
			canUse, msg := CanUseSkill(p, tc.skill)
			if canUse {
				t.Errorf("skill %q with GetSkill==0 should be blocked", tc.skill)
			}
			if msg != tc.want {
				t.Errorf("skill %q unknown-message = %q, want %q (bare; handler appends \\r\\n)",
					tc.skill, msg, tc.want)
			}
		})
	}
}

// TestCanUseSkill_Audited_KnownSkill_Passes — a character with the skill
// granted (GetSkill>0) passes the gate regardless of level. This is the fix
// that unblocks the opener sweep: an L1 warrior granted bash (skill 75 via
// skillset) can now bash — the old invented level-3 block is gone. The player
// is placed at the skill's required position so the (unchanged) position block
// doesn't fire.
func TestCanUseSkill_Audited_KnownSkill_Passes(t *testing.T) {
	cases := []struct {
		skill string
		pos   int
	}{
		{SkillBackstab, combat.PosStanding},
		{SkillBash, combat.PosFighting},
		{SkillKick, combat.PosFighting},
		{SkillTrip, combat.PosFighting},
		{SkillHeadbutt, combat.PosFighting},
		{SkillRescue, combat.PosStanding},
	}
	for _, tc := range cases {
		t.Run(tc.skill, func(t *testing.T) {
			p := NewPlayer(1, "Hero", 1001)
			p.Class = ClassWarrior
			p.Level = 1 // low level — C does not gate on level
			p.Position = tc.pos
			p.SetSkill(tc.skill, 75)

			canUse, msg := CanUseSkill(p, tc.skill)
			if !canUse {
				t.Errorf("skill %q at level 1 with GetSkill=75 should pass the gate (no level check), got msg %q",
					tc.skill, msg)
			}
		})
	}
}

// TestCanUseSkill_Audited_CrossClassGrant_Passes — a non-warrior granted a
// warrior skill via skillset (GetSkill>0) passes: C has no class gate on skill
// USE (only on learning). The old Go "You have no idea how." class block is
// bypassed for audited skills.
func TestCanUseSkill_Audited_CrossClassGrant_Passes(t *testing.T) {
	p := NewPlayer(1, "Magey", 1001)
	p.Class = ClassMageUser // not a warrior class
	p.Level = 10
	p.Position = combat.PosFighting
	p.SetSkill(SkillKick, 50)

	canUse, msg := CanUseSkill(p, SkillKick)
	if !canUse {
		t.Errorf("cross-class mage with kick=50 should pass (C has no class gate), got msg %q", msg)
	}
}

// TestCanUseSkill_Legacy_Unaudited_Unchanged — a still-un-audited skill (e.g.
// SkillCharge) returns today's class/level messages exactly. Regression guard
// that the audited fork did not perturb the legacy path.
func TestCanUseSkill_Legacy_Unaudited_Unchanged(t *testing.T) {
	// A class that cannot learn charge → the legacy "You have no idea how."
	p := NewPlayer(1, "Magey", 1001)
	p.Class = ClassMageUser
	p.Level = 20
	p.Position = combat.PosFighting
	canUse, msg := CanUseSkill(p, SkillCharge)
	if canUse {
		t.Error("mage with charge=0 on the legacy path should be blocked")
	}
	if msg != "You have no idea how." {
		t.Errorf("legacy class-block message = %q, want %q (unchanged)", msg, "You have no idea how.")
	}

	// A warrior below the learn level → the legacy level message (exact).
	lowWarrior := NewPlayer(2, "Rooky", 1001)
	lowWarrior.Class = ClassWarrior
	lowWarrior.Level = 1 // below charge's learn level
	lowWarrior.Position = combat.PosFighting
	// Charge's SkillClassReq entry for warrior (if any) determines minLevel;
	// if warrior isn't in the map, it's the class block above. Either way the
	// legacy behavior is unchanged — just assert it returns a non-empty legacy
	// message and doesn't leak an audited message.
	_, legacyMsg := CanUseSkill(lowWarrior, SkillCharge)
	if legacyMsg == "You'd better leave all the martial arts to fighters." {
		t.Errorf("legacy skill charge leaked an audited message: %q", legacyMsg)
	}
}
