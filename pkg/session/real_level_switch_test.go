package session

import "testing"

func TestPCSwitchSetsAndClearsSwitchedOriginalOnBody(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	body := h.player
	if original.GetLevel() < LVL_IMPL {
		original.SetLevel(LVL_IMPL)
	}

	// switchedPCFixture sets fields directly; exercise cmdSwitch and cmdReturn
	// on a fresh unswitched session to verify the live lifecycle hooks.
	_ = m.returnSwitch(s)
	if body.GetSwitchedOriginal() != nil {
		t.Fatalf("after returnSwitch, body.GetSwitchedOriginal() = %v, want nil", body.GetSwitchedOriginal())
	}

	if err := cmdSwitch(s, []string{body.GetName()}); err != nil {
		t.Fatalf("cmdSwitch failed: %v", err)
	}
	if !s.activePCSwitch() || body.GetSwitchedOriginal() != original {
		t.Fatalf("after cmdSwitch, switched=%v orig=%v, want %v", s.activePCSwitch(), body.GetSwitchedOriginal(), original)
	}

	if err := cmdReturn(s, nil); err != nil {
		t.Fatalf("cmdReturn failed: %v", err)
	}
	if body.GetSwitchedOriginal() != nil {
		t.Fatalf("after cmdReturn, body.GetSwitchedOriginal() = %v, want nil", body.GetSwitchedOriginal())
	}
}
