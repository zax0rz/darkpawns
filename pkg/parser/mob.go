// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// lineBuffer wraps a bufio.Scanner to allow one-line "unread" for the mob parser.
// Needed because parseMob reads until it sees the next #VNUM line, then must
// return that line to the caller rather than consuming it.
type lineBuffer struct {
	scanner  *bufio.Scanner
	buffered string
	has      bool
}

func (lb *lineBuffer) Scan() bool {
	if lb.has {
		lb.has = false
		return true
	}
	return lb.scanner.Scan()
}

func (lb *lineBuffer) Text() string {
	if lb.has {
		return lb.buffered
	}
	return lb.scanner.Text()
}

func (lb *lineBuffer) Unread(line string) {
	lb.buffered = line
	lb.has = true
}

func (lb *lineBuffer) Err() error {
	return lb.scanner.Err()
}

// Mob represents a parsed mobile from a .mob file.
type Mob struct {
	VNum           int
	Keywords       string
	ShortDesc      string
	LongDesc       string
	DetailedDesc   string
	ActionFlags    []string
	AffectFlags    []string
	Alignment      int
	Race           int
	Level          int
	THAC0          int
	AC             int
	HP             DiceRoll
	Damage         DiceRoll
	Gold           int
	Exp            int
	Position       int
	DefaultPos     int
	Sex            int
	Weight         int
	Height         int
	RaceStr        string
	Noise          string
	BareHandAttack int
	Str            int
	StrAdd         int
	Int            int
	Wis            int
	Dex            int
	Con            int
	Cha            int
	ScriptName     string
	LuaFunctions   int
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
	if err := validateWorldPath(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path) // #nosec G703 — world data, trusted internal path
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var mobs []Mob
	lb := &lineBuffer{scanner: bufio.NewScanner(file)}

	for lb.Scan() {
		line := strings.TrimSpace(lb.Text())

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

			mob, nextLine, err := parseMob(lb, vnum)
			if err != nil {
				return nil, fmt.Errorf("parse mob %d: %w", vnum, err)
			}
			mobs = append(mobs, mob)

			if nextLine != "" {
				lb.Unread(nextLine)
			}
		}
	}

	if err := lb.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return mobs, nil
}

