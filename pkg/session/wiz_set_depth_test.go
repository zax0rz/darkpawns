package session

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// These six rows are the direct C assignments in act.wizard.c:2724-2726,
// 2913-2921, and 2942-2946. cBit and goBit are independent expected values
// pinned from the C and Go flag namespaces. PRF C bits intentionally differ
// from the combined Go Flags storage positions; productionBit must agree with
// the independent Go position rather than defining the expected mask.
func TestCmdSetDirectBinaryFieldsMatchCBits(t *testing.T) {
	cases := []struct {
		field         string
		ackName       string
		cBit          int
		goBit         int
		productionBit int
		playerFlags   bool
	}{
		{field: "invstart", ackName: "Invstart", cBit: 14, goBit: 14, productionBit: game.PlrInvstart, playerFlags: true},
		{field: "roomflag", ackName: "Roomflag", cBit: 21, goBit: 39, productionBit: game.PrfRoomFlags},
		{field: "siteok", ackName: "Siteok", cBit: 7, goBit: 7, productionBit: game.PlrSiteok, playerFlags: true},
		{field: "deleted", ackName: "Deleted", cBit: 10, goBit: 10, productionBit: game.PlrDeleted, playerFlags: true},
		{field: "nowizlist", ackName: "Nowizlist", cBit: 12, goBit: 12, productionBit: game.PlrNowizlist, playerFlags: true},
		{field: "quest", ackName: "Quest", cBit: 9, goBit: 49, productionBit: game.PrfQuest},
	}

	const unrelatedPLR = game.PlrChosen
	const unrelatedPRF = game.PrfColor1
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			wiz, target := makeSetTestSession(t)
			target.player.SetPlrFlag(unrelatedPLR, true)
			target.player.SetPLRFlag(unrelatedPLR)
			target.player.SetPlrFlag(unrelatedPRF, true)

			initialFlags := target.player.GetFlags()
			initialPlayerFlags := target.player.PlayerFlags
			if tc.productionBit != tc.goBit {
				t.Fatalf("Go %s production bit = %d, want independently pinned Go bit %d (C bit %d)", tc.field, tc.productionBit, tc.goBit, tc.cBit)
			}
			bit := uint64(1) << uint(tc.goBit)
			if initialFlags&bit != 0 {
				t.Fatalf("C target bit %d unexpectedly set before command", tc.cBit)
			}

			for _, state := range []struct {
				word    string
				enabled bool
			}{{word: "on", enabled: true}, {word: "off", enabled: false}} {
				beforeFlags := target.player.GetFlags()
				beforePlayerFlags := target.player.PlayerFlags
				if err := ExecuteCommand(wiz, "set", []string{"Hero", tc.field, state.word}); err != nil {
					t.Fatalf("ExecuteCommand(set %s %s): %v", tc.field, state.word, err)
				}

				wantFlags := beforeFlags &^ bit
				if state.enabled {
					wantFlags = beforeFlags | bit
				}
				if got := target.player.GetFlags(); got != wantFlags {
					t.Fatalf("%s %s flags = %#x, want exactly %#x (C bit %d)", tc.field, state.word, got, wantFlags, tc.cBit)
				}

				wantPlayerFlags := beforePlayerFlags
				if tc.playerFlags {
					wantPlayerFlags &^= bit
					if state.enabled {
						wantPlayerFlags |= bit
					}
				}
				if got := target.player.PlayerFlags; got != wantPlayerFlags {
					t.Fatalf("%s %s typed PLR flags = %#x, want %#x", tc.field, state.word, got, wantPlayerFlags)
				}

				wantAck := fmt.Sprintf("%s %s for Hero.\r\n", tc.ackName, map[bool]string{true: "ON", false: "OFF"}[state.enabled])
				if got := readSessionText(t, wiz); got != wantAck {
					t.Fatalf("%s %s ack = %q, want %q", tc.field, state.word, got, wantAck)
				}
			}

			if target.player.GetFlags() != initialFlags || target.player.PlayerFlags != initialPlayerFlags {
				t.Fatalf("%s on/off round trip changed unrelated state: flags %#x/%#x, typed PLR %#x/%#x", tc.field, target.player.GetFlags(), initialFlags, target.player.PlayerFlags, initialPlayerFlags)
			}
		})
	}
}

