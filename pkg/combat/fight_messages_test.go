package combat

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFightMessages(t *testing.T) {
	input := strings.Join([]string{
		"* comment between records",
		"",
		"M",
		" 300",
		"first die attacker", "first die victim", "first die room",
		"first miss attacker", "first miss victim", "first miss room",
		"#", "#", "#",
		"first god attacker", "first god victim", "first god room",
		"M",
		" 300",
		"second die attacker", "second die victim", "second die room",
		"second miss attacker", "second miss victim", "second miss room",
		"second hit attacker", "second hit victim", "second hit room",
		"second god attacker", "second god victim", "second god room",
		"$",
	}, "\n")

	messages, err := ParseFightMessages(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseFightMessages() error = %v", err)
	}

	variants, ok := messages.Variants(TYPE_HIT)
	if !ok {
		t.Fatalf("TYPE_HIT (%d) was not loaded", TYPE_HIT)
	}
	if got, want := len(variants), 2; got != want {
		t.Fatalf("len(TYPE_HIT variants) = %d, want %d", got, want)
	}
	if got, want := variants[0].Die.Attacker, "second die attacker"; got != want {
		t.Errorf("first C-selected variant = %q, want %q", got, want)
	}
	if got, want := variants[1].Die.Attacker, "first die attacker"; got != want {
		t.Errorf("second C-selected variant = %q, want %q", got, want)
	}
	if got := variants[1].Hit; got != (FightMessageAction{}) {
		t.Errorf("# sentinel parsed as %#v, want an empty action", got)
	}
	if got, want := variants[0].God.Room, "second god room"; got != want {
		t.Errorf("god room message = %q, want %q", got, want)
	}
}

func TestParseFightMessagesRejectsTruncatedRecord(t *testing.T) {
	_, err := ParseFightMessages(strings.NewReader("M\n300\nonly one action\n"))
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF reading action 2") {
		t.Fatalf("ParseFightMessages() error = %v, want truncated-action error", err)
	}
}

func TestCanonicalFightMessagesData(t *testing.T) {
	path := filepath.Join("..", "..", "lib", "misc", "messages")
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read canonical messages: %v", err)
	}
	if got, want := fmt.Sprintf("%x", sha256.Sum256(data)), "42e6fbb41fb61eab15c4ebeb1be8cd258ee8251e2c50630d733780952c5c42df"; got != want {
		t.Fatalf("canonical messages SHA-256 = %s, want %s", got, want)
	}

	messages, err := LoadFightMessages(path)
	if err != nil {
		t.Fatalf("LoadFightMessages() error = %v", err)
	}
	if got, want := len(messages), 66; got != want {
		t.Errorf("loaded attack types = %d, want %d", got, want)
	}
	variantCount := 0
	for _, variants := range messages {
		variantCount += len(variants)
	}
	if got, want := variantCount, 83; got != want {
		t.Errorf("loaded variants = %d, want %d", got, want)
	}

	hit, ok := messages.Variants(TYPE_HIT)
	if !ok {
		t.Fatalf("TYPE_HIT (%d) missing from canonical messages", TYPE_HIT)
	}
	if got, want := len(hit), 4; got != want {
		t.Fatalf("TYPE_HIT variants = %d, want %d", got, want)
	}
	if got, want := hit[0].Die.Attacker, "You hit $N with a blow that breaks $S spine!"; got != want {
		t.Errorf("first C-selected TYPE_HIT death message = %q, want %q", got, want)
	}
	if got, want := hit[1].Miss.Attacker, "You try to hit $N who easily avoids the blow."; got != want {
		t.Errorf("second C-selected TYPE_HIT miss message = %q, want %q", got, want)
	}
	if got, want := hit[3].Miss.Attacker, "You swing your fist at $N, but miss $M!"; got != want {
		t.Errorf("fourth C-selected TYPE_HIT miss message = %q, want %q", got, want)
	}
	if got := hit[0].Hit; got != (FightMessageAction{}) {
		t.Errorf("TYPE_HIT # hit messages = %#v, want empty action", got)
	}
}

