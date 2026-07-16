package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
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

func TestCmdBash_AppliesWaitStateToMobTarget(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "rat",
			ShortDesc: "a rat",
			LongDesc:  "A rat is here.",
			Level:     1,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	basher := game.NewPlayer(1, "Basher", 1001)
	basher.Class = game.ClassWarrior
	basher.SetLevel(game.LVL_IMMORT) // auto-succeed bash
	basher.SetSkill(game.SkillBash, 100)
	basher.SetPosition(combat.PosFighting)
	basher.SetMove(100)
	if err := world.AddPlayer(basher); err != nil {
		t.Fatalf("AddPlayer basher: %v", err)
	}

	mob, err := world.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosFighting)

	session := &rescueCommandSession{player: basher, world: world}
	if err := CmdBash(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdBash returned error: %v", err)
	}

	if mob.GetWaitState() != 2 {
		t.Errorf("expected mob wait state 2 after bash, got %d", mob.GetWaitState())
	}
	if mob.GetPosition() != combat.PosSitting {
		t.Errorf("expected mob position sitting after bash, got %d", mob.GetPosition())
	}
}

// ---------------------------------------------------------------------------
// Utility skill command tests (DP-608 / DP-658)
// ---------------------------------------------------------------------------

type skillCommandSession struct {
	player       *game.Player
	world        *game.World
	combatEngine interface{}
	messages     []string
	tempData     map[string]interface{}
}

func (s *skillCommandSession) GetPlayer() *game.Player { return s.player }

func (s *skillCommandSession) SendMessage(message string) error {
	s.messages = append(s.messages, message)
	return nil
}

func (s *skillCommandSession) Send(message string)          { s.messages = append(s.messages, message) }
func (s *skillCommandSession) MarkDirty(vars ...string)     {}
func (s *skillCommandSession) GetManager() interface{}      { return nil }
func (s *skillCommandSession) GetWorld() *game.World        { return s.world }
func (s *skillCommandSession) GetCombatEngine() interface{} { return s.combatEngine }
func (s *skillCommandSession) RandomInt(maxValue int) int   { return 0 }

func (s *skillCommandSession) SetTempData(key string, value interface{}) {
	if s.tempData == nil {
		s.tempData = make(map[string]interface{})
	}
	s.tempData[key] = value
}

func (s *skillCommandSession) GetTempData(key string) interface{} {
	if s.tempData == nil {
		return nil
	}
	return s.tempData[key]
}

func (s *skillCommandSession) ClearTempData(key string) {
	if s.tempData != nil {
		delete(s.tempData, key)
	}
}

func newSkillCommandSession(t *testing.T) *skillCommandSession {
	t.Helper()

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	player := game.NewPlayer(1, "Tester", 1001)
	player.SetLevel(10)
	player.Stats = game.CharStats{Str: 18, Dex: 16, Int: 14, Wis: 12, Con: 12, Cha: 10}
	if err := world.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	return &skillCommandSession{player: player, world: world}
}

func TestNewbieThiefUtilityCommandsBypassGuildMinimumLevels(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.Class = game.ClassThief
	session.player.SetLevel(1)
	session.player.SetPosition(combat.PosStanding)
	session.player.SetSkill(game.SkillSneak, 10)
	session.player.SetSkill(game.SkillHide, 5)

	if err := CmdSneak(session, nil); err != nil {
		t.Fatalf("CmdSneak: %v", err)
	}
	if got := session.messages[len(session.messages)-1]; got != "Okay, you'll try to move silently for a while.\r\n" {
		t.Errorf("level-1 sneak output = %q", got)
	}

	if err := CmdHide(session, nil); err != nil {
		t.Fatalf("CmdHide: %v", err)
	}
	if got := session.messages[len(session.messages)-1]; got != "You attempt to hide yourself.\r\n" {
		t.Errorf("level-1 hide output = %q", got)
	}

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      2001,
			Keywords:  "mark target",
			ShortDesc: "a test mark",
			Level:     1,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld for steal: %v", err)
	}
	defer world.StopAITicker()

	thief := game.NewPlayer(2, "Newbie", 1001)
	thief.Class = game.ClassThief
	thief.SetLevel(1)
	thief.SetPosition(combat.PosStanding)
	thief.Stats.Str = 25
	thief.Stats.Dex = 25
	thief.SetSkill(game.SkillSteal, 1000)
	if err := world.AddPlayer(thief); err != nil {
		t.Fatalf("AddPlayer thief: %v", err)
	}
	mob, err := world.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetGold(0)

	stealSession := &skillCommandSession{player: thief, world: world}
	if err := CmdSteal(stealSession, []string{"coins", "mark"}); err != nil {
		t.Fatalf("CmdSteal: %v", err)
	}
	if got := stealSession.messages[len(stealSession.messages)-1]; got != "You couldn't get any gold...\r\n" {
		t.Errorf("level-1 steal output = %q", got)
	}
}

