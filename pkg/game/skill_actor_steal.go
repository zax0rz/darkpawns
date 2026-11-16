package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// CharacterSteal ports do_steal for a switched mobile body. C's NPC skill is
// the shared dummy_mob value; the command still reaches its final one-round
// wait when that value is zero (src/act.other.c:309-538; src/utils.h:343,462-464).
func CharacterSteal(ctx CharacterCommandContext, _ []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	raw := ctx.RawArgs
	itemName, rest := OneArgument(raw)
	victimName, _ := OneArgument(rest)
	resolved, found := ctx.World.ResolveCharInRoom(mob, victimName)
	if !found || resolved.Combatant == nil {
		mobileSend(ctx, mob, "Steal what from who?\r\n")
		return CharacterCommandHandled
	}
	victim := resolved.Combatant
	if victim == mob {
		mobileSend(ctx, mob, "Come on now, that's rather stupid!\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You can't contemplate stealing in such a place!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if !victim.IsNPC() && (!ctx.DirectCall || ctx.Subcmd == 0) {
		message := fmt.Sprintf("You can not steal from %s because you are not an Outlaw!\r\n", victim.GetName())
		mobileSend(ctx, mob, message)
		victim.SendMessage(fmt.Sprintf("%s failed to steal from you because %s is not an Outlaw!\r\n", mob.GetName(), mob.GetName()))
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101) - dexAppSkill(mob.GetDex()).PPocket
	if player, ok := victim.(*Player); ok && player.GetPosition() <= combat.PosSleeping && player.GetLevel() < LVL_IMMORT {
		percent = -1
	}
	if player, ok := victim.(*Player); ok && mob.GetLevel() < LVL_IMMORT &&
		(player.GetLevel() > mob.GetLevel()+3 || player.GetLevel() < mob.GetLevel()-3) {
		percent = 101
	}
	if victim.GetLevel() >= LVL_IMMORT || ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") ||
		isShopKeeperInWorld(ctx.World, victim) || combatantIsAffected(victim, affRobbed) {
		percent = 101
	}
	if mob.GetLevel() > LVL_IMMORT && victim.GetLevel() < mob.GetLevel() {
		percent = -1
	}

	ohoh := false
	if !strings.EqualFold(itemName, "coins") && !strings.EqualFold(itemName, "gold") {
		if item := mobileStealCarriedItem(mob, victim, itemName); item != nil {
			percent += item.GetWeight()
			if victim.GetLevel() > mob.GetLevel() {
				percent += victim.GetLevel() - mob.GetLevel()
			}
			if percent > ctx.World.DummyMobSkill(skillNumSteal) {
				ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
					"$N catches you trying to steal something...", "", ToChar, mob, ctx.PrivateOutput)
				Act(ctx.World, false, mob, asActor(victim), nil, nil,
					"$n tried to steal something from you!", "", ToVict)
				Act(ctx.World, true, mob, asActor(victim), nil, nil,
					"$n tries to steal something from $N.", "", ToNotVict)
				ohoh = true
			} else if mobileCanCarryStolen(mob, item) {
				if err := ctx.World.MoveObjectToMobInventoryFront(item, mob); err != nil {
					slog.Error("switched-body steal inventory transfer failed", "item", item.GetVNum(), "thief", mob.GetVNum(), "error", err)
				} else {
					applyRobbedAffect(victim)
					if !victim.IsNPC() {
						MudLog(fmt.Sprintf("(PS) %s stole %s from %s.", mob.GetName(), item.GetShortDesc(), victim.GetName()), MudlogComplete, LVL_IMMORT, true)
					}
					if ctx.DirectCall && ctx.Subcmd != 0 {
						ActWithPrivateSink(ctx.World, true, mob, nil, item, nil,
							"Somehow $p makes it's way into your pack.", "", ToChar, mob, ctx.PrivateOutput)
					} else {
						mobileSend(ctx, mob, "Got it!\r\n")
					}
				}
			} else if mobCarriedItemCount(mob)+1 >= mobMaxCarryCount(mob) {
				mobileSend(ctx, mob, "You cannot carry that much.\r\n")
			}
		} else if item, _ := mobileStealEquippedItem(mob, victim, itemName); item != nil {
			if victim.GetPosition() > combat.PosSleeping {
				mobileSend(ctx, mob, "Steal the equipment now?  Impossible!\r\n")
				return CharacterCommandHandled
			}
			if percent > 100 {
				ActWithPrivateSink(ctx.World, false, mob, asActor(victim), item, nil,
					"You try to unequip $p, but fail!", "", ToChar, mob, ctx.PrivateOutput)
				Act(ctx.World, false, mob, asActor(victim), item, nil,
					"$n tries to steal $p from $N, but fails!", "", ToNotVict)
				victim.SetPosition(combat.GetPositionFromHP(victim.GetHP(), victim.GetPosition()))
				if victim.GetPosition() != combat.PosSleeping {
					ohoh = true
				}
			} else if mobileCanCarryStolen(mob, item) {
				if err := ctx.World.MoveObjectToMobInventoryFront(item, mob); err != nil {
					slog.Error("switched-body steal equipment transfer failed", "item", item.GetVNum(), "thief", mob.GetVNum(), "error", err)
				} else {
					applyRobbedAffect(victim)
					if !victim.IsNPC() {
						MudLog(fmt.Sprintf("(PS) %s stole %s from %s.", mob.GetName(), item.GetShortDesc(), victim.GetName()), MudlogComplete, LVL_IMMORT, true)
					}
					ActWithPrivateSink(ctx.World, false, mob, nil, item, nil,
						"You unequip $p and steal it.", "", ToChar, mob, ctx.PrivateOutput)
					Act(ctx.World, true, mob, asActor(victim), item, nil,
						"$n steals $p from $N.", "", ToNotVict)
				}
			} else if mobCarriedItemCount(mob)+1 >= mobMaxCarryCount(mob) {
				mobileSend(ctx, mob, "You cannot carry that much.\r\n")
			}
		} else {
			ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
				"$E hasn't got that item.", "", ToChar, mob, ctx.PrivateOutput)
			return CharacterCommandHandled
		}
	} else if percent > ctx.World.DummyMobSkill(skillNumSteal) {
		ohoh = true
		mobileSend(ctx, mob, "Oops..\r\n")
		Act(ctx.World, false, mob, asActor(victim), nil, nil,
			"You discover that $n has $s hands in your wallet.", "", ToVict)
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$n tries to steal gold from $N.", "", ToNotVict)
	} else {
		gold := min(1782, targetGold(victim)*dprng.Number(1, 10)/100)
		if gold > 0 && spendTargetGold(victim, gold) {
			combatAddGold(mob, gold)
			if gold > 1 {
				mobileSend(ctx, mob, fmt.Sprintf("Bingo!  You got %d gold coins.\r\n", gold))
			} else {
				mobileSend(ctx, mob, "You manage to swipe a solitary gold coin.\r\n")
			}
		} else {
			mobileSend(ctx, mob, "You couldn't get any gold...\r\n")
		}
	}

	if ohoh && victim.IsNPC() && victim.GetPosition() > combat.PosSleeping {
		if targetMob, ok := victim.(*MobInstance); ok {
			if err := ctx.World.mobHit(targetMob, mob); err != nil {
				slog.Warn("switched-body steal retaliation failed", "victim", targetMob.GetName(), "error", err)
			}
		}
	}
	if ohoh && !victim.IsNPC() && (!ctx.DirectCall || ctx.Subcmd == 0) {
		MudLog(fmt.Sprintf("(PS) %s unsuccessfuly tried to steal from %s.", mob.GetName(), victim.GetName()), MudlogComplete, LVL_IMMORT, true)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	return CharacterCommandHandled
}

