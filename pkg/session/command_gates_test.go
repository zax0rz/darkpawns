package session

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCommandGateGoldenCoversCGoRegistrationsAndSocials(t *testing.T) {
	const wantCRows = 508
	cRows := 0
	var cNames []string
	for name, gate := range commandGates {
		if gate.Source == "" {
			t.Errorf("gate %q has no source or Go-only rationale", name)
		}
		if strings.HasPrefix(gate.Source, "C interpreter.c:") {
			cRows++
			cNames = append(cNames, name)
		}
	}
	if cRows != wantCRows {
		t.Errorf("C-derived gate rows = %d, want %d; regenerate command_gates.tsv", cRows, wantCRows)
	}
	sort.Strings(cNames)
	hash := sha256.New()
	for _, name := range cNames {
		gate := commandGates[name]
		_, _ = fmt.Fprintf(hash, "%s\t%d\t%d\n", name, gate.MinLevel, gate.MinPosition)
	}
	const wantCGateHash = "9042b9308c018738168dea43005197eb6924cc3470bc3d6714684d8e07f11041"
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != wantCGateHash {
		t.Errorf("C command gate hash = %s, want %s; regenerate from the reviewed oracle", got, wantCGateHash)
	}

	for _, entry := range cmdRegistry.GetAll() {
		if strings.HasPrefix(entry.Name, "dp954test") || strings.HasPrefix(entry.Name, "gatecascadetest") ||
			strings.HasPrefix(entry.Name, "gatefrozentest") || strings.HasPrefix(entry.Name, "gateswitchedtest") {
			continue
		}
		assertEntryGateMatchesGolden(t, entry.Name, entry.MinLevel, entry.MinPosition)
		for _, alias := range entry.Aliases {
			assertEntryGateMatchesGolden(t, alias, entry.MinLevel, entry.MinPosition)
		}
	}
	for name := range game.Socials {
		if _, ok := commandGates[name]; !ok {
			t.Errorf("social %q has no authoritative gate", name)
		}
	}
}

func assertEntryGateMatchesGolden(t *testing.T, name string, level, position int) {
	t.Helper()
	gate, ok := commandGates[name]
	if !ok {
		t.Errorf("registered command word %q is absent from command_gates.tsv", name)
		return
	}
	if level != gate.MinLevel || position != gate.MinPosition {
		t.Errorf("%q gate = (%d,%d), want (%d,%d) from %s",
			name, level, position, gate.MinLevel, gate.MinPosition, gate.Source)
	}
}

func TestCommandLevelConstantsMatchC(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{name: "LVL_IMMORT", got: LVL_IMMORT, want: 31},
		{name: "LVL_GOD", got: LVL_GOD, want: 34},
		{name: "LVL_GRGOD", got: LVL_GRGOD, want: 38},
		{name: "LVL_IMPL", got: LVL_IMPL, want: 40},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want C value %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestPositionFailMessageMatchesC(t *testing.T) {
	tests := []struct {
		position int
		want     string
	}{
		{combat.PosDead, "Lie still; you are DEAD!!! :-(\r\n"},
		{combat.PosMortally, "You are in a pretty bad shape, unable to do anything!\r\n"},
		{combat.PosIncap, "You are in a pretty bad shape, unable to do anything!\r\n"},
		{combat.PosStunned, "All you can do right now is think about the stars!\r\n"},
		{combat.PosSleeping, "In your dreams, or what?\r\n"},
		{combat.PosResting, "Nah... You feel too relaxed to do that..\r\n"},
		{combat.PosSitting, "Maybe you should get on your feet first?\r\n"},
		{combat.PosFighting, "No way!  You're fighting for your life!\r\n"},
	}
	for _, tt := range tests {
		if got := positionFailMessage(tt.position); got != tt.want {
			t.Errorf("positionFailMessage(%d) = %q, want %q", tt.position, got, tt.want)
		}
	}
}

func TestCommandGateCascadeHidesLevelBeforePositionAndFrozen(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "mortal", 1, 1001)
	s.player.SetPosition(combat.PosSleeping)
	s.player.Flags |= 1 << uint(game.PlrFrozen)

	const name = "gatecascadetest"
	cmdRegistry.Register(name, wrapArgs(func(*Session, []string) error {
		t.Fatal("over-level handler ran")
		return nil
	}), "gate cascade test", LVL_IMMORT, combat.PosStanding)

	if err := ExecuteCommand(s, name, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Huh?!?\r\n" {
		t.Errorf("cascade reply = %q, want hidden-command reply", got)
	}
}

func TestCommandGateCascadeFrozenAndSwitchedBeforePosition(t *testing.T) {
	t.Run("frozen", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "frozen", 1, 1001)
		s.player.SetPosition(combat.PosSleeping)
		s.player.Flags |= 1 << uint(game.PlrFrozen)

		const name = "gatefrozentest"
		cmdRegistry.Register(name, wrapArgs(func(*Session, []string) error {
			t.Fatal("frozen handler ran")
			return nil
		}), "frozen cascade test", 0, combat.PosStanding)
		if err := ExecuteCommand(s, name, nil); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, s); got != "You try, but the mind-numbing cold prevents you...\r\n" {
			t.Errorf("frozen reply = %q", got)
		}
	})

	t.Run("switched mob", func(t *testing.T) {
		m := makeTestManagerWithMobs(t)
		s := makeCommandTestSession(t, m, "wizard", LVL_IMMORT, 1001)
		s.player.SetPosition(combat.PosSleeping)
		s.isSwitched = true
		s.switchedOriginalLevel = LVL_IMMORT
		s.switchedMob = registerMob(t, m, 2001, 1001)

		const name = "gateswitchedtest"
		cmdRegistry.Register(name, wrapArgs(func(*Session, []string) error {
			t.Fatal("switched handler ran")
			return nil
		}), "switched cascade test", LVL_IMMORT, combat.PosStanding)
		if err := ExecuteCommand(s, name, nil); err != nil {
			t.Fatal(err)
		}
		if got := readMsgText(t, s); got != "You can't use immortal commands while switched.\r\n" {
			t.Errorf("switched reply = %q", got)
		}
	})
}
