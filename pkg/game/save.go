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
	"github.com/zax0rz/darkpawns/pkg/parser"
)

const (
	saveDir = "./data/players"
)

// savePlayerData is a JSON-serializable snapshot of a Player for save/load.
// It excludes runtime-only fields (mu, Send, Fighting, ConnectedAt, LastActive, etc.).
type savePlayerData struct {
	ID          int                 `json:"id"`
	Name        string              `json:"name"`
	Sex         int                 `json:"sex"`
	Level       int                 `json:"level"`
	Class       int                 `json:"class"`
	Race        int                 `json:"race"`
	Health      int                 `json:"health"`
	MaxHealth   int                 `json:"max_health"`
	Mana        int                 `json:"mana"`
	MaxMana     int                 `json:"max_mana"`
	Move        int                 `json:"move"`
	MaxMove     int                 `json:"max_move"`
	Gold        int                 `json:"gold"`
	Exp         int                 `json:"exp"`
	Alignment   int                 `json:"alignment"`
	RoomVNum    int                 `json:"room_vnum"`
	Position    int                 `json:"position"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	AC          int                 `json:"ac"`
	Hitroll     int                 `json:"hitroll"`
	Damroll     int                 `json:"damroll"`
	Strength    int                 `json:"strength"`
	THAC0       int                 `json:"thac0"`
	Hunger      int                 `json:"hunger"`
	Thirst      int                 `json:"thirst"`
	Drunk       int                 `json:"drunk"`
	Flags       uint64              `json:"flags"`
	AutoExit    bool                `json:"auto_exit"`
	Stats       CharStats           `json:"stats"`
	SpellMap    map[string]int      `json:"spell_map"`
	Skills      map[string]int      `json:"skills"`
	BankGold    int                 `json:"bank_gold"`
	Inventory   []saveItemData      `json:"inventory"`
	Equipment   []saveItemData      `json:"equipment"`
	Affects     []saveAffect        `json:"affects"`

	// Rent metadata — tracks why/how items were saved.
	RentCode      int   `json:"rent_code"`       // RentCrash, RentRented, RentCryo, RentTimedOut, RentForced
	RentTime      int64 `json:"rent_time"`       // Unix timestamp when saved
	NetCostPerDiem int  `json:"net_cost_per_diem"` // daily rent cost
	SavedGold      int  `json:"saved_gold"`       // gold at time of save
	SavedBankGold  int  `json:"saved_bank_gold"`   // bank gold at time of save
}

type saveItemData struct {
	VNum   int                    `json:"vnum"`
	Count  int                    `json:"count"`
	Locate int                    `json:"locate"` // 0=inventory, 1+=wear slot (C WEAR_*+1)
	State  map[string]interface{} `json:"state,omitempty"`
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
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}
	defer func() { _ = f.Close() }()

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
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open save file: %w", err)
	}
	defer func() { _ = f.Close() }()

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
// Acquires p.mu.RLock to prevent torn reads from concurrent mutations.
func playerToSaveData(p *Player) savePlayerData {
	p.mu.RLock()
	defer p.mu.RUnlock()

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
		BankGold:    p.BankGold,
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
			VNum:   vnum,
			Count:  1,
			Locate: 0,
			State:  item.GetSaveState(),
		})
	}

	// Flatten equipment to VNUM + state + locate (C WEAR_*+1)
	for slot, item := range p.Equipment.Slots {
		if item == nil {
			continue
		}
		vnum := item.VNum
		if item.Prototype != nil {
			vnum = item.Prototype.VNum
		}
		cPos, ok := goSlotToCWearPos(slot)
		locate := 0
		if ok {
			locate = cPos + 1 // C: locate = j+1 for equipped items
		}
		data.Equipment = append(data.Equipment, saveItemData{
			VNum:   vnum,
			Count:  1,
			Locate: locate,
			State:  item.GetSaveState(),
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
		ID:           data.ID,
		Name:         data.Name,
		Sex:          data.Sex,
		Level:        data.Level,
		Class:        data.Class,
		Race:         data.Race,
		Health:       data.Health,
		MaxHealth:    data.MaxHealth,
		Mana:         data.Mana,
		MaxMana:      data.MaxMana,
		Move:         data.Move,
		MaxMove:      data.MaxMove,
		Gold:         data.Gold,
		BankGold:     data.BankGold,
		Exp:          data.Exp,
		Alignment:    data.Alignment,
		RoomVNum:     data.RoomVNum,
		Position:     data.Position,
		Title:        data.Title,
		Description:  data.Description,
		AC:           data.AC,
		Hitroll:      data.Hitroll,
		Damroll:      data.Damroll,
		Strength:     data.Strength,
		THAC0:        data.THAC0,
		Hunger:       data.Hunger,
		Thirst:       data.Thirst,
		Drunk:        data.Drunk,
		Flags:        data.Flags,
		AutoExit:     data.AutoExit,
		Stats:        data.Stats,
		ActiveAffects: restoreAffects(data.Affects),
		SpellMap:     data.SpellMap,
		ConnectedAt:  time.Now(),
		LastActive:   time.Now(),
		Inventory:    NewInventory(),
		Equipment:    NewEquipment(),
	}
}

// restoreAffects converts saved affect data back into engine.Affect objects.
// Reconstructs proper Affect structs with computed timestamps.
func restoreAffects(saved []saveAffect) []*engine.Affect {
	if len(saved) == 0 {
		return nil
	}
	affects := make([]*engine.Affect, 0, len(saved))
	now := time.Now()
	for _, sa := range saved {
		a := &engine.Affect{
			Type:      sa.Type,
			Duration:  sa.Duration,
			Magnitude: sa.Magnitude,
			Flags:     sa.Flags,
			Source:    sa.Source,
			StackID:   sa.StackID,
			MaxStacks: sa.MaxStacks,
			AppliedAt: now,
			ExpiresAt: now.Add(time.Duration(sa.Duration) * engine.TickDuration),
		}
		affects = append(affects, a)
	}
	return affects
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


// ---------------------------------------------------------------------------
// World serialization — persists dynamic world state across server restarts.
//
// What is saved:
//   - Door states (opened/closed/locked by players)
//   - Active mob positions (mobs that moved from their spawn room)
//   - Room items (items on the ground)
//   - NextID counters (nextMobID, nextObjID)
//   - Gossip history (last 25 messages)
//
// What is NOT saved:
//   - Static room/mob/obj/zone definitions (reloaded from parser files)
//   - Player data (handled by SavePlayer/LoadPlayer)
//   - Spawner zone timers / zone dispatcher state (restarted on boot)
// ---------------------------------------------------------------------------

const worldStateFile = "./data/world_state.json"

// saveWorldData is the top-level JSON-serializable structure for world state.
type saveWorldData struct {
	NextMobID  int                    `json:"next_mob_id"`
	NextObjID  int                    `json:"next_obj_id"`
	DoorStates map[int]map[string]int `json:"door_states"` // roomVNum → direction → DoorState
	Mobs       []saveMobPosition      `json:"mobs"`
	RoomItems  map[int][]saveItemRef  `json:"room_items"` // roomVNum → items
	Gossip     []saveGossipEntry      `json:"gossip"`
}

// saveMobPosition captures a mob's runtime position and state.
type saveMobPosition struct {
	VNum      int `json:"vnum"`
	ID        int `json:"id"`
	RoomVNum  int `json:"room_vnum"`
	CurrentHP int `json:"current_hp"`
	MaxHP     int `json:"max_hp"`
}

// saveItemRef is a lightweight reference to an object in a room.
type saveItemRef struct {
	VNum int `json:"vnum"`
	ID   int `json:"id"`
}

// saveGossipEntry captures one gossip message for the review buffer.
type saveGossipEntry struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Invis   int    `json:"invis"`
}

// SerializeWorld serializes dynamic world state to JSON.
func SerializeWorld(w *World) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := saveWorldData{
		NextMobID:  w.nextMobID,
		NextObjID:  w.nextObjID,
		DoorStates: make(map[int]map[string]int),
		Mobs:       make([]saveMobPosition, 0, len(w.activeMobs)),
		RoomItems:  make(map[int][]saveItemRef),
		Gossip:     make([]saveGossipEntry, 0, len(w.gossipHistory)),
	}

	// Collect non-default door states.
	for vnum, room := range w.rooms {
		if room.Exits == nil {
			continue
		}
		for dir, exit := range room.Exits {
			if exit.DoorState != 0 {
				if data.DoorStates[vnum] == nil {
					data.DoorStates[vnum] = make(map[string]int)
				}
				data.DoorStates[vnum][dir] = exit.DoorState
			}
		}
	}

	// Collect active mob positions and HP.
	for _, mob := range w.activeMobs {
		mob.mu.RLock()
		data.Mobs = append(data.Mobs, saveMobPosition{
			VNum:      mob.VNum,
			ID:        mob.ID,
			RoomVNum:  mob.RoomVNum,
			CurrentHP: mob.CurrentHP,
			MaxHP:     mob.MaxHP,
		})
		mob.mu.RUnlock()
	}

	// Collect room items (objects on the ground).
	for roomVNum, items := range w.roomItems {
		refs := make([]saveItemRef, 0, len(items))
		for _, item := range items {
			refs = append(refs, saveItemRef{
				VNum: item.VNum,
				ID:   item.ID,
			})
		}
		if len(refs) > 0 {
			data.RoomItems[roomVNum] = refs
		}
	}

	// Copy gossip history.
	w.gossipMu.RLock()
	for _, entry := range w.gossipHistory {
		data.Gossip = append(data.Gossip, saveGossipEntry(entry))
	}
	w.gossipMu.RUnlock()

	out, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal world state: %w", err)
	}
	return string(out), nil
}

// DeserializeWorld restores dynamic world state from JSON.
// Must be called AFTER zone resets have spawned mobs (so we can reposition them).
func DeserializeWorld(data string, w *World) error {
	var sd saveWorldData
	if err := json.Unmarshal([]byte(data), &sd); err != nil {
		return fmt.Errorf("unmarshal world state: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Restore nextID counters — always take the saved value if higher.
	if sd.NextMobID > w.nextMobID {
		w.nextMobID = sd.NextMobID
	}
	if sd.NextObjID > w.nextObjID {
		w.nextObjID = sd.NextObjID
	}

	// Restore door states.
	for roomVNum, dirMap := range sd.DoorStates {
		room, ok := w.rooms[roomVNum]
		if !ok || room.Exits == nil {
			continue
		}
		for dir, state := range dirMap {
			if exit, ok := room.Exits[dir]; ok {
				exit.DoorState = state
				room.Exits[dir] = exit // map value — must reassign
			}
		}
	}

	// Reposition active mobs.
	// Build a lookup: VNum → []*MobInstance (mobs spawned by zone resets).
	mobsByVNum := make(map[int][]*MobInstance)
	for _, mob := range w.activeMobs {
		mobsByVNum[mob.VNum] = append(mobsByVNum[mob.VNum], mob)
	}

	// Track which mob instances have been matched so we don't reposition
	// the same instance twice.
	matched := make(map[int]bool)
	for _, saved := range sd.Mobs {
		candidates := mobsByVNum[saved.VNum]
		for _, mob := range candidates {
			if matched[mob.ID] {
				continue
			}
			mob.mu.Lock()
			mob.RoomVNum = saved.RoomVNum
			if saved.CurrentHP > 0 {
				mob.CurrentHP = saved.CurrentHP
			}
			if saved.MaxHP > 0 {
				mob.MaxHP = saved.MaxHP
			}
			mob.mu.Unlock()
			matched[mob.ID] = true
			break
		}
	}

	// Restore room items.
	// Items dropped on the ground need to be recreated from prototypes.
	for roomVNum, refs := range sd.RoomItems {
		if _, ok := w.rooms[roomVNum]; !ok {
			continue // room doesn't exist anymore
		}
		for _, ref := range refs {
			proto, ok := w.objs[ref.VNum]
			if !ok {
				slog.Warn("DeserializeWorld: unknown obj vnum", "vnum", ref.VNum)
				continue
			}
			obj := NewObjectInstance(proto, roomVNum)
			obj.ID = w.nextObjID
			w.nextObjID++
			obj.Location = LocRoom(roomVNum)
			w.objectInstances[obj.ID] = obj
			w.roomItems[roomVNum] = append(w.roomItems[roomVNum], obj)
		}
	}

	// Restore gossip history.
	w.gossipMu.Lock()
	w.gossipHistory = make([]gossipEntry, 0, len(sd.Gossip))
	for _, entry := range sd.Gossip {
		w.gossipHistory = append(w.gossipHistory, gossipEntry(entry))
	}
	w.gossipMu.Unlock()

	return nil
}

// SaveWorld persists the world state to disk.
func SaveWorld(w *World) error {
	data, err := SerializeWorld(w)
	if err != nil {
		return fmt.Errorf("serialize world: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(worldStateFile), 0750); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	f, err := os.Create(filepath.Clean(worldStateFile))
	if err != nil {
		return fmt.Errorf("create world state file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode world state: %w", err)
	}

	slog.Info("World state saved", "path", worldStateFile)
	return nil
}

// LoadWorld restores the world state from disk.
// Must be called after NewWorld() and StartZoneResets() so that mobs are
// already spawned and can be repositioned.
func LoadWorld(w *World) error {
	f, err := os.Open(filepath.Clean(worldStateFile))
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("No world state file found, starting fresh")
			return nil // not an error — first boot
		}
		return fmt.Errorf("open world state file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var raw string
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return fmt.Errorf("decode world state: %w", err)
	}

	if err := DeserializeWorld(raw, w); err != nil {
		return fmt.Errorf("deserialize world: %w", err)
	}

	slog.Info("World state loaded", "path", worldStateFile)
	return nil
}
// Used by CrashSave, RentSave, CryoSave, Idlesave.
func SavePlayerWithRent(p *Player, rentCode int, netCostPerDiem int) error {
	if p == nil {
		return fmt.Errorf("cannot save nil player")
	}
	if err := os.MkdirAll(saveDir, 0750); err != nil {
		return fmt.Errorf("create save dir: %w", err)
	}

	data := playerToSaveData(p)
	data.RentCode = rentCode
	data.RentTime = time.Now().Unix()
	data.NetCostPerDiem = netCostPerDiem
	data.SavedGold = p.Gold
	data.SavedBankGold = p.BankGold

	path := filepath.Join(saveDir, sanitizeName(p.Name)+".json")
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode save data: %w", err)
	}

	slog.Debug("Player saved with rent", "name", p.Name, "rent_code", rentCode, "cost", netCostPerDiem)
	return nil
}

// LoadSaveData loads raw save data (without creating a Player) for inspection.
// Used by CleanCrashFile and CrashLoad to check rent metadata.
func LoadSaveData(name string) (savePlayerData, error) {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return savePlayerData{}, fmt.Errorf("open save file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var data savePlayerData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return savePlayerData{}, fmt.Errorf("decode save data: %w", err)
	}
	return data, nil
}

// CrashLoad loads a player's saved items and handles rent cost deduction.
// Ported from C Crash_load() — returns:
//   0: successful load (rented/cryo), keep in rent room
//   1: crash load or failure, put in temple
//   2: rented equipment lost (no gold)
//
// getProto is a callback to look up object prototypes by vnum.
func CrashLoad(p *Player, getProto func(vnum int) (*parser.Obj, bool)) int {
	if p == nil {
		return 1
	}

	data, err := LoadSaveData(p.GetName())
	if err != nil {
		slog.Debug("CrashLoad: no save file", "name", p.GetName())
		p.SendMessage("No saved equipment found.\r\n")
		return 1
	}

	origRentCode := data.RentCode

	// Handle rent cost deduction for rented/timedout saves.
	if data.RentCode == RentRented || data.RentCode == RentTimedOut {
		numDays := float64(time.Now().Unix()-data.RentTime) / 86400.0
		cost := int(float64(data.NetCostPerDiem) * numDays)
		if cost > p.Gold+p.BankGold {
			slog.Info("Player rented equipment lost (no $)", "name", p.GetName())
			// Overwrite with crash save (C: Crash_crashsave)
			if err := SavePlayerWithRent(p, RentCrash, 0); err != nil {
				slog.Error("SavePlayerWithRent failed in rent cost check", "player", p.GetName(), "error", err)
			}
			return 2
		}
		// Deduct cost from bank first, then gold.
		p.BankGold -= max(cost-p.Gold, 0)
		p.Gold = max(p.Gold-cost, 0)
		if err := SavePlayer(p); err != nil {
			slog.Error("SavePlayer failed in rent unrent", "player", p.GetName(), "error", err)
		}
	}

	// Log entry.
	switch origRentCode {
	case RentRented:
		slog.Info("Player un-renting", "name", p.GetName())
	case RentCrash:
		slog.Info("Player retrieving crash-saved items", "name", p.GetName())
	case RentCryo:
		slog.Info("Player un-cryo'ing", "name", p.GetName())
	case RentForced, RentTimedOut:
		slog.Info("Player retrieving force-saved items", "name", p.GetName())
	default:
		slog.Warn("Player entering with undefined rent code", "name", p.GetName(), "code", origRentCode)
	}

	// Restore inventory items with AutoEquip.
	for _, item := range data.Inventory {
		proto, ok := getProto(item.VNum)
		if !ok {
			slog.Warn("CrashLoad: missing inv proto", "vnum", item.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if item.State != nil {
			for k, v := range item.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		AutoEquip(p, obj, item.Locate)
	}

	// Restore equipment items with AutoEquip.
	for _, item := range data.Equipment {
		proto, ok := getProto(item.VNum)
		if !ok {
			slog.Warn("CrashLoad: missing eq proto", "vnum", item.VNum)
			continue
		}
		obj := NewObjectInstance(proto, -1)
		if item.State != nil {
			for k, v := range item.State {
				obj.CustomData[k] = v
			}
			obj.MigrateCustomData()
		}
		AutoEquip(p, obj, item.Locate)
	}

	// Convert to crash save (rent.rentcode = RENT_CRASH, rewrite control block).
	if err := SavePlayerWithRent(p, RentCrash, 0); err != nil {
		slog.Error("SavePlayerWithRent failed in crash save conversion", "player", p.GetName(), "error", err)
	}

	if origRentCode == RentRented || origRentCode == RentCryo {
		return 0
	}
	return 1
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
