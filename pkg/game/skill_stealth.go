package game

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoSneak(ch *Player) SkillResult {
	if ch.GetSkill(SkillSneak) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Toggle off if already sneaking
	if ch.IsAffected(affSneak) {
		ch.SetAffect(affSneak, false)
		return SkillResult{Success: true, MessageToCh: "You stop sneaking."}
	}

	// Roll for success: percent = number(1,101)
	// prob = GET_SKILL(ch, SKILL_SNEAK) + dex_app_skill[GET_DEX(ch)].sneak
	// We don't have dex_app_skill table yet, use raw skill
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1
	prob := ch.GetSkill(SkillSneak)

	if percent > prob {
		return SkillResult{Success: false, MessageToCh: "You attempt to move silently, but make too much noise."}
	}

	ch.SetAffect(affSneak, true)
	return SkillResult{Success: true, MessageToCh: "Okay, you'll try to move silently for a while."}
}

// DoHide implements do_hide() from act.other.c lines 247-307.
func DoHide(ch *Player) SkillResult {
	if ch.GetSkill(SkillHide) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Toggle off if already hidden
	if ch.IsAffected(affHide) {
		ch.SetAffect(affHide, false)
		return SkillResult{Success: true, MessageToCh: "You step out of the shadows."}
	}

	// Roll for success
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1
	prob := ch.GetSkill(SkillHide)

	if percent > prob {
		return SkillResult{Success: false, MessageToCh: "You attempt to hide yourself, but fail."}
	}

	ch.SetAffect(affHide, true)
	return SkillResult{Success: true, MessageToCh: "You blend into the shadows."}
}

// DoSteal implements do_steal() from act.other.c lines 309-560.
// Simplified: steal gold or an item from target's inventory.
func DoSteal(ch *Player, target combat.Combatant, itemName string) SkillResult {
	if ch.GetSkill(SkillSteal) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Can't steal from yourself
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Come on now, that's rather stupid!"}
	}

	// Target can't be fighting
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't steal from someone who's fighting!"}
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// Steal gold
	if itemName == "coins" || itemName == "gold" {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		percent := rand.IntN(101) + 1
		prob := ch.GetSkill(SkillSteal)

		if percent > prob {
			return SkillResult{
				Success:       false,
				MessageToCh:   "Oops..",
				MessageToVict: ActMessage("You discover that $n has $s hands in your wallet.", chPronouns, &victPronouns, ""),
				MessageToRoom: ActMessage("$n tries to steal gold from $N.", chPronouns, &victPronouns, ""),
			}
		}

		// Calculate gold stolen: (GET_GOLD(vict) * number(1,10)) / 100, max 1782
		// We need access to target's gold — for players we can cast, for mobs we estimate
		var gold int
		if p, ok := target.(*Player); ok {
			// #nosec G404 — game RNG, not cryptographic
			// #nosec G404
			gold = (p.GetGold() * (rand.IntN(10) + 1)) / 100
			if gold > 1782 {
				gold = 1782
			}
			if gold > p.GetGold() {
				gold = p.GetGold()
			}
			p.SetGold(p.GetGold() - gold)
			ch.SetGold(ch.GetGold() + gold)
		} else {
			// Mob — steal small random amount
			// #nosec G404 — game RNG, not cryptographic
			// #nosec G404
			gold = rand.IntN(20) + 1
			ch.SetGold(ch.GetGold() + gold)
		}

		if gold > 1 {
			return SkillResult{
				Success:     true,
				MessageToCh: fmt.Sprintf("Bingo!  You got %d gold coins.", gold),
			}
		} else if gold == 1 {
			return SkillResult{Success: true, MessageToCh: "You manage to swipe a solitary gold coin."}
		}
		return SkillResult{Success: true, MessageToCh: "You couldn't get any gold..."}
	}

	// Steal item
	var item *ObjectInstance
	var found bool

	if p, ok := target.(*Player); ok {
		item, found = p.Inventory.FindItem(itemName)
		if !found {
			return SkillResult{Success: false, MessageToCh: ActMessage("$E hasn't got that item.", chPronouns, &victPronouns, "")}
		}
	} else if m, ok := target.(*MobInstance); ok {
		lowerName := strings.ToLower(itemName)
		for _, mobItem := range m.Inventory {
			if mobItem != nil {
				if strings.Contains(strings.ToLower(mobItem.GetKeywords()), lowerName) || strings.Contains(strings.ToLower(mobItem.GetShortDesc()), lowerName) {
					item = mobItem
					found = true
					break
				}
			}
		}
		if !found {
			return SkillResult{Success: false, MessageToCh: ActMessage("$E hasn't got that item.", chPronouns, &victPronouns, "")}
		}
	} else {
		return SkillResult{Success: false, MessageToCh: "You can't steal that."}
	}

	// Roll with weight penalty
	// #nosec G404 — game RNG, not cryptographic
	percent := rand.IntN(101) + 1
	percent += item.GetWeight()
	if target.GetLevel() > ch.GetLevel() {
		percent += target.GetLevel() - ch.GetLevel()
	}
	prob := ch.GetSkill(SkillSteal)

	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("$N catches you trying to steal something...", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tried to steal something from you!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to steal something from $N.", chPronouns, &victPronouns, ""),
		}
	}

	// Steal the item
	if p, ok := target.(*Player); ok {
		p.Inventory.removeItem(item)
	} else if m, ok := target.(*MobInstance); ok {
		m.RemoveFromInventory(item)
	}

	if err := ch.Inventory.addItem(item); err != nil {
		// Put back
		if p, ok := target.(*Player); ok {
			if err := p.Inventory.addItem(item); err != nil {
				slog.Error("DoSteal rollback failed", "player", p.Name, "item", item.GetShortDesc(), "error", err)
			}
		} else if m, ok := target.(*MobInstance); ok {
			m.AddToInventory(item)
		}
		return SkillResult{
			Success:     false,
			MessageToCh: ActMessage("You can't carry that much!\r\n", chPronouns, nil, ""),
		}
	}

	return SkillResult{
		Success:       true,
		MessageToCh:   ActMessage("You deftly steal $p from $N's pocket!", chPronouns, &victPronouns, item.GetShortDesc()),
		MessageToVict: "",
		MessageToRoom: "",
	}
}

// DoCarve implements do_carve() — carve food from a corpse.
