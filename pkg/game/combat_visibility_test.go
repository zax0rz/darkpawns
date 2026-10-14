package game

import "testing"

func TestCombatCanSeeMatchesDamageVisibilityGates(t *testing.T) {
	w, observer := newCombatTestWorld(t)
	target := NewPlayer(2, "InvisibleTarget", 1001)
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}
	guard, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob guard: %v", err)
	}
	if !w.SetRoomFlagBit(1001, 0) {
		t.Fatal("SetRoomFlagBit failed")
	}

	canSee := w.WireCombatCallbacks().CanSee
	if canSee(guard, target) {
		t.Fatal("mobile saw target in a dark room without infravision")
	}
	guard.SetAffected(affInfravision)
	target.SetAffect(affInvisible, true)
	if canSee(guard, target) {
		t.Fatal("mobile saw invisible target without detect-invisible")
	}
	guard.SetAffected(affDetectInvisible)
	if !canSee(guard, target) {
		t.Fatal("mobile with infravision and detect-invisible could not see target")
	}
	target.SetAffect(affHide, true)
	if !canSee(guard, target) {
		t.Fatal("AFF_HIDE incorrectly blocked CAN_SEE")
	}
	guard.SetAffected(affBlind)
	if canSee(guard, target) {
		t.Fatal("blind mobile saw target")
	}

	observer.SetLevel(10)
	observer.SetAffect(affBlind, true)
	observer.SetHolyLight(true)
	if !canSee(observer, target) {
		t.Fatal("HOLYLIGHT did not bypass blindness, darkness, and invisibility")
	}
	target.SetInvisLevel(31)
	if canSee(observer, target) {
		t.Fatal("HOLYLIGHT bypassed the earlier GET_REAL_LEVEL gate")
	}
	if !canSee(observer, observer) {
		t.Fatal("CAN_SEE self case was not visible")
	}
}
