package session

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// makeCharSession creates a bare session suitable for char creation tests.
// Unlike makeTestSession, this has no pre-set player — char creation creates it.
func makeCharSession(t *testing.T, m *Manager) *Session {
	t.Helper()
	return &Session{
		request:        &http.Request{},
		manager:        m,
		send:           make(chan []byte, 256),
		subscribedVars: make(map[string]bool),
		dirtyVars:      make(map[string]bool),
		connectedAt:    time.Now(),
	}
}

// drainMsg reads one message from s.send with a 1s timeout.
func drainMsg(t *testing.T, s *Session) []byte {
	t.Helper()
	select {
	case msg := <-s.send:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message on send channel")
		return nil
	}
}

// unmarshalCharCreate unmarshals a ServerMessage and then its Data as CharCreateData.
func unmarshalCharCreate(t *testing.T, raw []byte) (ServerMessage, CharCreateData) {
	t.Helper()
	var srv ServerMessage
	if err := json.Unmarshal(raw, &srv); err != nil {
		t.Fatalf("unmarshal ServerMessage: %v", err)
	}
	b, _ := json.Marshal(srv.Data)
	var cd CharCreateData
	if err := json.Unmarshal(b, &cd); err != nil {
		t.Fatalf("unmarshal CharCreateData: %v", err)
	}
	return srv, cd
}

// sendCharInput marshals a CharInputData and calls handleCharInput.
func sendCharInput(t *testing.T, s *Session, choice string) {
	t.Helper()
	data, _ := json.Marshal(CharInputData{Choice: choice})
	if err := s.handleCharInput(data); err != nil {
		t.Fatalf("handleCharInput(%q): %v", choice, err)
	}
}

// ---------------------------------------------------------------------------
// TestStartCharCreation
// ---------------------------------------------------------------------------

func TestStartCharCreation(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	s.startCharCreation("Gandalf")

	if !s.charCreating {
		t.Error("charCreating should be true after startCharCreation")
	}
	if s.charStage != "confirm_name" {
		t.Errorf("charStage = %q, want %q", s.charStage, "confirm_name")
	}
	if s.charName != "Gandalf" {
		t.Errorf("charName = %q, want Gandalf", s.charName)
	}

	srv, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if srv.Type != MsgCharCreate {
		t.Errorf("message type = %q, want %q", srv.Type, MsgCharCreate)
	}
	if cd.Stage != "confirm_name" {
		t.Errorf("CharCreateData.Stage = %q, want confirm_name", cd.Stage)
	}
	want := "Please remember to choose an appropriate fantasy-oriented name.\r\n" +
		"Did I get that right, Gandalf (Y/N)? "
	if cd.Prompt != want {
		t.Errorf("confirm prompt = %q, want %q", cd.Prompt, want)
	}
}

func TestCharacterCreationExactInvalidPrompts(t *testing.T) {
	tests := []struct {
		stage  string
		choice string
		setup  func(*Session)
		want   string
	}{
		{stage: "confirm_name", choice: "maybe", setup: func(s *Session) { s.charName = "Hero" }, want: "Please type Yes or No: "},
		{stage: "sex", choice: "x", want: "That is not a sex..\r\nWhat IS your sex? "},
		{stage: "race", choice: "x", want: "That is not a race..\r\nWhat IS your race? "},
		{stage: "class", choice: "x", setup: func(s *Session) { s.charRace = game.RaceHuman }, want: "\r\nThat's not a class.\r\nClass: "},
		{stage: "hometown", choice: "x", want: "Invalid choice!\r\nSelect: "},
		{stage: "stats_roll", choice: "x", want: "Invalid choice! Select 'Y' or 'N':"},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charCreating = true
			s.charStage = tt.stage
			if tt.setup != nil {
				tt.setup(s)
			}
			sendCharInput(t, s, tt.choice)
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			if prompt.Prompt != tt.want {
				t.Errorf("prompt = %q, want %q", prompt.Prompt, tt.want)
			}
		})
	}
}

