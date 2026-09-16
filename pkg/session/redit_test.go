package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newReditTestManager(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        1001,
				Name:        "Room A",
				Description: "A room.\n",
				Zone:        1,
				Flags:       []string{"0", "0", "0", "0"},
				Sector:      0,
				Exits:       make(map[string]parser.Exit),
			},
			{VNum: 1002, Name: "Room B", Zone: 1, Flags: []string{"0", "0", "0", "0"}},
		},
		Zones: []parser.Zone{{Number: 1, TopRoom: 1999}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return newTestManager(t, w, nil)
}

func TestReditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["redit"]
	if !ok {
		t.Fatal("redit command has no C gate")
	}
	if gate.MinLevel != game.LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("redit gate = (%d,%d), want (%d,%d)", gate.MinLevel, gate.MinPosition, game.LVL_IMMORT, combat.PosDead)
	}
	entry, ok := cmdRegistry.Lookup("redit")
	if !ok {
		t.Fatal("redit command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("redit registry gate = (%d,%d), want (%d,%d)", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestReditMainMenuAndNoDirtyQuit(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	got := readMsgText(t, s)
	want := "\r\n-- Room number : [1001]      Room zone: [1]\r\n" +
		"1) Name        : Room A\r\n" +
		"2) Description :\r\nA room.\r\n" +
		"3) Room flags  : NOBITS \r\n" +
		"4) Sector type : Inside\r\n" +
		"5) Exit north  : -1\r\n" +
		"6) Exit east   : -1\r\n" +
		"7) Exit south  : -1\r\n" +
		"8) Exit west   : -1\r\n" +
		"9) Exit up     : -1\r\n" +
		"A) Exit down   : -1\r\n" +
		"B) Extra descriptions menu\r\n" +
		"C) Copy another room description\r\n" +
		"S) Script menu\r\n" +
		"Q) Quit\r\n" +
		"Enter choice : "
	if got != want {
		t.Fatalf("main menu = %q, want %q", got, want)
	}
	s.handleReditInput("q")
	if got := readMsgTextOrEmpty(t, s); got != "" {
		t.Fatalf("clean quit output = %q, want empty", got)
	}
	if s.isRoomEditing() || s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
		t.Fatal("clean quit left REDIT active")
	}
}

func TestReditWorkingCopySaveAndAbort(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("1")
	_ = readMsgText(t, s)
	s.handleReditInput("Edited room")
	_ = readMsgText(t, s)
	if room := m.world.GetRoomInWorld(1001); room.Name != "Room A" {
		t.Fatalf("working name leaked before confirm: %q", room.Name)
	}
	s.handleReditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save this room internally? : " {
		t.Fatalf("confirm prompt = %q", got)
	}
	s.handleReditInput("n")
	if room := m.world.GetRoomInWorld(1001); room.Name != "Room A" {
		t.Fatalf("abort changed live name: %q", room.Name)
	}

	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("1")
	_ = readMsgText(t, s)
	s.handleReditInput("Saved room")
	_ = readMsgText(t, s)
	s.handleReditInput("q")
	_ = readMsgText(t, s)
	s.handleReditInput("y")
	if got := readMsgText(t, s); got != "Room saved to memory.\r\n" {
		t.Fatalf("save confirmation = %q", got)
	}
	if room := m.world.GetRoomInWorld(1001); room.Name != "Saved room" {
		t.Fatalf("saved name = %q", room.Name)
	}
}

func TestReditExitAndStringAbortReturnToMenu(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("5")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n1) Exit to     : 1001\r\n") {
		t.Fatalf("new exit menu = %q", got)
	}
	s.handleReditInput("2")
	_ = readMsgText(t, s)
	s.handleReditInput("draft exit")
	s.handleReditInput("/a")
	got := readMsgText(t, s)
	if !strings.HasPrefix(got, "\r\n1) Exit to     : 1001\r\n2) Description :-\r\n<NONE>\r\n") {
		t.Fatalf("exit abort menu = %q", got)
	}
	if exit := m.world.GetRoomInWorld(1001).Exits["north"]; exit.Description != "" {
		t.Fatalf("exit abort leaked description: %q", exit.Description)
	}
}

func TestReditDisconnectDropsWorkingStringEditor(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("2")
	_ = readMsgText(t, s)
	s.handleReditInput("unsaved description")
	s.cancelTextEdit()
	s.cancelRoomEdit()

	room, ok := m.world.SnapshotRoom(1001)
	if !ok || room.Description != "A room.\n" {
		t.Fatalf("disconnect changed room description = (%q, %v)", room.Description, ok)
	}
	if s.isRoomEditing() || s.isTextEditing() || s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
		t.Fatal("disconnect left REDIT state active")
	}
}

func TestReditWorkingCopySurvivesOverlappingAdminWrite(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("1")
	_ = readMsgText(t, s)
	s.handleReditInput("OLC draft")
	_ = readMsgText(t, s)
	if !m.world.SetRoomName(1001, "Web admin") {
		t.Fatal("web-admin room write failed")
	}
	if room := m.world.GetRoomInWorld(1001); room.Name != "Web admin" {
		t.Fatalf("web-admin name = %q", room.Name)
	}
	s.handleReditInput("q")
	_ = readMsgText(t, s)
	s.handleReditInput("n")
	if room := m.world.GetRoomInWorld(1001); room.Name != "Web admin" {
		t.Fatalf("discarded OLC draft replaced web-admin name: %q", room.Name)
	}
}

func TestReditNewRoomCommitAndDiskSave(t *testing.T) {
	m := newReditTestManager(t)
	s := makeCommandTestSession(t, m, "Reditgod", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(s, "redit", []string{"1050"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleReditInput("1")
	_ = readMsgText(t, s)
	s.handleReditInput("Inserted")
	_ = readMsgText(t, s)
	s.handleReditInput("q")
	_ = readMsgText(t, s)
	s.handleReditInput("y")
	_ = readMsgText(t, s)
	if room, ok := m.world.SnapshotRoom(1050); !ok || room.Name != "Inserted" || room.Zone != 1 {
		t.Fatalf("new room after commit = %#v, exists=%v", room, ok)
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "wld"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.world.WorldPath = dir
	if err := ExecuteCommand(s, "redit", []string{"save", "1"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Saving all rooms in zone.\r\n" {
		t.Fatalf("disk-save banner = %q", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "wld", "1.wld"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "#1050\nInserted~\nYou are in an unfinished room.\n~\n1 0 0 0 0 0\nS\n") {
		t.Fatalf("disk save missing new room: %q", data)
	}

	restarted, err := parser.ParseWldFile(filepath.Join(dir, "wld", "1.wld"))
	if err != nil {
		t.Fatalf("restart parse failed: %v", err)
	}
	var found *parser.Room
	for i := range restarted {
		if restarted[i].VNum == 1050 {
			found = &restarted[i]
			break
		}
	}
	if found == nil || found.Name != "Inserted" || found.Description != "You are in an unfinished room.\n" || found.Zone != 1 {
		t.Fatalf("restarted room = %#v, want saved room 1050", found)
	}
}

func readMsgTextOrEmpty(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case msg := <-s.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatal(err)
		}
		var event EventData
		data, err := json.Marshal(sm.Data)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatal(err)
		}
		return event.Text
	default:
		return ""
	}
}
