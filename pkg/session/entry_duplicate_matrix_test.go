package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/interpreter.c:1533-1571; src/comm.c:2127-2141: a menu
// descriptor has never entered the game and has no departure audience.
func TestEntryDuplicateMenuClose(t *testing.T) {
	for _, stage := range []string{"motd", "menu"} {
		t.Run(stage, func(t *testing.T) {
			m := makeTestManagerWithVoidRooms(t)
			observer := makeTestSession(t, m, "Watcher", 1001, true)
			observer.player = game.NewPlayer(2, "Watcher", 1001)
			registerTestSession(t, m, observer, "Watcher")
			old := makeCharSession(t, m)
			old.player = game.NewPlayer(1, "Returner", 1001)
			old.authenticated = true
			old.menuActive = true
			old.menuStage = stage
			if err := m.Register("Returner", old); err != nil {
				t.Fatal(err)
			}
			fresh := makeCharSession(t, m)
			fresh.player = game.NewPlayer(1, "Returner", 1001)
			if fresh.performDupeCheck() {
				t.Fatal("menu descriptor supplied a live body")
			}
			if !old.SendClosed() || renderedOutput(old) != "\r\nMultiple login detected -- disconnecting.\r\n" {
				t.Fatal("menu duplicate did not close with C notice")
			}
			if got := renderedOutput(observer); got != "" {
				t.Fatalf("menu close invented room departure: %q", got)
			}
			if got := renderedOutput(fresh); got != "" {
				t.Fatalf("non-reconnect login received takeover output: %q", got)
			}
			if _, ok := m.GetSession("Returner"); ok {
				t.Fatal("old menu registration retained")
			}
			if m.world.GetPlayerCount() != 1 {
				t.Fatal("menu close changed live world")
			}
		})
	}
}

// Existing supported single-body modes: src/interpreter.c:1618-1653.
// The parent case retains its ID-sweep design blocker; this matrix does not
// claim arbitrary duplicate ID/name topology.
func TestEntryDuplicateCleanupMatrix(t *testing.T) {
	for _, mode := range []string{"reconnect", "usurp", "unswitch", "editor"} {
		t.Run(mode, func(t *testing.T) {
			m, old, observer, fresh := dupeCheckFixture(t)
			old.player.SetHP(17)
			old.player.SetPosition(game.PosResting)
			old.player.SetIdleTimer(9)
			old.player.SetPlrFlag(game.PlrMailing, true)
			old.player.SetPlrFlag(game.PlrWriting, true)
			old.player.SetSkill("bash", 37)
			switch mode {
			case "reconnect":
				old.player.SetLinkless(true)
				old.DetachTransport()
			case "unswitch":
				old.isSwitched = true
			case "editor":
				old.textEdit = &textEditState{}
			}
			if !fresh.performDupeCheck() {
				t.Fatal("supported duplicate mode did not adopt body")
			}
			assertTookOver(t, m, old, fresh)
			p := fresh.player
			if p.GetHP() != 17 || p.GetPosition() != game.PosResting || p.GetSkill("bash") != 37 || p.GetIdleTimer() != 0 || p.GetFlags()&((1<<uint(game.PlrMailing))|(1<<uint(game.PlrWriting))) != 0 {
				t.Fatal("live body state lost or descriptor-owned flags not reset")
			}
			if !old.SendClosed() || fresh.menuActive || fresh.charCreating || !fresh.authenticated {
				t.Fatal("descriptor cleanup/playing state")
			}
			actor := renderedOutput(fresh)
			room := renderedOutput(observer)
			switch mode {
			case "usurp":
				if !strings.Contains(actor, "You take over your own body, already in use!\r\n") || !strings.Contains(room, "body has been taken over by a new spirit!") {
					t.Fatal("usurp audience")
				}
			case "unswitch":
				if !strings.Contains(actor, "Reconnecting to unswitched char.") || room != "" {
					t.Fatal("unswitch audience")
				}
			default:
				if !strings.Contains(actor, "Reconnecting.\r\n") || room != "Returner has reconnected.\r\n" {
					t.Fatalf("reconnect audience: %q %q", actor, room)
				}
			}
			m.UnregisterSession(old)
			if current, _ := m.GetSession("Returner"); current != fresh {
				t.Fatal("stale teardown unregistered replacement")
			}
		})
	}
}
