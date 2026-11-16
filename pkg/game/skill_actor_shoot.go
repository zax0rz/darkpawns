package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// characterShoot ports do_shoot for a switched mobile body. The body owns its
// inventory, equipment, skill, dexterity and room; the descriptor only
// receives private output (src/act.offensive.c:746-987).
func characterShoot(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	if ctx.World.DummyMobSkill(skillNumShoot) == 0 { // SKILL_SHOOT, src/spells.h:196.
		mobileSend(ctx, mob, "You have no idea how.\r\n")
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() != nil {
		mobileSend(ctx, mob, "But you are already engaged in close-range combat!\r\n")
		return CharacterCommandHandled
	}
	raw := ctx.RawArgs
	if raw == "" && len(args) > 0 {
		raw = strings.Join(args, " ")
	}
	projectileName, rest := OneArgument(raw)
	directionName, targetName := OneArgument(rest)
	if projectileName == "" {
		mobileSend(ctx, mob, "Shoot what where?\r\n")
		return CharacterCommandHandled
	}
	if directionName == "" {
		mobileSend(ctx, mob, "Where would you like to shoot it?\r\n")
		return CharacterCommandHandled
	}
	if targetName == "" {
		mobileSend(ctx, mob, "Who would you like to shoot it at in that direction?\r\n")
		return CharacterCommandHandled
	}
	projectile, found := ctx.World.ResolveObjectInInventory(mob, projectileName)
	if !found {
		mobileSend(ctx, mob, fmt.Sprintf("You don't seem to have any %ss.\r\n", projectileName))
		return CharacterCommandHandled
	}
	if projectile.GetTypeFlag() != int(ItemMissile) {
		mobileActToChar(ctx, mob, projectile, nil, "$p is not a projectile!")
		return CharacterCommandHandled
	}

	directions := []string{"north", "east", "south", "west", "up", "down"}
	dir := searchBlock(strings.ToLower(directionName), directions, false)
	if dir < 0 {
		// C's final else emits the lower-case refusal for an unrecognized
		// search_block() result (src/act.offensive.c:990-996).
		mobileSend(ctx, mob, "Interesting direction.\r\n")
		return CharacterCommandHandled
	}
	direction := directions[dir]
	from := []string{"the south", "the west", "the north", "the east", "below", "above"}[dir]
	room := ctx.World.GetRoomInWorld(mob.GetRoom())
	if room == nil {
		mobileSend(ctx, mob, "You are nowhere.\r\n")
		return CharacterCommandHandled
	}
	exit, hasExit := room.Exits[direction]
	if !hasExit {
		// A recognized direction with no exit reaches the earlier capitalized
		// refusal (src/act.offensive.c:814-820).
		mobileSend(ctx, mob, "Interesting Direction.\r\n")
		return CharacterCommandHandled
	}
	bow := mobileWielded(mob)
	if bow == nil || bow.GetTypeFlag() != int(ItemFireWeapon) {
		mobileSend(ctx, mob, "You must wield a bow or sling to fire a projectile.\r\n")
		return CharacterCommandHandled
	}
	if exit.ToRoom < 0 {
		mobileSend(ctx, mob, "Alas, you cannot shoot that way...\r\n")
		return CharacterCommandHandled
	}
	if exit.ExitInfo&parser.ExitClosed != 0 {
		if exit.Keywords != "" {
			mobileSend(ctx, mob, fmt.Sprintf("The %s seems to be closed.\r\n", CFName(exit.Keywords)))
		} else {
			mobileSend(ctx, mob, "It seems to be closed.\r\n")
		}
		return CharacterCommandHandled
	}
	targetRoom := ctx.World.GetRoomInWorld(exit.ToRoom)
	if targetRoom == nil {
		mobileSend(ctx, mob, "Alas, you cannot shoot that way...\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(targetRoom.VNum, "peaceful") || ctx.World.RoomHasFlag(room.VNum, "peaceful") {
		mobileSend(ctx, mob, "You feel too peaceful to contemplate violence.\r\n")
		return CharacterCommandHandled
	}
	target := ctx.World.ResolveShootTarget(exit.ToRoom, targetName)
	if target == nil {
		mobileSend(ctx, mob, "Twang...\r\n")
		if err := ctx.World.MoveObjectToRoomFront(projectile, exit.ToRoom); err != nil {
			slog.Error("switched-body shoot projectile placement failed", "error", err)
		}
		return CharacterCommandHandled
	}
	if !target.IsNPC() && (target.GetLevel() < 10 || target.GetLevel() > 30) {
		mobileSend(ctx, mob, "Maybe that isn't such a great idea...\r\n")
		return CharacterCommandHandled
	}
	if target.GetFightingBody() != nil {
		mobileSend(ctx, mob, "It looks like they are fighting, you can't aim properly.\r\n")
		return CharacterCommandHandled
	}
	if targetMob, ok := target.(*MobInstance); ok && targetMob.HasFlag(MobSentinel) {
		mobileSend(ctx, mob, "You cannot see well enough to aim...\r\n")
		return CharacterCommandHandled
	}

	Act(ctx.World, false, mob, nil, nil, nil,
		fmt.Sprintf("$n fires %s %s with %s.", IndefiniteArticle(projectileName), projectileName, bow.GetShortDesc()), "", ToRoom)
	mobileSend(ctx, mob, "Twang... your projectile flies into the distance.\r\n")
	if err := ctx.World.MoveObjectToNowhere(projectile); err != nil {
		slog.Error("switched-body shoot projectile detach failed", "error", err)
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	missile, _ := combat.RangedDexAdjustments(mob.GetDex())
	_, reaction := combat.RangedDexAdjustments(target.GetDex())
	probability := ctx.World.DummyMobSkill(skillNumShoot) + missile*10 - reaction*10
	if percent >= probability {
		sendCombatantMessage(target, fmt.Sprintf("Some kind of %s streaks in from %s and just misses you!\r\n", projectileName, from))
		Act(ctx.World, false, asActor(target), nil, nil, nil,
			fmt.Sprintf("Some kind of %s streaks in from %s and narrowly misses $n!", projectileName, from), "", ToRoom)
		if err := ctx.World.MoveObjectToRoomFront(projectile, target.GetRoom()); err != nil {
			slog.Error("switched-body missed projectile placement failed", "error", err)
		}
		return CharacterCommandHandled
	}

	damage := mob.GetDamroll() + dprng.Dice(projectile.GetValue(1), projectile.GetValue(2)) + dprng.Dice(bow.GetValue(1), bow.GetValue(2))
	mobileSend(ctx, mob, "You hear a roar of pain!\r\n")
	ctx.World.ExtractObject(projectile, mob.GetRoom())
	if target.IsNPC() {
		Act(ctx.World, false, asActor(target), nil, nil, nil,
			fmt.Sprintf("Some kind of %s streaks in from %s and strikes $n!", projectileName, from), "", ToRoom)
		sendCombatantMessage(target, "Suddenly some kind of projectile pierces your arm!\r\n")
		if ctx.World.ApplyRangedProjectileDamage(target, damage) {
			return CharacterCommandHandled
		}
		sendCombatantMessage(target, "You decide to go investigate...\r\n")
		targetMob, ok := target.(*MobInstance)
		if !ok {
			return CharacterCommandHandled
		}
		if err := ctx.World.TransferRangedVictim(targetMob, mob.GetRoom()); err != nil {
			slog.Error("switched-body shoot victim transfer failed", "error", err)
			return CharacterCommandHandled
		}
		Act(ctx.World, false, asActor(target), mob, nil, nil,
			"$n bursts into the room and scowls at $N.\r\n", "", ToNotVict)
		mobileSend(ctx, mob, fmt.Sprintf("%s bursts into the room and scowls at you!\r\n", target.GetName()))
		if engine, ok := ctx.World.combatEngine.(interface {
			PerformRangedRetaliation(combat.Combatant, combat.Combatant) error
		}); ok {
			if err := engine.PerformRangedRetaliation(target, mob); err != nil {
				slog.Warn("switched-body shoot retaliation failed", "target", target.GetName(), "error", err)
			}
		}
		return CharacterCommandHandled
	}
	sendCombatantMessage(target, fmt.Sprintf("Some kind of %s streaks in from %s and hits you!\r\n", projectileName, from))
	Act(ctx.World, false, asActor(target), nil, nil, nil,
		fmt.Sprintf("Some kind of %s streaks in from %s and strikes $n!", projectileName, from), "", ToRoom)
	ctx.World.ApplyRangedProjectileDamage(target, damage)
	return CharacterCommandHandled
}

func sendCombatantMessage(target combat.Combatant, message string) {
	switch body := target.(type) {
	case *Player:
		body.SendMessage(message)
	case *MobInstance:
		body.SendMessage(message)
	}
}