func TestCmdSkills(t *testing.T) {
	t.Run("no player", func(t *testing.T) {
		session := &skillCommandSession{}
		if err := CmdSkills(session, nil); err == nil {
			t.Fatal("expected error when player is nil")
		}
	})

	t.Run("no skill manager", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = nil
		if err := CmdSkills(session, nil); err != nil {
			t.Fatalf("CmdSkills error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "You have no skills") {
			t.Errorf("expected 'You have no skills', got: %v", session.messages)
		}
	})

	t.Run("no learned skills", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdSkills(session, nil); err != nil {
			t.Fatalf("CmdSkills error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "haven't learned any skills") {
			t.Errorf("expected 'haven't learned any skills', got: %v", session.messages)
		}
	})

	t.Run("lists learned skill", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 1, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdSkills(session, nil); err != nil {
			t.Fatalf("CmdSkills error: %v", err)
		}
		output := joinMessages(session.messages)
		if !strings.Contains(output, "Bash") {
			t.Errorf("expected skill name 'Bash' in output, got: %v", session.messages)
		}
		if !strings.Contains(output, "Skill points") {
			t.Errorf("expected skill points footer, got: %v", session.messages)
		}
	})
}

func TestCmdPractice(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		session := newSkillCommandSession(t)
		if err := CmdPractice(session, nil); err != nil {
			t.Fatalf("CmdPractice error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "Practice what?") {
			t.Errorf("expected 'Practice what?', got: %v", session.messages)
		}
	})

	t.Run("not learned", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdPractice(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdPractice error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "haven't learned") {
			t.Errorf("expected 'haven't learned', got: %v", session.messages)
		}
	})

	t.Run("already mastered", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 100, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdPractice(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdPractice error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "already mastered") {
			t.Errorf("expected 'already mastered', got: %v", session.messages)
		}
	})

	t.Run("practices learned skill", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 1, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdPractice(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdPractice error: %v", err)
		}
		output := joinMessages(session.messages)
		if !strings.Contains(output, "Bash") {
			t.Errorf("expected skill name in output, got: %v", session.messages)
		}
	})
}

func TestCmdLearn(t *testing.T) {
	t.Run("no args lists skills", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.InitializeDefaultSkills()
		session.player.SkillManager = sm
		if err := CmdLearn(session, nil); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "Available Skills") {
			t.Errorf("expected skill list output, got: %v", session.messages)
		}
	})

	t.Run("skill does not exist", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdLearn(session, []string{"notaskill"}); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "doesn't exist") {
			t.Errorf("expected 'doesn't exist', got: %v", session.messages)
		}
	})

	t.Run("already learned", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 1, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdLearn(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "already know") {
			t.Errorf("expected 'already know', got: %v", session.messages)
		}
	})

	t.Run("requirements not met", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SetLevel(1)
		session.player.Stats = game.CharStats{}
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 0, Difficulty: 5, MaxLevel: 100, Learned: false})
		session.player.SkillManager = sm
		if err := CmdLearn(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "don't meet the requirements") {
			t.Errorf("expected 'don't meet the requirements', got: %v", session.messages)
		}
	})

	t.Run("not enough skill points", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 0, Difficulty: 1, MaxLevel: 100, Learned: false})
		session.player.SkillManager = sm
		if err := CmdLearn(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "need 1 skill points") {
			t.Errorf("expected skill points message, got: %v", session.messages)
		}
	})

	t.Run("learn success", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.AddSkillPoints(10)
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 0, Difficulty: 1, MaxLevel: 100, Learned: false})
		session.player.SkillManager = sm
		if err := CmdLearn(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdLearn error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "successfully learn") {
			t.Errorf("expected 'successfully learn', got: %v", session.messages)
		}
	})
}

