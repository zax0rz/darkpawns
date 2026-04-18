// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	WearFlags    [3]int
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

// ParseObjFile parses a single .obj file and returns all objects.
func ParseObjFile(path string) ([]Obj, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var objs []Obj
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		if strings.HasPrefix(line, "#") {
			vnum, err := strconv.Atoi(line[1:])
			if err != nil {
				return nil, fmt.Errorf("invalid obj vnum: %s", line)
			}

			obj, err := parseObj(scanner, vnum)
			if err != nil {
				return nil, fmt.Errorf("parse obj %d: %w", vnum, err)
			}
			objs = append(objs, obj)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return objs, nil
}

func parseObj(scanner *bufio.Scanner, vnum int) (Obj, error) {
	obj := Obj{VNum: vnum}

	// Keywords (ends with ~)
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj keywords")
	}
	obj.Keywords = strings.TrimSuffix(scanner.Text(), "~")

	// Short description (ends with ~)
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj short desc")
	}
	obj.ShortDesc = strings.TrimSuffix(scanner.Text(), "~")

	// Long description (ends with ~)
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj long desc")
	}
	obj.LongDesc = strings.TrimSuffix(scanner.Text(), "~")

	// Action description (ends with ~, can be empty)
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj action desc")
	}
	obj.ActionDesc = strings.TrimSuffix(scanner.Text(), "~")

	// Type flag and flags line
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj type/flags line")
	}
	flags := strings.Fields(scanner.Text())
	if len(flags) >= 9 {
		obj.TypeFlag, _ = strconv.Atoi(flags[0])
		for i := 0; i < 4; i++ {
			obj.ExtraFlags[i] = parseFlag(flags[1+i])
		}
		for i := 0; i < 3; i++ {
			obj.WearFlags[i] = parseFlag(flags[5+i])
		}
	}

	// Values line
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj values line")
	}
	values := strings.Fields(scanner.Text())
	for i := 0; i < 4 && i < len(values); i++ {
		obj.Values[i], _ = strconv.Atoi(values[i])
	}

	// Weight, cost, load percent line
	if !scanner.Scan() {
		return obj, fmt.Errorf("expected obj weight/cost line")
	}
	wcl := strings.Fields(scanner.Text())
	if len(wcl) >= 3 {
		obj.Weight, _ = strconv.Atoi(wcl[0])
		obj.Cost, _ = strconv.Atoi(wcl[1])
		obj.LoadPercent, _ = strconv.ParseFloat(wcl[2], 64)
	}

	// Parse extra descriptions and affects until $ or #
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "$" || strings.HasPrefix(line, "#") {
			break
		}

		if line == "E" {
			// Extra description
			var ed ExtraDesc
			if scanner.Scan() {
				ed.Keywords = strings.TrimSuffix(scanner.Text(), "~")
			}
			if scanner.Scan() {
				ed.Description = strings.TrimSuffix(scanner.Text(), "~")
			}
			obj.ExtraDescs = append(obj.ExtraDescs, ed)
		}

		if line == "A" {
			// Affect
			if scanner.Scan() {
				affectFields := strings.Fields(scanner.Text())
				if len(affectFields) >= 2 {
					var aff ObjAffect
					aff.Location, _ = strconv.Atoi(affectFields[0])
					aff.Modifier, _ = strconv.Atoi(affectFields[1])
					obj.Affects = append(obj.Affects, aff)
				}
			}
		}
	}

	return obj, nil
}

// parseFlag converts an ASCII flag string like "abc" to an integer bitmask.
func parseFlag(s string) int {
	// CircleMUD uses lowercase letters a-z for bits 0-25, A-Z for bits 26-31
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
