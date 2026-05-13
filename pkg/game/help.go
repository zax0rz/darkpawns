package game

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// HelpEntry represents a single help file entry.
type HelpEntry struct {
	Keyword string // Primary keyword (for display)
	Entry   string // Full entry text
}

// LoadHelpFiles loads all .hlp files from the given directory into a help table.
// Format (from C db.c load_help):
//   keyword1 [keyword2 ...]
//   entry text lines
//   #
//   ...
//   $
//
// Each keyword line creates a separate HelpEntry with the same text.
// The '$' line terminates the file.
func LoadHelpFiles(dir string) ([]HelpEntry, error) {
	// Read the index file to get the list of .hlp files
	indexPath := filepath.Join(dir, "index")
	indexFile, err := os.Open(indexPath)
	if err != nil {
		// Try index.mini as fallback
		indexPath = filepath.Join(dir, "index.mini")
		indexFile, err = os.Open(indexPath)
		if err != nil {
			return nil, err
		}
	}
	defer indexFile.Close()

	var entries []HelpEntry
	scanner := bufio.NewScanner(indexFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "$" {
			continue
		}
		// line is a .hlp filename
		hlpEntries, err := loadHelpFile(filepath.Join(dir, line))
		if err != nil {
			continue // skip unreadable files
		}
		entries = append(entries, hlpEntries...)
	}
	return entries, nil
}

// loadHelpFile loads a single .hlp file and returns all entries in it.
func loadHelpFile(path string) ([]HelpEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []HelpEntry
	var currentKeywords []string
	var currentEntry strings.Builder

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// '$' terminates the entire file
		if strings.TrimSpace(line) == "$" {
			break
		}

		// '#' terminates the current entry
		if strings.TrimSpace(line) == "#" {
			// Save the current entry under all its keywords
			entryText := currentEntry.String()
			for _, kw := range currentKeywords {
				entries = append(entries, HelpEntry{
					Keyword: kw,
					Entry:   entryText,
				})
			}
			currentKeywords = nil
			currentEntry.Reset()
			continue
		}

		// If we have no keywords yet, this line is the keyword line
		if currentKeywords == nil {
			currentKeywords = strings.Fields(line)
			continue
		}

		// Otherwise, it's entry text
		currentEntry.WriteString(line)
		currentEntry.WriteString("\r\n")
	}

	// Handle any trailing entry (if file doesn't end with #)
	if currentEntry.Len() > 0 && len(currentKeywords) > 0 {
		entryText := currentEntry.String()
		for _, kw := range currentKeywords {
			entries = append(entries, HelpEntry{
				Keyword: kw,
				Entry:   entryText,
			})
		}
	}

	return entries, nil
}

// SearchHelp searches the help table for a keyword (case-insensitive).
// Returns the matching entry or nil if not found.
func SearchHelp(table []HelpEntry, keyword string) *HelpEntry {
	keyword = strings.ToLower(keyword)
	for i := range table {
		if strings.ToLower(table[i].Keyword) == keyword {
			return &table[i]
		}
	}
	return nil
}
