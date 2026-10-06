package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestHelpMudlogActingBody(t *testing.T) {
	for _, mode := range []string{"ordinary", "player", "mob"} {
		t.Run(mode, func(t *testing.T) {
			w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 90, Keywords: "guard", ShortDesc: "a retained guard", Level: 40}}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			m := newTestManager(t, w, nil)
			a := makeTestSession(t, m, "Wizard", 1001, true)
			a.player.SetLevel(40)
			a.transportDone = make(chan struct{})
			registerTestSession(t, m, a, "Wizard")
			h := makeTestSession(t, m, "Borrowed", 1002, true)
			h.player = game.NewPlayer(2, "Borrowed", 1002)
			h.transportDone = make(chan struct{})
			h.DetachTransport()
			h.player.SetLinkless(true)
			registerTestSession(t, m, h, "Borrowed")
			m.world.HelpTable = []game.HelpEntry{{Keyword: "known", Entry: "known"}}
			watcher := makeTestSession(t, m, "Helpwatch", 1001, true)
			watcher.player.SetLevel(34)
			watcher.player.SetPlrFlag(game.PrfLog2, true)
			registerTestSession(t, m, watcher, "Helpwatch")
			name := a.player.GetName()
			switch mode {
			case "player":
				if err := cmdSwitch(a, []string{"Borrowed"}); err != nil {
					t.Fatal(err)
				}
				name = h.player.GetName()
			case "mob":
				mob, err := w.SpawnMobQuiet(90, 1001)
				if err != nil {
					t.Fatal(err)
				}
				if err := cmdSwitch(a, []string{"guard"}); err != nil {
					t.Fatal(err)
				}
				if !a.isSwitched || a.switchedMob != mob {
					t.Fatal("switch did not attach mob")
				}
				name = mob.GetName()
			}
			drainSessionText(t, a)
			drainSessionText(t, watcher)
			file := captureMudlogFile(t)
			if err := ExecuteCommand(a, "help", []string{"no-such-help"}); err != nil {
				t.Fatal(err)
			}
			payload := "HELP: " + name + " attempted to get help on no-such-help"
			if got := strings.Join(drainSessionText(t, watcher), ""); got != "[ "+payload+" ]\r\n" {
				t.Fatalf("body log=%q want %q", got, payload)
			}
			if !strings.Contains(file.String(), payload) {
				t.Fatalf("wrong file name: %q", file.String())
			}
			if got := strings.Join(drainSessionText(t, a), ""); got != "There is no help on: no-such-help\r\n" {
				t.Fatalf("miss ack=%q", got)
			}
		})
	}
}
