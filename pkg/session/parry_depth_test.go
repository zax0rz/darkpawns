package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestParryRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["parry"]
	if !ok {
		t.Fatal("parry command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("parry gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("parry")
	if !ok {
		t.Fatal("parry command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("parry registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestCmdParryEntryMessages(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Session, *game.MobInstance)
		want  string
	}{
		{
			name:  "unlearned",
			setup: func(_ *Session, _ *game.MobInstance) {},
			want:  "You're not good enough at swordplay to parry!\r\n",
		},
		{
			name: "not fighting",
			setup: func(s *Session, _ *game.MobInstance) {
				s.player.SetSkill(game.SkillParry, 100)
			},
			want: "But you aren't fighting anyone!\r\n",
		},
		{
			name: "opponent not attacking",
			setup: func(s *Session, mob *game.MobInstance) {
				s.player.SetSkill(game.SkillParry, 100)
				s.player.SetFighting(mob.GetName())
			},
			want: "But noone's attacking you!\r\n",
		},
		{
			name: "unarmed",
			setup: func(s *Session, mob *game.MobInstance) {
				s.player.SetSkill(game.SkillParry, 100)
				s.player.SetFighting(mob.GetName())
				mob.SetFighting(s.player.GetName())
			},
			want: "Parry with what? You're unarmed!\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeGateTestManager(t, false)
			mob, err := m.world.SpawnMob(5000, 1001)
			if err != nil {
				t.Fatalf("SpawnMob: %v", err)
			}
			s := makeGateSession(t, m, 1, "Parryactor", 20)
			tt.setup(s, mob)

			if err := cmdParry(s, []string{"ignored", "argument"}); err != nil {
				t.Fatalf("cmdParry: %v", err)
			}
			if got := readSendText(t, s); got != tt.want {
				t.Fatalf("parry output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCmdParrySuccessUsesCActAudiencesAndWait(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Parry Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			LongDesc:  "A test target stands here.",
			Level:     15,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
			Race:      1,
		}},
		Objs: []parser.Obj{{
			VNum:      6001,
			Keywords:  "sword",
			ShortDesc: "a short sword",
			WearFlags: [4]int{8193},
			Weight:    3,
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	m := newTestManager(t, w, nil)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	actor := makeGateSession(t, m, 1, "Parryactor", 20)
	observer := makeGateSession(t, m, 2, "Parryobserver", 20)
	weapon, err := m.world.SpawnObject(6001, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := actor.player.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := m.world.EquipItem(actor.player, weapon, cWearWield); err != nil {
		t.Fatalf("EquipItem: %v", err)
	}
	actor.player.SetSkill(game.SkillParry, 100)
	actor.player.SetFighting(mob.GetName())
	mob.SetFighting(actor.player.GetName())

	if err := cmdParry(actor, []string{"ignored", "argument"}); err != nil {
		t.Fatalf("cmdParry: %v", err)
	}
	if got, want := readSendText(t, actor), "With a dazzling show of swordplay, you move into defensive position...\r\n"; got != want {
		t.Fatalf("actor parry output = %q, want %q", got, want)
	}
	if got, want := readSendText(t, observer), "Parryactor displays a dazzling show of swordplay, fending off a test target's every blow!\r\n"; got != want {
		t.Fatalf("observer parry output = %q, want %q", got, want)
	}
	if got, want := actor.player.GetWaitState(), 2*20; got != want {
		t.Fatalf("parry wait state = %d, want %d pulses", got, want)
	}
}
