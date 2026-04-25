// Ported from src/ban.c — banning/unbanning/checking sites and player names.
//
// Original CircleMUD code copyright (c) 1993, 94 by the Trustees of the Johns
// Hopkins University. CircleMUD is based on DikuMUD, Copyright (c) 1990, 1991.
//
// Dark Pawns modifications copyright (c) 1996, 97, 98 by the Dark Pawns
// Coding Team. All rights reserved.

package game

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Ban type constants — from src/ban.c
const (
	BanNot    = 0
	BanNew    = 1
	BanSelect = 2
	BanAll    = 3
)

// BanListEntry represents a single banned site/domain.
type BanListEntry struct {
	Site string    // Lowercased site/domain to ban
	Name string    // Name of the admin who created the ban
	Date time.Time // When the ban was created
	Type int       // Ban type (BanNot..BanAll)
}

// BanFile is the path to the banned sites file.
const BanFile = "data/banned"

// XnameFile is the path to the invalid name list.
const XnameFile = "data/xname"

// BannedSiteLength is the max length of a banned site string.
const BannedSiteLength = 80

var (
	mu           sync.RWMutex
	banList      []*BanListEntry
	invalidNames []string
)

// banTypeNames are the string labels for ban type constants.
var banTypeNames = []string{
	BanNot:    "no",
	BanNew:    "new",
	BanSelect: "select",
	BanAll:    "all",
}

// LoadBanned loads the ban list from BAN_FILE. Called at startup.
// Ported from load_banned() in ban.c.
func LoadBanned() error {
	mu.Lock()
	defer mu.Unlock()

	banList = nil

	f, err := os.Open(BanFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("opening ban file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Format: ban_type site_name date name
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		entry := &BanListEntry{
			Site: parts[1],
			Date: time.Unix(0, 0),
			Name: parts[3],
		}

		// Parse type
		for i, name := range banTypeNames {
			if parts[0] == name {
				entry.Type = i
				break
			}
		}

		// Parse date
		var unixTime int64
		_, _ = fmt.Sscanf(parts[2], "%d", &unixTime)
		if unixTime > 0 {
			entry.Date = time.Unix(unixTime, 0)
		}

		// Prepend (matching C: ban_list = next_node, next_node->next = ban_list)
		banList = append([]*BanListEntry{entry}, banList...)
	}

	if scanner.Err() != nil {
		return fmt.Errorf("reading ban file: %w", scanner.Err())
	}
	return nil
}

// IsBanned checks if a hostname matches any banned site.
// Returns the highest matching ban type, or BanNot if not banned.
// Ported from isbanned() in ban.c.
func IsBanned(hostname string) int {
	if hostname == "" {
		return BanNot
	}

	hostname = strings.ToLower(hostname)

	mu.RLock()
	defer mu.RUnlock()

	highest := BanNot
	for _, entry := range banList {
		if strings.Contains(hostname, entry.Site) {
			if entry.Type > highest {
				highest = entry.Type
			}
		}
	}
	return highest
}

// WriteBanList persists the ban list to BAN_FILE.
// Ported from write_ban_list() in ban.c — writes in reverse order (tail→head).
func WriteBanList() error {
	mu.RLock()
	defer mu.RUnlock()

	// Reversed copy — C writes from tail to head recursively
	reversed := make([]*BanListEntry, len(banList))
	for i, entry := range banList {
		reversed[len(banList)-1-i] = entry
	}

	return writeBanListLocked(reversed)
}

func writeBanListLocked(entries []*BanListEntry) error {
	f, err := os.Create(BanFile)
	if err != nil {
		return fmt.Errorf("writing ban file: %w", err)
	}
	defer f.Close()

	for _, entry := range entries {
		banType := "no"
		if entry.Type >= 0 && entry.Type < len(banTypeNames) {
			banType = banTypeNames[entry.Type]
		}
		var unixTime int64
		if !entry.Date.IsZero() {
			unixTime = entry.Date.Unix()
		}
		_, _ = fmt.Fprintf(f, "%s %s %d %s\n", banType, entry.Site, unixTime, entry.Name)
	}
	return nil
}

// AddBan adds a site to the ban list. Returns error if already banned.
// Ported from do_ban() add logic.
func AddBan(site, adminName, flag string) error {
	flagLower := strings.ToLower(flag)
	banType := -1
	for i := BanNew; i <= BanAll; i++ {
		if banTypeNames[i] == flagLower {
			banType = i
			break
		}
	}
	if banType < 0 {
		return fmt.Errorf("flag must be ALL, SELECT, or NEW")
	}

	siteLower := strings.ToLower(site)
	if len(siteLower) > BannedSiteLength {
		siteLower = siteLower[:BannedSiteLength]
	}

	mu.Lock()
	defer mu.Unlock()

	// Check for duplicate
	for _, entry := range banList {
		if strings.EqualFold(entry.Site, site) {
			return fmt.Errorf("that site has already been banned — unban it to change the ban type")
		}
	}

	entry := &BanListEntry{
		Site: siteLower,
		Name: adminName,
		Date: time.Now(),
		Type: banType,
	}

	banList = append([]*BanListEntry{entry}, banList...)
	return writeBanListLocked(banList)
}

// RemoveBan removes a site from the ban list. Returns error if not found.
// Ported from do_unban().
func RemoveBan(site string) error {
	mu.Lock()
	defer mu.Unlock()

	for i, entry := range banList {
		if strings.EqualFold(entry.Site, site) {
			banList = append(banList[:i], banList[i+1:]...)
			return writeBanListLocked(banList)
		}
	}
	return fmt.Errorf("that site is not currently banned")
}

// ListBans returns a copy of the current ban list for display.
func ListBans() []BanListEntry {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]BanListEntry, len(banList))
	for i, entry := range banList {
		result[i] = *entry
	}
	return result
}

