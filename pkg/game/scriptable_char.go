package game

import "github.com/zax0rz/darkpawns/pkg/scripting"

// scriptable_char.go — the ScriptableChar / ScriptablePlayer / ScriptableMob
// views the legacy (non-bridge) Lua char-table builder needs. Every accessor
// mirrors the C macro the bridge's CharFields reads, so the legacy builder and
// the bridge cannot drift (R1/R4/R5e).
//
// Production never reaches the legacy builder: RunScript bridges every server
// run (see RunScript's bridge comment). These accessors exist so that builder
// still emits C's char_to_table field set instead of its own invented one.

// AsPlayer implements scripting.ScriptableChar.
func (p *Player) AsPlayer() scripting.ScriptablePlayer { return p }

// AsMob implements scripting.ScriptableChar.
func (p *Player) AsMob() scripting.ScriptableMob { return nil }

// AsPlayer implements scripting.ScriptableChar.
func (m *MobInstance) AsPlayer() scripting.ScriptablePlayer { return nil }

// AsMob implements scripting.ScriptableChar.
func (m *MobInstance) AsMob() scripting.ScriptableMob { return m }

// GetAlias is ch->player.name — the keyword list, which for a player is the
// character name. Source: src/scripts.c:1839.
func (p *Player) GetAlias() string { return charKeywords(p) }

// GetAlias is ch->player.name — the mobile's keyword list.
// Source: src/scripts.c:1839.
func (m *MobInstance) GetAlias() string { return charKeywords(m) }

// GetLeaderName is ch->master as the follow target's name, "" when none.
// Source: src/scripts.c:1870-1873.
func (p *Player) GetLeaderName() string { return p.GetFollowing() }

// GetLeaderName is ch->master as the follow target's name, "" when none.
// Source: src/scripts.c:1870-1873.
func (m *MobInstance) GetLeaderName() string { return m.GetFollowing() }

// GetFollowerCount is num_followers(ch) (src/scripts.c:1884). The follow graph
// lives on the world, so a detached player reports 0.
func (p *Player) GetFollowerCount() int {
	w := p.worldRef
	if w == nil {
		return 0
	}
	return w.NumFollowers(p.GetName())
}

// GetJailTimer is GET_JAIL_TIMER(ch). Source: src/scripts.c:1882.
func (p *Player) GetJailTimer() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.JailTimer
}

// GetTattoo is GET_TATTOO(ch). Source: src/scripts.c:1883.
func (p *Player) GetTattoo() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Tattoo
}

// GetWornObjects returns the equipped objects in C's wear-slot order
// (GET_EQ(ch, 0..NUM_WEARS-1), present slots only).
// Source: src/scripts.c:1849-1863.
func (p *Player) GetWornObjects() []scripting.ScriptableObject {
	if p.Equipment == nil {
		return nil
	}
	var out []scripting.ScriptableObject
	for _, slot := range cWearToGoSlot {
		if item, ok := p.Equipment.GetItemInSlot(slot); ok && item != nil {
			out = append(out, &scriptableObjInstanceWrapper{item: item})
		}
	}
	return out
}

// GetWornObjects returns the mobile's equipped objects in C's wear-slot order.
// Source: src/scripts.c:1849-1863.
func (m *MobInstance) GetWornObjects() []scripting.ScriptableObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []scripting.ScriptableObject
	for where := 0; where < NumWears; where++ {
		if item := m.Equipment[where]; item != nil {
			out = append(out, &scriptableObjInstanceWrapper{item: item})
		}
	}
	return out
}

// GetCarriedObjects returns the mobile's carried objects (ch->carrying).
// Source: src/scripts.c:1841-1847.
func (m *MobInstance) GetCarriedObjects() []scripting.ScriptableObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]scripting.ScriptableObject, 0, len(m.Inventory))
	for _, item := range m.Inventory {
		if item != nil {
			out = append(out, &scriptableObjInstanceWrapper{item: item})
		}
	}
	return out
}
