// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Mob represents a parsed mobile from a .mob file.
type Mob struct {
	VNum         int
	Keywords     string
	ShortDesc    string
	LongDesc     string
	DetailedDesc string
	ActionFlags  []string
	AffectFlags  []string
	Alignment    int
	Race         int
	Level        int
	THAC0        int
	AC           int
	HP           DiceRoll
	Damage       DiceRoll
	Gold         int
	Exp          int
	Position     int
	DefaultPos   int
	Sex          int
	RaceStr      string
	Noise        string
	ScriptName   string
	LuaFunctions int
}

// DiceRoll represents a dice expression like 5d10+20.
type DiceRoll struct {
	Num   int
	Sides int
	Plus  int
}

func (d DiceRoll) String() string {
	return fmt.Sprintf("%dd%d+%d", d.Num, d.Sides, d.Plus)
}

// ParseMobFile parses a single .mob file and returns all mobs.
func ParseMobFile(path string) ([]Mob, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var mobs []Mob
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		if strings.HasPrefix(line, "#") {
			vnumStr := line[1:]
			
			// Special case: #99999 is end-of-file marker
			if vnumStr == "99999" {
				break
			}
			
			vnum, err := strconv.Atoi(vnumStr)
			if err != nil {
				return nil, fmt.Errorf("invalid mob vnum: %s", line)
			}

			mob, err := parseMob(scanner, vnum)
			if err != nil {
				return nil, fmt.Errorf("parse mob %d: %w", vnum, err)
			}
			mobs = append(mobs, mob)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return mobs, nil
}

func parseMob(scanner *bufio.Scanner, vnum int) (Mob, error) {
	mob := Mob{VNum: vnum}

	// Keywords (ends with ~)
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob keywords")
	}
	mob.Keywords = strings.TrimSuffix(scanner.Text(), "~")

	// Short description (ends with ~)
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob short desc")
	}
	mob.ShortDesc = strings.TrimSuffix(scanner.Text(), "~")

	// Long description (ends with ~)
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob long desc")
	}
	mob.LongDesc = strings.TrimSuffix(scanner.Text(), "~")

	// Detailed description (ends with ~)
	var descLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "~") {
			descLines = append(descLines, strings.TrimSuffix(line, "~"))
			break
		}
		descLines = append(descLines, line)
	}
	mob.DetailedDesc = strings.Join(descLines, "\n")

	// Action flags, affect flags, alignment, race (ends with E or S)
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob flags line")
	}
	flagsLine := scanner.Text()
	
	// Parse until we hit E or S (Simple flag)
	for !strings.HasSuffix(flagsLine, " E") && !strings.HasSuffix(flagsLine, "E") &&
		  !strings.HasSuffix(flagsLine, " S") && !strings.HasSuffix(flagsLine, "S") {
		if !scanner.Scan() {
			return mob, fmt.Errorf("expected end of flags (E or S)")
		}
		flagsLine = scanner.Text()
	}
	
	// Remove trailing E or S and parse
	flagsLine = strings.TrimSuffix(flagsLine, " E")
	flagsLine = strings.TrimSuffix(flagsLine, "E")
	flagsLine = strings.TrimSuffix(flagsLine, " S")
	flagsLine = strings.TrimSuffix(flagsLine, "S")
	
	// Parse action flags (bitmask as string), affect flags (bitmask), alignment
	// Format: <action_flags> <affect_flags> <alignment> <race>
	fields := strings.Fields(flagsLine)
	if len(fields) >= 3 {
		mob.Alignment, _ = strconv.Atoi(fields[2])
	}
	if len(fields) >= 4 {
		mob.Race, _ = strconv.Atoi(fields[3])
	}

	// Stats line: level thac0 ac hpdice damagedice
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob stats line")
	}
	stats := strings.Fields(scanner.Text())
	if len(stats) >= 9 {
		mob.Level, _ = strconv.Atoi(stats[0])
		mob.THAC0, _ = strconv.Atoi(stats[1])
		mob.AC, _ = strconv.Atoi(stats[2])
		mob.HP.Num, _ = strconv.Atoi(stats[3])
		mob.HP.Sides, _ = strconv.Atoi(stats[4])
		mob.HP.Plus, _ = strconv.Atoi(stats[5])
		mob.Damage.Num, _ = strconv.Atoi(stats[6])
		mob.Damage.Sides, _ = strconv.Atoi(stats[7])
		mob.Damage.Plus, _ = strconv.Atoi(stats[8])
	}

	// Gold and exp line
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob gold/exp line")
	}
	goldExp := strings.Fields(scanner.Text())
	if len(goldExp) >= 2 {
		mob.Gold, _ = strconv.Atoi(goldExp[0])
		mob.Exp, _ = strconv.Atoi(goldExp[1])
	}

	// Position, default position, sex line
	if !scanner.Scan() {
		return mob, fmt.Errorf("expected mob position line")
	}
	pos := strings.Fields(scanner.Text())
	if len(pos) >= 3 {
		mob.Position, _ = strconv.Atoi(pos[0])
		mob.DefaultPos, _ = strconv.Atoi(pos[1])
		mob.Sex, _ = strconv.Atoi(pos[2])
	}

	// Parse optional fields (Race, Noise, Script)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "E" || strings.HasPrefix(line, "$") || strings.HasPrefix(line, "#") {
			// Put back the line if it's the next mob or end of file
			break
		}
		
		if strings.HasPrefix(line, "Race:") {
			mob.RaceStr = strings.TrimSpace(strings.TrimPrefix(line, "Race:"))
		}
		
		if strings.HasPrefix(line, "Noise:") {
			// Noise might be on same line or next
			noise := strings.TrimPrefix(line, "Noise:")
			noise = strings.TrimSpace(noise)
			if strings.HasSuffix(noise, "~") {
				mob.Noise = strings.TrimSuffix(noise, "~")
			} else if noise == "" {
				// Noise is on next line
				if scanner.Scan() {
					mob.Noise = strings.TrimSuffix(scanner.Text(), "~")
				}
			} else {
				mob.Noise = noise
			}
		}
	}

	return mob, nil
}

// ParseAllMobFiles parses all .mob files in a directory.
func ParseAllMobFiles(dir string) ([]Mob, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var allMobs []Mob
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mob") {
			continue
		}

		path := dir + "/" + entry.Name()
		mobs, err := ParseMobFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		allMobs = append(allMobs, mobs...)
	}

	return allMobs, nil
}
