/* ************************************************************************
 *   File: alias.c                                        Part of CircleMUD *
 *  Usage: player alias system                                              *
 *                                                                         *
 *  All rights reserved.  See license.doc for complete information.        *
 *                                                                         *
 *  Copyright (C) 1993, 94 by the Trustees of the Johns Hopkins University *
 *  CircleMUD is based on DikuMUD, Copyright (C) 1990, 1991.               *
 ************************************************************************ */

/*
  All parts of this code not covered by the copyright by the Trustees of
  the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
  Dark Pawns Coding Team.

  This includes all original code done for Dark Pawns MUD by other authors.
  All code is the intellectual property of the author, and is used here
  by permission.

  No original code may be duplicated, reused, or executed without the
  written permission of the author. All rights reserved.

  See dp-team.txt or "help coding" online for members of the Dark Pawns
  Coding Team.
*/

// Ported from src/alias.c

package game

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PlayerAlias represents a single alias entry for a player.
// Ported from src/alias.c struct alias
type PlayerAlias struct {
	Alias       string
	Replacement string
	Type        int
}

// AliasDir returns the directory where per-player alias files are stored.
func AliasDir() string {
	return filepath.Join("data", "aliases")
}

// aliasFilePath returns the full path to a player's alias file.
func aliasFilePath(playerName string) string {
	return filepath.Join(AliasDir(), strings.ToLower(playerName)+".alias")
}

// WriteAliases writes a player's aliases to disk.
// Ported from src/alias.c write_aliases().
// Format: length-prefixed text for compatibility with C.
func WriteAliases(playerName string, aliases []PlayerAlias) error {
	dir := AliasDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating alias dir: %w", err)
	}

	fn := aliasFilePath(playerName)

	// unlink(fn) equivalent — remove existing file
	os.Remove(fn)

	if len(aliases) == 0 {
		return nil
	}

	file, err := os.Create(fn)
	if err != nil {
		return fmt.Errorf("creating alias file: %w", err)
	}
	defer file.Close()

	for _, a := range aliases {
		// Write alias length and alias string
		fmt.Fprintf(file, "%d\n", len(a.Alias))
		fmt.Fprintf(file, "%s\n", a.Alias)

		// Write replacement: skip leading spaces (as in C: while(*++buf == ' '))
		repl := strings.TrimLeft(a.Replacement, " ")
		fmt.Fprintf(file, "%d\n", len(repl))
		fmt.Fprintf(file, "%s\n", repl)

		// Write type
		fmt.Fprintf(file, "%d\n", a.Type)
	}

	return nil
}

// ReadAliases reads a player's aliases from disk.
// Ported from src/alias.c read_aliases().
// Returns nil slice if file doesn't exist or is empty.
func ReadAliases(playerName string) ([]PlayerAlias, error) {
	fn := aliasFilePath(playerName)

	file, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening alias file: %w", err)
	}
	defer file.Close()

	var aliases []PlayerAlias
	scanner := bufio.NewScanner(file)

	for {
		// Read alias length
		if !scanner.Scan() {
			break
		}
		lenStr := scanner.Text()
		length, err := strconv.Atoi(lenStr)
		if err != nil {
			break
		}

		// Read alias (length+1 bytes, including newline eaten by fgets in C)
		if !scanner.Scan() {
			break
		}
		aliasStr := scanner.Text()
		if len(aliasStr) > length {
			aliasStr = aliasStr[:length]
		}

		// Read replacement length
		if !scanner.Scan() {
			break
		}
		lenStr = scanner.Text()
		rlen, err := strconv.Atoi(lenStr)
		if err != nil {
			break
		}

		// Read replacement
		if !scanner.Scan() {
			break
		}
		repl := scanner.Text()
		if len(repl) > rlen {
			repl = repl[:rlen]
		}
		// C code prepends a space to replacement
		repl = " " + repl

		// Read type
		if !scanner.Scan() {
			break
		}
		typ, err := strconv.Atoi(scanner.Text())
		if err != nil {
			break
		}

		aliases = append(aliases, PlayerAlias{
			Alias:       aliasStr,
			Replacement: repl,
			Type:        typ,
		})
	}

	if err := scanner.Err(); err != nil {
		return aliases, fmt.Errorf("reading alias file: %w", err)
	}

	return aliases, nil
}

// FindAlias finds an alias entry by alias name, returning the index or -1.
func FindAlias(aliases []PlayerAlias, name string) int {
	for i, a := range aliases {
		if a.Alias == name {
			return i
		}
	}
	return -1
}
