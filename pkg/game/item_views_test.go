package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newViewTestWorld(t *testing.T) (*World, *Player, *[]string) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Name: "Test Room"}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	p := NewPlayer(1, "Viewer", 100)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	var msgs []string
	w.MessageSink = func(name string, msg []byte) {
		if name == p.Name {
			msgs = append(msgs, string(msg))
		}
	}
	return w, p, &msgs
}

func TestDoInventoryEmpty(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)
	w.DoInventory(p)
	got := strings.Join(*msgs, "")
	want := "You are carrying:\r\nNothing.\r\n"
	if got != want {
		t.Fatalf("empty inventory = %q, want %q", got, want)
	}
}

func TestDoInventoryGroupsIdenticalItems(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)

	for i := 0; i < 3; i++ {
		obj := NewObjectInstance(&parser.Obj{
			VNum:      1001,
			ShortDesc: "a wooden stick",
			Keywords:  "stick",
			Weight:    2,
		}, -1)
		if err := p.Inventory.AddItem(obj); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
	}
	obj := NewObjectInstance(&parser.Obj{
		VNum:      1002,
		ShortDesc: "a pebble",
		Keywords:  "pebble",
		Weight:    1,
	}, -1)
	if err := p.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	w.DoInventory(p)
	got := strings.Join(*msgs, "")

	if !strings.HasPrefix(got, "You are carrying:\r\n") {
		t.Fatalf("missing header: %q", got)
	}
	header := "\r\n Num  Item   " + strings.Repeat(" ", 51) + "Encumbrance\r\n"
	if !strings.Contains(got, header) {
		t.Fatalf("missing wide header: %q", got)
	}
	wantSingle := fmt.Sprintf("  1   %-63s%2d pt\r\n", "a pebble", 1)
	if !strings.Contains(got, wantSingle) {
		t.Fatalf("missing single item line: %q", got)
	}
	wantGroup := fmt.Sprintf("  3   %-63s%2d pts ea.\r\n", "a wooden stick", 2)
	if !strings.Contains(got, wantGroup) {
		t.Fatalf("missing grouped item line: %q", got)
	}
}

func TestDoEquipmentEmpty(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)
	w.DoEquipment(p)
	got := strings.Join(*msgs, "")
	want := "You are using:\r\n Nothing.\r\n"
	if got != want {
		t.Fatalf("empty equipment = %q, want %q", got, want)
	}
}

func TestDoEquipmentUsesCOrderAndLabels(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)

	sword := NewObjectInstance(&parser.Obj{
		VNum: 2001, ShortDesc: "a small sword", Keywords: "sword",
		WearFlags: [4]int{8193, 0, 0, 0}, Weight: 3,
	}, -1)
	tunic := NewObjectInstance(&parser.Obj{
		VNum: 2002, ShortDesc: "a frayed tunic", Keywords: "tunic",
		WearFlags: [4]int{9, 0, 0, 0}, Weight: 10,
	}, -1)
	torch := NewObjectInstance(&parser.Obj{
		VNum: 2003, ShortDesc: "a torch", Keywords: "torch",
		WearFlags: [4]int{16385, 0, 0, 0}, Weight: 1, TypeFlag: 1,
	}, -1)

	for _, item := range []*ObjectInstance{tunic, sword, torch} {
		if err := p.Inventory.AddItem(item); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
	}

	if err := w.EquipItem(p, tunic, eqWearBody); err != nil {
		t.Fatalf("Equip tunic: %v", err)
	}
	if err := w.EquipItem(p, sword, eqWearWield); err != nil {
		t.Fatalf("Equip sword: %v", err)
	}
	if err := w.EquipItem(p, torch, eqWearHold); err != nil {
		t.Fatalf("Equip torch: %v", err)
	}

	w.DoEquipment(p)
	got := strings.Join(*msgs, "")

	bodyIdx := strings.Index(got, "<worn on body>")
	wieldIdx := strings.Index(got, "<wielded>")
	holdIdx := strings.Index(got, "<held>")
	if bodyIdx == -1 || wieldIdx == -1 || holdIdx == -1 {
		t.Fatalf("missing slot labels: %q", got)
	}
	if bodyIdx >= wieldIdx || wieldIdx >= holdIdx {
		t.Fatalf("wrong C order: body=%d wield=%d hold=%d", bodyIdx, wieldIdx, holdIdx)
	}
	if !strings.Contains(got, "<worn on body>       a frayed tunic\r\n") {
		t.Fatalf("body line mismatch: %q", got)
	}
	if !strings.Contains(got, "<wielded>            a small sword\r\n") {
		t.Fatalf("wield line mismatch: %q", got)
	}
	if !strings.Contains(got, "<held>               a torch\r\n") {
		t.Fatalf("hold line mismatch: %q", got)
	}
}

func TestDoEquipmentUnseenItemShowsSomething(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)

	ring := NewObjectInstance(&parser.Obj{
		VNum: 3001, ShortDesc: "an invisible ring", Keywords: "ring",
		WearFlags: [4]int{3, 0, 0, 0}, Weight: 1,
	}, -1)
	ring.SetExtraFlag(0, extraFlagInvisible)
	if err := p.Inventory.AddItem(ring); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := w.EquipItem(p, ring, eqWearFingerR); err != nil {
		t.Fatalf("Equip ring: %v", err)
	}

	w.DoEquipment(p)
	got := strings.Join(*msgs, "")
	if !strings.Contains(got, "<worn on finger>     Something.\r\n") {
		t.Fatalf("unseen item line = %q", got)
	}
}
