package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func testOLCMemoryMudlog(t *testing.T, command string) {
	for _, save := range []bool{false, true} {
		t.Run(fmt.Sprint(save), func(t *testing.T) {
			w := makeSeditTestWorld(t)
			m := newTestManager(t, w, nil)
			a := makeCommandTestSession(t, m, "Memoryactor", 40, 3000)
			a.player.SetInvisLevel(40)
			a.player.SetPlrFlag(game.PrfLog1, true)
			a.player.SetPlrFlag(game.PrfLog2, true)
			watch := makeCommandTestSession(t, m, "Memorywatch", LVL_IMMORT, 3001)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			watch.player.SetPlrFlag(game.PrfLog2, true)
			below := makeCommandTestSession(t, m, "Memorybelow", LVL_IMMORT-1, 3001)
			below.player.SetPlrFlag(game.PrfLog1, true)
			below.player.SetPlrFlag(game.PrfLog2, true)
			normal := makeCommandTestSession(t, m, "Memorynormal", 40, 3001)
			normal.player.SetPlrFlag(game.PrfLog2, true)
			for _, s := range []*Session{a, watch, below, normal} {
				registerTestSession(t, m, s, s.player.Name)
			}
			var number, choice, kind, ack string
			switch command {
			case "redit":
				number = "3000"
				choice = "1"
				kind = "room"
				ack = "Room saved to memory.\r\n"
			case "zedit":
				number = "3000"
				choice = "z"
				kind = "zone info for room"
				ack = "Saving zone info in memory.\r\n"
			case "oedit":
				number = "3003"
				choice = "1"
				kind = "obj"
				ack = "Saving object to memory.\r\n"
			case "medit":
				number = "3002"
				choice = "2"
				kind = "mob"
				ack = "Saving mobile to memory.\r\n"
			case "sedit":
				number = "3001"
				choice = "7"
				kind = "shop"
				ack = "Saving shop to memory.\r\n"
			}
			input := func(line string) {
				switch command {
				case "redit":
					a.handleReditInput(line)
				case "zedit":
					a.handleZeditInput(line)
				case "oedit":
					a.handleOeditInput(line)
				case "medit":
					a.handleMeditInput(line)
				case "sedit":
					a.handleSeditInput(line)
				}
			}
			committed := func() bool {
				switch command {
				case "redit":
					v, _ := w.SnapshotRoom(3000)
					return v.Name == "memoryproof"
				case "zedit":
					v, _ := w.SnapshotZone(30)
					return v.Name == "memoryproof"
				case "oedit":
					v, _ := w.SnapshotObj(3003)
					return v.Keywords == "memoryproof"
				case "medit":
					v, _ := w.SnapshotMob(3002)
					return v.Keywords == "memoryproof"
				case "sedit":
					v, _ := w.SnapshotShop(3001)
					return v.Messages[0] == "%s memoryproof"
				}
				return false
			}
			if err := ExecuteCommand(a, command, []string{number}); err != nil {
				t.Fatal(err)
			}
			input(choice)
			input("memoryproof")
			input("q")
			for _, s := range []*Session{a, watch, below, normal} {
				drainSessionText(t, s)
			}
			file := captureMudlogFile(t)
			called := false
			game.SetLogWriter(&flagProbe{buf: file, when: func() {
				called = true
				if !committed() {
					t.Error("producer must follow memory commit")
				}
				if a.player.GetFlags()&(1<<uint(game.PlrWriting)) == 0 {
					t.Error("producer must precede writing cleanup")
				}
				got := strings.Join(drainSessionText(t, a), "")
				if command == "redit" {
					if a.roomEdit.pendingOutput != "" {
						t.Error("redit acknowledgement must not be buffered before producer")
					}
					if got != "" {
						t.Errorf("redit ack must follow producer: %q", got)
					}
				} else if got != ack {
					t.Errorf("ack must precede producer: %q", got)
				}
			}})
			if save {
				input("y")
			} else {
				input("n")
			}
			payload := fmt.Sprintf("OLC: Memoryactor edits %s %s", kind, number)
			got := strings.Join(drainSessionText(t, watch), "")
			if save {
				if !called || !strings.Contains(file.String(), payload) {
					t.Fatalf("missing memory producer: %q", file.String())
				}
				if got != "[ "+payload+" ]\r\n" {
					t.Fatalf("observer=%q", got)
				}
				if command == "redit" && strings.Join(drainSessionText(t, a), "") != ack {
					t.Fatal("missing post-log room acknowledgement")
				}
			} else if called || got != "" || committed() {
				t.Fatal("discard must not log or commit")
			}
			if len(below.send) != 0 || len(normal.send) != 0 {
				t.Fatal("memory producer threshold/type leak")
			}
			if a.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
				t.Fatal("editor writing flag retained")
			}
		})
	}
}
