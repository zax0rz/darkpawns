// Package game — clan system, ported from src/clan.c
package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
// #nosec G306
	return os.WriteFile(filePath, data, 0644)
}

func InitClans(filePath string) *ClanManager {
	m := NewClanManager()
// #nosec G304
	data, err := os.ReadFile(filePath)
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

func (w *World) doClanRename(ch *Player, arg string) {
	arg1, arg2 := halfChop(arg)

	if !isNumber(arg1) {
		ch.SendMessage("You need to specify a clan number.\r\n")
		return
	}
	clanIdx, _ := strconv.Atoi(arg1)
	if clanIdx < 0 || clanIdx >= w.Clans.ClanCount() {
		ch.SendMessage("There is no clan with that number.\r\n")
		return
	}

	if arg2 == "" {
		ch.SendMessage("What do you want to rename it?\r\n")
		return
	}

	c := w.Clans.GetClanByIndex(clanIdx)
	if c == nil {
		ch.SendMessage("There is no clan with that number.\r\n")
		return
	}
	if len(arg2) > 32 {
		arg2 = arg2[:32]
	}
	c.Name = strings.Title(strings.ToLower(arg2))
	w.SaveClans()
	ch.SendMessage("Clan renamed.\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_create
// ---------------------------------------------------------------------------

func (w *World) doClanCreate(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level < LVL_IMMORT {
		ch.SendMessage("You are not mighty enough to create new clans!\r\n")
		return
	}
	if w.Clans.ClanCount() >= MaxClans {
		ch.SendMessage("Max clans reached. WOW!\r\n")
		return
	}

	arg1, arg2 := halfChop(arg)

	leader, hasLeader := w.GetPlayer(arg1)
	if !hasLeader {
		ch.SendMessage("The leader of the new clan must be present.\r\n")
		return
	}

	if len(arg2) >= 32 {
		ch.SendMessage("Clan name too long! (32 characters max)\r\n")
		return
	}
	if leader.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot set an immortal as the leader of a clan.\r\n")
		return
	}
	if leader.ClanID != 0 && leader.ClanRank != 0 {
		ch.SendMessage("The leader already belongs to a clan!\r\n")
		return
	}

	if _, c := w.Clans.FindClan(arg2); c != nil {
		ch.SendMessage("That clan name already exists!\r\n")
		return
	}

	newClan := &Clan{
		Name:      strings.Title(strings.ToLower(arg2)),
		Ranks:     2,
		Members:   1,
		Power:     leader.Level,
		ApplLevel: DefaultAppLvl,
		Private:   ClanPublic,
	}
	newClan.RankName[0] = "Member"
	newClan.RankName[1] = "Leader"

	// All privileges default to leader rank
	for i := 0; i < 20; i++ {
		newClan.Privilege[i] = newClan.Ranks
	}

	w.Clans.AddClan(newClan)
	w.SaveClans()
	ch.SendMessage("Clan created.\r\n")

	// Assign leader
	leader.ClanID = newClan.ID
	leader.ClanRank = newClan.Ranks
	// Save player state (simplified)
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_destroy
// ---------------------------------------------------------------------------

func (w *World) doClanDestroy(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level < LVL_IMMORT {
		ch.SendMessage("Your not mighty enough to destroy clans!\r\n")
		return
	}

	i, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	// Clear clan from all online members
	for _, p := range w.players {
		if p.ClanID == c.ID {
			p.ClanID = 0
			p.ClanRank = 0
		}
	}

	// Remove clan
	w.Clans.RemoveClan(i)
	w.SaveClans()
	ch.SendMessage("Clan deleted.\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_enroll
// ---------------------------------------------------------------------------

func (w *World) doClanEnroll(ch *Player, arg string) {
	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	// If immortal, the arg already has the clan parsed out; for mortals argg is the arg
	var targetArg string
	if immcom {
		// arg was "player clan" — we already consumed clan via resolveClanForImmortal
		// Re-parse: first word is player, second is clan (already consumed)
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	// If no target, show applicant list
	if targetArg == "" {
		ch.SendMessage("The following players have applied to your clan:\r\n" +
			"-----------------------------------------------\r\n")
		for _, p := range w.players {
			if p.ClanID == c.ID && p.ClanRank == 0 {
				ch.SendMessage(fmt.Sprintf("%s\r\n", p.Name))
			}
		}
		return
	}

	if !immcom && ch.ClanRank < c.Privilege[CPEnroll] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}

	if victim.ClanID != c.ID {
		if victim.ClanRank > 0 {
			ch.SendMessage("They're already in a clan.\r\n")
			return
		}
		ch.SendMessage("They didn't request to join your clan.\r\n")
		return
	}

	if victim.ClanRank > 0 {
		ch.SendMessage("They're already in your clan.\r\n")
		return
	}
	if victim.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot enroll immortals in clans.\r\n")
		return
	}

	victim.ClanRank++
	c.Power += victim.Level
	c.Members++

	victim.SendMessage("You've been enrolled in the clan you chose!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_expel
// ---------------------------------------------------------------------------

func (w *World) doClanExpel(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPExpel] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if !immcom && victim.ClanRank >= ch.ClanRank {
		ch.SendMessage("You cannot kick out that person.\r\n")
		return
	}

	victim.ClanID = 0
	victim.ClanRank = 0
	c.Members--
	c.Power -= victim.Level

	victim.SendMessage("You've been kicked out of your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_promote
// ---------------------------------------------------------------------------

func (w *World) doClanPromote(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPPromote] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if victim.ClanRank == 0 {
		ch.SendMessage("They're not enrolled yet.\r\n")
		return
	}
	if !immcom && victim.ClanRank+1 > ch.ClanRank {
		ch.SendMessage("You cannot promote that person over your rank!\r\n")
		return
	}
	if victim.ClanRank == c.Ranks {
		ch.SendMessage("You cannot promote someone over the top rank!\r\n")
		return
	}

	victim.ClanRank++
	victim.SendMessage("You've been promoted within your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_demote
// ---------------------------------------------------------------------------

func (w *World) doClanDemote(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	_, c, immcom, ok := w.resolveClanForImmortal(ch, arg)
	if !ok {
		return
	}

	var targetArg string
	if immcom {
		arg1, _ := halfChop(arg)
		targetArg = arg1
	} else {
		targetArg = arg
	}

	if !immcom && ch.ClanRank < c.Privilege[CPDemote] {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	victim, hasVictim := w.GetPlayer(targetArg)
	if !hasVictim {
		ch.SendMessage("Er, Who ??\r\n")
		return
	}
	if victim.ClanID != c.ID {
		ch.SendMessage("They're not in your clan.\r\n")
		return
	}
	if victim.ClanRank == 1 {
		ch.SendMessage("They can't be demoted any further, use expel now.\r\n")
		return
	}
	if !immcom && victim.ClanRank >= ch.ClanRank {
		ch.SendMessage("You cannot demote a person of this rank!\r\n")
		return
	}

	victim.ClanRank--
	victim.SendMessage("You've been demoted within your clan!\r\n")
	ch.SendMessage("Done.\r\n")
	w.SaveClans()

}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_who
// ---------------------------------------------------------------------------

func (w *World) doClanWho(ch *Player) {
	if ch.ClanRank == 0 {
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	ch.SendMessage("\r\nClan members online\r\n" +
		"-------------------------\r\n")

	for _, p := range w.players {
		if p.ClanID == ch.ClanID && p.ClanRank > 0 && chCanSee(ch, p) {
			rankName := ""
			if p.ClanRank-1 >= 0 && p.ClanRank-1 < len(c.RankName) {
				rankName = c.RankName[p.ClanRank-1]
			}
			ch.SendMessage(fmt.Sprintf("%s %s\r\n", rankName, p.Name))
		}
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_members
// ---------------------------------------------------------------------------

func (w *World) doClanMembers(ch *Player) {
	if ch.ClanID == 0 || ch.ClanRank == 0 {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	// For now, only list online members (can't read all saved players without the pfile system)
	ch.SendMessage("\r\nList of your clan members (online)\r\n" +
		"-------------------------\r\n")

	for _, p := range w.players {
		if p.ClanID == ch.ClanID && p.ClanRank != 0 {
			rankName := ""
			if p.ClanRank-1 >= 0 && p.ClanRank-1 < len(c.RankName) {
				rankName = c.RankName[p.ClanRank-1]
			}
			ch.SendMessage(fmt.Sprintf("%s %s\r\n", rankName, p.Name))
		}
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_quit
// ---------------------------------------------------------------------------

func (w *World) doClanQuit(ch *Player) {
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("You cannot quit any clan!\r\n")
		return
	}

	clanNum, c := w.Clans.FindClanByID(ch.ClanID)
	if c == nil {
		ch.SendMessage("You aren't in a clan!\r\n")
		return
	}

	ch.ClanID = 0
	ch.ClanRank = 0
	c.Members--
	c.Power -= ch.Level

	if c.Members == 0 {
		w.Clans.RemoveClan(clanNum)
		ch.SendMessage("You've quit your clan and it has been disbanded.\r\n")
	} else {
		ch.SendMessage("You've quit your clan.\r\n")
	}

	w.SaveClans()
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_status
// ---------------------------------------------------------------------------

func (w *World) doClanStatus(ch *Player) {
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("You are immortal and cannot join any clan!\r\n")
		return
	}

	_, c := w.Clans.FindClanByID(ch.ClanID)

	if ch.ClanRank == 0 {
		if c != nil {
			ch.SendMessage(fmt.Sprintf("You applied to %s\r\n", c.Name))
			return
		}
		ch.SendMessage("You do not belong to a clan!\r\n")
		return
	}

	rankName := ""
	if c != nil && ch.ClanRank-1 >= 0 && ch.ClanRank-1 < len(c.RankName) {
		rankName = c.RankName[ch.ClanRank-1]
	}
	if c != nil {
		ch.SendMessage(fmt.Sprintf("You are %s (Rank %d) of %s\r\n",
			rankName, ch.ClanRank, c.Name))
	} else {
		ch.SendMessage(fmt.Sprintf("You are Rank %d of a clan", ch.ClanRank))
	}
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_apply
// ---------------------------------------------------------------------------

func (w *World) doClanApply(ch *Player, arg string) {
	if arg == "" {
		w.sendClanFormat(ch)
		return
	}
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("Gods cannot apply for any clan.\r\n")
		return
	}
	if ch.ClanRank > 0 {
		ch.SendMessage("You already belong to a clan!\r\n")
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	if ch.Level < c.ApplLevel {
		ch.SendMessage("You are not mighty enough to apply to this clan.\r\n")
		return
	}
	ch.GoldMu.Lock()
	if ch.Gold < c.AppFee {
		ch.GoldMu.Unlock()
		ch.SendMessage("You cannot afford the application fee!\r\n")
		return
	}

	ch.Gold -= c.AppFee
	ch.GoldMu.Unlock()
	c.Treasure += int64(c.AppFee)
	w.SaveClans()

	ch.ClanID = c.ID
	ch.SendMessage("You've applied to the clan!\r\n")
}

// ---------------------------------------------------------------------------
// Sub-command: do_clan_info
// ---------------------------------------------------------------------------

func (w *World) doClanInfo(ch *Player, arg string) {
	if w.Clans.ClanCount() == 0 {
		ch.SendMessage("No clans have formed yet.\r\n")
		return
	}

	if arg == "" {
		// Show all clans
		msg := "\r\n\t\tooO Clans of Dark Pawns Ooo\r\n"
		for i := 0; i < w.Clans.ClanCount(); i++ {
			c := w.Clans.GetClanByIndex(i)
			if c == nil {
				continue
			}
			if ch.Level >= LVL_IMMORT {
				msg += fmt.Sprintf("[%-2d]  %-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.ID, c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			} else if c.Private == 0 {
				msg += fmt.Sprintf("%-17s Members: %3d  Power: %3d  Appfee: %d Applvl: %d\r\n",
					c.Name, c.Members, c.Power, c.AppFee, c.ApplLevel)
			}
		}
		ch.SendMessage(msg)
		return
	}

	_, c := w.Clans.FindClan(arg)
	if c == nil {
		ch.SendMessage("Unknown clan.\r\n")
		return
	}

	msg := fmt.Sprintf("Info for the clan %s :\r\n", c.Name)
	msg += fmt.Sprintf("Ranks      : %d\r\nTitles     : ", c.Ranks)
	for j := 0; j < c.Ranks && j < len(c.RankName); j++ {
		msg += c.RankName[j] + " "
	}
	msg += fmt.Sprintf("\r\nMembers    : %d\r\nPower      : %d\r\nTreasure   : %d\r\nSpells     : ", c.Members, c.Power, c.Treasure)
	for j := 0; j < 5; j++ {
		if c.Spells[j] != 0 {
			msg += fmt.Sprintf("%d ", c.Spells[j])
		}
	}
	msg += "\r\n"
	msg += "Clan privileges:\r\n"
	for j := 0; j < NumCP; j++ {
		msg += fmt.Sprintf("   %-10s: %d\r\n", clanPrivileges[j], c.Privilege[j])
	}
	msg += "\r\n"
	msg += fmt.Sprintf("Description:\r\n%s\r\n\r\n", c.Plan)

	atWar := false
	for j := 0; j < 4; j++ {
		if c.AtWar[j] != 0 {
			atWar = true
			break
		}
	}
	if !atWar {
		msg += "This clan is at peace with all others.\r\n"
	} else {
		msg += "This clan is at war.\r\n"
	}
	ch.SendMessage(msg)
}

// ExecClanCommand dispatches the "clan" player command.
// In C: ACMD(do_clan)
func (w *World) ExecClanCommand(ch *Player, argument string) {
	arg1, arg2 := halfChop(argument)

	if isAbbrev(arg1, "rename") {
		w.doClanRename(ch, arg2)
		return
	}
	if isAbbrev(arg1, "create") {
		w.doClanCreate(ch, arg2)
		return
	}
	if isAbbrev(arg1, "destroy") {
		w.doClanDestroy(ch, arg2)
		return
	}
	if isAbbrev(arg1, "enroll") {
		w.doClanEnroll(ch, arg2)
		return
	}
	if isAbbrev(arg1, "expel") {
		w.doClanExpel(ch, arg2)
		return
	}
	if isAbbrev(arg1, "who") {
		w.doClanWho(ch)
		return
	}
	if isAbbrev(arg1, "status") {
		w.doClanStatus(ch)
		return
	}
	if isAbbrev(arg1, "info") {
		w.doClanInfo(ch, arg2)
		return
	}
	if isAbbrev(arg1, "apply") {
		w.doClanApply(ch, arg2)
		return
	}
	if isAbbrev(arg1, "demote") {
		w.doClanDemote(ch, arg2)
		return
	}
	if isAbbrev(arg1, "promote") {
		w.doClanPromote(ch, arg2)
		return
	}
	if isAbbrev(arg1, "members") {
		w.doClanMembers(ch)
		return
	}
	if isAbbrev(arg1, "quit") {
		w.doClanQuit(ch)
		return
	}
	if isAbbrev(arg1, "set") {
		w.doClanSet(ch, arg2)
		return
	}
	if isAbbrev(arg1, "private") {
		w.doClanPrivate(ch, arg2)
		return
	}
	if isAbbrev(arg1, "withdraw") {
		w.doClanBank(ch, arg2, CBWithdraw)
		return
	}
	if isAbbrev(arg1, "deposit") {
		w.doClanBank(ch, arg2, CBDeposit)
		return
	}

	w.sendClanFormat(ch)
}

// doClanBank handles bank operations for a clan (deposit/withdraw).
// In C: do_clan_bank()
func (w *World) doClanBank(ch *Player, arg string, action int) {
	var clanNum int
	var c *Clan

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if action == CBWithdraw && !c.CanWithdraw(ch) {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		switch action {
		case CBDeposit:
			ch.SendMessage("Deposit how much?\r\n")
		case CBWithdraw:
			ch.SendMessage("Withdraw how much?\r\n")
		default:
			ch.SendMessage("Bad clan banking call, please report to a God.\r\n")
		}
		return
	}

	if !isNumber(arg) {
		switch action {
		case CBDeposit:
			ch.SendMessage("Deposit what?\r\n")
		case CBWithdraw:
			ch.SendMessage("Withdraw what?\r\n")
		default:
			ch.SendMessage("Bad clan banking call, please report to a God.\r\n")
		}
		return
	}

	amount, _ := strconv.Atoi(arg)
	if amount <= 0 {
		ch.SendMessage("Amount must be positive.\r\n")
		return
	}

	_ = clanNum
	switch action {
	case CBWithdraw:
		if c.Treasure < int64(amount) {
			ch.SendMessage("The clan is not wealthy enough for your needs!\r\n")
			return
		}
		ch.SetGold(ch.GetGold() + amount)
		c.Treasure -= int64(amount)
		ch.SendMessage("You withdraw from the clan's treasure.\r\n")
	case CBDeposit:
		if ch.GetGold() < amount {
			ch.SendMessage("You do not have that kind of money!\r\n")
			return
		}
		ch.SetGold(ch.GetGold() - amount)
		c.Treasure += int64(amount)
		ch.SendMessage("You add to the clan's treasure.\r\n")
	}

	w.SaveClans()
}

// doClanPrivate toggles a clan between public and private room access.
// In C: do_clan_private()
func (w *World) doClanPrivate(ch *Player, arg string) {
	var clanNum int
	var c *Clan

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
		if ch.ClanRank != c.Ranks {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		clanNum, c = w.Clans.FindClan(arg)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	_ = clanNum

	if c.Private == ClanPublic {
		c.Private = ClanPrivate
		ch.SendMessage("Your clan is now private.\r\n")
		w.SaveClans()
		return
	}

	if c.Private == ClanPrivate {
		c.Private = ClanPublic
		ch.SendMessage("Your clan is now public.\r\n")
		w.SaveClans()
		return
	}
}

// doClanPlan shows/edits the clan's plan (description).
// In C: do_clan_plan() — uses string_write (descriptor-based editor).
//
// string_write is a descriptor-based multi-line editor activated by setting
// ch.WriteMagic and storing a callback. The session layer intercepts all
// subsequent input until a line containing only "@" is received, then calls
// the callback with the accumulated text.
//
// For now the plan is set via simple single-argument input from the god
// command, or cleared in anticipation of session-layer editor support.
// When the session editor layer is wired (string_write in session/), set
// ch.WriteMagic = magicToken and register a callback that writes the
// accumulated text into c.Plan.
func (w *World) doClanPlan(ch *Player, arg string) {
	var clanNum int
	var c *Clan

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
		if ch.ClanRank < c.Privilege[CPSetPlan] {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		if arg == "" {
			w.sendClanFormat(ch)
			return
		}
		clanNum, c = w.Clans.FindClan(arg)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	_ = clanNum

	if c.Plan != "" {
		ch.SendMessage(fmt.Sprintf("Old plan for clan <<%s>>:\r\n%s\r\n", c.Name, c.Plan))
	}
	ch.SendMessage(fmt.Sprintf("Enter the description, or plan for clan <<%s>>.\r\n", c.Name))
	ch.SendMessage("End with @ on a line by itself.\r\n")
	c.Plan = ""
	// C uses string_write(ch->desc, &clan[clan_num].plan, CLAN_PLAN_LENGTH, 0, NULL)
	// The session layer should check ch.WriteMagic after each input line
	// and accumulate text until "@" alone appears, then call the registered
	// callback. Once wired, this function would set:
	//   ch.WriteMagic = someToken
	// and register a callback that assigns c.Plan = accumulatedText and
	// calls w.SaveClans().
	w.SaveClans()
}

// doClanRanks manages clan rank names and adjusts existing members' ranks.
// In C: do_clan_ranks()
func (w *World) doClanRanks(ch *Player, arg string) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		a1, _ := halfChop(arg)
		arg = a1
		clanNum, c = w.Clans.FindClan(a1)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank != c.Ranks && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set how many ranks?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the ranks to what?\r\n")
		return
	}

	newRanks, _ := strconv.Atoi(arg)

	if newRanks == c.Ranks {
		ch.SendMessage("The clan already has this number of ranks.\r\n")
		return
	}

	if newRanks < 2 || newRanks > 20 {
		ch.SendMessage("Clans must have from 2 to 20 ranks.\r\n")
		return
	}

	if ch.Gold < 5000 && !immcom {
		ch.SendMessage("Changing the clan hierarchy requires 5,000 coins!\r\n")
		return
	}

	if !immcom {
		ch.Gold -= 5000
	}

	// Adjust existing clan members' ranks
	for _, p := range w.allPlayers() {
		if p.ClanID == c.ID {
			if p.ClanRank < c.Ranks && p.ClanRank > 0 {
				p.ClanRank = 1
			}
			if p.ClanRank == c.Ranks {
				p.ClanRank = newRanks
			}
		}
	}

	_ = clanNum

	c.Ranks = newRanks
	for i := 0; i < c.Ranks-1; i++ {
		c.RankName[i] = "Member"
	}
	c.RankName[c.Ranks-1] = "Leader"
	for i := 0; i < NumCP; i++ {
		c.Privilege[i] = newRanks
	}

	w.SaveClans()
}

// doClanTitles manages clan rank titles.
// In C: do_clan_titles()
func (w *World) doClanTitles(ch *Player, arg string) {
	var clanNum int
	var c *Clan

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
		if ch.ClanRank != c.Ranks {
			ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		a1, a2 := halfChop(arg)
		arg = a2
		if !isNumber(a1) {
			ch.SendMessage("You need to specify a clan number.\r\n")
			return
		}
		idx, _ := strconv.Atoi(a1)
		c = w.Clans.GetClanByIndex(idx)
		if c == nil {
			ch.SendMessage("There is no clan with that number.\r\n")
			return
		}
		clanNum = idx
	}

	a1, a2 := halfChop(arg)

	if !isNumber(a1) {
		ch.SendMessage("You need to specify a rank number.\r\n")
		return
	}

	rank, _ := strconv.Atoi(a1)

	if rank < 1 || rank > c.Ranks {
		ch.SendMessage("This clan has no such rank number.\r\n")
		return
	}

	if len(a2) < 1 || len(a2) > 19 {
		ch.SendMessage("You need a clan title of under 20 characters.\r\n")
		return
	}

	_ = clanNum

	c.RankName[rank-1] = a2
	w.SaveClans()
	ch.SendMessage("Done.\r\n")
}

// doClanPrivilege manages clan privilege levels for ranks.
// In C: do_clan_privilege()
func (w *World) doClanPrivilege(ch *Player, arg string) {
	a1, a2 := halfChop(arg)

	if isAbbrev(a1, "setplan") {
		w.doClanSP(ch, a2, CPSetPlan)
		return
	}
	if isAbbrev(a1, "enroll") {
		w.doClanSP(ch, a2, CPEnroll)
		return
	}
	if isAbbrev(a1, "expel") {
		w.doClanSP(ch, a2, CPExpel)
		return
	}
	if isAbbrev(a1, "promote") {
		w.doClanSP(ch, a2, CPPromote)
		return
	}
	if isAbbrev(a1, "demote") {
		w.doClanSP(ch, a2, CPDemote)
		return
	}
	if isAbbrev(a1, "withdraw") {
		w.doClanSP(ch, a2, CPWithdraw)
		return
	}
	if isAbbrev(a1, "setfees") {
		w.doClanSP(ch, a2, CPSetFees)
		return
	}
	if isAbbrev(a1, "setapplev") {
		w.doClanSP(ch, a2, CPSetAppLev)
		return
	}

	ch.SendMessage("\r\nClan privileges:\r\n")
	for i := 0; i < NumCP; i++ {
		ch.SendMessage(fmt.Sprintf("\t%s\r\n", clanPrivileges[i]))
	}
}

// doClanSP manages a single clan privilege for a rank.
// In C: do_clan_sp()
func (w *World) doClanSP(ch *Player, arg string, priv int) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		arg1, _ := halfChop(arg)
		arg = arg1
		// In C: uses arg1 (the clan name) for find_clan, same arg updated
		clanNum, c = w.Clans.FindClan(arg1)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank != c.Ranks && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set the privilege to which rank?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the privilege to what?\r\n")
		return
	}

	rank, _ := strconv.Atoi(arg)

	if rank < 1 || rank > c.Ranks {
		ch.SendMessage("There is no such rank in the clan.\r\n")
		return
	}

	_ = clanNum

	c.Privilege[priv] = rank
	w.SaveClans()
}

// doClanMoney manages clan dues and app fees.
// In C: do_clan_money()
func (w *World) doClanMoney(ch *Player, arg string, action int) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		arg1, arg2 := halfChop(arg)
		arg = arg1
		clanNum, c = w.Clans.FindClan(arg2)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank < c.Privilege[CPSetFees] && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set it to how much?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set it to what?\r\n")
		return
	}

	amount, _ := strconv.Atoi(arg)

	if amount < 0 || amount > 10000 {
		ch.SendMessage("Please pick a number between 0 and 10,000 coins.\r\n")
		return
	}

	_ = clanNum

	switch action {
	case CMAppFee:
		c.AppFee = amount
		ch.SendMessage("You change the application fee.\r\n")
	case CMDues:
		c.Dues = amount
		ch.SendMessage("You change the monthly dues.\r\n")
	default:
		ch.SendMessage("Problem in command, please report.\r\n")
	}

	w.SaveClans()
}

// doClanAppLevel manages clan application level requirements.
// In C: do_clan_application()
func (w *World) doClanAppLevel(ch *Player, arg string) {
	var clanNum int
	var c *Clan
	var immcom bool

	if arg == "" {
		w.sendClanFormat(ch)
		return
	}

	if ch.Level < LVL_IMMORT {
		clanNum, c = w.Clans.FindClanByID(ch.ClanID)
		if c == nil {
			ch.SendMessage("You don't belong to any clan!\r\n")
			return
		}
	} else {
		if ch.Level < LVL_GOD {
			ch.SendMessage("You do not have clan privileges.\r\n")
			return
		}
		immcom = true
		arg1, arg2 := halfChop(arg)
		arg = arg1
		clanNum, c = w.Clans.FindClan(arg2)
		if c == nil {
			ch.SendMessage("Unknown clan.\r\n")
			return
		}
	}

	if ch.ClanRank < c.Privilege[CPSetAppLev] && !immcom {
		ch.SendMessage("You're not influent enough in the clan to do that!\r\n")
		return
	}

	if arg == "" {
		ch.SendMessage("Set to which level?\r\n")
		return
	}

	if !isNumber(arg) {
		ch.SendMessage("Set the application level to what?\r\n")
		return
	}

	appLevel, _ := strconv.Atoi(arg)

	if appLevel < 1 || appLevel > 30 {
		ch.SendMessage("The application level can go from 1 to 30.\r\n")
		return
	}

	_ = clanNum

	c.ApplLevel = appLevel
	w.SaveClans()
}

// doClanSet dispatches the "clan set" subcommand.
// In C: do_clan_set()
func (w *World) doClanSet(ch *Player, arg string) {
	a1, a2 := halfChop(arg)

	if isAbbrev(a1, "plan") {
		w.doClanPlan(ch, a2)
		return
	}
	if isAbbrev(a1, "ranks") {
		w.doClanRanks(ch, a2)
		return
	}
	if isAbbrev(a1, "title") {
		w.doClanTitles(ch, a2)
		return
	}
	if isAbbrev(a1, "privilege") {
		w.doClanPrivilege(ch, a2)
		return
	}
	if isAbbrev(a1, "dues") {
		w.doClanMoney(ch, a2, 1)
		return
	}
	if isAbbrev(a1, "appfee") {
		w.doClanMoney(ch, a2, 2)
		return
	}
	if isAbbrev(a1, "applev") {
		w.doClanAppLevel(ch, a2)
		return
	}

	w.sendClanFormat(ch)
}
