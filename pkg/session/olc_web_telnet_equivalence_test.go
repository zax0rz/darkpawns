package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestWebAndTelnetRoomEditsProduceByteIdenticalZoneFiles is the webOLC design's
// acceptance sentence, executable: the same room edit performed once through
// the telnet menu choices and once through the web draft surface (DraftStore +
// ordered ops + CommitRoom — the exact machinery the HTTP handlers wrap) must
// produce byte-identical .wld output. Everything in the webOLC architecture is
// in service of this being true by construction.
func TestWebAndTelnetRoomEditsProduceByteIdenticalZoneFiles(t *testing.T) {
	build := func() (*game.World, *Manager, *Session) {
		w, err := game.NewWorld(&parser.World{
			Rooms: []parser.Room{
				{
					VNum: 1001, Name: "Original Room", Description: "A room.\n",
					Zone: 1, Flags: []string{"0", "0", "0", "0"}, Sector: 0,
					Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}},
				}, {VNum: 1002, Name: "Room B", Zone: 1, Flags: []string{"0", "0", "0", "0"}},
			},
			Zones: []parser.Zone{{Number: 1, TopRoom: 1999}},
		})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		w.WorldPath = t.TempDir()
		if err := os.MkdirAll(filepath.Join(w.WorldPath, "wld"), 0o755); err != nil {
			t.Fatal(err)
		}
		m := newTestManager(t, w, nil)
		return w, m, makeCommandTestSession(t, m, "Editgod", 40, 1001)
	}

	saveZone := func(w *game.World) string {
		zone, ok := olc.ZoneForVNum(w.GetAllZones(), 1001)
		if !ok {
			t.Fatal("no zone for 1001")
		}
		if err := saveReditZone(w, zone); err != nil {
			t.Fatalf("saveReditZone: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(w.WorldPath, "wld", "1.wld"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	// Path 1: telnet — the exact menu-choice sequence.
	_, mTelnet, telnetSession := build()
	if err := ExecuteCommand(telnetSession, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("1")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("Edited Room")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("2")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("/c")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("You are in the edited room.")
	telnetSession.handleReditInput("/s")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("4")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("3")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("q")
	_ = readMsgText(t, telnetSession)
	telnetSession.handleReditInput("y")
	if got := readMsgText(t, telnetSession); got != "Room saved to memory.\r\n" {
		t.Fatalf("telnet save = %q", got)
	}
	if mTelnet == nil {
		t.Fatal("unreachable")
	}
	telnetFile := saveZone(mTelnet.world)

	// Path 2: web — the draft surface the HTTP handlers wrap.
	wWeb, _, _ := build()
	owner := &webEquivalenceOwner{}
	room, ok := wWeb.SnapshotRoom(1001)
	if !ok {
		t.Fatal("snapshot 1001")
	}
	drafts := olc.NewDraftStore()
	draft, err := drafts.Open(owner.Identity(), olc.KindRoom, 1001, room)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	working := draft.Effective()
	draft, err = drafts.Patch(owner.Identity(), []olc.Operation{
		{Kind: olc.OpSetRoomName, Room: &working, Text: "Edited Room"},
		{Kind: olc.OpSetRoomDescription, Room: &working, Text: "You are in the edited room.\n"},
		{Kind: olc.OpSetRoomSector, Room: &working, Value: 3},
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	committed := olc.CommitRoom(olc.RoomCommitInput{
		Draft:  draft,
		Actor:  owner.Identity(),
		Commit: func(effective parser.Room) bool { return wWeb.CommitEditedRoom(effective) },
		MarkDirty: func() {
			zone, zok := olc.ZoneForVNum(wWeb.GetAllZones(), 1001)
			if zok {
				reditAddSaveRoom(zone.Number)
			}
		},
	})
	if !committed {
		t.Fatal("CommitRoom returned false")
	}
	webFile := saveZone(wWeb)

	if telnetFile != webFile {
		t.Fatalf("zone files differ:\ntelnet: %q\nweb:    %q", telnetFile, webFile)
	}
}

type webEquivalenceOwner struct{}

func (w *webEquivalenceOwner) Identity() string       { return "Webgod" }
func (w *webEquivalenceOwner) DisplayName() string    { return "Webgod" }
func (w *webEquivalenceOwner) Frontend() olc.Frontend { return olc.FrontendTelnet + "+test" }
