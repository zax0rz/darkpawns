package game

import (
	"fmt"
	"log/slog"
)

func (w *World) CheckIdling(p *Player) {
	if p == nil || p.IsNPC() {
		return // point_update calls check_idling for PCs only (limits.c:521-525)
	}

	p.mu.Lock()
	p.IdleTimer++
	timer := p.IdleTimer
	level := p.Level
	p.mu.Unlock()

	// C increments the timer for immortals too and gates only the
	// void/disconnect on level (limits.c:424-425) — the users idle column
	// ticks for everyone.
	if level >= LVL_IMMORT {
		return
	}

	if timer > IDLE_TO_VOID {
		p.mu.Lock()
		wasIn := p.WasInRoom
		roomVNum := p.RoomVNum
		fighting := p.Fighting
		p.mu.Unlock()

		if wasIn == 0 && roomVNum > 0 {
			// First idle threshold — pull to void room (vnum 1).
			p.mu.Lock()
			p.WasInRoom = roomVNum
			p.mu.Unlock()

			// C tears down both sides through stop_fighting before moving the
			// idler (limits.c:428-431). Stop the engine pair as well as the
			// character fields; otherwise the round list can retain a combatant
			// who has already moved to room 1.
			if fighting != "" {
				if stopper, ok := w.combatEngine.(interface{ StopCombat(string) }); ok {
					stopper.StopCombat(p.Name)
				} else {
					p.StopFighting()
				}
			}

			// C acts before the transfer, while the player still stands in
			// the room: act("$n disappears into the void.", TRUE, ch, 0, 0,
			// TO_ROOM) (limits.c:432-437). Act excludes the actor and gates
			// hidden viewers; SendToRoom-by-vnum would hand the actor their
			// own line whenever the transfer lands first.
			Act(w, true, p, nil, nil, nil, "$n disappears into the void.", "", ToRoom)
			p.SendMessage("You have been idle, and are pulled into a void.\r\n")
			// C saves before the transfer, while the character still stands
			// in the room: save_char(ch, NOWHERE) + Crash_crashsave
			// (limits.c:434-435, DP-1353).
			if res := w.SavePlayerRecord(p, "void pull", LoadRoomNowhere, SaveCrash); res == SaveFailed {
				slog.Error("store-of-record save failed on void pull", "player", p.Name)
			}
			if err := w.PlayerTransfer(p, 1); err != nil {
				slog.Warn("PlayerTransfer failed in idle check", "player", p.Name, "error", err)
			}
		} else if timer > IDLE_DISCONNECT {
			// Second threshold — limits.c:438-451: char_to_room(3),
			// close_socket + desc = NULL, free_rent's Crash_rentsave(ch, 0),
			// mudlog, extract_char. free_rent is YES (src/config.c:106), so
			// the object pass is the legal-quit rent pass: norent objects are
			// destroyed, the rent file keeps the rest, and extraction does
			// not drop anything in world[3].
			p.mu.Lock()
			p.WasInRoom = 0
			p.IdleDisconnect = true
			p.mu.Unlock()

			// char_to_room takes an RNUM, not a VNUM (src/limits.c:441).
			// Translate through C's vnum-ordered world table (src/db.c:3083-3097).
			if destination, ok := w.RoomVNumByIndex(3); ok {
				if err := w.PlayerTransfer(p, destination); err != nil {
					slog.Warn("PlayerTransfer failed in idle disconnect", "player", p.Name, "error", err)
				}
			} else {
				slog.Error("idle disconnect room index missing", "index", 3, "player", p.Name)
			}

			// src/limits.c:442-444 closes only the attached descriptor here,
			// before rent/extraction. The callback enters with no world/player lock.
			if w.IdleCloseDescriptor != nil {
				w.IdleCloseDescriptor(p)
			} else {
				Act(w, true, p, nil, nil, nil, "$n has lost $s link.", "", ToRoom)
				MudLog(fmt.Sprintf("Closing link to: %s.", p.Name), MudlogNormal, max(LVL_IMMORT, p.GetInvisLevel()), true)
			}

			w.RentOut(p)
			p.RentedOut = true

			MudLog(fmt.Sprintf("%s force-rented and extracted (idle).", p.Name), MudlogComplete, LVL_GOD, true) // limits.c:449-450
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
