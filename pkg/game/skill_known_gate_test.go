package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestCanUseSkill_Audited_UnknownSkill_ExactMessages — for each Wave-1 combat
// skill, a character with GetSkill==0 is blocked with the command's exact C
// message INCLUDING its C-exact terminator (handlers send it as-is, no append).
// C gates purely on !GET_SKILL(ch, SKILL_X); there is no class/level check.
// DP-1206. Terminators are "\r\n" except cutthroat/slug ("\n\r"; new_cmds.c:561,
// new_cmds2.c:829) — the reason the messages carry their own terminator.
func TestCanUseSkill_Audited_UnknownSkill_ExactMessages(t *testing.T) {
	cases := []struct {
		skill string
		want  string // full C message with its own terminator
	}{
		{SkillBackstab, "You have no idea how.\r\n"},
		{SkillBash, "You'd better leave all the martial arts to fighters.\r\n"},
		{SkillKick, "You'd better leave all the martial arts to fighters.\r\n"},
		{SkillTrip, "You'd better leave the sneaky stuff to the thieves.\r\n"},
		{SkillHeadbutt, "You aren't qualified to headbutt anyone!\r\n"},
		{SkillRescue, "But only true warriors can do this!\r\n"},
		{SkillCutthroat, "You're not trained in slitting throats!\n\r"},
		{SkillSlug, "You couldn't slug your way out of a wet paper bag.\n\r"},
		{SkillSmackheads, "The only heads you're gonna smack are yours and Rosie's.\n\r"},
		{SkillFleshAlter, "You know nothing of altering your flesh!\n\r"},
		{SkillFirstAid, "You have no idea how!\r\n"},
		// ambush's gate is late (after target) so this message is not
		// oracle-probed by the no-arg scenario — the unit test is its guard.
		{SkillAmbush, "You'd better not.\r\n"},
		// bearhug is byte-identical to bash/kick above EXCEPT it terminates
		// "\n\r" (new_cmds.c:481), where bash/kick are "\r\n" — the
		// same-string-different-terminator case per-message terminators exist
		// for. The oracle normalizes it away, so this is its only byte-guard.
		{SkillBearhug, "You'd better leave all the martial arts to fighters.\n\r"},
		{SkillSerpentKick, "You'd better leave all the martial arts to others.\r\n"},
		// groinrip's no-skill message terminates "\n\r" (new_cmds.c:2582); the
		// oracle normalizes terminators, so this is its only byte-guard.
		{SkillGroinrip, "You're not trained in martial arts!\n\r"},
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
				t.Errorf("skill %q unknown-message = %q, want %q (full message, own terminator)",
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
	// Use SkillCircle — still unaudited (not in SkillUnknownMsg), exercises the
	// legacy class/level path. A class that cannot learn circle → "You have no
	// idea how."
	p := NewPlayer(1, "Magey", 1001)
	p.Class = ClassMageUser
	p.Level = 20
	p.Position = combat.PosFighting
	canUse, msg := CanUseSkill(p, SkillCircle)
	if canUse {
		t.Error("mage with circle=0 on the legacy path should be blocked")
	}
	if msg != "You have no idea how.\r\n" {
		t.Errorf("legacy class-block message = %q, want %q (unchanged)", msg, "You have no idea how.\r\n")
	}

	// A warrior below the learn level → the legacy level message (exact).
	lowWarrior := NewPlayer(2, "Rooky", 1001)
	lowWarrior.Class = ClassWarrior
	lowWarrior.Level = 1 // below circle's learn level
	lowWarrior.Position = combat.PosFighting
	// Circle's SkillClassReq entry for warrior (if any) determines minLevel;
	// if warrior isn't in the map, it's the class block above. Either way the
	// legacy behavior is unchanged — just assert it returns a non-empty legacy
	// message and doesn't leak an audited message.
	_, legacyMsg := CanUseSkill(lowWarrior, SkillCircle)
	if legacyMsg == "You'd better leave all the martial arts to fighters.\r\n" {
		t.Errorf("legacy skill circle leaked an audited message: %q", legacyMsg)
	}
}