// BanListSummary returns a formatted table of all bans.
func BanListSummary() string {
	mu.RLock()
	defer mu.RUnlock()

	if len(banList) == 0 {
		return "No sites are banned."
	}

	sorted := make([]*BanListEntry, len(banList))
	copy(sorted, banList)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Site < sorted[j].Site
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-25s  %-8s  %-10s  %-16s\r\n",
		"Banned Site Name", "Ban Type", "Banned On", "Banned By"))
	b.WriteString(strings.Repeat("-", 25) + "  " +
		strings.Repeat("-", 8) + "  " +
		strings.Repeat("-", 10) + "  " +
		strings.Repeat("-", 16) + "\r\n")

	for _, entry := range sorted {
		dateStr := "Unknown"
		if !entry.Date.IsZero() {
			dateStr = entry.Date.Format("2006-01-02")
		}
		b.WriteString(fmt.Sprintf("%-25s  %-8s  %-10s  %-16s\r\n",
			entry.Site, BanTypeName(entry.Type), dateStr, entry.Name))
	}
	return b.String()
}

// BanTypeName returns the string name of a ban type constant.
func BanTypeName(banType int) string {
	if banType >= 0 && banType < len(banTypeNames) {
		return banTypeNames[banType]
	}
	return "ERROR"
}

// ReadInvalidList loads the invalid name list from disk.
// Ported from Read_Invalid_List() in ban.c.
func ReadInvalidList() error {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.Open(XnameFile)
	if err != nil {
		if os.IsNotExist(err) {
			invalidNames = nil
			return nil
		}
		return fmt.Errorf("opening invalid name file: %w", err)
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\n\r\t ")
		if len(line) > 1 {
			names = append(names, strings.ToLower(line))
		}
	}
	if scanner.Err() != nil {
		return fmt.Errorf("reading invalid name file: %w", scanner.Err())
	}
	invalidNames = names
	return nil
}

// HasActiveCharacter is a callback set by the session layer to check if
// a player name is currently logged in and playing.
var HasActiveCharacter func(name string) bool

// ValidName checks if a character name is valid:
//   1. Check for active login (same name logged in)
//   2. Check against invalid name list (profanity filter)
// Ported from Valid_Name() in ban.c.
func ValidName(name string) bool {
	if name == "" {
		return false
	}

	// Check for duplicate/active login
	if HasActiveCharacter != nil && HasActiveCharacter(name) {
		return true
	}

	mu.RLock()
	defer mu.RUnlock()

	if len(invalidNames) == 0 {
		return true
	}

	tempName := strings.Map(func(r rune) rune {
		return unicode.ToLower(r)
	}, name)

	for _, inv := range invalidNames {
		if strings.Contains(tempName, inv) {
			return false
		}
	}

	return true
}


