package game

// affect_update.go — periodic affect processing
//
// Ported from src/magic.c affect_update() lines 429-461.
// Called each mud hour to decrement affect durations
// and remove expired affects with wear-off messages.

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// SpellWearOffMsg returns the wear-off message for a given spell type.
// Source: src/spells.c spell_wear_off_msg[].
// Extended list matching CircleMUD original messages.
// wearOffMessages maps spell type index to wear-off message.
var wearOffMessages = map[int]string{}

func init() {
	for i, msg := range SpellWearOffMessages {
		wearOffMessages[i] = msg
	}
}

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
		p.mu.Unlock()
	}

	// Mob affect expiration — magic.c:431-457
	// C iterates character_list which includes both players and NPCs.
	mobs := w.GetAllMobs()
	for _, mob := range mobs {
		if !mob.IsAlive() {
			continue
		}

		mob.mu.Lock()
		type expireInfo struct {
			spellNum int
			wearOff  string
		}
		var expired []expireInfo
		roomVNum := mob.RoomVNum
		shortDesc := mob.GetShortDesc()
		for key, val := range mob.CustomData {
			if !strings.HasPrefix(key, "affect_") {
				continue
			}
			aff, ok := val.(*engine.Affect)
			if !ok {
				continue
			}
			if aff.Duration == -1 {
				continue // permanent affect
			}
			if aff.Duration >= 1 {
				aff.Duration--
				continue
			}
			// Duration == 0 — expires this tick.
			var spellNum int
			if _, err := fmt.Sscanf(key, "affect_%d", &spellNum); err == nil {
				expired = append(expired, expireInfo{spellNum: spellNum, wearOff: SpellWearOffMsg(aff.SpellID)})
			}
		}
		mob.mu.Unlock()

		// Send wear-off messages and remove affects outside the mob lock.
		// RemoveAffectBySpell acquires mob.mu internally via RemoveAffected.
		for _, info := range expired {
			if info.wearOff != "" {
				w.roomMessage(roomVNum, fmt.Sprintf("%s %s", shortDesc, info.wearOff))
			}
			mob.RemoveAffectBySpell(info.spellNum)
		}
	}
}
