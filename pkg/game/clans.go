// Package game — clan system, ported from src/clan.c
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Clan constants
// ---------------------------------------------------------------------------

const (
	MaxClans       = 20
	DefaultAppLvl  = 8
	ClanPlanLength = 1024
	NumCP          = 8
	ClanMaxRanks   = 20
)

// Clan privilege indices
const (
	CPSetPlan   = 0
	CPEnroll    = 1
	CPExpel     = 2
	CPPromote   = 3
	CPDemote    = 4
	CPSetFees   = 5
	CPWithdraw  = 6
	CPSetAppLev = 7
)

// Clan money actions
const (
	CMDues   = 1
	CMAppFee = 2
)

// Clan bank actions
const (
	CBDeposit  = 1
	CBWithdraw = 2
)

// Clan privacy
const (
	ClanPublic  = 0
	ClanPrivate = 1
)

// clanPrivileges names
var clanPrivileges = [NumCP + 1]string{
	"setplan", "enroll", "expel", "promote",
	"demote", "setfees", "withdraw", "setapplev",
}

// ---------------------------------------------------------------------------
// Clan data structure
// ---------------------------------------------------------------------------

type Clan struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Ranks       int        `json:"ranks"`
	RankName    [20]string `json:"rank_name"`
	Treasure    int64      `json:"treasure"`
	Members     int        `json:"members"`
	Power       int        `json:"power"`
	AppFee      int        `json:"app_fee"`
	Dues        int        `json:"dues"`
	Spells      [5]int     `json:"spells"`
	ApplLevel   int        `json:"app_level"`
	Privilege   [20]int    `json:"privilege"`
	AtWar       [4]int     `json:"at_war"`
	Plan        string     `json:"plan"`
	Description string     `json:"description"`
	Private     int        `json:"private"`
}

// ---------------------------------------------------------------------------
// ClanManager
// ---------------------------------------------------------------------------

// CanWithdraw returns true if the player has sufficient clan rank to withdraw.
func (c *Clan) CanWithdraw(ch *Player) bool {
	if ch.Level >= LVL_IMMORT {
		return true
	}
	return ch.ClanRank >= c.Privilege[CPWithdraw]
}

type ClanManager struct {
	mu     sync.RWMutex
	Clans  []*Clan `json:"clans"`
	nextID int
}

func NewClanManager() *ClanManager {
	return &ClanManager{
		Clans:  make([]*Clan, 0, MaxClans),
		nextID: 1,
	}
}

// ---------------------------------------------------------------------------
// ClanManager methods
// ---------------------------------------------------------------------------

func (m *ClanManager) FindClanByID(id int) (int, *Clan) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i, c := range m.Clans {
		if c.ID == id {
			return i, c
		}
	}
	return -1, nil
}

func (m *ClanManager) FindClan(name string) (int, *Clan) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i, c := range m.Clans {
		if strings.EqualFold(c.Name, name) {
			return i, c
		}
	}
	return -1, nil
}

func (m *ClanManager) SaveClans(filePath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Before saving, ensure plan is written to description
	for _, c := range m.Clans {
		if c.Plan != "" {
			if len(c.Plan) > ClanPlanLength-1 {
				c.Description = c.Plan[:ClanPlanLength-1]
			} else {
				c.Description = c.Plan
			}
		} else {
			c.Description = ""
		}
	}
	dir := filepath.Dir(filePath)
	// #nosec G301
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.Clans, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o600)
}