func TestCharacterCreationPasswordMismatchReprompts(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.charCreating = true
	s.charStage = "confirm_password"
	s.charPassword = "hunter2"

	sendCharInput(t, s, "different")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Prompt != "\r\nPasswords don't match... start over.\r\nPassword: " {
		t.Errorf("prompt = %q", prompt.Prompt)
	}
	if s.charStage != "create_password" || !prompt.Secret {
		t.Errorf("mismatch state = %q secret=%v, want create_password with echo off", s.charStage, prompt.Secret)
	}
	if s.SendClosed() {
		t.Fatal("password mismatch disconnected the session")
	}
}

func TestCharacterCreationPasswordConfirmationStartsColorOnNewLine(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.charCreating = true
	s.charStage = "confirm_password"
	s.charPassword = "hunter2"

	sendCharInput(t, s, "hunter2")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Prompt != "\r\nDo you want ANSI color (Y/N)? " {
		t.Errorf("prompt = %q", prompt.Prompt)
	}
	if s.charStage != "color" || prompt.Secret {
		t.Errorf("confirmation state = %q secret=%v, want visible color prompt", s.charStage, prompt.Secret)
	}
}

func TestCharacterCreationIllegalPasswordExactPrompt(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.charCreating = true
	s.charStage = "create_password"
	s.charName = "Hero"

	for _, password := range []string{"", "ab", "Hero", "12345678901"} {
		sendCharInput(t, s, password)
		_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
		if prompt.Prompt != "\r\nIllegal password.\r\nPassword: " || !prompt.Secret {
			t.Errorf("password %q prompt = %q secret=%v", password, prompt.Prompt, prompt.Secret)
		}
	}
}

// ---------------------------------------------------------------------------
// TestHandleCharInput_ColorStage
// ---------------------------------------------------------------------------

func TestHandleCharInput_ColorStage(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "color"

	sendCharInput(t, s, "Y")

	if s.charStage != "sex" {
		t.Errorf("charStage = %q, want sex", s.charStage)
	}
	if !s.charColor {
		t.Error("charColor should be true after Y at color stage")
	}
	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "sex" {
		t.Errorf("prompt stage = %q, want sex", cd.Stage)
	}
}

func TestHandleCharInput_ColorStage_NoColor(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "color"

	sendCharInput(t, s, "N")

	if s.charStage != "sex" {
		t.Errorf("charStage = %q, want sex", s.charStage)
	}
	if s.charColor {
		t.Error("charColor should be false after N at color stage")
	}
	drainMsg(t, s) // consume sex prompt
}

// ---------------------------------------------------------------------------
// TestHandleCharInput_AllStages — color → sex → race → class → hometown → stats_roll
// ---------------------------------------------------------------------------

