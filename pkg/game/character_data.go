package game

import (
	"encoding/json"
	"fmt"
)

// copyCharFileRest copies the char_file_u fields added in save version 2
// from src into d.
func (d *savePlayerData) copyCharFileRest(src *savePlayerData) {
	d.Practices, d.Height, d.Weight = src.Practices, src.Height, src.Weight
	d.Birth, d.Played = src.Birth, src.Played
	d.InvisLevel, d.FreezeLevel, d.WimpLevel, d.LoadRoom = src.InvisLevel, src.FreezeLevel, src.WimpLevel, src.LoadRoom
	d.Kills, d.PKs, d.Deaths, d.OrigCon = src.Kills, src.PKs, src.Deaths, src.OrigCon
	d.SavingThrows, d.Tattoo, d.TatTimer = src.SavingThrows, src.Tattoo, src.TatTimer
	d.MountVNum, d.MountCostDay, d.MountRentTime, d.LastDeath = src.MountVNum, src.MountCostDay, src.MountRentTime, src.LastDeath
	d.HolyLight, d.AutoGold, d.AutoSplit, d.NoBroadcast = src.HolyLight, src.AutoGold, src.AutoSplit, src.NoBroadcast
}

// EncodeCharacterData is the part of a character the game store's own
// columns don't hold: C's char_file_u minus name, password, description,
// title, room, level, exp, points, class, race, abilities, conditions,
// hometown, olc zone and objects (those have columns). It is the JSON save's
// record without inventory and equipment, so the two formats cannot drift.
func EncodeCharacterData(p *Player) ([]byte, error) {
	data := playerToSaveData(p)
	data.Inventory, data.Equipment = nil, nil
	out, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode character data: %w", err)
	}
	return out, nil
}

// ApplyCharacterData restores what EncodeCharacterData wrote onto a player
// rebuilt from the game store's columns. An empty record (a character saved
// before the column existed) leaves the player as the columns made them.
func ApplyCharacterData(p *Player, raw []byte) error {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return nil
	}
	var data savePlayerData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("decode character data: %w", err)
	}
	p.mu.Lock()
	p.Sex, p.Gold, p.BankGold, p.Alignment = data.Sex, data.Gold, data.BankGold, data.Alignment
	p.Flags = migrateFlags(data.SaveVersion, data.Flags)
	p.AutoExit = data.AutoExit
	p.AC, p.Hitroll, p.Damroll = data.AC, data.Hitroll, data.Damroll
	p.ClanID, p.ClanRank = data.ClanID, data.ClanRank
	p.PoofIn, p.PoofOut = data.PoofIn, data.PoofOut
	if data.SpellMap != nil {
		p.SpellMap = data.SpellMap
	}
	p.ActiveAffects = restoreAffects(data.Affects)
	p.Practices, p.Height, p.Weight = data.Practices, data.Height, data.Weight
	p.Birth, p.PlayedDuration = data.Birth, data.Played
	p.InvisLevel, p.FreezeLevel, p.WimpLevel = data.InvisLevel, data.FreezeLevel, data.WimpLevel
	// data.LoadRoom is recorded but not applied here: which room a login
	// enters is the game store's room column today, and C's load-room rule
	// is DP-1310's change.
	p.Kills, p.PKs, p.Deaths, p.OrigCon = data.Kills, data.PKs, data.Deaths, data.OrigCon
	p.SavingThrows, p.Tattoo, p.TatTimer = data.SavingThrows, data.Tattoo, data.TatTimer
	p.MountVNum, p.MountCostDay, p.MountRentTime, p.LastDeath = data.MountVNum, data.MountCostDay, data.MountRentTime, data.LastDeath
	p.HolyLight, p.AutoGold, p.AutoSplit, p.NoBroadcast = data.HolyLight, data.AutoGold, data.AutoSplit, data.NoBroadcast
	p.mu.Unlock()
	restoreSkills(p, data.Skills)
	return nil
}

// restoreSkills sets each saved skill's learned percentage.
func restoreSkills(p *Player, skills map[string]int) {
	restored := make(map[string]int, len(skills))
	for name, level := range skills {
		// A save may hold one skill under both its old catalog name and its
		// handler key (practised before and after DP-1342); keep the higher.
		key := canonicalSkillKey(name)
		if level > restored[key] {
			restored[key] = level
		}
	}
	for name, level := range restored {
		p.SetSkill(name, level)
	}
}
