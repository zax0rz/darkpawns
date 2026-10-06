package command

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// executeShoot follows src/act.offensive.c:904-987, without SkillResult/damage.
func executeShoot(s SessionInterface, ch *game.Player, target combat.Combatant, projectile, bow *game.ObjectInstance, name, direction string) error {
	world := s.GetWorld()
	victim, ok := target.(game.Actor)
	if !ok {
		return fmt.Errorf("shoot target is not an actor")
	}
	from := map[string]string{"north": "the south", "east": "the west", "south": "the north", "west": "the east", "up": "below", "down": "above"}[direction]
	game.Act(world, false, ch, nil, nil, nil, fmt.Sprintf("$n fires %s %s with %s.", game.IndefiniteArticle(name), name, bow.GetShortDesc()), "", game.ToRoom)
	if err := s.SendMessage("Twang... your projectile flies into the distance.\r\n"); err != nil {
		return err
	}
	if err := world.MoveObjectToNowhere(projectile); err != nil {
		return fmt.Errorf("detach shoot projectile: %w", err)
	}
	percent := dprng.Number(1, 101)
	missile, _ := combat.RangedDexAdjustments(ch.GetDex())
	_, reaction := combat.RangedDexAdjustments(target.GetDex())
	probability := ch.GetSkill(game.SkillShoot) + missile*10 - reaction*10
	if percent >= probability {
		victim.SendMessage(fmt.Sprintf("Some kind of %s streaks in from %s and just misses you!\r\n", name, from))
		game.Act(world, false, victim, nil, nil, nil, fmt.Sprintf("Some kind of %s streaks in from %s and narrowly misses $n!", name, from), "", game.ToRoom)
		if err := world.MoveObjectToRoomFront(projectile, target.GetRoom()); err != nil {
			return fmt.Errorf("drop missed projectile: %w", err)
		}
		return nil
	}
	damage := ch.GetDamroll() + dprng.Dice(projectile.GetValue(1), projectile.GetValue(2)) + dprng.Dice(bow.GetValue(1), bow.GetValue(2))
	if err := s.SendMessage("You hear a roar of pain!\r\n"); err != nil {
		return err
	}
	world.ExtractObject(projectile, ch.GetRoom())
	game.ImproveSkill(ch, game.SkillShoot)
	destination := fmt.Sprintf("Some kind of %s streaks in from %s and strikes $n!", name, from)
	if !target.IsNPC() {
		victim.SendMessage(fmt.Sprintf("Some kind of %s streaks in from %s and hits you!\r\n", name, from))
		game.Act(world, false, victim, nil, nil, nil, destination, "", game.ToRoom)
		world.ApplyRangedProjectileDamage(target, damage)
		return nil
	}
	game.Act(world, false, victim, nil, nil, nil, destination, "", game.ToRoom)
	victim.SendMessage("Suddenly some kind of projectile pierces your arm!\r\n")
	if world.ApplyRangedProjectileDamage(target, damage) {
		return nil
	}
	victim.SendMessage("You decide to go investigate...\r\n")
	mob, ok := target.(*game.MobInstance)
	if !ok {
		return fmt.Errorf("shoot NPC is not a mobile")
	}
	if err := world.TransferRangedVictim(mob, ch.GetRoom()); err != nil {
		return fmt.Errorf("relocate shoot victim: %w", err)
	}
	game.Act(world, false, victim, ch, nil, nil, "$n bursts into the room and scowls at $N.\r\n", "", game.ToNotVict)
	if err := s.SendMessage(fmt.Sprintf("%s bursts into the room and scowls at you!\r\n", target.GetName())); err != nil {
		return err
	}
	engine, ok := s.GetCombatEngine().(interface {
		PerformRangedRetaliation(combat.Combatant, combat.Combatant) error
	})
	if !ok {
		return fmt.Errorf("shoot retaliation engine unavailable")
	}
	return engine.PerformRangedRetaliation(target, ch)
}
