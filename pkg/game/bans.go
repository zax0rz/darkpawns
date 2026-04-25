// Package game — site ban system (ported from ban.c)
// Source: src/ban.c — load_banned(), _write_one_node(), write_ban_list(),
//         do_ban(), do_unban(), Read_Invalid_List(), Valid_Name()
// Port: Wave 13, 2026-04-25
package game

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Ban type constants.
// Source: db.h lines 204–207
const (
	BanNot    = 0 // Not banned (placeholder)
	BanNew    = 1 // New players banned (no new characters from this site)
	BanSelect = 2 // Only selected players may log in from this site
	BanAll    = 3 // All players banned from this site
)

// banTypeNames maps ban type integers to string labels.
// Source: ban.c ban_types[] — "no", "new", "select", "all", "ERROR"
var banTypeNames = []string{"no", "new", "select", "all", "ERROR"}

// BanEntry represents a single site ban record.
// Source: structs.h struct ban_list_element {
//   char site[BANNED_SITE_LENGTH+1]; char name[MAX_NAME_LENGTH+1];
//   long date; int type; struct ban_list_element *next;
// }
// BANNED_SITE_LENGTH = 50 (db.h line 209)
// MAX_NAME_LENGTH = 20 (structs.h)
type BanEntry struct {
	Site    string    // Hostname pattern to match (up to 50 chars)
	BanType int       // BanNot..BanAll
	Date    time.Time // When the ban was placed
	BannedBy string   // Name of the admin who placed the ban
}

// BanManager holds the active ban list and the invalid name list.
// Corresponds to the C globals ban_list and invalid_list / num_invalid.
type BanManager struct {
	bans         []BanEntry
	invalidNames []string // loaded by ReadInvalidList()
}

// NewBanManager creates an empty BanManager.
func NewBanManager() *BanManager {
	return &BanManager{}
}

// banTypeFromString maps a string ("no"/"new"/"select"/"all") to its int constant.
// Source: ban.c load_banned() inner loop lines 74–76
func banTypeFromString(s string) int {
	s = strings.ToLower(s)
	for i, name := range banTypeNames {
		if name == s {
			return i
		}
	}
	return BanNot
}

// banTypeName returns the string label for a ban type.
func banTypeName(t int) string {
	if t >= 0 && t < len(banTypeNames) {
		return banTypeNames[t]
	}
	return "ERROR"
}

