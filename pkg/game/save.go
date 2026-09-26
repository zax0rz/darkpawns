// Package game — player save/load via JSON serialization.
// Based on original C save.c pattern: players saved as ./data/players/{name}.json.
package game

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	// CurrentSaveVersion is the current save format version.
	// Bump this when making a breaking change to the save format.
	// Existing saves without a version field are treated as version 0.
	// Version 2 moved PRF preferences to prfBase in Flags; see migrateFlagsV1.
	CurrentSaveVersion = 2

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

	// The rest of C's char_file_u (structs.h): everything save_char keeps
	// that the fields above did not.
	Practices     int    `json:"practices"`       // player_specials.saved.spells_to_learn
	Height        int    `json:"height"`          // char_player_data.height
	Weight        int    `json:"weight"`          // char_player_data.weight
	Birth         int64  `json:"birth"`           // char_player_data.birth
	Played        int64  `json:"played"`          // char_player_data.played (seconds)
	InvisLevel    int    `json:"invis_level"`     // player_specials.saved.invis_level
	FreezeLevel   int    `json:"freeze_level"`    // player_specials.saved.freeze_level
	WimpLevel     int    `json:"wimp_level"`      // player_specials.saved.wimp_level
	LoadRoom      int    `json:"load_room"`       // player_specials.saved.load_room
	Kills         int    `json:"kills"`           // killcount
	PKs           int    `json:"pks"`             // pkcount
	Deaths        int    `json:"deaths"`          // deathcount
	OrigCon       int    `json:"orig_con"`        // orig_con
	SavingThrows  [5]int `json:"saving_throws"`   // char_specials.saved.apply_saving_throw
	Tattoo        int    `json:"tattoo"`          // tattoo
	TatTimer      int    `json:"tat_timer"`       // tattimer
	MountVNum     int    `json:"mount_vnum"`      // mount_vnum
	MountCostDay  int    `json:"mount_cost_day"`  // mount_cost_day
	MountRentTime int64  `json:"mount_rent_time"` // mount_rent
	LastDeath     int64  `json:"last_death"`      // lastdeath
	HolyLight     bool   `json:"holy_light"`      // PRF_HOLYLIGHT's runtime mirror
	AutoGold      bool   `json:"auto_gold"`       // PRF_AUTOGOLD's runtime mirror
	AutoSplit     bool   `json:"auto_split"`      // PRF_AUTOSPLIT's runtime mirror
	NoBroadcast   bool   `json:"no_broadcast"`    // PRF_NOBROAD's runtime mirror
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

	if err := encodeSave(f, f.Close, data); err != nil {
		return err
	}

	slog.Debug("Player saved", "name", player.Name, "path", path)
	return nil
}

