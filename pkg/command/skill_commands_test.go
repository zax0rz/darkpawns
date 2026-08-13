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
	// Use-based improvement (improve_skill, act.other.c:1704) may lawfully
	// append "Your skill in X improves." (R1) — the time-seeded dprng stream
	// makes that fire a few percent of runs, so assert the skill's own line
	// was emitted rather than being the last message (CI flake, DP-1215 era).
	if got := strings.Join(session.messages, ""); !strings.Contains(got, "Okay, you'll try to move silently for a while.\r\n") {
		t.Errorf("level-1 sneak output = %q", got)
	}

	if err := CmdHide(session, nil); err != nil {
		t.Fatalf("CmdHide: %v", err)
	}
	if got := strings.Join(session.messages, ""); !strings.Contains(got, "You attempt to hide yourself.\r\n") {
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

func TestCmdBerserk_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdBerserk(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdBerserk_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)

	if err := CmdBerserk(session, nil); err != nil {
		t.Fatalf("CmdBerserk: %v", err)
	}
}

func TestCmdDetect_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdDetect(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdDetect_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)

	if err := CmdDetect(session, nil); err != nil {
		t.Fatalf("CmdDetect: %v", err)
	}
}

func TestCmdDig_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdDig(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdDig_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)

	if err := CmdDig(session, nil); err != nil {
		t.Fatalf("CmdDig: %v", err)
	}
}

func TestCmdScrounge_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdScrounge(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdScrounge_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)

	if err := CmdScrounge(session, nil); err != nil {
		t.Fatalf("CmdScrounge: %v", err)
	}
}

func TestCmdStrike_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdStrike(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdStrike_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	if err := CmdStrike(session, []string{"target"}); err != nil {
		t.Fatalf("CmdStrike: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Yeah, right") {
		t.Errorf("expected 'Yeah, right', got: %v", session.messages)
	}
}

func TestCmdCutthroat_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdCutthroat(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdCutthroat_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdCutthroat(session, nil); err != nil {
		t.Fatalf("CmdCutthroat: %v", err)
	}
	// C do_cutthroat: GET_SKILL checked before no-arg — a no-skill caller gets
	// "You're not trained in slitting throats!" regardless of args.
	if !strings.Contains(joinMessages(session.messages), "slitting throats") {
		t.Errorf("expected skill-gate message, got: %v", session.messages)
	}
}

func TestCmdCutthroat_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdCutthroat(session, []string{"target"}); err != nil {
		t.Fatalf("CmdCutthroat: %v", err)
	}
	// C: "You're not trained in slitting throats!" (SkillUnknownMsg, DP-1206).
	if !strings.Contains(joinMessages(session.messages), "slitting throats") {
		t.Errorf("expected 'slitting throats', got: %v", session.messages)
	}
}