// C's nohassle field is legal for a LVL_GRGOD actor only when the target is
// that actor; another target reaches the explicit LVL_IMPL check at
// act.wizard.c:2851-2856. The level-38 fixture proves this branch after the
// generic set and field-level checks have already passed.
func TestCmdSetNohassleAuthorityAndSelfTarget(t *testing.T) {
	t.Run("self succeeds at field level", func(t *testing.T) {
		wiz, _ := makeSetTestSession(t)
		const cBit = 8   // src/structs.h:283, PRF_NOHASSLE
		const goBit = 28 // pkg/game/other_helpers.go:30, combined Flags storage
		if game.PrfNohassle != goBit {
			t.Fatalf("Go nohassle bit = %d, want independently pinned Go bit %d (C bit %d)", game.PrfNohassle, goBit, cBit)
		}
		const bit = uint64(1) << goBit
		for _, state := range []struct {
			word    string
			enabled bool
		}{{word: "on", enabled: true}, {word: "off", enabled: false}} {
			beforeFlags := wiz.player.GetFlags()
			if err := ExecuteCommand(wiz, "set", []string{"God", "nohassle", state.word}); err != nil {
				t.Fatalf("ExecuteCommand(set nohassle %s): %v", state.word, err)
			}
			wantFlags := beforeFlags &^ bit
			if state.enabled {
				wantFlags = beforeFlags | bit
			}
			if got := wiz.player.GetFlags(); got != wantFlags {
				t.Fatalf("self nohassle %s flags = %#x, want exactly %#x (C bit %d)", state.word, got, wantFlags, cBit)
			}
			wantAck := fmt.Sprintf("Nohassle %s for God.\r\n", map[bool]string{true: "ON", false: "OFF"}[state.enabled])
			if got := readSessionText(t, wiz); got != wantAck {
				t.Fatalf("self nohassle %s ack = %q, want %q", state.word, got, wantAck)
			}
		}
	})

	t.Run("other target rejected below implementor", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		before := target.player.GetFlags()
		for _, word := range []string{"on", "off"} {
			if err := ExecuteCommand(wiz, "set", []string{"Hero", "nohassle", word}); err != nil {
				t.Fatalf("ExecuteCommand(set Hero nohassle %s): %v", word, err)
			}
			if got := readSessionText(t, wiz); got != "You aren't godly enough for that!\r\n" {
				t.Fatalf("other-target nohassle %s ack = %q", word, got)
			}
			if got := target.player.GetFlags(); got != before {
				t.Fatalf("other-target nohassle %s changed target flags to %#x from %#x", word, got, before)
			}
		}
	})
}

// C's frozen branch has no extra authority check after the LVL_FREEZE field
// gate, but refuses pointer-identical self targets at act.wizard.c:2857-2863.
func TestCmdSetFrozenAuthorityAndSelfTarget(t *testing.T) {
	t.Run("other target succeeds at field level", func(t *testing.T) {
		wiz, target := makeSetTestSession(t)
		const cBit = 2  // src/structs.h:223, PLR_FROZEN
		const goBit = 2 // pkg/game/player_flags.go:8, combined Flags storage
		if game.PlrFrozen != goBit {
			t.Fatalf("Go frozen bit = %d, want independently pinned Go bit %d (C bit %d)", game.PlrFrozen, goBit, cBit)
		}
		const bit = uint64(1) << goBit
		for _, state := range []struct {
			word    string
			enabled bool
		}{{word: "on", enabled: true}, {word: "off", enabled: false}} {
			beforeFlags := target.player.GetFlags()
			if err := ExecuteCommand(wiz, "set", []string{"Hero", "frozen", state.word}); err != nil {
				t.Fatalf("ExecuteCommand(set frozen %s): %v", state.word, err)
			}
			wantFlags := beforeFlags &^ bit
			if state.enabled {
				wantFlags = beforeFlags | bit
			}
			if got := target.player.GetFlags(); got != wantFlags {
				t.Fatalf("other-target frozen %s flags = %#x, want exactly %#x (C bit %d)", state.word, got, wantFlags, cBit)
			}
			wantAck := fmt.Sprintf("Frozen %s for Hero.\r\n", map[bool]string{true: "ON", false: "OFF"}[state.enabled])
			if got := readSessionText(t, wiz); got != wantAck {
				t.Fatalf("other-target frozen %s ack = %q, want %q", state.word, got, wantAck)
			}
		}
	})

	t.Run("self refused after field checks", func(t *testing.T) {
		wiz, _ := makeSetTestSession(t)
		const cBit = 2  // src/structs.h:223, PLR_FROZEN
		const goBit = 2 // pkg/game/player_flags.go:8, combined Flags storage
		if game.PlrFrozen != goBit {
			t.Fatalf("Go frozen bit = %d, want independently pinned Go bit %d (C bit %d)", game.PlrFrozen, goBit, cBit)
		}
		const bit = uint64(1) << goBit
		before := wiz.player.GetFlags()
		for _, word := range []string{"on", "off"} {
			if err := ExecuteCommand(wiz, "set", []string{"God", "frozen", word}); err != nil {
				t.Fatalf("ExecuteCommand(set God frozen %s): %v", word, err)
			}
			if got := readSessionText(t, wiz); got != "Better not -- could be a long winter!\r\n" {
				t.Fatalf("self frozen %s ack = %q", word, got)
			}
			if got := wiz.player.GetFlags(); got != before || got&bit != 0 {
				t.Fatalf("self frozen %s changed flags to %#x from %#x", word, got, before)
			}
		}
	})
}
