package session

import (
	"sort"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSocialsRegistrationUsesCEntryGateAndCList(t *testing.T) {
	entry, ok := commandGates["socials"]
	if !ok {
		t.Fatal("socials command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("socials gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}

	if len(cSocialCommandOrder) != 184 {
		t.Fatalf("C social command rows = %d, want 184 (183 do_action rows plus insult)", len(cSocialCommandOrder))
	}
	names := cSocialsForLevel(1)
	if len(names) != 183 {
		t.Fatalf("level-one C social listing has %d rows, want 183", len(names))
	}
	if names[0] != "accuse" || names[len(names)-1] != "yuball" {
		t.Fatalf("level-one C social listing bounds = %q/%q, want accuse/yuball", names[0], names[len(names)-1])
	}
	if containsString(names, "snowball") {
		t.Error("level-one C social listing includes LVL_IMMORT snowball")
	}
	if !containsString(names, "insult") {
		t.Error("level-one C social listing omits C's explicit insult social")
	}

	immortalNames := cSocialsForLevel(game.LVL_IMMORT)
	if len(immortalNames) != 183 {
		t.Fatalf("immortal C social listing has %d rows, want 183 (wizhelp=false excludes snowball)", len(immortalNames))
	}
	if containsString(immortalNames, "snowball") {
		t.Error("immortal C social listing includes the LVL_IMMORT row while wizhelp is false")
	}

	if got := cSocialsForLevel(game.LVL_IMMORT - 1); len(got) != 183 {
		t.Fatalf("pre-immortal C social listing has %d rows, want 183", len(got))
	}
	for _, level := range []int{game.LVL_IMMORT, game.LVL_IMPL} {
		for _, entry := range cSocialCommandOrder {
			if entry.minLevel >= game.LVL_IMMORT && containsString(cSocialsForLevel(level), entry.name) {
				t.Errorf("level %d listing includes immortal-only social %q", level, entry.name)
			}
		}
	}

	ordered := append([]string(nil), names...)
	sort.Strings(ordered)
	for i := range names {
		if names[i] != ordered[i] {
			t.Fatalf("C social listing is not lexical at index %d: got %q, want %q", i, names[i], ordered[i])
		}
	}
	for _, extra := range []string{"hiss", "kneel", "mutter", "roll"} {
		if containsString(names, extra) {
			t.Errorf("Go-only social record %q leaked into the C command listing", extra)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
