package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestLookNullDescriptionFallsThroughToObjectView is the R5e look proof:
// find_exdesc returns the first keyword match's description pointer, and
// do_look tests it against NULL (src/act.informative.c:987-996, :1050-1052).
// A keyword match whose description pointer is NULL is therefore "not found":
// the caller falls through to the generic object view, and it does NOT continue
// scanning later nodes for another match.
func TestLookNullDescriptionFallsThroughToObjectView(t *testing.T) {
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Null Desc Room", Flags: []string{"0", "0", "0", "0"}}},
		Objs: []parser.Obj{{
			VNum:      2001,
			Keywords:  "altar",
			ShortDesc: "a stone altar",
			LongDesc:  "A stone altar is here.",
			ExtraDescs: []parser.ExtraDesc{
				// C after a load of "E\n<kw>~\n~\n": keyword set, description NULL.
				parser.NewLoadedExtraDesc("altar", ""),
				// A later node C never reaches for this arg.
				{Keywords: "altar", Description: "second"},
			},
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

	got := observationMessageText(world.DoLookTarget(viewer, "altar"))
	if strings.Contains(got, "second") {
		t.Fatalf("the scan continued past the first keyword match: %q", got)
	}
	if !strings.Contains(got, "You see nothing special..") {
		t.Fatalf("a NULL description must fall through to the generic object view: %q", got)
	}
}