// LoadBanned reads the ban file and populates the ban list.
// Source: ban.c load_banned() lines 52–83
//
// File format (one entry per line):
//   <ban_type> <site_name> <unix_timestamp> <banned_by_name>
func (bm *BanManager) LoadBanned(path string) {
	bm.bans = nil

	f, err := os.Open(path)
	if err != nil {
		slog.Warn("unable to open ban file", "path", path, "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var banTypeStr, site, name string
		var dateUnix int64
		n, err := fmt.Sscanf(line, "%s %s %d %s", &banTypeStr, &site, &dateUnix, &name)
		if err != nil || n != 4 {
			continue
		}
		entry := BanEntry{
			Site:     strings.ToLower(site),
			BanType:  banTypeFromString(banTypeStr),
			Date:     time.Unix(dateUnix, 0),
			BannedBy: name,
		}
		bm.bans = append(bm.bans, entry)
	}
}

// writeBanList writes the entire ban list to disk.
// Source: ban.c write_ban_list() lines 118–129
// The C version used a recursive _write_one_node() to write entries in reverse
// insertion order. We replicate that: write bans in reverse slice order so the
// most recently added entry ends up last in the file (head-insertion order).
func (bm *BanManager) WriteBanList(path string) error {
	if err := os.MkdirAll(pathDir(path), 0o755); err != nil {
		return fmt.Errorf("WriteBanList: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("WriteBanList: create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	// Write in reverse order to match C recursive _write_one_node() behavior.
	// Source: ban.c _write_one_node() — recurses to end then writes on unwind.
	for i := len(bm.bans) - 1; i >= 0; i-- {
		entry := bm.bans[i]
		fmt.Fprintf(w, "%s %s %d %s\n",
			banTypeName(entry.BanType),
			entry.Site,
			entry.Date.Unix(),
			entry.BannedBy,
		)
	}
	return w.Flush()
}

// IsBanned checks whether a hostname is banned and returns the effective ban level.
// Source: ban.c isbanned() lines 86–104
//
// Logic: lowercase the hostname, then for each ban entry check if the ban site
// is a substring of the hostname. Return the maximum ban level found.
// Returns 0 (BanNot) if hostname is empty or no match is found.
func (bm *BanManager) IsBanned(hostname string) int {
	if hostname == "" {
		return BanNot
	}
	lower := strings.ToLower(hostname)
	maxLevel := BanNot
	for _, entry := range bm.bans {
		if strings.Contains(lower, entry.Site) {
			if entry.BanType > maxLevel {
				maxLevel = entry.BanType
			}
		}
	}
	return maxLevel
}

// AddBan adds a new site ban entry (does not write to disk).
// Source: ban.c do_ban() lines 188–209
func (bm *BanManager) AddBan(site string, banType int, bannedBy string) error {
	site = strings.ToLower(site)
	// Check for duplicate — Source: ban.c do_ban() lines 182–187
	for _, entry := range bm.bans {
		if entry.Site == site {
			return fmt.Errorf("site %q is already banned; unban it first to change the type", site)
		}
	}
	bm.bans = append(bm.bans, BanEntry{
		Site:     site,
		BanType:  banType,
		Date:     time.Now(),
		BannedBy: bannedBy,
	})
	return nil
}

// RemoveBan removes a site ban entry. Returns an error if the site is not banned.
// Source: ban.c do_unban() lines 213–243
func (bm *BanManager) RemoveBan(site string) (*BanEntry, error) {
	site = strings.ToLower(site)
	for i, entry := range bm.bans {
		if entry.Site == site {
			removed := bm.bans[i]
			bm.bans = append(bm.bans[:i], bm.bans[i+1:]...)
			return &removed, nil
		}
	}
	return nil, fmt.Errorf("site %q is not currently banned", site)
}

// ListBans returns a formatted string listing all active bans.
// Source: ban.c do_ban() (no-argument path) lines 142–171
func (bm *BanManager) ListBans() string {
	if len(bm.bans) == 0 {
		return "No sites are banned.\r\n"
	}
	header := fmt.Sprintf("%-25.25s  %-8.8s  %-10.10s  %-16.16s\r\n",
		"Banned Site Name", "Ban Type", "Banned On", "Banned By")
	header += fmt.Sprintf("%-25.25s  %-8.8s  %-10.10s  %-16.16s\r\n",
		"-------------------------", "--------", "----------", "----------------")

	var sb strings.Builder
	sb.WriteString(header)
	for _, entry := range bm.bans {
		dateStr := "Unknown"
		if !entry.Date.IsZero() {
			dateStr = entry.Date.Format("2006-01-02")
		}
		sb.WriteString(fmt.Sprintf("%-25.25s  %-8.8s  %-10.10s  %-16.16s\r\n",
			entry.Site, banTypeName(entry.BanType), dateStr, entry.BannedBy))
	}
	return sb.String()
}

// ReadInvalidList loads the list of invalid/forbidden name substrings.
// Source: ban.c Read_Invalid_List() lines 289–313
//
// File format: one substring per line; lines shorter than 2 bytes are ignored.
// The C original truncates each name to MAX_NAME_LENGTH (20 chars) by reading
// with fgets(invalid_list[i], MAX_NAME_LENGTH, fp).
func (bm *BanManager) ReadInvalidList(path string) {
	bm.invalidNames = nil

	f, err := os.Open(path)
	if err != nil {
		slog.Warn("unable to open invalid name file", "path", path, "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 { // mirrors C: strlen(string) > 1
			continue
		}
		// Truncate to MAX_NAME_LENGTH (20) matching C behavior
		if len(line) > 20 {
			line = line[:20]
		}
		bm.invalidNames = append(bm.invalidNames, strings.ToLower(line))
	}
}

// ValidName returns true if the given name does not match any invalid substring
// and is not already in use by an online player.
// Source: ban.c Valid_Name() lines 257–286
//
// Logic: lowercase the candidate name, then check if any invalid_list entry
// appears as a substring of the name. Returns false if a match is found.
// The online-player duplicate check is handled at the session layer.
func (bm *BanManager) ValidName(name string) bool {
	if len(bm.invalidNames) == 0 {
		return true
	}
	lower := strings.ToLower(name)
	for _, invalid := range bm.invalidNames {
		if strings.Contains(lower, invalid) {
			return false
		}
	}
	return true
}

// DoBan handles the "ban" admin command.
// Source: ban.c do_ban() lines 132–210
// Arguments: "<flag> <site>" or "" to list bans.
// Returns the message to send to the player and an error if the command failed.
func (bm *BanManager) DoBan(banFilePath, playerName, argument string) string {
	argument = strings.TrimSpace(argument)
	if argument == "" {
		return bm.ListBans()
	}

	parts := strings.Fields(argument)
	if len(parts) < 2 {
		return "Usage: ban {all | select | new} site_name\r\n"
	}
	flag := strings.ToLower(parts[0])
	site := strings.ToLower(parts[1])

	if flag != "select" && flag != "all" && flag != "new" {
		return "Flag must be ALL, SELECT, or NEW.\r\n"
	}

	banType := banTypeFromString(flag)
	if err := bm.AddBan(site, banType, playerName); err != nil {
		return err.Error() + "\r\n"
	}

	if err := bm.WriteBanList(banFilePath); err != nil {
		slog.Error("failed to write ban list", "error", err)
	}

	slog.Info("ban added", "admin", playerName, "site", site, "type", flag)
	return "Site banned.\r\n"
}

// DoUnban handles the "unban" admin command.
// Source: ban.c do_unban() lines 213–244
// Returns the message to send to the player.
func (bm *BanManager) DoUnban(banFilePath, playerName, argument string) string {
	site := strings.TrimSpace(strings.ToLower(argument))
	if site == "" {
		return "A site to unban might help.\r\n"
	}

	removed, err := bm.RemoveBan(site)
	if err != nil {
		return "That site is not currently banned.\r\n"
	}

	if err := bm.WriteBanList(banFilePath); err != nil {
		slog.Error("failed to write ban list", "error", err)
	}

	slog.Info("ban removed", "admin", playerName, "type", banTypeName(removed.BanType), "site", site)
	return "Site unbanned.\r\n"
}

// pathDir returns the directory component of a file path.
// Used by WriteBanList to ensure parent directory exists.
func pathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// Go Improvements Over C
// ======================
// 1. MEMORY SAFETY: C used a singly-linked list with CREATE()/FREE() macros and
//    manual linked-list removal (REMOVE_FROM_LIST macro). Go uses []BanEntry slices —
//    no dangling pointers, no memory leaks.
//
// 2. GLOBAL STATE: C kept ban_list and invalid_list as module-level globals. Go
//    encapsulates them in BanManager, allowing multiple instances (e.g., for tests)
//    and clean dependency injection into World.
//
// 3. RECURSION: C's _write_one_node() used recursion to write the list in reverse
//    insertion order. Go replaces this with a simple reverse iteration — no stack
//    overflow risk on large ban lists.
//
// 4. ERROR HANDLING: C's load_banned() called perror() and returned silently on
//    file errors. Go returns structured errors and uses slog for visibility.
//
// 5. TIME: C stored ban dates as raw Unix timestamps (long). Go uses time.Time
//    with proper formatting and zero-value semantics.
//
// 6. POTENTIAL MODERNIZATION (do not implement now):
//    - Store bans in PostgreSQL so they survive process restarts without flat files.
//    - Add rate-limiting on IsBanned checks (could be called per connection attempt).
//    - Add CIDR-range ban support (C only did substring matching).
//    - BanManager could be embedded in World struct for single-point access.
