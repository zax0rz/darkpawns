package game

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// liveLikeProvider mimics Manager.EachSession: it delivers through
// Player.SendMessage, which re-acquires World.mu (read) for the MessageSink.
// The milestone rig's closure never takes that lock, which is why the old
// HcontrolPayHouse (MudLog under the write lock) stayed green in unit tests
// while the live census saw silence on both audiences.
type liveLikeProvider struct {
	players []*Player
}

func (p *liveLikeProvider) EachSession(fn func(player any, send func(msg string))) {
	for _, ch := range p.players {
		player := ch
		fn(player, player.SendMessage)
	}
}

func TestHcontrolPayDeliversToActorAndWatcher(t *testing.T) {
	oldFilename := houseControlFilename
	houseControlFilename = filepath.Join(t.TempDir(), "house", "house_control.json")
	t.Cleanup(func() { houseControlFilename = oldFilename })

	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "House", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	actor := NewPlayer(1, "Houselord", 1001)
	actor.SetLevel(LVL_GRGOD)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	watcher.SetLevel(38)
	watcher.SetPlrFlag(PrfLog1, true)
	watcher.SetPlrFlag(PrfLog2, true)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	w.HouseControl = []HouseControl{{VNum: 1001, Owner: 1}}

	lines := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { lines[name] += string(msg) }

	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(&liveLikeProvider{players: []*Player{actor, watcher}})
	t.Cleanup(func() { SetImmortalSessionProvider(old) })

	// The old write-lock-across-MudLog deadlocked here; run with a bounded
	// wait so a regression fails fast instead of hanging the suite.
	done := make(chan struct{})
	go func() {
		w.HcontrolPayHouse(actor, "1001")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("HcontrolPayHouse deadlocked: actor and watcher delivery did not finish")
	}

	if got, want := lines["Houselord"], "Payment recorded.\r\n"; got != want {
		t.Fatalf("actor lines = %q, want %q", got, want)
	}
	if got, want := lines["Watcher"], "[ Payment for house 1001 collected by Houselord. ]\r\n"; got != want {
		t.Fatalf("watcher lines = %q, want %q", got, want)
	}
	if w.HouseControl[0].LastPayment == 0 {
		t.Fatal("LastPayment was not stamped")
	}
}
