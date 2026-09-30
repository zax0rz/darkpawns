package game

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// removeRawKillAffects follows the live spell-removal path; stat modifiers are
// computed from ActiveAffects, so removal also restores the unmodified stats.
// C: src/fight.c:544-545; src/handler.c:428-437. No wear-off message is sent.
func (p *Player) removeRawKillAffects() {
	p.mu.RLock()
	affects := append([]*engine.Affect(nil), p.ActiveAffects...)
	p.mu.RUnlock()
	for _, af := range affects {
		if af != nil {
			for flag, bit := range EngineFlagToAffBit {
				if af.Flags&flag != 0 {
					p.RemoveAffectBit(bit)
				}
			}
			p.RemoveAffectBySpell(af.SpellID)
		}
	}
	for _, af := range append([]*engine.MasterAffect(nil), p.GetMasterAffects()...) {
		// Legacy master records still need their real removal path.
		//nolint:staticcheck // ActiveAffects use RemoveAffectBySpell above.
		engine.AffectRemove(p, af)
	}
}

func (m *MobInstance) removeRawKillAffects() {
	var spells []int
	m.mu.RLock()
	for _, value := range m.CustomData {
		if af, ok := value.(*engine.Affect); ok {
			spells = append(spells, af.SpellID)
		}
	}
	m.mu.RUnlock()
	for _, spell := range spells {
		m.RemoveAffectBySpell(spell)
	}
}

// C: src/fight.c:547-552. TattooAf is the existing tattoo stat-removal path.
func removeRawKillTattoo(p *Player) {
	if p.Tattoo != TattooNone {
		TattooAf(p, false)
		p.Tattoo = TattooNone
		p.TatTimer = 0
	}
}

// C: src/fight.c:554-561. Only vampires have their excess mana clamped.
func clearRawKillNightbreed(p *Player) {
	p.SetAffect(affWerewolf, false)
	if p.IsAffected(affVampire) {
		p.SetAffect(affVampire, false)
		if p.GetMana() > p.GetMaxMana() {
			p.SetMana(p.GetMaxMana())
		}
	}
}

// C: src/fight.c:566-572. Memory is gated by MOB_MEMORY; hunting is not.
func (w *World) forgetRawKillVictim(name string) {
	for _, mob := range w.GetAllMobs() {
		if mob.HasMobFlag(MobFlagMemory) {
			mob.Forget(name)
		}
		if mob.GetHunting() == name {
			mob.ClearHunting()
		}
	}
}

// makeRawKillBody shares the proven corpse/dust builders with handlePlayerDeath.
// C: src/fight.c:575-580 and make_corpse's disintegrate gate at :267-270.
func (w *World) makeRawKillBody(p *Player, attackType int, dust bool) {
	wasCrash := p.NeedsCrashSave()
	inventory := p.Inventory.FindItems("")
	var equipment []*ObjectInstance
	for pos := 0; pos < NumWears; pos++ {
		if slot, ok := CWearPosToSlot(pos); ok {
			if obj, found := p.Equipment.GetItemInSlot(slot); found {
				equipment = append(equipment, obj)
			}
		}
	}
	gold := p.GetGold()
	if dust || attackType == 93 {
		w.makeDust(p, inventory, equipment, p.GetRoom(), gold, attackType)
	} else {
		corpse := w.makeCorpse(p.GetName(), p.GetSex(), inventory, equipment, p.GetRoom(), attackType, gold, false)
		if err := w.MoveObjectToRoomFront(corpse, p.GetRoom()); err != nil {
			slog.Error("place RawKill corpse", "player", p.GetName(), "error", err)
		}
	}
	p.SetGold(0)
	// C obj_to_obj does not mark PLR_CRASH. Dust uses obj_from_char and does.
	if !dust && attackType != 93 {
		p.RestoreCrashFlagAfterCorpse(wasCrash)
	}
}

func rawKillRace(victim interface{}) int {
	switch v := victim.(type) {
	case *Player:
		return v.GetRace()
	case *MobInstance:
		if proto := v.Proto(); proto != nil {
			return proto.Race
		}
	}
	return 0
}
