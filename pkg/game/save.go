// Package game — player save/load via JSON serialization.
// Based on original C save.c pattern: players saved as ./data/players/{name}.json.
package game

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	saveDir = "./data/players"
)

// savePlayerData is a JSON-serializable snapshot of a Player for save/load.
// It excludes runtime-only fields (mu, Send, Fighting, ConnectedAt, LastActive, etc.).
type savePlayerData struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Sex         int            `json:"sex"`
	Level       int            `json:"level"`
	Class       int            `json:"class"`
	Race        int            `json:"race"`
	Health      int            `json:"health"`
	MaxHealth   int            `json:"max_health"`
	Mana        int            `json:"mana"`
	MaxMana     int            `json:"max_mana"`
	Move        int            `json:"move"`
	MaxMove     int            `json:"max_move"`
	Gold        int            `json:"gold"`
	Exp         int            `json:"exp"`
	Alignment   int            `json:"alignment"`
	RoomVNum    int            `json:"room_vnum"`
	Position    int            `json:"position"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	AC          int            `json:"ac"`
	Hitroll     int            `json:"hitroll"`
	Damroll     int            `json:"damroll"`
	Strength    int            `json:"strength"`
	THAC0       int            `json:"thac0"`
	Hunger      int            `json:"hunger"`
	Thirst      int            `json:"thirst"`
	Drunk       int            `json:"drunk"`
	Flags       uint64         `json:"flags"`
	AutoExit    bool           `json:"auto_exit"`
	Stats       CharStats      `json:"stats"`
	SpellMap    map[string]int `json:"spell_map"`
	Skills      map[string]int `json:"skills"`
	Inventory   []saveItemData `json:"inventory"`
	Equipment   []saveItemData `json:"equipment"`
	Affects     []saveAffect   `json:"affects"`
}

type saveItemData struct {
	VNum  int                    `json:"vnum"`
	Count int                    `json:"count"`
	State map[string]interface{} `json:"state,omitempty"`
}

type saveAffect struct {
	Type      engine.AffectType `json:"type"`
	Duration  int               `json:"duration"`
	Magnitude int               `json:"magnitude"`
	Flags     uint64            `json:"flags"`
	Source    string            `json:"source"`
	StackID   string            `json:"stack_id"`
	MaxStacks int               `json:"max_stacks"`
}

// SavePlayer serializes a player's state to disk as JSON.
// Save path: ./data/players/{name}.json
func SavePlayer(player *Player) error {
	if player == nil {
		return fmt.Errorf("cannot save nil player")
	}

	if err := os.MkdirAll(saveDir, 0750); err != nil {
		return fmt.Errorf("create save dir: %w", err)
	}

	data := playerToSaveData(player)

	path := filepath.Join(saveDir, sanitizeName(player.Name)+".json")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode save data: %w", err)
	}

	slog.Debug("Player saved", "name", player.Name, "path", path)
	return nil
}

// LoadPlayer loads a player's state from disk.
// Returns a Player with runtime fields initialized.
func LoadPlayer(name string) (*Player, error) {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open save file: %w", err)
	}
	defer f.Close()

	var data savePlayerData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode save data: %w", err)
	}

	return saveDataToPlayer(data), nil
}

// DeletePlayer removes a player's save file from disk.
func DeletePlayer(name string) error {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove save file: %w", err)
	}
	return nil
}

// PlayerSaveExists checks if a player save file exists.
func PlayerSaveExists(name string) bool {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	_, err := os.Stat(path)
	return err == nil
}

// playerToSaveData converts a Player to the serializable savePlayerData.
func playerToSaveData(p *Player) savePlayerData {
	data := savePlayerData{
		ID:          p.ID,
		Name:        p.Name,
		Sex:         p.Sex,
		Level:       p.Level,
		Class:       p.Class,
		Race:        p.Race,
		Health:      p.Health,
		MaxHealth:   p.MaxHealth,
		Mana:        p.Mana,
		MaxMana:     p.MaxMana,
		Move:        p.Move,
		MaxMove:     p.MaxMove,
		Gold:        p.Gold,
		Exp:         p.Exp,
		Alignment:   p.Alignment,
		RoomVNum:    p.RoomVNum,
		Position:    p.Position,
		Title:       p.Title,
		Description: p.Description,
		AC:          p.AC,
		Hitroll:     p.Hitroll,
		Damroll:     p.Damroll,
		Strength:    p.Strength,
		THAC0:       p.THAC0,
		Hunger:      p.Hunger,
		Thirst:      p.Thirst,
		Drunk:       p.Drunk,
		Flags:       p.Flags,
		AutoExit:    p.AutoExit,
		Stats:       p.Stats,
		SpellMap:    make(map[string]int),
	}

	// Copy spell map
	if p.SpellMap != nil {
		for k, v := range p.SpellMap {
			data.SpellMap[k] = v
		}
	}

	// Copy skills from SkillManager
	data.Skills = make(map[string]int)
	if p.SkillManager != nil {
		for _, skill := range p.SkillManager.GetLearnedSkills() {
			data.Skills[skill.Name] = skill.Level
		}
	}

	// Flatten inventory to VNUM + state
	for _, item := range p.Inventory.Items {
		if item == nil {
			continue
		}
		vnum := item.VNum
		if item.Prototype != nil {
			vnum = item.Prototype.VNum
		}
		data.Inventory = append(data.Inventory, saveItemData{
			VNum:  vnum,
			Count: 1,
			State: item.CustomData,
		})
	}

	// Flatten equipment to VNUM + state (from hit/mana/move save_data in merc.h:479-483)
	for _, item := range p.Equipment.Slots {
		if item == nil {
			continue
		}
		vnum := item.VNum
		if item.Prototype != nil {
			vnum = item.Prototype.VNum
		}
		data.Equipment = append(data.Equipment, saveItemData{
			VNum:  vnum,
			Count: 1,
			State: item.CustomData,
		})
	}

	// Serialize active affects
	for _, aff := range p.ActiveAffects {
		data.Affects = append(data.Affects, saveAffect{
			Type:      aff.Type,
			Duration:  aff.Duration,
			Magnitude: aff.Magnitude,
			Flags:     aff.Flags,
			Source:    aff.Source,
			StackID:   aff.StackID,
			MaxStacks: aff.MaxStacks,
		})
	}

	return data
}

// saveDataToPlayer converts savePlayerData back to a Player with runtime fields.
func saveDataToPlayer(data savePlayerData) *Player {
	return &Player{
		ID:            data.ID,
		Name:          data.Name,
		Sex:           data.Sex,
		Level:         data.Level,
		Class:         data.Class,
		Race:          data.Race,
		Health:        data.Health,
		MaxHealth:     data.MaxHealth,
		Mana:          data.Mana,
		MaxMana:       data.MaxMana,
		Move:          data.Move,
		MaxMove:       data.MaxMove,
		Gold:          data.Gold,
		Exp:           data.Exp,
		Alignment:     data.Alignment,
		RoomVNum:      data.RoomVNum,
		Position:      data.Position,
		Title:         data.Title,
		Description:   data.Description,
		AC:            data.AC,
		Hitroll:       data.Hitroll,
		Damroll:       data.Damroll,
		Strength:      data.Strength,
		THAC0:         data.THAC0,
		Hunger:        data.Hunger,
		Thirst:        data.Thirst,
		Drunk:         data.Drunk,
		Flags:         data.Flags,
		AutoExit:      data.AutoExit,
		Stats:         data.Stats,
		ActiveAffects: make([]*engine.Affect, len(data.Affects)),
		SpellMap:      data.SpellMap,
		ConnectedAt:   time.Now(),
		LastActive:    time.Now(),
		Send:          make(chan []byte, 100),
		Inventory:     NewInventory(),
		Equipment:     NewEquipment(),
	}
}

// SerializePlayer serializes a player to JSON for storage backends.
func SerializePlayer(p *Player) (string, error) {
	data := playerToSaveData(p)
	out, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal player: %w", err)
	}
	return string(out), nil
}

// DeserializePlayer deserializes a player from JSON produced by SerializePlayer.
func DeserializePlayer(data string) (*Player, error) {
	var sd savePlayerData
	if err := json.Unmarshal([]byte(data), &sd); err != nil {
		return nil, fmt.Errorf("unmarshal player: %w", err)
	}
	return saveDataToPlayer(sd), nil
}

// SerializeWorld serializes world state to JSON.
func SerializeWorld(w *World) (string, error) {
	// World serialization is a stub for future use — world data is loaded from
	// parser data files, not persisted state. This function will evolve when
	// dynamic world state (zone resets, mob spawns, lock states) needs saving.
	return "{}", nil
}

// DeserializeWorld deserializes world state from JSON.
func DeserializeWorld(data string) (*World, error) {
	// Stub: world state is loaded from static parser files.
	return nil, fmt.Errorf("world deserialization not implemented yet")
}

// sanitizeName ensures the player name is safe for use as a filename.
func sanitizeName(name string) string {
	safe := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			safe = append(safe, c)
		}
	}
	return string(safe)
}
