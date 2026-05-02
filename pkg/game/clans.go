// Package game — clan system, ported from src/clan.c
//nolint:unused // Clan system port — helpers not yet wired.
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.Clans, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
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

// ---------------------------------------------------------------------------
// sendClanFormat
// ---------------------------------------------------------------------------

func (w *World) sendClanFormat(ch *Player) {
	cIdx, _ := w.Clans.FindClanByID(ch.ClanID)
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
// Find helpers (package-level for clan lookup by name/id)
// ---------------------------------------------------------------------------

func (m *ClanManager) findClanOrError(clanNum int) (*Clan, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if clanNum < 0 || clanNum >= len(m.Clans) {
		return nil, false
	}
	return m.Clans[clanNum], true
}

// ---------------------------------------------------------------------------
// Resolve clan context for mortal vs immortal commands.
// Returns (clanIndex, clannum, clan, isImmortal, ok).
// For mortals: uses the player's own clan. For immortals: parses optional <clan> from arg.
// ---------------------------------------------------------------------------

func (w *World) resolveClanContext(ch *Player, argument string, requirePrivilege int) (int, *Clan, bool) {
	var clanNum int
	var c *Clan

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return -1, nil, false
		}
		if requirePrivilege >= 0 && ch.ClanRank < c.Privilege[requirePrivilege] {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return -1, nil, false
		}
		return clanNum, c, true
	}

	// Immortal route: parse optional clan name from argument
	// immcom is handled by return value
	if ch.Level < LVL_GOD {
		ch.SendMessage("You do not have clan privileges.\r\n")
		return -1, nil, false
	}
	_, arg2 := halfChop(argument)
	if arg2 == "" {
		ch.SendMessage("Unknown clan.\r\n")
		return -1, nil, false
	}
	clanNum, c = w.Clans.FindClan(arg2)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return -1, nil, false
	}
	return clanNum, c, true
}

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

