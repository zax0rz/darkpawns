// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// Max object affects, matching C's MAX_OBJ_AFFECT (structs.h line 656)
	MAX_OBJ_AFFECT = 6

	// Item type constants matching structs.h (used for container weight validation)
	ITEM_DRINKCON = 17
	ITEM_FOUNTAIN  = 23
)

// Obj represents a parsed object from a .obj file.
type Obj struct {
	VNum         int
	Keywords     string
	ShortDesc    string
	LongDesc     string
	ActionDesc   string
	TypeFlag     int
	ExtraFlags   [4]int
	WearFlags    [4]int
	Values       [4]int
	Weight       int
	Cost         int
	LoadPercent  float64
	Affects      []ObjAffect
	ExtraDescs   []ExtraDesc
	ScriptName   string
	LuaFunctions int
}

// ObjAffect represents an object affect (stat modifier).
type ObjAffect struct {
	Location int
	Modifier int
}

// ExtraDesc represents an extra description on an object.
type ExtraDesc struct {
	Keywords    string
	Description string
}

// lineBuffer wraps a bufio.Scanner to allow one-line "unread" for the obj parser.
// Needed because parseObj reads until it sees the next #VNUM line, then must
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

// ParseObjFile parses a single .obj file and returns all objects.
func ParseObjFile(path string) ([]Obj, error) {
	if err := validateWorldPath(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path) // #nosec G703 — world data, trusted internal path
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var objs []Obj
	lb := &lineBuffer{scanner: bufio.NewScanner(file)}

	for lb.Scan() {
		line := strings.TrimSpace(lb.Text())

		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		if strings.HasPrefix(line, "#") {
			vnum, err := strconv.Atoi(line[1:])
			if err != nil || vnum == 0 || vnum == 99999 {
				// #0, #99999, or parse error = end of file sentinel
				break
			}

			obj, nextLine, err := parseObj(lb, vnum)
			if err != nil {
				return nil, fmt.Errorf("parse obj %d: %w", vnum, err)
			}
			objs = append(objs, obj)

			// parseObj consumed the next #VNUM line — put it back so the
			// outer loop sees it on the next iteration.
			if nextLine != "" {
				lb.Unread(nextLine)
			}
		}
	}

	if err := lb.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return objs, nil
}

