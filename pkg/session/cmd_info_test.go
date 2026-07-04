package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func readSessionText(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case msg := <-s.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		switch sm.Type {
		case MsgText:
			var td TextData
			data, _ := json.Marshal(sm.Data)
			_ = json.Unmarshal(data, &td)
			return td.Text
		case MsgEvent:
			var ed EventData
			data, _ := json.Marshal(sm.Data)
			_ = json.Unmarshal(data, &ed)
			if ed.Type == "text" {
				return ed.Text
			}
		}
		t.Fatalf("unexpected message type %q", sm.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for session message")
	}
	panic("unreachable")
}

func TestAlignmentText(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{1000, "Epitome of Righteousness"},
		{900, "angels jealous"},
		{600, "path of right"},
		{0, "boring"},
		{-600, "evil it hurts"},
		{-1000, "Epitome of Evil"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := alignmentText(tt.val)
			if !strings.Contains(got, tt.want) {
				t.Errorf("alignmentText(%d) = %q, expected substring %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestAcText(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{100, "naked"},
		{50, "well clothed"},
		{0, "well armored"},
		{-60, "battle armor"},
		{-140, "dragon"},
		{-200, "god"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := acText(tt.val)
			if !strings.Contains(got, tt.want) {
				t.Errorf("acText(%d) = %q, expected substring %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestPositionText(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{0, "DEAD"},
		{4, "sleeping"},
		{8, "standing"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := positionText(tt.val)
			if !strings.Contains(got, tt.want) {
				t.Errorf("positionText(%d) = %q, expected substring %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestArticleFor(t *testing.T) {
	tests := []struct {
		val  string
		want string
	}{
		{"Elf", "an"},
		{"Dwarf", "a"},
		{"Kender", "a"},
		{"Human", "a"},
		{"", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got := articleFor(tt.val)
			if got != tt.want {
				t.Errorf("articleFor(%q) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestCmdLevels(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := cmdLevels(s)
	if err != nil {
		t.Fatalf("cmdLevels failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "[ 1]") {
		t.Errorf("expected level info output, got %q", got)
	}
}

func TestCmdAbils(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Stats.Str = 15

	err := cmdAbils(s)
	if err != nil {
		t.Fatalf("cmdAbils failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(strings.ToLower(got), "ability scores") {
		t.Errorf("expected abils output, got %q", got)
	}
}

func TestCmdCoins(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Gold = 100
	s.player.BankGold = 500

	err := cmdCoins(s)
	if err != nil {
		t.Fatalf("cmdCoins failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "carrying 100 coins") {
		t.Errorf("expected coins output, got %q", got)
	}
}

func TestCmdScore(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = 5
	s.player.Exp = 1000

	err := cmdScore(s)
	if err != nil {
		t.Fatalf("cmdScore failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Hit points") {
		t.Errorf("expected score output, got %q", got)
	}
	// DP-913: armor line must not double the subject ("You You are well armored.").
	if strings.Contains(got, "You You") {
		t.Errorf("armor line doubles the subject, got %q", got)
	}
	if !strings.Contains(got, "You are ") {
		t.Errorf("expected single-subject armor line, got %q", got)
	}
}

func TestCmdSaySelfEcho(t *testing.T) {
	// DP-913: self-echo must conjugate in the second person ("You say"),
	// not third ("You says"). The room echo keeps third person.
	cases := []struct {
		name, text, wantSelf string
	}{
		{"plain", "hello", "You say 'hello'"},
		{"exclaim", "hi!", "You exclaim 'hi!'"},
		{"question", "what?", "You ask 'what?'"},
		{"statement", "ok.", "You state 'ok.'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeTestSession(t, m, "Alice", 1001, true)

			if err := cmdSay(s, []string{tc.text}); err != nil {
				t.Fatalf("cmdSay failed: %v", err)
			}
			got := readSessionText(t, s)
			if !strings.Contains(got, tc.wantSelf) {
				t.Errorf("self-echo = %q, want substring %q", got, tc.wantSelf)
			}
			if strings.Contains(got, "You says") {
				t.Errorf("self-echo still uses third-person verb: %q", got)
			}
		})
	}
}

func TestCmdHelp(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Test general help
	err := cmdHelp(s, nil)
	if err != nil {
		t.Fatalf("cmdHelp failed: %v", err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "Help Topics") {
		t.Errorf("expected help overview, got %q", got)
	}

	// Test help for a topic
	err = cmdHelp(s, []string{"commands"})
	if err != nil {
		t.Fatalf("cmdHelp failed: %v", err)
	}
	got = readSessionText(t, s)
	if !strings.Contains(got, "commands") {
		t.Errorf("expected topic help, got %q", got)
	}
}

func TestCmdWhere(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	err := cmdWhere(s)
	if err != nil {
		t.Fatalf("cmdWhere failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Room A") {
		t.Errorf("expected where output, got %q", got)
	}
}

func TestCmdWho(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	err := cmdWho(s)
	if err != nil {
		t.Fatalf("cmdWho failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") {
		t.Errorf("expected who output, got %q", got)
	}
}

func TestCmdSummon(t *testing.T) {
	m := makeTestManager(t)
	s1 := makeTestSession(t, m, "Alice", 1001, true)
	s2 := makeTestSession(t, m, "Bob", 1002, true)

	m.mu.Lock()
	m.sessions["alice"] = s1
	m.sessions["bob"] = s2
	m.mu.Unlock()

	err := cmdSummon(s1, []string{"Bob"})
	if err != nil {
		t.Fatalf("cmdSummon failed: %v", err)
	}

	got := readSessionText(t, s1)
	if !strings.Contains(got, "materializes before you") {
		t.Errorf("expected summon success output, got %q", got)
	}
}
