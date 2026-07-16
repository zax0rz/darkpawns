package game

// ConsumeSpellReagent extracts a named carried component through the canonical
// ObjectLocation movement path. The spells package discovers this narrow method
// without importing game, avoiding its old incompatible inventory interface.
func (p *Player) ConsumeSpellReagent(name string) bool {
	if p == nil || p.Inventory == nil {
		return false
	}
	item, found := p.Inventory.FindItem(name)
	if !found {
		return false
	}
	p.mu.RLock()
	w := p.worldRef
	p.mu.RUnlock()
	if w == nil {
		return false
	}
	w.ExtractObject(item, p.GetRoomVNum())
	return true
}
