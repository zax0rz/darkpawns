package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1198: equipment display must match C do_equipment() exactly.
// C truth (src/act.informative.c:1468-1495):
//
//	"You are using:\r\n"                          ← ALWAYS, even when empty
//	per worn slot (wear-position order):
//	  visible   → where[i] + short desc
//	  invisible → where[i] + "Something.\r\n"
//	nothing worn → " Nothing.\r\n"                 ← leading space, after header
//
// C does NOT print an AC line in equipment — AC belongs to score.
// These tests retire the DP-1198 marker.

// C WEAR_* position ints (src/structs.h), used by World.EquipItem.
const (
	cWearBody  = 5
	cWearWield = 16
)

func setupEquipmentTest(t *testing.T) (*Manager, *Session) {
	t.Helper()
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Tester", 1001, true)
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	m.sessions[s.player.Name] = s
	return m, s
}

func TestEquipmentEmptyShowsCHeaderAndNothing(t *testing.T) {
	_, s := setupEquipmentTest(t)

	if err := cmdEquipment(s, nil); err != nil {
		t.Fatalf("cmdEquipment: %v", err)
	}
	got := readSessionText(t, s)
	want := "You are using:\r\n Nothing.\r\n"
	if got != want {
		t.Fatalf("empty equipment = %q, want %q", got, want)
	}
}

func TestEquipmentOneItemShowsHeaderAndSlotLabel(t *testing.T) {
	m, s := setupEquipmentTest(t)

	sword := game.NewObjectInstance(&parser.Obj{
		VNum:      2001,
		ShortDesc: "a small sword",
		Keywords:  "sword",
		WearFlags: [4]int{8193, 0, 0, 0}, // ITEM_WEAR_TAKE | ITEM_WEAR_WIELD
		Weight:    3,
	}, -1)
	if err := s.player.Inventory.AddItem(sword); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := m.world.EquipItem(s.player, sword, cWearWield); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}

	if err := cmdEquipment(s, nil); err != nil {
		t.Fatalf("cmdEquipment: %v", err)
	}
	got := readSessionText(t, s)

	if !strings.HasPrefix(got, "You are using:\r\n") {
		t.Fatalf("missing C header: %q", got)
	}
	if !strings.Contains(got, "<wielded>            a small sword\r\n") {
		t.Fatalf("missing wielded line: %q", got)
	}
	if strings.Contains(got, "Armor Class") || strings.Contains(got, "AC") {
		t.Fatalf("equipment must not print AC line (AC belongs to score): %q", got)
	}
	if strings.Contains(got, "not wearing") {
		t.Fatalf("must not print 'not wearing' — empty case uses ' Nothing.': %q", got)
	}
}

func TestEquipmentDoesNotPrintACLine(t *testing.T) {
	m, s := setupEquipmentTest(t)

	tunic := game.NewObjectInstance(&parser.Obj{
		VNum:      2002,
		ShortDesc: "a leather tunic",
		Keywords:  "tunic",
		WearFlags: [4]int{9, 0, 0, 0}, // ITEM_WEAR_TAKE | ITEM_WEAR_BODY
		Weight:    5,
	}, -1)
	if err := s.player.Inventory.AddItem(tunic); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := m.world.EquipItem(s.player, tunic, cWearBody); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}

	if err := cmdEquipment(s, nil); err != nil {
		t.Fatalf("cmdEquipment: %v", err)
	}
	got := readSessionText(t, s)

	if strings.Contains(got, "Armor Class") {
		t.Fatalf("equipment display must not contain AC line: %q", got)
	}
	if !strings.Contains(got, "<worn on body>       a leather tunic\r\n") {
		t.Fatalf("missing body line: %q", got)
	}
}