func parseMob(lb *lineBuffer, vnum int) (Mob, string, error) {
	mob := Mob{VNum: vnum}
	var nextLine string

	// Keywords (ends with ~)
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob keywords")
	}
	mob.Keywords = strings.TrimSuffix(lb.Text(), "~")

	// Short description (ends with ~)
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob short desc")
	}
	mob.ShortDesc = strings.TrimSuffix(lb.Text(), "~")

	// C source (parse_mobile): auto-lowercase articles in short_desc
	// "A", "An", "The" at start of short desc -> "a", "an", "the"
	{
		sd := mob.ShortDesc
		if sd != "" {
			// Find first run of non-space starting word
			trimmed := strings.TrimLeftFunc(sd, unicode.IsSpace)
			if trimmed != sd {
				// Skip leading spaces, store them
				leadLen := len(sd) - len(trimmed)
				// Find first word
				fields := strings.Fields(trimmed)
				if len(fields) > 0 {
					lw := strings.ToLower(fields[0])
					if lw == "a" || lw == "an" || lw == "the" {
						offset := leadLen
						mob.ShortDesc = sd[:offset] + lw + sd[offset+len(fields[0]):]
					}
				}
			} else {
				fields := strings.Fields(sd)
				if len(fields) > 0 {
					lw := strings.ToLower(fields[0])
					if lw == "a" || lw == "an" || lw == "the" {
						mob.ShortDesc = lw + sd[len(fields[0]):]
					}
				}
			}
		}
	}

	// Long description (ends with ~)
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob long desc")
	}
	mob.LongDesc = strings.TrimSuffix(lb.Text(), "~")

	// Detailed description (ends with ~)
	var descLines []string
	for lb.Scan() {
		line := lb.Text()
		if strings.HasSuffix(line, "~") {
			descLines = append(descLines, strings.TrimSuffix(line, "~"))
			break
		}
		descLines = append(descLines, line)
	}
	mob.DetailedDesc = strings.Join(descLines, "\n")

	// Action flags, affect flags, alignment, race (ends with E or S)
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob flags line")
	}
	flagsLine := lb.Text()

	// Parse until we hit E or S (Simple flag)
	for !strings.HasSuffix(flagsLine, " E") && !strings.HasSuffix(flagsLine, "E") &&
		!strings.HasSuffix(flagsLine, " S") && !strings.HasSuffix(flagsLine, "S") {
		if !lb.Scan() {
			return mob, "", fmt.Errorf("expected end of flags (E or S)")
		}
		flagsLine = lb.Text()
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
	} else {
		// C source: default race = RACE_OTHER (7)
		mob.Race = 7
	}

	// Stats line: level thac0 ac hpdice damagedice
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob stats line")
	}
	stats := strings.Fields(lb.Text())
	if len(stats) >= 9 {
		mob.Level, _ = strconv.Atoi(stats[0])
		mob.THAC0, _ = strconv.Atoi(stats[1])

		// C source: mob_proto[i].points.armor = 10 * t[2]
		// AC in area file is raw; C multiplies by 10
		rawAC, _ := strconv.Atoi(stats[2])
		mob.AC = 10 * rawAC

		mob.HP.Num, _ = strconv.Atoi(stats[3])
		mob.HP.Sides, _ = strconv.Atoi(stats[4])
		mob.HP.Plus, _ = strconv.Atoi(stats[5])
		mob.Damage.Num, _ = strconv.Atoi(stats[6])
		mob.Damage.Sides, _ = strconv.Atoi(stats[7])
		mob.Damage.Plus, _ = strconv.Atoi(stats[8])
	}

	// C source (parse_simple_mob): base stats start at 11
	mob.Str = 11
	mob.Int = 11
	mob.Wis = 11
	mob.Dex = 11
	mob.Con = 11
	mob.Cha = 11

	// C source (parse_simple_mob): level-based stat boosts for mobs level 15+
	// For each stat: stat += MIN(number(0, statmod), 7) where statmod = level - 15
	if mob.Level > 15 {
		statmod := mob.Level - 15
		add := func() int {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			v := rand.Intn(statmod + 1) // number(0, statmod) = rand.Intn(statmod+1)
			if v > 7 {
				return 7
			}
			return v
		}
		mob.Str += add()
		mob.Int += add()
		mob.Wis += add()
		mob.Dex += add()
		mob.Con += add()
		mob.Cha += add()
	}

	// Gold and exp line
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob gold/exp line")
	}
	goldExp := strings.Fields(lb.Text())
	if len(goldExp) >= 2 {
		mob.Gold, _ = strconv.Atoi(goldExp[0])
		mob.Exp, _ = strconv.Atoi(goldExp[1])
	}

	// Position, default position, sex line
	if !lb.Scan() {
		return mob, "", fmt.Errorf("expected mob position line")
	}
	pos := strings.Fields(lb.Text())
	if len(pos) >= 3 {
		mob.Position, _ = strconv.Atoi(pos[0])
		mob.DefaultPos, _ = strconv.Atoi(pos[1])
		mob.Sex, _ = strconv.Atoi(pos[2])
	}

	// C source: default weight=200, height=198
	mob.Weight = 200
	mob.Height = 198

	// Parse optional fields (E-specs: Race, Noise, Script, BareHandAttack, etc.)
	// C source: interpret_espec() handles these E-spec keywords
	for lb.Scan() {
		line := strings.TrimSpace(lb.Text())

		if line == "E" || strings.HasPrefix(line, "$") || strings.HasPrefix(line, "#") {
			nextLine = line
			break
		}

		if strings.HasPrefix(line, "BareHandAttack:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "BareHandAttack:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 0 {
					v = 0
				}
				if v > 99 {
					v = 99
				}
				mob.BareHandAttack = v
			}
			continue
		}

		if strings.HasPrefix(line, "Str:") && !strings.HasPrefix(line, "StrAdd:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Str:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Str = v
			}
			continue
		}

		if strings.HasPrefix(line, "StrAdd:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "StrAdd:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 0 {
					v = 0
				}
				if v > 100 {
					v = 100
				}
				mob.StrAdd = v
			}
			continue
		}

		if strings.HasPrefix(line, "Int:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Int:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Int = v
			}
			continue
		}

		if strings.HasPrefix(line, "Wis:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Wis:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Wis = v
			}
			continue
		}

		if strings.HasPrefix(line, "Dex:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Dex:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Dex = v
			}
			continue
		}

		if strings.HasPrefix(line, "Con:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Con:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Con = v
			}
			continue
		}

		if strings.HasPrefix(line, "Cha:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Cha:"))
			if v, err := strconv.Atoi(val); err == nil {
				if v < 3 {
					v = 3
				}
				if v > 25 {
					v = 25
				}
				mob.Cha = v
			}
			continue
		}

		if strings.HasPrefix(line, "Race:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Race:"))
			if r, err := strconv.Atoi(val); err == nil {
				mob.Race = r
			}
			mob.RaceStr = val
			continue
		}

		if strings.HasPrefix(line, "Noise:") {
			noise := strings.TrimPrefix(line, "Noise:")
			noise = strings.TrimSpace(noise)
			if strings.HasSuffix(noise, "~") {
				mob.Noise = strings.TrimSuffix(noise, "~")
			} else if noise == "" {
				if lb.Scan() {
					mob.Noise = strings.TrimSuffix(lb.Text(), "~")
				}
			} else {
				mob.Noise = noise
			}
			continue
		}

		if strings.HasPrefix(line, "Script:") {
			scriptLine := strings.TrimPrefix(line, "Script:")
			scriptLine = strings.TrimSpace(scriptLine)
			sf := strings.Fields(scriptLine)
			if len(sf) >= 2 {
				mob.ScriptName = sf[0]
				mob.LuaFunctions, _ = strconv.Atoi(sf[1])
			}
			continue
		}
	}

	return mob, nextLine, nil
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

// Scripting interface implementations for parser.Mob

func (m *Mob) GetShortDesc() string {
	return m.ShortDesc
}

func (m *Mob) GetGold() int {
	return m.Gold
}

func (m *Mob) GetLevel() int {
	return m.Level
}

func (m *Mob) GetScriptName() string {
	return m.ScriptName
}

func (m *Mob) GetLuaFunctions() int {
	return m.LuaFunctions
}

func (m *Mob) GetAlignment() int {
	return m.Alignment
}

