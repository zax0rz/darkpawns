package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCmdAdmobsLevelGate(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Admobslevel", 1001, true)
	s.player.Level = LVL_GRGOD

	if err := ExecuteCommand(s, "admobs", []string{"ignored"}); err != nil {
		t.Fatalf("ExecuteCommand admobs: %v", err)
	}
	if got, want := readSessionOutput(t, s), "Huh?!?\r\n"; got != want {
		t.Fatalf("level-gate output = %q, want %q", got, want)
	}
}

func readSessionOutput(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case msg := <-s.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var ed EventData
		data, _ := json.Marshal(sm.Data)
		_ = json.Unmarshal(data, &ed)
		return ed.Text
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for session output")
	}
	return ""
}

func TestCmdSethunt_NoArg(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = 31 // LVL_IMMORT
	registerInWorld(t, s)

	if err := cmdSethunt(s, nil); err != nil {
		t.Fatalf("cmdSethunt nil args: %v", err)
	}
	// C do_sethunt no-arg: "Who do you wish to hunt?\n\r"
	if got, want := readSessionOutput(t, s), "Who do you wish to hunt?\n\r"; got != want {
		t.Fatalf("no-arg output = %q, want %q", got, want)
	}
}

func TestCmdSethunt_HunterEqualsVictim(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = 31 // LVL_IMMORT
	registerInWorld(t, s)

	if err := cmdSethunt(s, []string{"Victim", "Victim"}); err != nil {
		t.Fatalf("cmdSethunt same name: %v", err)
	}
	// C: "Yeah right.\n\r"
	if got, want := readSessionOutput(t, s), "Yeah right.\n\r"; got != want {
		t.Fatalf("same-name output = %q, want %q", got, want)
	}
}

func TestCmdSethunt_MissingVictim(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = 31 // LVL_IMMORT
	registerInWorld(t, s)

	if err := cmdSethunt(s, []string{"Nobody", "HunterMob"}); err != nil {
		t.Fatalf("cmdSethunt missing victim: %v", err)
	}
	// C do_sethunt victim-miss: "No-one by that name around.\n\r"
	if got, want := readSessionOutput(t, s), "No-one by that name around.\n\r"; got != want {
		t.Fatalf("victim-miss output = %q, want %q", got, want)
	}
}

func TestCmdSethunt_MissingHunter(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = 31 // LVL_IMMORT
	registerInWorld(t, s)
	victim := makeTestSession(t, m, "Target", 1001, true)
	victim.player.Level = 1
	registerInWorld(t, victim)

	if err := cmdSethunt(s, []string{"Target", "NonExistentMob"}); err != nil {
		t.Fatalf("cmdSethunt missing hunter: %v", err)
	}
	// C do_sethunt hunter-miss: "Who shall be the hunter?\n\r"
	if got, want := readSessionOutput(t, s), "Who shall be the hunter?\n\r"; got != want {
		t.Fatalf("hunter-miss output = %q, want %q", got, want)
	}
}