func TestCmdCarve_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdCarve(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdCarve_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdCarve(session, nil); err != nil {
		t.Fatalf("CmdCarve: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "carve what") {
		t.Errorf("expected 'carve what', got: %v", session.messages)
	}
}

func TestCmdCarve_FightingPosition(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosFighting)
	if err := CmdCarve(session, []string{"corpse"}); err != nil {
		t.Fatalf("CmdCarve: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "How can you think of food") {
		t.Errorf("expected food message, got: %v", session.messages)
	}
}

func TestCmdCompare_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdCompare(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdCompare_Fighting(t *testing.T) {
	session := newSkillCommandSession(t)
	// C do_compare checks FIGHTING(ch) (the actual fight target), not position.
	session.player.Fighting = "goblin"
	if err := CmdCompare(session, []string{"sword"}); err != nil {
		t.Fatalf("CmdCompare: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "pretty busy") {
		t.Errorf("expected busy message, got: %v", session.messages)
	}
}

func TestCmdCompare_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdCompare(session, nil); err != nil {
		t.Fatalf("CmdCompare: %v", err)
	}
	// C do_compare: empty/missing args → get_obj_in_list_vis fails →
	// "Looks like you don't have those objects.." (no CanUseSkill gate).
	if !strings.Contains(joinMessages(session.messages), "don't have those objects") {
		t.Errorf("expected 'don't have those objects', got: %v", session.messages)
	}
}

func TestCmdSharpen_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdSharpen(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdSharpen_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSharpen(session, nil); err != nil {
		t.Fatalf("CmdSharpen: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Sharpen what") {
		t.Errorf("expected 'Sharpen what', got: %v", session.messages)
	}
}

func TestCmdSharpen_Fighting(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosFighting)
	if err := CmdSharpen(session, []string{"sword"}); err != nil {
		t.Fatalf("CmdSharpen: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "too busy") {
		t.Errorf("expected 'too busy', got: %v", session.messages)
	}
}

func TestCmdBehead_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdBehead(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdBehead_Fighting(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosFighting)
	if err := CmdBehead(session, []string{"corpse"}); err != nil {
		t.Fatalf("CmdBehead: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "a little busy") {
		t.Errorf("expected 'little busy', got: %v", session.messages)
	}
}

func TestCmdBehead_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdBehead(session, nil); err != nil {
		t.Fatalf("CmdBehead: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Behead who") {
		t.Errorf("expected 'Behead who', got: %v", session.messages)
	}
}

func TestCmdDisembowel_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdDisembowel(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdDisembowel_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdDisembowel(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdDisembowel: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "You have no idea how") {
		t.Errorf("expected 'You have no idea how', got: %v", session.messages)
	}
}

func TestCmdDisembowel_NoTarget(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdDisembowel(session, nil); err != nil {
		t.Fatalf("CmdDisembowel: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "You have no idea how") {
		t.Errorf("expected 'You have no idea how', got: %v", session.messages)
	}
}

func TestCmdDragonKick_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdDragonKick(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdDragonKick_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdDragonKick(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdDragonKick: %v", err)
	}
	// C do_dragon_kick: "What's that, idiot-san?\r\n" (SkillUnknownMsg audit).
	if !strings.Contains(joinMessages(session.messages), "idiot-san") {
		t.Errorf("expected 'idiot-san', got: %v", session.messages)
	}
}

func TestCmdDragonKick_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdDragonKick(session, nil); err != nil {
		t.Fatalf("CmdDragonKick: %v", err)
	}
	// C do_dragon_kick: "What's that, idiot-san?\r\n" (SkillUnknownMsg audit).
	if !strings.Contains(joinMessages(session.messages), "idiot-san") {
		t.Errorf("expected 'idiot-san', got: %v", session.messages)
	}
}

func TestCmdTigerPunch_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdTigerPunch(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdTigerPunch_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdTigerPunch(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdTigerPunch: %v", err)
	}
	// C: "What's that, idiot-san?\r\n" (SkillUnknownMsg, DP-1206).
	if !strings.Contains(joinMessages(session.messages), "idiot-san") {
		t.Errorf("expected 'idiot-san', got: %v", session.messages)
	}
}

func TestCmdFirstAid_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdFirstAid(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdFirstAid_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdFirstAid(session, nil); err != nil {
		t.Fatalf("CmdFirstAid: %v", err)
	}
	// C do_first_aid: GET_SKILL checked before no-arg — no-skill player gets
	// "You have no idea how!" (SkillUnknownMsg, DP-1206).
	if !strings.Contains(joinMessages(session.messages), "no idea how") {
		t.Errorf("expected skill-gate message, got: %v", session.messages)
	}
}

func TestCmdFirstAid_NoTarget(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdFirstAid(session, []string{"ghost"}); err != nil {
		t.Fatalf("CmdFirstAid: %v", err)
	}
	// Skill gate fires before target lookup (C do_first_aid order).
	if !strings.Contains(joinMessages(session.messages), "no idea how") {
		t.Errorf("expected skill-gate message, got: %v", session.messages)
	}
}

func TestCmdTurn_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdTurn(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdTurn_NoFightingNoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdTurn(session, nil); err != nil {
		t.Fatalf("CmdTurn: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Turn who") {
		t.Errorf("expected 'Turn who', got: %v", session.messages)
	}
}

func TestCmdSerpentKick_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdSerpentKick(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdSerpentKick_NoFightingNoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSerpentKick(session, nil); err != nil {
		t.Fatalf("CmdSerpentKick: %v", err)
	}
	// C do_serpent_kick: GET_SKILL checked before target — no-skill player gets
	// "You'd better leave all the martial arts to others."
	if !strings.Contains(joinMessages(session.messages), "martial arts to others") {
		t.Errorf("expected skill-gate message, got: %v", session.messages)
	}
}

func TestCmdDisarm_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdDisarm(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdDisarm_NoFightingNoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdDisarm(session, nil); err != nil {
		t.Fatalf("CmdDisarm: %v", err)
	}
	// C do_disarm: GET_SKILL(DISARM) checked before target lookup — a no-skill
	// caller gets "You'd better leave all the martial arts to fighters."
	if !strings.Contains(joinMessages(session.messages), "martial arts") {
		t.Errorf("expected skill-gate message, got: %v", session.messages)
	}
}

func TestCmdMindlink_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdMindlink(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdMindlink_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdMindlink(session, nil); err != nil {
		t.Fatalf("CmdMindlink: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Link your mind") {
		t.Errorf("expected 'Link your mind', got: %v", session.messages)
	}
}

func TestCmdMold_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdMold(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdMold_NotEnoughArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdMold(session, []string{"clay"}); err != nil {
		t.Fatalf("CmdMold: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "mold <object>") {
		t.Errorf("expected usage message, got: %v", session.messages)
	}
}

func TestCmdWhois_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdWhois(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdWhois_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdWhois(session, nil); err != nil {
		t.Fatalf("CmdWhois: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "whom do you wish") {
		t.Errorf("expected 'whom do you wish', got: %v", session.messages)
	}
}

func TestCmdReview_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdReview(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdReview_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdReview(session, nil); err != nil {
		t.Fatalf("CmdReview: %v", err)
	}
}

func TestCmdKujiKiri_NoPlayer(t *testing.T) {
	handler := CmdKujiKiri("rin")
	session := &skillCommandSession{}
	if err := handler(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdKujiKiri_Executes(t *testing.T) {
	handler := CmdKujiKiri("rin")
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := handler(session, nil); err != nil {
		t.Fatalf("CmdKujiKiri: %v", err)
	}
}

func TestCmdPoint_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdPoint(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdPoint_Executes(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdPoint(session, []string{"north"}); err != nil {
		t.Fatalf("CmdPoint: %v", err)
	}
}

func TestCmdPalm_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdPalm(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdPalm_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdPalm(session, nil); err != nil {
		t.Fatalf("CmdPalm: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Palm what") {
		t.Errorf("expected 'Palm what', got: %v", session.messages)
	}
}

func TestCmdFleshAlter_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdFleshAlter(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdFleshAlter_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdFleshAlter(session, nil); err != nil {
		t.Fatalf("CmdFleshAlter: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "altering your flesh") {
		t.Errorf("expected 'altering your flesh', got: %v", session.messages)
	}
}

func TestCmdSpike_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSpike(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdSpike: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "You have no idea how") {
		t.Errorf("expected 'You have no idea how', got: %v", session.messages)
	}
}

func TestCmdStake_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdStake(session, []string{"rat"}); err != nil {
		t.Fatalf("CmdStake: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "You have no idea how") {
		t.Errorf("expected 'You have no idea how', got: %v", session.messages)
	}
}

func TestCmdSpike_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdSpike(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdSpike_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSpike(session, nil); err != nil {
		t.Fatalf("CmdSpike: %v", err)
	}
	// C do_spike no-arg → "Whom do you wish to spike?"
	if !strings.Contains(joinMessages(session.messages), "wish to spike") {
		t.Errorf("expected 'wish to spike', got: %v", session.messages)
	}
}

func TestCmdStake_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdStake(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdStake_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdStake(session, nil); err != nil {
		t.Fatalf("CmdStake: %v", err)
	}
	// Arg check fires before skill check for this command
	if !strings.Contains(joinMessages(session.messages), "wish to stake") {
		t.Errorf("expected 'wish to stake', got: %v", session.messages)
	}
}

func TestCmdBite_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdBite(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdBite_NoFightingNoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdBite(session, nil); err != nil {
		t.Fatalf("CmdBite: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Bite who") {
		t.Errorf("expected 'Bite who', got: %v", session.messages)
	}
}

func TestCmdBearhug_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdBearhug(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdBearhug_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdBearhug(session, nil); err != nil {
		t.Fatalf("CmdBearhug: %v", err)
	}
	// C: "You'd better leave all the martial arts to fighters.\n\r"
	if !strings.Contains(joinMessages(session.messages), "martial arts to fighters") {
		t.Errorf("expected 'martial arts to fighters', got: %v", session.messages)
	}
}

func TestCmdSmackheads_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSmackheads(session, []string{"one", "two"}); err != nil {
		t.Fatalf("CmdSmackheads: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Rosie") {
		t.Errorf("expected 'Rosie', got: %v", session.messages)
	}
}

func TestCmdSlug_NoSkill(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdSlug(session, nil); err != nil {
		t.Fatalf("CmdSlug: %v", err)
	}
	// C: "You couldn't slug your way out of a wet paper bag." (SkillUnknownMsg, DP-1206).
	if !strings.Contains(joinMessages(session.messages), "wet paper bag") {
		t.Errorf("expected 'wet paper bag', got: %v", session.messages)
	}
}

func TestCmdSmackheads_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdSmackheads(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdSlug_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdSlug(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdTag_NoPlayer(t *testing.T) {
	session := &skillCommandSession{}
	if err := CmdTag(session, nil); err == nil {
		t.Fatal("expected error when player is nil")
	}
}

func TestCmdTag_NoArgs(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosStanding)
	if err := CmdTag(session, nil); err != nil {
		t.Fatalf("CmdTag: %v", err)
	}
	if !strings.Contains(joinMessages(session.messages), "Tag who") {
		t.Errorf("expected 'Tag who', got: %v", session.messages)
	}
}
