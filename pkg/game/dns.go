package game

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// dnsHashBuckets is C's DNS_HASH_NUM (src/db.h:199).
const dnsHashBuckets = 257

type dnsEntry struct {
	ip   [4]int
	name string
}

// ExecDns implements C do_dns (src/act.wizard.c:3106-3208). It returns the
// list page text to the session layer; all other branches send their direct
// player-facing text here. The session owns page_string because paging is a
// descriptor/transport concern in Go.
func (w *World) ExecDns(ch *Player, argument string) string {
	if ch == nil {
		return ""
	}

	w.dnsMu.Lock()
	defer w.dnsMu.Unlock()
	w.loadDNSLocked()

	arg1, arg2 := halfChop(argument)
	if arg1 == "" {
		ch.SendMessage("You shouldn't be using this if you don't know what it does!\r\n")
		return ""
	}

	switch {
	case isDNSAbbreviation(arg1, "delete"):
		w.dnsDeleteLocked(ch, arg2)
	case isDNSAbbreviation(arg1, "add"):
		w.dnsAddLocked(ch, arg2)
	case isDNSAbbreviation(arg1, "list"):
		return w.dnsListLocked()
	}
	return ""
}

func isDNSAbbreviation(argument, word string) bool {
	return argument != "" && strings.HasPrefix(strings.ToLower(word), strings.ToLower(argument))
}

func (w *World) dnsDeleteLocked(ch *Player, argument string) {
	var ip [4]int
	if strings.TrimSpace(argument) == "" || scanDNSIP(argument, &ip, 3) != 3 {
		ch.SendMessage("Delete what?\r\n")
		return
	}

	for bucket := range w.dnsCache {
		for i := range w.dnsCache[bucket] {
			entry := &w.dnsCache[bucket][i]
			if entry.ip[0] == ip[0] && entry.ip[1] == ip[1] && entry.ip[2] == ip[2] {
				ch.SendMessage(fmt.Sprintf("Deleting %s.\r\n", entry.name))
				entry.ip[0] = -1
			}
		}
	}
	w.saveDNSLocked()
}

func (w *World) dnsAddLocked(ch *Player, argument string) {
	ipText, name := twoDNSArguments(argument)
	if ipText == "" || name == "" {
		ch.SendMessage("Add what?\r\n")
		return
	}

	var ip [4]int
	ip[3] = -1
	if scanDNSIP(ipText, &ip, 4) < 3 {
		ch.SendMessage("Add what?\r\n")
		return
	}

	bucket := (ip[0] + ip[1] + ip[2]) % dnsHashBuckets
	if bucket < 0 {
		// C would index out of bounds for a negative hash. Keep malformed input
		// from corrupting the Go process while preserving valid C inputs.
		ch.SendMessage("Add what?\r\n")
		return
	}
	w.dnsCache[bucket] = append([]dnsEntry{{ip: ip, name: name}}, w.dnsCache[bucket]...)
	w.saveDNSLocked()
	ch.SendMessage("OK!\r\n")
}

// twoDNSArguments mirrors two_arguments: both tokens use one_argument's
// fill-word skipping and lowercasing. DNS names in the original are one token.
func twoDNSArguments(argument string) (string, string) {
	first, rest := oneArgument(argument)
	second, _ := oneArgument(rest)
	return first, second
}

func scanDNSIP(text string, ip *[4]int, fields int) int {
	returnFields := 0
	switch fields {
	case 3:
		returnFields, _ = fmt.Sscanf(text, "%d.%d.%d", &ip[0], &ip[1], &ip[2])
	case 4:
		returnFields, _ = fmt.Sscanf(text, "%d.%d.%d.%d", &ip[0], &ip[1], &ip[2], &ip[3])
	}
	return returnFields
}

func (w *World) dnsListLocked() string {
	var buf strings.Builder
	buf.WriteString("IP Address        Host Name\r\n")
	for bucket := range w.dnsCache {
		for _, entry := range w.dnsCache[bucket] {
			if entry.ip[0] < 0 {
				break
			}
			fmt.Fprintf(&buf, "%s.%s.%s.", formatDNSOctet(entry.ip[0]),
				formatDNSOctet(entry.ip[1]), formatDNSOctet(entry.ip[2]))
			if entry.ip[3] < 0 {
				buf.WriteString("   ")
			} else {
				buf.WriteString(formatDNSOctet(entry.ip[3]))
			}
			fmt.Fprintf(&buf, "   %s\r\n", entry.name)
		}
	}
	return buf.String()
}

func formatDNSOctet(value int) string {
	switch {
	case value < 10:
		return fmt.Sprintf("00%d", value)
	case value < 100:
		return fmt.Sprintf("0%d", value)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func (w *World) dnsFilePath() string {
	worldPath := filepath.Clean(w.WorldPath)
	if worldPath == "." || worldPath == "" {
		return filepath.Join("etc", "dns")
	}
	return filepath.Join(filepath.Dir(worldPath), "etc", "dns")
}

func (w *World) loadDNSLocked() {
	if w.dnsLoaded {
		return
	}
	w.dnsLoaded = true

	file, err := os.Open(w.dnsFilePath())
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("unable to open DNS cache", "path", w.dnsFilePath(), "error", err)
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Warn("unable to close DNS cache", "path", w.dnsFilePath(), "error", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "~" {
			break
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		var ip [4]int
		if scanDNSIP(fields[0], &ip, 4) < 3 {
			continue
		}
		bucket := (ip[0] + ip[1] + ip[2]) % dnsHashBuckets
		if bucket < 0 {
			continue
		}
		w.dnsCache[bucket] = append([]dnsEntry{{ip: ip, name: fields[1]}}, w.dnsCache[bucket]...)
	}
	if err := scanner.Err(); err != nil {
		slog.Warn("unable to read DNS cache", "path", w.dnsFilePath(), "error", err)
	}
}

func (w *World) saveDNSLocked() {
	path := w.dnsFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		slog.Error("unable to create DNS cache directory", "path", filepath.Dir(path), "error", err)
		return
	}
	file, err := os.Create(filepath.Clean(path))
	if err != nil {
		slog.Error("unable to create DNS cache", "path", path, "error", err)
		return
	}
	writer := bufio.NewWriter(file)
	for bucket := range w.dnsCache {
		for _, entry := range w.dnsCache[bucket] {
			if _, err := fmt.Fprintf(writer, "%d.%d.%d.%d %s\n", entry.ip[0], entry.ip[1], entry.ip[2], entry.ip[3], entry.name); err != nil {
				slog.Error("unable to write DNS cache", "path", path, "error", err)
				_ = file.Close()
				return
			}
		}
	}
	if _, err := writer.WriteString("~\n"); err != nil {
		slog.Error("unable to finish DNS cache", "path", path, "error", err)
	}
	if err := writer.Flush(); err != nil {
		slog.Error("unable to flush DNS cache", "path", path, "error", err)
	}
	if err := file.Close(); err != nil {
		slog.Error("unable to close DNS cache", "path", path, "error", err)
	}
}
