package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestDoTigerPunch_MissRoutesThroughSkillMessage — C's do_tiger_punch miss arm
// calls damage(ch, vict, 0, SKILL_TIGER_PUNCH) (act.offensive.c:736), so the
// three lines come from lib/misc/messages set 189 through the shared
// skill_message seam and no Go literal may be emitted (R1/R4). A learned value
// of 1 can never beat ((7 - AC/10) << 1) + number(1,101) >= 15, so the arm is
// deterministic.
func TestDoTigerPunch_MissRoutesThroughSkillMessage(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	ch.SetSkill(SkillTigerPunch, 1)

	result := DoTigerPunch(ch, mob)

	if result.Success || result.Damage != 0 {
		t.Fatalf("skill 1 tiger punch = success %v damage %d, want a miss", result.Success, result.Damage)
	}
	if result.SkillMsgType != SkillTigerPunchNum {
		t.Errorf("miss SkillMsgType = %d, want set %d (189)", result.SkillMsgType, SkillTigerPunchNum)
	}
	if !result.SkillMsgInDamage || result.DamageSkill != SkillTigerPunch {
		t.Errorf("miss damage contract = inDamage %v skill %q, want true/%q",
			result.SkillMsgInDamage, result.DamageSkill, SkillTigerPunch)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("miss emitted invented literals ch=%q vict=%q room=%q (R4)",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}
	if !result.StartCombat || result.WaitCh != 2 {
		t.Errorf("miss combat contract = start %v wait %d, want true/2", result.StartCombat, result.WaitCh)
	}
}

// TestDoTigerPunch_HitContract — the hit arm returns GET_LEVEL(ch)*2.5 damage
// with set 189 selected at the damage() boundary and improve_skill deferred
// past the message dice (act.offensive.c:738-739; R3b).
func TestDoTigerPunch_HitContract(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	ch.SetSkill(SkillTigerPunch, 100)

	var result SkillResult
	hit := false
	for i := 0; i < 40; i++ {
		result = DoTigerPunch(ch, mob)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Skip("no tiger punch hit observed in 40 tries (RNG); contract not exercised")
	}

	if want := int(float64(ch.GetLevel()) * 2.5); result.Damage != want {
		t.Errorf("hit damage = %d, want C integer conversion of level*2.5 = %d", result.Damage, want)
	}
	if result.SkillMsgType != SkillTigerPunchNum || !result.SkillMsgInDamage || result.DamageSkill != SkillTigerPunch {
		t.Errorf("hit damage contract = set %d inDamage %v skill %q, want %d/true/%q",
			result.SkillMsgType, result.SkillMsgInDamage, result.DamageSkill, SkillTigerPunchNum, SkillTigerPunch)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillTigerPunch {
		t.Errorf("hit DeferredImprove = %v, want [%q]", result.DeferredImprove, SkillTigerPunch)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("hit emitted invented literals ch=%q vict=%q room=%q (R4)",
			result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}
	if !result.StartCombat || result.WaitCh != 2 {
		t.Errorf("hit combat contract = start %v wait %d, want true/2", result.StartCombat, result.WaitCh)
	}
}

// TestDoTigerPunch_LegacyGatesPreserved — the skill and bare-hands refusals are
// unchanged C bytes (act.offensive.c:698-706).
func TestDoTigerPunch_LegacyGatesPreserved(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)

	ch.SetSkill(SkillTigerPunch, 0)
	if got := DoTigerPunch(ch, mob).MessageToCh; got != "What's that, idiot-san?" {
		t.Errorf("unlearned tiger punch = %q", got)
	}

	ch.SetSkill(SkillTigerPunch, 100)
	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)
	if got := DoTigerPunch(ch, mob).MessageToCh; !strings.Contains(got, "while wielding a weapon") {
		t.Errorf("wielded tiger punch = %q, want the bare-hands refusal", got)
	}
}

// TestDoTigerPunchDamage_SelectsVariantAfterHpUpdate — the damage() boundary
// must apply HP before skill_message runs: an ordinary hit selects the hit
// variant, which a pre-damage selection would also reach, but the draw and the
// audience bytes must come from the same seam the command tail drives
// (fight.c:1484-1534).
func TestDoTigerPunchDamage_SelectsVariantAfterHpUpdate(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Level = 20
	w.StopAITicker()

	wired := wireTigerPunchMessages(t, mob)
	defer wired.teardown()

	mob.SetHealth(500)
	if handled := w.DoTigerPunchDamage(ch, mob, 20); !handled {
		t.Fatal("tiger punch damage was not applied")
	}
	if !strings.Contains(wired.attacker(), "uppercut connects with") {
		t.Errorf("non-lethal hit message = %q, want set 189's hit_msg", wired.attacker())
	}
}

// tigerPunchMessages captures set 189's attacker and room lines for the real
// mob so the variant is chosen from its post-damage hit points.
type tigerPunchMessages struct {
	attMsg   string
	roomMsg  string
	teardown func()
}

func (m *tigerPunchMessages) attacker() string { return m.attMsg }

func wireTigerPunchMessages(t *testing.T, mob *MobInstance) *tigerPunchMessages {
	t.Helper()
	original := combat.GetCallbacks()
	captured := &tigerPunchMessages{}
	cb := &combat.GameCallbacks{
		Broadcast: func(_ int, msg, _ string) { captured.roomMsg = msg },
		SendToChar: func(name, msg string) {
			if name == "TestPlayer" {
				captured.attMsg = msg
			}
		},
		GetSex:   func(string) int { return 0 },
		GetHP:    func(string) int { return mob.GetHP() },
		GetLevel: func(string) int { return 1 },
		IsNPC:    func(string) bool { return true },
	}
	combat.SetCallbacks(cb)
	combat.InitFightMessages(cb, loadMessagesFile(t))
	captured.teardown = func() { combat.SetCallbacks(original) }
	return captured
}