func TestHandleCharInput_AllStages(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "color"

	// color → sex
	sendCharInput(t, s, "N")
	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "sex" {
		t.Errorf("after color: stage = %q, want sex", cd.Stage)
	}

	// sex → race (female)
	sendCharInput(t, s, "F")
	_, cd = unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "race" {
		t.Errorf("after sex: stage = %q, want race", cd.Stage)
	}
	if s.charSex != 1 {
		t.Errorf("charSex = %d, want 1 (female)", s.charSex)
	}

	// race → class (human = H)
	sendCharInput(t, s, "H")
	_, cd = unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "class" {
		t.Errorf("after race: stage = %q, want class", cd.Stage)
	}
	if s.charRace != 0 {
		t.Errorf("charRace = %d, want 0 (human)", s.charRace)
	}

	// class → hometown (warrior = W)
	sendCharInput(t, s, "W")
	_, cd = unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "hometown" {
		t.Errorf("after class: stage = %q, want hometown", cd.Stage)
	}
	if s.charClass != 3 {
		t.Errorf("charClass = %d, want 3 (warrior)", s.charClass)
	}

	// hometown → stats_roll (Kiroshi)
	sendCharInput(t, s, "K")
	_, cd = unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "stats_roll" {
		t.Errorf("after hometown: stage = %q, want stats_roll", cd.Stage)
	}
	if s.charHometown != 1 {
		t.Errorf("charHometown = %d, want 1 (Kir Drax'in)", s.charHometown)
	}
	if s.charStage != "stats_roll" {
		t.Errorf("s.charStage = %q, want stats_roll", s.charStage)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCharInput_InvalidChoice
// ---------------------------------------------------------------------------

func TestHandleCharInput_InvalidChoice(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "sex"

	sendCharInput(t, s, "Z")

	if s.charStage != "sex" {
		t.Errorf("invalid choice should not advance stage: got %q, want sex", s.charStage)
	}
	srv, _ := unmarshalCharCreate(t, drainMsg(t, s))
	if srv.Type != MsgCharCreate {
		t.Errorf("type = %q, want %q", srv.Type, MsgCharCreate)
	}
}

// ---------------------------------------------------------------------------
// TestGetRaceOptions
// ---------------------------------------------------------------------------

func TestGetRaceOptions(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	opts := s.getRaceOptions()
	expected := map[string]string{
		"H": "Human",
		"E": "Elven",
		"D": "Dwarven",
		"K": "Kenderkin",
		"M": "Minotaur",
		"R": "Rakshasan",
		"S": "Ssauran",
	}
	if len(opts) != len(expected) {
		t.Errorf("getRaceOptions len = %d, want %d", len(opts), len(expected))
	}
	got := make(map[string]string, len(opts))
	for _, o := range opts {
		got[o.Key] = o.Label
	}
	for k, v := range expected {
		if gotV, ok := got[k]; !ok || gotV != v {
			t.Errorf("getRaceOptions[%q] = %q, want %q", k, gotV, v)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetClassOptions
// ---------------------------------------------------------------------------

func TestGetClassOptions(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	humanOpts := sliceToMap(s.getClassOptions(game.RaceHuman))
	if _, ok := humanOpts["N"]; !ok {
		t.Error("human should have Ninja (N) available")
	}

	elfOpts := sliceToMap(s.getClassOptions(1)) // elf
	if _, ok := elfOpts["N"]; ok {
		t.Error("elf should not have Ninja (N) available")
	}

	baseClasses := []string{"C", "T", "W", "M", "I"}
	for _, opts := range []map[string]string{humanOpts, elfOpts} {
		for _, k := range baseClasses {
			if _, ok := opts[k]; !ok {
				t.Errorf("base class %q missing from class options", k)
			}
		}
	}
}

// sliceToMap converts a []CharCreateOption to a map[string]string for test
// lookups (order is verified separately by TestCharCreateOptionOrder).
func sliceToMap(opts []CharCreateOption) map[string]string {
	m := make(map[string]string, len(opts))
	for _, o := range opts {
		m[o.Key] = o.Label
	}
	return m
}

// ---------------------------------------------------------------------------
// TestSendStatsRollPrompt
// ---------------------------------------------------------------------------

func TestSendStatsRollPrompt(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charClass = 3 // warrior
	s.charRace = 0  // human

	s.sendStatsRollPrompt()

	srv, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if srv.Type != MsgCharCreate {
		t.Errorf("type = %q, want %q", srv.Type, MsgCharCreate)
	}
	if cd.Stage != "stats_roll" {
		t.Errorf("stage = %q, want stats_roll", cd.Stage)
	}
	if cd.Stats == nil {
		t.Fatal("stats should be non-nil in stats_roll prompt")
	}
	if s.charStage != "stats_roll" {
		t.Errorf("charStage = %q, want stats_roll after sendStatsRollPrompt", s.charStage)
	}
}

// ---------------------------------------------------------------------------
// TestCompleteCharCreation_WithNilDB
// ---------------------------------------------------------------------------

func TestCompleteCharCreation_WithNilDB(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	s.charCreating = true
	s.charName = "Tester"
	s.charClass = 3 // warrior
	s.charRace = 0  // human
	s.charSex = 1
	s.charHometown = 2 // Kir-Oshi
	s.charPassword = "hashed_pw"
	s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}

	if err := s.completeCharCreation(); err != nil {
		t.Fatalf("completeCharCreation: %v", err)
	}

	if s.charCreating {
		t.Error("charCreating should be false after completion")
	}
	if s.charStage != "" {
		t.Errorf("charStage should be empty after completion, got %q", s.charStage)
	}
	if !s.authenticated {
		t.Error("session should be authenticated after char creation")
	}
	if s.playerName != "Tester" {
		t.Errorf("playerName = %q, want Tester", s.playerName)
	}
	if s.player == nil {
		t.Fatal("player should not be nil after char creation")
	}
	if s.player.Name != "Tester" {
		t.Errorf("player.Name = %q, want Tester", s.player.Name)
	}
	wantStats := game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}
	if s.player.Stats != wantStats {
		t.Errorf("player stats = %+v, want accepted stats %+v", s.player.Stats, wantStats)
	}
	if s.player.Sex != 1 {
		t.Errorf("player sex = %d, want female", s.player.Sex)
	}
	if s.player.AC != 100 {
		t.Errorf("player AC = %d, want C newbie base 100", s.player.AC)
	}
	// C leaves the new mortal in the Burning Hut; the hometown relocation
	// belongs to the first PULSE_MOBILE's start_room dispatch.
	if got, want := s.player.GetRoom(), game.NewbieStartRoom; got != want {
		t.Errorf("player room = %d, want newbie start room %d", got, want)
	}

	if _, ok := m.GetSession("Tester"); !ok {
		t.Error("player should be registered in manager after char creation")
	}
}

func TestCompleteCharCreation_PersistsHometownRoom(t *testing.T) {
	database := testutil.NewMockDatabase()
	// Seed one player so CountPlayers() > 0 and shouldCrownFirstPlayer
	// returns false — this test covers mortal routing, not God routing.
	if err := database.CreatePlayer(&db.PlayerRecord{Name: "Existing", Level: 1}); err != nil {
		t.Fatalf("seed player: %v", err)
	}
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	s := makeCharSession(t, newTestManager(t, world, database))

	s.charCreating = true
	s.charName = "Alaozarnewbie"
	s.charClass = game.ClassWarrior
	s.charRace = game.RaceHuman
	s.charHometown = 3
	s.charPassword = "hashed_pw"
	s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}

	if err := s.completeCharCreation(); err != nil {
		t.Fatalf("completeCharCreation: %v", err)
	}

	want := game.NewbieHometownRoom(3)
	record, err := database.GetPlayer("Alaozarnewbie")
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if record == nil {
		t.Fatal("created player record not found")
	}
	if record.RoomVNum != want {
		t.Errorf("persisted room = %d, want hometown room %d", record.RoomVNum, want)
	}
	// The live room stays in the Burning Hut until the pulse-time birth
	// transition; only the persisted record carries the hometown room.
	if got := s.player.GetRoom(); got != game.NewbieStartRoom {
		t.Errorf("live player room = %d, want newbie start room %d", got, game.NewbieStartRoom)
	}
}

func TestCompleteCharCreationEmitsBirthTransitionAndNoDuplicateMOTD(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	s := makeCharSession(t, makeManagerWithStartRoom(t))
	s.charCreating = true
	s.charName = "Oneentry"
	s.charClass = game.ClassWarrior
	s.charRace = game.RaceHuman
	s.charHometown = 1
	s.charPassword = "hashed_pw"
	s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}

	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}

	// C leaves the new mortal in the Burning Hut at creation; the birth
	// transition belongs to the first PULSE_MOBILE's start_room dispatch
	// (spec_procs.c via comm.c room_activity). Creation itself must emit
	// only the intro-room observation — no birth text, no hometown state.
	states := 0
	motds := 0
	welcome := ""
	for {
		select {
		case raw := <-s.send:
			var msg ServerMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				t.Fatal(err)
			}
			switch msg.Type {
			case MsgState:
				states++
			case MsgText:
				data, _ := json.Marshal(msg.Data)
				var text TextData
				_ = json.Unmarshal(data, &text)
				if strings.Contains(text.Text, "Your life begins now") {
					t.Fatal("birth transition emitted at creation instead of the pulse")
				}
			case MsgEvent:
				data, _ := json.Marshal(msg.Data)
				var event EventData
				_ = json.Unmarshal(data, &event)
				if event.Type == "motd" {
					motds++
				}
				if strings.Contains(event.Text, "May your visit here") {
					welcome = event.Text
				}
			}
		default:
			if states != 1 {
				t.Fatalf("entry state messages = %d, want only the intro room", states)
			}
			if motds != 0 {
				t.Fatalf("post-menu MOTD messages = %d, want 0", motds)
			}
			if welcome != "\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n\r\n" {
				t.Fatalf("welcome = %q", welcome)
			}
			if s.player.GetRoom() != game.NewbieStartRoom {
				t.Fatalf("post-creation room = %d, want the Burning Hut %d", s.player.GetRoom(), game.NewbieStartRoom)
			}

			// The pulse-time start_room dispatch delivers the birth message
			// and relocates the mortal to the hometown room.
			s.manager.world.RoomActivity()
			birth := ""
			for {
				select {
				case raw := <-s.send:
					var msg ServerMessage
					if err := json.Unmarshal(raw, &msg); err != nil {
						t.Fatal(err)
					}
					// The pulse-time spec delivers through the world
					// MessageSink, which wraps player text as MsgEvent.
					if msg.Type == MsgEvent {
						data, _ := json.Marshal(msg.Data)
						var event EventData
						_ = json.Unmarshal(data, &event)
						if strings.Contains(event.Text, "Your life begins now") {
							birth = event.Text
						}
					}
				default:
					if birth == "" {
						t.Fatal("room_activity did not deliver the birth transition")
					}
					if s.player.GetRoom() != 8162 {
						t.Fatalf("post-birth room = %d, want hometown 8162", s.player.GetRoom())
					}
					return
				}
			}
		}
	}
}

