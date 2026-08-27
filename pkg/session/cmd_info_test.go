package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
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

func TestScoreManaLabelMatchesCClassBranch(t *testing.T) {
	for class := game.ClassMageUser; class <= game.ClassMystic; class++ {
		want := "Mana"
		if class == game.ClassPsionic || class == game.ClassMystic {
			want = "Mind/Psi"
		}
		if got := scoreManaLabel(class); got != want {
			t.Errorf("class %d mana label = %q, want %q", class, got, want)
		}
	}
}

func TestCmdScoreFixedFixtureGolden(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scoretest", 1001, true)
	p := s.player
	p.Class = game.ClassWarrior
	p.Race = game.RaceHuman
	p.Stats.Str = 10
	p.Stats.Dex = 10
	p.Title = "the Warrior"
	p.Hometown = 1
	p.Level = 1
	p.Exp = 1
	p.Health = 22
	p.MaxHealth = 22
	p.Mana = 100
	p.MaxMana = 100
	p.Move = 85
	p.MaxMove = 85
	p.Alignment = 0
	p.AC = 100
	p.Gold = 0
	p.BankGold = 0
	p.Kills = 0
	p.PKs = 0
	p.Deaths = 0
	p.Position = game.PosStanding
	p.Hunger = 36
	p.Thirst = 36
	p.Drunk = 0
	p.Birth = time.Now().Unix()
	p.ConnectedAt = time.Now()
	p.PlayedDuration = 0

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	const want = "Scoretest                           Age: 17 years (It's your birthday today.)\r\n" +
		"Hit points: 22(22)  Mana points: 100(100)  Movement points: 85(85)\r\n" +
		"You are neutral, how boring.\r\n" +
		"You are naked, have you no shame?\r\n" +
		"Experience:    1 points\r\n" +
		"Coins carried: 0 gold coins    Coins in bank: 0 gold coins\r\n" +
		"Kills: 0  Pks: 0  Deaths: 0\r\n" +
		"You need 1499 exp to reach your next level.\r\n" +
		"You have been playing for 0 days and 0 hours.\r\n" +
		"You are a citizen of Kir Drax'in.\r\n" +
		"This ranks you as Scoretest the Warrior (level 1).\r\n" +
		"You are a Human Warrior.\r\n" +
		"Your pack is empty.\r\n" +
		"You are standing.\r\n"
	if got := readSessionText(t, s); got != want {
		t.Fatalf("score output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
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
			s.player.Stats.Int = 10
			s.player.Stats.Wis = 10
			registerInWorld(t, s)

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

	// With an empty help table, a topic lookup reports the C "No help
	// available" line. The invented overview/registry fallback is gone (R4).
	m.world.HelpTable = nil
	m.world.HelpScreen = ""
	err := cmdHelp(s, []string{"commands"})
	if err != nil {
		t.Fatalf("cmdHelp failed: %v", err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "No help available") {
		t.Errorf("empty-table help = %q; want \"No help available\" (invented overview removed)", got)
	}
}

func TestCmdWhere(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	err := cmdWhere(s, nil)
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

	err := cmdWho(s, nil)
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
