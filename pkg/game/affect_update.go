package game

// affect_update.go — periodic affect processing
//
// Ported from src/magic.c affect_update() lines 429-461.
// Called each mud hour to decrement affect durations
// and remove expired affects with wear-off messages.

import (
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// SpellWearOffMsg returns the C wear-off message for a one-based spell number.
// C's spell_wear_off_msg[0] is reserved, so Go entry n-1 represents spell n.
var wearOffMessages = map[int]string{}

func init() {
	for i, msg := range SpellWearOffMessages {
		wearOffMessages[i+1] = msg
	}
}

// SpellWearOffMsg gets the wear-off message for a spell type.
func SpellWearOffMsg(spellType int) string {
	if spellType <= 0 {
		return ""
	}
	if msg, ok := wearOffMessages[spellType]; ok {
		return msg
	}
	return ""
}

// AffectUpdate decrements affect durations and removes expired affects.
// Source: src/magic.c affect_update() lines 431-461.
func (w *World) AffectUpdate() {
	for _, character := range w.cOrderedCharacters() {
		if p := character.player; p != nil {
			// Snapshot the player's current affects under p.mu (not w.mu).
			// ActiveAffects belongs to the Player, not the World.
			p.mu.RLock()
			affects := make([]*engine.Affect, len(p.ActiveAffects))
			copy(affects, p.ActiveAffects)
			p.mu.RUnlock()

			var remaining []*engine.Affect

			for _, af := range affects {
				if af.Duration == -1 {
					// Permanent affect (immortal only). Matches C: duration == -1 → no action.
					remaining = append(remaining, af)
					continue
				}
				if af.Duration >= 1 {
					// Active spell — decrement. Matches C: duration >= 1 → duration--.
					af.Duration--
					remaining = append(remaining, af)
					continue
				}
				// Duration == 0 (or unexpected negative) — expires this tick.
				// Matches C: else branch → affect_remove.
				if msg := SpellWearOffMsg(af.SpellID); msg != "" {
					p.SendMessage(msg + "\r\n")
				}
				engine.AffectFromChar(p, af.SpellID)
			}

			p.mu.Lock()
			p.ActiveAffects = remaining
			if len(remaining) != len(affects) {
				p.affectTotalLocked()
			}
			p.mu.Unlock()
			if len(remaining) != len(affects) {
				p.refreshAttributeCapacity()
			}
			continue
		}

		// Mob affect expiration — src/magic.c:433-455.
		// C iterates character_list, which includes both players and NPCs.
		mob := character.mob
		if mob == nil {
			continue
		}
		if !mob.IsAlive() {
			continue
		}

		// C walks ch->affected newest-first (src/magic.c:435); OrderedAffects is
		// that order. Each expired record is removed, and C sends the wear-off to
		// the character itself (send_to_char): a descriptorless mobile is silent,
		// while a mobile a wizard has switched onto delivers to that wizard's
		// descriptor (MobInstance.SendMessage; src/act.wizard.c do_switch sets
		// victim->desc = the wizard's descriptor).
		affects := mob.OrderedAffects()
		for i, af := range affects {
			switch {
			case af.Duration == -1:
				// Permanent affect (immortal only): no action.
			case af.Duration >= 1:
				af.Duration--
			default:
				// Expired this tick. C sends the wear-off only for the last record
				// of its spell (src/magic.c:439-444): no next record, a different
				// spell, or a next record still alive. A multi-record spell thus
				// speaks once, from its oldest expiring record.
				var next *engine.Affect
				if i+1 < len(affects) {
					next = affects[i+1]
				}
				if next == nil || next.SpellID != af.SpellID || next.Duration > 0 {
					if msg := SpellWearOffMsg(af.SpellID); msg != "" {
						mob.SendMessage(msg + "\r\n")
					}
				}
				mob.RemoveAffectBySpell(af.SpellID)
			}
		}
	}
}