func InitClans(filePath string) *ClanManager {
	m := NewClanManager()
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		if os.IsNotExist(err) {
			if saveErr := m.SaveClans(filePath); saveErr != nil {
				BasicMudLogf("SYSERR: Failed to create new clan file: %v", saveErr)
			}
		} else {
			BasicMudLogf("SYSERR: Unable to read clan file: %v", err)
		}
		return m
	}

	var clans []*Clan
	if err := json.Unmarshal(data, &clans); err != nil {
		BasicMudLogf("SYSERR: Unable to parse clan file: %v", err)
		return m
	}

	m.Clans = clans
	for _, c := range m.Clans {
		if c.ID >= m.nextID {
			m.nextID = c.ID + 1
		}
		// Restore plan from description
		if c.Description != "" {
			c.Plan = c.Description
		}
	}

	// Reset cached member counts and recalculate from player DB
	for _, c := range m.Clans {
		c.Members = 0
		c.Power = 0
	}
	clanMap := make(map[int]*Clan)
	for _, c := range m.Clans {
		clanMap[c.ID] = c
	}
	files, _ := os.ReadDir(saveDir)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			name := strings.TrimSuffix(f.Name(), ".json")
			p, err := LoadPlayer(name)
			if err != nil {
				continue
			}
			if p.ClanID != 0 {
				if c, ok := clanMap[p.ClanID]; ok {
					c.Members++
					c.Power += p.Level
				}
			}
		}
	}

	BasicMudLogf("   Loaded %d clans.", len(m.Clans))
	return m
}

func (m *ClanManager) ClanCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Clans)
}

func (m *ClanManager) AddClan(c *Clan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID = m.nextID
	m.nextID++
	m.Clans = append(m.Clans, c)
}

func (m *ClanManager) RemoveClan(idx int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Clans = append(m.Clans[:idx], m.Clans[idx+1:]...)
}

func (m *ClanManager) GetClanByIndex(idx int) *Clan {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if idx < 0 || idx >= len(m.Clans) {
		return nil
	}
	return m.Clans[idx]
}

// ---------------------------------------------------------------------------
// halfChop in act_comm.go, atoi in act_item.go, isNumber in spec_procs4.go