// parseObj parses one object record. Returns the object, the next unconsumed
// line (the #VNUM that terminated the E/A block), and any error.
func parseObj(lb *lineBuffer, vnum int) (Obj, string, error) {
	obj := Obj{VNum: vnum}

	// Keywords (ends with ~)
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj keywords")
	}
	obj.Keywords = strings.TrimSuffix(lb.Text(), "~")

	// Short description (ends with ~)
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj short desc")
	}
	obj.ShortDesc = strings.TrimSuffix(lb.Text(), "~")

	// Long description (ends with ~)
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj long desc")
	}
	obj.LongDesc = strings.TrimSuffix(lb.Text(), "~")

	// Action description (ends with ~, can be empty)
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj action desc")
	}
	obj.ActionDesc = strings.TrimSuffix(lb.Text(), "~")

	// Type flag and flags line
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj type/flags line")
	}
	flags := strings.Fields(lb.Text())
	if len(flags) >= 9 {
		obj.TypeFlag, _ = strconv.Atoi(flags[0])
		for i := 0; i < 4; i++ {
			obj.ExtraFlags[i] = parseFlag(flags[1+i])
		}
		for i := 0; i < 4; i++ {
			obj.WearFlags[i] = parseFlag(flags[5+i])
		}
	}

	// Values line
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj values line")
	}
	values := strings.Fields(lb.Text())
	for i := 0; i < 4 && i < len(values); i++ {
		obj.Values[i], _ = strconv.Atoi(values[i])
	}

	// Weight, cost, load percent line
	if !lb.Scan() {
		return obj, "", fmt.Errorf("expected obj weight/cost line")
	}
	wcl := strings.Fields(lb.Text())
	if len(wcl) >= 3 {
		obj.Weight, _ = strconv.Atoi(wcl[0])
		obj.Cost, _ = strconv.Atoi(wcl[1])
		obj.LoadPercent, _ = strconv.ParseFloat(wcl[2], 64)
	}

	// Auto-cap the first letter of the long description (matching C behavior in parse_object)
	if len(obj.LongDesc) > 0 {
		runes := []rune(obj.LongDesc)
		runes[0] = toUpper(runes[0])
		obj.LongDesc = string(runes)
	}

	// Container weight validation (matching C behavior in parse_object)
	// For drink containers and fountains, ensure weight >= max fill volume
	if obj.TypeFlag == ITEM_DRINKCON || obj.TypeFlag == ITEM_FOUNTAIN {
		if obj.Weight < obj.Values[1] {
			obj.Weight = obj.Values[1] + 5
		}
	}

	// Parse extra descriptions (E), affects (A), and scripts (S) until "$" or next "#VNUM".
	// When we see a "#" line, return it as nextLine so the caller can unread it.
	var nextLine string
	for lb.Scan() {
		line := strings.TrimSpace(lb.Text())

		if line == "$" {
			break
		}
		if strings.HasPrefix(line, "#") {
			// Next object's vnum — hand back to caller
			nextLine = line
			break
		}

		if line == "E" {
			// Extra description: keywords~ then multi-line desc ending with ~
			var ed ExtraDesc
			if lb.Scan() {
				ed.Keywords = strings.TrimSuffix(lb.Text(), "~")
			}
			var descLines []string
			for lb.Scan() {
				descLine := lb.Text()
				trimmed := strings.TrimSpace(descLine)
				if strings.HasSuffix(trimmed, "~") {
					descLines = append(descLines, strings.TrimSuffix(trimmed, "~"))
					break
				}
				descLines = append(descLines, descLine)
			}
			ed.Description = strings.Join(descLines, "\n")
			obj.ExtraDescs = append(obj.ExtraDescs, ed)
		}

		if line == "S" {
			// Script line: S <name> <lua_functions>
			// Matching C behavior in parse_object (case 'S')
			if lb.Scan() {
				scriptLine := strings.TrimSpace(lb.Text())
				scriptFields := strings.Fields(scriptLine)
				if len(scriptFields) >= 1 {
					obj.ScriptName = scriptFields[0]
				}
				if len(scriptFields) >= 2 {
					obj.LuaFunctions, _ = strconv.Atoi(scriptFields[1])
				}
			}
		}

		if line == "A" {
			// Affect: location modifier
			// C enforces MAX_OBJ_AFFECT (structs.h:656) — limit in Go too
			if len(obj.Affects) >= MAX_OBJ_AFFECT {
				// Consume the affect line but discard it
				if lb.Scan() {
					// consumed and discarded
				}
				continue
			}
			if lb.Scan() {
				affectFields := strings.Fields(lb.Text())
				if len(affectFields) >= 2 {
					var aff ObjAffect
					aff.Location, _ = strconv.Atoi(affectFields[0])
					aff.Modifier, _ = strconv.Atoi(affectFields[1])
					obj.Affects = append(obj.Affects, aff)
				}
			}
		}
	}

	return obj, nextLine, nil
}

// parseFlag converts a flag value to an integer.
// Dark Pawns stores flags as plain integers (e.g. "8193"), not CircleMUD's
// letter-encoded bitmasks. We try integer parse first, fall back to letters.
// This matches C's asciiflag_conv() in db.c:751-772 which does the same logic.
func parseFlag(s string) int {
	// Try plain integer first (Dark Pawns format)
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	// Fall back to letter-encoded bitmask (original CircleMUD format, asciiflag_conv)
	result := 0
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			result |= 1 << (c - 'a')
		} else if c >= 'A' && c <= 'Z' {
			result |= 1 << (26 + c - 'A')
		}
	}
	return result
}

// toUpper converts a rune to uppercase, matching C toupper() behavior.
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

// ParseAllObjFiles parses all .obj files in a directory.
func ParseAllObjFiles(dir string) ([]Obj, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var allObjs []Obj
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".obj") {
			continue
		}

		path := dir + "/" + entry.Name()
		objs, err := ParseObjFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		allObjs = append(allObjs, objs...)
	}

	return allObjs, nil
}
