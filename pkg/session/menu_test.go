package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

func sendMenuInput(t *testing.T, s *Session, choice string) {
	t.Helper()
	data, err := json.Marshal(CharInputData{Choice: choice})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleMenuInput(data); err != nil {
		t.Fatalf("handleMenuInput(%q): %v", choice, err)
	}
}

func TestNewCharacterMOTDTransitionsToMenu(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "motd"
	s.charName = "Menuhero"

	sendCharInput(t, s, "")

	if !s.menuActive || s.menuStage != "menu" {
		t.Fatalf("menu state = (%v, %q), want active menu", s.menuActive, s.menuStage)
	}
	if s.charCreating {
		t.Fatal("character creation input should yield to the menu")
	}
	if s.player != nil {
		t.Fatal("new player must not enter the world before choosing option 1")
	}
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Stage != "menu" || len(prompt.Options) != 6 {
		t.Fatalf("menu prompt = stage %q with %d options", prompt.Stage, len(prompt.Options))
	}
	if prompt.Prompt != menuText {
		t.Errorf("menu prompt = %q, want %q", prompt.Prompt, menuText)
	}
}

func TestMenuExitExactPrompt(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.showMainMenu()
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "0")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Prompt != "Goodbye.\r\n" {
		t.Errorf("exit prompt = %q", prompt.Prompt)
	}
}

func TestMenuInvalidChoiceAndBackgroundReturnToMenu(t *testing.T) {
	m := makeTestManager(t)
	textDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(textDir, "background"), []byte("The old kingdoms\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = textDir
	setTeditTestCache(t, "background", "")
	s := makeCharSession(t, m)
	s.showMainMenu()
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "x")
	text := string(drainMsg(t, s))
	if !strings.Contains(text, "not a menu choice") {
		t.Fatalf("invalid-choice response = %s", text)
	}
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "3")
	background := string(drainMsg(t, s))
	if !strings.Contains(background, "old kingdoms") {
		t.Fatalf("background response = %s", background)
	}
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Stage != "menu" {
		t.Fatalf("stage after background = %q, want menu", prompt.Stage)
	}
}

func TestMenuDescriptionEditorForNewCharacter(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charName = "Scribe"
	s.showMainMenu()
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "2")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "A weathered traveler.")
	sendMenuInput(t, s, "Eyes fixed on the horizon.")
	sendMenuInput(t, s, "/s")

	if got := s.menuDescription; got != "A weathered traveler.\r\nEyes fixed on the horizon." {
		t.Fatalf("description = %q", got)
	}
	_ = drainMsg(t, s)
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Stage != "menu" {
		t.Fatalf("stage after save = %q", prompt.Stage)
	}
}

func TestMenuPasswordChangeForNewCharacter(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charName = "Cipher"
	oldHash, err := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	s.charPassword = string(oldHash)
	s.showMainMenu()
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "4")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if !prompt.Secret {
		t.Fatal("old password prompt must request masked input")
	}
	sendMenuInput(t, s, "oldpass")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "newpass")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "newpass")

	if bcrypt.CompareHashAndPassword([]byte(s.charPassword), []byte("newpass")) != nil {
		t.Fatal("new password was not installed")
	}
	if bcrypt.CompareHashAndPassword([]byte(s.charPassword), []byte("oldpass")) == nil {
		t.Fatal("old password still matches")
	}
}

// TestReturningLoginMOTDSelectsByLevel pins C's login MOTD selection
// (src/interpreter.c:1919-1921): a returning immortal reads imotd, a mortal
// reads motd. The port sent motd to everyone, so a returning immortal read the
// wrong file's bytes — invisible to every existing scenario, because the
// corpus had no immortal relogin/restart vehicle until <RESTART>.
func TestReturningLoginMOTDSelectsByLevel(t *testing.T) {
	cases := []struct {
		name     string
		level    int
		wantFile string
		wantText string
	}{
		{name: "mortal", level: game.LVL_IMMORT - 1, wantFile: "motd", wantText: "mortal banner"},
		{name: "immortal", level: game.LVL_IMMORT, wantFile: "imotd", wantText: "immortal banner"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			textDir := t.TempDir()
			files := map[string]string{"motd": "mortal banner\n", "imotd": "immortal banner\n"}
			for file, text := range files {
				if err := os.WriteFile(filepath.Join(textDir, file), []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
				setTeditTestCache(t, file, "")
			}
			m.world.LibTextDir = textDir

			s := makeCharSession(t, m)
			s.player = game.NewPlayer(1, "MotdCase", game.MortalStartRoom)
			s.player.SetLevel(tc.level)
			if got := s.loginMOTDFile(); got != tc.wantFile {
				t.Fatalf("loginMOTDFile() at level %d = %q, want %q", tc.level, got, tc.wantFile)
			}

			s.startReturningMenu("")
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			if prompt.Stage != "motd" {
				t.Fatalf("returning prompt stage = %q, want motd", prompt.Stage)
			}
			if !strings.Contains(prompt.Prompt, tc.wantText) {
				t.Fatalf("%s login prompt = %q, want it to contain %q", tc.name, prompt.Prompt, tc.wantText)
			}
		})
	}
}

// TestReturningPlayerStopsAtMenuThenEntersWorld seeds a returning character and
// proves the menu holds it out of the world until option 1.
func TestReturningPlayerStopsAtMenuThenEntersWorld(t *testing.T) {
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{
		Name: "Returner", Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 1, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	s := makeCharSession(t, m)
	if err := s.handleLogin(loginMsg("Returner", "hunter2")); err != nil {
		t.Fatal(err)
	}

	if !s.menuActive || s.menuStage != "motd" {
		t.Fatalf("returning menu state = (%v, %q)", s.menuActive, s.menuStage)
	}
	if _, ok := m.GetSession("Returner"); ok {
		t.Fatal("returning player registered before choosing option 1")
	}
	if _, ok := world.GetPlayer("Returner"); ok {
		t.Fatal("returning player entered world before choosing option 1")
	}

	sendMenuInput(t, s, "")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "1")
	if s.menuActive {
		t.Fatal("menu still active after entering game")
	}
	if _, ok := m.GetSession("Returner"); !ok {
		t.Fatal("returning player was not registered")
	}
	if _, ok := world.GetPlayer("Returner"); !ok {
		t.Fatal("returning player was not added to world")
	}
	if got := s.player.GetRoom(); got != record.RoomVNum {
		t.Fatalf("returning player room = %d, want saved room %d", got, record.RoomVNum)
	}
}

func TestMenuDeleteReturningPlayer(t *testing.T) {
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{Name: "Doomed", Password: string(hash)}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatal(err)
	}
	s := makeCharSession(t, m)
	s.player = game.NewCharacter(record.ID, record.Name, game.ClassWarrior, game.RaceHuman)
	s.authenticated = true
	s.menuPasswordHash = string(hash)
	s.showMainMenu()
	_ = drainMsg(t, s)

	sendMenuInput(t, s, "5")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "hunter2")
	_ = drainMsg(t, s)
	sendMenuInput(t, s, "yes")

	if got, err := database.GetPlayer("Doomed"); err != nil || got != nil {
		t.Fatalf("deleted player = %#v, err %v", got, err)
	}
	if !s.SendClosed() {
		t.Fatal("delete should close the session")
	}
}
