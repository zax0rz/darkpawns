package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func shadowOutput(output map[string]*strings.Builder, name string) string {
	if message := output[name]; message != nil {
		return message.String()
	}
	return ""
}

func TestDoFollowShadowSuccessAppliesQuietAffect(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	actor.SetLevel(12)
	actor.SetSkill(SkillShadow, 100)
	leader := addMovementPlayer(t, w, "Shadowleader", 1001)
	observer := addMovementPlayer(t, w, "Shadowobserver", 1001)
	output := captureMovementOutput(w)

	dprng.ResetStream(1)
	w.DoFollow(actor, leader.Name, true)

	if got := shadowOutput(output, actor.Name); got != "You now follow Shadowleader.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	if got := shadowOutput(output, leader.Name); got != "" {
		t.Fatalf("leader received quiet-shadow output %q", got)
	}
	if got := shadowOutput(output, observer.Name); got != "" {
		t.Fatalf("observer received quiet-shadow output %q", got)
	}
	if got := actor.GetFollowing(); got != leader.Name {
		t.Fatalf("following = %q, want %q", got, leader.Name)
	}
	if !actor.IsAffected(affDodge) {
		t.Fatal("successful shadow did not set AFF_DODGE")
	}
	if len(actor.ActiveAffects) != 1 {
		t.Fatalf("active affects = %d, want one shadow affect", len(actor.ActiveAffects))
	}
	affect := actor.ActiveAffects[0]
	if affect.SpellID != skillNumShadow || affect.Duration != 12 ||
		affect.Location != engine.ApplyNone || affect.Magnitude != 0 ||
		affect.Flags != engine.AFFDodge || affect.Source != SkillShadow {
		t.Fatalf("shadow affect = %+v", affect)
	}
}

func TestDoFollowShadowFailureUsesOrdinaryFollowAudience(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	actor.SetLevel(1)
	actor.SetSkill(SkillShadow, 0)
	leader := addMovementPlayer(t, w, "Shadowfailleader", 1001)
	observer := addMovementPlayer(t, w, "Shadowfailobserver", 1001)
	output := captureMovementOutput(w)

	dprng.ResetStream(1)
	w.DoFollow(actor, leader.Name, true)

	if got := shadowOutput(output, actor.Name); got != "You now follow Shadowfailleader.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	if got := shadowOutput(output, leader.Name); got != "TestPlayer starts following you.\r\n" {
		t.Fatalf("leader output = %q", got)
	}
	if got := shadowOutput(output, observer.Name); got != "TestPlayer starts to follow Shadowfailleader.\r\n" {
		t.Fatalf("observer output = %q", got)
	}
	if actor.IsAffected(affDodge) || len(actor.ActiveAffects) != 0 {
		t.Fatalf("failed shadow left affect state: bit=%t affects=%d", actor.IsAffected(affDodge), len(actor.ActiveAffects))
	}
}

func TestDoFollowShadowConsumesSkillRollDraw(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	actor.SetLevel(1)
	actor.SetSkill(SkillShadow, 0)
	leader := addMovementPlayer(t, w, "Shadowdrawleader", 1001)

	const seed = 1
	dprng.ResetStream(seed)
	w.DoFollow(actor, leader.Name, true)
	gotNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	dprng.Number(0, 101)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("next RNG draw = %d, want %d after one shadow skill roll", gotNext, wantNext)
	}
}

func TestStopFollowerShadowClearsAffectAndSuppressesAudience(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	actor.SetLevel(12)
	actor.SetSkill(SkillShadow, 100)
	leader := addMovementPlayer(t, w, "Shadowleader", 1001)
	observer := addMovementPlayer(t, w, "Shadowobserver", 1001)

	dprng.ResetStream(1)
	w.DoFollow(actor, leader.Name, true)
	output := captureMovementOutput(w)
	w.DoFollow(actor, "self", true)

	if got := shadowOutput(output, actor.Name); got != "You stop shadowing Shadowleader.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	if got := shadowOutput(output, leader.Name); got != "" {
		t.Fatalf("leader received stop-shadow output %q", got)
	}
	if got := shadowOutput(output, observer.Name); got != "" {
		t.Fatalf("observer received stop-shadow output %q", got)
	}
	if actor.GetFollowing() != "" {
		t.Fatalf("following = %q, want empty", actor.GetFollowing())
	}
	if actor.IsAffected(affDodge) || len(actor.ActiveAffects) != 0 {
		t.Fatalf("stop-shadow left affect state: bit=%t affects=%d", actor.IsAffected(affDodge), len(actor.ActiveAffects))
	}
}

func TestDoFollowShadowSwitchReplacesPriorAffect(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	actor.SetLevel(12)
	actor.SetSkill(SkillShadow, 100)
	first := addMovementPlayer(t, w, "Shadowfirst", 1001)
	second := addMovementPlayer(t, w, "Shadowsecond", 1001)

	dprng.ResetStream(1)
	w.DoFollow(actor, first.Name, true)
	dprng.ResetStream(1)
	w.DoFollow(actor, second.Name, true)

	if got := actor.GetFollowing(); got != second.Name {
		t.Fatalf("following = %q, want %q", got, second.Name)
	}
	if len(actor.ActiveAffects) != 1 || !actor.IsAffected(affDodge) {
		t.Fatalf("re-shadow state = affects %d, dodge %t", len(actor.ActiveAffects), actor.IsAffected(affDodge))
	}
	if actor.ActiveAffects[0].SpellID != skillNumShadow || actor.ActiveAffects[0].Duration != 12 {
		t.Fatalf("replacement shadow affect = %+v", actor.ActiveAffects[0])
	}
}
