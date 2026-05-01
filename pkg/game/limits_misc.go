package game

import (
	"fmt"
	"log/slog"
)

func (w *World) CheckIdling(p *Player) {
	if p == nil {
		return
	}

	p.mu.RLock()
	level := p.Level
	p.mu.RUnlock()

	if level >= LVL_IMMORT || p.IsNPC() {
		return
	}

	p.mu.Lock()
	p.IdleTimer++
	timer := p.IdleTimer
	p.mu.Unlock()

	if timer > IDLE_TO_VOID {
		p.mu.Lock()
		wasIn := p.WasInRoom
		roomVNum := p.RoomVNum
		fighting := p.Fighting
		p.mu.Unlock()

		if wasIn == 0 && roomVNum > 0 {
			// First idle threshold — pull to void room (vnum 1)
			p.mu.Lock()
			p.WasInRoom = roomVNum
			if fighting != "" {
					p.Fighting = ""
			}
			p.mu.Unlock()

			if err := w.PlayerTransfer(p, 1); err != nil {
				slog.Warn("PlayerTransfer failed in idle check", "player", p.Name, "error", err)
			}
			p.SendMessage("You have been idle, and are pulled into a void.\r\n")
			w.SendToRoom(roomVNum, fmt.Sprintf("%s disappears into the void.\r\n", p.Name))
		} else if timer > IDLE_DISCONNECT {
			// Second threshold — force rent and disconnect
			p.mu.Lock()
			p.WasInRoom = 0
			p.mu.Unlock()

			if err := w.PlayerTransfer(p, 3); err != nil {
				slog.Warn("PlayerTransfer failed in idle disconnect", "player", p.Name, "error", err)
			}

			slog.Info("player idle extracted", "name", p.Name)
			ExtractChar(p)
		}
	}
}

// sumEquipAffect sums equipment affect modifiers for a given location.
// If requireSleeping is true, positive modifiers are only counted when sleeping.
// Negative modifiers always apply (matching limits.c behavior).
// Source: limits.c:89-95, 156-162, 224-230
func (p *Player) sumEquipAffect(location int, requireSleeping bool) int {
	if p.Equipment == nil {
		return 0
	}
	total := 0
	for _, item := range p.Equipment.GetEquippedItems() {
		if item == nil || item.Prototype == nil {
			continue
		}
		for _, af := range item.Prototype.Affects {
			if af.Location != location {
				continue
			}
			if requireSleeping && af.Modifier > 0 {
				continue
			}
			total += af.Modifier
		}
	}
	return total
}

// isFighting returns true if the player is currently in combat.
// Equivalent to C's FIGHTING(ch) macro.
func isFighting(p *Player) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Fighting != ""
}