// isClanNumber mirrors C's is_number in src/interpreter.c:1175. Unlike the
// permissive fmt.Sscanf helper used by older Go command code, C accepts only
// decimal digit bytes (and, historically, also treats the empty string as a
// number; callers in clan.c guard empty arguments where needed).
func isClanNumber(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// capClanName mirrors C's CAP macro (src/utils.h:166): uppercase only the
// first byte and preserve the remainder exactly.
func capClanName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ClanFilePath returns the default path for the clan data file.
func ClanFilePath() string {
	return "./data/clans.json"
}

func (w *World) SaveClans() {
	if w.Clans == nil {
		return
	}
	if err := w.Clans.SaveClans(ClanFilePath()); err != nil {
		BasicMudLogf("SYSERR: Failed to save clans: %v", err)
	}
}

// BeginClanPlanWrite starts the C string_write equivalent for do_clan_plan.
func (w *World) BeginClanPlanWrite(ch *Player, c *Clan) {
	ch.ClanPlanWriting = true
	ch.ClanPlanClanID = c.ID
	ch.ClanPlanBuffer = ""
	ch.SetPlrFlag(PlrWriting, true)
}

// HandleClanPlanInput consumes one line while the clan plan editor is active.
// It returns true when the line belonged to the editor, including the final @.
func (w *World) HandleClanPlanInput(ch *Player, line string) bool {
	if !ch.ClanPlanWriting {
		return false
	}
	if strings.HasPrefix(line, "@") {
		if _, c := w.Clans.FindClanByID(ch.ClanPlanClanID); c != nil {
			c.Plan = ch.ClanPlanBuffer
			w.SaveClans()
		}
		ch.ClanPlanWriting = false
		ch.ClanPlanClanID = 0
		ch.ClanPlanBuffer = ""
		ch.SetPlrFlag(PlrWriting, false)
		return true
	}
	if ch.ClanPlanBuffer == "" {
		ch.ClanPlanBuffer = line + "\r\n"
	} else {
		ch.ClanPlanBuffer += line + "\r\n"
	}
	return true
}

// ---------------------------------------------------------------------------
// sendClanFormat
// ---------------------------------------------------------------------------

func (w *World) sendClanFormat(ch *Player) {
	// Guard against an uninitialized clan manager (e.g. a minimal/harness world
	// that never loaded data/clans.json). C clan.c:find_clan_by_id returns -1
	// for a missing clan, so a nil manager is equivalent to "no clans loaded"
	// → cIdx = -1 (no clan), which is the right answer for a clanless mortal.
	cIdx := -1
	if w.Clans != nil {
		cIdx, _ = w.Clans.FindClanByID(ch.ClanID)
	}
	r := ch.ClanRank

	ch.SendMessage("Clan commands available to you:\r\n" +
		"   clan status\r\n" +
		"   clan info <clan>\r\n")

	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("   clan create     <leader> <clan name>\r\n" +
			"   clan destroy    <clan>\r\n" +
			"   clan rename     <#> <name>\r\n" +
			"   clan enroll     <player> <clan>\r\n" +
			"   clan expel      <player> <clan>\r\n" +
			"   clan promote    <player> <clan>\r\n" +
			"   clan demote     <player> <clan>\r\n" +
			"   clan withdraw   <amount> <clan>\r\n" +
			"   clan deposit    <amount> <clan>\r\n" +
			"   clan set ranks  <rank>   <clan>\r\n" +
			"   clan set appfee <amount> <clan>\r\n" +
			"   clan set dues   <amount> <clan>\r\n" +
			"   clan set applev <level>  <clan>\r\n" +
			"   clan set plan   <clan>\r\n" +
			"   clan private <clan>\r\n" +
			"   clan set privilege  <privilege>   <rank> <clan>\r\n" +
			"   clan set title  <clan number> <rank> <title>\r\n")
	} else {
		if ch.ClanID == 0 {
			ch.SendMessage("   clan apply      <clan>\r\n")
		}
		if r > 0 && cIdx >= 0 {
			c := w.Clans.GetClanByIndex(cIdx)
			if c != nil {
				ch.SendMessage("   clan who\r\n")
				ch.SendMessage("   clan members\r\n")
				ch.SendMessage("   clan quit\r\n")
				ch.SendMessage("   clan deposit    <amount>\r\n")
				if r >= c.Privilege[CPWithdraw] {
					ch.SendMessage("   clan withdraw   <amount>\r\n")
				}
				if r >= c.Privilege[CPEnroll] {
					ch.SendMessage("   clan enroll     <player>\r\n")
				}
				if r >= c.Privilege[CPExpel] {
					ch.SendMessage("   clan expel      <player>\r\n")
				}
				if r >= c.Privilege[CPPromote] {
					ch.SendMessage("   clan promote    <player>\r\n")
				}
				if r >= c.Privilege[CPDemote] {
					ch.SendMessage("   clan demote     <player>\r\n")
				}
				if r >= c.Privilege[CPSetAppLev] {
					ch.SendMessage("   clan set applev <level>\r\n")
				}
				if r >= c.Privilege[CPSetFees] {
					ch.SendMessage("   clan set appfee <amount>\r\n" +
						"   clan set dues   <amount>\r\n")
				}
				if r >= c.Privilege[CPSetPlan] {
					ch.SendMessage("   clan set plan\r\n")
				}
				if r == c.Ranks {
					ch.SendMessage("   clan private\r\n" +
						"   clan set ranks  <rank>\r\n" +
						"   clan set title  <rank> <title>\r\n" +
						"   clan set privilege  <privilege> <rank>\r\n")
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Resolve clan context for immortal commands.
// ---------------------------------------------------------------------------

func (w *World) resolveClanForImmortal(ch *Player, argument string) (int, *Clan, bool, bool) {
	if ch.Level < LVL_IMMORT {
		clanNum, c := w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return -1, nil, false, false
		}
		return clanNum, c, false, true
	}
	if ch.Level < LVL_GOD {
		ch.SendMessage("You do not have clan privileges.\r\n")
		return -1, nil, false, false
	}
	_, arg2 := halfChop(argument)
	clanNum, c := w.Clans.FindClan(arg2)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return -1, nil, false, false
	}
	return clanNum, c, true, true
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_rename
// ---------------------------------------------------------------------------