func TestCmdForget(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		session := newSkillCommandSession(t)
		if err := CmdForget(session, nil); err != nil {
			t.Fatalf("CmdForget error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "Forget what?") {
			t.Errorf("expected 'Forget what?', got: %v", session.messages)
		}
	})

	t.Run("not learned", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdForget(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdForget error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "haven't learned") {
			t.Errorf("expected 'haven't learned', got: %v", session.messages)
		}
	})

	t.Run("sets pending forget", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 1, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdForget(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdForget error: %v", err)
		}
		if session.GetTempData("skill_to_forget") != "bash" {
			t.Errorf("expected temp data skill_to_forget=bash, got %v", session.GetTempData("skill_to_forget"))
		}
	})
}

func TestCmdConfirmForget(t *testing.T) {
	t.Run("no pending skill", func(t *testing.T) {
		session := newSkillCommandSession(t)
		if err := CmdConfirmForget(session, nil); err != nil {
			t.Fatalf("CmdConfirmForget error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "No skill pending") {
			t.Errorf("expected 'No skill pending', got: %v", session.messages)
		}
	})

	t.Run("forgets pending skill", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 1, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		session.SetTempData("skill_to_forget", "bash")
		if err := CmdConfirmForget(session, nil); err != nil {
			t.Fatalf("CmdConfirmForget error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "You forget") {
			t.Errorf("expected 'You forget', got: %v", session.messages)
		}
		if session.GetTempData("skill_to_forget") != nil {
			t.Errorf("expected temp data cleared, got %v", session.GetTempData("skill_to_forget"))
		}
	})
}

func TestCmdUseSkill(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		session := newSkillCommandSession(t)
		if err := CmdUseSkill(session, nil); err != nil {
			t.Fatalf("CmdUseSkill error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "Use what skill?") {
			t.Errorf("expected 'Use what skill?', got: %v", session.messages)
		}
	})

	t.Run("not learned", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdUseSkill(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdUseSkill error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "haven't learned") {
			t.Errorf("expected 'haven't learned', got: %v", session.messages)
		}
	})

	t.Run("uses learned skill", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 50, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdUseSkill(session, []string{"bash", "target"}); err != nil {
			t.Fatalf("CmdUseSkill error: %v", err)
		}
		output := joinMessages(session.messages)
		if !strings.Contains(output, "attempt to use Bash") {
			t.Errorf("expected use attempt message, got: %v", session.messages)
		}
	})
}

func TestCmdSkillInfo(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		session := newSkillCommandSession(t)
		if err := CmdSkillInfo(session, nil); err != nil {
			t.Fatalf("CmdSkillInfo error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "Info on what skill?") {
			t.Errorf("expected 'Info on what skill?', got: %v", session.messages)
		}
	})

	t.Run("skill does not exist", func(t *testing.T) {
		session := newSkillCommandSession(t)
		session.player.SkillManager = engine.NewSkillManager()
		if err := CmdSkillInfo(session, []string{"notaskill"}); err != nil {
			t.Fatalf("CmdSkillInfo error: %v", err)
		}
		if !strings.Contains(joinMessages(session.messages), "doesn't exist") {
			t.Errorf("expected 'doesn't exist', got: %v", session.messages)
		}
	})

	t.Run("learned skill info", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 10, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: true})
		session.player.SkillManager = sm
		if err := CmdSkillInfo(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdSkillInfo error: %v", err)
		}
		output := joinMessages(session.messages)
		if !strings.Contains(output, "Bash") || !strings.Contains(output, "Learned") {
			t.Errorf("expected learned skill info, got: %v", session.messages)
		}
	})

	t.Run("unlearned skill info", func(t *testing.T) {
		session := newSkillCommandSession(t)
		sm := engine.NewSkillManager()
		sm.RegisterSkill(&engine.Skill{Name: "bash", DisplayName: "Bash", Type: engine.SkillTypeCombat, Level: 0, Practice: 0, Difficulty: 1, MaxLevel: 100, Learned: false})
		session.player.SkillManager = sm
		if err := CmdSkillInfo(session, []string{"bash"}); err != nil {
			t.Fatalf("CmdSkillInfo error: %v", err)
		}
		output := joinMessages(session.messages)
		if !strings.Contains(output, "Not learned") {
			t.Errorf("expected 'Not learned' info, got: %v", session.messages)
		}
	})
}

func joinMessages(messages []string) string {
	return strings.Join(messages, "")
}
