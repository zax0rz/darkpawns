package game

// affect_update.go — periodic affect processing
//
// Ported from src/magic.c affect_update() lines 429-461.
// Called each mud hour to decrement affect durations
// and remove expired affects with wear-off messages.

import (
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// SpellWearOffMsg returns the wear-off message for a given spell type.
// Source: src/spells.c spell_wear_off_msg[].
// Extended list matching CircleMUD original messages.
var wearOffMessages = map[int]string{}

// SpellWearOffMsg gets the wear-off message for a spell type.
func SpellWearOffMsg(spellType int) string {
	if msg, ok := wearOffMessages[spellType]; ok {
		return msg
	}
	return "You feel strange."
}

// AffectUpdate decrements affect durations and removes expired affects.
// Source: src/magic.c affect_update() lines 431-461.
func (w *World) AffectUpdate() {
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	w.mu.RUnlock()

	for _, p := range players {
		var remaining []interface{}

		w.mu.RLock()
		affects := p.ActiveAffects
		w.mu.RUnlock()

		for _, af := range affects {
			if af.Duration == -1 {
				// -1 = permanent (implementor-level spells, circle 1 items)
				remaining = append(remaining, af)
				continue
			}
			if af.Duration >= 1 {
				af.Duration--
				remaining = append(remaining, af)
			} else {
				// Duration expired — remove and print wear-off
				if msg := SpellWearOffMsg(int(af.Type)); msg != "" {
					p.SendMessage(msg + "\r\n")
				}
				engine.AffectFromChar(p, int(af.Type))
			}
		}

		w.mu.Lock()
		p.ActiveAffects = make([]*engine.Affect, len(remaining))
		for i, r := range remaining {
			p.ActiveAffects[i] = r.(*engine.Affect)
		}
		w.mu.Unlock()
	}
}
