// Package game — alias persistence (ported from alias.c)
// Source: src/alias.c — write_aliases(), read_aliases()
// Port: Wave 13, 2026-04-25
package game

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AliasType constants match the original C alias type field.
// Source: interpreter.h — ALIAS_SIMPLE = 0, ALIAS_COMPLEX = 1
const (
	AliasSimple  = 0 // Simple substitution (no semicolon replacement)
	AliasComplex = 1 // Complex alias (semicolon-delimited multi-command)
)

// Alias represents a single player command alias.
// Source: structs.h struct alias { char *alias; char *replacement; int type; struct alias *next; }
// PlayerAlias is the type used in the Player struct for alias storage.
type PlayerAlias = Alias

type Alias struct {
	Alias       string // The trigger word (what the player types)
	Replacement string // What it expands to (always stored with leading space, per C original)
	Type        int    // AliasSimple or AliasComplex
}

// aliasDir is the directory where alias files are stored.
// Mirrors the C get_filename() ALIAS_FILE path: "plralias/<initial>/<name>.alias"
const aliasDir = "./data/aliases"

// aliasFilePath returns the path to a player's alias file.
// Source: utils.c get_filename() case ALIAS_FILE — "plralias/<initial>/<name>.alias"
func aliasFilePath(playerName string) string {
	if len(playerName) == 0 {
		return ""
	}
	initial := strings.ToLower(string(playerName[0]))
	return filepath.Join(aliasDir, initial, strings.ToLower(playerName)+".alias")
}

// WriteAliases writes a player's alias list to disk.
// Source: alias.c write_aliases() lines 41–71
//
// File format (per alias entry):
//
//	<len of alias>\n
//	<alias string>\n
//	<len of replacement (trimmed)>\n
//	<replacement string (trimmed)>\n
//	<type>\n
//
// The C original strips the leading space from replacement before writing the
// length, then writes the trimmed string. On read, it prepends a space back.
func WriteAliases(playerName string, aliases []Alias) error {
	if len(aliases) == 0 {
		// No aliases — delete file if it exists (mirrors C unlink())
		path := aliasFilePath(playerName)
		_ = os.Remove(path)
		return nil
	}

	path := aliasFilePath(playerName)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("WriteAliases: mkdir %s: %w", filepath.Dir(path), err)
	}

	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("WriteAliases: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	for _, a := range aliases {
		// Strip leading space from replacement before writing length/content,
		// matching the C str_dup(temp->replacement) / while(*++buf == ' ') idiom.
		// Source: alias.c lines 62–65
		trimmed := strings.TrimLeft(a.Replacement, " ")
		_, _ = fmt.Fprintf(
			w, "%d\n%s\n%d\n%s\n%d\n",
			len(a.Alias), a.Alias,
			len(trimmed), trimmed,
			a.Type,
		)
	}
	return w.Flush()
}

// ReadAliases reads a player's alias list from disk.
// Returns an empty slice (not an error) if no alias file exists.
// Source: alias.c read_aliases() lines 73–109
//
// File format: see WriteAliases for the on-disk layout.
// The C original prepends a space to the replacement on read:
//
//	strcpy(temp_buf," "); strcat(temp_buf,buf); — alias.c line 97–98
func ReadAliases(playerName string) ([]Alias, error) {
	path := aliasFilePath(playerName)
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no alias file = no aliases
		}
		return nil, fmt.Errorf("ReadAliases: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var aliases []Alias
	scanner := bufio.NewScanner(f)

	readLine := func() (string, bool) {
		if scanner.Scan() {
			return scanner.Text(), true
		}
		return "", false
	}

	for {
		// Read alias length (unused — just for parity with C format)
		lenLine, ok := readLine()
		if !ok {
			break
		}
		_, err := strconv.Atoi(strings.TrimSpace(lenLine))
		if err != nil {
			break // malformed file
		}

		aliasStr, ok := readLine()
		if !ok {
			break
		}

		// Read replacement length (unused)
		_, ok = readLine()
		if !ok {
			break
		}

		replacementStr, ok := readLine()
		if !ok {
			break
		}

		// Prepend space to replacement, matching C: strcpy(temp_buf," "); strcat(temp_buf,buf)
		// Source: alias.c lines 97–98
		replacement := " " + replacementStr

		typeLine, ok := readLine()
		if !ok {
			break
		}
		aliasType, err := strconv.Atoi(strings.TrimSpace(typeLine))
		if err != nil {
			break
		}

		aliases = append(aliases, Alias{
			Alias:       aliasStr,
			Replacement: replacement,
			Type:        aliasType,
		})
	}

	return aliases, nil
}

