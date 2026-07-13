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
