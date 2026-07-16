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

// ---------------------------------------------------------------------------
// DP-1133: object-flag color parity with C show_obj_to_char / oc_show_list
// ---------------------------------------------------------------------------

// newGlowFlaggedObject builds a wieldable object carrying the bless, magic,
// and glow extra flags (ITEM_BLESS/ITEM_MAGIC/ITEM_GLOW).
func newGlowFlaggedObject(vnum int) *ObjectInstance {
	obj := NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: "a runed orb",
		LongDesc:  "A runed orb has been left here.",
		Keywords:  "orb",
		WearFlags: [4]int{8193, 0, 0, 0},
		Weight:    1,
	}, -1)
	obj.SetExtraFlag(0, itemExtraBless)
	obj.SetExtraFlag(0, itemExtraMagic)
	obj.SetExtraFlag(0, itemExtraGlow)
	return obj
}

func enableDetectAffects(p *Player) {
	p.SetAffect(affDetectAlign, true)
	p.SetAffect(affDetectMagic, true)
}

// setViewerColorLevel sets the PRF_COLOR bits so the viewer's computed color
// level is exactly level (PrfColor1 adds 1, PrfColor2 adds 2).
func setViewerColorLevel(p *Player, level int) {
	p.SetPlrFlag(PrfColor1, level == 1 || level == 3)
	p.SetPlrFlag(PrfColor2, level >= 2)
}

// C show_obj_to_char wraps the bless/magic/glow annotations in KBLU/KYEL/KWHT
// only at COLOR_LEV(ch)==C_CMP (complete color, level 3).
func TestDoEquipmentColorizesObjectFlagsAtCompleteColor(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	orb := newGlowFlaggedObject(5001)
	if err := p.Inventory.AddItem(orb); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := w.EquipItem(p, orb, eqWearWield); err != nil {
		t.Fatalf("Equip orb: %v", err)
	}

	w.DoEquipment(p)
	got := strings.Join(*msgs, "")
	want := "<wielded>            a runed orb (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)\r\n"
	if !strings.Contains(got, want) {
		t.Fatalf("DoEquipment complete-color line = %q, want substring %q", got, want)
	}
}

// At color levels 0-2 C keeps the annotations byte-for-byte plain.
func TestDoEquipmentObjectFlagsStayPlainBelowCompleteColor(t *testing.T) {
	for level := 0; level <= 2; level++ {
		t.Run(fmt.Sprintf("level%d", level), func(t *testing.T) {
			w, p, msgs := newViewTestWorld(t)
			enableDetectAffects(p)
			setViewerColorLevel(p, level)

			orb := newGlowFlaggedObject(5002)
			if err := p.Inventory.AddItem(orb); err != nil {
				t.Fatalf("AddItem: %v", err)
			}
			if err := w.EquipItem(p, orb, eqWearWield); err != nil {
				t.Fatalf("Equip orb: %v", err)
			}

			w.DoEquipment(p)
			got := strings.Join(*msgs, "")
			want := "<wielded>            a runed orb (blue glow) (yellow glow) (glowing)\r\n"
			if !strings.Contains(got, want) {
				t.Fatalf("DoEquipment level %d line = %q, want substring %q", level, got, want)
			}
			if strings.Contains(got, "\x1b[") {
				t.Fatalf("DoEquipment level %d emitted ANSI: %q", level, got)
			}
		})
	}
}

// DoInventory renders C's oc_show_list path, which prints "...it glows blue/
// gold/white" without ANSI at every color level — even complete.
func TestDoInventoryLeavesObjectFlagsUncolored(t *testing.T) {
	w, p, msgs := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	orb := newGlowFlaggedObject(5003)
	if err := p.Inventory.AddItem(orb); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	w.DoInventory(p)
	got := strings.Join(*msgs, "")
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("DoInventory emitted ANSI on the oc_show_list path: %q", got)
	}
	for _, want := range []string{"...it glows blue", "...it glows gold", "...it glows white"} {
		if !strings.Contains(got, want) {
			t.Fatalf("DoInventory lost plain annotation %q: %q", want, got)
		}
	}
}

