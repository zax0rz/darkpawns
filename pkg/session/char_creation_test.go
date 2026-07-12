package session

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
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
		t.Errorf("charHometown = %d, want 1 (Kiroshi)", s.charHometown)
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
		"M": "Minotauran",
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
	s.charSex = 0
	s.charHometown = 2 // Old City — room 8004 won't exist in test world; sendWelcome returns early
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

	if _, ok := m.GetSession("Tester"); !ok {
		t.Error("player should be registered in manager after char creation")
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
// TestNewCharFlowSkipsPasswordWhenSupplied (DP-909)
// ---------------------------------------------------------------------------

// TestNewCharFlowSkipsPasswordWhenSupplied confirms that when the auth layer
// already supplied a password, confirming the name jumps straight to the color
// stage — the password is collected exactly once, matching C (interpreter.c
// CON_NEWPASSWD/CON_CNFPASSWD is the single collection point).
func TestNewCharFlowSkipsPasswordWhenSupplied(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	// Auth layer supplied a password.
	s.startNewCharFlow("Hero", "supersecretpassword")
	if !s.charPasswordSupplied {
		t.Fatal("charPasswordSupplied should be true after startNewCharFlow with a password")
	}
	// Drain the confirm_name prompt.
	drainMsg(t, s)

	// Confirm the name.
	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage confirm Y: %v", err)
	}

	// Fantasy-name reminder text is sent before the next prompt.
	drainMsg(t, s)

	// The next prompt must be the color stage, NOT create_password — the
	// password was already collected by the auth layer, so no second prompt.
	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "color" {
		t.Errorf("after confirming name with a supplied password, stage = %q, want %q (no double password prompt)",
			cd.Stage, "color")
	}
	// The supplied password must now be hashed (bcrypt hashes start with $2).
	if len(s.charPassword) == 0 || s.charPassword[0] != '$' {
		t.Errorf("charPassword should be bcrypt-hashed after skip, got %q", s.charPassword)
	}
}

// TestNewCharFlowPromptsPasswordWhenNotSupplied confirms the nanny still
// collects the password itself when no password was pre-supplied (the pure-
// nanny flow C uses), so the skip is conditional, not unconditional.
func TestNewCharFlowPromptsPasswordWhenNotSupplied(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	s.startNewCharFlow("Hero", "") // no password from auth layer
	if s.charPasswordSupplied {
		t.Fatal("charPasswordSupplied should be false when no password supplied")
	}
	drainMsg(t, s) // confirm_name prompt

	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCharInput,
		Data: json.RawMessage(`{"choice":"Y"}`),
	})
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage confirm Y: %v", err)
	}

	// Fantasy-name reminder text is sent before the next prompt.
	drainMsg(t, s)

	_, cd := unmarshalCharCreate(t, drainMsg(t, s))
	if cd.Stage != "create_password" {
		t.Errorf("without a supplied password, stage = %q, want create_password", cd.Stage)
	}
}
