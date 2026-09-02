package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCmdSerpentKickTrainingMobileMatchesC(t *testing.T) {
	seed := uint32(0)
	for candidate := uint32(1); candidate <= 10000; candidate++ {
		dprng.ResetStream(candidate)
		if 14+dprng.Number(1, 101) > 50 {
			continue
		}
		dprng.Dice(1, 3) // set 156's skill_message variant
		if dprng.Number(0, 80) == 0 {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("no hit seed reached the C training-mobile draw")
	}

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Serpent Arena", Zone: 1},
			{VNum: 18201, Name: "Kir-Oshi Temple", Zone: 1},
		},
		Mobs: []parser.Mob{
			{
				VNum: 16303, Keywords: "target", ShortDesc: "a target",
				HP:    parser.DiceRoll{Num: 1, Sides: 1, Plus: 999},
				Level: 1, AC: 0,
			},
			{
				VNum: 18221, Keywords: "training serpent", ShortDesc: "a training serpent",
				HP:    parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
				Level: 1, AC: 0, Exp: 777,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	actor := game.NewPlayer(1, "Serpentatt", 1001)
	actor.SetLevel(20)
	actor.SetSkill(game.SkillSerpentKick, 50)
	actor.Stats.Int = 100
	actor.Stats.Wis = 100
	if err := world.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	target, err := world.SpawnMobQuiet(16303, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet target: %v", err)
	}

	rig := newDrawOrderRig(t, actor.Name)
	defer rig.teardown()
	session := &skillCommandSession{player: actor, world: world, combatEngine: rig.engine}
	dprng.ResetStream(seed)
	if err := CmdSerpentKick(session, []string{"target"}); err != nil {
		t.Fatalf("CmdSerpentKick: %v", err)
	}
	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Dice(1, 3)
	dprng.Number(0, 80)
	dprng.Dice(1, 1) // read_mobile() rolls the training prototype's HP first
	dprng.Number(1, 200)
	wantSkill := 50 + dprng.Number(1, 3)
	if actor.GetSkill(game.SkillSerpentKick) != wantSkill {
		t.Errorf("post-training skill = %d, want %d; improve_skill must follow the C create_mobile draw",
			actor.GetSkill(game.SkillSerpentKick), wantSkill)
	}

	var training *game.MobInstance
	for _, mob := range world.GetMobsInRoom(18201) {
		if mob.GetVNum() == 18221 {
			training = mob
			break
		}
	}
	if training == nil {
		t.Fatal("successful C training draw did not create mob 18221 in room 18201")
	}
	if training.GetLevel() != 23 || training.GetMaxHP() != 253 || training.GetHP() != 253 {
		t.Errorf("training stats = level %d HP %d/%d, want 23 and 253/253",
			training.GetLevel(), training.GetHP(), training.GetMaxHP())
	}
	if training.GetAC() != -130 || training.GetHitroll() != 23 || training.GetDamroll() != 16 {
		t.Errorf("training combat stats = AC %d hitroll %d damroll %d, want -130/23/16",
			training.GetAC(), training.GetHitroll(), training.GetDamroll())
	}
	if got := training.GetDamageRoll(); got != (combat.DiceRoll{Num: 15, Sides: 4}) {
		t.Errorf("training damage dice = %+v, want 15d4", got)
	}
	if training.GetHunting() != actor.Name {
		t.Errorf("training hunting target = %q, want %q", training.GetHunting(), actor.Name)
	}
	if training.GetExp() != 0 {
		t.Errorf("training exp = %d, want 0", training.GetExp())
	}
	if strings.Contains(strings.Join(session.messages, ""), "appears") {
		t.Errorf("training spawn announced like an ordinary world spawn: %q", strings.Join(session.messages, ""))
	}
	_ = target
}
