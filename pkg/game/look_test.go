package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestListObjToCharCorpse verifies listObjToChar doesn't panic on a
// Prototype-nil object (a corpse) and renders its Runtime long description,
// matching list_obj_to_char mode 20 used by C room observation.
func TestListObjToCharCorpse(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	corpse := w.makeCorpse("a goblin", 1, nil, nil, 1001, 0, 0, true)
	w.AddItemToRoom(corpse, 1001)

	room := w.GetRoomInWorld(1001)
	if room == nil {
		t.Fatalf("expected room 1001 to exist")
	}

	w.listObjToChar(room, ch) // must not panic on the Prototype-nil corpse
	if msg := lastMsg(); !strings.Contains(msg, "The corpse of a goblin is lying here.") {
		t.Errorf("expected corpse long desc in room listing, got %q", msg)
	}
}

// observationMessageText joins the deferred message formats of an observation
// result so tests can assert on the exact bytes a viewer would receive.
func observationMessageText(result ObservationResult) string {
	var b strings.Builder
	for _, message := range result.Messages {
		b.WriteString(message.Format)
		b.WriteString("\n")
	}
	return b.String()
}

// DP-1133: C show_obj_to_char (mode 5) colorizes the bless/magic/glow
// annotations at complete color level.
func TestDoLookTargetObjectColorizesObjectFlagsAtCompleteColor(t *testing.T) {
	w, p, _ := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	orb := newGlowFlaggedObject(5101)
	if err := p.Inventory.AddItem(orb); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	got := observationMessageText(w.DoLookTarget(p, "orb"))
	want := "You see nothing special.. (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)"
	if !strings.Contains(got, want) {
		t.Fatalf("object look at complete color = %q, want substring %q", got, want)
	}
}

// DP-1133: mode 6 (object with an extra description) goes through the same
// C show_obj_to_char flag code and must colorize identically.
func TestDoLookTargetObjectExtraDescColorizesObjectFlags(t *testing.T) {
	w, p, _ := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	orb := NewObjectInstance(&parser.Obj{
		VNum:      5102,
		ShortDesc: "a runed orb",
		Keywords:  "orb",
		Weight:    1,
		ExtraDescs: []parser.ExtraDesc{{
			Keywords:    "orb",
			Description: "Ancient runes cover the orb.",
		}},
	}, -1)
	orb.SetExtraFlag(0, itemExtraGlow)
	if err := p.Inventory.AddItem(orb); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	got := observationMessageText(w.DoLookTarget(p, "orb"))
	if !strings.Contains(got, "Ancient runes cover the orb.") {
		t.Fatalf("mode 6 look lost the extra description: %q", got)
	}
	if !strings.Contains(got, " (\x1b[37mglowing\x1b[0m)") {
		t.Fatalf("mode 6 look at complete color = %q, want white glowing annotation", got)
	}
}

// DP-1133: equipment shown by look <character> is C show_obj_to_char mode 1
// and colorizes at complete color level.
func TestDoLookTargetPlayerEquipmentColorizesObjectFlags(t *testing.T) {
	w, p, _ := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	wielder := NewPlayer(2, "Wielder", 100)
	if err := w.AddPlayer(wielder); err != nil {
		t.Fatalf("AddPlayer wielder: %v", err)
	}
	orb := newGlowFlaggedObject(5103)
	if err := wielder.Inventory.AddItem(orb); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := w.EquipItem(wielder, orb, eqWearWield); err != nil {
		t.Fatalf("Equip orb: %v", err)
	}

	got := observationMessageText(w.DoLookTarget(p, "wielder"))
	want := "a runed orb (\x1b[34mblue glow\x1b[0m) (\x1b[33myellow glow\x1b[0m) (\x1b[37mglowing\x1b[0m)"
	if !strings.Contains(got, want) {
		t.Fatalf("player equipment look at complete color = %q, want substring %q", got, want)
	}
}

