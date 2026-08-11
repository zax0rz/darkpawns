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
//
// noctell is excluded from the flag-toggling loop because C gates it on clan
// membership (act.other.c:1194) — a clanless fresh player gets "You aren't
// even in a clan!" and the flag never flips. It has its own test below.
//
// The wantFirst field pins the C-verbatim tog_messages[SCMD_*][TOG_ON] string
// for the toggles whose Go port was previously an invented re-skin — these
// assertions would have failed against the pre-fidelity-fix messages.
func TestDoGenTogRealCommandNames(t *testing.T) {
	cases := []struct {
		cmd       string
		flag      int
		wantFirst string // C tog_messages[subcmd][TOG_ON] when the flag flips ON
	}{
		{"nosummon", PrfSummonable, "You may now be summoned by other players.\r\n"},
		{"nohassle", PrfNohassle, "Nohassle enabled.\r\n"},
		{"brief", PrfBrief, "Brief mode on.\r\n"},
		{"compact", PrfCompact, "Compact mode on.\r\n"},
		{"notell", PrfNotell, "You are now deaf to tells.\r\n"},
		{"noauction", PrfNoAuctions, "You are now deaf to auctions.\r\n"},
		{"noshout", PrfDeaf, "You are now deaf to shouts.\r\n"},
		{"nogossip", PrfNoGossip, "You are now deaf to gossip.\r\n"},
		{"nograts", PrfNoGratz, "You are now deaf to the congratulation messages.\r\n"},
		{"quest", PrfQuest, "Okay, you are part of the Quest!\r\n"},
		{"norepeat", PrfNoRepeat, "You will no longer have your communication repeated.\r\n"},
		{"nonewbie", PrfNoNewbie, "Newbie channel off.\r\n"},
		{"nobroadcast", PrfNoBroad, "Broadcast channel is now off.\r\n"},
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
		msg := lastMsg()
		if strings.Contains(msg, "Unknown toggle") {
			t.Errorf("%s: got %q, want a real toggle message", c.cmd, msg)
		}
		if c.wantFirst != "" && msg != c.wantFirst {
			t.Errorf("%s first-toggle message:\n got %q\nwant %q (C act.other.c tog_messages)", c.cmd, msg, c.wantFirst)
		}

		w.doGenTog(ch, nil, c.cmd, "")
		if ch.GetFlags()&(1<<c.flag) != 0 {
			t.Errorf("%s: flag should be cleared after second toggle", c.cmd)
		}
	}
}

// TestDoGenTogNoCTellClanGate verifies the SCMD_NOCTELL clan gate
// (act.other.c:1194): a clanless player cannot toggle clan tells — C prints
// "You aren't even in a clan!" and never touches PRF_NOCTELL. The Go port
// formerly invented a "You are now deaf to clan tells." message and flipped
// the flag unconditionally.
func TestDoGenTogNoCTellClanGate(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	if ch.ClanID != 0 {
		t.Fatalf("test player should be clanless, got ClanID=%d", ch.ClanID)
	}

	w.doGenTog(ch, nil, "noctell", "")
	if ch.GetFlags()&(1<<PrfNoCTell) != 0 {
		t.Errorf("clanless noctell must not set PRF_NOCTELL")
	}
	if msg := lastMsg(); msg != "You aren't even in a clan!\r\n" {
		t.Errorf("noctell clan-gate message:\n got %q\nwant %q", msg, "You aren't even in a clan!\r\n")
	}
}

// TestDoGenTogImmortalGated verifies nohassle/nowiz/roomflags/holylight stay
// behind the registry's LVL_IMMORT gate — doGenTog's own internal nowiz
// check (the only one of the four enforced inside the function itself,
// matching src/act.other.c:1200) still rejects a mortal who somehow reaches
// it with C's verbatim "Huh?!?" (the dispatcher-level gate prints the same).
func TestDoGenTogImmortalGated(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	ch.SetLevel(1)

	w.doGenTog(ch, nil, "nowiz", "")
	if ch.GetFlags()&(1<<PrfNowiz) != 0 {
		t.Errorf("mortal should not be able to set nowiz")
	}
	if msg := lastMsg(); msg != "Huh?!?\r\n" {
		t.Errorf("nowiz-blocked message:\n got %q\nwant %q", msg, "Huh?!?\r\n")
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
