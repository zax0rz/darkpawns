package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestReditMenuColorsMatchGetCharCols(t *testing.T) {
	const colorMain = "\r\n-- Room number : [\x1b[36m1001\x1b[0m]      Room zone: [\x1b[36m1\x1b[0m]\r\n" +
		"\x1b[32m1\x1b[0m) Name        : \x1b[33mRoom A\r\n" +
		"\x1b[32m2\x1b[0m) Description :\r\n\x1b[33mA room.\r\n" +
		"\x1b[32m3\x1b[0m) Room flags  : \x1b[36mNOBITS \r\n" +
		"\x1b[32m4\x1b[0m) Sector type : \x1b[36mInside\r\n" +
		"\x1b[32m5\x1b[0m) Exit north  : \x1b[36m-1\r\n" +
		"\x1b[32m6\x1b[0m) Exit east   : \x1b[36m-1\r\n" +
		"\x1b[32m7\x1b[0m) Exit south  : \x1b[36m-1\r\n" +
		"\x1b[32m8\x1b[0m) Exit west   : \x1b[36m-1\r\n" +
		"\x1b[32m9\x1b[0m) Exit up     : \x1b[36m-1\r\n" +
		"\x1b[32mA\x1b[0m) Exit down   : \x1b[36m-1\r\n" +
		"\x1b[32mB\x1b[0m) Extra descriptions menu\r\n" +
		"\x1b[32mC\x1b[0m) Copy another room description\r\n" +
		"\x1b[32mS\x1b[0m) Script menu\r\n" +
		"\x1b[32mQ\x1b[0m) Quit\r\n" +
		"Enter choice : "
	const colorExit = "\r\n\x1b[32m1\x1b[0m) Exit to     : \x1b[36m1001\r\n" +
		"\x1b[32m2\x1b[0m) Description :-\r\n\x1b[33m<NONE>\r\n" +
		"\x1b[32m3\x1b[0m) Door name   : \x1b[33m<NONE>\r\n" +
		"\x1b[32m4\x1b[0m) Key         : \x1b[36m0\r\n" +
		"\x1b[32m5\x1b[0m) Door flags  : \x1b[36mNo door\r\n" +
		"\x1b[32m6\x1b[0m) Purge exit.\r\n" +
		"Enter choice, 0 to quit : "

	for _, tc := range []struct {
		name  string
		level int
	}{
		{"color-off", 0},
		{"color-normal", 2},
		{"color-complete", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newReditTestManager(t)
			s := makeCommandTestSession(t, m, "Colorgod", game.LVL_IMPL, 1001)
			switch tc.level {
			case 2:
				s.player.SetPlrFlag(game.PrfColor2, true)
			case 3:
				s.player.SetPlrFlag(game.PrfColor1, true)
				s.player.SetPlrFlag(game.PrfColor2, true)
			}
			if err := ExecuteCommand(s, "redit", nil); err != nil {
				t.Fatal(err)
			}
			wantMain := colorMain
			wantExit := colorExit
			if tc.level < 2 {
				wantMain = strings.ReplaceAll(strings.ReplaceAll(colorMain, "\x1b[36m", ""), "\x1b[0m", "")
				wantMain = strings.ReplaceAll(strings.ReplaceAll(wantMain, "\x1b[32m", ""), "\x1b[33m", "")
				wantExit = strings.ReplaceAll(strings.ReplaceAll(colorExit, "\x1b[36m", ""), "\x1b[0m", "")
				wantExit = strings.ReplaceAll(strings.ReplaceAll(wantExit, "\x1b[32m", ""), "\x1b[33m", "")
			}
			if got := readMsgText(t, s); got != wantMain {
				t.Fatalf("main menu = %q, want %q", got, wantMain)
			}
			s.handleReditInput("5")
			if got := readMsgText(t, s); got != wantExit {
				t.Fatalf("exit menu = %q, want %q", got, wantExit)
			}
			s.handleReditInput("0")
			_ = readMsgText(t, s)
			s.handleReditInput("3")
			got := readMsgText(t, s)
			if tc.level >= 2 {
				if !strings.HasPrefix(got, "\r\n\x1b[32m 1\x1b[0m) DARK                 \x1b[32m 2\x1b[0m) DEATH") {
					t.Fatalf("flag menu header = %q", got)
				}
				if !strings.HasSuffix(got, "\r\nRoom flags: \x1b[36mNOBITS \x1b[0m\r\nEnter room flags, 0 to quit : ") {
					t.Fatalf("flag menu footer = %q", got)
				}
			} else {
				if !strings.HasPrefix(got, "\r\n 1) DARK                  2) DEATH") {
					t.Fatalf("plain flag menu header = %q", got)
				}
			}
			s.handleReditInput("0")
			_ = readMsgText(t, s)
			s.handleReditInput("4")
			got = readMsgText(t, s)
			if tc.level >= 2 {
				if !strings.HasPrefix(got, "\r\n\x1b[32m 0\x1b[0m) Inside               \x1b[32m 1\x1b[0m) City") {
					t.Fatalf("sector menu header = %q", got)
				}
			} else if !strings.HasPrefix(got, "\r\n 0) Inside                1) City") {
				t.Fatalf("plain sector menu header = %q", got)
			}
		})
	}
}

