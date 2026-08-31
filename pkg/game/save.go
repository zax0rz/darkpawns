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
	// CurrentSaveVersion is the current save format version.
	// Bump this when making a breaking change to the save format.
	// Existing saves without a version field are treated as version 0.
	CurrentSaveVersion = 1

	saveDir = "./data/players"
)

// savePlayerData is a JSON-serializable snapshot of a Player for save/load.
// It excludes runtime-only fields (mu, Send, Fighting, ConnectedAt, LastActive, etc.).
type savePlayerData struct {
	SaveVersion int            `json:"save_version"` // bumped on save format changes
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
	BankGold    int            `json:"bank_gold"`
	ClanID      int            `json:"clan_id"`
	ClanRank    int            `json:"clan_rank"`
	Inventory   []SaveItemData `json:"inventory"`
	Equipment   []SaveItemData `json:"equipment"`
	Affects     []saveAffect   `json:"affects"`

	// Poof messages — immortals only
	PoofIn  string `json:"poof_in,omitempty"`
	PoofOut string `json:"poof_out,omitempty"`
}

type SaveItemData struct {
	VNum           int                    `json:"vnum"`
	Count          int                    `json:"count"`
	Locate         int                    `json:"locate"` // 0=inventory, 1+=wear slot (C WEAR_*+1)
	State          map[string]interface{} `json:"state,omitempty"`
	ContainerVNum  int                    `json:"container_vnum,omitempty"`  // parent container VNum (0 = root)
	ContainerIndex int                    `json:"container_index,omitempty"` // index of container in the save list
}

type saveAffect struct {
	SpellID   int    `json:"spell_id"`   // SPELL_* or SKILL_* number (0 = not spell-based)
	Location  int    `json:"location"`   // APPLY_* constant — which stat to modify
	Duration  int    `json:"duration"`   // Ticks remaining
	Magnitude int    `json:"magnitude"`  // Stat modifier
	Flags     uint64 `json:"flags"`      // AFF_* bitvector
	Source    string `json:"source"`     // Human-readable name
	StackID   string `json:"stack_id"`   // Dedup key
	MaxStacks int    `json:"max_stacks"` // Max stacks
	// Deprecated: Type is kept for backward compatibility with old save files.
	// New saves write SpellID + Location. Old saves are read via Type fallback.
	Type int `json:"type,omitempty"` //nolint:govet // deprecated compat field
}

