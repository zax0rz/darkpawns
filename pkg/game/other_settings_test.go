package game

import (
	"strings"
	"testing"
)

// TestDoGenTogRealCommandNames verifies doGenTog dispatches by the literal
// command name a player types (src/interpreter.c registers each toggle as
// its own top-level command), including the four cases where that name
// differs from the historical internal map key: "nosummon" (not "summon"),
// "noshout" (not "deaf"), "nograts" (not "nogratz"), "nobroadcast" (not
// "nobroad"). Before the Bucket E fix, doGenTog's cmdMap was keyed by the
// internal names directly, so these real command names always fell through
// to "Unknown toggle."
func TestDoGenTogRealCommandNames(t *testing.T) {
	cases := []struct {
		cmd  string
		flag int
	}{
		{"nosummon", PrfSummonable},
		{"nohassle", PrfNohassle},
		{"brief", PrfBrief},
		{"compact", PrfCompact},
		{"notell", PrfNotell},
		{"noauction", PrfNoAuctions},
		{"noshout", PrfDeaf},
		{"nogossip", PrfNoGossip},
		{"nograts", PrfNoGratz},
		{"quest", PrfQuest},
		{"norepeat", PrfNoRepeat},
		{"nonewbie", PrfNoNewbie},
		{"noctell", PrfNoCTell},
		{"nobroadcast", PrfNoBroad},
	}

	for _, c := range cases {
		w, ch, lastMsg := newDonateTestWorld(t)
		if ch.GetFlags()&(1<<c.flag) != 0 {
			t.Fatalf("%s: flag should start unset", c.cmd)
		}

		w.doGenTog(ch, nil, c.cmd, "")
		if ch.GetFlags()&(1<<c.flag) == 0 {
			t.Errorf("%s: flag should be set after first toggle", c.cmd)
		}
		if msg := lastMsg(); strings.Contains(msg, "Unknown toggle") {
			t.Errorf("%s: got %q, want a real toggle message", c.cmd, msg)
		}

		w.doGenTog(ch, nil, c.cmd, "")
		if ch.GetFlags()&(1<<c.flag) != 0 {
			t.Errorf("%s: flag should be cleared after second toggle", c.cmd)
		}
	}
}

// TestDoGenTogImmortalGated verifies nohassle/nowiz/roomflags/holylight stay
// behind the registry's LVL_IMMORT gate — doGenTog's own internal nowiz
// check (the only one of the four enforced inside the function itself,
// matching src/act.other.c) still rejects a mortal who somehow reaches it.
func TestDoGenTogImmortalGated(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	ch.SetLevel(1)

	w.doGenTog(ch, nil, "nowiz", "")
	if ch.GetFlags()&(1<<PrfNowiz) != 0 {
		t.Errorf("mortal should not be able to set nowiz")
	}
	if msg := lastMsg(); !strings.Contains(msg, "not holy enough") {
		t.Errorf("nowiz-blocked message: got %q", msg)
	}
}

func TestObservationPreferencesReadPersistedPRFBits(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)

	w.doGenTog(ch, nil, "roomflags", "")
	if !ch.GetRoomFlags() {
		t.Fatal("roomflags getter did not observe PRF_ROOMFLAGS toggle")
	}
	w.doGenTog(ch, nil, "roomflags", "")
	if ch.GetRoomFlags() {
		t.Fatal("roomflags getter remained set after PRF_ROOMFLAGS cleared")
	}

	w.doGenTog(ch, nil, "holylight", "")
	if !ch.GetHolyLight() {
		t.Fatal("holylight getter did not observe PRF_HOLYLIGHT toggle")
	}
	w.doGenTog(ch, nil, "holylight", "")
	if ch.GetHolyLight() {
		t.Fatal("holylight getter remained set after PRF_HOLYLIGHT cleared")
	}
}

// TestDoGenTogUnknownCommand verifies a command name with no mapping (e.g.
// the old, incorrect "autocxits"/"npcident" stub entries that were removed)
// falls through cleanly instead of touching player state.
func TestDoGenTogUnknownCommand(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	before := ch.GetFlags()

	w.doGenTog(ch, nil, "autocxits", "")
	if msg := lastMsg(); !strings.Contains(msg, "Unknown toggle") {
		t.Errorf("unmapped command message: got %q", msg)
	}
	if ch.GetFlags() != before {
		t.Errorf("unmapped command should not change player flags")
	}
}