func TestReditConcurrentEntryAdmitsExactlyOneEditor(t *testing.T) {
	m := newReditTestManager(t)
	const contenders = 8
	sessions := make([]*Session, contenders)
	for i := range sessions {
		sessions[i] = makeCommandTestSession(t, m, fmt.Sprintf("Racer%d", i), game.LVL_IMPL, 1001)
	}

	admitted := make([]bool, contenders)
	refused := make([]bool, contenders)
	var wg sync.WaitGroup
	for i := range sessions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := ExecuteCommand(sessions[i], "redit", nil); err != nil {
				t.Errorf("contender %d: %v", i, err)
				return
			}
			select {
			case msg := <-sessions[i].send:
				var sm ServerMessage
				if err := json.Unmarshal(msg, &sm); err != nil {
					t.Errorf("contender %d: %v", i, err)
					return
				}
				data, _ := json.Marshal(sm.Data)
				var event EventData
				if err := json.Unmarshal(data, &event); err != nil {
					t.Errorf("contender %d: %v", i, err)
					return
				}
				switch {
				case strings.Contains(event.Text, "-- Room number : [1001]"):
					admitted[i] = true
				case strings.HasPrefix(event.Text, "That room is currently being edited by Racer"):
					refused[i] = true
				default:
					t.Errorf("contender %d output = %q", i, event.Text)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("contender %d: no admission response", i)
			}
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, a := range admitted {
		if a {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent redit admitted %d editors, want exactly 1 (admitted=%v refused=%v)", winners, admitted, refused)
	}
	refusals := 0
	for _, r := range refused {
		if r {
			refusals++
		}
	}
	if refusals != contenders-1 {
		t.Fatalf("refusals = %d, want %d (refused=%v)", refusals, contenders-1, refused)
	}

	winner := 0
	for i, a := range admitted {
		if a {
			winner = i
			break
		}
	}
	// A disconnect must release the reservation: the dropped editor's cleanup
	// path (HandleTransportDisconnect → cancelRoomEdit) frees the room.
	sessions[winner].cancelRoomEdit()
	if err := ExecuteCommand(sessions[1], "redit", nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, sessions[1]); !strings.Contains(got, "-- Room number : [1001]") {
		t.Fatalf("editor not re-admitted after disconnect release: %q", got)
	}
	// A clean quit (unmodified room → immediate abort, no save prompt) must
	// release the reservation too.
	sessions[1].handleReditInput("q")
	if got := readMsgTextOrEmpty(t, sessions[1]); got != "" {
		t.Fatalf("clean quit output = %q, want empty", got)
	}
	if err := ExecuteCommand(sessions[2], "redit", nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, sessions[2]); !strings.Contains(got, "-- Room number : [1001]") {
		t.Fatalf("editor not re-admitted after clean quit: %q", got)
	}
}

// TestReditSaveSurvivesRealRestart boots a world from a disposable lib tree,
// edits an existing room through the full REDIT dialogue, saves the zone to
// disk, and then boots a FRESH world through the production loader
// (parser.ParseWorld + game.NewWorld, cmd/server/main.go's boot path) from
// the saved directory. A new session revisits the edited room: the proof
// covers the edited name, description, exit target/keywords/key/door flags,
// the untouched exit description, and the new extra description — not merely
// re-parsing the .wld file.
func TestReditSaveSurvivesRealRestart(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"wld", "zon", "mob", "obj", "shp"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	zoneData := "#1\nRestart Zone~\n1999 30 2\nS\n"
	if err := os.WriteFile(filepath.Join(dir, "zon", "1.zon"), []byte(zoneData), 0o644); err != nil {
		t.Fatal(err)
	}
	wldData := "#1001\nOriginal Room~\nThe original description.~\n1 0 0 0 0 0\n" +
		"D0\nThe original exit.~\ndoor~\n0 0 1002\nS\n" +
		"#1002\nRoom B~\nSparse.~\n1 0 0 0 0 0\nS\n" +
		"$~\n"
	if err := os.WriteFile(filepath.Join(dir, "wld", "1.wld"), []byte(wldData), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := parser.ParseWorld(dir)
	if err != nil {
		t.Fatalf("boot original world: %v", err)
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.WorldPath = dir
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Bootgod", game.LVL_IMPL, 1001)

	if err := ExecuteCommand(s, "redit", nil); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s) // main menu

	s.handleReditInput("1")
	_ = readMsgText(t, s) // name prompt
	s.handleReditInput("Restarted Room")
	_ = readMsgText(t, s) // main menu

	s.handleReditInput("2")
	_ = readMsgText(t, s) // description editor banner
	s.handleReditInput("/c")
	_ = readMsgText(t, s) // buffer cleared notice
	s.handleReditInput("You are in the restarted room.")
	s.handleReditInput("/s")
	_ = readMsgText(t, s) // main menu

	s.handleReditInput("5")
	_ = readMsgText(t, s) // exit menu
	s.handleReditInput("1")
	_ = readMsgText(t, s) // exit-number prompt
	s.handleReditInput("1002")
	_ = readMsgText(t, s) // exit menu
	s.handleReditInput("3")
	_ = readMsgText(t, s) // keywords prompt
	s.handleReditInput("gate")
	_ = readMsgText(t, s) // exit menu
	s.handleReditInput("4")
	_ = readMsgText(t, s) // key prompt
	s.handleReditInput("1101")
	_ = readMsgText(t, s) // exit menu
	s.handleReditInput("5")
	_ = readMsgText(t, s) // door flag menu
	s.handleReditInput("2")
	_ = readMsgText(t, s) // exit menu (pickproof)
	s.handleReditInput("0")
	_ = readMsgText(t, s) // main menu

	s.handleReditInput("B")
	_ = readMsgText(t, s) // extra-desc menu
	s.handleReditInput("1")
	_ = readMsgText(t, s) // keywords prompt
	s.handleReditInput("sign")
	_ = readMsgText(t, s) // extra-desc menu
	s.handleReditInput("2")
	_ = readMsgText(t, s) // extra description editor banner
	s.handleReditInput("A worn wooden sign.")
	s.handleReditInput("/s")
	_ = readMsgText(t, s) // extra-desc menu
	s.handleReditInput("0")
	_ = readMsgText(t, s) // main menu

	s.handleReditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save this room internally? : " {
		t.Fatalf("save prompt = %q", got)
	}
	s.handleReditInput("y")
	if got := readMsgText(t, s); got != "Room saved to memory.\r\n" {
		t.Fatalf("save confirmation = %q", got)
	}
	if err := ExecuteCommand(s, "redit", []string{"save", "1"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Saving all rooms in zone.\r\n" {
		t.Fatalf("disk-save banner = %q", got)
	}

	// The restart boundary: boot a fresh world through the production loader
	// from the same directory the zone save wrote.
	restartedParsed, err := parser.ParseWorld(dir)
	if err != nil {
		t.Fatalf("boot restarted world: %v", err)
	}
	restartedWorld, err := game.NewWorld(restartedParsed)
	if err != nil {
		t.Fatalf("NewWorld restarted: %v", err)
	}
	t.Cleanup(restartedWorld.StopAITicker)

	room, ok := restartedWorld.SnapshotRoom(1001)
	if !ok {
		t.Fatal("restarted world lost room 1001")
	}
	if room.Name != "Restarted Room" {
		t.Errorf("restarted name = %q", room.Name)
	}
	if room.Description != "You are in the restarted room.\n" {
		t.Errorf("restarted description = %q", room.Description)
	}
	exit, ok := room.Exits["north"]
	if !ok {
		t.Fatal("restarted room lost its north exit")
	}
	if exit.ToRoom != 1002 || exit.Key != 1101 || exit.Keywords != "gate" {
		t.Errorf("restarted exit = %+v", exit)
	}
	if exit.DoorState != 2 || exit.ExitInfo != parser.ExitIsDoor|parser.ExitPickproof {
		t.Errorf("restarted door flags = state %d info %d", exit.DoorState, exit.ExitInfo)
	}
	if exit.Description != "The original exit." {
		t.Errorf("untouched exit description = %q", exit.Description)
	}
	if len(room.ExtraDescs) != 1 || room.ExtraDescs[0].Keywords != "sign" || room.ExtraDescs[0].Description != "A worn wooden sign.\n" {
		t.Errorf("restarted extra descs = %+v", room.ExtraDescs)
	}

	// Revisit through a fresh session's room display, not just raw state.
	m2 := newTestManager(t, restartedWorld, nil)
	visitor := makeCommandTestSession(t, m2, "Revisitor", game.LVL_IMPL, 1001)
	if err := ExecuteCommand(visitor, "look", nil); err != nil {
		t.Fatal(err)
	}
	// A connected client's room display is the look state payload; assert the
	// edited room through the same fields the client renders.
	var state struct {
		Room struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"room"`
	}
	select {
	case msg := <-visitor.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatalf("look state message: %v", err)
		}
		data, err := json.Marshal(sm.Data)
		if err != nil {
			t.Fatalf("look state data: %v", err)
		}
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatalf("look state room: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no look state message from the restarted world")
	}
	if state.Room.Name != "Restarted Room" {
		t.Errorf("revisit display name = %q", state.Room.Name)
	}
	if strings.TrimSpace(state.Room.Description) != "You are in the restarted room." {
		t.Errorf("revisit display description = %q", state.Room.Description)
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