// encodeSave writes data as indented JSON to w, then invokes closeFn.
// A close failure is reported when encoding succeeded: on filesystems with
// delayed writeback a short write can surface at close time, and reporting
// it keeps a truncated save from looking successful. C's save_char returns
// void and never signals failure, so this is Go-internal hardening only —
// no player-facing behavior changes.
func encodeSave(w io.Writer, closeFn func() error, data any) (err error) {
	defer func() {
		if cerr := closeFn(); cerr != nil && err == nil {
			err = fmt.Errorf("close save file: %w", cerr)
		}
	}()
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err = enc.Encode(data); err != nil {
		return fmt.Errorf("encode save data: %w", err)
	}
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
	// Version 1 is migrated on load (migrateFlags); older than that is
	// version 0, which shares version 1's layout.
	if data.SaveVersion > CurrentSaveVersion {
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
// Each component is read under its own lock: scalars via the p.mu-based
// getters, direct fields under a single p.mu.RLock (never nested with the
// getters — a writer queuing between two RLocks deadlocks), inventory under
// inv.mu, equipment under eq.mu (DP-1247).
func playerToSaveData(p *Player) savePlayerData {
	// Direct fields with no getter, read under one short RLock.
	p.mu.RLock()
	poofIn, poofOut := p.PoofIn, p.PoofOut
	title, description := p.Title, p.Description
	clanID, clanRank := p.ClanID, p.ClanRank
	stats := p.Stats
	loadRoomVNum := p.LoadRoomVNum
	// C saves points.armor/hitroll/damroll without affects or equipment
	// (char_to_store removes them first); the getters return totals.
	baseAC, baseHitroll, baseDamroll := p.AC, p.Hitroll, p.Damroll
	extra := savePlayerData{
		Practices: p.Practices, Height: p.Height, Weight: p.Weight,
		Birth: p.Birth, Played: p.PlayedDuration,
		InvisLevel: p.InvisLevel, FreezeLevel: p.FreezeLevel, WimpLevel: p.WimpLevel,
		LoadRoom: p.LoadRoomVNum, Kills: p.Kills, PKs: p.PKs, Deaths: p.Deaths,
		OrigCon: p.OrigCon, SavingThrows: p.SavingThrows, Tattoo: p.Tattoo, TatTimer: p.TatTimer,
		MountVNum: p.MountVNum, MountCostDay: p.MountCostDay, MountRentTime: p.MountRentTime,
		LastDeath: p.LastDeath, HolyLight: p.HolyLight, AutoGold: p.AutoGold,
		AutoSplit: p.AutoSplit, NoBroadcast: p.NoBroadcast,
	}
	hasLoadroom := p.Flags&(1<<uint(PlrLoadroom)) != 0
	affects := append([]*engine.Affect(nil), p.ActiveAffects...)
	p.mu.RUnlock()

	roomVNum := p.GetRoom()
	if hasLoadroom {
		roomVNum = loadRoomVNum
	}

	data := savePlayerData{
		SaveVersion: CurrentSaveVersion,
		ID:          p.ID,
		Name:        p.Name,
		PoofIn:      poofIn,
		PoofOut:     poofOut,
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
		BankGold:    p.GetBankGold(),
		ClanID:      clanID,
		ClanRank:    clanRank,
		Exp:         p.GetExp(),
		Alignment:   p.GetAlignment(),
		RoomVNum:    roomVNum,
		Position:    p.GetPosition(),
		Title:       title,
		Description: description,
		AC:          baseAC,
		Hitroll:     baseHitroll,
		Damroll:     baseDamroll,
		Strength:    p.GetStrength(),
		THAC0:       p.GetTHAC0(),
		Hunger:      p.GetCondition(CondFull),
		Thirst:      p.GetCondition(CondThirst),
		Drunk:       p.GetCondition(CondDrunk),
		Flags:       p.GetFlags(),
		AutoExit:    p.GetAutoExit(),
		Stats:       stats,
		SpellMap:    make(map[string]int),
	}
	data.copyCharFileRest(&extra)

	// Copy spell map under p.mu (all SpellMap writers hold p.mu).
	p.mu.RLock()
	for k, v := range p.SpellMap {
		data.SpellMap[k] = v
	}
	p.mu.RUnlock()

	// Copy skills from SkillManager
	data.Skills = make(map[string]int)
	if p.SkillManager != nil {
		for _, skill := range p.SkillManager.GetLearnedSkills() {
			data.Skills[skill.Name] = skill.Level
		}
	}

	// Flatten inventory to VNUM + state (snapshot under inv.mu)
	for _, item := range p.Inventory.Snapshot() {
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

	// Flatten equipment to VNUM + state + locate (C WEAR_*+1) (snapshot under eq.mu)
	for slot, item := range p.Equipment.Snapshot() {
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

	// Serialize active affects (copied above under p.mu)
	for _, aff := range affects {
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
		Flags:         migrateFlags(data.SaveVersion, data.Flags),
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
	if data.SaveVersion >= 2 {
		p.Practices, p.Height, p.Weight = data.Practices, data.Height, data.Weight
		p.Birth, p.PlayedDuration = data.Birth, data.Played
		p.InvisLevel, p.FreezeLevel, p.WimpLevel = data.InvisLevel, data.FreezeLevel, data.WimpLevel
		p.Kills, p.PKs, p.Deaths, p.OrigCon = data.Kills, data.PKs, data.Deaths, data.OrigCon
		p.SavingThrows, p.Tattoo, p.TatTimer = data.SavingThrows, data.Tattoo, data.TatTimer
		p.MountVNum, p.MountCostDay, p.MountRentTime, p.LastDeath = data.MountVNum, data.MountCostDay, data.MountRentTime, data.LastDeath
		p.HolyLight, p.AutoGold, p.AutoSplit, p.NoBroadcast = data.HolyLight, data.AutoGold, data.AutoSplit, data.NoBroadcast
	}
	// The skills were saved and never read back.
	restoreSkills(p, data.Skills)
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

	if sd.SaveVersion > CurrentSaveVersion {
		slog.Warn("player save version mismatch (deserialize)",
			"player", sd.Name,
			"file_version", sd.SaveVersion,
			"expected_version", CurrentSaveVersion,
			"action", "loading with possible data loss")
	}

	return saveDataToPlayer(sd), nil
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

// flagsV1PRF is where save versions 0 and 1 kept each PRF flag in Flags,
// indexed by C's PRF number. PRF_QUEST and PRF_SUMMONABLE sat after the
// rest.
var flagsV1PRF = [32]int{
	20, 21, 22, 23, 24, 25, 26, 27, 28, 49, 48, 30, 29, 31, 32, 33,
	34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 50, 51,
}

// migrateFlags returns a save's Flags in the current layout.
func migrateFlags(version int, flags uint64) uint64 {
	if version < 2 {
		return migrateFlagsV1(flags)
	}
	return flags
}

// migrateFlagsV1 moves a version 0/1 save's preferences from their old bits
// to prfBase + C's PRF number. PLR bits below 20 are unchanged. Bits 20 and
// 21 were shared by PRF_BRIEF/PLR_REMORT and PRF_COMPACT/PLR_EXTRACT; they are
// read as the preferences, which is what nearly every save meant.
func migrateFlagsV1(old uint64) uint64 {
	out := old & (1<<20 - 1)
	for prf, bit := range flagsV1PRF {
		if old&(1<<uint(bit)) != 0 {
			out |= 1 << uint(prfBase+prf)
		}
	}
	return out
}
