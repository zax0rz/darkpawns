package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestDoSerpentKickResultContract pins the command-specific C contract from
// new_cmds2.c:693-743. The damage call owns set_fighting and the numbered
// skill-message selection; the command wrapper owns the post-damage training
// draw and deferred improvement.
func TestDoSerpentKickResultContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.Level = 20
	ch.SetSkill(SkillSerpentKick, 100)
	mob := spawnTargetMob(t, w)

	var result SkillResult
	var hit bool
	for seed := uint32(1); seed <= 100; seed++ {
		dprng.ResetStream(seed)
		result = DoSerpentKick(ch, mob, w)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("expected a serpent-kick hit with skill 100 within 100 seeds")
	}

	if result.Damage != 30 {
		t.Errorf("hit damage = %d, want 30 (int(level*1.5))", result.Damage)
	}
	if result.SkillMsgType != SkillSerpentKickNum {
		t.Errorf("skill message set = %d, want %d", result.SkillMsgType, SkillSerpentKickNum)
	}
	if result.DamageSkill != SkillSerpentKick {
		t.Errorf("damage skill = %q, want %q", result.DamageSkill, SkillSerpentKick)
	}
	if !result.StartCombat {
		t.Error("hit should start combat through damage()")
	}
	if result.WaitCh != 2 {
		t.Errorf("hit wait = %d, want 2 violence pulses", result.WaitCh)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillSerpentKick {
		t.Errorf("deferred improvement = %v, want one serpent-kick improvement", result.DeferredImprove)
	}
	if result.SpawnMobVNum != 18221 || result.SpawnMobLevel != 23 || result.SpawnMobRoom != 18201 || !result.SpawnMobHunting {
		t.Errorf("training spawn = vnum %d level %d room %d hunting %v, want 18221/23/18201/true",
			result.SpawnMobVNum, result.SpawnMobLevel, result.SpawnMobRoom, result.SpawnMobHunting)
	}
}

func TestDoSerpentKickMissUsesCMessageContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.SetSkill(SkillSerpentKick, 1)
	mob := spawnTargetMob(t, w)

	var result SkillResult
	var miss bool
	for seed := uint32(1); seed <= 100; seed++ {
		dprng.ResetStream(seed)
		result = DoSerpentKick(ch, mob, w)
		if !result.Success {
			miss = true
			break
		}
	}
	if !miss {
		t.Fatal("expected a serpent-kick miss with skill 1 within 100 seeds")
	}
	if result.Damage != 0 || result.SkillMsgType != SkillSerpentKickNum || !result.StartCombat {
		t.Errorf("miss contract = damage %d set %d start %v, want 0/%d/true",
			result.Damage, result.SkillMsgType, result.StartCombat, SkillSerpentKickNum)
	}
	if result.WaitCh != 2 {
		t.Errorf("miss wait = %d, want 2 violence pulses", result.WaitCh)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss carries invented literal messages: ch=%q victim=%q room=%q",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}
}

func TestDoSerpentKickSleepingTargetAutoHits(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.SetSkill(SkillSerpentKick, 1)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)
	ac := 50 // keeps the C prob=110 override above every percent draw.
	mob.Runtime.ACOverride = &ac

	dprng.ResetStream(1)
	result := DoSerpentKick(ch, mob, w)
	if !result.Success {
		t.Fatalf("sleeping target should auto-hit under C prob=110, got %+v", result)
	}
	if result.Damage != 15 {
		t.Errorf("sleeping-target damage = %d, want 15 at level 10", result.Damage)
	}
}

func TestDoSerpentKickGateOrder(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.SetSkill(SkillSerpentKick, 100)
	mob := spawnTargetMob(t, w)

	self := DoSerpentKick(ch, ch, w)
	if self.MessageToCh != "Aren't we funny today...\r\n" {
		t.Errorf("self-target message = %q", self.MessageToCh)
	}

	ch.SetAffect(affMounted, true)
	mounted := DoSerpentKick(ch, mob, w)
	if mounted.MessageToCh != "Dismount first!\r\n" {
		t.Errorf("mounted message = %q", mounted.MessageToCh)
	}
}

func TestConfigureCreatedMobileMatchesC(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.ConfigureCreatedMobile(23)

	if got := mob.GetLevel(); got != 23 {
		t.Errorf("created level = %d, want 23", got)
	}
	if got := mob.GetExp(); got != 0 {
		t.Errorf("created exp = %d, want 0", got)
	}
	if got := mob.GetAC(); got != -130 {
		t.Errorf("created AC = %d, want -130", got)
	}
	if got := mob.GetHitroll(); got != 23 {
		t.Errorf("created hitroll = %d, want 23", got)
	}
	if got := mob.GetDamroll(); got != 16 {
		t.Errorf("created damroll = %d, want 16", got)
	}
	if got := mob.GetDamageRoll(); got != (combat.DiceRoll{Num: 15, Sides: 4}) {
		t.Errorf("created damage dice = %+v, want 15d4", got)
	}
	if got := mob.GetMaxHP(); got != 253 {
		t.Errorf("created max HP = %d, want 253", got)
	}
	if got := mob.GetHP(); got != 253 {
		t.Errorf("created current HP = %d, want 253", got)
	}
}