// FindAlias searches a slice of aliases for one with the given trigger word.
// Returns the alias and true, or zero value and false if not found.
// Source: interpreter.c find_alias() — case-insensitive search through linked list.
func FindAlias(aliases []Alias, trigger string) (Alias, bool) {
	trigger = strings.ToLower(trigger)
	for _, a := range aliases {
		if strings.ToLower(a.Alias) == trigger {
			return a, true
		}
	}
	return Alias{}, false
}

// ExpandAlias expands an alias according to interpreter.c's perform_alias and
// perform_complex_alias paths. The returned commands are ordered as the C
// input queue receives them. Simple aliases return one direct replacement and
// deliberately discard the original arguments; complex aliases may return one
// command per semicolon and substitute the original arguments into $1..$9/$*.
func ExpandAlias(aliases []Alias, command string) ([]string, bool) {
	trimmed := strings.TrimLeft(command, " \t\n\v\f\r")
	if trimmed == "" {
		return nil, false
	}

	triggerEnd := strings.IndexAny(trimmed, " \t\n\v\f\r")
	if triggerEnd < 0 {
		triggerEnd = len(trimmed)
	}
	trigger := strings.ToLower(trimmed[:triggerEnd])
	a, found := FindAlias(aliases, trigger)
	if !found {
		return []string{command}, false
	}

	if a.Type == AliasSimple {
		// C strcpy(orig, a->replacement): the command's original arguments
		// are not retained for simple aliases.
		return []string{a.Replacement}, true
	}

	rest := ""
	if triggerEnd < len(trimmed) {
		rest = trimmed[triggerEnd:]
	}
	return expandComplexAlias(a.Replacement, rest), true
}

// PerformAlias preserves the original single-string API for callers that only
// need the first expanded command. Session dispatch uses ExpandAlias so it can
// put the remaining complex commands at the front of the input queue.
func PerformAlias(aliases []Alias, command string) (string, bool) {
	expanded, found := ExpandAlias(aliases, command)
	if !found {
		return command, false
	}
	if len(expanded) == 0 {
		return "", true
	}
	return expanded[0], true
}

// expandComplexAlias is the direct Go equivalent of C's
// perform_complex_alias. C tokenizes the text after the trigger into at most
// nine space-delimited tokens, uses $1..$9 for those tokens, and uses $* for
// the entire original text after the trigger.
func expandComplexAlias(replacement, original string) []string {
	tokens := strings.Fields(original)
	if len(tokens) > 9 {
		tokens = tokens[:9]
	}

	commands := make([]string, 0, 1)
	var current strings.Builder
	for i := 0; i < len(replacement); i++ {
		switch replacement[i] {
		case ';':
			commands = append(commands, current.String())
			current.Reset()
		case '$':
			if i+1 >= len(replacement) {
				continue
			}
			i++
			next := replacement[i]
			switch {
			case next >= '1' && next <= '9' && int(next-'1') < len(tokens):
				current.WriteString(tokens[int(next-'1')])
			case next == '*':
				current.WriteString(original)
			default:
				// This mirrors the C fallback: after consuming '$', it
				// writes the following byte, redoubling it only when that
				// byte is itself '$'.
				current.WriteByte(next)
				if next == '$' {
					current.WriteByte('$')
				}
			}
		default:
			current.WriteByte(replacement[i])
		}
	}
	commands = append(commands, current.String())
	return commands
}

// Go Improvements Over C
// ======================
// 1. MEMORY SAFETY: C used a singly-linked list (struct alias *next) allocated with
//    CREATE(). Go uses a plain []Alias slice — no manual malloc/free, no memory leaks.
//
// 2. FILE I/O: C used fprintf()/fscanf() with fixed 127-byte buffers. Go uses
//    bufio.Scanner which handles arbitrary line lengths safely.
//
// 3. DIRECTORY HIERARCHY: C get_filename() created the initial-letter subdirectory
//    manually. Go uses os.MkdirAll() which is atomic and idempotent.
//
// 4. ERROR PROPAGATION: C's write_aliases() called fopen() and silently returned on
//    error. Go returns errors to the caller for proper logging.
//
// 5. ENCODING: C stored lengths as separate lines for fscanf compatibility. The Go
//    port preserves this format for backwards file compatibility but could use JSON
//    or a simpler line-per-field format in a future migration (deferred: phase 4).
//
// 6. POTENTIAL MODERNIZATION (do not implement now):
//    - Store aliases in PostgreSQL alongside player data rather than flat files.
//    - Use atomic file writes (write to temp, rename) for crash safety.
//    - Add alias count limit (original had none — could be abused with many aliases).