// DP-1133 negative control: room object lines render C's list_obj_to_char →
// oc_show_list path, whose annotations stay plain at every color level. Test
// roomObjectLines directly so unrelated room-view ANSI (titles) is out of
// scope.
func TestRoomObjectLinesLeaveObjectFlagsUncolored(t *testing.T) {
	w, p, _ := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	orb := newGlowFlaggedObject(5104)
	w.AddItemToRoom(orb, 100)

	room := w.GetRoomInWorld(100)
	if room == nil {
		t.Fatalf("expected room 100 to exist")
	}
	got := strings.Join(w.roomObjectLines(p, room), "\n")
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("room object lines emitted ANSI on the oc_show_list path: %q", got)
	}
	if !strings.Contains(got, "(blue glow) (yellow glow) (glowing)") {
		t.Fatalf("room object lines lost plain annotations: %q", got)
	}
}

// DP-1133 negative control: container contents render the same plain C
// oc_show_list vocabulary.
func TestDoLookInContainerLeavesObjectFlagsUncolored(t *testing.T) {
	w, p, _ := newViewTestWorld(t)
	enableDetectAffects(p)
	setViewerColorLevel(p, 3)

	sack := NewObjectInstance(&parser.Obj{
		VNum: 5105, ShortDesc: "a leather sack", Keywords: "sack",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{1000, 0, 0, 0},
	}, -1)
	sack.Contains = append(sack.Contains, newGlowFlaggedObject(5106))
	if err := p.Inventory.AddItem(sack); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	got := observationMessageText(w.DoLookIn(p, "sack"))
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("container contents emitted ANSI on the oc_show_list path: %q", got)
	}
	if !strings.Contains(got, "(blue glow) (yellow glow) (glowing)") {
		t.Fatalf("container contents lost plain annotations: %q", got)
	}
}

func TestObservationResultDefersDeliveryUntilRendered(t *testing.T) {
	world, err := NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum:        1001,
		Name:        "Deferred Hall",
		Description: "The result owns this description.",
		Flags:       []string{"0", "0", "0", "0"},
	}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	viewer := NewPlayer(1, "Viewer", 1001)
	if err := world.AddPlayer(viewer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	var delivered strings.Builder
	world.MessageSink = func(_ string, message []byte) { delivered.Write(message) }

	result := world.DoLookRoom(viewer, true)
	if delivered.Len() != 0 {
		t.Fatalf("canonical operation delivered text before rendering: %q", delivered.String())
	}
	if result.Room == nil || result.Room.VNum != 1001 {
		t.Fatalf("RoomView = %#v, want room 1001", result.Room)
	}

	world.RenderObservationMessages(result)
	if !strings.Contains(delivered.String(), "Deferred Hall") {
		t.Fatalf("rendered messages omitted room name: %q", delivered.String())
	}
}

func TestLookTargetRoomExtraDescriptionPrecedesObjectExtraDescription(t *testing.T) {
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{
			VNum:  1001,
			Name:  "Precedence Room",
			Flags: []string{"0", "0", "0", "0"},
			ExtraDescs: []parser.ExtraDesc{{
				Keywords:    "sign plaque",
				Description: "The room-authored sign wins.",
			}},
		}},
		Objs: []parser.Obj{{
			VNum:      2001,
			Keywords:  "sign object",
			ShortDesc: "an object sign",
			LongDesc:  "An object sign is here.",
			ExtraDescs: []parser.ExtraDesc{{
				Keywords:    "sign",
				Description: "The object's sign loses.",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	viewer := NewPlayer(1, "Viewer", 1001)
	object, err := world.SpawnObject(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	world.AddItemToRoom(object, 1001)

	result := world.DoLookTarget(viewer, "sign")
	if len(result.Messages) != 1 {
		t.Fatalf("messages = %#v, want exactly the room extra description", result.Messages)
	}
	if got := result.Messages[0].Format; got != "The room-authored sign wins." {
		t.Fatalf("first target match = %q, want room extra description", got)
	}
}
