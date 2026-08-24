package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

const (
	considerNotFound = "Consider killing who?\r\n"
	considerSelf     = "Easy!  Very easy indeed!\r\n"
)

// DoConsider compares ch with a visible character in the same room.
// Source: act.informative.c do_consider() lines 2330-2431.
func (w *World) DoConsider(ch *Player, argument string) {
	if ch == nil {
		return
	}

	name, _ := oneArgument(argument) // C do_consider: one_argument (fill-skip, lowercase)
	if name == "" {
		ch.SendMessage(considerNotFound)
		return
	}
	if strings.EqualFold(name, "self") || strings.EqualFold(name, "me") || isnameWithAbbrevs(name, ch.GetName()) {
		ch.SendMessage(considerSelf)
		return
	}

	target, found := w.ResolveCharInRoom(ch, name)
	if !found || target.Combatant == nil {
		ch.SendMessage(considerNotFound)
		return
	}
	if target.Combatant == ch {
		ch.SendMessage(considerSelf)
		return
	}

	victim := asActor(target.Combatant)
	if victim == nil {
		ch.SendMessage(considerNotFound)
		return
	}

	// Preserve C's RNG draw order: the evaluator's damage roll is sampled
	// before the victim's.
	characterDamage := considerDamage(ch)
	victimDamage := considerDamage(target.Combatant)
	damdiff := victimDamage - characterDamage
	hitdiff := target.Combatant.GetHP() - ch.GetHP()
	leveldiff := target.Combatant.GetLevel() - ch.GetLevel()
	message := considerDamageAssessment(damdiff) +
		considerHealthAssessment(hitdiff, ch.GetHP()) +
		considerLevelAssessment(leveldiff, ch.GetLevel())

	// C emits the completed evaluation once, to the evaluating character only.
	// Act owns $N/$E/$S expansion so Player's 0=male, 1=female, 2=neutral sex
	// encoding follows the same pronoun path as every other act() message.
	Act(w, true, ch, victim, nil, nil, message, "", ToChar)
}

func considerDamage(character combat.Combatant) int {
	damage := combat.StrAppToDam(character) + character.GetDamroll()
	roller := combat.GetRoller()

	if character.IsNPC() {
		damageRoll := character.GetDamageRoll()
		return damage + roller.Dice(damageRoll.Num, damageRoll.Sides) + damageRoll.Plus
	}

	if player, ok := character.(*Player); ok && player.Equipment != nil {
		if weapon, wielded := player.Equipment.GetItemInSlot(SlotWield); wielded && weapon != nil {
			if weapon.Prototype != nil {
				damage += roller.Dice(weapon.Prototype.Values[1], weapon.Prototype.Values[2])
			}
			return damage
		}
	}

	return damage + roller.Number(0, character.GetLevel()/3)
}

func considerDamageAssessment(damdiff int) string {
	switch {
	case damdiff > 20:
		return "$N looks like $E could eat you for lunch, "
	case damdiff > 10:
		return "$N looks like $E could tear you up in a fight, "
	case damdiff > 5:
		return "$N looks like $E could hurt you in a fight, "
	case damdiff > -3:
		return "$N looks like a fair fight, "
	case damdiff > -5:
		return "$N looks like an easy kill, "
	case damdiff > -10:
		return "$N looks like a very easy kill, "
	default:
		return "$N might not even be worth the effort to kill, "
	}
}

func considerHealthAssessment(hitdiff, characterHP int) string {
	switch {
	case hitdiff > 4*characterHP:
		return "looks to be\r\nin much better physical shape than you, "
	case hitdiff > 2*characterHP:
		return "looks to be\r\nin a lot better physical shape than you, "
	case hitdiff > characterHP:
		return "looks to be\r\nin better physical shape than you, "
	case hitdiff >= 0:
		return "looks to be\r\nin about the same physical shape as you, "
	case hitdiff > -25:
		return "looks to be\r\nin a little worse physical shape as you, "
	case hitdiff > -50:
		return "looks to be\r\nin a worse physical shape than you, "
	default:
		return "looks to be\r\nin a lot worse physical shape than you, "
	}
}

func considerLevelAssessment(leveldiff, characterLevel int) string {
	switch {
	case leveldiff > characterLevel:
		return "and moves with an ease telling\r\nof many won battles."
	case leveldiff > characterLevel/2:
		return "and seems to know $S opponent."
	case leveldiff > -1:
		return "and seems about as confident in\r\nbattle as you do."
	case leveldiff > -3:
		return "and seems less confident in\r\nbattle than you do."
	case leveldiff > -5:
		return "and seems much less confident in\r\nbattle than you do."
	case leveldiff > -7:
		return "and seems ready to run from a\r\nfight."
	default:
		return "and seems like $E's never been\r\nin battle before."
	}
}
