package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestReditInsertionMudlogBoundary(t *testing.T) {
	for _, kind := range []string{"insert", "replace", "abort", "concurrent-insert"} {
		t.Run(kind, func(t *testing.T) {
			isolateOLCSaveList(t)
			w := makeSeditTestWorld(t)
			m := newTestManager(t, w, nil)
			actor := makeCommandTestSession(t, m, "Roomactor", 40, 3000)
			watch := makeCommandTestSession(t, m, "Roomwatch", 31, 3001)
			watch.player.SetPlrFlag(game.PrfLog1, true)
			actor.player.SetInvisLevel(40)
			for _, s := range []*Session{actor, watch} {
				registerTestSession(t, m, s, s.playerName)
			}
			if err := ExecuteCommand(actor, "zedit", []string{"3000"}); err != nil {
				t.Fatal(err)
			}
			for _, line := range []string{"n", "0", "L", "0", "1", "n", "1", "L", "n", "1", "q", "y"} {
				actor.handleZeditInput(line)
			}
			if actor.zedit != nil {
				t.Fatal("real zedit loop creation did not finish")
			}
			if _, ok := w.CreateZone(31); !ok {
				t.Fatal("second zone")
			}
			if !w.CommitEditedZone(31, 0, parser.Zone{Number: 31, TopRoom: 3199, Commands: []parser.ZoneCommand{{Command: "L", Arg1: 3100, Arg2: 1, Arg3: -1}}}) {
				t.Fatal("cross-zone loop")
			}
			number := "3050"
			if kind == "replace" {
				number = "3000"
			}
			if err := ExecuteCommand(actor, "redit", []string{number}); err != nil {
				t.Fatal(err)
			}
			actor.handleReditInput("1")
			actor.handleReditInput("Diagnostic room")
			actor.handleReditInput("q")
			if kind == "concurrent-insert" {
				if !w.CommitEditedRoom(parser.Room{VNum: 3050, Zone: 30, Name: "Other commit"}) {
					t.Fatal("independent room insertion")
				}
			}
			drainSessionText(t, actor)
			drainSessionText(t, watch)
			file := captureMudlogFile(t)
			payload := "SYSERR: OLC: redit_save_internally: Unknown comand"
			count := 0
			sink := w.MessageSink
			w.MessageSink = func(name string, b []byte) {
				if name == "Roomwatch" && strings.Contains(string(b), payload) {
					count++
					if room, ok := w.SnapshotRoom(3050); !ok || room.Name != "Diagnostic room" {
						t.Error("diagnostic must follow room publication")
					}
					if olcSaveList.Dirty(olc.KindRoom, 30) {
						t.Error("diagnostic must precede dirty mark")
					}
				}
				sink(name, b)
			}
			answer := "y"
			if kind == "abort" {
				answer = "n"
			}
			actor.handleReditInput(answer)
			want := 0
			if kind == "insert" {
				want = 3
			}
			if count != want {
				t.Fatalf("%s diagnostic count=%d, want %d", kind, count, want)
			}
			if got := strings.Count(file.String(), payload); got != want {
				t.Errorf("file diagnostic count=%d, want %d", got, want)
			}
			if got := strings.Join(drainSessionText(t, watch), ""); got != strings.Repeat("[ "+payload+" ]\r\n", want) {
				t.Errorf("BRF/31/no-invis bytes=%q", got)
			}
		})
	}
}