// SavePlayer serializes a player's state to disk as JSON.
// Save path: ./data/players/{name}.json
func SavePlayer(player *Player) error {
	if player == nil {
		return fmt.Errorf("cannot save nil player")
	}

	if err := os.MkdirAll(saveDir, 0o750); err != nil {
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

	// Version check: 0 means old format (pre-versioning), silently upgrade.
	// Non-zero mismatch means a future or corrupted save — warn but still load.
	if data.SaveVersion != 0 && data.SaveVersion != CurrentSaveVersion {
		slog.Warn("player save version mismatch",
			"player", name,
			"file_version", data.SaveVersion,
			"expected_version", CurrentSaveVersion,
			"action", "loading with possible data loss")
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
	roomVNum := p.RoomVNum
	if p.Flags&(1<<uint(PlrLoadroom)) != 0 {
		roomVNum = p.LoadRoomVNum
	}

	data := savePlayerData{
		SaveVersion: CurrentSaveVersion,
		ID:          p.ID,
		Name:        p.Name,
		PoofIn:      p.PoofIn,
		PoofOut:     p.PoofOut,
		Sex:         p.GetSex(),
		Level:       p.GetLevel(),
		Class:       p.GetClass(),
		Race:        p.GetRace(),
		Health:      p.GetHP(),
		MaxHealth:   p.GetMaxHP(),
		Mana:        p.GetMana(),
		MaxMana:     p.GetMaxMana(),
		Move:        p.GetMove(),
		MaxMove:     p.GetMaxMove(),
		Gold:        p.GetGold(),
		BankGold:    p.BankGold,
		ClanID:      p.ClanID,
		ClanRank:    p.ClanRank,
		Exp:         p.GetExp(),
		Alignment:   p.GetAlignment(),
		RoomVNum:    roomVNum,
		Position:    p.GetPosition(),
		Title:       p.Title,
		Description: p.Description,
		AC:          p.GetAC(),
		Hitroll:     p.GetHitroll(),
		Damroll:     p.GetDamroll(),
		Strength:    p.GetStrength(),
		THAC0:       p.THAC0,
		Hunger:      p.Conditions[CondFull],
		Thirst:      p.Conditions[CondThirst],
		Drunk:       p.Conditions[CondDrunk],
		Flags:       p.GetFlags(),
		AutoExit:    p.GetAutoExit(),
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
		data.Inventory = append(data.Inventory, SaveItemData{
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
		cPos, ok := SlotToCWearPos(slot)
		locate := 0
		if ok {
			locate = cPos + 1 // C: locate = j+1 for equipped items
		}
		data.Equipment = append(data.Equipment, SaveItemData{
			VNum:   vnum,
			Count:  1,
			Locate: locate,
			State:  item.GetSaveState(),
		})
	}

	// Serialize active affects
	for _, aff := range p.ActiveAffects {
		data.Affects = append(data.Affects, saveAffect{
			SpellID:   aff.SpellID,
			Location:  aff.Location,
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
	p := &Player{
		ID:            data.ID,
		Name:          data.Name,
		PoofIn:        data.PoofIn,
		PoofOut:       data.PoofOut,
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
		BankGold:      data.BankGold,
		ClanID:        data.ClanID,
		ClanRank:      data.ClanRank,
		Exp:           data.Exp,
		Alignment:     data.Alignment,
		RoomVNum:      data.RoomVNum,
		LoadRoomVNum:  -1,
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
		OrigCon:       data.Stats.Con,
		ActiveAffects: restoreAffects(data.Affects),
		SpellMap:      data.SpellMap,
		ConnectedAt:   time.Now(),
		LastActive:    time.Now(),
		Inventory:     NewInventory(),
		Equipment:     NewEquipment(),
	}
	if data.Flags&(1<<uint(PlrLoadroom)) != 0 {
		p.LoadRoomVNum = data.RoomVNum
	}
	// Initialize race-hate slots to empty (-1); old saves do not contain this field.
	for i := range p.RaceHates {
		p.RaceHates[i] = -1
	}
	// Sync runtime condition array with serialized hunger/thirst/drunk values.
	p.Conditions[CondFull] = p.Hunger
	p.Conditions[CondThirst] = p.Thirst
	p.Conditions[CondDrunk] = p.Drunk
	p.Inventory.SetCapacity(p.Stats.Str, p.Stats.StrAdd, p.Stats.Dex, p.Level)
	return p
}

// restoreAffects converts saved affect data back into engine.Affect objects.
// Supports both new format (SpellID + Location) and legacy format (Type field).
// Reconstructs proper Affect structs with computed timestamps.
func restoreAffects(saved []saveAffect) []*engine.Affect {
	if len(saved) == 0 {
		return nil
	}
	affects := make([]*engine.Affect, 0, len(saved))
	now := time.Now()
	for _, sa := range saved {
		a := &engine.Affect{
			Duration:  sa.Duration,
			Magnitude: sa.Magnitude,
			Flags:     sa.Flags,
			Source:    sa.Source,
			StackID:   sa.StackID,
			MaxStacks: sa.MaxStacks,
			AppliedAt: now,
			ExpiresAt: now.Add(time.Duration(sa.Duration) * engine.TickDuration),
		}

		// New format: SpellID + Location are explicitly saved
		if sa.SpellID != 0 || sa.Location != 0 {
			a.SpellID = sa.SpellID
			a.Location = sa.Location
			a.Type = sa.Location //nolint:staticcheck // SA1019: backward-compatible deserialization of existing save files
		} else if sa.Type != 0 {
			// Legacy format: Type field contains the old AffectType enum value.
			// Status affects (>=100) map to flags; stat affects map to location.
			a.Type = sa.Type //nolint:staticcheck // SA1019: backward-compatible deserialization of existing save files
			if flags, ok := engine.StatusAffectFlags[sa.Type]; ok {
				a.Flags = flags
			} else {
				a.Location = sa.Type
			}
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

	if sd.SaveVersion != 0 && sd.SaveVersion != CurrentSaveVersion {
		slog.Warn("player save version mismatch (deserialize)",
			"player", sd.Name,
			"file_version", sd.SaveVersion,
			"expected_version", CurrentSaveVersion,
			"action", "loading with possible data loss")
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
	SaveVersion int                    `json:"save_version"` // bumped on save format changes
	NextMobID   int                    `json:"next_mob_id"`
	NextObjID   int                    `json:"next_obj_id"`
	DoorStates  map[int]map[string]int `json:"door_states"` // roomVNum → direction → legacy runtime state (0/1/2)
	Mobs        []saveMobPosition      `json:"mobs"`
	RoomItems   map[int][]saveItemRef  `json:"room_items"` // roomVNum → items
	Gossip      []saveGossipEntry      `json:"gossip"`
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
		SaveVersion: CurrentSaveVersion,
		NextMobID:   w.nextMobID,
		NextObjID:   w.nextObjID,
		DoorStates:  make(map[int]map[string]int),
		Mobs:        make([]saveMobPosition, 0, len(w.activeMobs)),
		RoomItems:   make(map[int][]saveItemRef),
		Gossip:      make([]saveGossipEntry, 0, len(w.gossipHistory)),
	}

	// Collect non-default door states.
	for vnum, room := range w.rooms {
		if room.Exits == nil {
			continue
		}
		for dir, exit := range room.Exits {
			state := parser.LegacyDoorState(exit.ExitInfo)
			if state != 0 {
				if data.DoorStates[vnum] == nil {
					data.DoorStates[vnum] = make(map[string]int)
				}
				data.DoorStates[vnum][dir] = state
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
// Lock ordering: w.mu → mob.mu (acquired per mob while w.mu is held). (DP-372)
func DeserializeWorld(data string, w *World) error {
	var sd saveWorldData
	if err := json.Unmarshal([]byte(data), &sd); err != nil {
		return fmt.Errorf("unmarshal world state: %w", err)
	}

	if sd.SaveVersion != 0 && sd.SaveVersion != CurrentSaveVersion {
		slog.Warn("world save version mismatch",
			"file_version", sd.SaveVersion,
			"expected_version", CurrentSaveVersion,
			"action", "loading with possible data loss")
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
				exit.ExitInfo = parser.ApplyDoorReset(exit.ExitInfo, state)
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

	if err := os.MkdirAll(filepath.Dir(worldStateFile), 0o750); err != nil {
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