func TestFightMessageSelectionConsumesOneDrawAndUsesActTokens(t *testing.T) {
	originalCallbacks := GetCallbacks()
	defer SetCallbacks(originalCallbacks)

	var attackerMessage string
	cb := &GameCallbacks{
		Broadcast: func(int, string, string) {},
		SendToChar: func(name, message string) {
			if name == "Attacker" {
				attackerMessage = message
			}
		},
		GetSex: func(name string) int {
			if name == "Victim" {
				return 1
			}
			return 0
		},
		GetHP:    func(string) int { return 10 },
		GetLevel: func(string) int { return 1 },
	}
	SetCallbacks(cb)
	InitFightMessages(cb, FightMessages{
		TYPE_HIT: {
			{Miss: FightMessageAction{Attacker: "first"}},
			{Miss: FightMessageAction{Attacker: "$e misses $N and cannot hit $M."}},
		},
	})

	roller := NewScriptedRoller([]int{2, 99})
	WithRoller(roller, func() {
		if handled := cb.SkillMessage(0, "Attacker", "Victim", TYPE_HIT, 100); !handled {
			t.Fatal("TYPE_HIT miss was not handled")
		}
	})

	if got, want := roller.Index, 1; got != want {
		t.Fatalf("fight-message selection draws = %d, want %d", got, want)
	}
	if got, want := attackerMessage, "He misses Victim and cannot hit her."; got != want {
		t.Fatalf("attacker message = %q, want %q", got, want)
	}
}

func TestDamMessageSeverityBoundariesConsumeNoDraws(t *testing.T) {
	originalCallbacks := GetCallbacks()
	defer SetCallbacks(originalCallbacks)

	var attackerMessage string
	cb := defaultCombatCallbacks()
	cb.SendToChar = func(name, message string) {
		if name == "Attacker" {
			attackerMessage = message
		}
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Attacker", room: 100, sex: 0, position: PosStanding}
	victim := &mockCombatant{name: "Victim", room: 100, sex: 1, position: PosStanding}
	tests := []struct {
		damage int
		want   string
	}{
		{0, "You try to hit Victim, but miss."},
		{2, "You scratch Victim as you hit her."},
		{3, "You barely hit Victim."},
		{4, "You barely hit Victim."},
		{5, "You hit Victim."},
		{6, "You hit Victim."},
		{7, "You hit Victim hard."},
		{10, "You hit Victim hard."},
		{11, "You hit Victim very hard."},
		{14, "You hit Victim very hard."},
		{15, "You hit Victim extremely hard."},
		{19, "You hit Victim extremely hard."},
		{20, "You massacre Victim to small fragments with your hit."},
		{23, "You massacre Victim to small fragments with your hit."},
		{24, "You OBLITERATE Victim with your deadly hit!!"},
		{33, "You OBLITERATE Victim with your deadly hit!!"},
		{34, "You EVISCERATE Victim with your incredible hit!!"},
		{43, "You EVISCERATE Victim with your incredible hit!!"},
		{44, "You DESTROY Victim with your ungodly hit!!"},
		{53, "You DESTROY Victim with your ungodly hit!!"},
		{54, "You ROCK THE HELL OUT OF Victim with your ultimate hit!!"},
	}

	roller := NewScriptedRoller([]int{7, 8, 9})
	WithRoller(roller, func() {
		for _, test := range tests {
			attackerMessage = ""
			DamMessage(test.damage, attacker, victim, 0)
			if attackerMessage != test.want {
				t.Errorf("DamMessage(%d) = %q, want %q", test.damage, attackerMessage, test.want)
			}
		}
	})
	if got := roller.Index; got != 0 {
		t.Fatalf("DamMessage consumed %d RNG draws, want 0", got)
	}
}
