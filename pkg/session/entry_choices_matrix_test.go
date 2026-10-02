package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// C src/interpreter.c:1988-2105,1674-1690; src/class.c:122-166.
func TestEntryCreationChoiceMatrix(t *testing.T) {
	for _, tt := range []struct{ stage, input, next string }{
		{"confirm_name", "Yes please", "create_password"},
		{"confirm_name", "No thanks", "get_name"},
		{"color", "yes", "sex"},
		{"color", "  No thanks", "sex"},
		{"sex", "male", "race"},
		{"sex", "female", "race"},
		{"race", "human", "class"},
		{"race", "elf", "class"},
		{"race", "dwarf", "class"},
		{"race", "kender", "class"},
		{"race", "minotaur", "class"},
		{"race", "rakshasa", "class"},
		{"race", "ssaur", "class"},
		{"hometown", "kir", "stats_roll"},
		{"hometown", "oshi", "stats_roll"},
		{"hometown", "alaozar", "stats_roll"},
	} {
		t.Run(tt.stage+tt.input, func(t *testing.T) {
			s := makeCharSession(t, makeTestManager(t))
			s.charCreating = true
			s.charStage = tt.stage
			s.charName = "Hero"
			sendCharInput(t, s, tt.input)
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			if s.charStage != tt.next || prompt.Stage != tt.next {
				t.Fatalf("%q: state=%q prompt=%+v", tt.input, s.charStage, prompt)
			}
			if tt.stage == "race" {
				expected := map[string]int{"human": game.RaceHuman, "elf": game.RaceElf, "dwarf": game.RaceDwarf, "kender": game.RaceKender, "minotaur": game.RaceMinotaur, "rakshasa": game.RaceRakshasa, "ssaur": game.RaceSsaur}[tt.input]
				if s.charRace != expected {
					t.Fatalf("race=%d want=%d", s.charRace, expected)
				}
			}
			if tt.stage == "color" && s.charColor != (strings.TrimSpace(tt.input)[0] == 'y') {
				t.Fatal("wrong color flag")
			}
		})
	}
	for _, race := range []int{game.RaceHuman, game.RaceElf, game.RaceDwarf, game.RaceKender, game.RaceMinotaur, game.RaceRakshasa, game.RaceSsaur} {
		for _, class := range []struct {
			input string
			id    int
		}{{"mage", game.ClassMageUser}, {"cleric", game.ClassCleric}, {"thief", game.ClassThief}, {"warrior", game.ClassWarrior}, {"ionic", game.ClassPsionic}, {"ninja", game.ClassNinja}, {"avatar", -1}, {"paladin", -1}, {"ss", -1}, {"ranger", -1}, {"y", -1}, {"", -1}} {
			s := makeCharSession(t, makeTestManager(t))
			s.charCreating = true
			s.charStage = "class"
			s.charRace = race
			sendCharInput(t, s, " "+class.input)
			_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
			valid := class.id >= 0 && (class.id != game.ClassNinja || race == game.RaceHuman)
			if valid {
				if s.charStage != "hometown" || s.charClass != class.id {
					t.Fatalf("race %d class %s: %+v", race, class.input, prompt)
				}
			} else if s.charStage != "class" || prompt.Prompt != "\r\nThat's not a class.\r\nClass: " {
				t.Fatalf("invalid race %d class %s: %+v", race, class.input, prompt)
			}
		}
	}
}

func TestEntryRaceHelpMatrix(t *testing.T) {
	for _, input := range []string{"?", "?h suffix", "?E", "?d", "?k", "?m", "?r", "?s", "?x", "? h"} {
		s := makeCharSession(t, makeTestManager(t))
		s.charCreating = true
		s.charStage = "race"
		sendCharInput(t, s, "  "+input)
		_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
		if s.charStage != "race" || !strings.HasSuffix(prompt.Prompt, RaceMenuText+"\r\nRace: ") {
			t.Fatalf("help %q: %+v", input, prompt)
		}
		if input == "?x" || input == "? h" {
			if !strings.HasPrefix(prompt.Prompt, "That is not a race..\r\n") {
				t.Fatalf("invalid help: %q", prompt.Prompt)
			}
		}
	}
}

func TestEntryCreationChoicesWebSocket(t *testing.T) {
	m := entryTransportManager(t, entryDatabase(t))
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer server.Close()
	conn := entryWebSocketDial(t, server.URL)
	defer func() { _ = conn.Close() }()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": "Hero"})
	_ = wsReadUntilType(t, conn, MsgCharCreate)
	for _, step := range []struct{ input, stage string }{{"yes please", "create_password"}, {"abc", "confirm_password"}, {"abc", "color"}, {"no thanks", "sex"}, {"female", "race"}, {"?h suffix", "race"}, {"human", "class"}, {"avatar", "class"}, {"ninja", "hometown"}, {"oshi", "stats_roll"}} {
		wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": step.input})
		var prompt CharCreateData
		entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
		if prompt.Stage != step.stage {
			t.Fatalf("input %q: %+v", step.input, prompt)
		}
		if step.input == "avatar" && prompt.Prompt != "\r\nThat's not a class.\r\nClass: " {
			t.Fatalf("remort accepted: %+v", prompt)
		}
	}
}
