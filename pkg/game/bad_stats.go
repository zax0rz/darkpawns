package game

import (
	"strconv"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// badStat preserves the last-zero-wins STR/INT/WIS/CHA/DEX order of
// src/handler.c:640-675; CON is deliberately absent.
func badStat(stats CharStats) byte {
	var bad byte
	for _, entry := range []struct {
		value int
		kind  byte
	}{{stats.Str, 's'}, {stats.Int, 'i'}, {stats.Wis, 'w'}, {stats.Cha, 'c'}, {stats.Dex, 'd'}} {
		if entry.value == 0 {
			bad = entry.kind
		}
	}
	return bad
}

// checkEquipmentStats runs after equipment and affect_total have released
// their locks. The saved-equipment entry counterpart is DP-1419's
// ReplayCrashLoadBadStats, which needs no attached world reference.
func (p *Player) checkEquipmentStats() {
	p.mu.RLock()
	w := p.worldRef
	stats := p.effectiveAttributesLocked()
	p.mu.RUnlock()
	if w != nil {
		w.applyBadStat(p, badStat(stats))
	}
}

// badStatLine is check_for_bad_stats' player-facing text for the selected
// stat, with C's LFCR terminators (src/handler.c:652-672). A zero bad byte
// (no zeroed stat) selects nothing.
func badStatLine(bad byte) string {
	switch bad {
	case 's':
		return "You are too weak to fight!\n\r"
	case 'i', 'w':
		return "You are too dumb to do much of anything!\n\r"
	case 'c':
		return "The world hates you!\n\r"
	case 'd':
		return "You trip over your own feet and hit your head!\n\r"
	}
	return ""
}

// ReplayCrashLoadBadStats ports Crash_load's per-item equip checks. C loads a
// crash file object by object and calls auto_equip, which calls equip_char and
// therefore check_for_bad_stats for every item it wears (src/objsave.c:557;
// src/handler.c:749). equip_char's affect_total rebuilds aff_abils from the
// slots filled SO FAR (src/handler.c:337-372), so a stat already at zero fires
// again on every later item: a zero DEX helm in an early slot costs 40 HP per
// remaining worn item, not once.
//
// The replay walks the restored worn items in ascending C wear position, which
// is the order Crash_crashsave wrote them (src/objsave.c:807-812), computes
// aff_abils over that prefix, and runs the shared badStat/applyBadStat pair.
//
// deliver receives each player-facing line in C's order. The entry caller has
// no world-registered descriptor yet — C's send_to_char writes ch->desc
// directly (src/interpreter.c:2184-2189) — so it supplies the session's own
// transport; nil keeps the live path's body.SendMessage.
func (w *World) ReplayCrashLoadBadStats(p *Player, deliver func(string)) {
	if p == nil {
		return
	}
	// C's reset_char leaves ch->in_room NOWHERE for the whole of Crash_load
	// (src/db.c:2945; src/interpreter.c:2174-2184), so damage()'s room act()
	// finds no audience and logs C's own SYSERR (src/comm.c:2519-2526). The
	// restored record carries the load room, so hold the room at NOWHERE for
	// the replay and give it back for the entry save that follows.
	room := p.GetRoom()
	p.SetRoom(roomNowhere)
	defer p.SetRoom(room)

	items := COrderedWorn(p)
	for i := range items {
		p.mu.RLock()
		stats := p.affectTotalForItemsLocked(items[:i+1])
		p.mu.RUnlock()
		w.applyBadStatDeliver(p, badStat(stats), deliver)
	}
}

func (w *World) applyBadStat(body combat.Combatant, bad byte) {
	w.applyBadStatDeliver(body, bad, nil)
}

// applyBadStatDeliver is applyBadStat with an explicit player-text seam: see
// ReplayCrashLoadBadStats for why the entry path delivers its own bytes.
func (w *World) applyBadStatDeliver(body combat.Combatant, bad byte, deliver func(string)) {
	text := badStatLine(bad)
	if mob, ok := body.(*MobInstance); ok {
		if text != "" && w.MobileMessageSink != nil {
			w.MobileMessageSink(mob, []byte(text))
		}
	} else if text != "" {
		if deliver != nil {
			deliver(text)
		} else {
			body.SendMessage(text)
		}
	}
	switch bad {
	case 'c':
		var pest *MobInstance
		for _, candidate := range w.GetAllMobs() {
			if candidate.GetVNum() == 7907 && (pest == nil || candidate.ID > pest.ID) {
				pest = candidate
			}
		}
		if pest != nil && pest.GetHunting() == "" {
			if prey, ok := body.(*Player); ok {
				if prey.GetID() <= 0 {
					return
				}
				LogHuntingStart(pest, prey)
			}
			pest.SetHunting(body.GetName())
			if prey, ok := body.(*Player); ok {
				pest.SetHuntingID(strconv.Itoa(prey.GetID()))
			}
			if prey, ok := body.(*MobInstance); ok {
				pest.mu.Lock()
				pest.HuntingMobID = prey.ID
				pest.mu.Unlock()
			}
		}
	case 'd':
		w.selfDamage(body, 40, combat.TYPE_SUFFERING)
	}
}
