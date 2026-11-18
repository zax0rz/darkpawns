package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func makeScrollPrototype() *parser.Obj {
	return &parser.Obj{
		VNum:      99002,
		Keywords:  "test scroll",
		ShortDesc: "a test scroll",
		LongDesc:  "A test scroll lies here.\n",
		TypeFlag:  game.ITEM_SCROLL,
		WearFlags: [4]int{1 << 0}, // TAKE
		Values:    [4]int{1, -1, -1, -1},
		Weight:    1,
		Cost:      1,
	}
}

func TestRecite_ActMessagesMatchCForActorAndRoomObserver(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Reciter", 1001, true)
	observer := makeTestSession(t, m, "Watcher", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, observer)

	scroll := game.NewObjectInstance(makeScrollPrototype(), -1)
	if err := actor.player.Inventory.AddItem(scroll); err != nil {
		t.Fatalf("add scroll to inventory: %v", err)
	}

	if err := cmdRecite(actor, []string{"scroll"}); err != nil {
		t.Fatalf("cmdRecite: %v", err)
	}

	actorText := readSessionText(t, actor)
	if want := "You recite a test scroll which dissolves.\r\n"; actorText != want {
		t.Errorf("actor message = %q, want %q", actorText, want)
	}
	observerText := readSessionText(t, observer)
	if want := "Reciter recites a test scroll.\r\n"; observerText != want {
		t.Errorf("observer message = %q, want %q", observerText, want)
	}
	if strings.Contains(observerText, "$") {
		t.Errorf("observer saw an unresolved act token: %q", observerText)
	}
}

// drainSendChannel returns all text accumulated on the session's send channel.
func drainSendChannel(t *testing.T, s *Session) string {
	t.Helper()
	var out strings.Builder
	for {
		select {
		case msg := <-s.send:
			out.Write(msg)
		default:
			return out.String()
		}
	}
}
