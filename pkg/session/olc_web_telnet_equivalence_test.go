package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestWebAndTelnetEntityEditsProduceByteIdenticalZoneFiles extends the P4
// room proof across the remaining four writers. Each telnet sequence and its
// web equivalent drives the same Apply/commit primitive before the existing
// writer is invoked, so a frontend cannot quietly acquire a second encoding.
func TestWebAndTelnetEntityEditsProduceByteIdenticalZoneFiles(t *testing.T) {
	type entityCase struct {
		name   string
		ext    string
		telnet func(*Session)
		web    func(*game.World, *olc.DraftStore, string) error
	}

	build := func() (*game.World, *Manager) {
		root := t.TempDir()
		for _, dir := range []string{"mob", "obj", "shp", "zon", "wld"} {
			if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		w, err := game.NewWorld(&parser.World{
			SourceDir: root,
			Rooms: []parser.Room{
				{VNum: 1001, Name: "Room", Zone: 1, Exits: map[string]parser.Exit{}},
				{VNum: 1002, Name: "Room 2", Zone: 1, Exits: map[string]parser.Exit{}},
			},
			Mobs:  []parser.Mob{{VNum: 1101, Keywords: "guard", ShortDesc: "a guard", LongDesc: "A guard is here.\r\n", DetailedDesc: "A guard stands here.\r\n", ActionFlags: []string{"ISNPC"}, Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 1}, Damage: parser.DiceRoll{Num: 1, Sides: 1, Plus: 1}}},
			Objs:  []parser.Obj{{VNum: 1201, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword lies here."}},
			Shops: []parser.ShopProto{{VNum: 1301, KeeperVNum: 1101, Products: []int{1201}, OpenHour1: 0, CloseHour1: 28, OpenHour2: 0, CloseHour2: 28}},
			Zones: []parser.Zone{{Number: 1, Name: "Zone", TopRoom: 1999, Commands: []parser.ZoneCommand{{Command: "M", Arg1: 1101, Arg2: 1, Arg3: 1001}}}},
		})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		w.WorldPath = root
		t.Cleanup(w.StopAITicker)
		return w, newTestManager(t, w, nil)
	}

	read := func(dir, ext string, w *game.World) (string, error) {
		zone, ok := olc.ZoneForVNum(w.GetAllZones(), 1001)
		if !ok {
			return "", os.ErrNotExist
		}
		var err error
		switch ext {
		case "mob":
			err = saveMeditZone(w, zone)
		case "obj":
			err = saveOeditZone(w, zone)
		case "shp":
			err = saveSeditZone(w, zone)
		case "zon":
			err = saveZeditZone(w, zone)
		}
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(filepath.Join(dir, map[string]string{"mob": "mob/1.mob", "obj": "obj/1.obj", "shp": "shp/1.shp", "zon": "zon/1.zon"}[ext]))
		return string(data), err
	}

	cases := []entityCase{
		{
			name: "mob",
			ext:  "mob",
			telnet: func(s *Session) {
				_ = cmdMedit(s, []string{"1101"})
				_ = readMsgText(t, s)
				s.handleMeditInput("2")
				_ = readMsgText(t, s)
				s.handleMeditInput("edited guard")
				_ = readMsgText(t, s)
				s.handleMeditInput("q")
				_ = readMsgText(t, s)
				s.handleMeditInput("y")
				_ = readMsgText(t, s)
			},
			web: func(w *game.World, drafts *olc.DraftStore, owner string) error {
				mob, ok := w.SnapshotMob(1101)
				if !ok {
					return os.ErrNotExist
				}
				if _, err := drafts.OpenEntity(owner, olc.KindMob, 1101, olc.EntityValue{Mob: mob}); err != nil {
					return err
				}
				draft, err := drafts.PatchEntity(owner, []olc.Operation{{Kind: olc.OpSetMobKeywords, Text: "edited guard"}})
				if err != nil {
					return err
				}
				return commitWebMob(w, draft)
			},
		},
		{
			name: "object",
			ext:  "obj",
			telnet: func(s *Session) {
				_ = cmdOedit(s, []string{"1201"})
				_ = readMsgText(t, s)
				s.handleOeditInput("1")
				_ = readMsgText(t, s)
				s.handleOeditInput("edited sword")
				_ = readMsgText(t, s)
				s.handleOeditInput("q")
				_ = readMsgText(t, s)
				s.handleOeditInput("y")
				_ = readMsgText(t, s)
			},
			web: func(w *game.World, drafts *olc.DraftStore, owner string) error {
				object, ok := w.SnapshotObj(1201)
				if !ok {
					return os.ErrNotExist
				}
				if _, err := drafts.OpenEntity(owner, olc.KindObject, 1201, olc.EntityValue{Object: object}); err != nil {
					return err
				}
				draft, err := drafts.PatchEntity(owner, []olc.Operation{{Kind: olc.OpSetObjKeywords, Text: "edited sword"}})
				if err != nil {
					return err
				}
				return commitWebObj(w, draft)
			},
		},
		{
			name: "shop",
			ext:  "shp",
			telnet: func(s *Session) {
				_ = cmdSedit(s, []string{"1301"})
				_ = readMsgText(t, s)
				s.handleSeditInput("1")
				_ = readMsgText(t, s)
				s.handleSeditInput("12")
				_ = readMsgText(t, s)
				s.handleSeditInput("q")
				_ = readMsgText(t, s)
				s.handleSeditInput("y")
				_ = readMsgText(t, s)
			},
			web: func(w *game.World, drafts *olc.DraftStore, owner string) error {
				shop, ok := w.SnapshotShop(1301)
				if !ok {
					return os.ErrNotExist
				}
				proto := parser.ShopProto{VNum: shop.VNum, Products: append([]int(nil), shop.SellTypes...), BuyProfit: shop.ProfitBuy, SellProfit: shop.ProfitSell, KeeperVNum: shop.KeeperVNum, WithWho: shop.WithWho, Rooms: append([]int(nil), shop.Rooms...), OpenHour1: shop.OpenHour1, CloseHour1: shop.CloseHour1, OpenHour2: shop.OpenHour2, CloseHour2: shop.CloseHour2}
				if _, err := drafts.OpenEntity(owner, olc.KindShop, 1301, olc.EntityValue{Shop: proto}); err != nil {
					return err
				}
				draft, err := drafts.PatchEntity(owner, []olc.Operation{{Kind: olc.OpSetShopOpenHour1, Value: 12}})
				if err != nil {
					return err
				}
				return commitWebShop(w, draft)
			},
		},
		{
			name: "zone",
			ext:  "zon",
			telnet: func(s *Session) {
				_ = cmdZedit(s, []string{"1001"})
				_ = readMsgText(t, s)
				s.handleZeditInput("z")
				_ = readMsgText(t, s)
				s.handleZeditInput("Edited Zone")
				_ = readMsgText(t, s)
				s.handleZeditInput("q")
				_ = readMsgText(t, s)
				s.handleZeditInput("y")
				_ = readMsgText(t, s)
			},
			web: func(w *game.World, drafts *olc.DraftStore, owner string) error {
				zone, ok := w.SnapshotZone(1)
				if !ok {
					return os.ErrNotExist
				}
				zone.Commands = olc.ZoneCommandsForRoom(zone.Commands, 1001)
				if _, err := drafts.OpenEntity(owner, olc.KindZone, 1001, olc.EntityValue{Zone: olc.ZoneDraft{Zone: zone, RoomVNum: 1001}}); err != nil {
					return err
				}
				draft, err := drafts.PatchEntity(owner, []olc.Operation{{Kind: olc.OpSetZoneName, Text: "Edited Zone"}})
				if err != nil {
					return err
				}
				return commitWebZone(w, draft)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			wTelnet, manager := build()
			telnet := makeCommandTestSession(t, manager, "Equiv", 40, 1001)
			test.telnet(telnet)
			telnetBytes, err := read(wTelnet.WorldPath, test.ext, wTelnet)
			if err != nil {
				t.Fatal(err)
			}

			wWeb, _ := build()
			drafts := olc.NewDraftStore()
			if err := test.web(wWeb, drafts, "web-equiv"); err != nil {
				t.Fatal(err)
			}
			webBytes, err := read(wWeb.WorldPath, test.ext, wWeb)
			if err != nil {
				t.Fatal(err)
			}
			if telnetBytes != webBytes {
				t.Fatalf("%s bytes differ:\ntelnet=%q\nweb=%q", test.name, telnetBytes, webBytes)
			}
		})
	}
}

func commitWebMob(w *game.World, draft olc.EntityDraft) error {
	zone, ok := olc.ZoneForVNum(w.GetAllZones(), draft.VNum)
	if !ok {
		return fmt.Errorf("mob has no zone")
	}
	lock := olc.ZoneSaveLock(zone.Number)
	lock.Lock()
	defer lock.Unlock()
	if !olc.CommitMob(olc.MobCommitInput{Draft: draft, Commit: w.CommitEditedMob}) {
		return fmt.Errorf("mob commit failed")
	}
	return nil
}

func commitWebObj(w *game.World, draft olc.EntityDraft) error {
	zone, ok := olc.ZoneForVNum(w.GetAllZones(), draft.VNum)
	if !ok {
		return fmt.Errorf("object has no zone")
	}
	lock := olc.ZoneSaveLock(zone.Number)
	lock.Lock()
	defer lock.Unlock()
	if !olc.CommitObj(olc.ObjectCommitInput{Draft: draft, Commit: w.CommitEditedObj}) {
		return fmt.Errorf("object commit failed")
	}
	return nil
}

func commitWebShop(w *game.World, draft olc.EntityDraft) error {
	zone, ok := olc.ZoneForVNum(w.GetAllZones(), draft.VNum)
	if !ok {
		return fmt.Errorf("shop has no zone")
	}
	lock := olc.ZoneSaveLock(zone.Number)
	lock.Lock()
	defer lock.Unlock()
	if !olc.CommitShop(olc.ShopCommitInput{Draft: draft, Commit: w.CommitEditedShop}) {
		return fmt.Errorf("shop commit failed")
	}
	return nil
}

func commitWebZone(w *game.World, draft olc.EntityDraft) error {
	zone, ok := olc.ZoneForVNum(w.GetAllZones(), draft.VNum)
	if !ok {
		return fmt.Errorf("zone has no room")
	}
	lock := olc.ZoneSaveLock(zone.Number)
	lock.Lock()
	defer lock.Unlock()
	if !olc.CommitZone(olc.ZoneCommitInput{Draft: draft, Commit: func(room int, working parser.Zone) bool { return w.CommitEditedZone(zone.Number, room, working) }}) {
		return fmt.Errorf("zone commit failed")
	}
	return nil
}

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