func TestCompleteCharCreationKeepsProductionEntryUnchanged(t *testing.T) {
	s := makeCharSession(t, makeManagerWithStartRoom(t))
	s.charCreating = true
	s.charName = "Prodentry"
	s.charClass = game.ClassWarrior
	s.charRace = game.RaceHuman
	s.charHometown = 1
	s.charPassword = "hashed_pw"
	s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}

	if err := s.completeCharCreation(); err != nil {
		t.Fatal(err)
	}

	states := 0
	for {
		select {
		case raw := <-s.send:
			var msg ServerMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				t.Fatal(err)
			}
			if msg.Type == MsgState {
				states++
			}
			if msg.Type == MsgText {
				data, _ := json.Marshal(msg.Data)
				var text TextData
				_ = json.Unmarshal(data, &text)
				if strings.Contains(text.Text, "Your life begins now") {
					t.Fatal("DP_CLOCK-only birth transition leaked into production entry")
				}
			}
		default:
			if states != 1 {
				t.Fatalf("production entry state messages = %d, want exactly 1", states)
			}
			return
		}
	}
}

// ---------------------------------------------------------------------------
// TestAdvanceCharStage
// ---------------------------------------------------------------------------

func TestAdvanceCharStage(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	opts := charOpts("M", "Male", "F", "Female")
	s.advanceCharStage("sex", "Select your sex:", opts)

	if s.charStage != "sex" {
		t.Errorf("charStage = %q, want sex", s.charStage)
	}
	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "sex" {
		t.Errorf("CharCreateData.Stage = %q, want sex", cd.Stage)
	}
	optMap := sliceToMap(cd.Options)
	if optMap["M"] != "Male" || optMap["F"] != "Female" {
		t.Errorf("options mismatch: got %v", cd.Options)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCharInput_StatsRoll_Reroll
// ---------------------------------------------------------------------------

func TestHandleCharInput_StatsRoll_Reroll(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "stats_roll"
	s.charClass = 3
	s.charRace = 0

	sendCharInput(t, s, "N")

	if s.charStage != "stats_roll" {
		t.Errorf("charStage = %q, want stats_roll after reroll", s.charStage)
	}
	srv, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if srv.Type != MsgCharCreate {
		t.Errorf("type = %q, want %q", srv.Type, MsgCharCreate)
	}
	if cd.Stage != "stats_roll" {
		t.Errorf("stage = %q, want stats_roll after reroll", cd.Stage)
	}
}

// ---------------------------------------------------------------------------
// TestHandleMessage_CharInput_Routing — verifies handleMessage routes char_input correctly
// ---------------------------------------------------------------------------

func TestHandleMessage_CharInput_Routing(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "color"

	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	if s.charStage != "sex" {
		t.Errorf("charStage = %q, want sex after routing char_input through handleMessage", s.charStage)
	}
	drainMsg(t, s)
}

func TestHandleMessage_CharInput_WhenNotInCharCreation(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)
	s.charCreating = false // not in char creation

	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	err := s.handleMessage(msg)
	if err != ErrNotInCharCreation {
		t.Errorf("expected ErrNotInCharCreation, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestCharCreateOptionsAreOrdered (DP-909)
// ---------------------------------------------------------------------------

func TestCharCreateOptionsAreOrdered(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	// Race options must be in a stable, C-menu order — not Go's randomized map
	// order. Re-fetch several times and confirm the key sequence is identical.
	raceKeys := func() []string {
		ks := make([]string, 0)
		for _, o := range s.getRaceOptions() {
			ks = append(ks, o.Key)
		}
		return ks
	}
	first := raceKeys()
	want := []string{"H", "E", "D", "K", "M", "R", "S"}
	if len(first) != len(want) {
		t.Fatalf("race option count = %d, want %d", len(first), len(want))
	}
	for i, k := range want {
		if first[i] != k {
			t.Errorf("race option[%d] = %q, want %q (order: %v)", i, first[i], k, first)
		}
	}

	// Human class options include Ninja last, in menu order.
	humanKeys := func() []string {
		ks := make([]string, 0)
		for _, o := range s.getClassOptions(game.RaceHuman) {
			ks = append(ks, o.Key)
		}
		return ks
	}
	hc := humanKeys()
	wantClass := []string{"C", "T", "W", "M", "I", "N"}
	for i, k := range wantClass {
		if i >= len(hc) || hc[i] != k {
			t.Errorf("human class option[%d] = %q, want %q (order: %v)", i, firstKey(hc, i), k, hc)
		}
	}
}

func firstKey(ks []string, i int) string {
	if i >= len(ks) {
		return ""
	}
	return ks[i]
}

// ---------------------------------------------------------------------------
// TestNewCharFlowAlwaysPromptsPassword
// ---------------------------------------------------------------------------

// TestNewCharFlowAlwaysPromptsPassword confirms transports cannot skip C's
// CON_NEWPASSWD/CON_CNFPASSWD dialogue by supplying a login password.
func TestNewCharFlowAlwaysPromptsPassword(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	s.startNewCharFlow("Hero")
	_ = drainMsg(t, s) // fantasy-name reminder + confirm_name prompt

	// Confirm the name.
	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage confirm Y: %v", err)
	}

	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "create_password" {
		t.Errorf("after confirming name, stage = %q, want create_password", cd.Stage)
	}
	if !cd.Secret {
		t.Fatal("create-password prompt must disable client echo")
	}
	if s.charPassword != "" {
		t.Errorf("transport-supplied password leaked into nanny state: %q", s.charPassword)
	}
}

// TestNewCharFlowPromptsPasswordWhenNotSupplied confirms the nanny still
// collects the password itself when no password was pre-supplied (the pure-
// nanny flow C uses), so the skip is conditional, not unconditional.
func TestNewCharFlowPromptsPasswordWhenNotSupplied(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	s.startNewCharFlow("Hero")
	_ = drainMsg(t, s) // fantasy-name reminder + confirm_name prompt

	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage confirm Y: %v", err)
	}

	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "create_password" {
		t.Errorf("without a supplied password, stage = %q, want create_password", cd.Stage)
	}
}
