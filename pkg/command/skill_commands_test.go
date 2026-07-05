package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type rescueCommandSession struct {
	player       *game.Player
	world        *game.World
	combatEngine interface{}
	messages     []string
	tempData     map[string]interface{}
}

func (s *rescueCommandSession) GetPlayer() *game.Player { return s.player }

func (s *rescueCommandSession) SendMessage(message string) error {
	s.messages = append(s.messages, message)
	return nil
}

func (s *rescueCommandSession) Send(message string)          { s.messages = append(s.messages, message) }
func (s *rescueCommandSession) MarkDirty(vars ...string)     {}
func (s *rescueCommandSession) GetManager() interface{}      { return nil }
func (s *rescueCommandSession) GetWorld() *game.World        { return s.world }
func (s *rescueCommandSession) GetCombatEngine() interface{} { return s.combatEngine }
func (s *rescueCommandSession) RandomInt(maxValue int) int   { return 0 }
func (s *rescueCommandSession) SetTempData(key string, value interface{}) {
	if s.tempData == nil {
		s.tempData = make(map[string]interface{})
	}
	s.tempData[key] = value
}

func (s *rescueCommandSession) GetTempData(key string) interface{} {
	if s.tempData == nil {
		return nil
	}
	return s.tempData[key]
}

func (s *rescueCommandSession) ClearTempData(key string) {
	if s.tempData != nil {
		delete(s.tempData, key)
	}
}

func newRescueCommandSession(t *testing.T, combatEngine interface{}) *rescueCommandSession {
	t.Helper()

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	rescuer := game.NewPlayer(1, "Rescuer", 1001)
	rescuer.Class = game.ClassWarrior
	rescuer.SetLevel(4)
	rescuer.SetSkill(game.SkillRescue, 50)
	if err := world.AddPlayer(rescuer); err != nil {
		t.Fatalf("AddPlayer rescuer: %v", err)
	}

	target := game.NewPlayer(2, "Target", 1001)
	if err := world.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}

	return &rescueCommandSession{
		player:       rescuer,
		world:        world,
		combatEngine: combatEngine,
	}
}

func TestCmdRescueUnavailableCombatEngineDoesNotPanic(t *testing.T) {
	tests := []struct {
		name         string
		combatEngine interface{}
	}{
		{name: "nil", combatEngine: nil},
		{name: "incompatible", combatEngine: struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := newRescueCommandSession(t, tt.combatEngine)

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CmdRescue panicked: %v", r)
				}
			}()

			if err := CmdRescue(session, []string{"target"}); err != nil {
				t.Fatalf("CmdRescue returned error: %v", err)
			}
			if len(session.messages) != 1 {
				t.Fatalf("expected one message, got %d", len(session.messages))
			}
			if !strings.Contains(session.messages[0], "Combat is unavailable") {
				t.Fatalf("unexpected message: %q", session.messages[0])
			}
		})
	}
}

func newBashCommandSession(t *testing.T) *rescueCommandSession {
	t.Helper()

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	basher := game.NewPlayer(1, "Basher", 1001)
	basher.Class = game.ClassWarrior
	basher.SetLevel(10)
	basher.SetSkill(game.SkillBash, 100)
	basher.SetPosition(combat.PosFighting)
	if err := world.AddPlayer(basher); err != nil {
		t.Fatalf("AddPlayer basher: %v", err)
	}

	target := game.NewPlayer(2, "Target", 1001)
	target.SetPosition(combat.PosFighting)
	if err := world.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}

	return &rescueCommandSession{
		player: basher,
		world:  world,
	}
}

func TestCmdBash_FightingTargetFallback(t *testing.T) {
	session := newBashCommandSession(t)
	session.player.SetFighting("Target")

	if err := CmdBash(session, nil); err != nil {
		t.Fatalf("CmdBash returned error: %v", err)
	}

	for _, msg := range session.messages {
		if strings.Contains(msg, "Bash who?") {
			t.Errorf("expected bash to target fighting opponent, got: %q", msg)
		}
	}
}

func TestCmdBash_NoFightingNoArgs(t *testing.T) {
	session := newBashCommandSession(t)

	if err := CmdBash(session, nil); err != nil {
		t.Fatalf("CmdBash returned error: %v", err)
	}

	found := false
	for _, msg := range session.messages {
		if strings.Contains(msg, "Bash who?") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Bash who?' when not fighting and no args, got: %v", session.messages)
	}
}