// The bless annotation requires AFF_DETECT_ALIGN and the magic annotation
// requires AFF_DETECT_MAGIC, independent of color level.
func TestObjectFlagAnnotationsRequireDetectAffects(t *testing.T) {
	cases := []struct {
		name        string
		detectAlign bool
		detectMagic bool
		wantBlue    bool
		wantYellow  bool
	}{
		{"no detect affects", false, false, false, false},
		{"detect align only", true, false, true, false},
		{"detect magic only", false, true, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, p, msgs := newViewTestWorld(t)
			setViewerColorLevel(p, 3)
			p.SetAffect(affDetectAlign, tc.detectAlign)
			p.SetAffect(affDetectMagic, tc.detectMagic)

			orb := newGlowFlaggedObject(5004)
			if err := p.Inventory.AddItem(orb); err != nil {
				t.Fatalf("AddItem: %v", err)
			}
			if err := w.EquipItem(p, orb, eqWearWield); err != nil {
				t.Fatalf("Equip orb: %v", err)
			}

			w.DoEquipment(p)
			got := strings.Join(*msgs, "")
			if strings.Contains(got, "blue glow") != tc.wantBlue {
				t.Fatalf("blue glow presence = %v, want %v: %q", !tc.wantBlue, tc.wantBlue, got)
			}
			if strings.Contains(got, "yellow glow") != tc.wantYellow {
				t.Fatalf("yellow glow presence = %v, want %v: %q", !tc.wantYellow, tc.wantYellow, got)
			}
			if !strings.Contains(got, "glowing") {
				t.Fatalf("glow annotation missing (needs no detect affect): %q", got)
			}
		})
	}
}

// Exact helper/context matrix: one shared renderer, both color modes, all
// four color levels. Complete level 3 wraps only the three glow phrases in
// ANSI; levels 0-2 and the plain renderer stay byte-for-byte identical to C's
// uncolored text.
func TestObjectFlagAnnotationsColorMatrix(t *testing.T) {
	const plain = " (blue glow) (yellow glow) (glowing)"
	const colored = " (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)"
	for level := 0; level <= 3; level++ {
		t.Run(fmt.Sprintf("level%d", level), func(t *testing.T) {
			_, p, _ := newViewTestWorld(t)
			enableDetectAffects(p)
			setViewerColorLevel(p, level)
			orb := newGlowFlaggedObject(5005)

			if got := objectVisibleFlags(p, orb); got != plain {
				t.Fatalf("objectVisibleFlags at level %d = %q, want plain %q", level, got, plain)
			}
			want := plain
			if level == 3 {
				want = colored
			}
			if got := coloredObjectVisibleFlags(p, orb); got != want {
				t.Fatalf("coloredObjectVisibleFlags at level %d = %q, want %q", level, got, want)
			}
		})
	}
}

// The invisible and humming annotations are never colored in C, even at
// complete color level.
func TestObjectFlagAnnotationsInvisibleAndHumStayPlain(t *testing.T) {
	_, p, _ := newViewTestWorld(t)
	setViewerColorLevel(p, 3)

	obj := NewObjectInstance(&parser.Obj{
		VNum: 5006, ShortDesc: "a buzzing shroud", Keywords: "shroud", Weight: 1,
	}, -1)
	obj.SetExtraFlag(0, itemExtraInvisible)
	obj.SetExtraFlag(0, itemExtraHum)

	got := coloredObjectVisibleFlags(p, obj)
	if want := " (invisible) (humming)"; got != want {
		t.Fatalf("coloredObjectVisibleFlags(invisible+hum) = %q, want %q", got, want)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("invisible/hum annotations carried ANSI: %q", got)
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
