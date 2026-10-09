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
// their locks. The saved-equipment entry counterpart is audited separately
// by DP-1419 (after DP-1404); reconstruction has no attached world yet.
func (p *Player) checkEquipmentStats() {
	p.mu.RLock()
	w := p.worldRef
	stats := p.effectiveAttributesLocked()
	p.mu.RUnlock()
	if w != nil {
		w.applyBadStat(p, badStat(stats))
	}
}

func (w *World) applyBadStat(body combat.Combatant, bad byte) {
	var text string
	switch bad {
	case 's':
		text = "You are too weak to fight!\n\r"
	case 'i', 'w':
		text = "You are too dumb to do much of anything!\n\r"
	case 'c':
		text = "The world hates you!\n\r"
	case 'd':
		text = "You trip over your own feet and hit your head!\n\r"
	}
	if mob, ok := body.(*MobInstance); ok {
		if text != "" && w.MobileMessageSink != nil {
			w.MobileMessageSink(mob, []byte(text))
		}
	} else if text != "" {
		body.SendMessage(text)
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
