package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSetRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("set")
	if !ok {
		t.Fatal("set is not registered")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("set gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosDead)
	}
}

func TestSetFieldTableMatchesCOrderAndGates(t *testing.T) {
	want := []struct {
		name  string
		level int
		pcnpc uint8
		typ   setFieldType
	}{
		{"brief", LVL_GOD, setPC, setBinary},
		{"invstart", LVL_GOD, setPC, setBinary},
		{"title", LVL_GOD, setPC, setMisc},
		{"nosummon", LVL_GRGOD, setPC, setBinary},
		{"maxhit", LVL_GRGOD, setPC | setNPC, setNumber},
		{"maxmana", LVL_GRGOD, setPC | setNPC, setNumber},
		{"maxmove", LVL_GRGOD, setPC | setNPC, setNumber},
		{"hit", LVL_GRGOD, setPC | setNPC, setNumber},
		{"mana", LVL_GRGOD, setPC | setNPC, setNumber},
		{"move", LVL_GRGOD, setPC | setNPC, setNumber},
		{"align", LVL_GOD, setPC | setNPC, setNumber},
		{"str", LVL_GRGOD, setPC | setNPC, setNumber},
		{"stradd", LVL_GRGOD, setPC | setNPC, setNumber},
		{"int", LVL_GRGOD, setPC | setNPC, setNumber},
		{"wis", LVL_GRGOD, setPC | setNPC, setNumber},
		{"dex", LVL_GRGOD, setPC | setNPC, setNumber},
		{"con", LVL_GRGOD, setPC | setNPC, setNumber},
		{"sex", LVL_GRGOD, setPC | setNPC, setMisc},
		{"ac", LVL_GRGOD, setPC | setNPC, setNumber},
		{"gold", LVL_GOD, setPC | setNPC, setNumber},
		{"bank", LVL_GOD, setPC, setNumber},
		{"exp", LVL_GRGOD, setPC | setNPC, setNumber},
		{"hitroll", LVL_GRGOD, setPC | setNPC, setNumber},
		{"damroll", LVL_GRGOD, setPC | setNPC, setNumber},
		{"invis", LVL_IMPL, setPC, setNumber},
		{"nohassle", LVL_GRGOD, setPC, setBinary},
		{"frozen", LVL_GRGOD, setPC, setBinary},
		{"practices", LVL_GRGOD, setPC, setNumber},
		{"lessons", LVL_GRGOD, setPC, setNumber},
		{"drunk", LVL_GRGOD, setPC | setNPC, setMisc},
		{"hunger", LVL_GRGOD, setPC | setNPC, setMisc},
		{"thirst", LVL_GRGOD, setPC | setNPC, setMisc},
		{"outlaw", LVL_GOD, setPC, setBinary},
		{"name", LVL_GRGOD, setPC, setMisc},
		{"level", LVL_GRGOD, setPC | setNPC, setNumber},
		{"room", LVL_IMPL, setPC | setNPC, setNumber},
		{"roomflag", LVL_GRGOD, setPC, setBinary},
		{"siteok", LVL_GRGOD, setPC, setBinary},
		{"deleted", LVL_GRGOD, setPC, setBinary},
		{"class", LVL_GRGOD, setPC | setNPC, setMisc},
		{"nowizlist", LVL_GOD, setPC, setBinary},
		{"quest", LVL_GOD, setPC, setBinary},
		{"loadroom", LVL_GRGOD, setPC, setMisc},
		{"color", LVL_GOD, setPC, setBinary},
		{"idnum", LVL_IMPL - 1, setPC, setNumber},
		{"passwd", LVL_IMPL - 1, setPC, setMisc},
		{"nodelete", LVL_GOD, setPC, setBinary},
		{"cha", LVL_GRGOD, setPC | setNPC, setNumber},
		{"olc", LVL_GOD + 1, setPC, setNumber},
		{"race", LVL_GOD, setPC, setMisc},
		{"kills", LVL_GRGOD, setPC | setNPC, setNumber},
		{"pks", LVL_GRGOD, setPC | setNPC, setNumber},
		{"deaths", LVL_GRGOD, setPC | setNPC, setNumber},
		{"home", LVL_GRGOD, setPC, setNumber},
		{"tattoo", LVL_GRGOD, setPC, setNumber},
		{"origcon", LVL_GRGOD, setPC, setNumber},
		{"chosen", LVL_GRGOD, setPC, setBinary},
		{"clan", LVL_GRGOD, setPC, setNumber},
		{"played", LVL_IMPL, setPC, setNumber},
	}
	if len(setFields) != len(want) {
		t.Fatalf("set field count = %d, want %d", len(setFields), len(want))
	}
	for i, got := range setFields {
		if got.name != want[i].name || got.level != want[i].level || got.pcnpc != want[i].pcnpc || got.typ != want[i].typ {
			t.Errorf("set field %d = %#v, want %#v", i, got, want[i])
		}
	}
}

func TestSetRawValueUsesCFieldRemainder(t *testing.T) {
	wiz, target := makeSetTestSession(t)
	if err := executeCommandRaw(wiz, "set", []string{"Hero", "title", "the", "Depth"}, true, "Hero title the  Depth  "); err != nil {
		t.Fatalf("executeCommandRaw(set): %v", err)
	}
	if got := readSessionText(t, wiz); got != "Hero's title is now: the  Depth  \r\n" {
		t.Fatalf("ack = %q, want C half_chop spacing", got)
	}
	if got := target.player.GetTitle(); got != "the  Depth  " {
		t.Fatalf("title = %q, want raw value", got)
	}
}

func TestSetCAtoiMatchesAtoiPrefixBehavior(t *testing.T) {
	for input, want := range map[string]int{"": 0, "-12tail": -12, "+7 rest": 7, "words": 0} {
		if got := cAtoi(input); got != want {
			t.Errorf("cAtoi(%q) = %d, want %d", input, got, want)
		}
	}
}