func mobileStealCarriedItem(thief *MobInstance, victim combat.Combatant, name string) *ObjectInstance {
	var items []*ObjectInstance
	switch body := victim.(type) {
	case *Player:
		if body.Inventory != nil {
			items = body.Inventory.FindItems("")
		}
	case *MobInstance:
		items = body.InventorySnapshot()
	}
	for _, item := range items {
		if item != nil && isnameWithAbbrevs(name, item.GetKeywords()) && CanSeeObject(thief, item) {
			return item
		}
	}
	return nil
}

func mobileStealEquippedItem(thief *MobInstance, victim combat.Combatant, name string) (*ObjectInstance, int) {
	switch body := victim.(type) {
	case *Player:
		for _, item := range COrderedWorn(body) {
			if item != nil && cIsName(name, item.GetKeywords()) && CanSeeObject(thief, item) {
				return item, int(item.Location.Slot)
			}
		}
	case *MobInstance:
		body.mu.RLock()
		defer body.mu.RUnlock()
		for slot := 0; slot < NumWears; slot++ {
			item := body.Equipment[slot]
			if item != nil && cIsName(name, item.GetKeywords()) && CanSeeObject(thief, item) {
				return item, slot
			}
		}
	}
	return nil, 0
}

func mobileCanCarryStolen(thief *MobInstance, item *ObjectInstance) bool {
	return mobCarriedItemCount(thief)+1 < mobMaxCarryCount(thief) &&
		mobCarriedWeight(thief)+item.GetWeight() < mobMaxCarryWeight(thief)
}
