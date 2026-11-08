package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPerformGetFromRoomQuietSuppressesRoomAct pins DP-1508 (R1/R5h): the
// quiet argument ports perform_get_from_room's third parameter
// (src/act.item.c:269-283). A quiet pickup (do_palm, src/new_cmds.c:2122) shows
// the actor "You get $p." but no "$n gets $p." to the room; an ordinary get
// (src/act.item.c:362, quiet=false) keeps the room act. Revert proof: forcing
// the room act on the quiet path fails the observer assertion.
func TestPerformGetFromRoomQuietSuppressesRoomAct(t *testing.T) {
	for _, tc := range []struct {
		name     string
		quiet    bool
		wantRoom string
	}{
		{"quiet pickup hides the room act", true, ""},
		{"normal get keeps the room act", false, "Pracgetter gets a loaf of bread."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWorld(&parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)

			var actorMsgs, obsMsgs strings.Builder
			w.MessageSink = func(name string, msg []byte) {
				switch name {
				case "Pracgetter":
					actorMsgs.Write(msg)
				case "Pracobserver":
					obsMsgs.Write(msg)
				}
			}

			actor := NewPlayer(1, "Pracgetter", 1001)
			actor.Stats = CharStats{Str: 20, Dex: 20}
			actor.CopyBaseAttributes()
			actor.SetPosition(8)
			if err := w.AddPlayer(actor); err != nil {
				t.Fatal(err)
			}
			observer := NewPlayer(2, "Pracobserver", 1001)
			observer.SetPosition(8)
			if err := w.AddPlayer(observer); err != nil {
				t.Fatal(err)
			}

			obj := NewObjectInstance(&parser.Obj{
				VNum: 8010, Keywords: "loaf bread", ShortDesc: "a loaf of bread",
				WearFlags: [4]int{1}, Weight: 1,
			}, 1001)
			if err := w.MoveObjectToRoomFront(obj, 1001); err != nil {
				t.Fatal(err)
			}

			w.performGetFromRoom(actor, obj, tc.quiet)

			if !strings.Contains(actorMsgs.String(), "You get a loaf of bread.") {
				t.Fatalf("actor got %q, want the pickup line", actorMsgs.String())
			}
			if tc.wantRoom == "" {
				if strings.Contains(obsMsgs.String(), "gets") {
					t.Fatalf("quiet pickup leaked a room act: %q", obsMsgs.String())
				}
			} else if !strings.Contains(obsMsgs.String(), tc.wantRoom) {
				t.Fatalf("observer got %q, want %q", obsMsgs.String(), tc.wantRoom)
			}
			if obj.Location.Kind != ObjInInventory {
				t.Fatalf("object location = %v, want carried", obj.Location.Kind)
			}
		})
	}
}
